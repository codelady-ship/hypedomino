package medal

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type Medals interface {
	Collection(ctx context.Context, account string) (Collection, error)
	SetShowcase(ctx context.Context, account string, medals []string, now time.Time) ([]string, error)
}

type UnownedObserver interface {
	Observe(ctx context.Context, account string)
}

type Handler struct {
	verifier token.Verifier
	store    Medals
	unowned  UnownedObserver
	logger   *slog.Logger
	now      func() time.Time
	mux      *http.ServeMux
}

func NewHandler(verifier token.Verifier, store Medals, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, store: store, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /medals", h.collection)
	h.mux.HandleFunc("PUT /medals/showcase", h.showcase)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type errorResponse struct {
	Error string `json:"error"`
}

type showcaseRequest struct {
	Medals []string `json:"medals"`
}

type showcaseResponse struct {
	Showcase []string `json:"showcase"`
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

func (h *Handler) collection(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	collection, err := h.store.Collection(r.Context(), claims.Account)
	if err != nil {
		h.fail(w, err)
		return
	}
	reply(w, http.StatusOK, collection)
}

func (h *Handler) showcase(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	var request showcaseRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&request); err != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: "malformed"})
		return
	}
	showcase, err := h.store.SetShowcase(r.Context(), claims.Account, request.Medals, h.now())
	switch {
	case errors.Is(err, ErrShowcase):
		reply(w, http.StatusBadRequest, errorResponse{Error: ErrShowcase.Error()})
	case errors.Is(err, ErrNotOwned):
		h.observeUnowned(r.Context(), claims.Account)
		reply(w, http.StatusConflict, errorResponse{Error: ErrNotOwned.Error()})
	case err != nil:
		h.fail(w, err)
	default:
		reply(w, http.StatusOK, showcaseResponse{Showcase: showcase})
	}
}

func (h *Handler) FlagUnowned(observer UnownedObserver) {
	h.unowned = observer
}

func (h *Handler) observeUnowned(ctx context.Context, account string) {
	if h.unowned != nil {
		h.unowned.Observe(ctx, account)
	}
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrNoCatalog) {
		reply(w, http.StatusServiceUnavailable, errorResponse{Error: ErrNoCatalog.Error()})
		return
	}
	h.logger.Error("medal request failed", "error", err)
	reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
