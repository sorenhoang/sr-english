package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sorenhoang/sr-english/api/internal/text"
)

var ErrPassageNotFound = errors.New("passage not found")

type Summary struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	CEFRLevel   string     `json:"cefr_level"`
	Topics      []string   `json:"topics"`
	Kinds       []string   `json:"kinds"`
	SourceURL   string     `json:"source_url"`
	License     string     `json:"license"`
	AudioURL    *string    `json:"audio_url"`
	PublishedAt *time.Time `json:"published_at"`
}

// SavedToken is a passage token plus whether its sense is already in the user's deck.
type SavedToken struct {
	text.Token
	Saved     bool    `json:"saved"`
	VIMeaning *string `json:"vi_meaning,omitempty"`
}

type Passage struct {
	Summary
	Body      string       `json:"body"`
	Tokens    []SavedToken `json:"tokens"`
	Sentences []string     `json:"sentences"`
}

type Filter struct {
	Level, Topic, Kind string
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return Repository{pool: pool}
}

const summaryColumns = `p.id, p.title, p.cefr_level, p.topics, p.kinds, p.source_url, p.license, p.audio_url, p.published_at`

func scanSummary(row pgx.Row) (Summary, error) {
	var s Summary
	err := row.Scan(&s.ID, &s.Title, &s.CEFRLevel, &s.Topics, &s.Kinds, &s.SourceURL, &s.License, &s.AudioURL, &s.PublishedAt)
	return s, err
}

// List filters by level, topic and kind ("" = any), newest first.
func (r Repository) List(ctx context.Context, f Filter, offset int64, limit int) ([]Summary, int64, error) {
	const where = `p.deleted_at IS NULL AND ($1 = '' OR p.cefr_level = $1) AND ($2 = '' OR $2 = ANY(p.topics))
		AND ($3 = '' OR $3 = ANY(p.kinds))`
	var total int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM passages p WHERE `+where, f.Level, f.Topic, f.Kind).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count passages: %w", err)
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+summaryColumns+` FROM passages p WHERE `+where+`
		ORDER BY p.published_at DESC NULLS LAST, p.id
		OFFSET $4 LIMIT $5`, f.Level, f.Topic, f.Kind, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("query passages: %w", err)
	}
	defer rows.Close()
	items := []Summary{}
	for rows.Next() {
		s, err := scanSummary(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan passage: %w", err)
		}
		items = append(items, s)
	}
	return items, total, rows.Err()
}

// Get returns a passage with each token flagged when the user already has a card for its sense.
func (r Repository) Get(ctx context.Context, userID, id string) (Passage, error) {
	var p Passage
	var tokens, sentences string
	row := r.pool.QueryRow(ctx, `
		SELECT `+summaryColumns+`, p.body, p.tokens::text, p.sentences::text
		FROM passages p WHERE p.id = $1 AND p.deleted_at IS NULL`, id)
	err := row.Scan(&p.ID, &p.Title, &p.CEFRLevel, &p.Topics, &p.Kinds, &p.SourceURL, &p.License, &p.AudioURL,
		&p.PublishedAt, &p.Body, &tokens, &sentences)
	if errors.Is(err, pgx.ErrNoRows) {
		return Passage{}, ErrPassageNotFound
	}
	if err != nil {
		return Passage{}, fmt.Errorf("get passage: %w", err)
	}
	var raw []text.Token
	if err := json.Unmarshal([]byte(tokens), &raw); err != nil {
		return Passage{}, fmt.Errorf("decode tokens: %w", err)
	}
	if err := json.Unmarshal([]byte(sentences), &p.Sentences); err != nil {
		return Passage{}, fmt.Errorf("decode sentences: %w", err)
	}

	known, err := r.savedSenses(ctx, userID, raw)
	if err != nil {
		return Passage{}, err
	}
	meanings, err := r.meanings(ctx, raw)
	if err != nil {
		return Passage{}, err
	}
	custom, err := r.savedCustomWords(ctx, userID)
	if err != nil {
		return Passage{}, err
	}
	p.Tokens = make([]SavedToken, len(raw))
	for i, t := range raw {
		saved := known[t.SenseID]
		if t.SenseID == "" && t.Kind == text.KindWord {
			saved = custom[strings.ToLower(strings.ReplaceAll(t.Text, "’", "'"))]
		}
		st := SavedToken{Token: t, Saved: saved}
		if m, ok := meanings[t.SenseID]; ok {
			st.VIMeaning = &m
		}
		p.Tokens[i] = st
	}
	return p, nil
}

func senseIDs(tokens []text.Token) []string {
	seen := map[string]bool{}
	var ids []string
	for _, t := range tokens {
		if t.SenseID != "" && !seen[t.SenseID] {
			seen[t.SenseID] = true
			ids = append(ids, t.SenseID)
		}
	}
	return ids
}

// meanings returns the Vietnamese meaning of each public sense in the passage, for the tap-to-save sheet.
func (r Repository) meanings(ctx context.Context, tokens []text.Token) (map[string]string, error) {
	out := map[string]string{}
	ids := senseIDs(tokens)
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, vi_meaning FROM senses
		WHERE id = ANY($1::uuid[]) AND deleted_at IS NULL AND owner_user_id IS NULL AND vi_meaning IS NOT NULL`, ids)
	if err != nil {
		return nil, fmt.Errorf("query meanings: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, m string
		if err := rows.Scan(&id, &m); err != nil {
			return nil, fmt.Errorf("scan meaning: %w", err)
		}
		out[id] = m
	}
	return out, rows.Err()
}

// savedCustomWords returns the headwords of the user's self-added senses that have a card.
func (r Repository) savedCustomWords(ctx context.Context, userID string) (map[string]bool, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT s.headword FROM senses s
		JOIN cards c ON c.sense_id = s.id AND c.user_id = $1 AND c.deleted_at IS NULL
		WHERE s.owner_user_id = $1 AND s.deleted_at IS NULL`, userID)
	if err != nil {
		return nil, fmt.Errorf("query custom words: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var h string
		if err := rows.Scan(&h); err != nil {
			return nil, fmt.Errorf("scan custom word: %w", err)
		}
		out[h] = true
	}
	return out, rows.Err()
}

func (r Repository) savedSenses(ctx context.Context, userID string, tokens []text.Token) (map[string]bool, error) {
	ids := senseIDs(tokens)
	known := map[string]bool{}
	if len(ids) == 0 {
		return known, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT sense_id::text FROM cards WHERE user_id = $1 AND deleted_at IS NULL AND sense_id = ANY($2::uuid[])`, userID, ids)
	if err != nil {
		return nil, fmt.Errorf("query saved senses: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan saved sense: %w", err)
		}
		known[id] = true
	}
	return known, rows.Err()
}

// Complete marks a passage done for the user; repeating it is a no-op.
func (r Repository) Complete(ctx context.Context, userID, id string) error {
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM passages WHERE id = $1 AND deleted_at IS NULL)`, id).Scan(&exists); err != nil {
		return fmt.Errorf("check passage: %w", err)
	}
	if !exists {
		return ErrPassageNotFound
	}
	if _, err := r.pool.Exec(ctx, `
		INSERT INTO passage_progress (user_id, passage_id, completed_at) VALUES ($1, $2, now())
		ON CONFLICT (user_id, passage_id) WHERE deleted_at IS NULL DO NOTHING`, userID, id); err != nil {
		return fmt.Errorf("complete passage: %w", err)
	}
	return nil
}

// Suggest returns one unfinished passage of the given kind at the user's level, or none.
func (r Repository) Suggest(ctx context.Context, userID, level, kind string) ([]Summary, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+summaryColumns+` FROM passages p
		WHERE p.deleted_at IS NULL AND p.cefr_level = $2 AND $3 = ANY(p.kinds)
		  AND NOT EXISTS (SELECT 1 FROM passage_progress pp
		                  WHERE pp.user_id = $1 AND pp.passage_id = p.id AND pp.deleted_at IS NULL)
		ORDER BY p.published_at DESC NULLS LAST, p.id
		LIMIT 1`, userID, level, kind)
	s, err := scanSummary(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("suggest passage: %w", err)
	}
	return []Summary{s}, nil
}
