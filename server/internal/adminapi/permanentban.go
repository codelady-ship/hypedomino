package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/approval"
	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/risk"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

const PermanentBanAction = "permanent_ban"

type permanentBanChange struct {
	ReviewID int64             `json:"review_id"`
	Category sanction.Category `json:"category,omitempty"`
}

type accountState struct {
	State string `json:"state"`
}

type permanentlyBannedState struct {
	State           string            `json:"state"`
	SessionsRevoked int               `json:"sessions_revoked"`
	SanctionID      int64             `json:"sanction_id"`
	Category        sanction.Category `json:"category"`
}

func (h *Handler) proposePermanentBan(w http.ResponseWriter, r *http.Request, p principal) {
	var note banNote
	if !decode(w, r, &note) {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		h.refuse(w, risk.ErrNoReview)
		return
	}
	var change json.RawMessage
	var proposal approval.Proposal
	err = h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := h.sensitive(p, note.changeNote, now); err != nil {
			return nil, err
		}
		category, err := sanction.ParseCategory(note.Category)
		if err != nil {
			return nil, err
		}
		change = marshal(permanentBanChange{ReviewID: id, Category: category})
		accountID, err := risk.OpenReviewAccount(r.Context(), tx, id)
		if err != nil {
			return nil, err
		}
		if proposal, err = h.approvals.In(tx).Propose(r.Context(), approval.PermanentBan, accountID, change, note.Reason, p.actor(), now); err != nil {
			return nil, err
		}
		return []audit.Entry{{Action: "submit", Object: proposal.Object, After: change, Reason: note.Reason, Ref: note.Ref}}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusAccepted, viewProposal(proposal))
}

func (h *Handler) banPermanently(ctx context.Context, tx pgx.Tx, p principal, proposal approval.Proposal, note changeNote, now time.Time) error {
	var change permanentBanChange
	if err := json.Unmarshal(proposal.Change, &change); err != nil {
		return err
	}
	decided, err := risk.DecideReview(ctx, tx, change.ReviewID, risk.DecisionConfirm, p.account.ID, now)
	if err != nil {
		return err
	}
	var before string
	if err := tx.QueryRow(ctx, `select state from accounts where id = $1`, decided.Account).Scan(&before); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `update accounts set state = $2 where id = $1`, decided.Account, account.StateBanned); err != nil {
		return err
	}
	category, err := sanction.ParseCategory(string(change.Category))
	if err != nil {
		category = sanction.Other
	}
	imposed, err := sanction.Impose(ctx, tx, decided.Account, sanction.PermanentBan, category, now, nil)
	if err != nil {
		return err
	}
	revoked, err := token.RevokeAccountIn(ctx, tx, decided.Account, token.ReasonBanned, now)
	if err != nil {
		return err
	}
	decision := audit.Entry{Action: "approve", Object: proposal.Object, After: proposal.Change, Reason: note.Reason, Ref: note.Ref}
	recorded, err := h.audit.In(tx).Append(ctx, h.stamp(p, decision), now)
	if err != nil {
		return err
	}
	reviewed := reviewState{AccountID: decided.Account, Shadowbanned: decided.StillShadowbanned}
	entries := []audit.Entry{
		{Action: ConfirmReviewAction, Object: audit.Object{Type: rbac.Queues, ID: strconv.FormatInt(change.ReviewID, 10)},
			Before: marshal(reviewState{AccountID: decided.Account, Shadowbanned: decided.WasShadowbanned}), After: marshal(reviewed)},
		{Action: PermanentBanAction, Object: proposal.Object, Before: marshal(accountState{State: before}),
			After: marshal(permanentlyBannedState{State: account.StateBanned, SessionsRevoked: revoked, SanctionID: imposed.ID, Category: category})},
	}
	for _, entry := range entries {
		entry.Reason, entry.ApprovalID = proposal.Reason, recorded.ID
		if _, err := h.audit.In(tx).Append(ctx, h.stamp(p, entry), now); err != nil {
			return err
		}
	}
	return nil
}
