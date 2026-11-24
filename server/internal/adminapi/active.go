package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/gamestrings"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

type activeRequest struct {
	Bundles        map[gamestrings.Language]int                    `json:"bundles"`
	FallbackChains map[gamestrings.Language][]gamestrings.Language `json:"fallback_chains"`
	changeNote
}

type activeView struct {
	ConfigVersion  int64                                           `json:"config_version"`
	Bundles        map[gamestrings.Language]int                    `json:"bundles"`
	FallbackChains map[gamestrings.Language][]gamestrings.Language `json:"fallback_chains"`
}

type published struct {
	version      int64
	raw          []byte
	localization gamestrings.Localization
}

func latestLocalization(ctx context.Context, db snapshot.Querier) (published, error) {
	version, raw, err := snapshot.Latest(ctx, db, gamestrings.Domain)
	if errors.Is(err, snapshot.ErrNone) {
		return published{localization: gamestrings.Localization{Bundles: map[gamestrings.Language]int{}, FallbackChains: gamestrings.DefaultFallbackChains()}}, nil
	}
	if err != nil {
		return published{}, err
	}
	localization, err := gamestrings.ParseLocalization(raw)
	return published{version: version, raw: raw, localization: localization}, err
}

func viewActive(p published) activeView {
	return activeView{ConfigVersion: p.version, Bundles: p.localization.Bundles, FallbackChains: p.localization.FallbackChains}
}

func rollsBack(previous, next map[gamestrings.Language]int) bool {
	for language, version := range previous {
		if next[language] < version {
			return true
		}
	}
	return false
}

func lockDomain(ctx context.Context, tx pgx.Tx, domain string) error {
	_, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtext('config_snapshots:' || $1))`, domain)
	return err
}

func (h *Handler) mayRollBack(p principal, note changeNote, now time.Time) error {
	if h.allows(p, rbac.Snapshots, rbac.Rollback) != nil && h.allows(p, rbac.Snapshots, rbac.RollbackOwnDomains) != nil {
		return errForbidden
	}
	return h.sensitive(p, note, now)
}

func (h *Handler) activeStrings(w http.ResponseWriter, r *http.Request, p principal) {
	current, err := latestLocalization(r.Context(), h.pool)
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, viewActive(current))
}

func (h *Handler) activateStrings(w http.ResponseWriter, r *http.Request, p principal) {
	var request activeRequest
	if !decode(w, r, &request) {
		return
	}
	raw := marshal(gamestrings.Localization{Bundles: request.Bundles, FallbackChains: request.FallbackChains})
	next, err := gamestrings.ParseLocalization(raw)
	if err != nil {
		h.refuse(w, err)
		return
	}
	var activated published
	err = h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := lockDomain(r.Context(), tx, gamestrings.Domain); err != nil {
			return nil, err
		}
		current, err := latestLocalization(r.Context(), tx)
		if err != nil {
			return nil, err
		}
		action := "publish"
		if rollsBack(current.localization.Bundles, next.Bundles) {
			action = "rollback"
			if err := h.mayRollBack(p, request.changeNote, now); err != nil {
				return nil, err
			}
		}
		if _, err := h.strings.In(tx).Resolve(r.Context(), next); err != nil {
			return nil, err
		}
		version, err := snapshot.Publish(r.Context(), tx, gamestrings.Domain, raw)
		if err != nil {
			return nil, err
		}
		activated = published{version: version, raw: raw, localization: next}
		entry := audit.Entry{
			Action: action, Object: audit.Object{Type: rbac.Snapshots, ID: gamestrings.Domain}, After: json.RawMessage(raw),
			ConfigVersion: version, Reason: request.Reason, Ref: request.Ref,
		}
		if current.raw != nil {
			entry.Before = current.raw
		}
		return []audit.Entry{entry}, nil
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, viewActive(activated))
}
