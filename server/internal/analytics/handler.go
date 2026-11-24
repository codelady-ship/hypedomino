package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/consent"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type Consents interface {
	Granted(ctx context.Context, account string, purpose consent.Purpose) (bool, error)
}

type Handler struct {
	verifier token.Verifier
	store    *Store
	consents Consents
	logger   *slog.Logger
	now      func() time.Time
	mux      *http.ServeMux
}

type errorResponse struct {
	Error string `json:"error"`
}

type acceptedResponse struct {
	Accepted int `json:"accepted"`
}

func NewHandler(verifier token.Verifier, store *Store, consents Consents, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, store: store, consents: consents, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /analytics/events", h.ingest)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) ingest(w http.ResponseWriter, r *http.Request) {
	claims, err := token.Authorize(h.verifier, r)
	switch {
	case errors.Is(err, token.ErrExpired):
		reply(w, http.StatusUnauthorized, errorResponse{Error: "access_expired"})
		return
	case err != nil:
		reply(w, http.StatusUnauthorized, errorResponse{Error: "access_rejected"})
		return
	}
	var batch Batch
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes)).Decode(&batch); err != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: ErrMalformed.Error()})
		return
	}
	accepted, err := batch.Accepted()
	if err != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}
	granted, err := h.consents.Granted(r.Context(), claims.Account, consent.Analytics)
	switch {
	case err != nil:
		h.logger.Error("analytics consent check failed", "error", err)
		reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
		return
	case !granted:
		reply(w, http.StatusForbidden, errorResponse{Error: ErrNoConsent.Error()})
		return
	}
	err = h.store.Record(r.Context(), claims.Account, accepted, h.now())
	switch {
	case errors.Is(err, ErrIDTaken):
		reply(w, http.StatusConflict, errorResponse{Error: err.Error()})
	case err != nil:
		h.logger.Error("analytics ingest failed", "error", err)
		reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
	default:
		reply(w, http.StatusOK, acceptedResponse{Accepted: len(accepted.Events)})
	}
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
