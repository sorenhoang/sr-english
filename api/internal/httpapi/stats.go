package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sorenhoang/sr-english/api/internal/settings"
	"github.com/sorenhoang/sr-english/api/internal/stats"
)

type statsHandler struct {
	settings settings.Repository
	stats    stats.Repository
}

func (h statsHandler) register(r chi.Router) {
	r.Get("/stats", h.get)
}

func (h statsHandler) get(w http.ResponseWriter, r *http.Request) {
	ctx, userID := r.Context(), currentUser(r).ID
	s, err := h.settings.Get(ctx, userID)
	if err != nil {
		internalError(w, err)
		return
	}
	loc, err := time.LoadLocation(s.Timezone)
	if err != nil {
		slog.Warn("unknown timezone, using UTC", "timezone", s.Timezone, "err", err)
		loc = time.UTC
	}
	out, err := h.stats.Get(ctx, userID, time.Now(), loc)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
