package apitest

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// UserID calls /me so the user row exists and returns its id.
func (e *Env) UserID(t *testing.T, token string) string {
	t.Helper()
	status, body := e.Do(t, http.MethodGet, "/api/v1/me", token, nil)
	if status != http.StatusOK {
		t.Fatalf("apitest: /me status %d: %s", status, body)
	}
	var me struct {
		ID string `json:"id"`
	}
	Decode(t, body, &me)
	return me.ID
}

// InsertSense adds a CEFR sense; enriched senses get every meaning field filled.
func (e *Env) InsertSense(t *testing.T, headword, pos, level string, enriched bool) string {
	t.Helper()
	var vi, def *string
	if enriched {
		v, d := "nghĩa "+headword, "definition of "+headword
		vi, def = &v, &d
	}
	var id string
	if err := e.Pool.QueryRow(context.Background(), `
		INSERT INTO senses (headword, pos, cefr_level, variants, source, vi_meaning, en_definition,
		                    ipa_uk, ipa_us, example_en, example_vi)
		VALUES ($1, $2, $3, ARRAY[$1], 'cefrj', $4, $5, '/uk/', '/us/', 'An example.', 'Một ví dụ.')
		RETURNING id`, headword, pos, level, vi, def).Scan(&id); err != nil {
		t.Fatalf("apitest: insert sense %s: %v", headword, err)
	}
	return id
}

// InsertCard adds a card for userID with the given due time.
func (e *Env) InsertCard(t *testing.T, userID, senseID string, due time.Time) string {
	t.Helper()
	var id string
	if err := e.Pool.QueryRow(context.Background(), `
		INSERT INTO cards (user_id, sense_id, due, state, reps) VALUES ($1, $2, $3, 2, 1) RETURNING id`,
		userID, senseID, due).Scan(&id); err != nil {
		t.Fatalf("apitest: insert card: %v", err)
	}
	return id
}

// InsertReview records a review at the given time (isNew marks a first-time learn).
func (e *Env) InsertReview(t *testing.T, userID, cardID, senseID string, at time.Time, isNew bool) {
	t.Helper()
	if _, err := e.Pool.Exec(context.Background(), `
		INSERT INTO reviews (user_id, card_id, sense_id, rating, is_new, reviewed_at) VALUES ($1, $2, $3, 3, $4, $5)`,
		userID, cardID, senseID, isNew, at); err != nil {
		t.Fatalf("apitest: insert review: %v", err)
	}
}

// PassageToken is the subset of text.Token fixtures need.
type PassageToken struct {
	Text     string `json:"text"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Start16  int    `json:"start16"`
	End16    int    `json:"end16"`
	Kind     string `json:"kind"`
	SenseID  string `json:"sense_id,omitempty"`
	Headword string `json:"headword,omitempty"`
	Level    string `json:"level,omitempty"`
	Counted  bool   `json:"counted"`
	Sentence int    `json:"sentence"`
}

// InsertPassage adds a passage; audio "" means reading only.
func (e *Env) InsertPassage(t *testing.T, title, level string, topics []string, audio string, tokens []PassageToken, sentences []string) string {
	t.Helper()
	kinds := []string{"reading"}
	var audioURL *string
	if audio != "" {
		kinds = append(kinds, "listening")
		audioURL = &audio
	}
	tok, err := json.Marshal(tokens)
	if err != nil {
		t.Fatalf("apitest: marshal tokens: %v", err)
	}
	sen, err := json.Marshal(sentences)
	if err != nil {
		t.Fatalf("apitest: marshal sentences: %v", err)
	}
	var id string
	if err := e.Pool.QueryRow(context.Background(), `
		INSERT INTO passages (title, body, cefr_level, topics, kinds, source, source_url, license, audio_url, tokens, sentences)
		VALUES ($1, $2, $3, $4, $5, 'voa', $6, 'Public domain · learningenglish.voanews.com', $7, $8, $9) RETURNING id`,
		title, title+" body.", level, topics, kinds, "https://voa.test/"+title, audioURL, string(tok), string(sen)).Scan(&id); err != nil {
		t.Fatalf("apitest: insert passage: %v", err)
	}
	return id
}
