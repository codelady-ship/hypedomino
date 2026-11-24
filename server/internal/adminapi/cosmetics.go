package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/contentpublish"
	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/gamestrings"
	"github.com/Alievs-corp/hypedomino/server/internal/mission"
	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
)

var errCosmeticArtMissing = errors.New("adminapi: a cosmetic item is added only after its static upload passed its checks")

type cosmeticRulesView struct {
	Types        []string        `json:"types"`
	Rarities     []string        `json:"rarities"`
	Availability []string        `json:"availability"`
	SourceTypes  []string        `json:"source_types"`
	ArtSources   []string        `json:"art_sources"`
	Builtin      json.RawMessage `json:"builtin"`
}

type cosmeticCatalogView struct {
	catalogView
	Rules cosmeticRulesView `json:"rules"`
}

func (h *Handler) cosmeticCatalog(w http.ResponseWriter, r *http.Request, p principal) {
	version, raw, err := latestSnapshot(r.Context(), h.pool, cosmetics.Domain)
	if err != nil {
		h.refuse(w, err)
		return
	}
	builtin, err := cosmetics.BuiltinManifest().Encode()
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, cosmeticCatalogView{
		catalogView: viewCatalog(version, raw),
		Rules: cosmeticRulesView{
			Types:        cosmetics.Types,
			Rarities:     rarityOrder(cosmetics.DefaultRarities()),
			Availability: []string{cosmetics.Live, cosmetics.Retired, cosmetics.Disabled},
			SourceTypes:  cosmetics.SourceTypes,
			ArtSources:   cosmetics.ArtSources,
			Builtin:      builtin,
		},
	})
}

func asPublisher(handle func(func() configDomain) func(http.ResponseWriter, *http.Request, principal), domain func(publisher string) configDomain) authedHandler {
	return func(w http.ResponseWriter, r *http.Request, p principal) {
		handle(func() configDomain { return domain(p.account.ID) })(w, r, p)
	}
}

func (h *Handler) cosmeticsDomain(rollback, incident bool, publisher string) configDomain {
	return configDomain{name: cosmetics.Domain, object: rbac.CosmeticItems,
		prepare: func(ctx context.Context, tx pgx.Tx, before, after []byte) ([]byte, error) {
			previous, err := publishedManifest(before)
			if err != nil {
				return nil, err
			}
			next, err := nextManifest(previous, after, rollback)
			if err != nil {
				return nil, err
			}
			added, err := cosmetics.Added(previous, next)
			if err != nil {
				return nil, err
			}
			if err := reviewedByAnother(ctx, tx, added, publisher); err != nil {
				return nil, err
			}
			if err := staticsPassed(ctx, tx, added); err != nil {
				return nil, err
			}
			raw, err := next.Encode()
			if err != nil {
				return nil, err
			}
			items, err := cosmetics.Parse(raw)
			if err != nil {
				return nil, err
			}
			if err := cosmetics.ValidateManifest(next, cosmetics.DefaultRarities()); err != nil {
				return nil, err
			}
			if err := cosmetics.CheckSuccessor(previous, next); err != nil {
				return nil, err
			}
			if err := h.grantsStayListed(ctx, tx, items); err != nil {
				return nil, err
			}
			if !incident {
				if err := h.namedInEnglish(ctx, tx, items); err != nil {
					return nil, err
				}
			}
			return raw, h.uploadManifest(ctx, raw)
		},
		accept: func(_, after []byte) error {
			_, err := cosmetics.ParseManifest(after)
			return err
		},
	}
}

func (h *Handler) uploadManifest(ctx context.Context, raw []byte) error {
	if h.config.Content == nil {
		return errContentStoreOff
	}
	file := cosmetics.File(cosmetics.Hash(raw))
	object := objectstore.Object{ContentType: "application/json", CacheControl: contentpublish.ImmutableCacheControl, Body: raw}
	if err := h.config.Content.PutObject(ctx, file, object, h.now()); err != nil {
		return fmt.Errorf("%w: %s: %v", errContentUpload, file, err)
	}
	return nil
}

func (h *Handler) namedInEnglish(ctx context.Context, tx pgx.Tx, items *cosmetics.Catalog) error {
	store := h.strings.In(tx)
	var unnamed, keys []string
	for _, item := range items.Items() {
		key := cosmetics.NameKeyOf(item.ID)
		value, err := store.Value(ctx, key, gamestrings.English)
		if err != nil {
			return err
		}
		if value.Status != gamestrings.Approved {
			unnamed = append(unnamed, item.ID)
			keys = append(keys, key)
		}
	}
	if len(unnamed) > 0 {
		return unapprovedKeysError{refusal: fmt.Errorf("%w: V-16 no approved en name for %s", cosmetics.ErrInvalid, strings.Join(unnamed, ", ")), keys: keys}
	}
	return nil
}

func reviewedByAnother(ctx context.Context, tx pgx.Tx, added []string, publisher string) error {
	if len(added) == 0 {
		return nil
	}
	var item string
	err := tx.QueryRow(ctx, `select item_id from cosmetic_assets where item_id = any($1) and uploaded_by = $2 limit 1`, added, publisher).Scan(&item)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("%w: %s", errOwnUpload, item)
}

func staticsPassed(ctx context.Context, tx pgx.Tx, added []string) error {
	if len(added) == 0 {
		return nil
	}
	missing, err := withoutPassedStatic(ctx, tx, added)
	if err != nil || len(missing) == 0 {
		return err
	}
	return fmt.Errorf("%w: %s", errCosmeticArtMissing, strings.Join(missing, ", "))
}

func (h *Handler) switchOffCosmetic(w http.ResponseWriter, r *http.Request, p principal) {
	var note changeNote
	if !decode(w, r, &note) {
		return
	}
	var version int64
	var published []byte
	err := h.audited(r.Context(), p, func(tx pgx.Tx, now time.Time) ([]audit.Entry, error) {
		if err := h.incidentSwitchOff(p, note, now); err != nil {
			return nil, err
		}
		raw, err := disabledManifest(r.Context(), tx, r.PathValue("id"))
		if err != nil {
			return nil, err
		}
		entry, raw, err := publishIn(r.Context(), tx, h.cosmeticsDomain(false, true, p.account.ID), string(rbac.Retire), raw, note)
		version, published = entry.ConfigVersion, raw
		return []audit.Entry{entry}, err
	})
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, viewCatalog(version, published))
}

func (h *Handler) incidentSwitchOff(p principal, note changeNote, now time.Time) error {
	if strings.TrimSpace(note.Ref) == "" {
		return fmt.Errorf("%w: ref names the incident", errMalformed)
	}
	return h.sensitive(p, note, now)
}

func disabledManifest(ctx context.Context, tx pgx.Tx, id string) ([]byte, error) {
	if err := lockDomain(ctx, tx, cosmetics.Domain); err != nil {
		return nil, err
	}
	_, before, err := latestSnapshot(ctx, tx, cosmetics.Domain)
	if err != nil {
		return nil, err
	}
	current, err := publishedManifest(before)
	if err != nil {
		return nil, err
	}
	switched, err := current.Disable(id)
	if err != nil {
		return nil, err
	}
	return switched.Encode()
}

func publishedManifest(before []byte) (cosmetics.Manifest, error) {
	if before == nil {
		return cosmetics.BuiltinManifest(), nil
	}
	return cosmetics.ParseManifest(before)
}

func nextManifest(previous cosmetics.Manifest, after []byte, rollback bool) (cosmetics.Manifest, error) {
	if !rollback {
		return cosmetics.Stamp(previous, after)
	}
	target, err := cosmetics.ParseManifest(after)
	if err != nil {
		return cosmetics.Manifest{}, err
	}
	return cosmetics.Restore(previous, target)
}

func (h *Handler) grantsStayListed(ctx context.Context, tx pgx.Tx, items *cosmetics.Catalog) error {
	medals, err := publishedMedals(ctx, tx)
	if err != nil {
		return err
	}
	_, raw, err := latestSnapshot(ctx, tx, mission.Domain)
	if err != nil {
		return err
	}
	if raw != nil {
		missions, err := mission.Parse(raw)
		if err != nil {
			return err
		}
		if err := missionSourcesIn(items, missions, medals); err != nil {
			return err
		}
	}
	if _, raw, err = latestSnapshot(ctx, tx, pass.Domain); err != nil {
		return err
	}
	if raw != nil {
		config, err := pass.Parse(raw)
		if err != nil {
			return err
		}
		if err := trackSourcesIn(items, config); err != nil {
			return err
		}
	}
	ranked, err := publishedRating(ctx, tx)
	if err != nil {
		return err
	}
	if ranked != nil {
		if err := rankRewardSourcesIn(items, ranked, medals); err != nil {
			return err
		}
	}
	if _, raw, err = latestSnapshot(ctx, tx, shop.Domain); err != nil || raw == nil {
		return err
	}
	catalog, err := shop.Parse(raw, h.config.Shop)
	if err != nil {
		return err
	}
	return catalog.CheckSources(items)
}
