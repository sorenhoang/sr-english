// Package text turns a passage into tokens mapped to CEFR senses and estimates its level.
package text

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/aaaton/golem/v4"
	"github.com/aaaton/golem/v4/dicts/en"
)

type Kind string

const (
	KindWord   Kind = "word"
	KindName   Kind = "name"
	KindNumber Kind = "number"
	KindURL    Kind = "url"
)

// Levels in ascending order; a passage's level is the first one that reaches the coverage threshold.
var Levels = []string{"A1", "A2", "B1", "B2", "C1", "C2"}

// Entry is one sense of the word list, as the analyzer needs it.
type Entry struct {
	SenseID  string
	Headword string
	Variants []string
	POS      string
	Level    string
}

// Token offsets: Start/End are UTF-8 byte offsets (Go), Start16/End16 are UTF-16 code units (JavaScript).
type Token struct {
	Text     string `json:"text"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Start16  int    `json:"start16"`
	End16    int    `json:"end16"`
	Kind     Kind   `json:"kind"`
	Headword string `json:"headword,omitempty"`
	SenseID  string `json:"sense_id,omitempty"`
	Level    string `json:"level,omitempty"`
	Counted  bool   `json:"counted"`
	Sentence int    `json:"sentence"`
}

type Result struct {
	Tokens    []Token
	Sentences []string
}

type Analyzer struct {
	lem      *golem.Lemmatizer
	single   map[string]Entry
	multi    map[string]Entry
	maxWords int
}

func NewAnalyzer(entries []Entry) (*Analyzer, error) {
	lem, err := golem.New(en.New())
	if err != nil {
		return nil, fmt.Errorf("load lemmatizer: %w", err)
	}
	a := &Analyzer{lem: lem, single: map[string]Entry{}, multi: map[string]Entry{}, maxWords: 1}
	for _, e := range entries {
		forms := append([]string{e.Headword}, e.Variants...)
		for _, f := range forms {
			key := strings.ToLower(strings.Join(strings.Fields(f), " "))
			if key == "" {
				continue
			}
			target := a.single
			if n := len(strings.Fields(key)); n > 1 {
				target = a.multi
				a.maxWords = max(a.maxWords, n)
			}
			// Several senses can share a form (noun/verb); keep the easiest so coverage isn't overstated.
			if old, ok := target[key]; !ok || levelRank(e.Level) < levelRank(old.Level) {
				target[key] = e
			}
		}
	}
	return a, nil
}

var tokenRe = regexp.MustCompile(`https?://\S+|\d+(?:[.,]\d+)*(?:st|nd|rd|th|s)?|\p{L}+(?:['’-]\p{L}+)*`)

// abbreviations never end a sentence even when followed by a capital ("Mr. Smith", "U.S. Army").
var abbreviations = map[string]bool{
	"mr": true, "mrs": true, "ms": true, "dr": true, "jr": true, "prof": true,
	"u.s": true, "u.k": true, "u.n": true, "e.g": true, "i.e": true, "etc": true, "vs": true,
	"a.m": true, "p.m": true, "jan": true, "feb": true, "aug": true, "sept": true, "oct": true, "nov": true, "dec": true,
}

func (a *Analyzer) Analyze(body string) Result {
	sentences, spans := splitSentences(body)
	u16 := utf16Index(body)
	raw := tokenRe.FindAllStringIndex(body, -1)
	var tokens []Token
	for i := 0; i < len(raw); i++ {
		start, end := raw[i][0], raw[i][1]
		word := body[start:end]
		tok := Token{Text: word, Start: start, End: end, Sentence: sentenceOf(spans, start)}
		switch {
		case strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://"):
			end = start + len(strings.TrimRight(word, ".,;:!?)\"'"))
			tok.Text, tok.End, tok.Kind = body[start:end], end, KindURL
		case unicode.IsDigit([]rune(word)[0]):
			tok.Kind = KindNumber
		default:
			if n, e, ok := a.matchMulti(body, raw, i); ok {
				last := raw[i+n-1][1]
				tok.Text, tok.End = body[start:last], last
				tok.Kind, tok.Counted = KindWord, true
				tok.Headword, tok.SenseID, tok.Level = e.Headword, e.SenseID, e.Level
				i += n - 1
				break
			}
			e, known := a.lookup(word)
			if !known && isCapitalized(word) && stem(word) != "I" && !sentenceInitial(spans, start) {
				tok.Kind = KindName
				break
			}
			tok.Kind, tok.Counted = KindWord, true
			if known {
				tok.Headword, tok.SenseID, tok.Level = e.Headword, e.SenseID, e.Level
			}
		}
		tok.Start16, tok.End16 = u16[tok.Start], u16[tok.End]
		tokens = append(tokens, tok)
	}
	return Result{Tokens: tokens, Sentences: sentences}
}

// Level is the lowest CEFR level whose known words cover at least threshold of the counted tokens.
func (r Result) Level(threshold float64) string {
	counted := 0
	known := make([]int, len(Levels))
	for _, t := range r.Tokens {
		if !t.Counted {
			continue
		}
		counted++
		if rank := levelRank(t.Level); rank < len(Levels) {
			known[rank]++
		}
	}
	if counted == 0 {
		return Levels[0]
	}
	cumulative := 0
	for i, lvl := range Levels {
		cumulative += known[i]
		if float64(cumulative)/float64(counted) >= threshold {
			return lvl
		}
	}
	return Levels[len(Levels)-1]
}

func (a *Analyzer) lookup(word string) (Entry, bool) {
	lower := strings.ToLower(strings.ReplaceAll(word, "’", "'"))
	for _, form := range []string{lower, strings.ToLower(stem(word))} {
		if e, ok := a.single[form]; ok {
			return e, true
		}
		for _, lemma := range a.lem.Lemmas(form) {
			if e, ok := a.single[lemma]; ok {
				return e, true
			}
		}
	}
	return Entry{}, false
}

// stem drops a possessive or contraction ending: "children’s" → "children", "I’m" → "I", "don't" → "do".
func stem(word string) string {
	w := strings.ReplaceAll(word, "’", "'")
	switch strings.ToLower(w) {
	case "can't":
		return "can"
	case "won't":
		return "will"
	case "shan't":
		return "shall"
	}
	if base, ok := strings.CutSuffix(w, "n't"); ok && base != "" {
		return base
	}
	if i := strings.Index(w, "'"); i > 0 {
		return w[:i]
	}
	return w
}

// matchMulti tries the longest multi-word entry starting at raw[i], over words separated only by spaces.
func (a *Analyzer) matchMulti(body string, raw [][]int, i int) (int, Entry, bool) {
	for n := min(a.maxWords, len(raw)-i); n >= 2; n-- {
		words := make([]string, 0, n)
		ok := true
		for k := 0; k < n; k++ {
			if k > 0 && strings.TrimSpace(body[raw[i+k-1][1]:raw[i+k][0]]) != "" {
				ok = false
				break
			}
			words = append(words, strings.ToLower(body[raw[i+k][0]:raw[i+k][1]]))
		}
		if !ok {
			continue
		}
		for _, key := range a.multiKeys(words) {
			if e, found := a.multi[key]; found {
				return n, e, true
			}
		}
	}
	return 0, Entry{}, false
}

// multiKeys covers inflection on the first word ("rolled back") or on every word.
func (a *Analyzer) multiKeys(words []string) []string {
	firstLemma := append([]string{a.lem.Lemma(words[0])}, words[1:]...)
	allLemmas := make([]string, len(words))
	for i, w := range words {
		allLemmas[i] = a.lem.Lemma(w)
	}
	return []string{strings.Join(words, " "), strings.Join(firstLemma, " "), strings.Join(allLemmas, " ")}
}

// splitSentences ends a sentence at . ! ? (plus closing quotes) followed by whitespace and a capital,
// digit or opening quote, unless the word before the dot is a known abbreviation.
func splitSentences(body string) ([]string, [][2]int) {
	var sentences []string
	var spans [][2]int
	emit := func(from, to int) {
		chunk := body[from:to]
		trimmed := strings.TrimSpace(chunk)
		if trimmed == "" {
			return
		}
		start := from + strings.Index(chunk, trimmed)
		sentences = append(sentences, trimmed)
		spans = append(spans, [2]int{start, start + len(trimmed)})
	}
	from := 0
	for i := 0; i < len(body); i++ {
		if !strings.ContainsRune(".!?", rune(body[i])) {
			continue
		}
		end := i + 1
		for end < len(body) && strings.ContainsRune(".!?\"')", rune(body[end])) {
			end++
		}
		for _, q := range []string{"”", "’"} {
			if strings.HasPrefix(body[end:], q) {
				end += len(q)
			}
		}
		if end < len(body) && !startsNewSentence(body[end:]) {
			i = end - 1
			continue
		}
		if body[i] == '.' && abbreviations[strings.ToLower(lastWord(body[from:i]))] {
			i = end - 1
			continue
		}
		emit(from, end)
		from = end
		i = end - 1
	}
	emit(from, len(body))
	return sentences, spans
}

func startsNewSentence(rest string) bool {
	trimmed := strings.TrimLeftFunc(rest, unicode.IsSpace)
	if len(trimmed) == len(rest) || trimmed == "" {
		return trimmed == ""
	}
	r := []rune(trimmed)[0]
	return unicode.IsUpper(r) || unicode.IsDigit(r) || strings.ContainsRune("\"“‘'(", r)
}

// lastWord returns the token right before a dot, keeping inner dots ("U.S").
func lastWord(s string) string {
	i := strings.LastIndexFunc(s, func(r rune) bool { return unicode.IsSpace(r) || r == '(' || r == '"' || r == '“' })
	return s[i+1:]
}

// utf16Index maps every byte offset that starts a rune (and len(s)) to its UTF-16 offset.
func utf16Index(s string) []int {
	idx := make([]int, len(s)+1)
	units := 0
	for i, r := range s {
		idx[i] = units
		if r >= 0x10000 {
			units += 2
		} else {
			units++
		}
	}
	idx[len(s)] = units
	return idx
}

func sentenceOf(spans [][2]int, pos int) int {
	for i, s := range spans {
		if pos >= s[0] && pos < s[1] {
			return i
		}
	}
	return -1
}

func sentenceInitial(spans [][2]int, pos int) bool {
	for _, s := range spans {
		if s[0] == pos {
			return true
		}
	}
	return false
}

func isCapitalized(word string) bool {
	for _, r := range word {
		return unicode.IsUpper(r)
	}
	return false
}

// levelRank orders CEFR levels; unknown or empty levels sort after C2.
func levelRank(level string) int {
	for i, l := range Levels {
		if l == level {
			return i
		}
	}
	return len(Levels)
}
