package httpapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/sorenhoang/sr-english/api/internal/apitest"
	"github.com/sorenhoang/sr-english/api/internal/config"
)

type settingsBody struct {
	Level    string `json:"level"`
	DailyNew int    `json:"daily_new"`
	Timezone string `json:"timezone"`
}

type errorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestSettingsShouldReturnDefaultsOnFirstRead(t *testing.T) {
	env := apitest.New(t)
	status, body := env.Do(t, http.MethodGet, "/api/v1/settings", env.Token(t, "sub-a", apitest.AllowedEmail), nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d (%s), want 200", status, body)
	}
	var got settingsBody
	apitest.Decode(t, body, &got)
	want := settingsBody{Level: "A1", DailyNew: config.DefaultDailyNew, Timezone: "UTC"}
	if got != want {
		t.Fatalf("settings = %+v, want %+v", got, want)
	}
}

func TestSettingsShouldPersistValidUpdate(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	update := settingsBody{Level: "B1", DailyNew: 15, Timezone: "Asia/Ho_Chi_Minh"}

	status, body := env.Do(t, http.MethodPost, "/api/v1/settings/update", token, update)
	if status != http.StatusOK {
		t.Fatalf("update status = %d (%s), want 200", status, body)
	}
	_, body = env.Do(t, http.MethodGet, "/api/v1/settings", token, nil)
	var got settingsBody
	apitest.Decode(t, body, &got)
	if got != update {
		t.Fatalf("settings after update = %+v, want %+v", got, update)
	}
}

func TestSettingsShouldReject400WhenInputIsInvalid(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	cases := []struct {
		name    string
		body    settingsBody
		message string
	}{
		{"bad level", settingsBody{Level: "Z9", DailyNew: 10, Timezone: "UTC"}, "level is invalid"},
		{"bad timezone", settingsBody{Level: "A1", DailyNew: 10, Timezone: "Mars/Olympus"}, "timezone is invalid"},
		{"daily_new zero", settingsBody{Level: "A1", DailyNew: 0, Timezone: "UTC"}, "daily_new"},
		{"daily_new above max", settingsBody{Level: "A1", DailyNew: config.DefaultDailyNewMax + 1, Timezone: "UTC"}, "daily_new"},
	}
	for _, c := range cases {
		status, body := env.Do(t, http.MethodPost, "/api/v1/settings/update", token, c.body)
		if status != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", c.name, status)
			continue
		}
		var e errorBody
		apitest.Decode(t, body, &e)
		if !strings.Contains(e.Error.Message, c.message) {
			t.Errorf("%s: message = %q, want it to contain %q", c.name, e.Error.Message, c.message)
		}
	}
	_, body := env.Do(t, http.MethodGet, "/api/v1/settings", token, nil)
	var got settingsBody
	apitest.Decode(t, body, &got)
	if got.Level != "A1" || got.DailyNew != config.DefaultDailyNew {
		t.Fatalf("settings changed by rejected updates: %+v", got)
	}
}

func TestSettingsShouldNotLeakBetweenUsers(t *testing.T) {
	env := apitest.New(t)
	tokenA := env.Token(t, "sub-a", apitest.AllowedEmail)
	tokenB := env.Token(t, "sub-b", apitest.OtherEmail)

	status, _ := env.Do(t, http.MethodPost, "/api/v1/settings/update", tokenA,
		settingsBody{Level: "C1", DailyNew: 20, Timezone: "Europe/London"})
	if status != http.StatusOK {
		t.Fatalf("update A status = %d, want 200", status)
	}
	_, body := env.Do(t, http.MethodGet, "/api/v1/settings", tokenB, nil)
	var got settingsBody
	apitest.Decode(t, body, &got)
	if got.Level != "A1" || got.Timezone != "UTC" {
		t.Fatalf("user B settings = %+v, want defaults (not A's)", got)
	}
}
