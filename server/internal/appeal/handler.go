package appeal

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/liveops"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

const MaxBodyBytes = 16 * 1024

type Submitter func(ctx context.Context, account string, sanctionID int64, text string, now time.Time) (Appeal, error)

type Handler struct {
	verifier token.AppealVerifier
	submit   Submitter
	settings func() liveops.Appeals
	logger   *slog.Logger
	now      func() time.Time
	mux      *http.ServeMux
}

type submitRequest struct {
	SanctionID int64  `json:"sanction_id"`
	Text       string `json:"text"`
}

type submittedResponse struct {
	ID         int64     `json:"id"`
	SanctionID int64     `json:"sanction_id"`
	State      string    `json:"state"`
	CreatedAt  time.Time `json:"created_at"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewHandler(verifier token.AppealVerifier, submit Submitter, settings func() liveops.Appeals, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, submit: submit, settings: settings, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /appeals", h.file)
	return h
}

func StoreSubmitter(db Beginner) Submitter {
	return func(ctx context.Context, account string, sanctionID int64, text string, now time.Time) (Appeal, error) {
		return Submit(ctx, db, account, sanctionID, text, now)
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) file(w http.ResponseWriter, r *http.Request) {
	claims, err := token.AuthorizeAppeal(h.verifier, r)
	switch {
	case errors.Is(err, token.ErrExpired):
		reply(w, http.StatusUnauthorized, errorResponse{Error: "appeal_token_expired"})
		return
	case err != nil:
		reply(w, http.StatusUnauthorized, errorResponse{Error: "appeal_token_rejected"})
		return
	}
	var request submitRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes)).Decode(&request); err != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: "malformed"})
		return
	}
	if request.SanctionID != claims.Sanction {
		reply(w, http.StatusForbidden, errorResponse{Error: "sanction_mismatch"})
		return
	}
	text, err := CleanText(request.Text, h.settings().TextMaxLength)
	if err != nil {
		h.refuse(w, err)
		return
	}
	filed, err := h.submit(r.Context(), claims.Account, request.SanctionID, text, h.now())
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusCreated, submittedResponse{ID: filed.ID, SanctionID: filed.SanctionID, State: filed.State, CreatedAt: filed.CreatedAt})
}

func (h *Handler) refuse(w http.ResponseWriter, err error) {
	for _, known := range []struct {
		err    error
		status int
		code   string
	}{
		{ErrTextRequired, http.StatusBadRequest, ErrTextRequired.Error()},
		{ErrTextTooLong, http.StatusBadRequest, ErrTextTooLong.Error()},
		{sanction.ErrUnknown, http.StatusNotFound, "unknown_sanction"},
		{ErrNotAppealable, http.StatusConflict, ErrNotAppealable.Error()},
		{ErrEnded, http.StatusConflict, ErrEnded.Error()},
		{ErrOpen, http.StatusConflict, ErrOpen.Error()},
		{ErrRejected, http.StatusConflict, ErrRejected.Error()},
	} {
		if errors.Is(err, known.err) {
			reply(w, known.status, errorResponse{Error: known.code})
			return
		}
	}
	h.logger.Error("appeal failed", "error", err)
	reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
