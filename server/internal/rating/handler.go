package rating

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type Mailbox interface {
	Rewards(ctx context.Context, account string) ([]Reward, error)
	MarkRewardsSeen(ctx context.Context, account string, now time.Time) (int64, error)
	Emblems(ctx context.Context, account string) ([]Emblem, error)
}

type Handler struct {
	verifier token.Verifier
	mailbox  Mailbox
	logger   *slog.Logger
	now      func() time.Time
	mux      *http.ServeMux
}

func NewHandler(verifier token.Verifier, mailbox Mailbox, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, mailbox: mailbox, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /ranked/rewards", h.rewards)
	h.mux.HandleFunc("POST /ranked/rewards/seen", h.seen)
	h.mux.HandleFunc("GET /ranked/emblems", h.emblems)
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

func (h *Handler) rewards(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	rewards, err := h.mailbox.Rewards(r.Context(), claims.Account)
	if err != nil {
		h.fail(w, err)
		return
	}
	unseen := 0
	for _, reward := range rewards {
		if !reward.Seen {
			unseen++
		}
	}
	reply(w, http.StatusOK, map[string]any{"rewards": rewards, "unseen": unseen})
}

func (h *Handler) seen(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	marked, err := h.mailbox.MarkRewardsSeen(r.Context(), claims.Account, h.now())
	if err != nil {
		h.fail(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"seen": marked})
}

func (h *Handler) emblems(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	emblems, err := h.mailbox.Emblems(r.Context(), claims.Account)
	if err != nil {
		h.fail(w, err)
		return
	}
	reply(w, http.StatusOK, map[string]any{"emblems": emblems})
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	h.logger.Error("ranked request failed", "error", err)
	reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
