package adminapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/promo"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
)

const (
	CreatePromoTemplateAction = "create_promo_template"
	EditPromoTemplateAction   = "edit_promo_template"
)

type promoTemplateRequest struct {
	ID       string          `json:"id"`
	Campaign string          `json:"campaign"`
	Kind     string          `json:"kind"`
	Bundle   json.RawMessage `json:"bundle"`
	changeNote
}

type promoTemplatesView struct {
	Templates []promo.Template `json:"templates"`
}

func (r promoTemplateRequest) template(id, creator string) (promo.Template, error) {
	t := promo.Template{ID: id, Campaign: r.Campaign, Kind: r.Kind, Creator: creator}
	if len(r.Bundle) == 0 || string(r.Bundle) == "null" {
		return t, nil
	}
	bundle, err := reward.ParseBundle(r.Bundle)
	if err != nil {
		return promo.Template{}, fmt.Errorf("%w: %v", promo.ErrInvalid, err)
	}
	t.Bundle = &bundle
	return t, nil
}

func (h *Handler) listPromoTemplates(w http.ResponseWriter, r *http.Request, p principal) {
	promos, err := h.promoStore()
	if err != nil {
		h.refuse(w, err)
		return
	}
	templates, err := promos.Templates(r.Context())
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, promoTemplatesView{Templates: templates})
}

func (h *Handler) createPromoTemplate(w http.ResponseWriter, r *http.Request, p principal) {
	h.savePromoTemplate(w, r, p, false)
}

func (h *Handler) editPromoTemplate(w http.ResponseWriter, r *http.Request, p principal) {
	h.savePromoTemplate(w, r, p, true)
}

func (h *Handler) savePromoTemplate(w http.ResponseWriter, r *http.Request, p principal, existing bool) {
	promos, err := h.promoStore()
	if err != nil {
		h.refuse(w, err)
		return
	}
	var request promoTemplateRequest
	if !decode(w, r, &request) {
		return
	}
	id, status := request.ID, http.StatusCreated
	if existing {
		id, status = r.PathValue("id"), http.StatusOK
	}
	var saved promo.Template
	err = h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := h.noted(request.changeNote); err != nil {
			return nil, err
		}
		t, err := request.template(id, p.account.ID)
		if err != nil {
			return nil, err
		}
		entry := audit.Entry{Object: audit.Object{Type: rbac.PromoCodes, ID: id}, Reason: request.Reason, Ref: request.Ref}
		if !existing {
			saved, err = promos.In(tx).CreateTemplate(r.Context(), t, now)
			entry.Action, entry.After = CreatePromoTemplateAction, marshal(saved)
			return []audit.Entry{entry}, err
		}
		var before promo.Template
		before, saved, err = promos.In(tx).EditTemplate(r.Context(), t, now)
		entry.Action, entry.Before, entry.After = EditPromoTemplateAction, marshal(before), marshal(saved)
		return []audit.Entry{entry}, err
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, status, saved)
}
