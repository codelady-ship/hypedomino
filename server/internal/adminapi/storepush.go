package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/storeprice"
	"github.com/Alievs-corp/hypedomino/server/internal/storesync"
)

const PushStorePricesAction = "push"

var errNotPublished = errors.New("adminapi: no store prices are published")

type storePushView struct {
	ConfigVersion int64              `json:"config_version"`
	Stores        []storesync.Report `json:"stores"`
}

func (h *Handler) publishedTargets(ctx context.Context) (int64, []storeprice.Target, error) {
	version, raw, err := latestSnapshot(ctx, h.pool, storeprice.Domain)
	if err != nil {
		return 0, nil, err
	}
	if raw == nil {
		return 0, nil, errNotPublished
	}
	config, err := storeprice.Parse(raw)
	if err != nil {
		return 0, nil, err
	}
	return version, config.Targets(), nil
}

func (h *Handler) planStorePush(w http.ResponseWriter, r *http.Request, _ principal) {
	version, targets, err := h.publishedTargets(r.Context())
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, storePushView{ConfigVersion: version, Stores: h.config.Stores.Run(r.Context(), targets, false)})
}

func (h *Handler) applyStorePush(w http.ResponseWriter, r *http.Request, p principal) {
	var note changeNote
	if !decode(w, r, &note) {
		return
	}
	if err := h.sensitive(p, note, h.now()); err != nil {
		h.refuse(w, err)
		return
	}
	ctx := context.WithoutCancel(r.Context())
	version, targets, err := h.publishedTargets(ctx)
	if err != nil {
		h.refuse(w, err)
		return
	}
	reports := h.config.Stores.Run(ctx, targets, true)
	if err := h.recordStorePush(ctx, p, note, version, reports); err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, storePushView{ConfigVersion: version, Stores: reports})
}

func (h *Handler) recordStorePush(ctx context.Context, p principal, note changeNote, version int64, reports []storesync.Report) error {
	return h.audited(ctx, p, func(pgx.Tx, time.Time) ([]audit.Entry, error) {
		entries := []audit.Entry{}
		for _, report := range reports {
			if !report.Configured {
				continue
			}
			counts, err := json.Marshal(report.Counts)
			if err != nil {
				return nil, err
			}
			entries = append(entries, audit.Entry{
				Action: PushStorePricesAction, Object: audit.Object{Type: rbac.StoreProducts, ID: report.Store},
				After: counts, Reason: note.Reason, Ref: note.Ref, ConfigVersion: version,
			})
		}
		return entries, nil
	})
}
