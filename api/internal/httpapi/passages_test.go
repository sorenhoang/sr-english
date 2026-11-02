package httpapi_test

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/sorenhoang/sr-english/api/internal/apitest"
)

type passageSummary struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Level    string   `json:"cefr_level"`
	Topics   []string `json:"topics"`
	Kinds    []string `json:"kinds"`
	Source   string   `json:"source_url"`
	License  string   `json:"license"`
	AudioURL *string  `json:"audio_url"`
}

type passagesPage struct {
	Items []passageSummary `json:"items"`
	Total int              `json:"total"`
}

func listPassages(t *testing.T, env *apitest.Env, token string, body map[string]any) passagesPage {
	t.Helper()
	status, raw := env.Do(t, http.MethodPost, "/api/v1/passages/list", token, body)
	if status != http.StatusOK {
		t.Fatalf("passages/list status = %d (%s), want 200", status, raw)
	}
	var p passagesPage
	apitest.Decode(t, raw, &p)
	return p
}

func titles(p passagesPage) []string {
	var out []string
	for _, it := range p.Items {
		out = append(out, it.Title)
	}
	slices.Sort(out)
	return out
}

func TestPassagesListShouldFilterByLevelTopicAndKindWithSourceAndLicense(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	env.InsertPassage(t, "birds", "B1", []string{"the-natural-world"}, "https://audio.test/birds.mp3", nil, nil)
	env.InsertPassage(t, "school", "B1", []string{"education"}, "", nil, nil)
	env.InsertPassage(t, "hard", "C1", []string{"education"}, "", nil, nil)

	if got := titles(listPassages(t, env, token, map[string]any{"level": "B1", "offset": 0, "limit": 10})); !slices.Equal(got, []string{"birds", "school"}) {
		t.Errorf("level B1 = %v, want [birds school]", got)
	}
	if got := titles(listPassages(t, env, token, map[string]any{"topic": "education", "offset": 0, "limit": 10})); !slices.Equal(got, []string{"hard", "school"}) {
		t.Errorf("topic education = %v, want [hard school]", got)
	}
	listening := listPassages(t, env, token, map[string]any{"kind": "listening", "offset": 0, "limit": 10})
	if got := titles(listening); !slices.Equal(got, []string{"birds"}) {
		t.Fatalf("kind listening = %v, want [birds]", got)
	}
	it := listening.Items[0]
	if it.Source == "" || it.License != "Public domain · learningenglish.voanews.com" || it.AudioURL == nil {
		t.Fatalf("passage = %+v, want source_url, VOA license and audio_url", it)
	}
}

func TestPassageShouldMarkTokensAlreadyInMyDeck(t *testing.T) {
	env := apitest.New(t)
	tokenA := env.Token(t, "sub-a", apitest.AllowedEmail)
	tokenB := env.Token(t, "sub-b", apitest.OtherEmail)
	userA := env.UserID(t, tokenA)
	env.UserID(t, tokenB)
	known := env.InsertSense(t, "bird", "noun", "A1", true)
	fresh := env.InsertSense(t, "fly", "verb", "A1", true)
	env.InsertCard(t, userA, known, time.Now().Add(time.Hour))
	id := env.InsertPassage(t, "birds", "A1", []string{"the-natural-world"}, "", []apitest.PassageToken{
		{Text: "Birds", Start: 0, End: 5, Start16: 0, End16: 5, Kind: "word", SenseID: known, Headword: "bird", Counted: true},
		{Text: "fly", Start: 6, End: 9, Start16: 6, End16: 9, Kind: "word", SenseID: fresh, Headword: "fly", Counted: true},
	}, []string{"Birds fly."})

	saved := func(token string) map[string]bool {
		status, raw := env.Do(t, http.MethodGet, "/api/v1/passage?id="+id, token, nil)
		if status != http.StatusOK {
			t.Fatalf("passage status = %d (%s), want 200", status, raw)
		}
		var p struct {
			Body   string `json:"body"`
			Tokens []struct {
				Text  string `json:"text"`
				Saved bool   `json:"saved"`
			} `json:"tokens"`
			Sentences []string `json:"sentences"`
		}
		apitest.Decode(t, raw, &p)
		if p.Body == "" || len(p.Sentences) != 1 {
			t.Fatalf("passage body/sentences missing: %+v", p)
		}
		out := map[string]bool{}
		for _, tok := range p.Tokens {
			out[tok.Text] = tok.Saved
		}
		return out
	}
	if got := saved(tokenA); !got["Birds"] || got["fly"] {
		t.Errorf("user A saved flags = %v, want Birds saved, fly not", got)
	}
	if got := saved(tokenB); got["Birds"] {
		t.Errorf("user B saved flags = %v, want nothing saved (A's deck is not B's)", got)
	}
}

func TestPassageShouldReturn404ForUnknownID(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	if status, _ := env.Do(t, http.MethodGet, "/api/v1/passage?id=6f1c1d1e-0000-4000-8000-000000000000", token, nil); status != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", status)
	}
}

func TestSessionTodayShouldSuggestUnfinishedReadingAndListeningAtMyLevel(t *testing.T) {
	env := apitest.New(t)
	token := env.Token(t, "sub-a", apitest.AllowedEmail)
	env.UserID(t, token)
	setLevel(t, env, token, "B1", 10)
	done := env.InsertPassage(t, "done", "B1", []string{"education"}, "https://audio.test/done.mp3", nil, nil)
	env.InsertPassage(t, "todo-read", "B1", []string{"education"}, "", nil, nil)
	env.InsertPassage(t, "too-hard", "C2", []string{"education"}, "https://audio.test/hard.mp3", nil, nil)

	if status, raw := env.Do(t, http.MethodPost, "/api/v1/passage/complete?id="+done, token, nil); status != http.StatusOK {
		t.Fatalf("complete status = %d (%s), want 200", status, raw)
	}

	status, raw := env.Do(t, http.MethodGet, "/api/v1/session/today", token, nil)
	if status != http.StatusOK {
		t.Fatalf("today status = %d", status)
	}
	var resp struct {
		Tasks []struct {
			Type  string           `json:"type"`
			Items []passageSummary `json:"items"`
		} `json:"tasks"`
	}
	apitest.Decode(t, raw, &resp)
	got := map[string][]string{}
	for _, task := range resp.Tasks {
		for _, it := range task.Items {
			got[task.Type] = append(got[task.Type], it.Title)
		}
	}
	if !slices.Equal(got["reading"], []string{"todo-read"}) {
		t.Errorf("reading task = %v, want [todo-read] (completed and off-level passages excluded)", got["reading"])
	}
	if _, ok := got["listening"]; ok {
		t.Errorf("listening task = %v, want no task (only listening passage at B1 is completed)", got["listening"])
	}
}
