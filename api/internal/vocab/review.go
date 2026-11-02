package vocab

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"
)

var ErrSenseNotFound = errors.New("sense not found")

type Card struct {
	CardID        string    `json:"card_id"`
	SenseID       string    `json:"sense_id"`
	Due           time.Time `json:"due"`
	Stability     float64   `json:"stability"`
	Difficulty    float64   `json:"difficulty"`
	ElapsedDays   uint64    `json:"elapsed_days"`
	ScheduledDays uint64    `json:"scheduled_days"`
	Reps          uint64    `json:"reps"`
	Lapses        uint64    `json:"lapses"`
	State         int       `json:"state"`
	LastReview    time.Time `json:"last_review"`
}

// NewScheduler uses go-fsrs defaults with the configured retention; fuzz stays off so schedules are reproducible.
func NewScheduler(desiredRetention float64) *fsrs.FSRS {
	p := fsrs.DefaultParam()
	p.RequestRetention = desiredRetention
	p.EnableFuzz = false
	return fsrs.NewFSRS(p)
}

// Review records one answer and reschedules the user's card (creating it on first sight) in one transaction.
func (r Repository) Review(ctx context.Context, scheduler *fsrs.FSRS, userID, senseID string, rating fsrs.Rating) (Card, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Card{}, fmt.Errorf("begin review: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.Warn("rollback review", "err", err)
		}
	}()

	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM senses WHERE id = $1 AND deleted_at IS NULL
		               AND (owner_user_id IS NULL OR owner_user_id = $2))`, senseID, userID).Scan(&exists); err != nil {
		return Card{}, fmt.Errorf("check sense: %w", err)
	}
	if !exists {
		return Card{}, ErrSenseNotFound
	}

	// Serialize reviews of one (user, sense): FOR UPDATE can't lock a card that doesn't exist yet,
	// so two first reviews would otherwise both INSERT and one would hit the unique index.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1 || ':' || $2, 0))`, userID, senseID); err != nil {
		return Card{}, fmt.Errorf("lock card: %w", err)
	}
	// Taken after the lock so a queued review never predates the one it waited for.
	// Postgres keeps microseconds; truncating keeps the returned card identical to the stored one.
	now := time.Now().UTC().Truncate(time.Microsecond)

	var cardID string
	var c fsrs.Card
	var state int16
	var lastReview *time.Time
	err = tx.QueryRow(ctx, `
		SELECT id, due, stability, difficulty, elapsed_days, scheduled_days, reps, lapses, state, last_review
		FROM cards WHERE user_id = $1 AND sense_id = $2 AND deleted_at IS NULL
		FOR UPDATE`, userID, senseID).
		Scan(&cardID, &c.Due, &c.Stability, &c.Difficulty, &c.ElapsedDays, &c.ScheduledDays, &c.Reps, &c.Lapses, &state, &lastReview)
	isNew := errors.Is(err, pgx.ErrNoRows)
	switch {
	case isNew:
		c = fsrs.NewCard()
	case err != nil:
		return Card{}, fmt.Errorf("load card: %w", err)
	default:
		c.State = fsrs.State(state)
		if lastReview != nil {
			c.LastReview = *lastReview
		}
	}

	next := scheduler.Next(c, now, rating).Card
	if isNew {
		err = tx.QueryRow(ctx, `
			INSERT INTO cards (user_id, sense_id, due, stability, difficulty, elapsed_days, scheduled_days, reps, lapses, state, last_review)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`,
			userID, senseID, next.Due, next.Stability, next.Difficulty, next.ElapsedDays, next.ScheduledDays,
			next.Reps, next.Lapses, int16(next.State), next.LastReview).Scan(&cardID)
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE cards SET due = $2, stability = $3, difficulty = $4, elapsed_days = $5, scheduled_days = $6,
			       reps = $7, lapses = $8, state = $9, last_review = $10
			WHERE id = $1 AND user_id = $11`,
			cardID, next.Due, next.Stability, next.Difficulty, next.ElapsedDays, next.ScheduledDays,
			next.Reps, next.Lapses, int16(next.State), next.LastReview, userID)
	}
	if err != nil {
		return Card{}, fmt.Errorf("save card: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO reviews (user_id, card_id, sense_id, rating, is_new, reviewed_at) VALUES ($1, $2, $3, $4, $5, $6)`,
		userID, cardID, senseID, int16(rating), isNew, now); err != nil {
		return Card{}, fmt.Errorf("insert review: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Card{}, fmt.Errorf("commit review: %w", err)
	}
	return Card{
		CardID: cardID, SenseID: senseID, Due: next.Due, Stability: next.Stability, Difficulty: next.Difficulty,
		ElapsedDays: next.ElapsedDays, ScheduledDays: next.ScheduledDays, Reps: next.Reps, Lapses: next.Lapses,
		State: int(next.State), LastReview: next.LastReview,
	}, nil
}
