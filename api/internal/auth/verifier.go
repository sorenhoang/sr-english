// Package auth verifies Supabase access tokens against the project's JWKS.
package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

type VerifierConfig struct {
	JWKSURL  string
	Issuer   string
	Audience string
}

type Identity struct {
	Sub   string
	Email string
}

type Verifier struct {
	keys   keyfunc.Keyfunc
	parser *jwt.Parser
}

// Supabase signs user tokens with asymmetric keys (ES256 by default, RS256 for older projects).
var validMethods = []string{"ES256", "RS256"}

func NewVerifier(ctx context.Context, cfg VerifierConfig) (*Verifier, error) {
	keys, err := keyfunc.NewDefaultCtx(ctx, []string{cfg.JWKSURL})
	if err != nil {
		return nil, fmt.Errorf("load jwks: %w", err)
	}
	return &Verifier{
		keys: keys,
		parser: jwt.NewParser(
			jwt.WithValidMethods(validMethods),
			jwt.WithIssuer(cfg.Issuer),
			jwt.WithAudience(cfg.Audience),
			jwt.WithExpirationRequired(),
		),
	}, nil
}

func (v *Verifier) Verify(token string) (Identity, error) {
	claims := jwt.MapClaims{}
	if _, err := v.parser.ParseWithClaims(token, claims, v.keys.Keyfunc); err != nil {
		return Identity{}, fmt.Errorf("parse token: %w", err)
	}
	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	if sub == "" || email == "" {
		return Identity{}, errors.New("token missing sub or email")
	}
	return Identity{Sub: sub, Email: email}, nil
}
