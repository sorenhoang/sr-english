package httpapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/sorenhoang/sr-english/api/internal/apitest"
)

type savedWord struct {
	CardID   string `json:"card_id"`
	SenseID  string `json:"sense_id"`
	Headword string `json:"headword"`
}

func passageWithWords(t *testing.T, env *apitest.Env) (passageID, knownSense string) {
	t.Helper()
	knownSense = env.InsertSense(t, "mitigate", "verb", "C1", true)
	passageID = env.InsertPassage(t, "outage", "B2", []string{"communication-and-technology"}, "", []apitest.PassageToken{
		{Text: "We", Start: 0, End: 2, Kind: "word", Counted: true, Sentence: 0},
		{Text: "rested", Start: 3, End: 9, Kind: "word", Counted: true, Sentence: 0},
		{Text: "Then", Start: 11, End: 15, Kind: "word", Counted: true, Sentence: 1},
		{Text: "we", Start: 16, End: 18, Kind: "word", Counted: true, Sentence: 1},
		{Text: "mitigated", Start: 19, End: 28, Kind: "word", SenseID: knownSense, Headword: "mitigate", Level: "C1", Counted: true, Sentence: 1},
		{Text: "Kubernetes", Start: 33, End: 43, Kind: "word", Counted: true, Sentence: 1},
	}, []string{"We rested.", "Then we mitigated the Kubernetes outage."})
	return passageID, knownSense
}

func saveWord(t *testing.T, env *apitest.Env, token, passageID string, index int) (int, savedWord) {
	t.Helper()
	status, raw := env.Do(t, http.MethodPost, "/api/v1/passage/save-word", token, map[string]any{"passage_id": passageID, "token_index": index})
	var w savedWord
	if status == http.StatusOK {
		apitest.Decode(t, raw, &w)
	}
	return status, w
}

func TestSaveWordShouldCreateCardWithContextSentenceAndPassage(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	userID := env.UserID(t, token)
	passage, sense := passageWithWords(t, env)

	status, w := saveWord(t, env, token, passage, 4)
	if status != http.StatusOK || w.SenseID != sense {
		t.Fatalf("save status %d word %+v, want 200 for sense %s", status, w, sense)
	}
	var sentence, ctxPassage string
	if err := env.Pool.QueryRow(context.Background(), `
		SELECT context_sentence, context_passage_id::text FROM cards WHERE user_id = $1 AND sense_id = $2 AND deleted_at IS NULL`,
		userID, sense).Scan(&sentence, &ctxPassage); err != nil {
		t.Fatalf("load card: %v", err)
	}
	if sentence != "Then we mitigated the Kubernetes outage." || ctxPassage != passage {
		t.Fatalf("card context = %q / %s, want the token's sentence and passage %s", sentence, ctxPassage, passage)
	}
}

func TestSaveWordShouldBeIdempotent(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	userID := env.UserID(t, token)
	passage, sense := passageWithWords(t, env)

	_, first := saveWord(t, env, token, passage, 4)
	status, second := saveWord(t, env, token, passage, 4)
	if status != http.StatusOK || second.CardID != first.CardID {
		t.Fatalf("second save status %d card %s, want 200 and the same card %s", status, second.CardID, first.CardID)
	}
	var n int
	if err := env.Pool.QueryRow(context.Background(), `SELECT count(*) FROM cards WHERE user_id = $1 AND sense_id = $2`, userID, sense).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("cards = %d, want 1", n)
	}
}

func TestSaveWordShouldCreateCustomSenseForWordsOutsideTheList(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	userID := env.UserID(t, token)
	passage, _ := passageWithWords(t, env)

	status, w := saveWord(t, env, token, passage, 5)
	if status != http.StatusOK || w.Headword != "kubernetes" {
		t.Fatalf("save status %d word %+v, want 200 with headword kubernetes", status, w)
	}
	var owner, source string
	if err := env.Pool.QueryRow(context.Background(), `SELECT owner_user_id::text, source FROM senses WHERE id = $1`, w.SenseID).Scan(&owner, &source); err != nil {
		t.Fatalf("load sense: %v", err)
	}
	if owner != userID || source != "custom" {
		t.Fatalf("custom sense owner %s source %s, want owner %s and source custom", owner, source, userID)
	}
	var cards int
	if err := env.Pool.QueryRow(context.Background(), `SELECT count(*) FROM cards WHERE user_id = $1 AND sense_id = $2`, userID, w.SenseID).Scan(&cards); err != nil {
		t.Fatalf("count cards: %v", err)
	}
	if cards != 1 {
		t.Fatalf("cards for custom sense = %d, want 1", cards)
	}
}

func TestSaveWordShouldRejectBadIndexAndUnknownPassage(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	passage, _ := passageWithWords(t, env)

	for _, idx := range []int{-1, 99} {
		if status, _ := saveWord(t, env, token, passage, idx); status != http.StatusBadRequest {
			t.Errorf("token_index %d: status = %d, want 400", idx, status)
		}
	}
	if status, _ := saveWord(t, env, token, "6f1c1d1e-0000-4000-8000-000000000000", 0); status != http.StatusNotFound {
		t.Errorf("unknown passage: status = %d, want 404", status)
	}
}

func TestSaveWordShouldKeepCardsPrivateToTheSaver(t *testing.T) {
	env := apitest.New(t)
	tokenA := env.Token(t, "sub-a", apitest.AllowedEmail)
	tokenB := env.Token(t, "sub-b", apitest.OtherEmail)
	env.UserID(t, tokenA)
	userB := env.UserID(t, tokenB)
	passage, _ := passageWithWords(t, env)

	if status, _ := saveWord(t, env, tokenA, passage, 4); status != http.StatusOK {
		t.Fatalf("save status = %d", status)
	}
	var n int
	if err := env.Pool.QueryRow(context.Background(), `SELECT count(*) FROM cards WHERE user_id = $1`, userB).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Fatalf("user B cards = %d, want 0", n)
	}
	_, page := listWords(t, env, tokenB, map[string]any{"offset": 0, "limit": 10})
	if page.Total != 0 {
		t.Fatalf("user B words = %+v, want none", page)
	}
}
