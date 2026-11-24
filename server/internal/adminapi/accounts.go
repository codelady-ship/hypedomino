package adminapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/adminaccount"
	"github.com/Alievs-corp/hypedomino/server/internal/approval"
	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

type changeNote struct {
	Reason string `json:"reason"`
	Ref    string `json:"ref"`
}

type createAccountRequest struct {
	Email       string      `json:"email"`
	DisplayName string      `json:"display_name"`
	Roles       []rbac.Role `json:"roles"`
	changeNote
}

type setRolesRequest struct {
	Roles []rbac.Role `json:"roles"`
	changeNote
}

type accountsResponse struct {
	Accounts []accountView `json:"accounts"`
}

type auditedAccount struct {
	Email       string      `json:"email"`
	DisplayName string      `json:"display_name"`
	Roles       []rbac.Role `json:"roles"`
	State       string      `json:"state"`
}

func snapshotOf(a adminaccount.Account) json.RawMessage {
	raw, _ := json.Marshal(auditedAccount{Email: a.Email, DisplayName: a.DisplayName, Roles: a.Roles, State: string(a.State)})
	return raw
}

func accountEntry(action string, note changeNote, before, after adminaccount.Account) audit.Entry {
	entry := audit.Entry{Action: action, Object: audit.Object{Type: rbac.AdminAccounts, ID: after.ID}, Reason: note.Reason, Ref: note.Ref}
	if before.ID != "" {
		entry.Before = snapshotOf(before)
	}
	entry.After = snapshotOf(after)
	return entry
}

type accountChange func(accounts *adminaccount.Store, now time.Time) (adminaccount.Account, audit.Entry, error)

func (h *Handler) changeAccount(ctx context.Context, p principal, change accountChange) (adminaccount.Account, error) {
	var changed adminaccount.Account
	err := h.audited(ctx, p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		account, entry, err := change(h.accounts.In(tx), now)
		changed = account
		return []audit.Entry{entry}, err
	})
	return changed, err
}

func (h *Handler) replyAccount(w http.ResponseWriter, status int, account adminaccount.Account, err error) {
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, status, viewAccount(account))
}

func (h *Handler) listAccounts(w http.ResponseWriter, r *http.Request, p principal) {
	accounts, err := h.accounts.List(r.Context())
	if err != nil {
		h.refuse(w, err)
		return
	}
	views := make([]accountView, 0, len(accounts))
	for _, account := range accounts {
		views = append(views, viewAccount(account))
	}
	reply(w, http.StatusOK, accountsResponse{Accounts: views})
}

func (h *Handler) createAccount(w http.ResponseWriter, r *http.Request, p principal) {
	var request createAccountRequest
	if !decode(w, r, &request) {
		return
	}
	if slices.Contains(request.Roles, rbac.SuperAdmin) {
		h.refuse(w, errGrantSeparately)
		return
	}
	account, err := h.changeAccount(r.Context(), p, func(accounts *adminaccount.Store, now time.Time) (adminaccount.Account, audit.Entry, error) {
		created, err := accounts.Create(r.Context(), request.Email, request.DisplayName, request.Roles, now)
		return created, accountEntry("create", request.changeNote, adminaccount.Account{}, created), err
	})
	h.replyAccount(w, http.StatusCreated, account, err)
}

func grantsSuperAdmin(before, after []rbac.Role) bool {
	return slices.Contains(after, rbac.SuperAdmin) && !slices.Contains(before, rbac.SuperAdmin)
}

func (h *Handler) setRoles(w http.ResponseWriter, r *http.Request, p principal) {
	var request setRolesRequest
	if !decode(w, r, &request) {
		return
	}
	id := r.PathValue("id")
	if id == p.account.ID {
		h.refuse(w, errOwnAccount)
		return
	}
	var account adminaccount.Account
	var proposal approval.Proposal
	err := h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		accounts := h.accounts.In(tx)
		before, err := accounts.Lock(r.Context(), id)
		if err != nil {
			return nil, err
		}
		if grantsSuperAdmin(before.Roles, request.Roles) {
			proposal, err = h.proposeSuperAdmin(r.Context(), tx, p, before.ID, request, now)
			proposed := before
			proposed.Roles = request.Roles
			return []audit.Entry{accountEntry("submit", request.changeNote, before, proposed)}, err
		}
		account, err = accounts.SetRoles(r.Context(), id, request.Roles)
		return []audit.Entry{accountEntry("edit", request.changeNote, before, account)}, err
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	if proposal.ID != 0 {
		reply(w, http.StatusAccepted, viewProposal(proposal))
		return
	}
	reply(w, http.StatusOK, viewAccount(account))
}

func (h *Handler) proposeSuperAdmin(ctx context.Context, tx pgx.Tx, p principal, accountID string, request setRolesRequest, now time.Time) (approval.Proposal, error) {
	if err := h.allows(p, rbac.AdminAccounts, rbac.GrantSuperAdmin); err != nil {
		return approval.Proposal{}, err
	}
	if len(request.Roles) == 0 || slices.ContainsFunc(request.Roles, func(role rbac.Role) bool { return !rbac.IsRole(role) }) {
		return approval.Proposal{}, fmt.Errorf("%w: unknown role in %v", adminaccount.ErrInvalid, request.Roles)
	}
	if err := h.sensitive(p, request.changeNote, now); err != nil {
		return approval.Proposal{}, err
	}
	return h.approvals.In(tx).Propose(ctx, approval.SuperAdminGrant, accountID, marshal(rolesChange{Roles: request.Roles}), request.Reason, p.actor(), now)
}

func (h *Handler) disable(w http.ResponseWriter, r *http.Request, p principal) {
	var request changeNote
	if !decode(w, r, &request) {
		return
	}
	id := r.PathValue("id")
	if id == p.account.ID {
		h.refuse(w, errOwnAccount)
		return
	}
	account, err := h.changeAccount(r.Context(), p, func(accounts *adminaccount.Store, now time.Time) (adminaccount.Account, audit.Entry, error) {
		before, err := accounts.Lock(r.Context(), id)
		if err != nil {
			return adminaccount.Account{}, audit.Entry{}, err
		}
		after, err := accounts.Disable(r.Context(), id, now)
		return after, accountEntry("retire", request, before, after), err
	})
	h.replyAccount(w, http.StatusOK, account, err)
}
