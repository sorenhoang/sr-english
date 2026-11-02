package httpapi

import (
	"errors"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/sorenhoang/sr-english/api/internal/content"
	"github.com/sorenhoang/sr-english/api/internal/settings"
)

type passagesHandler struct {
	content content.Repository
}

type passagesListRequest struct {
	pageRequest
	Level string `json:"level"`
	Topic string `json:"topic"`
	Kind  string `json:"kind"`
}

func (h passagesHandler) register(r chi.Router) {
	r.Post("/passages/list", h.list)
	r.Get("/passage", h.get)
	r.Post("/passage/complete", h.complete)
	r.Post("/passage/save-word", h.saveWord)
}

type saveWordRequest struct {
	PassageID  string `json:"passage_id"`
	TokenIndex *int   `json:"token_index"`
}

func (h passagesHandler) saveWord(w http.ResponseWriter, r *http.Request) {
	var req saveWordRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if _, err := uuid.Parse(req.PassageID); err != nil {
		writeError(w, http.StatusBadRequest, "validation", "passage_id is invalid")
		return
	}
	if req.TokenIndex == nil {
		writeError(w, http.StatusBadRequest, "validation", "token_index is required")
		return
	}
	saved, err := h.content.SaveWord(r.Context(), currentUser(r).ID, req.PassageID, *req.TokenIndex)
	switch {
	case errors.Is(err, content.ErrPassageNotFound):
		writeError(w, http.StatusNotFound, "not_found", "passage not found")
	case errors.Is(err, content.ErrBadToken):
		writeError(w, http.StatusBadRequest, "validation", "token_index is invalid")
	case err != nil:
		internalError(w, err)
	default:
		writeJSON(w, http.StatusOK, saved)
	}
}

func (h passagesHandler) list(w http.ResponseWriter, r *http.Request) {
	var req passagesListRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	msg := req.validate()
	switch {
	case msg != "":
	case req.Level != "" && !slices.Contains(settings.Levels, req.Level):
		msg = "level is invalid"
	case req.Topic != "" && !slices.Contains(content.Topics, req.Topic):
		msg = "topic is invalid"
	case req.Kind != "" && req.Kind != content.KindReading && req.Kind != content.KindListening:
		msg = "kind is invalid"
	}
	if msg != "" {
		writeError(w, http.StatusBadRequest, "validation", msg)
		return
	}
	items, total, err := h.content.List(r.Context(), content.Filter{Level: req.Level, Topic: req.Topic, Kind: req.Kind}, req.Offset, req.Limit)
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, newPage(items, total, req.pageRequest))
}

func (h passagesHandler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := passageID(w, r)
	if !ok {
		return
	}
	p, err := h.content.Get(r.Context(), currentUser(r).ID, id)
	if errors.Is(err, content.ErrPassageNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "passage not found")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h passagesHandler) complete(w http.ResponseWriter, r *http.Request) {
	id, ok := passageID(w, r)
	if !ok {
		return
	}
	err := h.content.Complete(r.Context(), currentUser(r).ID, id)
	if errors.Is(err, content.ErrPassageNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "passage not found")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"completed": true})
}

func passageID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.URL.Query().Get("id")
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, http.StatusBadRequest, "validation", "id is invalid")
		return "", false
	}
	return id, true
}
