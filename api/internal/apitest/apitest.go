// Package apitest starts the real HTTP stack (router + Postgres + JWT verification) for controller tests.
package apitest

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sorenhoang/sr-english/api/internal/auth"
	"github.com/sorenhoang/sr-english/api/internal/config"
	"github.com/sorenhoang/sr-english/api/internal/httpapi"
	"github.com/sorenhoang/sr-english/api/internal/testdb"
)

const (
	Issuer   = "https://test-project.supabase.co/auth/v1"
	Audience = "authenticated"
	keyID    = "test-key"

	// Emails on the allowlist; anything else must be rejected with 403.
	AllowedEmail = "learner@example.com"
	OtherEmail   = "second@example.com"
)

type Env struct {
	URL  string
	Pool *pgxpool.Pool
	key  *ecdsa.PrivateKey
}

// New wires the router exactly as main does, with a local JWKS server instead of Supabase.
func New(t *testing.T) *Env {
	t.Helper()
	return NewWithDeps(t, func(*httpapi.Deps) {})
}

// NewWithDeps lets a test adjust Deps (e.g. config limits) before the server starts.
func NewWithDeps(t *testing.T, adjust func(*httpapi.Deps)) *Env {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("apitest: generate key: %v", err)
	}
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{publicJWK(key)}}); err != nil {
			t.Errorf("apitest: encode jwks: %v", err)
		}
	}))
	t.Cleanup(jwks.Close)

	verifier, err := auth.NewVerifier(context.Background(), auth.VerifierConfig{
		JWKSURL: jwks.URL, Issuer: Issuer, Audience: Audience,
	})
	if err != nil {
		t.Fatalf("apitest: verifier: %v", err)
	}
	pool := testdb.New(t)
	deps := httpapi.Deps{
		Pool:          pool,
		Verifier:      verifier,
		AllowedEmails: []string{AllowedEmail, OtherEmail},
		Learning:      config.DefaultLearning(),
	}
	adjust(&deps)
	srv := httptest.NewServer(httpapi.NewRouter(deps))
	t.Cleanup(srv.Close)
	return &Env{URL: srv.URL, Pool: pool, key: key}
}

// Claims returns valid Supabase-style claims for sub/email, expiring in an hour.
func Claims(sub, email string) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"sub": sub, "email": email, "iss": Issuer, "aud": Audience,
		"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(), "role": "authenticated",
	}
}

// Token signs valid claims with the key the server trusts.
func (e *Env) Token(t *testing.T, sub, email string) string {
	t.Helper()
	return e.Sign(t, Claims(sub, email))
}

func (e *Env) Sign(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	return sign(t, e.key, claims)
}

// SignWithUntrustedKey signs with a key the JWKS does not publish.
func (e *Env) SignWithUntrustedKey(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	other, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("apitest: generate key: %v", err)
	}
	return sign(t, other, claims)
}

// Do sends a request; token "" sends no Authorization header. body nil sends no body.
func (e *Env) Do(t *testing.T, method, path, token string, body any) (int, []byte) {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("apitest: marshal body: %v", err)
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.URL+path, r)
	if err != nil {
		t.Fatalf("apitest: new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("apitest: %s %s: %v", method, path, err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Logf("apitest: close body: %v", err)
		}
	}()
	out, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("apitest: read body: %v", err)
	}
	return res.StatusCode, out
}

// Decode unmarshals a response body or fails the test.
func Decode(t *testing.T, body []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("apitest: decode %s: %v", body, err)
	}
}

func sign(t *testing.T, key *ecdsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	tok.Header["kid"] = keyID
	s, err := tok.SignedString(key)
	if err != nil {
		t.Fatalf("apitest: sign: %v", err)
	}
	return s
}

func publicJWK(key *ecdsa.PrivateKey) map[string]string {
	pub, err := key.PublicKey.ECDH()
	if err != nil {
		panic(err)
	}
	raw := pub.Bytes() // 0x04 || X || Y
	enc := base64.RawURLEncoding.EncodeToString
	return map[string]string{
		"kty": "EC", "crv": "P-256", "kid": keyID, "alg": "ES256", "use": "sig",
		"x": enc(raw[1:33]), "y": enc(raw[33:65]),
	}
}
