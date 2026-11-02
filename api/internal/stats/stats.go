// Package stats computes the streak and today's counts, always in the user's current timezone.
package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// maxStreakDays bounds the date scan; ten years of daily study is plenty.
const maxStreakDays = 3650

type Stats struct {
	Streak       int `json:"streak"`
	TodayNew     int `json:"today_new"`
	TodayReviews int `json:"today_reviews"`
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return Repository{pool: pool}
}

// Get derives everything from reviews on each call, so changing timezone recomputes the streak.
func (r Repository) Get(ctx context.Context, userID string, now time.Time, loc *time.Location) (Stats, error) {
	var s Stats
	localNow := now.In(loc)
	dayStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, loc)
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE is_new), count(*) FILTER (WHERE NOT is_new)
		FROM reviews WHERE user_id = $1 AND deleted_at IS NULL AND reviewed_at >= $2`,
		userID, dayStart).Scan(&s.TodayNew, &s.TodayReviews); err != nil {
		return Stats{}, fmt.Errorf("count today: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT (reviewed_at AT TIME ZONE $2)::date AS day
		FROM reviews WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY day DESC
		LIMIT $3`, userID, loc.String(), maxStreakDays)
	if err != nil {
		return Stats{}, fmt.Errorf("query study days: %w", err)
	}
	defer rows.Close()
	var days []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			return Stats{}, fmt.Errorf("scan study day: %w", err)
		}
		days = append(days, d)
	}
	if err := rows.Err(); err != nil {
		return Stats{}, fmt.Errorf("iterate study days: %w", err)
	}
	s.Streak = streak(days, civilDate(localNow))
	return s, nil
}

// streak counts consecutive study days ending today, or yesterday if today has no study yet.
func streak(daysDesc []time.Time, today time.Time) int {
	if len(daysDesc) == 0 {
		return 0
	}
	expect := today
	if civilDate(daysDesc[0]).Equal(today.AddDate(0, 0, -1)) {
		expect = today.AddDate(0, 0, -1)
	}
	n := 0
	for _, d := range daysDesc {
		if !civilDate(d).Equal(expect) {
			break
		}
		n++
		expect = expect.AddDate(0, 0, -1)
	}
	return n
}

// civilDate drops time and zone so dates from Postgres (UTC midnight) and Go compare by calendar day.
func civilDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
