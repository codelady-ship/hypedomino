package friends

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

const MaxBodyBytes = 1024

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

type requestReply struct {
	Status string `json:"status"`
}

func NewHandler(verifier token.Verifier, store *Store, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, store: store, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /friends", h.authed(h.list))
	h.mux.HandleFunc("GET /friends/code", h.authed(func(r *http.Request, account string) (any, error) {
		code, err := h.store.Code(r.Context(), account)
		if err != nil {
			return nil, err
		}
		return map[string]string{"code": DisplayCode(code)}, nil
	}))
	h.mux.HandleFunc("POST /friends/requests", h.authed(h.request))
	h.mux.HandleFunc("POST /friends/requests/{id}/accept", h.authed(func(r *http.Request, account string) (any, error) {
		return nil, h.store.Accept(r.Context(), account, r.PathValue("id"), h.now())
	}))
	h.mux.HandleFunc("POST /friends/requests/{id}/decline", h.authed(func(r *http.Request, account string) (any, error) {
		return nil, h.store.Withdraw(r.Context(), r.PathValue("id"), account)
	}))
	h.mux.HandleFunc("DELETE /friends/requests/{id}", h.authed(func(r *http.Request, account string) (any, error) {
		return nil, h.store.Withdraw(r.Context(), account, r.PathValue("id"))
	}))
	h.mux.HandleFunc("DELETE /friends/{id}", h.authed(func(r *http.Request, account string) (any, error) {
		return nil, h.store.Remove(r.Context(), account, r.PathValue("id"))
	}))
	h.mux.HandleFunc("PUT /friends/blocks/{id}", h.authed(func(r *http.Request, account string) (any, error) {
		return nil, h.store.Block(r.Context(), account, r.PathValue("id"), h.now())
	}))
	h.mux.HandleFunc("DELETE /friends/blocks/{id}", h.authed(func(r *http.Request, account string) (any, error) {
		return nil, h.store.Unblock(r.Context(), account, r.PathValue("id"))
	}))
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) list(r *http.Request, account string) (any, error) {
	return h.store.List(r.Context(), account)
}

func (h *Handler) request(r *http.Request, account string) (any, error) {
	var target Target
	if err := json.NewDecoder(r.Body).Decode(&target); err != nil {
		return nil, ErrNoTarget
	}
	befriended, err := h.store.Request(r.Context(), account, target, h.now())
	if err != nil {
		return nil, err
	}
	if befriended {
		return requestReply{Status: "friends"}, nil
	}
	return requestReply{Status: "requested"}, nil
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
	for _, known := range []struct {
		err    error
		status int
	}{
		{ErrUnknownPlayer, http.StatusNotFound},
		{ErrSelf, http.StatusBadRequest},
		{ErrNoTarget, http.StatusBadRequest},
		{ErrAlreadyFriends, http.StatusConflict},
		{ErrFriendsFull, http.StatusConflict},
		{ErrRequestsFull, http.StatusConflict},
		{ErrNoRequest, http.StatusNotFound},
		{ErrNotFriends, http.StatusNotFound},
	} {
		if errors.Is(err, known.err) {
			reply(w, known.status, errorResponse{Error: known.err.Error()})
			return
		}
	}
	h.logger.Error("friends request failed", "error", err)
	reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
