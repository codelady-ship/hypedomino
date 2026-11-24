package adminapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/playerrecord"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

const (
	ShadowbanAction     = "shadowban"
	LiftShadowbanAction = "lift_shadowban"
)

var (
	errShadowbanned    = errors.New("adminapi: the account is already shadowbanned")
	errNotShadowbanned = errors.New("adminapi: the account is not shadowbanned")
)

type shadowbanState struct {
	ShadowbannedAt *time.Time `json:"shadowbanned_at"`
}

type shadowbanView struct {
	AccountID      string     `json:"account_id"`
	ShadowbannedAt *time.Time `json:"shadowbanned_at"`
}

func (h *Handler) OnShadowban(hide func(ctx context.Context, account string) error) {
	h.hide = hide
}

func (h *Handler) shadowban(w http.ResponseWriter, r *http.Request, p principal) {
	h.changeShadowban(w, r, p, true)
}

func (h *Handler) liftShadowban(w http.ResponseWriter, r *http.Request, p principal) {
	h.changeShadowban(w, r, p, false)
}

func (h *Handler) changeShadowban(w http.ResponseWriter, r *http.Request, p principal, ban bool) {
	var note changeNote
	if !decode(w, r, &note) {
		return
	}
	accountID := r.PathValue("id")
	var after shadowbanState
	err := h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := h.sensitive(p, note, now); err != nil {
			return nil, err
		}
		if err := playerrecord.Lock(r.Context(), tx, accountID); err != nil {
			return nil, err
		}
		var before shadowbanState
		if err := tx.QueryRow(r.Context(), `select shadowbanned_at from accounts where id = $1`, accountID).Scan(&before.ShadowbannedAt); err != nil {
			return nil, err
		}
		switch {
		case ban && before.ShadowbannedAt != nil:
			return nil, errShadowbanned
		case !ban && before.ShadowbannedAt == nil:
			return nil, errNotShadowbanned
		}
		action := LiftShadowbanAction
		if ban {
			at := now.UTC().Truncate(time.Microsecond)
			after.ShadowbannedAt, action = &at, ShadowbanAction
		}
		if _, err := tx.Exec(r.Context(), `update accounts set shadowbanned_at = $2 where id = $1`, accountID, after.ShadowbannedAt); err != nil {
			return nil, err
		}
		return []audit.Entry{{Action: action, Object: audit.Object{Type: rbac.Players, ID: accountID}, Before: marshal(before), After: marshal(after), Reason: note.Reason, Ref: note.Ref}}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	if ban && h.hide != nil {
		if err := h.hide(r.Context(), accountID); err != nil {
			h.logger.Error("shadowbanned account still on a board until it rebuilds", "account", accountID, "error", err)
		}
	}
	reply(w, http.StatusOK, shadowbanView{AccountID: accountID, ShadowbannedAt: after.ShadowbannedAt})
}
