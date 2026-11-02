// Package vocab reads and writes senses, cards and reviews.
package vocab

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Item is a sense as the study screens need it; CardID is nil for words never studied.
type Item struct {
	SenseID      string   `json:"sense_id"`
	CardID       *string  `json:"card_id"`
	Headword     string   `json:"headword"`
	POS          string   `json:"pos"`
	CEFRLevel    string   `json:"cefr_level"`
	IPAUK        *string  `json:"ipa_uk"`
	IPAUS        *string  `json:"ipa_us"`
	VIMeaning    *string  `json:"vi_meaning"`
	ENDefinition *string  `json:"en_definition"`
	ExampleEN    *string  `json:"example_en"`
	ExampleVI    *string  `json:"example_vi"`
	Distractors  []string `json:"distractors"`
}

// distractorCount is how many wrong answers a multiple-choice question shows.
const distractorCount = 3

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return Repository{pool: pool}
}

const itemColumns = `s.id, s.headword, s.pos, COALESCE(s.cefr_level, ''), s.ipa_uk, s.ipa_us,
	s.vi_meaning, s.en_definition, s.example_en, s.example_vi`

// distractorPool is how many candidate meanings are sampled per part of speech before picking per item.
const distractorPool = 50

// Due returns the user's cards due at or before now, oldest first.
func (r Repository) Due(ctx context.Context, userID string, now time.Time, limit int) ([]Item, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, `+itemColumns+`
		FROM cards c JOIN senses s ON s.id = c.sense_id AND s.deleted_at IS NULL
		WHERE c.user_id = $1 AND c.deleted_at IS NULL AND c.due <= $2
		  -- self-added words wait here until they have a meaning to quiz on
		  AND s.vi_meaning IS NOT NULL
		ORDER BY c.due
		LIMIT $3`, userID, now, limit)
	if err != nil {
		return nil, fmt.Errorf("query due cards: %w", err)
	}
	items, err := scanItems(rows)
	if err != nil {
		return nil, err
	}
	return items, r.addDistractors(ctx, items)
}

// New returns enriched senses at level that the user has no card for yet.
func (r Repository) New(ctx context.Context, userID, level string, limit int) ([]Item, error) {
	if limit <= 0 {
		return []Item{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT NULL::uuid, `+itemColumns+`
		FROM senses s
		WHERE s.deleted_at IS NULL AND s.owner_user_id IS NULL AND s.cefr_level = $2 AND s.vi_meaning IS NOT NULL
		  AND NOT EXISTS (SELECT 1 FROM cards c WHERE c.user_id = $1 AND c.sense_id = s.id AND c.deleted_at IS NULL)
		ORDER BY s.headword, s.pos, s.id
		LIMIT $3`, userID, level, limit)
	if err != nil {
		return nil, fmt.Errorf("query new senses: %w", err)
	}
	items, err := scanItems(rows)
	if err != nil {
		return nil, err
	}
	return items, r.addDistractors(ctx, items)
}

// LearnedSince counts first-time learns (is_new reviews) at or after since.
func (r Repository) LearnedSince(ctx context.Context, userID string, since time.Time) (int, error) {
	var n int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM reviews
		WHERE user_id = $1 AND deleted_at IS NULL AND is_new AND reviewed_at >= $2`, userID, since).Scan(&n); err != nil {
		return 0, fmt.Errorf("count learned today: %w", err)
	}
	return n, nil
}

func scanItems(rows pgx.Rows) ([]Item, error) {
	defer rows.Close()
	items := []Item{}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.CardID, &it.SenseID, &it.Headword, &it.POS, &it.CEFRLevel, &it.IPAUK, &it.IPAUS,
			&it.VIMeaning, &it.ENDefinition, &it.ExampleEN, &it.ExampleVI); err != nil {
			return nil, fmt.Errorf("scan item: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate items: %w", err)
	}
	return items, nil
}

// addDistractors samples public meanings once per part of speech (one query per POS, not per item)
// and gives each item distractorCount of them that differ from its own meaning.
func (r Repository) addDistractors(ctx context.Context, items []Item) error {
	pools := map[string][]string{}
	for i := range items {
		pos := items[i].POS
		if _, ok := pools[pos]; !ok {
			candidates, err := r.meaningsFor(ctx, pos)
			if err != nil {
				return err
			}
			pools[pos] = candidates
		}
		items[i].Distractors = pick(pools[pos], items[i])
	}
	return nil
}

func (r Repository) meaningsFor(ctx context.Context, pos string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT vi_meaning FROM (
			SELECT vi_meaning FROM senses
			WHERE deleted_at IS NULL AND owner_user_id IS NULL AND pos = $1 AND vi_meaning IS NOT NULL
			ORDER BY random() LIMIT $2
		) sample`, pos, distractorPool)
	if err != nil {
		return nil, fmt.Errorf("query distractor pool: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, fmt.Errorf("scan distractor: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// pick rotates through the pool from an offset derived from the sense id, skipping the item's own meaning.
func pick(pool []string, it Item) []string {
	out := []string{}
	if len(pool) == 0 {
		return out
	}
	offset := 0
	for _, ch := range it.SenseID {
		offset += int(ch)
	}
	for i := 0; i < len(pool) && len(out) < distractorCount; i++ {
		m := pool[(offset+i)%len(pool)]
		if it.VIMeaning == nil || m != *it.VIMeaning {
			out = append(out, m)
		}
	}
	return out
}

// Word is one entry in the user's "my words" list.
type Word struct {
	CardID    string    `json:"card_id"`
	SenseID   string    `json:"sense_id"`
	Headword  string    `json:"headword"`
	POS       string    `json:"pos"`
	CEFRLevel string    `json:"cefr_level"`
	VIMeaning *string   `json:"vi_meaning"`
	Due       time.Time `json:"due"`
	Reps      int       `json:"reps"`
	State     int       `json:"state"`
}

// ListWords pages through the user's cards, newest first; level "" means all levels.
func (r Repository) ListWords(ctx context.Context, userID, level string, offset int64, limit int) ([]Word, int64, error) {
	const where = `c.user_id = $1 AND c.deleted_at IS NULL AND s.deleted_at IS NULL AND ($2 = '' OR s.cefr_level = $2)`
	var total int64
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM cards c JOIN senses s ON s.id = c.sense_id WHERE `+where, userID, level).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count words: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT c.id, s.id, s.headword, s.pos, COALESCE(s.cefr_level, ''), s.vi_meaning, c.due, c.reps, c.state
		FROM cards c JOIN senses s ON s.id = c.sense_id
		WHERE `+where+`
		ORDER BY c.created_at DESC, c.id
		OFFSET $3 LIMIT $4`, userID, level, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("query words: %w", err)
	}
	defer rows.Close()
	words := []Word{}
	for rows.Next() {
		var w Word
		if err := rows.Scan(&w.CardID, &w.SenseID, &w.Headword, &w.POS, &w.CEFRLevel, &w.VIMeaning, &w.Due, &w.Reps, &w.State); err != nil {
			return nil, 0, fmt.Errorf("scan word: %w", err)
		}
		words = append(words, w)
	}
	return words, total, rows.Err()
}
