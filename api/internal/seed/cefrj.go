// Package seed loads CEFR word lists (CEFR-J A1–B2, Octanove C1–C2) into senses.
package seed

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// maxLoggedLines keeps the warning readable when a whole file is malformed.
const maxLoggedLines = 20

var validLevels = map[string]bool{"A1": true, "A2": true, "B1": true, "B2": true, "C1": true, "C2": true}

type Result struct {
	Loaded  int
	Skipped int
}

// Load upserts one sense per valid row. Re-running only refreshes level/variants, so enrichment and cards survive.
func Load(ctx context.Context, pool *pgxpool.Pool, r io.Reader, source string) (Result, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return Result{}, fmt.Errorf("read header: %w", err)
	}
	col, err := columns(header)
	if err != nil {
		return Result{}, err
	}

	var res Result
	var skippedLines []int
	for line := 2; ; line++ {
		rec, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return res, fmt.Errorf("read line %d: %w", line, err)
		}
		raw := strings.TrimSpace(field(rec, col["headword"]))
		level := strings.TrimSpace(field(rec, col["CEFR"]))
		variants := splitVariants(raw)
		if len(variants) == 0 || !validLevels[level] {
			res.Skipped++
			skippedLines = append(skippedLines, line)
			continue
		}
		pos := strings.TrimSpace(field(rec, col["pos"]))
		if _, err := pool.Exec(ctx, `
			INSERT INTO senses (headword, pos, cefr_level, variants, source)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (headword, pos) WHERE owner_user_id IS NULL AND deleted_at IS NULL
			DO UPDATE SET cefr_level = EXCLUDED.cefr_level, variants = EXCLUDED.variants, source = EXCLUDED.source
			WHERE senses.cefr_level IS DISTINCT FROM EXCLUDED.cefr_level
			   OR senses.variants IS DISTINCT FROM EXCLUDED.variants
			   OR senses.source IS DISTINCT FROM EXCLUDED.source`,
			variants[0], pos, level, variants, source); err != nil {
			return res, fmt.Errorf("upsert line %d (%s): %w", line, raw, err)
		}
		res.Loaded++
	}
	if res.Skipped > 0 {
		slog.Warn("seed skipped invalid rows", "source", source, "skipped", res.Skipped, "lines", firstN(skippedLines, maxLoggedLines))
	}
	return res, nil
}

func columns(header []string) (map[string]int, error) {
	col := map[string]int{}
	for i, h := range header {
		col[strings.TrimPrefix(strings.TrimSpace(h), "\ufeff")] = i
	}
	for _, need := range []string{"headword", "pos", "CEFR"} {
		if _, ok := col[need]; !ok {
			return nil, fmt.Errorf("csv header missing %q", need)
		}
	}
	return col, nil
}

func field(rec []string, i int) string {
	if i < len(rec) {
		return rec[i]
	}
	return ""
}

// "adviser/advisor" → ["adviser", "advisor"]; the first spelling is the headword.
func splitVariants(raw string) []string {
	var out []string
	for _, v := range strings.Split(raw, "/") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func firstN(xs []int, n int) []int {
	if len(xs) > n {
		return xs[:n]
	}
	return xs
}
