package httpapi

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sorenhoang/sr-english/api/internal/config"
	"github.com/sorenhoang/sr-english/api/internal/settings"
)

type settingsHandler struct {
	repo     settings.Repository
	learning config.Learning
}

func (h settingsHandler) register(r chi.Router) {
	r.Get("/settings", h.get)
	r.Post("/settings/update", h.update)
}

func (h settingsHandler) get(w http.ResponseWriter, r *http.Request) {
	s, err := h.repo.Get(r.Context(), currentUser(r).ID)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h settingsHandler) update(w http.ResponseWriter, r *http.Request) {
	var s settings.Settings
	if !decodeJSON(w, r, &s) {
		return
	}
	if msg := h.validate(s); msg != "" {
		writeError(w, http.StatusBadRequest, "validation", msg)
		return
	}
	if err := h.repo.Save(r.Context(), currentUser(r).ID, s); err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h settingsHandler) validate(s settings.Settings) string {
	if !slices.Contains(settings.Levels, s.Level) {
		return "level is invalid"
	}
	if s.Timezone == "" || s.Timezone == "Local" {
		return "timezone is invalid"
	}
	// Compare the canonical name: case-insensitive filesystems would otherwise accept "asia/ho_chi_minh".
	if loc, err := time.LoadLocation(s.Timezone); err != nil || loc.String() != s.Timezone {
		return "timezone is invalid"
	}
	if s.DailyNew < 1 || s.DailyNew > h.learning.DailyNewMax {
		return fmt.Sprintf("daily_new must be between 1 and %d", h.learning.DailyNewMax)
	}
	return ""
}

func internalError(w http.ResponseWriter, err error) {
	slog.Error("request failed", "err", err)
	writeError(w, http.StatusInternalServerError, "internal", "internal server error")
}
