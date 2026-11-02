package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/sorenhoang/sr-english/api/internal/apitest"
	"github.com/sorenhoang/sr-english/api/internal/config"
)

type cardBody struct {
	CardID        string    `json:"card_id"`
	SenseID       string    `json:"sense_id"`
	Due           time.Time `json:"due"`
	Stability     float64   `json:"stability"`
	Difficulty    float64   `json:"difficulty"`
	ElapsedDays   uint64    `json:"elapsed_days"`
	ScheduledDays uint64    `json:"scheduled_days"`
	Reps          uint64    `json:"reps"`
	Lapses        uint64    `json:"lapses"`
	State         int       `json:"state"`
	LastReview    time.Time `json:"last_review"`
}

func (c cardBody) fsrsCard() fsrs.Card {
	return fsrs.Card{Due: c.Due, Stability: c.Stability, Difficulty: c.Difficulty, ElapsedDays: c.ElapsedDays,
		ScheduledDays: c.ScheduledDays, Reps: c.Reps, Lapses: c.Lapses, State: fsrs.State(c.State), LastReview: c.LastReview}
}

// scheduler mirrors the documented config: go-fsrs defaults, desired retention from config, no fuzz.
func scheduler() *fsrs.FSRS {
	p := fsrs.DefaultParam()
	p.RequestRetention = config.DefaultDesiredRetention
	p.EnableFuzz = false
	return fsrs.NewFSRS(p)
}

func assertMatches(t *testing.T, label string, got cardBody, want fsrs.Card) {
	t.Helper()
	if d := got.Due.Sub(want.Due); d > time.Millisecond || d < -time.Millisecond {
		t.Errorf("%s due = %v, want %v", label, got.Due, want.Due)
	}
	if got.Stability != want.Stability || got.Difficulty != want.Difficulty || got.Reps != want.Reps ||
		got.Lapses != want.Lapses || fsrs.State(got.State) != want.State || got.ScheduledDays != want.ScheduledDays {
		t.Errorf("%s card = %+v, want fsrs %+v", label, got, want)
	}
}

func review(t *testing.T, env *apitest.Env, token, senseID string, rating int) (int, cardBody) {
	t.Helper()
	status, body := env.Do(t, http.MethodPost, "/api/v1/review", token, map[string]any{"sense_id": senseID, "rating": rating})
	var c cardBody
	if status == http.StatusOK {
		apitest.Decode(t, body, &c)
	}
	return status, c
}

func TestReviewShouldCreateCardScheduledByFSRSWhenSenseIsNew(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	userID := env.UserID(t, token)
	sense := env.InsertSense(t, "abandon", "verb", "B1", true)

	status, card := review(t, env, token, sense, int(fsrs.Good))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	assertMatches(t, "first review", card, scheduler().Next(fsrs.NewCard(), card.LastReview, fsrs.Good).Card)

	var cards, newReviews int
	ctx := context.Background()
	if err := env.Pool.QueryRow(ctx, `SELECT count(*) FROM cards WHERE user_id = $1 AND sense_id = $2`, userID, sense).Scan(&cards); err != nil {
		t.Fatalf("count cards: %v", err)
	}
	if err := env.Pool.QueryRow(ctx, `SELECT count(*) FROM reviews WHERE user_id = $1 AND sense_id = $2 AND is_new AND rating = 3`, userID, sense).Scan(&newReviews); err != nil {
		t.Fatalf("count reviews: %v", err)
	}
	if cards != 1 || newReviews != 1 {
		t.Fatalf("cards = %d, new reviews = %d; want 1 and 1", cards, newReviews)
	}
}

func TestReviewShouldRescheduleExistingCardExactlyLikeFSRS(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	sense := env.InsertSense(t, "abandon", "verb", "B1", true)

	_, first := review(t, env, token, sense, int(fsrs.Good))
	status, second := review(t, env, token, sense, int(fsrs.Again))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if second.CardID != first.CardID {
		t.Fatalf("card id changed %s → %s, want the same card", first.CardID, second.CardID)
	}
	assertMatches(t, "second review", second, scheduler().Next(first.fsrsCard(), second.LastReview, fsrs.Again).Card)
}

func TestReviewShouldRejectBadRatingAndUnknownSense(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	sense := env.InsertSense(t, "abandon", "verb", "B1", true)

	for _, rating := range []int{0, 5} {
		status, body := env.Do(t, http.MethodPost, "/api/v1/review", token, map[string]any{"sense_id": sense, "rating": rating})
		var e errorBody
		if status == http.StatusBadRequest {
			apitest.Decode(t, body, &e)
		}
		if status != http.StatusBadRequest || e.Error.Message != "rating is invalid" {
			t.Errorf("rating %d: status %d message %q, want 400 'rating is invalid'", rating, status, e.Error.Message)
		}
	}
	if status, _ := review(t, env, token, "6f1c1d1e-0000-4000-8000-000000000000", 3); status != http.StatusNotFound {
		t.Errorf("unknown sense: status = %d, want 404", status)
	}
}

func TestReviewShouldNotTouchAnotherUsersCard(t *testing.T) {
	env := apitest.New(t)
	tokenA := env.Token(t, "sub-a", apitest.AllowedEmail)
	tokenB := env.Token(t, "sub-b", apitest.OtherEmail)
	sense := env.InsertSense(t, "abandon", "verb", "B1", true)

	_, cardA := review(t, env, tokenA, sense, int(fsrs.Good))
	_, cardB := review(t, env, tokenB, sense, int(fsrs.Again))
	if cardB.CardID == cardA.CardID {
		t.Fatalf("user B reused user A's card %s", cardA.CardID)
	}
	var reps int
	if err := env.Pool.QueryRow(context.Background(), `SELECT reps FROM cards WHERE id = $1`, cardA.CardID).Scan(&reps); err != nil {
		t.Fatalf("load card A: %v", err)
	}
	if reps != 1 {
		t.Fatalf("user A card reps = %d after B's review, want 1", reps)
	}
}
