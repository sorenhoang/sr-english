package httpapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/sorenhoang/sr-english/api/internal/apitest"
	"github.com/sorenhoang/sr-english/api/internal/httpapi"
)

type sessionItem struct {
	SenseID   string  `json:"sense_id"`
	CardID    *string `json:"card_id"`
	Headword  string  `json:"headword"`
	CEFRLevel string  `json:"cefr_level"`
	VIMeaning string  `json:"vi_meaning"`
}

type sessionResponse struct {
	Tasks []struct {
		Type  string        `json:"type"`
		Items []sessionItem `json:"items"`
	} `json:"tasks"`
}

func (s sessionResponse) task(t *testing.T, typ string) []sessionItem {
	t.Helper()
	for _, task := range s.Tasks {
		if task.Type == typ {
			return task.Items
		}
	}
	t.Fatalf("task %q missing in %+v", typ, s.Tasks)
	return nil
}

func setLevel(t *testing.T, env *apitest.Env, token, level string, dailyNew int) {
	t.Helper()
	status, body := env.Do(t, http.MethodPost, "/api/v1/settings/update", token,
		map[string]any{"level": level, "daily_new": dailyNew, "timezone": "UTC"})
	if status != http.StatusOK {
		t.Fatalf("set level: %d %s", status, body)
	}
}

func today(t *testing.T, env *apitest.Env, token string) sessionResponse {
	t.Helper()
	status, body := env.Do(t, http.MethodGet, "/api/v1/session/today", token, nil)
	if status != http.StatusOK {
		t.Fatalf("session/today status = %d (%s), want 200", status, body)
	}
	var s sessionResponse
	apitest.Decode(t, body, &s)
	return s
}

func TestSessionTodayShouldListDueCardsOldestFirstAndOnlyMine(t *testing.T) {
	env := apitest.New(t)
	tokenA := env.Token(t, "sub-a", apitest.AllowedEmail)
	tokenB := env.Token(t, "sub-b", apitest.OtherEmail)
	userA, userB := env.UserID(t, tokenA), env.UserID(t, tokenB)
	now := time.Now()

	older := env.InsertSense(t, "older", "noun", "A1", true)
	newer := env.InsertSense(t, "newer", "noun", "A1", true)
	future := env.InsertSense(t, "future", "noun", "A1", true)
	theirs := env.InsertSense(t, "theirs", "noun", "A1", true)
	env.InsertCard(t, userA, newer, now.Add(-time.Hour))
	env.InsertCard(t, userA, older, now.Add(-48*time.Hour))
	env.InsertCard(t, userA, future, now.Add(48*time.Hour))
	env.InsertCard(t, userB, theirs, now.Add(-time.Hour))

	review := today(t, env, tokenA).task(t, "vocab_review")
	if len(review) != 2 || review[0].Headword != "older" || review[1].Headword != "newer" {
		t.Fatalf("vocab_review = %+v, want [older newer] (due ≤ now, oldest first, only user A)", review)
	}
}

func TestSessionTodayShouldOfferEnrichedNewWordsAtMyLevelWithoutCards(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	userID := env.UserID(t, token)
	setLevel(t, env, token, "B1", 3)

	env.InsertSense(t, "fresh1", "verb", "B1", true)
	env.InsertSense(t, "fresh2", "verb", "B1", true)
	env.InsertSense(t, "bare", "verb", "B1", false)
	env.InsertSense(t, "easy", "verb", "A1", true)
	known := env.InsertSense(t, "known", "verb", "B1", true)
	env.InsertCard(t, userID, known, time.Now().Add(24*time.Hour))

	items := today(t, env, token).task(t, "vocab_new")
	got := map[string]bool{}
	for _, it := range items {
		got[it.Headword] = true
		if it.CEFRLevel != "B1" || it.VIMeaning == "" {
			t.Errorf("new item %+v, want B1 with a vi_meaning", it)
		}
	}
	if len(items) != 2 || !got["fresh1"] || !got["fresh2"] {
		t.Fatalf("vocab_new = %v, want exactly fresh1 + fresh2 (not bare, easy or known)", got)
	}
}

func TestSessionTodayShouldSubtractWordsAlreadyLearnedToday(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	userID := env.UserID(t, token)
	setLevel(t, env, token, "B1", 2)

	for _, w := range []string{"w1", "w2", "w3"} {
		env.InsertSense(t, w, "noun", "B1", true)
	}
	learned := env.InsertSense(t, "learned", "noun", "B1", true)
	card := env.InsertCard(t, userID, learned, time.Now().Add(24*time.Hour))
	env.InsertReview(t, userID, card, learned, time.Now(), true)

	if items := today(t, env, token).task(t, "vocab_new"); len(items) != 1 {
		t.Fatalf("vocab_new has %d items, want daily_new 2 − 1 learned today = 1", len(items))
	}
}

func TestSessionTodayShouldCapReviewsAtConfiguredMaximum(t *testing.T) {
	env := apitest.NewWithDeps(t, func(d *httpapi.Deps) { d.Learning.MaxReviewsPerDay = 2 })
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	userID := env.UserID(t, token)
	for i, w := range []string{"r1", "r2", "r3"} {
		id := env.InsertSense(t, w, "noun", "A1", true)
		env.InsertCard(t, userID, id, time.Now().Add(-time.Duration(i+1)*time.Hour))
	}

	if review := today(t, env, token).task(t, "vocab_review"); len(review) != 2 {
		t.Fatalf("vocab_review has %d items, want MAX_REVIEWS_PER_DAY = 2", len(review))
	}
}
