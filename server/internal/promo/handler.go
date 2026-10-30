package promo

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

const MaxBodyBytes = 512

var ErrUnavailable = errors.New("promo.unavailable")

type Promos interface {
	Redeem(ctx context.Context, account, code string, now time.Time) (Redemption, error)
}

type Handler struct {
	verifier token.Verifier
	promos   Promos
	logger   *slog.Logger
	now      func() time.Time
	mux      *http.ServeMux
}

func NewHandler(verifier token.Verifier, promos Promos, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, promos: promos, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /promo/redeem", h.redeem)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type redeemRequest struct {
	Code string `json:"code"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) redeem(w http.ResponseWriter, r *http.Request) {
	claims, err := token.Authorize(h.verifier, r)
	switch {
	case errors.Is(err, token.ErrExpired):
		reply(w, http.StatusUnauthorized, errorResponse{Error: "access_expired"})
		return
	case err != nil:
		reply(w, http.StatusUnauthorized, errorResponse{Error: "access_rejected"})
		return
	}
	if h.promos == nil {
		reply(w, http.StatusServiceUnavailable, errorResponse{Error: ErrUnavailable.Error()})
		return
	}
	var request redeemRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes)).Decode(&request); err != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: "malformed"})
		return
	}
	redemption, err := h.promos.Redeem(r.Context(), claims.Account, request.Code, h.now())
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, redemption)
}

var statuses = []struct {
	err    error
	status int
}{
	{ErrCodeFormat, http.StatusBadRequest},
	{ErrNotEligible, http.StatusForbidden},
	{ErrUnknown, http.StatusNotFound},
	{ErrNotActive, http.StatusConflict},
	{ErrDisabled, http.StatusConflict},
	{ErrNotStarted, http.StatusConflict},
	{ErrExpired, http.StatusConflict},
	{ErrExhausted, http.StatusConflict},
	{ErrAlreadyRedeemed, http.StatusConflict},
	{inventory.ErrSourceNotListed, http.StatusConflict},
	{ErrRateLimited, http.StatusTooManyRequests},
	{ErrLocked, http.StatusTooManyRequests},
}

func (h *Handler) refuse(w http.ResponseWriter, err error) {
	for _, known := range statuses {
		if errors.Is(err, known.err) {
			reply(w, known.status, errorResponse{Error: known.err.Error()})
			return
		}
	}
	h.logger.Error("promo request failed", "error", err)
	reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
