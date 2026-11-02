package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	fsrs "github.com/open-spaced-repetition/go-fsrs/v3"

	"github.com/sorenhoang/sr-english/api/internal/vocab"
)

type reviewHandler struct {
	vocab     vocab.Repository
	scheduler *fsrs.FSRS
}

type reviewRequest struct {
	SenseID string `json:"sense_id"`
	Rating  int    `json:"rating"`
}

func (h reviewHandler) register(r chi.Router) {
	r.Post("/review", h.review)
}

func (h reviewHandler) review(w http.ResponseWriter, r *http.Request) {
	var req reviewRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Rating < int(fsrs.Again) || req.Rating > int(fsrs.Easy) {
		writeError(w, http.StatusBadRequest, "validation", "rating is invalid")
		return
	}
	if _, err := uuid.Parse(req.SenseID); err != nil {
		writeError(w, http.StatusBadRequest, "validation", "sense_id is invalid")
		return
	}
	card, err := h.vocab.Review(r.Context(), h.scheduler, currentUser(r).ID, req.SenseID, fsrs.Rating(req.Rating))
	if errors.Is(err, vocab.ErrSenseNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "sense not found")
		return
	}
	if err != nil {
		internalError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, card)
}
