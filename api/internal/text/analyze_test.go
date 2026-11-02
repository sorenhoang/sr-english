package text_test

import (
	"strings"
	"testing"

	"github.com/sorenhoang/sr-english/api/internal/text"
)

func lexicon(t *testing.T, entries ...text.Entry) *text.Analyzer {
	t.Helper()
	a, err := text.NewAnalyzer(entries)
	if err != nil {
		t.Fatalf("NewAnalyzer: %v", err)
	}
	return a
}

func entry(headword, level string) text.Entry {
	return text.Entry{SenseID: "id-" + headword, Headword: headword, Variants: []string{headword}, POS: "verb", Level: level}
}

func wordTokens(r text.Result) []text.Token {
	var out []text.Token
	for _, tok := range r.Tokens {
		if tok.Kind == text.KindWord {
			out = append(out, tok)
		}
	}
	return out
}

func TestAnalyzeShouldLemmatizeInflectedForms(t *testing.T) {
	a := lexicon(t, entry("fail", "A2"), entry("go", "A1"), entry("child", "A1"))
	r := a.Analyze("It failed. They went. The children laughed.")

	want := map[string]string{"failed": "fail", "went": "go", "children": "child"}
	for _, tok := range wordTokens(r) {
		if lemma, ok := want[strings.ToLower(tok.Text)]; ok {
			if tok.Headword != lemma {
				t.Errorf("%q headword = %q, want %q", tok.Text, tok.Headword, lemma)
			}
			delete(want, strings.ToLower(tok.Text))
		}
	}
	if len(want) != 0 {
		t.Fatalf("tokens not matched: %v", want)
	}
}

func TestAnalyzeShouldMatchMultiWordEntriesAsOneTokenPreferringTheLongest(t *testing.T) {
	a := lexicon(t, entry("roll", "B1"), entry("roll back", "C1"), entry("release", "B1"))
	body := "We rolled back the release."
	r := a.Analyze(body)

	var found *text.Token
	for i, tok := range r.Tokens {
		if tok.Headword == "roll back" {
			found = &r.Tokens[i]
		}
		if tok.Headword == "roll" {
			t.Fatalf("matched the shorter entry 'roll' for %q, want 'roll back'", tok.Text)
		}
	}
	if found == nil {
		t.Fatalf("no token matched 'roll back' in %+v", r.Tokens)
	}
	if found.Text != "rolled back" || body[found.Start:found.End] != "rolled back" || found.Level != "C1" {
		t.Fatalf("multi-word token = %+v, want text/span 'rolled back' at level C1", *found)
	}
}

func TestAnalyzeShouldExcludeNamesNumbersAndURLsFromCoverage(t *testing.T) {
	a := lexicon(t, entry("pay", "A1"), entry("dollar", "A1"), entry("at", "A1"))
	r := a.Analyze("Yesterday Alice paid 20 dollars at https://example.com/shop.")

	counted := 0
	for _, tok := range r.Tokens {
		if tok.Counted {
			counted++
		}
		switch tok.Text {
		case "Alice":
			if tok.Kind != text.KindName || tok.Counted {
				t.Errorf("Alice = %+v, want a name that is not counted", tok)
			}
		case "20":
			if tok.Kind != text.KindNumber || tok.Counted {
				t.Errorf("20 = %+v, want a number that is not counted", tok)
			}
		case "https://example.com/shop":
			if tok.Kind != text.KindURL || tok.Counted {
				t.Errorf("url = %+v, want a URL that is not counted", tok)
			}
		}
	}
	// Yesterday (unknown word, still counted), paid, dollars, at.
	if counted != 4 {
		t.Fatalf("counted tokens = %d, want 4", counted)
	}
}

func TestLevelShouldBeLowestLevelReachingTheCoverageThreshold(t *testing.T) {
	a := lexicon(t, entry("go", "A1"), entry("abandon", "B2"))
	nineteenEasy := strings.Repeat("go ", 19)

	if got := a.Analyze(nineteenEasy + "abandon").Level(0.95); got != "A1" {
		t.Errorf("19 A1 + 1 B2 at 0.95 → %s, want A1 (A1 covers exactly 95%%)", got)
	}
	if got := a.Analyze(strings.Repeat("go ", 18) + "abandon abandon").Level(0.95); got != "B2" {
		t.Errorf("18 A1 + 2 B2 at 0.95 → %s, want B2 (A1 covers only 90%%)", got)
	}
}

func TestAnalyzeShouldRecordSpanAndSentenceForEveryToken(t *testing.T) {
	a := lexicon(t, entry("go", "A1"), entry("home", "A1"))
	body := "I go. She went home."
	r := a.Analyze(body)

	for _, tok := range r.Tokens {
		if body[tok.Start:tok.End] != tok.Text {
			t.Errorf("token %q span [%d:%d] = %q", tok.Text, tok.Start, tok.End, body[tok.Start:tok.End])
		}
		if tok.Text == "went" {
			if tok.Sentence < 0 || tok.Sentence >= len(r.Sentences) || r.Sentences[tok.Sentence] != "She went home." {
				t.Errorf("went sentence = %d (%v), want 'She went home.'", tok.Sentence, r.Sentences)
			}
		}
	}
}
