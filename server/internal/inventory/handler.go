package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

const MaxBodyBytes = 4 * 1024

type Inventory interface {
	Owned(ctx context.Context, account string) ([]Owned, error)
	Equip(ctx context.Context, account, slot string, ids []string, now time.Time) (profile.Loadout, error)
	Freeze(ctx context.Context, account string) (profile.Loadout, error)
	EmojiSlots() int
}

type UnownedObserver interface {
	Observe(ctx context.Context, account string)
}

type Handler struct {
	verifier  token.Verifier
	inventory Inventory
	unowned   UnownedObserver
	logger    *slog.Logger
	now       func() time.Time
	mux       *http.ServeMux
}

func NewHandler(verifier token.Verifier, inventory Inventory, logger *slog.Logger, now func() time.Time) *Handler {
	h := &Handler{verifier: verifier, inventory: inventory, logger: logger, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /inventory", h.list)
	h.mux.HandleFunc("PUT /inventory/equip", h.equip)
	return h
}

func (h *Handler) FlagUnowned(observer UnownedObserver) {
	h.unowned = observer
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

type itemView struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	SourceType string `json:"source_type"`
	SourceRef  string `json:"source_ref,omitempty"`
	GrantedAt  string `json:"granted_at,omitempty"`
}

type listView struct {
	Items      []itemView      `json:"items"`
	Loadout    profile.Loadout `json:"loadout"`
	EmojiSlots int             `json:"emoji_slots"`
}

type equipRequest struct {
	Slot  string   `json:"slot"`
	Items []string `json:"items"`
}

type loadoutView struct {
	Loadout profile.Loadout `json:"loadout"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	owned, err := h.inventory.Owned(r.Context(), claims.Account)
	if err != nil {
		h.fail(w, err)
		return
	}
	loadout, err := h.inventory.Freeze(r.Context(), claims.Account)
	if err != nil {
		h.fail(w, err)
		return
	}
	items := make([]itemView, 0, len(owned))
	for _, o := range owned {
		view := itemView{ID: o.ID, Type: o.Type, SourceType: o.Source.Type, SourceRef: o.Source.Ref}
		if !o.GrantedAt.IsZero() {
			view.GrantedAt = o.GrantedAt.UTC().Format(time.RFC3339)
		}
		items = append(items, view)
	}
	reply(w, http.StatusOK, listView{Items: items, Loadout: loadout, EmojiSlots: h.inventory.EmojiSlots()})
}

func (h *Handler) equip(w http.ResponseWriter, r *http.Request) {
	claims, ok := h.authenticate(w, r)
	if !ok {
		return
	}
	var request equipRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes)).Decode(&request); err != nil {
		reply(w, http.StatusBadRequest, errorResponse{Error: "malformed"})
		return
	}
	loadout, err := h.inventory.Equip(r.Context(), claims.Account, request.Slot, request.Items, h.now())
	if err != nil {
		h.observeUnowned(r.Context(), claims.Account, err)
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, loadoutView{Loadout: loadout})
}

func (h *Handler) observeUnowned(ctx context.Context, account string, err error) {
	if h.unowned != nil && errors.Is(err, ErrNotOwned) {
		h.unowned.Observe(ctx, account)
	}
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

func (h *Handler) refuse(w http.ResponseWriter, err error) {
	for _, known := range []struct {
		err    error
		status int
	}{
		{ErrUnknownSlot, http.StatusBadRequest},
		{ErrTooMany, http.StatusBadRequest},
		{ErrUnknownItem, http.StatusNotFound},
		{ErrNotOwned, http.StatusForbidden},
		{ErrWrongType, http.StatusConflict},
		{ErrDisabled, http.StatusConflict},
		{ErrNoProfile, http.StatusConflict},
	} {
		if errors.Is(err, known.err) {
			reply(w, known.status, errorResponse{Error: known.err.Error()})
			return
		}
	}
	h.fail(w, err)
}

func (h *Handler) fail(w http.ResponseWriter, err error) {
	h.logger.Error("inventory request failed", "error", err)
	reply(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
}

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}
