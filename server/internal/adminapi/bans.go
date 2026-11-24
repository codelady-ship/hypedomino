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
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

const (
	TemporaryBanAction     = "temporary_ban"
	LiftTemporaryBanAction = "lift_temporary_ban"
	RankedBanAction        = "ranked_ban"
	LiftRankedBanAction    = "lift_ranked_ban"
	DefaultRankedBanHours  = 72
)

var (
	errBanNotConfigured = errors.New("adminapi: HD_TEMP_BAN_HOURS is not set")
	errNotBanned        = errors.New("adminapi: the account has no running temporary ban")
	errNotRankedBanned  = errors.New("adminapi: the account has no running ranked ban")
)

type banNote struct {
	changeNote
	Category string `json:"category"`
}

type banState struct {
	BannedUntil *time.Time `json:"banned_until"`
}

type bannedState struct {
	BannedUntil     *time.Time        `json:"banned_until"`
	SessionsRevoked int               `json:"sessions_revoked"`
	SanctionID      int64             `json:"sanction_id"`
	Category        sanction.Category `json:"category"`
}

type banView struct {
	AccountID       string            `json:"account_id"`
	BannedUntil     *time.Time        `json:"banned_until"`
	SessionsRevoked int               `json:"sessions_revoked"`
	SanctionID      int64             `json:"sanction_id,omitempty"`
	Category        sanction.Category `json:"category,omitempty"`
}

type rankedBanState struct {
	SanctionID        int64             `json:"sanction_id,omitempty"`
	Category          sanction.Category `json:"category,omitempty"`
	RankedBannedUntil *time.Time        `json:"ranked_banned_until"`
}

type rankedBanView struct {
	AccountID string `json:"account_id"`
	rankedBanState
}

type banChange func(ctx context.Context, tx pgx.Tx, accountID string, note banNote, now time.Time) (any, audit.Entry, error)

func (h *Handler) changeBan(w http.ResponseWriter, r *http.Request, p principal, change banChange) {
	var note banNote
	if !decode(w, r, &note) {
		return
	}
	accountID := r.PathValue("id")
	var view any
	err := h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := h.sensitive(p, note.changeNote, now); err != nil {
			return nil, err
		}
		if err := playerrecord.Lock(r.Context(), tx, accountID); err != nil {
			return nil, err
		}
		var entry audit.Entry
		var err error
		if view, entry, err = change(r.Context(), tx, accountID, note, now); err != nil {
			return nil, err
		}
		entry.Object, entry.Reason, entry.Ref = audit.Object{Type: rbac.Players, ID: accountID}, note.Reason, note.Ref
		return []audit.Entry{entry}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, view)
}

func currentTemporaryBan(ctx context.Context, tx pgx.Tx, accountID string) (*time.Time, error) {
	var current *time.Time
	if err := tx.QueryRow(ctx, `select banned_until from accounts where id = $1`, accountID).Scan(&current); err != nil || current == nil {
		return nil, err
	}
	utc := current.UTC()
	return &utc, nil
}

func (h *Handler) banTemporarily(w http.ResponseWriter, r *http.Request, p principal) {
	if h.config.TempBan <= 0 {
		h.refuse(w, errBanNotConfigured)
		return
	}
	h.changeBan(w, r, p, func(ctx context.Context, tx pgx.Tx, accountID string, note banNote, now time.Time) (any, audit.Entry, error) {
		category, err := sanction.ParseCategory(note.Category)
		if err != nil {
			return nil, audit.Entry{}, err
		}
		current, err := currentTemporaryBan(ctx, tx, accountID)
		if err != nil {
			return nil, audit.Entry{}, err
		}
		until := now.Add(h.config.TempBan).UTC().Truncate(time.Microsecond)
		if _, err := tx.Exec(ctx, `update accounts set banned_until = $2 where id = $1`, accountID, until); err != nil {
			return nil, audit.Entry{}, err
		}
		imposed, err := sanction.Impose(ctx, tx, accountID, sanction.TemporaryBan, category, now, &until)
		if err != nil {
			return nil, audit.Entry{}, err
		}
		revoked, err := token.RevokeAccountIn(ctx, tx, accountID, token.ReasonBanned, now)
		if err != nil {
			return nil, audit.Entry{}, err
		}
		after := bannedState{BannedUntil: &until, SessionsRevoked: revoked, SanctionID: imposed.ID, Category: category}
		entry := audit.Entry{Action: TemporaryBanAction, Before: marshal(banState{BannedUntil: current}), After: marshal(after)}
		return banView{AccountID: accountID, BannedUntil: &until, SessionsRevoked: revoked, SanctionID: imposed.ID, Category: category}, entry, nil
	})
}

func (h *Handler) liftTemporaryBan(w http.ResponseWriter, r *http.Request, p principal) {
	h.changeBan(w, r, p, func(ctx context.Context, tx pgx.Tx, accountID string, _ banNote, now time.Time) (any, audit.Entry, error) {
		current, err := currentTemporaryBan(ctx, tx, accountID)
		if err != nil {
			return nil, audit.Entry{}, err
		}
		if current == nil || !now.Before(*current) {
			return nil, audit.Entry{}, errNotBanned
		}
		if _, err := tx.Exec(ctx, `update accounts set banned_until = null where id = $1`, accountID); err != nil {
			return nil, audit.Entry{}, err
		}
		if _, err := sanction.Lift(ctx, tx, accountID, sanction.TemporaryBan, now); err != nil {
			return nil, audit.Entry{}, err
		}
		entry := audit.Entry{Action: LiftTemporaryBanAction, Before: marshal(banState{BannedUntil: current}), After: marshal(banState{})}
		return banView{AccountID: accountID}, entry, nil
	})
}

func currentRankedBan(ctx context.Context, tx pgx.Tx, accountID string, now time.Time) (rankedBanState, error) {
	running, banned, err := sanction.Current(ctx, tx, accountID, now, sanction.RankedBan)
	if err != nil || !banned {
		return rankedBanState{}, err
	}
	return rankedBanState{SanctionID: running.ID, Category: running.Category, RankedBannedUntil: running.EndsAt}, nil
}

func (h *Handler) banFromRanked(w http.ResponseWriter, r *http.Request, p principal) {
	h.changeBan(w, r, p, func(ctx context.Context, tx pgx.Tx, accountID string, note banNote, now time.Time) (any, audit.Entry, error) {
		category, err := sanction.ParseCategory(note.Category)
		if err != nil {
			return nil, audit.Entry{}, err
		}
		before, err := currentRankedBan(ctx, tx, accountID, now)
		if err != nil {
			return nil, audit.Entry{}, err
		}
		until := now.Add(h.config.RankedBan).UTC().Truncate(time.Microsecond)
		imposed, err := sanction.Impose(ctx, tx, accountID, sanction.RankedBan, category, now, &until)
		if err != nil {
			return nil, audit.Entry{}, err
		}
		after := rankedBanState{SanctionID: imposed.ID, Category: category, RankedBannedUntil: &until}
		entry := audit.Entry{Action: RankedBanAction, Before: marshal(before), After: marshal(after)}
		return rankedBanView{AccountID: accountID, rankedBanState: after}, entry, nil
	})
}

func (h *Handler) liftRankedBan(w http.ResponseWriter, r *http.Request, p principal) {
	h.changeBan(w, r, p, func(ctx context.Context, tx pgx.Tx, accountID string, _ banNote, now time.Time) (any, audit.Entry, error) {
		before, err := currentRankedBan(ctx, tx, accountID, now)
		if err != nil {
			return nil, audit.Entry{}, err
		}
		if before.SanctionID == 0 {
			return nil, audit.Entry{}, errNotRankedBanned
		}
		if _, err := sanction.Lift(ctx, tx, accountID, sanction.RankedBan, now); err != nil {
			return nil, audit.Entry{}, err
		}
		entry := audit.Entry{Action: LiftRankedBanAction, Before: marshal(before), After: marshal(rankedBanState{})}
		return rankedBanView{AccountID: accountID}, entry, nil
	})
}
