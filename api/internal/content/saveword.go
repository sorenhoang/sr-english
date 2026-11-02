package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/sorenhoang/sr-english/api/internal/text"
)

var ErrBadToken = errors.New("token_index does not point at a word in this passage")

type SavedWord struct {
	CardID   string `json:"card_id"`
	SenseID  string `json:"sense_id"`
	Headword string `json:"headword"`
}

// customPOS marks self-added words whose part of speech we don't know.
const customPOS = "unknown"

// SaveWord adds the token's sense (or a new private sense for unlisted words) to the user's deck,
// keeping the sentence it came from. Saving an already-saved word returns the existing card.
func (r Repository) SaveWord(ctx context.Context, userID, passageID string, index int) (SavedWord, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return SavedWord{}, fmt.Errorf("begin save word: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			slog.Warn("rollback save word", "err", err)
		}
	}()

	var tokensJSON, sentencesJSON string
	err = tx.QueryRow(ctx, `SELECT tokens::text, sentences::text FROM passages WHERE id = $1 AND deleted_at IS NULL`, passageID).
		Scan(&tokensJSON, &sentencesJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return SavedWord{}, ErrPassageNotFound
	}
	if err != nil {
		return SavedWord{}, fmt.Errorf("load passage: %w", err)
	}
	var tokens []text.Token
	var sentences []string
	if err := json.Unmarshal([]byte(tokensJSON), &tokens); err != nil {
		return SavedWord{}, fmt.Errorf("decode tokens: %w", err)
	}
	if err := json.Unmarshal([]byte(sentencesJSON), &sentences); err != nil {
		return SavedWord{}, fmt.Errorf("decode sentences: %w", err)
	}
	if index < 0 || index >= len(tokens) || tokens[index].Kind != text.KindWord {
		return SavedWord{}, ErrBadToken
	}
	tok := tokens[index]
	sentence := ""
	if tok.Sentence >= 0 && tok.Sentence < len(sentences) {
		sentence = sentences[tok.Sentence]
	}

	w := SavedWord{SenseID: tok.SenseID, Headword: tok.Headword}
	if w.SenseID != "" {
		var active bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM senses WHERE id = $1 AND deleted_at IS NULL)`, w.SenseID).Scan(&active); err != nil {
			return SavedWord{}, fmt.Errorf("check sense: %w", err)
		}
		if !active {
			return SavedWord{}, ErrBadToken
		}
	}
	if w.SenseID == "" {
		w.Headword = strings.ToLower(strings.ReplaceAll(tok.Text, "’", "'"))
		if err := tx.QueryRow(ctx, `
			INSERT INTO senses (headword, pos, variants, source, owner_user_id) VALUES ($1, $2, ARRAY[$1], 'custom', $3)
			ON CONFLICT (owner_user_id, headword) WHERE owner_user_id IS NOT NULL AND deleted_at IS NULL
			DO UPDATE SET headword = EXCLUDED.headword -- no-op update so RETURNING yields the existing id
			RETURNING id`, w.Headword, customPOS, userID).Scan(&w.SenseID); err != nil {
			return SavedWord{}, fmt.Errorf("create custom sense: %w", err)
		}
	}

	// Same lock key as vocab.Review so saving and a first review can't both create the card.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1 || ':' || $2, 0))`, userID, w.SenseID); err != nil {
		return SavedWord{}, fmt.Errorf("lock card: %w", err)
	}
	err = tx.QueryRow(ctx, `SELECT id FROM cards WHERE user_id = $1 AND sense_id = $2 AND deleted_at IS NULL`, userID, w.SenseID).Scan(&w.CardID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			INSERT INTO cards (user_id, sense_id, due, context_sentence, context_passage_id)
			VALUES ($1, $2, now(), $3, $4) RETURNING id`, userID, w.SenseID, sentence, passageID).Scan(&w.CardID)
	}
	if err != nil {
		return SavedWord{}, fmt.Errorf("save card: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return SavedWord{}, fmt.Errorf("commit save word: %w", err)
	}
	return w, nil
}
