package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/penalties"
	"github.com/Alievs-corp/hypedomino/server/internal/playerrecord"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

const (
	LiftPenaltyAction   = "lift_penalty"
	RemoveOffenseAction = "remove_offense"
)

var (
	errNothingToLift  = errors.New("adminapi: the account has no running penalty")
	errUnknownOffense = errors.New("adminapi: the account has no such offense")
)

type removeOffenseRequest struct {
	Queue   string `json:"queue"`
	MatchID string `json:"match_id"`
	Reason  string `json:"reason"`
	Ref     string `json:"ref"`
}

func (h *Handler) liftPenalty(w http.ResponseWriter, r *http.Request, p principal) {
	var note changeNote
	if !decode(w, r, &note) {
		return
	}
	h.changePenalties(w, r, p, note, LiftPenaltyAction, func(record *penalties.Record, now time.Time) error {
		_, banned := record.Banned(now)
		if !banned && record.QueueDelay(matchmaking.QueueRanked) == 0 && record.QueueDelay(matchmaking.QueueCasual) == 0 {
			return errNothingToLift
		}
		record.Lift()
		return nil
	})
}

func (h *Handler) removeOffense(w http.ResponseWriter, r *http.Request, p principal) {
	var request removeOffenseRequest
	if !decode(w, r, &request) {
		return
	}
	queue := matchmaking.QueueRanked
	switch request.Queue {
	case "ranked":
	case "casual":
		queue = matchmaking.QueueCasual
	default:
		reply(w, http.StatusBadRequest, map[string]string{"error": "invalid"})
		return
	}
	note := changeNote{Reason: request.Reason, Ref: request.Ref}
	h.changePenalties(w, r, p, note, RemoveOffenseAction, func(record *penalties.Record, _ time.Time) error {
		if !record.Remove(queue, request.MatchID) {
			return errUnknownOffense
		}
		return nil
	})
}

func (h *Handler) changePenalties(w http.ResponseWriter, r *http.Request, p principal, note changeNote, action string, change func(*penalties.Record, time.Time) error) {
	accountID := r.PathValue("id")
	var after penalties.Record
	err := h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := h.sensitive(p, note, now); err != nil {
			return nil, err
		}
		if err := playerrecord.Lock(r.Context(), tx, accountID); err != nil {
			return nil, err
		}
		before, err := lockedPenalties(r.Context(), tx, accountID)
		if err != nil {
			return nil, err
		}
		after = before
		after.Ranked.Offenses = append([]penalties.Offense(nil), before.Ranked.Offenses...)
		after.Casual.Offenses = append([]penalties.Offense(nil), before.Casual.Offenses...)
		if err := change(&after, now); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(r.Context(), `update penalty_records set record = $2, updated_at = $3 where account_id = $1`, accountID, marshal(after), now); err != nil {
			return nil, err
		}
		if err := liftEndedPenaltyBan(r.Context(), tx, accountID, before, after, now); err != nil {
			return nil, err
		}
		return []audit.Entry{{Action: action, Object: audit.Object{Type: rbac.Players, ID: accountID}, Before: marshal(before), After: marshal(after), Reason: note.Reason, Ref: note.Ref}}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, after)
}

func liftEndedPenaltyBan(ctx context.Context, tx pgx.Tx, accountID string, before, after penalties.Record, now time.Time) error {
	_, wasBanned := before.Banned(now)
	_, stillBanned := after.Banned(now)
	if !wasBanned || stillBanned {
		return nil
	}
	_, err := sanction.Lift(ctx, tx, accountID, sanction.PenaltyBan, now)
	return err
}

func lockedPenalties(ctx context.Context, tx pgx.Tx, accountID string) (penalties.Record, error) {
	var raw []byte
	err := tx.QueryRow(ctx, `select record from penalty_records where account_id = $1 for update`, accountID).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return penalties.Record{}, nil
	}
	if err != nil {
		return penalties.Record{}, err
	}
	var record penalties.Record
	return record, json.Unmarshal(raw, &record)
}

func (h *Handler) playerPenalties(w http.ResponseWriter, r *http.Request, p principal) {
	accountID := r.PathValue("id")
	var view penalties.View
	err := h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		record, err := lockedPenalties(r.Context(), tx, accountID)
		if err != nil {
			return nil, err
		}
		config, err := publishedPenalties(r.Context(), tx)
		if err != nil {
			return nil, err
		}
		view = record.View(config, now)
		return []audit.Entry{{Action: "view", Object: audit.Object{Type: rbac.Players, ID: accountID}}}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, view)
}
