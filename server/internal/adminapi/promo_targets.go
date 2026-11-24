package adminapi

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/promo"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

const (
	MaxPromoTargetBytes = 256 * 1024

	SetPromoTargetsAction   = "set_promo_targets"
	AddPromoTargetAction    = "add_promo_target"
	RemovePromoTargetAction = "remove_promo_target"
)

type promoPersonalView struct {
	Accounts []string         `json:"accounts"`
	Redeemed []promo.Redeemed `json:"redeemed"`
}

type promoDetailView struct {
	Code     promo.Code         `json:"code"`
	Personal *promoPersonalView `json:"personal,omitempty"`
}

type promoTargetsRequest struct {
	Accounts []string `json:"accounts"`
	changeNote
}

type promoTargetsView struct {
	ID       string `json:"id"`
	Accounts int    `json:"accounts"`
	Added    int    `json:"added"`
	Removed  int    `json:"removed"`
}

type promoTargetState struct {
	Accounts int `json:"accounts"`
	Added    int `json:"added,omitempty"`
	Removed  int `json:"removed,omitempty"`
}

type promoTargetEntry struct {
	Promo string `json:"promo"`
}

func (h *Handler) promoDetail(w http.ResponseWriter, r *http.Request, p principal) {
	promos, err := h.promoStore()
	if err != nil {
		h.refuse(w, err)
		return
	}
	id := r.PathValue("id")
	code, err := promos.Get(r.Context(), id)
	if err != nil {
		h.refuse(w, err)
		return
	}
	view := promoDetailView{Code: code}
	if rbac.Allows(p.account.Roles, rbac.PromoCodes, rbac.EditTargetedAccounts) {
		personal := promoPersonalView{}
		if personal.Accounts, err = promos.Targets(r.Context(), id); err != nil {
			h.refuse(w, err)
			return
		}
		if personal.Redeemed, err = promos.Redemptions(r.Context(), id); err != nil {
			h.refuse(w, err)
			return
		}
		view.Personal = &personal
	}
	reply(w, http.StatusOK, view)
}

func (h *Handler) setPromoTargets(w http.ResponseWriter, r *http.Request, p principal) {
	promos, err := h.promoStore()
	if err != nil {
		h.refuse(w, err)
		return
	}
	var request promoTargetsRequest
	if !decodeUpTo(w, r, &request, MaxPromoTargetBytes) {
		return
	}
	id := r.PathValue("id")
	view := promoTargetsView{ID: id}
	err = h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := h.sensitive(p, request.changeNote, now); err != nil {
			return nil, err
		}
		change, err := promos.In(tx).SetTargets(r.Context(), id, request.Accounts)
		if err != nil {
			return nil, err
		}
		view.Accounts, view.Added, view.Removed = change.Count, len(change.Added), len(change.Removed)
		return targetEntries(id, change, request.changeNote), nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, view)
}

func targetEntries(id string, change promo.TargetChange, note changeNote) []audit.Entry {
	before := change.Count - len(change.Added) + len(change.Removed)
	entries := []audit.Entry{{
		Action: SetPromoTargetsAction, Object: audit.Object{Type: rbac.PromoCodes, ID: id},
		Before: marshal(promoTargetState{Accounts: before}),
		After:  marshal(promoTargetState{Accounts: change.Count, Added: len(change.Added), Removed: len(change.Removed)}),
		Reason: note.Reason, Ref: note.Ref,
	}}
	for _, account := range change.Added {
		entries = append(entries, playerTargetEntry(AddPromoTargetAction, account, id, note))
	}
	for _, account := range change.Removed {
		entries = append(entries, playerTargetEntry(RemovePromoTargetAction, account, id, note))
	}
	return entries
}

func playerTargetEntry(action, account, id string, note changeNote) audit.Entry {
	return audit.Entry{
		Action: action, Object: audit.Object{Type: rbac.Players, ID: account},
		After: marshal(promoTargetEntry{Promo: id}), Reason: note.Reason, Ref: note.Ref,
	}
}
