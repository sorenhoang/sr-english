package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/sorenhoang/sr-english/api/internal/apitest"
)

type meResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func TestAuthShouldReturn401WhenAuthorizationHeaderIsMissing(t *testing.T) {
	env := apitest.New(t)
	status, _ := env.Do(t, http.MethodGet, "/api/v1/me", "", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", status)
	}
}

func TestAuthShouldReturn401WhenTokenIsInvalid(t *testing.T) {
	env := apitest.New(t)
	expired := apitest.Claims("sub-expired", apitest.AllowedEmail)
	expired["exp"] = time.Now().Add(-time.Minute).Unix()
	wrongIssuer := apitest.Claims("sub-iss", apitest.AllowedEmail)
	wrongIssuer["iss"] = "https://evil.example.com/auth/v1"

	cases := map[string]string{
		"untrusted signature": env.SignWithUntrustedKey(t, apitest.Claims("sub-sig", apitest.AllowedEmail)),
		"expired":             env.Sign(t, expired),
		"wrong issuer":        env.Sign(t, wrongIssuer),
		"garbage":             "not-a-jwt",
	}
	for name, token := range cases {
		status, _ := env.Do(t, http.MethodGet, "/api/v1/me", token, nil)
		if status != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want 401", name, status)
		}
	}
}

func TestAuthShouldReturn403WhenEmailIsNotAllowlisted(t *testing.T) {
	env := apitest.New(t)
	status, _ := env.Do(t, http.MethodGet, "/api/v1/me", env.Token(t, "sub-stranger", "stranger@example.com"), nil)
	if status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", status)
	}
	var users int
	if err := env.Pool.QueryRow(context.Background(), `SELECT count(*) FROM users`).Scan(&users); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if users != 0 {
		t.Fatalf("users = %d, want 0 (rejected callers are not stored)", users)
	}
}

func TestAuthShouldLetAllowlistedUserThroughAndUpsertOnce(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-learner", apitest.AllowedEmail)

	status, body := env.Do(t, http.MethodGet, "/api/v1/me", token, nil)
	if status != http.StatusOK {
		t.Fatalf("first call status = %d (%s), want 200", status, body)
	}
	var first meResponse
	apitest.Decode(t, body, &first)
	if first.Email != apitest.AllowedEmail || first.ID == "" {
		t.Fatalf("me = %+v, want email %s and an id", first, apitest.AllowedEmail)
	}

	_, body = env.Do(t, http.MethodGet, "/api/v1/me", token, nil)
	var second meResponse
	apitest.Decode(t, body, &second)
	if second.ID != first.ID {
		t.Fatalf("second call id = %s, want same user %s", second.ID, first.ID)
	}
	var rows int
	if err := env.Pool.QueryRow(context.Background(), `SELECT count(*) FROM users WHERE auth_sub = 'sub-learner'`).Scan(&rows); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if rows != 1 {
		t.Fatalf("users for sub = %d, want 1", rows)
	}
}
