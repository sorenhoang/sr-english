package content

import (
	"regexp"
	"sort"
	"strings"
)

var wordRe = regexp.MustCompile(`\p{L}+`)

// Topics follow the Cambridge A2 Key / B1 Preliminary vocabulary-list topic names (slugs only).
const TopicGeneral = "general"

var Topics = []string{
	"clothes", "communication-and-technology", "education", "entertainment-and-media", "environment",
	"food-and-drink", "health", "hobbies-and-leisure", "house-and-home", "language", "personal-feelings",
	"places", "services", "shopping", "social-interaction", "sport", "the-natural-world", "transport",
	"travel-and-holidays", "weather", "work-and-jobs", TopicGeneral,
}

// categoryTopics maps VOA Learning English section names to Cambridge topics.
var categoryTopics = map[string]string{
	"health & lifestyle":      "health",
	"science & technology":    "communication-and-technology",
	"education":               "education",
	"student union":           "education",
	"arts & culture":          "entertainment-and-media",
	"english @ the movies":    "entertainment-and-media",
	"american stories":        "entertainment-and-media",
	"everyday grammar":        "language",
	"words and their stories": "language",
	"ask a teacher":           "language",
	"sports":                  "sport",
	"environment":             "environment",
	"economy":                 "work-and-jobs",
	"travel":                  "travel-and-holidays",
}

// keywordTopics are word prefixes (matched at the start of a word) that suggest a topic when the section name doesn't.
var keywordTopics = map[string][]string{
	"the-natural-world":            {"animal", "butterfl", "bird", "forest", "insect", "species", "wildlife", "ocean", "plant", "tree"},
	"environment":                  {"climate", "pollution", "carbon", "emission", "recycl", "warming"},
	"health":                       {"health", "doctor", "disease", "hospital", "medic", "patient", "vaccin"},
	"education":                    {"school", "student", "teacher", "universit", "college", "classroom"},
	"communication-and-technology": {"internet", "phone", "computer", "technolog", "robot", "online", "smartphone"},
	"food-and-drink":               {"food", "restaurant", "cook", "coffee", "rice", "meal"},
	"work-and-jobs":                {"job", "worker", "employ", "salary", "company", "business"},
	"sport":                        {"sport", "football", "soccer", "olympic", "athlete", "tennis"},
	"travel-and-holidays":          {"travel", "tourist", "touris", "vacation", "holiday", "airport"},
	"weather":                      {"weather", "storm", "rain", "snow", "hurricane", "temperature"},
	"entertainment-and-media":      {"film", "movie", "music", "singer", "television", "museum", "artist", "painting"},
	"transport":                    {"cars", "train", "buses", "traffic", "airline", "railway"},
	"shopping":                     {"shop", "price", "store", "buy"},
}

// minKeywordHits avoids tagging a story from one passing mention.
const minKeywordHits = 2

// maxTopics keeps tagging focused; a passage gets at most this many topics.
const maxTopics = 2

func assignTopics(categories []string, title, body string) []string {
	for _, c := range categories {
		if t, ok := categoryTopics[strings.ToLower(strings.TrimSpace(c))]; ok {
			return []string{t}
		}
	}
	words := wordRe.FindAllString(strings.ToLower(title+" "+body), -1)
	type hit struct {
		topic string
		n     int
	}
	var hits []hit
	for topic, keywords := range keywordTopics {
		n := 0
		for _, prefix := range keywords {
			for _, w := range words {
				if strings.HasPrefix(w, prefix) {
					n++
				}
			}
		}
		if n >= minKeywordHits {
			hits = append(hits, hit{topic, n})
		}
	}
	if len(hits) == 0 {
		return []string{TopicGeneral}
	}
	sort.Slice(hits, func(i, j int) bool {
		if hits[i].n != hits[j].n {
			return hits[i].n > hits[j].n
		}
		return hits[i].topic < hits[j].topic
	})
	out := []string{}
	for _, h := range hits[:min(maxTopics, len(hits))] {
		out = append(out, h.topic)
	}
	return out
}
