package mission

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type Missions interface {
	Board(ctx context.Context, account string, now time.Time) (Board, error)
	Reroll(ctx context.Context, account, missionID string, now time.Time) (Board, error)
}

type Handler struct {
	verifier token.Verifier
	store    Missions
	logger   *slog.Logger
	now      func() time.Time
	mux      *http.ServeMux
}

func NewHandler(verifier token.Verifier, store Missions, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, store: store, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /missions", h.board)
	h.mux.HandleFunc("POST /missions/reroll", h.reroll)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type errorResponse struct {
	Error string `json:"error"`
}

type rerollRequest struct {
	Mission string `json:"mission"`
}

func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request) (token.Claims, bool) {
	claims, err := token.Authorize(h.verifier, r)
	switch {
	case errors.Is(err, token.ErrExpired):
		reply(w, http.StatusUnauthorized, errorResponse{Error: "access_expired"})
		return token.Claims{}, false
	case err != nil:
		reply(w, http.StatusUnauthorized, errorResponse{Error: "access_rejected"})
		return token.Claims{}, false
	}
	return claims, true
}

func (h *Handler) board(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	board, err := h.store.Board(r.Context(), claims.Account, h.now())
	h.answer(w, board, err)
}

func (h *Handler) reroll(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	var request rerollRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&request); err != nil || request.Mission == "" {
		reply(w, http.StatusBadRequest, errorResponse{Error: "malformed"})
		return
	}
	board, err := h.store.Reroll(r.Context(), claims.Account, request.Mission, h.now())
	h.answer(w, board, err)
}

func (h *Handler) answer(w http.ResponseWriter, board Board, err error) {
	switch {
	case err == nil:
		reply(w, http.StatusOK, board)
	case errors.Is(err, ErrNotAssigned), errors.Is(err, ErrRerollUsed), errors.Is(err, ErrNoReplacement):
		reply(w, http.StatusConflict, errorResponse{Error: err.Error()})
	case errors.Is(err, ErrNoCatalog):
		reply(w, http.StatusServiceUnavailable, errorResponse{Error: ErrNoCatalog.Error()})
	default:
		h.logger.Error("mission request failed", "error", err)
		reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
	}
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
