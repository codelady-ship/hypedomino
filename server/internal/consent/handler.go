package consent

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

const (
	MaxBodyBytes    = 4 * 1024
	FirstLaunchPath = "/consents/first-launch"
)

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
	h.mux.HandleFunc("GET /consents", h.authed(func(r *http.Request, account string) (Choices, error) {
		return h.store.Choices(r.Context(), account)
	}))
	h.mux.HandleFunc("PUT /consents", h.authed(h.save))
	h.mux.HandleFunc("POST "+FirstLaunchPath, h.authed(h.saveFirstLaunch))
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) save(r *http.Request, account string) (Choices, error) {
	var change Change
	if err := json.NewDecoder(r.Body).Decode(&change); err != nil {
		return Choices{}, ErrMalformed
	}
	return h.store.Save(r.Context(), account, change, h.now())
}

func (h *Handler) saveFirstLaunch(r *http.Request, account string) (Choices, error) {
	var first FirstLaunch
	if err := json.NewDecoder(r.Body).Decode(&first); err != nil || !first.Known() {
		return Choices{}, ErrMalformed
	}
	return h.store.SaveFirstLaunch(r.Context(), account, first, h.now())
}

func (h *Handler) authed(serve func(r *http.Request, account string) (Choices, error)) http.HandlerFunc {
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
		choices, err := serve(r, claims.Account)
		if err != nil {
			h.refuse(w, err)
			return
		}
		reply(w, http.StatusOK, choices)
	}
}

func (h *Handler) refuse(w http.ResponseWriter, err error) {
	for _, known := range []error{ErrMalformed, ErrContext} {
		if errors.Is(err, known) {
			reply(w, http.StatusBadRequest, errorResponse{Error: known.Error()})
			return
		}
	}
	h.logger.Error("consent request failed", "error", err)
	reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
