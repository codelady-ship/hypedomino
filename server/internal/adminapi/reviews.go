package adminapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/playerrecord"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/risk"
)

const (
	QueuePageSize       = 50
	QueuePageMax        = 200
	ReviewSignalLimit   = 50
	ClearReviewAction   = "clear_review"
	ConfirmReviewAction = "confirm_review"
)

type reviewsResponse struct {
	Reviews []reviewEntry `json:"reviews"`
}

type reviewEntry struct {
	risk.Review
	queueDeadline
	Decisions []playerrecord.Decision `json:"decisions"`
}

type reviewState struct {
	AccountID    string `json:"account_id"`
	Shadowbanned bool   `json:"shadowbanned"`
}

func queuePageLimit(raw string) (int, error) {
	limit, err := optionalNumber(raw)
	if err != nil {
		return 0, err
	}
	if limit == 0 {
		return QueuePageSize, nil
	}
	return int(min(limit, QueuePageMax)), nil
}

func (h *Handler) listReviews(w http.ResponseWriter, r *http.Request, p principal) {
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
	reviews, err := risk.OpenReviews(r.Context(), h.pool, risk.ReviewQuery{
		Cause: query.Get("cause"), Account: query.Get("account"), After: after, Limit: limit,
	}, ReviewSignalLimit)
	if err != nil {
		h.refuse(w, err)
		return
	}
	accounts := make([]string, len(reviews))
	for index, review := range reviews {
		accounts[index] = review.Account
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
	views := make([]reviewEntry, len(reviews))
	for index, review := range reviews {
		views[index] = reviewEntry{Review: review, queueDeadline: deadlineOf(review.OpenedAt, sla.Review, now), Decisions: decisions[review.Account]}
		if views[index].Decisions == nil {
			views[index].Decisions = []playerrecord.Decision{}
		}
	}
	reply(w, http.StatusOK, reviewsResponse{Reviews: views})
}

func (h *Handler) decideReview(decision, action string) authedHandler {
	return func(w http.ResponseWriter, r *http.Request, p principal) {
		var note changeNote
		if !decode(w, r, &note) {
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			h.refuse(w, risk.ErrNoReview)
			return
		}
		var state reviewState
		err = h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
			if err := h.sensitive(p, note, now); err != nil {
				return nil, err
			}
			decided, err := risk.DecideReview(r.Context(), tx, id, decision, p.account.ID, now)
			if err != nil {
				return nil, err
			}
			state = reviewState{AccountID: decided.Account, Shadowbanned: decided.StillShadowbanned}
			return []audit.Entry{{
				Action: action, Object: audit.Object{Type: rbac.Queues, ID: r.PathValue("id")},
				Before: marshal(reviewState{AccountID: decided.Account, Shadowbanned: decided.WasShadowbanned}), After: marshal(state),
				Reason: note.Reason, Ref: note.Ref,
			}}, nil
		})
		if err != nil {
			h.refuse(w, err)
			return
		}
		reply(w, http.StatusOK, state)
	}
}
