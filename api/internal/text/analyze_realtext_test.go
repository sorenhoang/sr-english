package text_test

import (
	"testing"
	"unicode/utf16"

	"github.com/sorenhoang/sr-english/api/internal/text"
)

func TestAnalyzeShouldGiveUTF16OffsetsThatMatchJavaScriptSlicing(t *testing.T) {
	a := lexicon(t, entry("go", "A1"), entry("home", "A1"))
	body := "“Café” — she said, then went home."
	units := utf16.Encode([]rune(body))
	r := a.Analyze(body)

	found := false
	for _, tok := range r.Tokens {
		got := string(utf16.Decode(units[tok.Start16:tok.End16]))
		if got != tok.Text {
			t.Errorf("token %q utf16 span [%d:%d] = %q", tok.Text, tok.Start16, tok.End16, got)
		}
		if tok.Text == "home" {
			found = true
		}
	}
	if !found {
		t.Fatalf("token 'home' missing from %+v", r.Tokens)
	}
}

func TestAnalyzeShouldNotSplitSentencesOnAbbreviationsOrDecimals(t *testing.T) {
	a := lexicon(t, entry("official", "B1"), entry("percent", "A2"))
	r := a.Analyze("U.S. officials said Monday. Prices rose 3.5 percent in Washington.")

	want := map[string]string{
		"officials": "U.S. officials said Monday.",
		"percent":   "Prices rose 3.5 percent in Washington.",
	}
	for _, tok := range r.Tokens {
		if sentence, ok := want[tok.Text]; ok {
			if tok.Sentence < 0 || r.Sentences[tok.Sentence] != sentence {
				t.Errorf("%q sentence = %v, want %q", tok.Text, r.Sentences, sentence)
			}
			delete(want, tok.Text)
		}
	}
	if len(want) != 0 {
		t.Fatalf("tokens missing: %v", want)
	}
}

func TestAnalyzeShouldTreatContractionsAndCurlyApostrophesAsWords(t *testing.T) {
	a := lexicon(t, entry("know", "A1"), entry("child", "A1"))
	r := a.Analyze("Well, I’m sure the children’s teacher knows it. In the 1990s we knew.")

	for _, tok := range r.Tokens {
		switch tok.Text {
		case "I’m":
			if tok.Kind != text.KindWord || !tok.Counted {
				t.Errorf("I’m = %+v, want a counted word, not a name", tok)
			}
		case "children’s":
			if tok.Headword != "child" {
				t.Errorf("children’s headword = %q, want child", tok.Headword)
			}
		case "s":
			t.Errorf("'1990s' produced a stray token %+v", tok)
		}
	}
}
