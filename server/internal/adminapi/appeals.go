package adminapi

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/appeal"
	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/playerrecord"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

const (
	UpholdAppealAction = "uphold_appeal"
	LiftAppealAction   = "lift_appeal"
)

type appealsResponse struct {
	Appeals []appealEntry `json:"appeals"`
}

type appealEntry struct {
	appeal.Entry
	queueDeadline
	Decisions []playerrecord.Decision `json:"decisions"`
}

type appealState struct {
	AccountID  string        `json:"account_id"`
	SanctionID int64         `json:"sanction_id"`
	Kind       sanction.Kind `json:"kind"`
	State      string        `json:"state"`
	Reversed   bool          `json:"reversed"`
}

func (h *Handler) listAppeals(w http.ResponseWriter, r *http.Request, p principal) {
	query := r.URL.Query()
	after, err := optionalNumber(query.Get("after"))
	if err != nil {
		h.refuse(w, err)
		return
	}
	limit, err := queuePageLimit(query.Get("limit"))
	if err != nil {
		h.refuse(w, err)
		return
	}
	entries, err := appeal.Open(r.Context(), h.pool, appeal.Query{Kind: query.Get("kind"), Account: query.Get("account"), After: after, Limit: limit})
	if err != nil {
		h.refuse(w, err)
		return
	}
	accounts := make([]string, len(entries))
	for index, entry := range entries {
		accounts[index] = entry.AccountID
	}
	decisions, err := playerrecord.Decisions(r.Context(), h.pool, accounts, playerrecord.ListLimit)
	if err != nil {
		h.refuse(w, err)
		return
	}
	sla, err := h.queueSla(r.Context())
	if err != nil {
		h.refuse(w, err)
		return
	}
	now := h.now()
	views := make([]appealEntry, len(entries))
	for index, entry := range entries {
		views[index] = appealEntry{Entry: entry, queueDeadline: deadlineOf(entry.CreatedAt, sla.Appeal, now), Decisions: decisions[entry.AccountID]}
		if views[index].Decisions == nil {
			views[index].Decisions = []playerrecord.Decision{}
		}
	}
	reply(w, http.StatusOK, appealsResponse{Appeals: views})
}

func (h *Handler) decideAppeal(decision, action string) authedHandler {
	return func(w http.ResponseWriter, r *http.Request, p principal) {
		var note changeNote
		if !decode(w, r, &note) {
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			h.refuse(w, appeal.ErrNoAppeal)
			return
		}
		var state appealState
		err = h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
			if err := h.sensitive(p, note, now); err != nil {
				return nil, err
			}
			accountID, err := appeal.AccountOf(r.Context(), tx, id)
			if err != nil {
				return nil, err
			}
			if err := playerrecord.Lock(r.Context(), tx, accountID); err != nil {
				return nil, err
			}
			decided, appealed, err := appeal.Decide(r.Context(), tx, id, decision, p.account.ID, now)
			if err != nil {
				return nil, err
			}
			state = appealState{AccountID: accountID, SanctionID: appealed.ID, Kind: appealed.Kind, State: decided.State}
			if decision == appeal.StateLifted {
				if state.Reversed, err = reverseSanction(r.Context(), tx, accountID, appealed, now); err != nil {
					return nil, err
				}
			}
			before := appealState{AccountID: accountID, SanctionID: appealed.ID, Kind: appealed.Kind, State: appeal.StateOpen}
			return []audit.Entry{{
				Action: action, Object: audit.Object{Type: rbac.Queues, ID: r.PathValue("id")},
				Before: marshal(before), After: marshal(state), Reason: note.Reason, Ref: note.Ref,
			}}, nil
		})
		if err != nil {
			h.refuse(w, err)
			return
		}
		reply(w, http.StatusOK, state)
	}
}

func reverseSanction(ctx context.Context, tx pgx.Tx, accountID string, appealed sanction.Sanction, now time.Time) (bool, error) {
	lifted, err := sanction.LiftByID(ctx, tx, appealed.ID, now)
	if err != nil || !lifted {
		return false, err
	}
	switch appealed.Kind {
	case sanction.TemporaryBan:
		_, err = tx.Exec(ctx, `update accounts set banned_until = null where id = $1`, accountID)
	case sanction.PermanentBan:
		_, err = tx.Exec(ctx, `update accounts set state = $2 where id = $1 and state = $3`, accountID, account.StateActive, account.StateBanned)
	case sanction.PenaltyBan:
		err = liftPenaltyRecord(ctx, tx, accountID, now)
	case sanction.ForcedRename:
		_, err = tx.Exec(ctx, `update profiles set rename_required = false, updated_at = $2 where account_id = $1`, accountID, now)
	}
	return err == nil, err
}

func liftPenaltyRecord(ctx context.Context, tx pgx.Tx, accountID string, now time.Time) error {
	record, err := lockedPenalties(ctx, tx, accountID)
	if err != nil {
		return err
	}
	record.Lift()
	_, err = tx.Exec(ctx, `update penalty_records set record = $2, updated_at = $3 where account_id = $1`, accountID, marshal(record), now)
	return err
}
