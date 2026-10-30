package shop

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const MaxBodyBytes = 1024

type Shop interface {
	View(ctx context.Context, account string, now time.Time) (View, error)
	Buy(ctx context.Context, account, offer, requestID string, now time.Time) (Purchase, error)
	Begin(ctx context.Context, account, offer string, now time.Time) (Checkout, error)
}

type Handler struct {
	verifier token.Verifier
	shop     Shop
	logger   *slog.Logger
	now      func() time.Time
	mux      *http.ServeMux
}

func NewHandler(verifier token.Verifier, shop Shop, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, shop: shop, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /shop", h.view)
	h.mux.HandleFunc("POST /shop/buy", h.buy)
	h.mux.HandleFunc("POST /shop/begin", h.begin)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type intent struct {
	Offer     string `json:"offer"`
	RequestID string `json:"request_id"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) view(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	view, err := h.shop.View(r.Context(), claims.Account, h.now())
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, view)
}

func (h *Handler) buy(w http.ResponseWriter, r *http.Request) {
	claims, request, ok := h.accept(w, r)
	if !ok {
		return
	}
	purchase, err := h.shop.Buy(r.Context(), claims.Account, request.Offer, request.RequestID, h.now())
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, purchase)
}

func (h *Handler) begin(w http.ResponseWriter, r *http.Request) {
	claims, request, ok := h.accept(w, r)
	if !ok {
		return
	}
	checkout, err := h.shop.Begin(r.Context(), claims.Account, request.Offer, h.now())
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, checkout)
}

func (h *Handler) accept(w http.ResponseWriter, r *http.Request) (token.Claims, intent, bool) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return token.Claims{}, intent{}, false
	}
	var request intent
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes)).Decode(&request); err != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: "malformed"})
		return token.Claims{}, intent{}, false
	}
	return claims, request, true
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

var statuses = []struct {
	err    error
	status int
	code   string
}{
	{ErrRequest, http.StatusBadRequest, ErrRequest.Error()},
	{ErrNotEligible, http.StatusForbidden, ErrNotEligible.Error()},
	{ErrUnknownOffer, http.StatusNotFound, ErrUnknownOffer.Error()},
	{wallet.ErrInsufficient, http.StatusPaymentRequired, "wallet.insufficient"},
	{ErrUnavailable, http.StatusConflict, ErrUnavailable.Error()},
	{ErrLimit, http.StatusConflict, ErrLimit.Error()},
	{ErrOwned, http.StatusConflict, ErrOwned.Error()},
	{ErrStoreOnly, http.StatusConflict, ErrStoreOnly.Error()},
	{ErrCoinsOnly, http.StatusConflict, ErrCoinsOnly.Error()},
	{ErrUnsupported, http.StatusConflict, ErrUnsupported.Error()},
	{inventory.ErrSourceNotListed, http.StatusConflict, inventory.ErrSourceNotListed.Error()},
	{ErrNoCatalog, http.StatusServiceUnavailable, ErrNoCatalog.Error()},
}

func (h *Handler) refuse(w http.ResponseWriter, err error) {
	for _, known := range statuses {
		if errors.Is(err, known.err) {
			reply(w, known.status, errorResponse{Error: known.code})
			return
		}
	}
	h.logger.Error("shop request failed", "error", err)
	reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
