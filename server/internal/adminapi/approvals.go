package adminapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/approval"
	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

type rolesChange struct {
	Roles []rbac.Role `json:"roles"`
}

type proposalView struct {
	ID            int64           `json:"id"`
	Kind          approval.Kind   `json:"kind"`
	ObjectType    rbac.Object     `json:"object_type"`
	ObjectID      string          `json:"object_id"`
	Change        json.RawMessage `json:"change"`
	Reason        string          `json:"reason"`
	ProposerID    string          `json:"proposer_id"`
	ProposerRoles []rbac.Role     `json:"proposer_roles"`
	ProposedAt    time.Time       `json:"proposed_at"`
	ExpiresAt     time.Time       `json:"expires_at"`
	State         approval.State  `json:"state"`
	DeciderID     string          `json:"decider_id,omitempty"`
	DecidedAt     *time.Time      `json:"decided_at,omitempty"`
}

type proposalsResponse struct {
	Proposals []proposalView `json:"proposals"`
}

func viewProposal(p approval.Proposal) proposalView {
	return proposalView{
		ID: p.ID, Kind: p.Kind, ObjectType: p.Object.Type, ObjectID: p.Object.ID, Change: p.Change, Reason: p.Reason,
		ProposerID: p.Proposer.ID, ProposerRoles: p.Proposer.Roles, ProposedAt: p.ProposedAt, ExpiresAt: p.ExpiresAt,
		State: p.State, DeciderID: p.Decider.ID, DecidedAt: optionalTime(p.DecidedAt),
	}
}

func proposalID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0, approval.ErrNotFound
	}
	return id, nil
}

func (h *Handler) pendingApprovals(w http.ResponseWriter, r *http.Request, p principal) {
	pending, err := h.approvals.Pending(r.Context(), h.now())
	if err != nil {
		h.refuse(w, err)
		return
	}
	views := []proposalView{}
	for _, proposal := range pending {
		if proposal.Proposer.ID == p.account.ID || approval.Decides(p.account.Roles, proposal.Kind) {
			views = append(views, viewProposal(proposal))
		}
	}
	reply(w, http.StatusOK, proposalsResponse{Proposals: views})
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request, p principal) {
	var note changeNote
	if !decode(w, r, &note) {
		return
	}
	id, err := proposalID(r)
	if err != nil {
		h.refuse(w, err)
		return
	}
	now := h.now()
	if err := h.sensitive(p, note, now); err != nil {
		h.refuse(w, err)
		return
	}
	var approved approval.Proposal
	err = pgx.BeginFunc(r.Context(), h.pool, func(tx pgx.Tx) error {
		var err error
		approved, err = h.approvals.In(tx).Approve(r.Context(), id, p.actor(), now)
		if err != nil {
			return err
		}
		switch approved.Kind {
		case approval.SuperAdminGrant:
			return h.grantSuperAdmin(r.Context(), tx, p, approved, note, now)
		case approval.PlayerGrant:
			return h.approvePlayerGrant(r.Context(), tx, p, approved, note, now)
		case approval.PromoCode:
			return h.activatePromo(r.Context(), tx, p, approved, note, now)
		case approval.LiveControl:
			return h.applyLiveControls(r.Context(), tx, p, approved, note, now)
		case approval.BreakGlass:
			return h.confirmBreakGlass(r.Context(), tx, p, approved, note, now)
		case approval.PermanentBan:
			return h.banPermanently(r.Context(), tx, p, approved, note, now)
		}
		return errUnsupportedKind
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, viewProposal(approved))
}

func (h *Handler) grantSuperAdmin(ctx context.Context, tx pgx.Tx, p principal, proposal approval.Proposal, note changeNote, now time.Time) error {
	var change rolesChange
	if err := json.Unmarshal(proposal.Change, &change); err != nil {
		return err
	}
	accounts := h.accounts.In(tx)
	before, err := accounts.Lock(ctx, proposal.Object.ID)
	if err != nil {
		return err
	}
	after, err := accounts.SetRoles(ctx, before.ID, change.Roles)
	if err != nil {
		return err
	}
	decision := accountEntry("approve", note, before, after)
	recorded, err := h.audit.In(tx).Append(ctx, h.stamp(p, decision), now)
	if err != nil {
		return err
	}
	applied := h.stamp(p, accountEntry("edit", changeNote{Reason: proposal.Reason}, before, after))
	applied.ApprovalID = recorded.ID
	_, err = h.audit.In(tx).Append(ctx, applied, now)
	return err
}

func (h *Handler) reject(w http.ResponseWriter, r *http.Request, p principal) {
	var note changeNote
	if !decode(w, r, &note) {
		return
	}
	id, err := proposalID(r)
	if err != nil {
		h.refuse(w, err)
		return
	}
	var rejected approval.Proposal
	err = h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := h.sensitive(p, note, now); err != nil {
			return nil, err
		}
		var err error
		rejected, err = h.approvals.In(tx).Reject(r.Context(), id, p.actor(), now)
		if err != nil {
			return nil, err
		}
		entries := []audit.Entry{{Action: "reject", Object: rejected.Object, After: rejected.Change, Reason: note.Reason, Ref: note.Ref}}
		if rejected.Kind != approval.PromoCode {
			return entries, nil
		}
		discarded, err := h.discardPromo(r.Context(), tx, rejected, now)
		return append(entries, discarded), err
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, viewProposal(rejected))
}
