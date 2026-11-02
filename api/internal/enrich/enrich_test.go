package enrich_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sorenhoang/sr-english/api/internal/enrich"
	"github.com/sorenhoang/sr-english/api/internal/testdb"
)

// fakeLLM answers per headword found in the prompt and records which headwords were asked.
type fakeLLM struct {
	mu      sync.Mutex
	answers map[string]string
	asked   []string
}

func (f *fakeLLM) Complete(_ context.Context, prompt string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for word, answer := range f.answers {
		if strings.Contains(prompt, `"`+word+`"`) {
			f.asked = append(f.asked, word)
			return answer, nil
		}
	}
	return "", fmt.Errorf("unexpected prompt: %s", prompt)
}

func validAnswer(word string) string {
	return fmt.Sprintf(`{"ipa_uk":"/uk-%[1]s/","ipa_us":"/us-%[1]s/","vi_meaning":"nghĩa %[1]s","en_definition":"def %[1]s","example_en":"I %[1]s.","example_vi":"Tôi %[1]s."}`, word)
}

func insertSense(t *testing.T, pool *pgxpool.Pool, headword, level string, viMeaning *string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO senses (headword, pos, cefr_level, variants, source, vi_meaning)
		VALUES ($1, 'verb', $2, ARRAY[$1], 'cefrj', $3)`, headword, level, viMeaning); err != nil {
		t.Fatalf("insert sense %s: %v", headword, err)
	}
}

type senseRow struct {
	ipaUK, ipaUS, vi, def, exEN, exVI *string
	reviewed                         bool
}

func loadSense(t *testing.T, pool *pgxpool.Pool, headword string) senseRow {
	t.Helper()
	var r senseRow
	if err := pool.QueryRow(context.Background(), `
		SELECT ipa_uk, ipa_us, vi_meaning, en_definition, example_en, example_vi, reviewed
		FROM senses WHERE headword = $1`, headword).
		Scan(&r.ipaUK, &r.ipaUS, &r.vi, &r.def, &r.exEN, &r.exVI, &r.reviewed); err != nil {
		t.Fatalf("load sense %s: %v", headword, err)
	}
	return r
}

func TestRunShouldWriteAllFieldsWhenLLMAnswersValidJSON(t *testing.T) {
	pool := testdb.New(t)
	insertSense(t, pool, "abandon", "B1", nil)
	llm := &fakeLLM{answers: map[string]string{"abandon": validAnswer("abandon")}}

	res, err := enrich.Run(context.Background(), pool, llm, "B1", 10)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Updated != 1 || res.Failed != 0 {
		t.Fatalf("result = %+v, want Updated 1 Failed 0", res)
	}
	got := loadSense(t, pool, "abandon")
	if got.ipaUK == nil || *got.ipaUK != "/uk-abandon/" || got.ipaUS == nil || *got.ipaUS != "/us-abandon/" ||
		got.vi == nil || *got.vi != "nghĩa abandon" || got.def == nil || *got.def != "def abandon" ||
		got.exEN == nil || *got.exEN != "I abandon." || got.exVI == nil || *got.exVI != "Tôi abandon." {
		t.Fatalf("sense after enrich = %+v, want every field filled from the LLM answer", got)
	}
	if got.reviewed {
		t.Fatalf("reviewed = true, want false for machine-generated content")
	}
}

func TestRunShouldLeaveSenseUntouchedWhenAnswerIsBrokenOrIncomplete(t *testing.T) {
	pool := testdb.New(t)
	insertSense(t, pool, "broken", "B1", nil)
	insertSense(t, pool, "partial", "B1", nil)
	insertSense(t, pool, "fine", "B1", nil)
	llm := &fakeLLM{answers: map[string]string{
		"broken":  `{"ipa_uk": "/x/", "vi_meaning": `,
		"partial": `{"ipa_uk":"/p/","ipa_us":"/p/","en_definition":"d","example_en":"e","example_vi":"v"}`,
		"fine":    validAnswer("fine"),
	}}

	res, err := enrich.Run(context.Background(), pool, llm, "B1", 10)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Updated != 1 || res.Failed != 2 {
		t.Fatalf("result = %+v, want Updated 1 Failed 2", res)
	}
	for _, w := range []string{"broken", "partial"} {
		got := loadSense(t, pool, w)
		if got.vi != nil || got.ipaUK != nil {
			t.Fatalf("%s was modified (%+v), want untouched", w, got)
		}
	}
	if got := loadSense(t, pool, "fine"); got.vi == nil || *got.vi != "nghĩa fine" {
		t.Fatalf("fine vi_meaning = %v, want enriched", got.vi)
	}
}

func TestRunShouldSkipEnrichedSensesAndRespectLevelAndLimit(t *testing.T) {
	pool := testdb.New(t)
	done := "đã có"
	insertSense(t, pool, "already", "B1", &done)
	insertSense(t, pool, "first", "B1", nil)
	insertSense(t, pool, "second", "B1", nil)
	insertSense(t, pool, "other", "A2", nil)
	llm := &fakeLLM{answers: map[string]string{
		"already": validAnswer("already"), "first": validAnswer("first"),
		"second": validAnswer("second"), "other": validAnswer("other"),
	}}

	res, err := enrich.Run(context.Background(), pool, llm, "B1", 1)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.Updated != 1 {
		t.Fatalf("Updated = %d, want 1 (limit)", res.Updated)
	}
	if len(llm.asked) != 1 {
		t.Fatalf("LLM asked %v, want exactly 1 call", llm.asked)
	}
	if llm.asked[0] == "already" || llm.asked[0] == "other" {
		t.Fatalf("LLM asked about %q, want only B1 senses without vi_meaning", llm.asked[0])
	}
	if got := loadSense(t, pool, "already"); got.vi == nil || *got.vi != done {
		t.Fatalf("already vi_meaning = %v, want unchanged %q", got.vi, done)
	}
}
