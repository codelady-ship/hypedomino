package adminapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/playerrecord"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

const (
	DefaultMuteHours = 24
	MuteAction       = "mute"
	UnmuteAction     = "unmute"
)

var errNotMuted = errors.New("adminapi: the account has no running mute")

type muteState struct {
	MutedUntil *time.Time `json:"muted_until"`
}

type muteView struct {
	AccountID  string     `json:"account_id"`
	MutedUntil *time.Time `json:"muted_until"`
}

func (h *Handler) mute(w http.ResponseWriter, r *http.Request, p principal) {
	h.changeMute(w, r, p, true)
}

func (h *Handler) unmute(w http.ResponseWriter, r *http.Request, p principal) {
	h.changeMute(w, r, p, false)
}

func (h *Handler) changeMute(w http.ResponseWriter, r *http.Request, p principal, mute bool) {
	var note changeNote
	if !decode(w, r, &note) {
		return
	}
	accountID := r.PathValue("id")
	var after muteState
	err := h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := h.sensitive(p, note, now); err != nil {
			return nil, err
		}
		if err := playerrecord.Lock(r.Context(), tx, accountID); err != nil {
			return nil, err
		}
		var before muteState
		if err := tx.QueryRow(r.Context(), `select muted_until from accounts where id = $1`, accountID).Scan(&before.MutedUntil); err != nil {
			return nil, err
		}
		action := UnmuteAction
		if mute {
			until := now.Add(h.config.Mute).UTC().Truncate(time.Microsecond)
			after.MutedUntil, action = &until, MuteAction
		} else if before.MutedUntil == nil || !now.Before(*before.MutedUntil) {
			return nil, errNotMuted
		}
		if _, err := tx.Exec(r.Context(), `update accounts set muted_until = $2 where id = $1`, accountID, after.MutedUntil); err != nil {
			return nil, err
		}
		return []audit.Entry{{Action: action, Object: audit.Object{Type: rbac.Players, ID: accountID}, Before: marshal(before), After: marshal(after), Reason: note.Reason, Ref: note.Ref}}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, muteView{AccountID: accountID, MutedUntil: after.MutedUntil})
}
