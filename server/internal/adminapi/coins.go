package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/approval"
	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/playerrecord"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const (
	DefaultGrantApprovalCoins = 2000
	AdjustCoinsAction         = "adjust_coins"
	adminSource               = "admin"
)

var (
	requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	errBadAdjustment = errors.New("adminapi: a coin adjustment needs a non-zero amount and a request id")
)

type coinsChange struct {
	Coins     int64  `json:"coins"`
	RequestID string `json:"request_id"`
	Proposer  string `json:"proposer_id"`
}

type adjustCoinsRequest struct {
	Amount    int64  `json:"amount"`
	RequestID string `json:"request_id"`
	changeNote
}

type balanceState struct {
	Balance int64 `json:"balance"`
}

type adjustedState struct {
	Balance     int64 `json:"balance"`
	Amount      int64 `json:"amount"`
	LedgerEntry int64 `json:"ledger_entry"`
}

type adjustmentView struct {
	AccountID   string `json:"account_id"`
	Amount      int64  `json:"amount"`
	Balance     int64  `json:"balance"`
	LedgerEntry int64  `json:"ledger_entry"`
	Repeated    bool   `json:"repeated"`
}

func (h *Handler) needsSecondAdmin(coins int64) bool {
	if coins < 0 {
		coins = -coins
	}
	return coins > h.config.GrantApprovalCoins
}

func (h *Handler) adjustCoins(w http.ResponseWriter, r *http.Request, p principal) {
	var request adjustCoinsRequest
	if !decode(w, r, &request) {
		return
	}
	if request.Amount == 0 || !requestIDPattern.MatchString(request.RequestID) {
		h.refuse(w, errBadAdjustment)
		return
	}
	accountID := r.PathValue("id")
	change := coinsChange{Coins: request.Amount, RequestID: request.RequestID, Proposer: p.account.ID}
	var proposal approval.Proposal
	var applied wallet.Applied
	err := h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := h.sensitive(p, request.changeNote, now); err != nil {
			return nil, err
		}
		if err := playerrecord.Lock(r.Context(), tx, accountID); err != nil {
			return nil, err
		}
		if h.needsSecondAdmin(request.Amount) {
			var err error
			proposal, err = h.approvals.In(tx).Propose(r.Context(), approval.PlayerGrant, accountID, marshal(change), request.Reason, p.actor(), now)
			entry := audit.Entry{Action: "submit", Object: audit.Object{Type: rbac.Players, ID: accountID}, After: marshal(change), Reason: request.Reason, Ref: request.Ref}
			return []audit.Entry{entry}, err
		}
		var entry audit.Entry
		var err error
		applied, entry, err = h.applyCoins(r.Context(), tx, accountID, change, now)
		if err != nil || applied.Repeated {
			return nil, err
		}
		entry.Reason, entry.Ref = request.Reason, request.Ref
		return []audit.Entry{entry}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	if proposal.ID != 0 {
		reply(w, http.StatusAccepted, viewProposal(proposal))
		return
	}
	reply(w, http.StatusOK, adjustmentView{AccountID: accountID, Amount: applied.Amount, Balance: applied.BalanceAfter, LedgerEntry: applied.ID, Repeated: applied.Repeated})
}

func (h *Handler) applyCoins(ctx context.Context, tx pgx.Tx, accountID string, change coinsChange, now time.Time) (wallet.Applied, audit.Entry, error) {
	applied, err := h.wallet.ApplyIn(ctx, tx, wallet.Entry{
		Account: accountID, Amount: change.Coins, Reason: wallet.AdminAdjust,
		Source: wallet.Source{Type: adminSource, Ref: change.Proposer}, Key: fmt.Sprintf("admin:%s:%s", change.Proposer, change.RequestID),
	}, now)
	if err != nil {
		return wallet.Applied{}, audit.Entry{}, err
	}
	if applied.Amount != change.Coins {
		return wallet.Applied{}, audit.Entry{}, fmt.Errorf("%w: request id %s was used for %d coins", errBadAdjustment, change.RequestID, applied.Amount)
	}
	entry := audit.Entry{
		Action: AdjustCoinsAction, Object: audit.Object{Type: rbac.Players, ID: accountID},
		Before: marshal(balanceState{Balance: applied.BalanceAfter - applied.Amount}),
		After:  marshal(adjustedState{Balance: applied.BalanceAfter, Amount: applied.Amount, LedgerEntry: applied.ID}),
	}
	return applied, entry, nil
}

func (h *Handler) grantPlayerCoins(ctx context.Context, tx pgx.Tx, p principal, proposal approval.Proposal, note changeNote, now time.Time) error {
	var change coinsChange
	if err := json.Unmarshal(proposal.Change, &change); err != nil {
		return err
	}
	if err := playerrecord.Lock(ctx, tx, proposal.Object.ID); err != nil {
		return err
	}
	applied, entry, err := h.applyCoins(ctx, tx, proposal.Object.ID, change, now)
	if err != nil {
		return err
	}
	decision := audit.Entry{Action: "approve", Object: proposal.Object, After: proposal.Change, Reason: note.Reason, Ref: note.Ref}
	recorded, err := h.audit.In(tx).Append(ctx, h.stamp(p, decision), now)
	if err != nil || applied.Repeated {
		return err
	}
	entry.Reason, entry.ApprovalID = proposal.Reason, recorded.ID
	_, err = h.audit.In(tx).Append(ctx, h.stamp(p, entry), now)
	return err
}
