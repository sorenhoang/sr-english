// Package user stores the app's view of a Supabase user.
package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return Repository{pool: pool}
}

// Upsert keys users by Supabase sub so a changed email updates the same row.
func (r Repository) Upsert(ctx context.Context, sub, email string) (User, error) {
	var u User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (auth_sub, email) VALUES ($1, $2)
		ON CONFLICT (auth_sub) WHERE deleted_at IS NULL
		DO UPDATE SET email = EXCLUDED.email WHERE users.email IS DISTINCT FROM EXCLUDED.email
		RETURNING id, email`, sub, email).Scan(&u.ID, &u.Email)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return User{}, fmt.Errorf("upsert user: %w", err)
	}
	// DO UPDATE ... WHERE false returns no row; the user already exists unchanged.
	if err2 := r.pool.QueryRow(ctx, `
		SELECT id, email FROM users WHERE auth_sub = $1 AND deleted_at IS NULL`, sub).Scan(&u.ID, &u.Email); err2 != nil {
		return User{}, fmt.Errorf("select existing user: %w", err2)
	}
	return u, nil
}
