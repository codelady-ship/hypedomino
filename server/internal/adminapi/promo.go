package adminapi

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/approval"
	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/promo"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
)

const (
	MaxPromoBatchCount = 10000
	PromoPageSize      = 50
	PromoPageMax       = 200

	CreatePromoAction   = "create_promo"
	BatchPromoAction    = "create_promo_batch"
	DisablePromoAction  = "disable_promo"
	ActivatePromoAction = "activate_promo"
	DiscardPromoAction  = "discard_promo"
)

var (
	errPromoNotConfigured = errors.New("adminapi: promo codes need HD_PROMO_KEY")
	errBadBatchCount      = fmt.Errorf("adminapi: a batch has 1 to %d codes", MaxPromoBatchCount)
	errAccountsBySupport  = fmt.Errorf("%w: support lists the accounts of a targeted code", promo.ErrInvalid)
)

type promoRequest struct {
	ID          string          `json:"id"`
	Campaign    string          `json:"campaign"`
	Kind        string          `json:"kind"`
	Bundle      json.RawMessage `json:"bundle"`
	Start       time.Time       `json:"start"`
	End         time.Time       `json:"end"`
	Cap         int             `json:"cap"`
	Code        string          `json:"code"`
	Accounts    []string        `json:"accounts"`
	Count       int             `json:"count"`
	PassPremium bool            `json:"passPremium"`
	changeNote
}

type promoChange struct {
	Batch bool       `json:"batch"`
	Count int        `json:"count"`
	Code  promo.Code `json:"code"`
}

type promoState struct {
	State string `json:"state"`
	Codes int    `json:"codes"`
}

type promoListView struct {
	Campaigns []promo.CampaignUsage `json:"campaigns"`
	Codes     []promo.Code          `json:"codes"`
}

type createdPromoView struct {
	Code       promo.Code `json:"code"`
	Plaintext  string     `json:"plaintext"`
	ApprovalID int64      `json:"approval_id,omitempty"`
}

type disabledPromoView struct {
	ID       string `json:"id"`
	Disabled int    `json:"disabled"`
}

func (h *Handler) promoStore() (*promo.Store, error) {
	if h.promos == nil {
		return nil, errPromoNotConfigured
	}
	return h.promos, nil
}

func (r promoRequest) definition(creator string) (promo.Definition, error) {
	bundle, err := reward.ParseBundle(r.Bundle)
	if err != nil {
		return promo.Definition{}, fmt.Errorf("%w: %v", promo.ErrInvalid, err)
	}
	return promo.Definition{
		ID: r.ID, Campaign: r.Campaign, Kind: r.Kind, Bundle: bundle, Start: r.Start, End: r.End,
		Cap: r.Cap, Code: r.Code, Accounts: r.Accounts, Creator: creator, PassPremium: r.PassPremium,
	}, nil
}

func (h *Handler) listPromos(w http.ResponseWriter, r *http.Request, p principal) {
	promos, err := h.promoStore()
	if err != nil {
		h.refuse(w, err)
		return
	}
	campaigns, err := promos.Campaigns(r.Context())
	if err != nil {
		h.refuse(w, err)
		return
	}
	query := r.URL.Query()
	limit, err := optionalNumber(query.Get("limit"))
	if err != nil {
		h.refuse(w, err)
		return
	}
	if limit == 0 {
		limit = PromoPageSize
	}
	codes, err := promos.List(r.Context(), promo.ListQuery{
		Campaign: query.Get("campaign"), State: query.Get("state"), IDPrefix: query.Get("id"),
		Before: query.Get("before"), Limit: int(min(limit, PromoPageMax)),
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, promoListView{Campaigns: campaigns, Codes: codes})
}

func (h *Handler) proposePromo(ctx context.Context, tx pgx.Tx, p principal, objectID string, change promoChange, reason string, now time.Time) (int64, error) {
	if change.Code.State != promo.StatePending {
		return 0, nil
	}
	proposal, err := h.approvals.In(tx).Propose(ctx, approval.PromoCode, objectID, marshal(change), reason, p.actor(), now)
	return proposal.ID, err
}

func (h *Handler) createPromo(w http.ResponseWriter, r *http.Request, p principal) {
	promos, err := h.promoStore()
	if err != nil {
		h.refuse(w, err)
		return
	}
	var request promoRequest
	if !decode(w, r, &request) {
		return
	}
	var view createdPromoView
	err = h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := h.noted(request.changeNote); err != nil {
			return nil, err
		}
		if len(request.Accounts) > 0 {
			return nil, errAccountsBySupport
		}
		definition, err := request.definition(p.account.ID)
		if err != nil {
			return nil, err
		}
		created, err := promos.In(tx).Create(r.Context(), definition, now)
		if err != nil {
			return nil, err
		}
		view.Code, view.Plaintext = created.Code, created.Plaintext
		change := promoChange{Count: 1, Code: created.Code}
		if view.ApprovalID, err = h.proposePromo(r.Context(), tx, p, created.Code.ID, change, request.Reason, now); err != nil {
			return nil, err
		}
		return []audit.Entry{{
			Action: CreatePromoAction, Object: audit.Object{Type: rbac.PromoCodes, ID: created.Code.ID},
			After: marshal(change), Reason: request.Reason, Ref: request.Ref,
		}}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusCreated, view)
}

func (h *Handler) createPromoBatch(w http.ResponseWriter, r *http.Request, p principal) {
	promos, err := h.promoStore()
	if err != nil {
		h.refuse(w, err)
		return
	}
	var request promoRequest
	if !decode(w, r, &request) {
		return
	}
	var created []promo.Created
	var approvalID int64
	err = h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := h.noted(request.changeNote); err != nil {
			return nil, err
		}
		if request.Count < 1 || request.Count > MaxPromoBatchCount {
			return nil, errBadBatchCount
		}
		definition, err := request.definition(p.account.ID)
		if err != nil {
			return nil, err
		}
		definition.Kind = promo.SingleUse
		if created, err = promos.In(tx).Batch(r.Context(), definition, request.Count, now); err != nil {
			return nil, err
		}
		sample := created[0].Code
		sample.ID = request.ID
		change := promoChange{Batch: true, Count: len(created), Code: sample}
		if approvalID, err = h.proposePromo(r.Context(), tx, p, request.ID, change, request.Reason, now); err != nil {
			return nil, err
		}
		return []audit.Entry{{
			Action: BatchPromoAction, Object: audit.Object{Type: rbac.PromoCodes, ID: request.ID},
			After: marshal(change), Reason: request.Reason, Ref: request.Ref,
		}}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	writeCodes(w, request.ID, created, approvalID)
}

func writeCodes(w http.ResponseWriter, batch string, created []promo.Created, approvalID int64) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", batch+".csv"))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Hd-Promo-State", created[0].Code.State)
	if approvalID != 0 {
		w.Header().Set("Hd-Approval-Id", strconv.FormatInt(approvalID, 10))
	}
	w.WriteHeader(http.StatusCreated)
	out := csv.NewWriter(w)
	out.Write([]string{"promo_id", "code"})
	for _, c := range created {
		out.Write([]string{c.Code.ID, c.Plaintext})
	}
	out.Flush()
}

func (h *Handler) disablePromo(w http.ResponseWriter, r *http.Request, p principal) {
	h.disablePromos(w, r, p, false)
}

func (h *Handler) disablePromoBatch(w http.ResponseWriter, r *http.Request, p principal) {
	h.disablePromos(w, r, p, true)
}

func (h *Handler) disablePromos(w http.ResponseWriter, r *http.Request, p principal, batch bool) {
	promos, err := h.promoStore()
	if err != nil {
		h.refuse(w, err)
		return
	}
	var note changeNote
	if !decode(w, r, &note) {
		return
	}
	id := r.PathValue("id")
	view := disabledPromoView{ID: id}
	err = h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := h.sensitive(p, note, now); err != nil {
			return nil, err
		}
		var err error
		if view.Disabled, err = disableCodes(r.Context(), promos.In(tx), id, batch, now); err != nil || view.Disabled == 0 {
			return nil, err
		}
		return []audit.Entry{{
			Action: DisablePromoAction, Object: audit.Object{Type: rbac.PromoCodes, ID: id},
			After: marshal(promoState{State: promo.StateDisabled, Codes: view.Disabled}), Reason: note.Reason, Ref: note.Ref,
		}}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, view)
}

func disableCodes(ctx context.Context, promos *promo.Store, id string, batch bool, now time.Time) (int, error) {
	if batch {
		return promos.DisableBatch(ctx, id, now)
	}
	code, err := promos.Get(ctx, id)
	if err != nil || code.State == promo.StateDisabled {
		return 0, err
	}
	_, err = promos.Disable(ctx, id, now)
	return 1, err
}

func (h *Handler) activatePromo(ctx context.Context, tx pgx.Tx, p principal, proposal approval.Proposal, note changeNote, now time.Time) error {
	promos, err := h.promoStore()
	if err != nil {
		return err
	}
	var change promoChange
	if err := json.Unmarshal(proposal.Change, &change); err != nil {
		return err
	}
	activated := 1
	if change.Batch {
		activated, err = promos.In(tx).ApproveBatch(ctx, proposal.Object.ID, p.account.ID)
	} else {
		_, err = promos.In(tx).Approve(ctx, proposal.Object.ID, p.account.ID, now)
	}
	if err != nil {
		return err
	}
	decision := audit.Entry{Action: "approve", Object: proposal.Object, After: proposal.Change, Reason: note.Reason, Ref: note.Ref}
	recorded, err := h.audit.In(tx).Append(ctx, h.stamp(p, decision), now)
	if err != nil {
		return err
	}
	applied := audit.Entry{
		Action: ActivatePromoAction, Object: proposal.Object, Reason: proposal.Reason, ApprovalID: recorded.ID,
		After: marshal(promoState{State: promo.StateActive, Codes: activated}),
	}
	_, err = h.audit.In(tx).Append(ctx, h.stamp(p, applied), now)
	return err
}

func (h *Handler) discardPromo(ctx context.Context, tx pgx.Tx, proposal approval.Proposal, now time.Time) (audit.Entry, error) {
	promos, err := h.promoStore()
	if err != nil {
		return audit.Entry{}, err
	}
	var change promoChange
	if err := json.Unmarshal(proposal.Change, &change); err != nil {
		return audit.Entry{}, err
	}
	disabled, err := disableCodes(ctx, promos.In(tx), proposal.Object.ID, change.Batch, now)
	if err != nil {
		return audit.Entry{}, err
	}
	return audit.Entry{
		Action: DiscardPromoAction, Object: proposal.Object, Reason: proposal.Reason,
		After: marshal(promoState{State: promo.StateDisabled, Codes: disabled}),
	}, nil
}
