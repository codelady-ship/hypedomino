package purchase

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const (
	MaxBodyBytes   = 64 * 1024
	NotificationID = "token"
)

type Purchases interface {
	Complete(ctx context.Context, account, offer string, platform Platform, token string, now time.Time) (Result, error)
	Notice(ctx context.Context, platform Platform, notice Notice, now time.Time) (bool, error)
}

type Notifier interface {
	Notice(body []byte) (Notice, error)
}

type Handler struct {
	verifier   token.Verifier
	purchases  Purchases
	google     Notifier
	apple      Notifier
	pushSecret string
	logger     *slog.Logger
	now        func() time.Time
	mux        *http.ServeMux
}

func NewHandler(verifier token.Verifier, purchases Purchases, google, apple Notifier, pushSecret string, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, purchases: purchases, google: google, apple: apple, pushSecret: pushSecret, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /shop/complete", h.complete)
	h.mux.HandleFunc("POST /store/google/notifications", h.googleNotice)
	h.mux.HandleFunc("POST /store/apple/notifications", h.appleNotice)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type completeRequest struct {
	Offer    string   `json:"offer"`
	Platform Platform `json:"platform"`
	Token    string   `json:"token"`
}

type noticeResponse struct {
	Handled bool `json:"handled"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) complete(w http.ResponseWriter, r *http.Request) {
	claims, err := token.Authorize(h.verifier, r)
	switch {
	case errors.Is(err, token.ErrExpired):
		reply(w, http.StatusUnauthorized, errorResponse{Error: "access_expired"})
		return
	case err != nil:
		reply(w, http.StatusUnauthorized, errorResponse{Error: "access_rejected"})
		return
	}
	var request completeRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes)).Decode(&request); err != nil || request.Token == "" {
		reply(w, http.StatusBadRequest, errorResponse{Error: "malformed"})
		return
	}
	result, err := h.purchases.Complete(r.Context(), claims.Account, request.Offer, request.Platform, request.Token, h.now())
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, result)
}

func (h *Handler) googleNotice(w http.ResponseWriter, r *http.Request) {
	if h.google == nil {
		reply(w, http.StatusNotFound, errorResponse{Error: ErrPlatform.Error()})
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.URL.Query().Get(NotificationID)), []byte(h.pushSecret)) != 1 || h.pushSecret == "" {
		reply(w, http.StatusUnauthorized, errorResponse{Error: "notification_rejected"})
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
	if err != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: "malformed"})
		return
	}
	notice, err := h.google.Notice(body)
	h.handle(w, PlatformGoogle, notice, err)
}

func (h *Handler) appleNotice(w http.ResponseWriter, r *http.Request) {
	if h.apple == nil {
		reply(w, http.StatusNotFound, errorResponse{Error: ErrPlatform.Error()})
		return
	}
	var request struct {
		SignedPayload string `json:"signedPayload"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes)).Decode(&request); err != nil || request.SignedPayload == "" {
		reply(w, http.StatusBadRequest, errorResponse{Error: "malformed"})
		return
	}
	notice, err := h.apple.Notice([]byte(request.SignedPayload))
	h.handle(w, PlatformApple, notice, err)
}

func (h *Handler) handle(w http.ResponseWriter, platform Platform, notice Notice, err error) {
	switch {
	case errors.Is(err, ErrSignature), errors.Is(err, ErrApp):
		h.logger.Warn("store notification rejected", "platform", platform, "error", err)
		reply(w, http.StatusUnauthorized, errorResponse{Error: "notification_rejected"})
		return
	case errors.Is(err, ErrEnvironment):
		reply(w, http.StatusOK, noticeResponse{Handled: false})
		return
	case err != nil:
		reply(w, http.StatusBadRequest, errorResponse{Error: "malformed"})
		return
	}
	handled, err := h.purchases.Notice(context.Background(), platform, notice, h.now())
	if err != nil {
		h.logger.Error("store notification failed", "platform", platform, "kind", notice.Kind, "error", err)
		reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
		return
	}
	reply(w, http.StatusOK, noticeResponse{Handled: handled})
}

var statuses = []struct {
	err    error
	status int
}{
	{ErrMalformed, http.StatusBadRequest},
	{ErrSignature, http.StatusBadRequest},
	{ErrNotFound, http.StatusBadRequest},
	{ErrAccount, http.StatusForbidden},
	{ErrReused, http.StatusForbidden},
	{ErrBlocked, http.StatusForbidden},
	{shop.ErrUnknownOffer, http.StatusNotFound},
	{ErrPlatform, http.StatusConflict},
	{ErrProduct, http.StatusConflict},
	{ErrCancelled, http.StatusConflict},
	{ErrEnvironment, http.StatusConflict},
	{ErrApp, http.StatusConflict},
	{shop.ErrCoinsOnly, http.StatusConflict},
	{shop.ErrUnsupported, http.StatusConflict},
	{inventory.ErrSourceNotListed, http.StatusConflict},
	{wallet.ErrPurchaseReuse, http.StatusConflict},
	{ErrStore, http.StatusBadGateway},
	{shop.ErrNoCatalog, http.StatusServiceUnavailable},
}

func (h *Handler) refuse(w http.ResponseWriter, err error) {
	for _, known := range statuses {
		if errors.Is(err, known.err) {
			reply(w, known.status, errorResponse{Error: known.err.Error()})
			return
		}
	}
	h.logger.Error("purchase request failed", "error", err)
	reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
