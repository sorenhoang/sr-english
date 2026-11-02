// Package enrich fills IPA, Vietnamese meaning, definition and examples for senses using an LLM.
package enrich

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Completer is the only thing enrich needs from an LLM; tests pass a fake.
type Completer interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

type Result struct {
	Updated int
	Failed  int
}

type fields struct {
	IPAUK        string `json:"ipa_uk"`
	IPAUS        string `json:"ipa_us"`
	VIMeaning    string `json:"vi_meaning"`
	ENDefinition string `json:"en_definition"`
	ExampleEN    string `json:"example_en"`
	ExampleVI    string `json:"example_vi"`
}

func (f fields) missing() []string {
	var out []string
	for _, kv := range [][2]string{
		{"ipa_uk", f.IPAUK}, {"ipa_us", f.IPAUS}, {"vi_meaning", f.VIMeaning},
		{"en_definition", f.ENDefinition}, {"example_en", f.ExampleEN}, {"example_vi", f.ExampleVI},
	} {
		if strings.TrimSpace(kv[1]) == "" {
			out = append(out, kv[0])
		}
	}
	return out
}

type sense struct {
	id, headword, pos string
}

// Run enriches up to limit senses of one level that have no vi_meaning yet; a bad answer only skips that sense.
func Run(ctx context.Context, pool *pgxpool.Pool, llm Completer, level string, limit int) (Result, error) {
	if limit <= 0 {
		return Result{}, fmt.Errorf("limit must be positive, got %d", limit)
	}
	rows, err := pool.Query(ctx, `
		SELECT id, headword, pos FROM senses
		WHERE deleted_at IS NULL AND cefr_level = $1 AND vi_meaning IS NULL
		ORDER BY headword, pos
		LIMIT $2`, level, limit)
	if err != nil {
		return Result{}, fmt.Errorf("select senses: %w", err)
	}
	var todo []sense
	for rows.Next() {
		var s sense
		if err := rows.Scan(&s.id, &s.headword, &s.pos); err != nil {
			rows.Close()
			return Result{}, fmt.Errorf("scan sense: %w", err)
		}
		todo = append(todo, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Result{}, fmt.Errorf("iterate senses: %w", err)
	}

	var res Result
	for _, s := range todo {
		f, err := ask(ctx, llm, s, level)
		if err != nil {
			slog.Warn("enrich skipped sense", "headword", s.headword, "pos", s.pos, "err", err)
			res.Failed++
			continue
		}
		tag, err := pool.Exec(ctx, `
			UPDATE senses SET ipa_uk = $2, ipa_us = $3, vi_meaning = $4, en_definition = $5,
			       example_en = $6, example_vi = $7, reviewed = FALSE
			WHERE id = $1 AND deleted_at IS NULL AND vi_meaning IS NULL`,
			s.id, f.IPAUK, f.IPAUS, f.VIMeaning, f.ENDefinition, f.ExampleEN, f.ExampleVI)
		if err != nil {
			return res, fmt.Errorf("update sense %s: %w", s.headword, err)
		}
		if tag.RowsAffected() == 0 {
			slog.Info("enrich skipped sense changed meanwhile", "headword", s.headword)
			continue
		}
		res.Updated++
	}
	return res, nil
}

func ask(ctx context.Context, llm Completer, s sense, level string) (fields, error) {
	answer, err := llm.Complete(ctx, prompt(s, level))
	if err != nil {
		return fields{}, fmt.Errorf("llm: %w", err)
	}
	var f fields
	if err := json.Unmarshal([]byte(extractJSON(answer)), &f); err != nil {
		return fields{}, fmt.Errorf("parse answer: %w", err)
	}
	if m := f.missing(); len(m) > 0 {
		return fields{}, fmt.Errorf("answer missing %s", strings.Join(m, ", "))
	}
	return f, nil
}

func prompt(s sense, level string) string {
	return fmt.Sprintf(`You write entries for a Vietnamese learner's English vocabulary app.
Word: %q
Part of speech: %s
CEFR level: %s

Reply with ONLY a JSON object with these string fields:
- "ipa_uk": British IPA in slashes
- "ipa_us": American IPA in slashes
- "vi_meaning": short, natural Vietnamese meaning for this part of speech
- "en_definition": simple English definition at or below the word's level
- "example_en": one natural example sentence using the word
- "example_vi": Vietnamese translation of the example`, s.headword, s.pos, level)
}

// Models sometimes wrap JSON in prose or code fences; keep only the outermost object.
func extractJSON(s string) string {
	start, end := strings.Index(s, "{"), strings.LastIndex(s, "}")
	if start < 0 || end < start {
		return s
	}
	return s[start : end+1]
}
