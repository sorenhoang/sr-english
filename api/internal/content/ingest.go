// Package content ingests public-domain VOA Learning English stories as leveled passages.
package content

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sorenhoang/sr-english/api/internal/text"
)

const (
	SourceVOA  = "voa"
	LicenseVOA = "Public domain · learningenglish.voanews.com"

	KindReading   = "reading"
	KindListening = "listening"
)

type Ingester struct {
	pool      *pgxpool.Pool
	analyzer  *text.Analyzer
	threshold float64
}

// NewIngester builds the word-level analyzer from the public word list once.
func NewIngester(ctx context.Context, pool *pgxpool.Pool, coverageThreshold float64) (*Ingester, error) {
	analyzer, err := LoadAnalyzer(ctx, pool)
	if err != nil {
		return nil, err
	}
	return &Ingester{pool: pool, analyzer: analyzer, threshold: coverageThreshold}, nil
}

// LoadAnalyzer reads every public sense into a text.Analyzer.
func LoadAnalyzer(ctx context.Context, pool *pgxpool.Pool) (*text.Analyzer, error) {
	rows, err := pool.Query(ctx, `
		SELECT id, headword, variants, pos, COALESCE(cefr_level, '') FROM senses
		WHERE deleted_at IS NULL AND owner_user_id IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("load word list: %w", err)
	}
	defer rows.Close()
	var entries []text.Entry
	for rows.Next() {
		var e text.Entry
		if err := rows.Scan(&e.SenseID, &e.Headword, &e.Variants, &e.POS, &e.Level); err != nil {
			return nil, fmt.Errorf("scan word: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate words: %w", err)
	}
	return text.NewAnalyzer(entries)
}

// Ingest parses one VOA page and upserts it by source URL; returns the passage id.
func (i *Ingester) Ingest(ctx context.Context, sourceURL, rawHTML string) (string, error) {
	p, err := parseVOA(rawHTML)
	if err != nil {
		return "", fmt.Errorf("parse %s: %w", sourceURL, err)
	}
	analysis := i.analyzer.Analyze(p.Body)
	tokens, err := json.Marshal(nonNil(analysis.Tokens))
	if err != nil {
		return "", fmt.Errorf("encode tokens: %w", err)
	}
	sentences, err := json.Marshal(nonNil(analysis.Sentences))
	if err != nil {
		return "", fmt.Errorf("encode sentences: %w", err)
	}
	kinds := []string{KindReading}
	var audio *string
	if p.AudioURL != "" {
		kinds = append(kinds, KindListening)
		audio = &p.AudioURL
	}

	// JSON goes in as text: under the simple protocol pgx would send []byte as bytea.
	var id string
	err = i.pool.QueryRow(ctx, `
		INSERT INTO passages (title, body, cefr_level, topics, kinds, source, source_url, license, audio_url,
		                      published_at, tokens, sentences)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (source_url) WHERE deleted_at IS NULL DO UPDATE SET
			title = EXCLUDED.title, body = EXCLUDED.body, cefr_level = EXCLUDED.cefr_level, topics = EXCLUDED.topics,
			kinds = EXCLUDED.kinds, audio_url = EXCLUDED.audio_url, published_at = EXCLUDED.published_at,
			tokens = EXCLUDED.tokens, sentences = EXCLUDED.sentences
		RETURNING id`,
		p.Title, p.Body, analysis.Level(i.threshold), assignTopics(p.Categories, p.Title, p.Body), kinds,
		SourceVOA, sourceURL, LicenseVOA, audio, p.Published, string(tokens), string(sentences)).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("upsert passage %s: %w", sourceURL, err)
	}
	return id, nil
}

func nonNil[T any](xs []T) []T {
	if xs == nil {
		return []T{}
	}
	return xs
}
