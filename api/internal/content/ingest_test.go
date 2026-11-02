package content_test

import (
	"context"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sorenhoang/sr-english/api/internal/content"
	"github.com/sorenhoang/sr-english/api/internal/testdb"
)

const monarchURL = "https://learningenglish.voanews.com/a/monarch-butterfly-count-nears-30-year-low/7966513.html"

func seedWords(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, w := range []struct{ word, pos, level string }{
		{"number", "noun", "A1"}, {"group", "noun", "A1"}, {"say", "verb", "A1"}, {"butterfly", "noun", "A2"},
		{"winter", "noun", "A1"}, {"drop", "verb", "A2"}, {"low", "adjective", "A1"},
	} {
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO senses (headword, pos, cefr_level, variants, source) VALUES ($1, $2, $3, ARRAY[$1], 'cefrj')`,
			w.word, w.pos, w.level); err != nil {
			t.Fatalf("seed %s: %v", w.word, err)
		}
	}
}

func ingester(t *testing.T, pool *pgxpool.Pool) *content.Ingester {
	t.Helper()
	ing, err := content.NewIngester(context.Background(), pool, 0.95)
	if err != nil {
		t.Fatalf("NewIngester: %v", err)
	}
	return ing
}

type passageRow struct {
	title, body, license, level string
	audio                       *string
	kinds, topics               []string
	published                   time.Time
	tokens                      int
}

func loadPassage(t *testing.T, pool *pgxpool.Pool, url string) passageRow {
	t.Helper()
	var p passageRow
	if err := pool.QueryRow(context.Background(), `
		SELECT title, body, license, cefr_level, audio_url, kinds, topics, published_at, jsonb_array_length(tokens)
		FROM passages WHERE source_url = $1 AND deleted_at IS NULL`, url).
		Scan(&p.title, &p.body, &p.license, &p.level, &p.audio, &p.kinds, &p.topics, &p.published, &p.tokens); err != nil {
		t.Fatalf("load passage %s: %v", url, err)
	}
	return p
}

func TestIngestShouldStoreTitleBodyAudioDateSourceAndLicenseFromAVOAPage(t *testing.T) {
	pool := testdb.New(t)
	seedWords(t, pool)
	html, err := os.ReadFile("testdata/voa_monarch.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	if _, err := ingester(t, pool).Ingest(context.Background(), monarchURL, string(html)); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	p := loadPassage(t, pool, monarchURL)
	if p.title != "Monarch Butterfly Count Nears 30-Year Low" {
		t.Errorf("title = %q", p.title)
	}
	if !strings.HasPrefix(p.body, "An American nonprofit group says") ||
		strings.Contains(p.body, "No media source") || strings.Contains(p.body, "Words in This Story") {
		t.Errorf("body = %.120q…, want the story text only", p.body)
	}
	if p.audio == nil || !strings.HasSuffix(*p.audio, ".mp3") {
		t.Errorf("audio_url = %v, want an mp3", p.audio)
	}
	if !p.published.Equal(time.Date(2025, 2, 15, 21, 55, 0, 0, time.UTC)) {
		t.Errorf("published_at = %v, want 2025-02-15T21:55Z", p.published)
	}
	if p.license != "Public domain · learningenglish.voanews.com" {
		t.Errorf("license = %q", p.license)
	}
	slices.Sort(p.kinds)
	if !slices.Equal(p.kinds, []string{"listening", "reading"}) {
		t.Errorf("kinds = %v, want listening + reading (page has audio)", p.kinds)
	}
	if p.level == "" || p.tokens == 0 {
		t.Errorf("level %q tokens %d, want an analyzed passage", p.level, p.tokens)
	}
}

func page(title, category, body string, withAudio bool) string {
	audio := ""
	if withAudio {
		audio = `<audio src="https://voa-audio.voanews.eu/vle/x.mp3"></audio>`
	}
	return `<html><head><meta name="keywords" content="` + category + `" /></head><body>
<h1> ` + title + ` </h1><time pubdate="pubdate" datetime="2024-05-01T10:00:00&#x2B;00:00">May 1</time>
<div class="wsw">` + audio + `<p>` + body + `</p><h2><strong>Words in This Story</strong></h2><p>word – n. meaning</p></div></body></html>`
}

func TestIngestShouldMarkPagesWithoutAudioAsReadingOnly(t *testing.T) {
	pool := testdb.New(t)
	url := "https://learningenglish.voanews.com/a/no-audio/1.html"
	if _, err := ingester(t, pool).Ingest(context.Background(), url, page("Quiet Story", "As It Is", "A group says the number is low.", false)); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	p := loadPassage(t, pool, url)
	if !slices.Equal(p.kinds, []string{"reading"}) || p.audio != nil {
		t.Fatalf("kinds %v audio %v, want reading only and no audio", p.kinds, p.audio)
	}
}

func TestIngestShouldNotDuplicateWhenTheSameURLIsIngestedAgain(t *testing.T) {
	pool := testdb.New(t)
	ing := ingester(t, pool)
	url := "https://learningenglish.voanews.com/a/twice/2.html"
	for range 2 {
		if _, err := ing.Ingest(context.Background(), url, page("Twice", "As It Is", "The group says hello.", true)); err != nil {
			t.Fatalf("Ingest: %v", err)
		}
	}
	var n int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM passages WHERE source_url = $1`, url).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Fatalf("passages for url = %d, want 1", n)
	}
}

func TestIngestShouldAssignCambridgeTopicsFromCategoryThenKeywordsThenGeneral(t *testing.T) {
	pool := testdb.New(t)
	ing := ingester(t, pool)
	cases := []struct{ url, category, body, want string }{
		{"https://x/a/cat/1.html", "Health & Lifestyle", "People like to talk.", "health"},
		{"https://x/a/kw/2.html", "As It Is", "Butterflies and other animals live in the forest. The animals need trees.", "the-natural-world"},
		{"https://x/a/none/3.html", "As It Is", "Someone said something yesterday.", "general"},
	}
	for _, c := range cases {
		if _, err := ing.Ingest(context.Background(), c.url, page("Story", c.category, c.body, false)); err != nil {
			t.Fatalf("Ingest %s: %v", c.url, err)
		}
		if p := loadPassage(t, pool, c.url); !slices.Contains(p.topics, c.want) {
			t.Errorf("%s: topics = %v, want to include %q", c.category, p.topics, c.want)
		}
	}
}
