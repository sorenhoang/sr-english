package httpapi_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/sorenhoang/sr-english/api/internal/apitest"
)

func readingSuggestion(t *testing.T, env *apitest.Env, token string) []string {
	t.Helper()
	status, raw := env.Do(t, http.MethodGet, "/api/v1/session/today", token, nil)
	if status != http.StatusOK {
		t.Fatalf("today status = %d", status)
	}
	var resp struct {
		Tasks []struct {
			Type  string `json:"type"`
			Items []struct {
				Title string `json:"title"`
			} `json:"items"`
		} `json:"tasks"`
	}
	apitest.Decode(t, raw, &resp)
	var out []string
	for _, task := range resp.Tasks {
		if task.Type == "reading" {
			for _, it := range task.Items {
				out = append(out, it.Title)
			}
		}
	}
	return out
}

func TestPassageCompleteShouldOnlyAffectTheUserWhoCompletedIt(t *testing.T) {
	env := apitest.New(t)
	tokenA := env.Token(t, "sub-a", apitest.AllowedEmail)
	tokenB := env.Token(t, "sub-b", apitest.OtherEmail)
	env.UserID(t, tokenA)
	env.UserID(t, tokenB)
	setLevel(t, env, tokenA, "B1", 10)
	setLevel(t, env, tokenB, "B1", 10)
	id := env.InsertPassage(t, "shared", "B1", []string{"education"}, "", nil, nil)

	for range 2 {
		if status, raw := env.Do(t, http.MethodPost, "/api/v1/passage/complete?id="+id, tokenA, nil); status != http.StatusOK {
			t.Fatalf("complete status = %d (%s), want 200 (repeat is a no-op)", status, raw)
		}
	}
	if got := readingSuggestion(t, env, tokenA); len(got) != 0 {
		t.Errorf("user A reading = %v, want none after completing", got)
	}
	if got := readingSuggestion(t, env, tokenB); len(got) != 1 || got[0] != "shared" {
		t.Errorf("user B reading = %v, want [shared] (A's progress is not B's)", got)
	}
	if status, _ := env.Do(t, http.MethodPost, "/api/v1/passage/complete?id=6f1c1d1e-0000-4000-8000-000000000000", tokenA, nil); status != http.StatusNotFound {
		t.Errorf("complete unknown passage: status = %d, want 404", status)
	}
}

func TestPassagesShouldHideSoftDeletedPassages(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	id := env.InsertPassage(t, "gone", "B1", []string{"education"}, "", nil, nil)
	if _, err := env.Pool.Exec(context.Background(), `UPDATE passages SET deleted_at = now() WHERE id = $1`, id); err != nil {
		t.Fatalf("soft delete: %v", err)
	}
	if page := listPassages(t, env, token, map[string]any{"offset": 0, "limit": 10}); page.Total != 0 {
		t.Errorf("list = %+v, want no soft-deleted passages", page)
	}
	if status, _ := env.Do(t, http.MethodGet, "/api/v1/passage?id="+id, token, nil); status != http.StatusNotFound {
		t.Errorf("get soft-deleted: status = %d, want 404", status)
	}
}
