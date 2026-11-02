package httpapi

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sorenhoang/sr-english/api/internal/config"
	"github.com/sorenhoang/sr-english/api/internal/content"
	"github.com/sorenhoang/sr-english/api/internal/settings"
	"github.com/sorenhoang/sr-english/api/internal/vocab"
)

const (
	taskVocabReview = "vocab_review"
	taskVocabNew    = "vocab_new"
	taskReading     = content.KindReading
	taskListening   = content.KindListening
)

type task struct {
	Type  string `json:"type"`
	Items any    `json:"items"`
}

type sessionHandler struct {
	settings settings.Repository
	vocab    vocab.Repository
	content  content.Repository
	learning config.Learning
}

func (h sessionHandler) register(r chi.Router) {
	r.Get("/session/today", h.today)
}

func (h sessionHandler) today(w http.ResponseWriter, r *http.Request) {
	ctx, userID := r.Context(), currentUser(r).ID
	// ?extra=N asks for N more new words on top of today's remaining quota ("học thêm").
	extra := 0
	if v := r.URL.Query().Get("extra"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > h.learning.DailyNewMax {
			writeError(w, http.StatusBadRequest, "validation", fmt.Sprintf("extra must be between 0 and %d", h.learning.DailyNewMax))
			return
		}
		extra = n
	}
	s, err := h.settings.Get(ctx, userID)
	if err != nil {
		internalError(w, err)
		return
	}
	now := time.Now()
	due, err := h.vocab.Due(ctx, userID, now, h.learning.MaxReviewsPerDay)
	if err != nil {
		internalError(w, err)
		return
	}
	learned, err := h.vocab.LearnedSince(ctx, userID, startOfDay(now, s.Timezone))
	if err != nil {
		internalError(w, err)
		return
	}
	fresh, err := h.vocab.New(ctx, userID, s.Level, max(s.DailyNew-learned, 0)+extra)
	if err != nil {
		internalError(w, err)
		return
	}
	tasks := []task{{Type: taskVocabReview, Items: due}, {Type: taskVocabNew, Items: fresh}}
	for _, kind := range []string{taskReading, taskListening} {
		suggestion, err := h.content.Suggest(ctx, userID, s.Level, kind)
		if err != nil {
			internalError(w, err)
			return
		}
		if len(suggestion) > 0 {
			tasks = append(tasks, task{Type: kind, Items: suggestion})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

// startOfDay is midnight of now's calendar day in the user's timezone (UTC if the name is unknown).
func startOfDay(now time.Time, timezone string) time.Time {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Settings validation only stores valid zones, so this means tzdata is broken; don't fail the request.
		slog.Warn("unknown timezone, using UTC", "timezone", timezone, "err", err)
		loc = time.UTC
	}
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
}
