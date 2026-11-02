package httpapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/sorenhoang/sr-english/api/internal/apitest"
	"github.com/sorenhoang/sr-english/api/internal/httpapi"
)

func TestSessionTodayShouldIgnoreOtherUsersCardsAndReviewsForNewWords(t *testing.T) {
	env := apitest.New(t)
	tokenA := env.Token(t, "sub-a", apitest.AllowedEmail)
	tokenB := env.Token(t, "sub-b", apitest.OtherEmail)
	env.UserID(t, tokenA)
	userB := env.UserID(t, tokenB)
	setLevel(t, env, tokenA, "B1", 2)

	shared := env.InsertSense(t, "shared", "noun", "B1", true)
	env.InsertSense(t, "solo", "noun", "B1", true)
	cardB := env.InsertCard(t, userB, shared, time.Now().Add(24*time.Hour))
	env.InsertReview(t, userB, cardB, shared, time.Now(), true)

	items := today(t, env, tokenA).task(t, "vocab_new")
	got := map[string]bool{}
	for _, it := range items {
		got[it.Headword] = true
	}
	if len(items) != 2 || !got["shared"] || !got["solo"] {
		t.Fatalf("user A vocab_new = %v, want shared + solo (B's card and B's learn today must not count for A)", got)
	}
}

func TestSessionTodayShouldCountLearnedTodayFromMidnightInMyTimezone(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	userID := env.UserID(t, token)
	const zone = "Pacific/Kiritimati" // UTC+14: its midnight never lines up with UTC midnight
	status, body := env.Do(t, http.MethodPost, "/api/v1/settings/update", token,
		map[string]any{"level": "B1", "daily_new": 3, "timezone": zone})
	if status != http.StatusOK {
		t.Fatalf("settings: %d %s", status, body)
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		t.Fatalf("load zone: %v", err)
	}
	local := time.Now().In(loc)
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)

	for _, w := range []string{"n1", "n2", "n3", "n4"} {
		env.InsertSense(t, w, "noun", "B1", true)
	}
	yesterday := env.InsertSense(t, "yesterday", "noun", "A1", true)
	todayWord := env.InsertSense(t, "todayword", "noun", "A1", true)
	c1 := env.InsertCard(t, userID, yesterday, time.Now().Add(72*time.Hour))
	c2 := env.InsertCard(t, userID, todayWord, time.Now().Add(72*time.Hour))
	env.InsertReview(t, userID, c1, yesterday, midnight.Add(-time.Second), true) // local yesterday
	env.InsertReview(t, userID, c2, todayWord, midnight, true)                  // local today

	if items := today(t, env, token).task(t, "vocab_new"); len(items) != 2 {
		t.Fatalf("vocab_new has %d items, want daily_new 3 − 1 learned since local midnight in %s = 2", len(items), zone)
	}
}

func TestSessionTodayShouldKeepTheOldestDueCardsWhenCapped(t *testing.T) {
	env := apitest.NewWithDeps(t, func(d *httpapi.Deps) { d.Learning.MaxReviewsPerDay = 2 })
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	userID := env.UserID(t, token)
	for i, w := range []string{"recent", "middle", "oldest"} {
		id := env.InsertSense(t, w, "noun", "A1", true)
		env.InsertCard(t, userID, id, time.Now().Add(-time.Duration(i+1)*time.Hour))
	}
	deleted := env.InsertSense(t, "deleted", "noun", "A1", true)
	card := env.InsertCard(t, userID, deleted, time.Now().Add(-10*time.Hour))
	if _, err := env.Pool.Exec(context.Background(), `UPDATE cards SET deleted_at = now() WHERE id = $1`, card); err != nil {
		t.Fatalf("soft delete card: %v", err)
	}

	review := today(t, env, token).task(t, "vocab_review")
	if len(review) != 2 || review[0].Headword != "oldest" || review[1].Headword != "middle" {
		t.Fatalf("vocab_review = %+v, want [oldest middle] (soft-deleted card excluded)", review)
	}
}

func TestSessionTodayShouldNeverUsePrivateSensesAsDistractors(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	tokenB := env.Token(t, "sub-b", apitest.OtherEmail)
	env.UserID(t, token)
	userB := env.UserID(t, tokenB)
	setLevel(t, env, token, "B1", 1)
	env.InsertSense(t, "target", "noun", "B1", true)
	env.InsertSense(t, "public", "noun", "A1", true)
	if _, err := env.Pool.Exec(context.Background(), `
		INSERT INTO senses (headword, pos, variants, source, owner_user_id, vi_meaning)
		VALUES ('secret', 'noun', ARRAY['secret'], 'custom', $1, 'bí mật của B')`, userB); err != nil {
		t.Fatalf("insert private sense: %v", err)
	}

	status, body := env.Do(t, http.MethodGet, "/api/v1/session/today", token, nil)
	if status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	var resp struct {
		Tasks []struct {
			Type  string `json:"type"`
			Items []struct {
				Distractors []string `json:"distractors"`
			} `json:"items"`
		} `json:"tasks"`
	}
	apitest.Decode(t, body, &resp)
	for _, task := range resp.Tasks {
		for _, it := range task.Items {
			for _, d := range it.Distractors {
				if d == "bí mật của B" {
					t.Fatalf("distractors %v include another user's private sense", it.Distractors)
				}
			}
		}
	}
}
