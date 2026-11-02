package httpapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/sorenhoang/sr-english/api/internal/apitest"
)

type wordsPage struct {
	Items []struct {
		Headword  string `json:"headword"`
		CEFRLevel string `json:"cefr_level"`
	} `json:"items"`
	Total   int  `json:"total"`
	Offset  int  `json:"offset"`
	Limit   int  `json:"limit"`
	HasMore bool `json:"has_more"`
}

func listWords(t *testing.T, env *apitest.Env, token string, body map[string]any) (int, wordsPage) {
	t.Helper()
	status, raw := env.Do(t, http.MethodPost, "/api/v1/words/list", token, body)
	var p wordsPage
	if status == http.StatusOK {
		apitest.Decode(t, raw, &p)
	}
	return status, p
}

func TestWordsListShouldPageThroughMyCardsAndFilterByLevel(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	userID := env.UserID(t, token)
	for _, w := range []struct{ word, level string }{{"one", "A1"}, {"two", "A1"}, {"three", "B2"}} {
		env.InsertCard(t, userID, env.InsertSense(t, w.word, "noun", w.level, true), time.Now().Add(time.Hour))
	}
	env.InsertSense(t, "unlearned", "noun", "A1", true)

	status, page := listWords(t, env, token, map[string]any{"offset": 0, "limit": 2})
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if len(page.Items) != 2 || page.Total != 3 || !page.HasMore || page.Offset != 0 || page.Limit != 2 {
		t.Fatalf("first page = %+v, want 2 of 3 items with has_more", page)
	}
	_, last := listWords(t, env, token, map[string]any{"offset": 2, "limit": 2})
	if len(last.Items) != 1 || last.HasMore {
		t.Fatalf("last page = %+v, want 1 item and has_more false", last)
	}
	_, b2 := listWords(t, env, token, map[string]any{"level": "B2", "offset": 0, "limit": 10})
	if b2.Total != 1 || len(b2.Items) != 1 || b2.Items[0].Headword != "three" {
		t.Fatalf("B2 filter = %+v, want only 'three'", b2)
	}
	seen := map[string]bool{}
	for _, it := range append(page.Items, last.Items...) {
		seen[it.Headword] = true
	}
	if seen["unlearned"] || len(seen) != 3 {
		t.Fatalf("listed words = %v, want exactly one/two/three (only words with a card)", seen)
	}
}

func TestWordsListShouldReject400OnBadPaging(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	for name, body := range map[string]map[string]any{
		"negative offset": {"offset": -1, "limit": 10},
		"zero limit":      {"offset": 0, "limit": 0},
	} {
		if status, _ := listWords(t, env, token, body); status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, status)
		}
	}
}

func TestWordsListShouldOnlyShowMyWords(t *testing.T) {
	env := apitest.New(t)
	tokenA := env.Token(t, "sub-a", apitest.AllowedEmail)
	tokenB := env.Token(t, "sub-b", apitest.OtherEmail)
	env.UserID(t, tokenA)
	userB := env.UserID(t, tokenB)
	env.InsertCard(t, userB, env.InsertSense(t, "theirs", "noun", "A1", true), time.Now())

	status, page := listWords(t, env, tokenA, map[string]any{"offset": 0, "limit": 10})
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if page.Total != 0 || len(page.Items) != 0 {
		t.Fatalf("user A words = %+v, want none", page)
	}
}
