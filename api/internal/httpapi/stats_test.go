package httpapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/sorenhoang/sr-english/api/internal/apitest"
)

type statsBody struct {
	Streak       int `json:"streak"`
	TodayNew     int `json:"today_new"`
	TodayReviews int `json:"today_reviews"`
}

func stats(t *testing.T, env *apitest.Env, token string) statsBody {
	t.Helper()
	status, body := env.Do(t, http.MethodGet, "/api/v1/stats", token, nil)
	if status != http.StatusOK {
		t.Fatalf("stats status = %d (%s), want 200", status, body)
	}
	var s statsBody
	apitest.Decode(t, body, &s)
	return s
}

func setTimezone(t *testing.T, env *apitest.Env, token, zone string) {
	t.Helper()
	status, body := env.Do(t, http.MethodPost, "/api/v1/settings/update", token,
		map[string]any{"level": "A1", "daily_new": 10, "timezone": zone})
	if status != http.StatusOK {
		t.Fatalf("set timezone: %d %s", status, body)
	}
}

// localMidnight returns midnight of today in zone, minus daysAgo days.
func localMidnight(t *testing.T, zone string, daysAgo int) time.Time {
	t.Helper()
	loc, err := time.LoadLocation(zone)
	if err != nil {
		t.Fatalf("load zone: %v", err)
	}
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day()-daysAgo, 0, 0, 0, 0, loc)
}

func seedReviewCard(t *testing.T, env *apitest.Env, userID, word string) (string, string) {
	t.Helper()
	sense := env.InsertSense(t, word, "noun", "A1", true)
	return env.InsertCard(t, userID, sense, time.Now().Add(72*time.Hour)), sense
}

func TestStatsShouldCountTodayAndStreakAcrossLocalMidnight(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	userID := env.UserID(t, token)
	const zone = "Asia/Ho_Chi_Minh"
	setTimezone(t, env, token, zone)
	card, sense := seedReviewCard(t, env, userID, "w")

	todayStart := localMidnight(t, zone, 0)
	env.InsertReview(t, userID, card, sense, todayStart.Add(-10*time.Minute), true) // 23:50 yesterday
	env.InsertReview(t, userID, card, sense, todayStart.Add(10*time.Minute), true)  // 00:10 today
	env.InsertReview(t, userID, card, sense, todayStart.Add(20*time.Minute), false) // a review, not a new word

	got := stats(t, env, token)
	if got.Streak != 2 {
		t.Errorf("streak = %d, want 2 (23:50 and 00:10 local are different days)", got.Streak)
	}
	if got.TodayNew != 1 || got.TodayReviews != 1 {
		t.Errorf("today new/reviews = %d/%d, want 1/1", got.TodayNew, got.TodayReviews)
	}
}

func TestStatsShouldResetStreakAfterAMissedLocalDay(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	userID := env.UserID(t, token)
	const zone = "Asia/Ho_Chi_Minh"
	setTimezone(t, env, token, zone)
	card, sense := seedReviewCard(t, env, userID, "w")

	// Studied 4 and 3 days ago, skipped 2 days ago, studied yesterday and today.
	for _, daysAgo := range []int{4, 3, 1, 0} {
		env.InsertReview(t, userID, card, sense, localMidnight(t, zone, daysAgo).Add(12*time.Hour), false)
	}
	if got := stats(t, env, token).Streak; got != 2 {
		t.Fatalf("streak = %d, want 2 (yesterday + today after the gap)", got)
	}
}

func TestStatsShouldKeepStreakWhenTodayNotStudiedYet(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	userID := env.UserID(t, token)
	const zone = "Asia/Ho_Chi_Minh"
	setTimezone(t, env, token, zone)
	card, sense := seedReviewCard(t, env, userID, "w")
	for _, daysAgo := range []int{2, 1} {
		env.InsertReview(t, userID, card, sense, localMidnight(t, zone, daysAgo).Add(12*time.Hour), false)
	}
	if got := stats(t, env, token).Streak; got != 2 {
		t.Fatalf("streak = %d, want 2 (today not over yet, streak still alive)", got)
	}
}

func TestStatsShouldRecomputeStreakWhenTimezoneChanges(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	userID := env.UserID(t, token)
	card, sense := seedReviewCard(t, env, userID, "w")

	// Two reviews 20 minutes apart straddling Ho Chi Minh midnight: 2 days there, 1 day in UTC.
	setTimezone(t, env, token, "Asia/Ho_Chi_Minh")
	mid := localMidnight(t, "Asia/Ho_Chi_Minh", 0)
	env.InsertReview(t, userID, card, sense, mid.Add(-10*time.Minute), false)
	env.InsertReview(t, userID, card, sense, mid.Add(10*time.Minute), false)
	if got := stats(t, env, token).Streak; got != 2 {
		t.Fatalf("Ho Chi Minh streak = %d, want 2", got)
	}

	setTimezone(t, env, token, "UTC")
	if got := stats(t, env, token).Streak; got != 1 {
		t.Fatalf("UTC streak = %d, want 1 (same reviews, one UTC day)", got)
	}
}

func TestStatsShouldOnlyCountMyReviews(t *testing.T) {
	env := apitest.New(t)
	tokenA := env.Token(t, "sub-a", apitest.AllowedEmail)
	tokenB := env.Token(t, "sub-b", apitest.OtherEmail)
	env.UserID(t, tokenA)
	userB := env.UserID(t, tokenB)
	card, sense := seedReviewCard(t, env, userB, "w")
	env.InsertReview(t, userB, card, sense, time.Now(), true)

	if got := stats(t, env, tokenA); got != (statsBody{}) {
		t.Fatalf("user A stats = %+v, want zeros (B's reviews must not count)", got)
	}
}
