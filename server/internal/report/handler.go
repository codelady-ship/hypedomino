package report

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

const MaxBodyBytes = 512

type Handler struct {
	verifier token.Verifier
	store    *Store
	logger   *slog.Logger
	now      func() time.Time
	mux      *http.ServeMux
}

type errorResponse struct {
	Error string `json:"error"`
}

type receivedResponse struct {
	Status string `json:"status"`
}

func NewHandler(verifier token.Verifier, store *Store, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, store: store, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /reports", h.file)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) file(w http.ResponseWriter, r *http.Request) {
	claims, err := token.Authorize(h.verifier, r)
	switch {
	case errors.Is(err, token.ErrExpired):
		reply(w, http.StatusUnauthorized, errorResponse{Error: "access_expired"})
		return
	case err != nil:
		reply(w, http.StatusUnauthorized, errorResponse{Error: "access_rejected"})
		return
	}
	var report Report
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes)).Decode(&report); err != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: ErrReason.Error()})
		return
	}
	if err := h.store.File(r.Context(), claims.Account, report, h.now()); err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusAccepted, receivedResponse{Status: "received"})
}

func (h *Handler) refuse(w http.ResponseWriter, err error) {
	for _, known := range []struct {
		err    error
		status int
	}{
		{ErrUnknownPlayer, http.StatusNotFound},
		{ErrSelf, http.StatusBadRequest},
		{ErrReason, http.StatusBadRequest},
		{ErrMatch, http.StatusBadRequest},
	} {
		if errors.Is(err, known.err) {
			reply(w, known.status, errorResponse{Error: known.err.Error()})
			return
		}
	}
	h.logger.Error("report failed", "error", err)
	reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
