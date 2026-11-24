package dataexport

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

type Documents interface {
	Export(ctx context.Context, account string, now time.Time) ([]byte, error)
}

type Handler struct {
	verifier  token.Verifier
	store     *Store
	documents Documents
	files     Files
	settings  Settings
	logger    *slog.Logger
	now       func() time.Time
	mux       *http.ServeMux
}

type exportResponse struct {
	Error     string    `json:"error,omitempty"`
	URL       string    `json:"url,omitempty"`
	ExpiresAt time.Time `json:"expires_at,omitzero"`
	RetryAt   time.Time `json:"retry_at,omitzero"`
}

func NewHandler(verifier token.Verifier, store *Store, documents Documents, files Files, settings Settings, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, store: store, documents: documents, files: files, settings: settings, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("POST /account/export", h.export)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) export(w http.ResponseWriter, r *http.Request) {
	claims, err := token.Authorize(h.verifier, r)
	switch {
	case errors.Is(err, token.ErrExpired):
		reply(w, http.StatusUnauthorized, exportResponse{Error: "access_expired"})
		return
	case err != nil:
		reply(w, http.StatusUnauthorized, exportResponse{Error: "access_rejected"})
		return
	}
	now := h.now()
	key := "exports/" + uuid.NewString() + ".json"
	retryAt, err := h.store.Reserve(r.Context(), claims.Account, key, now, h.settings)
	if errors.Is(err, ErrCooldown) {
		reply(w, http.StatusTooManyRequests, exportResponse{Error: "export_cooldown", RetryAt: retryAt})
		return
	}
	if err != nil {
		h.fail(w, "reserve", err)
		return
	}
	if err := h.upload(r.Context(), claims.Account, key, now); err != nil {
		if released := h.store.Release(context.WithoutCancel(r.Context()), key); released != nil {
			h.logger.Error("data export release failed", "key", key, "error", released)
		}
		h.fail(w, "build", err)
		return
	}
	reply(w, http.StatusCreated, exportResponse{URL: h.files.SignedGet(key, now), ExpiresAt: now.Add(h.settings.LinkTTL).UTC()})
}

func (h *Handler) upload(ctx context.Context, account, key string, now time.Time) error {
	document, err := h.documents.Export(ctx, account, now)
	if err != nil {
		return err
	}
	return h.files.Put(ctx, key, "application/json", document, now)
}

func (h *Handler) fail(w http.ResponseWriter, step string, err error) {
	h.logger.Error("data export failed", "step", step, "error", err)
	reply(w, http.StatusServiceUnavailable, exportResponse{Error: "export_unavailable"})
}

func reply(w http.ResponseWriter, status int, body exportResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
