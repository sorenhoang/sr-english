package httpapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/sorenhoang/sr-english/api/internal/apitest"
)

func TestSessionTodayShouldOfferExtraNewWordsAfterTheDailyQuotaIsDone(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	userID := env.UserID(t, token)
	setLevel(t, env, token, "B1", 2)
	for _, w := range []string{"x1", "x2", "x3", "x4", "x5"} {
		env.InsertSense(t, w, "noun", "B1", true)
	}
	for _, w := range []string{"done1", "done2"} {
		id := env.InsertSense(t, w, "noun", "A1", true)
		card := env.InsertCard(t, userID, id, time.Now().Add(72*time.Hour))
		env.InsertReview(t, userID, card, id, time.Now(), true)
	}

	if got := today(t, env, token).task(t, "vocab_new"); len(got) != 0 {
		t.Fatalf("quota done: vocab_new = %d, want 0", len(got))
	}
	status, body := env.Do(t, http.MethodGet, "/api/v1/session/today?extra=3", token, nil)
	if status != http.StatusOK {
		t.Fatalf("extra status = %d (%s)", status, body)
	}
	var s sessionResponse
	apitest.Decode(t, body, &s)
	if got := s.task(t, "vocab_new"); len(got) != 3 {
		t.Fatalf("extra=3: vocab_new = %d, want 3", len(got))
	}
	for _, bad := range []string{"-1", "abc", "51"} {
		if status, _ := env.Do(t, http.MethodGet, "/api/v1/session/today?extra="+bad, token, nil); status != http.StatusBadRequest {
			t.Errorf("extra=%s: status = %d, want 400 (0..DAILY_NEW_MAX)", bad, status)
		}
	}
}
