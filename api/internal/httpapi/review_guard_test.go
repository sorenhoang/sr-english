package httpapi_test

import (
	"context"
	"net/http"
	"sync"
	"testing"

	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/sorenhoang/sr-english/api/internal/apitest"
)

func TestReviewShouldReturn404ForAnotherUsersPrivateSenseAndDeletedSense(t *testing.T) {
	env := apitest.New(t)
	tokenA := env.Token(t, "sub-a", apitest.AllowedEmail)
	tokenB := env.Token(t, "sub-b", apitest.OtherEmail)
	userA := env.UserID(t, tokenA)
	ctx := context.Background()

	var private string
	if err := env.Pool.QueryRow(ctx, `
		INSERT INTO senses (headword, pos, variants, source, owner_user_id)
		VALUES ('mine', 'noun', ARRAY['mine'], 'custom', $1) RETURNING id`, userA).Scan(&private); err != nil {
		t.Fatalf("insert private sense: %v", err)
	}
	deleted := env.InsertSense(t, "gone", "noun", "A1", true)
	if _, err := env.Pool.Exec(ctx, `UPDATE senses SET deleted_at = now() WHERE id = $1`, deleted); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	if status, _ := review(t, env, tokenB, private, int(fsrs.Good)); status != http.StatusNotFound {
		t.Errorf("B reviewing A's private sense: status = %d, want 404", status)
	}
	if status, _ := review(t, env, tokenA, private, int(fsrs.Good)); status != http.StatusOK {
		t.Errorf("A reviewing own private sense: status = %d, want 200", status)
	}
	if status, _ := review(t, env, tokenA, deleted, int(fsrs.Good)); status != http.StatusNotFound {
		t.Errorf("soft-deleted sense: status = %d, want 404", status)
	}
}

func TestReviewShouldHandleConcurrentFirstReviewsOfTheSameSense(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	userID := env.UserID(t, token)
	const parallel, rounds = 20, 5
	for round := range rounds {
		sense := env.InsertSense(t, "word"+string(rune('a'+round)), "verb", "B1", true)
		concurrentFirstReviews(t, env, token, userID, sense, parallel)
	}
}

func concurrentFirstReviews(t *testing.T, env *apitest.Env, token, userID, sense string, parallel int) {
	t.Helper()
	statuses := make([]int, parallel)
	var wg sync.WaitGroup
	for i := range parallel {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses[i], _ = review(t, env, token, sense, int(fsrs.Good))
		}()
	}
	wg.Wait()

	for i, s := range statuses {
		if s != http.StatusOK {
			t.Errorf("request %d: status = %d, want 200", i, s)
		}
	}
	var cards, reviews int
	ctx := context.Background()
	if err := env.Pool.QueryRow(ctx, `SELECT count(*) FROM cards WHERE user_id = $1 AND sense_id = $2`, userID, sense).Scan(&cards); err != nil {
		t.Fatalf("count cards: %v", err)
	}
	if err := env.Pool.QueryRow(ctx, `SELECT count(*) FROM reviews WHERE user_id = $1 AND sense_id = $2`, userID, sense).Scan(&reviews); err != nil {
		t.Fatalf("count reviews: %v", err)
	}
	if cards != 1 || reviews != parallel {
		t.Fatalf("cards = %d, reviews = %d; want 1 card and %d reviews", cards, reviews, parallel)
	}
}
