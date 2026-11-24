package push

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

const MaxBodyBytes = 4 * 1024

type Handler struct {
	verifier token.Verifier
	store    *Store
	logger   *slog.Logger
	now      func() time.Time
	mux      *http.ServeMux
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewHandler(verifier token.Verifier, store *Store, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, store: store, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("PUT /push/devices", h.authed(h.register))
	h.mux.HandleFunc("DELETE /push/devices", h.authed(h.unregister))
	h.mux.HandleFunc("GET /push/settings", h.authed(func(r *http.Request, account string) (any, error) {
		return h.store.Settings(r.Context(), account)
	}))
	h.mux.HandleFunc("PUT /push/settings", h.authed(h.saveSettings))
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) register(r *http.Request, account string) (any, error) {
	var device Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		return nil, ErrMalformed
	}
	return nil, h.store.Register(r.Context(), account, device, h.now())
}

func (h *Handler) unregister(r *http.Request, account string) (any, error) {
	var device Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		return nil, ErrMalformed
	}
	return nil, h.store.Unregister(r.Context(), account, device.Token)
}

func (h *Handler) saveSettings(r *http.Request, account string) (any, error) {
	settings, err := h.store.Settings(r.Context(), account)
	if err != nil {
		return nil, err
	}
	change := SettingsChange{Settings: settings}
	if err := json.NewDecoder(r.Body).Decode(&change); err != nil {
		return nil, ErrMalformed
	}
	return change.Settings, h.store.SaveSettings(r.Context(), account, change, h.now())
}

func (h *Handler) authed(serve func(r *http.Request, account string) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, err := token.Authorize(h.verifier, r)
		switch {
		case errors.Is(err, token.ErrExpired):
			reply(w, http.StatusUnauthorized, errorResponse{Error: "access_expired"})
			return
		case err != nil:
			reply(w, http.StatusUnauthorized, errorResponse{Error: "access_rejected"})
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
		result, err := serve(r, claims.Account)
		if err != nil {
			h.refuse(w, err)
			return
		}
		if result == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		reply(w, http.StatusOK, result)
	}
}

func (h *Handler) refuse(w http.ResponseWriter, err error) {
	for _, known := range []error{ErrNoToken, ErrUnknownPlatform, ErrMalformed, ErrConsentContext} {
		if errors.Is(err, known) {
			reply(w, http.StatusBadRequest, errorResponse{Error: known.Error()})
			return
		}
	}
	h.logger.Error("push request failed", "error", err)
	reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
