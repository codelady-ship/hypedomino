package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/approval"
	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/medal"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/playerrecord"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const (
	GrantRewardAction = "grant_reward"
	mythicRarity      = "mythic"
)

var (
	errBadGrant      = errors.New("adminapi: a grant needs a bundle, a listed source and a request id")
	grantSourceTypes = []string{"shop", "pass", "mission", "promo", "rank_reward", "first_purchase", "event", "chest"}
)

type grantChange struct {
	Bundle    *reward.Bundle   `json:"bundle"`
	Source    cosmetics.Source `json:"source"`
	RequestID string           `json:"request_id"`
	Proposer  string           `json:"proposer_id"`
}

type grantRequest struct {
	Bundle    json.RawMessage  `json:"bundle"`
	Source    cosmetics.Source `json:"source"`
	RequestID string           `json:"request_id"`
	changeNote
}

type grantView struct {
	AccountID string        `json:"account_id"`
	Report    reward.Report `json:"report"`
	Repeated  bool          `json:"repeated"`
}

func (r grantRequest) change(proposer string) (grantChange, error) {
	bundle, err := reward.ParseBundle(r.Bundle)
	switch {
	case err != nil:
		return grantChange{}, err
	case !requestIDPattern.MatchString(r.RequestID), !slices.Contains(grantSourceTypes, r.Source.Type), r.Source.Ref == "":
		return grantChange{}, errBadGrant
	}
	if err := bundle.ValidateFor(wallet.AdminAdjust); err != nil {
		return grantChange{}, err
	}
	return grantChange{Bundle: &bundle, Source: r.Source, RequestID: r.RequestID, Proposer: proposer}, nil
}

func (h *Handler) grantReward(w http.ResponseWriter, r *http.Request, p principal) {
	var request grantRequest
	if !decode(w, r, &request) {
		return
	}
	change, err := request.change(p.account.ID)
	if err != nil {
		h.refuse(w, err)
		return
	}
	engine, catalog, err := h.rewardEngine(r.Context())
	if err != nil {
		h.refuse(w, err)
		return
	}
	accountID := r.PathValue("id")
	var proposal approval.Proposal
	var report reward.Report
	err = h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := h.sensitive(p, request.changeNote, now); err != nil {
			return nil, err
		}
		if err := playerrecord.Lock(r.Context(), tx, accountID); err != nil {
			return nil, err
		}
		if h.grantNeedsSecondAdmin(*change.Bundle, catalog) {
			var err error
			proposal, err = h.approvals.In(tx).Propose(r.Context(), approval.PlayerGrant, accountID, marshal(change), request.Reason, p.actor(), now)
			entry := audit.Entry{Action: "submit", Object: audit.Object{Type: rbac.Players, ID: accountID}, After: marshal(change), Reason: request.Reason, Ref: request.Ref}
			return []audit.Entry{entry}, err
		}
		var entry audit.Entry
		var err error
		report, entry, err = applyGrant(r.Context(), tx, engine, accountID, change, now)
		if err != nil || report.Repeated {
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
	reply(w, http.StatusOK, grantView{AccountID: accountID, Report: report, Repeated: report.Repeated})
}

func (h *Handler) grantNeedsSecondAdmin(bundle reward.Bundle, catalog *cosmetics.Catalog) bool {
	if bundle.Coins > h.config.GrantApprovalCoins {
		return true
	}
	return slices.ContainsFunc(bundle.Cosmetics, func(id string) bool {
		item, ok := catalog.Item(id)
		return ok && item.Rarity == mythicRarity
	})
}

func (h *Handler) rewardEngine(ctx context.Context) (*reward.Engine, *cosmetics.Catalog, error) {
	catalog, err := cosmetics.Published(ctx, h.pool)
	if err != nil {
		return nil, nil, err
	}
	medals := medal.NewWatcher(h.pool, h.logger)
	if _, err := medals.Load(ctx); err != nil && !errors.Is(err, snapshot.ErrNone) {
		return nil, nil, err
	}
	seasons := pass.NewWatcher(h.pool, h.logger)
	if _, err := seasons.Load(ctx); err != nil && !errors.Is(err, snapshot.ErrNone) {
		return nil, nil, err
	}
	items := inventory.NewStore(h.pool, catalog, inventory.DefaultConfig())
	if _, _, running := seasons.Current(); !running {
		return reward.NewEngine(h.pool, h.wallet, items, medal.NewStore(h.pool, medals), nil, reward.DefaultConfig()), catalog, nil
	}
	passes := pass.NewStore(h.pool, seasons)
	engine := reward.NewEngine(h.pool, h.wallet, items, medal.NewStore(h.pool, medals), passes, reward.DefaultConfig())
	passes.SetRewards(engine)
	return engine, catalog, nil
}

func applyGrant(ctx context.Context, tx pgx.Tx, engine *reward.Engine, accountID string, change grantChange, now time.Time) (reward.Report, audit.Entry, error) {
	report, err := engine.ApplyIn(ctx, tx, reward.Grant{
		Account: accountID, Bundle: *change.Bundle, Source: change.Source, Reason: wallet.AdminAdjust,
		LedgerSource: wallet.Source{Type: adminSource, Ref: change.Proposer},
		Key:          fmt.Sprintf("admin-grant:%s:%s", change.Proposer, change.RequestID),
	}, now)
	if err != nil {
		return reward.Report{}, audit.Entry{}, err
	}
	if report.Repeated && !sameGrant(report, change) {
		return reward.Report{}, audit.Entry{}, fmt.Errorf("%w: request id %s was used for bundle %s", errBadGrant, change.RequestID, report.BundleID)
	}
	entry := audit.Entry{Action: GrantRewardAction, Object: audit.Object{Type: rbac.Players, ID: accountID}, After: marshal(report)}
	return report, entry, nil
}

func sameGrant(report reward.Report, change grantChange) bool {
	return report.BundleID == change.Bundle.ID && report.Revision == change.Bundle.Revision && report.Source == change.Source
}

func (h *Handler) approvePlayerGrant(ctx context.Context, tx pgx.Tx, p principal, proposal approval.Proposal, note changeNote, now time.Time) error {
	var change grantChange
	if err := json.Unmarshal(proposal.Change, &change); err != nil {
		return err
	}
	if change.Bundle == nil {
		return h.grantPlayerCoins(ctx, tx, p, proposal, note, now)
	}
	engine, _, err := h.rewardEngine(ctx)
	if err != nil {
		return err
	}
	if err := playerrecord.Lock(ctx, tx, proposal.Object.ID); err != nil {
		return err
	}
	report, entry, err := applyGrant(ctx, tx, engine, proposal.Object.ID, change, now)
	if err != nil {
		return err
	}
	decision := audit.Entry{Action: "approve", Object: proposal.Object, After: proposal.Change, Reason: note.Reason, Ref: note.Ref}
	recorded, err := h.audit.In(tx).Append(ctx, h.stamp(p, decision), now)
	if err != nil || report.Repeated {
		return err
	}
	entry.Reason, entry.ApprovalID = proposal.Reason, recorded.ID
	_, err = h.audit.In(tx).Append(ctx, h.stamp(p, entry), now)
	return err
}
