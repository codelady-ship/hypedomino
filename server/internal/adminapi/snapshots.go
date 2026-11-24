package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

const ConfigHistoryLimit = 50

type configDomain struct {
	name    string
	object  rbac.Object
	prepare func(ctx context.Context, tx pgx.Tx, before, after []byte) ([]byte, error)
	accept  func(before, after []byte) error
}

type sensitiveCheck func(p principal, note changeNote, now time.Time) error

func latestSnapshot(ctx context.Context, db snapshot.Querier, domain string) (int64, []byte, error) {
	version, raw, err := snapshot.Latest(ctx, db, domain)
	if errors.Is(err, snapshot.ErrNone) {
		return 0, nil, nil
	}
	return version, raw, err
}

func (h *Handler) earlierSnapshot(ctx context.Context, domain string, version int64) ([]byte, error) {
	target, err := snapshot.Version(ctx, h.pool, domain, version)
	if errors.Is(err, snapshot.ErrNone) {
		return nil, errNoSuchVersion
	}
	if err != nil {
		return nil, err
	}
	return target.Content, nil
}

func (h *Handler) replaceSnapshot(ctx context.Context, p principal, d configDomain, action string, raw []byte, note changeNote, check sensitiveCheck) (int64, []byte, error) {
	var version int64
	err := h.audited(ctx, p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if check != nil {
			if err := check(p, note, now); err != nil {
				return nil, err
			}
		}
		entry, published, err := publishIn(ctx, tx, d, action, raw, note)
		if err != nil {
			return nil, err
		}
		version, raw = entry.ConfigVersion, published
		return []audit.Entry{entry}, nil
	})
	return version, raw, err
}

func publishIn(ctx context.Context, tx pgx.Tx, d configDomain, action string, raw []byte, note changeNote) (audit.Entry, []byte, error) {
	if err := lockDomain(ctx, tx, d.name); err != nil {
		return audit.Entry{}, nil, err
	}
	_, before, err := latestSnapshot(ctx, tx, d.name)
	if err != nil {
		return audit.Entry{}, nil, err
	}
	if d.prepare != nil {
		if raw, err = d.prepare(ctx, tx, before, raw); err != nil {
			return audit.Entry{}, nil, err
		}
	}
	if err := d.accept(before, raw); err != nil {
		return audit.Entry{}, nil, err
	}
	version, err := snapshot.Publish(ctx, tx, d.name, raw)
	if err != nil {
		return audit.Entry{}, nil, err
	}
	entry := audit.Entry{
		Action: action, Object: audit.Object{Type: d.object, ID: d.name}, After: json.RawMessage(raw),
		Before: before, ConfigVersion: version, Reason: note.Reason, Ref: note.Ref,
	}
	return entry, raw, nil
}

func (h *Handler) domainHistory(domain string) func(http.ResponseWriter, *http.Request, principal) {
	return func(w http.ResponseWriter, r *http.Request, p principal) {
		history, err := snapshot.History(r.Context(), h.pool, domain, ConfigHistoryLimit)
		if err != nil {
			h.refuse(w, err)
			return
		}
		versions := make([]catalogVersionView, 0, len(history))
		for _, published := range history {
			versions = append(versions, catalogVersionView{ConfigVersion: published.Version, PublishedAt: published.At, Catalog: published.Content})
		}
		reply(w, http.StatusOK, catalogHistoryView{Versions: versions})
	}
}

func (h *Handler) domainCatalog(domain string) func(http.ResponseWriter, *http.Request, principal) {
	return func(w http.ResponseWriter, r *http.Request, p principal) {
		version, raw, err := latestSnapshot(r.Context(), h.pool, domain)
		if err != nil {
			h.refuse(w, err)
			return
		}
		reply(w, http.StatusOK, viewCatalog(version, raw))
	}
}

func (h *Handler) publishDomain(domain func() configDomain) func(http.ResponseWriter, *http.Request, principal) {
	return h.publishNoted(func(changeNote) configDomain { return domain() })
}

func (h *Handler) publishNoted(domain func(note changeNote) configDomain) func(http.ResponseWriter, *http.Request, principal) {
	return func(w http.ResponseWriter, r *http.Request, p principal) {
		var request publishCatalogRequest
		if !decodeUpTo(w, r, &request, MaxCatalogBytes) {
			return
		}
		if len(request.Catalog) == 0 {
			h.refuse(w, fmt.Errorf("%w: catalog", errMalformed))
			return
		}
		version, published, err := h.replaceSnapshot(r.Context(), p, domain(request.changeNote), "publish", request.Catalog, request.changeNote, nil)
		if err != nil {
			h.refuse(w, err)
			return
		}
		reply(w, http.StatusOK, viewCatalog(version, published))
	}
}

type activeVersionView struct {
	ConfigVersion int64      `json:"config_version"`
	Domain        string     `json:"domain,omitempty"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`
}

func (h *Handler) activeConfigVersion(w http.ResponseWriter, r *http.Request, p principal) {
	active, err := snapshot.Newest(r.Context(), h.pool)
	if errors.Is(err, snapshot.ErrNone) {
		reply(w, http.StatusOK, activeVersionView{})
		return
	}
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, activeVersionView{ConfigVersion: active.Version, Domain: active.Domain, PublishedAt: &active.At})
}

func (h *Handler) rollbackDomain(domain func() configDomain) func(http.ResponseWriter, *http.Request, principal) {
	return func(w http.ResponseWriter, r *http.Request, p principal) {
		var request rollbackRequest
		if !decode(w, r, &request) {
			return
		}
		d := domain()
		raw, err := h.earlierSnapshot(r.Context(), d.name, request.Version)
		if err != nil {
			h.refuse(w, err)
			return
		}
		version, published, err := h.replaceSnapshot(r.Context(), p, d, "rollback", raw, request.changeNote, h.mayRollBack)
		if err != nil {
			h.refuse(w, err)
			return
		}
		reply(w, http.StatusOK, viewCatalog(version, published))
	}
}
