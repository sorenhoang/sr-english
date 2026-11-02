package httpapi

import (
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"

	"github.com/sorenhoang/sr-english/api/internal/settings"
	"github.com/sorenhoang/sr-english/api/internal/vocab"
)

// maxPageSize bounds list endpoints so one request can't pull a whole table.
const maxPageSize = 100

type pageRequest struct {
	Offset int64 `json:"offset"`
	Limit  int   `json:"limit"`
}

func (p pageRequest) validate() string {
	if p.Offset < 0 {
		return "offset must be ≥ 0"
	}
	if p.Limit < 1 || p.Limit > maxPageSize {
		return "limit must be between 1 and 100"
	}
	return ""
}

type page[T any] struct {
	Items   []T   `json:"items"`
	Total   int64 `json:"total"`
	Offset  int64 `json:"offset"`
	Limit   int   `json:"limit"`
	HasMore bool  `json:"has_more"`
}

func newPage[T any](items []T, total int64, p pageRequest) page[T] {
	return page[T]{Items: items, Total: total, Offset: p.Offset, Limit: p.Limit, HasMore: p.Offset+int64(len(items)) < total}
}

type wordsHandler struct {
	vocab vocab.Repository
}

type wordsListRequest struct {
	pageRequest
	Level string `json:"level"`
}

func (h wordsHandler) register(r chi.Router) {
	r.Post("/words/list", h.list)
}

func (h wordsHandler) list(w http.ResponseWriter, r *http.Request) {
	var req wordsListRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if msg := req.validate(); msg != "" {
		writeError(w, http.StatusBadRequest, "validation", msg)
		return
	}
	if req.Level != "" && !slices.Contains(settings.Levels, req.Level) {
		writeError(w, http.StatusBadRequest, "validation", "level is invalid")
		return
	}
	words, total, err := h.vocab.ListWords(r.Context(), currentUser(r).ID, req.Level, req.Offset, req.Limit)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newPage(words, total, req.pageRequest))
}
