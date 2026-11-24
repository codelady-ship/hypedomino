package adminapi

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/playerrecord"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/report"
)

const (
	UpholdReportsAction = "uphold_reports"
	RejectReportsAction = "reject_reports"
)

type reportsResponse struct {
	Targets []reportTarget `json:"targets"`
}

type reportTarget struct {
	report.Target
	queueDeadline
	Decisions []playerrecord.Decision `json:"decisions"`
}

type reportsDecision struct {
	AccountID string `json:"account_id"`
	State     string `json:"state"`
	Closed    int64  `json:"closed"`
}

func (h *Handler) listReports(w http.ResponseWriter, r *http.Request, p principal) {
	query := r.URL.Query()
	after := query.Get("after")
	limit, err := queuePageLimit(query.Get("limit"))
	if err != nil || (after != "" && uuid.Validate(after) != nil) {
		h.refuse(w, errMalformed)
		return
	}
	targets, err := report.Open(r.Context(), h.pool, report.Query{
		Reason: query.Get("reason"), Account: query.Get("account"), After: after, Limit: limit,
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	accounts := make([]string, len(targets))
	for index, target := range targets {
		accounts[index] = target.AccountID
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
	views := make([]reportTarget, len(targets))
	for index, target := range targets {
		views[index] = reportTarget{Target: target, queueDeadline: deadlineOf(target.Reports[0].UpdatedAt, sla.Report, now), Decisions: decisions[target.AccountID]}
		if views[index].Decisions == nil {
			views[index].Decisions = []playerrecord.Decision{}
		}
	}
	reply(w, http.StatusOK, reportsResponse{Targets: views})
}

func (h *Handler) decideReports(state, action string) authedHandler {
	return func(w http.ResponseWriter, r *http.Request, p principal) {
		var note changeNote
		if !decode(w, r, &note) {
			return
		}
		decision := reportsDecision{AccountID: r.PathValue("id"), State: state}
		err := h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
			if err := h.sensitive(p, note, now); err != nil {
				return nil, err
			}
			if err := playerrecord.Lock(r.Context(), tx, decision.AccountID); err != nil {
				return nil, err
			}
			closed, err := report.Decide(r.Context(), tx, decision.AccountID, state, p.account.ID, now)
			if err != nil {
				return nil, err
			}
			decision.Closed = closed
			return []audit.Entry{{Action: action, Object: audit.Object{Type: rbac.Players, ID: decision.AccountID}, After: marshal(decision), Reason: note.Reason, Ref: note.Ref}}, nil
		})
		if err != nil {
			h.refuse(w, err)
			return
		}
		reply(w, http.StatusOK, decision)
	}
}
