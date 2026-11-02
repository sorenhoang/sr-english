// Package settings stores each user's study preferences.
package settings

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sorenhoang/sr-english/api/internal/config"
)

const (
	DefaultLevel    = "A1"
	DefaultTimezone = "UTC"
)

var Levels = []string{"A1", "A2", "B1", "B2", "C1", "C2"}

type Settings struct {
	Level    string `json:"level"`
	DailyNew int    `json:"daily_new"`
	Timezone string `json:"timezone"`
}

type Repository struct {
	pool     *pgxpool.Pool
	learning config.Learning
}

func NewRepository(pool *pgxpool.Pool, learning config.Learning) Repository {
	return Repository{pool: pool, learning: learning}
}

// Get returns stored settings, or defaults when the user never saved any.
func (r Repository) Get(ctx context.Context, userID string) (Settings, error) {
	var s Settings
	err := r.pool.QueryRow(ctx, `
		SELECT level, daily_new, timezone FROM user_settings
		WHERE user_id = $1 AND deleted_at IS NULL`, userID).Scan(&s.Level, &s.DailyNew, &s.Timezone)
	if errors.Is(err, pgx.ErrNoRows) {
		return Settings{Level: DefaultLevel, DailyNew: r.learning.DailyNewDefault, Timezone: DefaultTimezone}, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("get settings: %w", err)
	}
	return s, nil
}

func (r Repository) Save(ctx context.Context, userID string, s Settings) error {
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO user_settings (user_id, level, daily_new, timezone) VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) WHERE deleted_at IS NULL
		DO UPDATE SET level = EXCLUDED.level, daily_new = EXCLUDED.daily_new, timezone = EXCLUDED.timezone`,
		userID, s.Level, s.DailyNew, s.Timezone); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	return nil
}
