package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
)

type Deletions interface {
	Lookup(ctx context.Context, identity idtoken.Identity) (account.Account, error)
	RequestDeletion(ctx context.Context, id string, now time.Time) (time.Time, error)
	CancelDeletion(ctx context.Context, id string) error
}

type Deletion struct {
	Accounts Deletions
	Grace    time.Duration
	InMatch  func(ctx context.Context, account string) (bool, error)
	Hide     func(ctx context.Context, account string) error
	Now      func() time.Time
}

type identityRequest struct {
	Provider string `json:"provider"`
	IDToken  string `json:"id_token"`
	Nonce    string `json:"nonce"`
}

type deletionResponse struct {
	Error    string    `json:"error,omitempty"`
	DeleteAt time.Time `json:"delete_at"`
}

func (h *Handler) EnableDeletion(deletion Deletion) {
	h.deletion = deletion
	h.mux.HandleFunc("POST /auth/deletion", h.requestDeletion)
	h.mux.HandleFunc("POST /auth/deletion/cancel", h.cancelDeletion)
}

func (h *Handler) deleteAt(a account.Account) time.Time {
	if a.DeletionAt == nil {
		return time.Time{}
	}
	return a.DeletionAt.UTC().Add(h.deletion.Grace)
}

func (h *Handler) signedIn(w http.ResponseWriter, r *http.Request) (account.Account, bool, bool) {
	var request identityRequest
	if !decode(w, r, &request) {
		return account.Account{}, false, false
	}
	provider, ok := parseProvider(request.Provider)
	if !ok {
		reply(w, http.StatusBadRequest, errorResponse{Error: "unknown_provider"})
		return account.Account{}, false, false
	}
	identity, err := h.verifier.Verify(r.Context(), provider, request.IDToken, request.Nonce)
	if err != nil {
		h.refuse(w, err)
		return account.Account{}, false, false
	}
	a, err := h.deletion.Accounts.Lookup(r.Context(), identity)
	if err != nil && !errors.Is(err, account.ErrDeletionPending) {
		h.refuse(w, err)
		return account.Account{}, false, false
	}
	return a, err != nil, true
}

func (h *Handler) requestDeletion(w http.ResponseWriter, r *http.Request) {
	a, pending, ok := h.signedIn(w, r)
	switch {
	case !ok:
		return
	case pending:
		reply(w, http.StatusAccepted, deletionResponse{DeleteAt: h.deleteAt(a)})
		return
	}
	if h.deletion.InMatch != nil {
		playing, err := h.deletion.InMatch(r.Context(), a.ID)
		if err != nil {
			h.refuse(w, err)
			return
		}
		if playing {
			reply(w, http.StatusConflict, errorResponse{Error: "in_match"})
			return
		}
	}
	requested, err := h.deletion.Accounts.RequestDeletion(r.Context(), a.ID, h.deletion.Now())
	if err != nil {
		h.refuse(w, err)
		return
	}
	if _, err := h.tokens.LogoutAll(r.Context(), a.ID); err != nil {
		h.refuse(w, err)
		return
	}
	h.connections.RevokeAccount(a.ID)
	if h.deletion.Hide != nil {
		if err := h.deletion.Hide(r.Context(), a.ID); err != nil {
			h.logger.Error("hide deleting account from boards failed", "account", a.ID, "error", err)
		}
	}
	a.DeletionAt = &requested
	reply(w, http.StatusAccepted, deletionResponse{DeleteAt: h.deleteAt(a)})
}

func (h *Handler) cancelDeletion(w http.ResponseWriter, r *http.Request) {
	a, pending, ok := h.signedIn(w, r)
	switch {
	case !ok:
		return
	case !pending:
		reply(w, http.StatusConflict, errorResponse{Error: "no_deletion_pending"})
		return
	}
	if err := h.deletion.Accounts.CancelDeletion(r.Context(), a.ID); err != nil {
		h.refuse(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
