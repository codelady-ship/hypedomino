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

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
)

const bundleLibraryDomain = "reward_bundles"

var errBundleExists = errors.New("adminapi: a reward bundle with this id already exists")

type bundleLibrary struct {
	Bundles []reward.Bundle `json:"bundles"`
}

type createBundleRequest struct {
	ID string `json:"id"`
	editBundleRequest
}

type bundleCreateView struct {
	Bundle        reward.Bundle `json:"bundle"`
	ConfigVersion int64         `json:"config_version"`
}

func parseBundleLibrary(raw []byte) (bundleLibrary, error) {
	library := bundleLibrary{}
	if raw == nil {
		return library, nil
	}
	if err := json.Unmarshal(raw, &library); err != nil {
		return bundleLibrary{}, fmt.Errorf("%w: %v", reward.ErrInvalidBundle, err)
	}
	seen := map[string]bool{}
	for _, bundle := range library.Bundles {
		if err := bundle.Validate(); err != nil {
			return bundleLibrary{}, err
		}
		if seen[bundle.ID] {
			return bundleLibrary{}, fmt.Errorf("%w: %s", errBundleExists, bundle.ID)
		}
		seen[bundle.ID] = true
	}
	return library, nil
}

func bundleLibraryDomainConfig() configDomain {
	return configDomain{name: bundleLibraryDomain, object: rbac.RewardBundles, accept: func(_, after []byte) error {
		_, err := parseBundleLibrary(after)
		return err
	}}
}

func bundleLibraryAdding(bundle reward.Bundle) configDomain {
	domain := bundleLibraryDomainConfig()
	domain.prepare = func(_ context.Context, _ pgx.Tx, before, _ []byte) ([]byte, error) {
		library, err := parseBundleLibrary(before)
		if err != nil {
			return nil, err
		}
		library.Bundles = append(library.Bundles, bundle)
		return json.Marshal(library)
	}
	return domain
}

func bundleInLibrary(raw []byte, next reward.Bundle) ([]byte, error) {
	library, err := parseBundleLibrary(raw)
	if err != nil {
		return nil, err
	}
	for i := range library.Bundles {
		if library.Bundles[i].ID == next.ID {
			library.Bundles[i] = next
		}
	}
	return json.Marshal(library)
}

func (h *Handler) libraryBundles(ctx context.Context, index bundleIndex) error {
	_, raw, err := latestSnapshot(ctx, h.pool, bundleLibraryDomain)
	if err != nil || raw == nil {
		return err
	}
	library, err := parseBundleLibrary(raw)
	if err != nil {
		return err
	}
	for _, bundle := range library.Bundles {
		if err := index.add(bundle, bundleUse{Domain: bundleLibraryDomain, Object: bundle.ID}); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) createRewardBundle(w http.ResponseWriter, r *http.Request, p principal) {
	var request createBundleRequest
	if !decode(w, r, &request) {
		return
	}
	if err := h.noted(request.changeNote); err != nil {
		h.refuse(w, err)
		return
	}
	bundle := reward.Bundle{ID: request.ID, Revision: 1, Coins: request.Coins, Crystals: request.Crystals, Cosmetics: request.Cosmetics, Medals: request.Medals, PassXP: request.PassXP}
	if err := h.newBundle(r.Context(), bundle); err != nil {
		h.refuse(w, err)
		return
	}
	version, err := h.publishNewBundle(r.Context(), p, bundle, request.changeNote)
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusCreated, bundleCreateView{Bundle: bundle, ConfigVersion: version})
}

func (h *Handler) newBundle(ctx context.Context, bundle reward.Bundle) error {
	if err := bundle.Validate(); err != nil {
		return err
	}
	view, err := h.rewardBundles(ctx)
	if err != nil {
		return err
	}
	if slices.ContainsFunc(view.Bundles, func(row bundleRow) bool { return row.ID == bundle.ID }) {
		return fmt.Errorf("%w: %s", errBundleExists, bundle.ID)
	}
	return nil
}

func (h *Handler) publishNewBundle(ctx context.Context, p principal, bundle reward.Bundle, note changeNote) (int64, error) {
	var version int64
	err := h.audited(ctx, p, func(tx pgx.Tx, _ time.Time) ([]audit.Entry, error) {
		if err := bundleContentPublished(ctx, tx, bundle); err != nil {
			return nil, err
		}
		entry, _, err := publishIn(ctx, tx, bundleLibraryAdding(bundle), "publish", nil, note)
		if err != nil {
			return nil, err
		}
		version = entry.ConfigVersion
		created, err := json.Marshal(bundle)
		if err != nil {
			return nil, err
		}
		return []audit.Entry{entry, {
			Action: "create", Object: audit.Object{Type: rbac.RewardBundles, ID: bundle.ID}, After: json.RawMessage(created),
			ConfigVersion: version, Reason: note.Reason, Ref: note.Ref,
		}}, nil
	})
	return version, err
}

func bundleContentPublished(ctx context.Context, tx pgx.Tx, bundle reward.Bundle) error {
	items, err := cosmetics.Published(ctx, tx)
	if err != nil {
		return err
	}
	medals, err := publishedMedals(ctx, tx)
	if err != nil {
		return err
	}
	for _, id := range bundle.Cosmetics {
		if _, ok := items.Item(id); !ok {
			return fmt.Errorf("%w: %s grants the unpublished item %s", reward.ErrInvalidBundle, bundle.ID, id)
		}
	}
	for _, id := range bundle.Medals {
		if _, ok := medals.Medal(id); !ok {
			return fmt.Errorf("%w: %s grants the unpublished medal %s", reward.ErrInvalidBundle, bundle.ID, id)
		}
	}
	return nil
}
