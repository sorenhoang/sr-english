package content

import (
	"errors"
	"html"
	"regexp"
	"strings"
	"time"
)

// page is what we extract from one VOA Learning English article.
type page struct {
	Title      string
	Body       string
	AudioURL   string
	Published  *time.Time
	Categories []string
}

// Selectors match VOA's article markup as of 2025 (h1 title, <time datetime>, <audio src>, div.wsw body).
var (
	titleRe    = regexp.MustCompile(`(?s)<h1[^>]*>(.*?)</h1>`)
	timeRe     = regexp.MustCompile(`<time[^>]*datetime="([^"]+)"`)
	audioRe    = regexp.MustCompile(`<audio[^>]*\ssrc="(https://[^"]+\.mp3)"`)
	keywordsRe = regexp.MustCompile(`<meta name="keywords" content="([^"]*)"`)
	paraRe     = regexp.MustCompile(`(?s)<p[^>]*>(.*?)</p>`)
	tagRe      = regexp.MustCompile(`<[^>]+>`)
	spaceRe    = regexp.MustCompile(`\s+`)
)

// bodyStops end the story text: the glossary and the page furniture that follow it.
var bodyStops = []string{"Words in This Story", "<div class=\"c-author", "Related"}

var skipParagraphs = map[string]bool{"No media source currently available": true}

func parseVOA(raw string) (page, error) {
	var p page
	if m := titleRe.FindStringSubmatch(raw); m != nil {
		p.Title = clean(m[1])
	}
	if p.Title == "" {
		return page{}, errors.New("page has no <h1> title")
	}
	if m := timeRe.FindStringSubmatch(raw); m != nil {
		if t, err := time.Parse(time.RFC3339, html.UnescapeString(m[1])); err == nil {
			utc := t.UTC()
			p.Published = &utc
		}
	}
	if m := audioRe.FindStringSubmatch(raw); m != nil {
		p.AudioURL = m[1]
	}
	if m := keywordsRe.FindStringSubmatch(raw); m != nil {
		for _, c := range strings.Split(html.UnescapeString(m[1]), ",") {
			if c = strings.TrimSpace(c); c != "" {
				p.Categories = append(p.Categories, c)
			}
		}
	}

	start := strings.Index(raw, `<div class="wsw">`)
	if start < 0 {
		return page{}, errors.New("page has no div.wsw story body")
	}
	story := raw[start:]
	for _, stop := range bodyStops {
		if i := strings.Index(story, stop); i > 0 {
			story = story[:i]
		}
	}
	var paras []string
	for _, m := range paraRe.FindAllStringSubmatch(story, -1) {
		if text := clean(m[1]); text != "" && !skipParagraphs[text] {
			paras = append(paras, text)
		}
	}
	if len(paras) == 0 {
		return page{}, errors.New("story body has no paragraphs")
	}
	p.Body = strings.Join(paras, "\n\n")
	return p, nil
}

func clean(fragment string) string {
	return strings.TrimSpace(spaceRe.ReplaceAllString(html.UnescapeString(tagRe.ReplaceAllString(fragment, " ")), " "))
}
