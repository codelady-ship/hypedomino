package dailylogin

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type Logins interface {
	Status(ctx context.Context, account string, now time.Time) (Status, error)
	Claim(ctx context.Context, account string, now time.Time) (Status, error)
}

type Handler struct {
	verifier token.Verifier
	store    Logins
	logger   *slog.Logger
	now      func() time.Time
	mux      *http.ServeMux
}

func NewHandler(verifier token.Verifier, store Logins, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, store: store, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /daily-login", h.status)
	h.mux.HandleFunc("POST /daily-login/claim", h.claim)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type errorResponse struct {
	Error string `json:"error"`
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

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	status, err := h.store.Status(r.Context(), claims.Account, h.now())
	h.answer(w, status, err)
}

func (h *Handler) claim(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	status, err := h.store.Claim(r.Context(), claims.Account, h.now())
	h.answer(w, status, err)
}

func (h *Handler) answer(w http.ResponseWriter, status Status, err error) {
	switch {
	case err == nil:
		reply(w, http.StatusOK, status)
	case errors.Is(err, ErrClaimed):
		reply(w, http.StatusConflict, errorResponse{Error: ErrClaimed.Error()})
	case errors.Is(err, ErrNoCalendar):
		reply(w, http.StatusServiceUnavailable, errorResponse{Error: ErrNoCalendar.Error()})
	default:
		h.logger.Error("daily login request failed", "error", err)
		reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
	}
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
