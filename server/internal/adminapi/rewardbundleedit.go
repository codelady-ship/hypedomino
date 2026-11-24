package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/mission"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
)

var (
	errBundleUnknown = errors.New("adminapi: no such reward bundle")
	errBundleDiffers = errors.New("adminapi: this id holds different content, make it one on the screens of its objects first")
	errBundleHeld    = errors.New("adminapi: this bundle is edited on the screen of its object")
	errBundleStale   = errors.New("adminapi: the bundle changed since it was read")
	errBundleSame    = errors.New("adminapi: the bundle has no change")
)

type editBundleRequest struct {
	Revision  int      `json:"revision"`
	Coins     int64    `json:"coins"`
	Crystals  int64    `json:"crystals"`
	Cosmetics []string `json:"cosmetics"`
	Medals    []string `json:"medals"`
	PassXP    int      `json:"passXp"`
	changeNote
}

type bundleEditView struct {
	Bundle   reward.Bundle    `json:"bundle"`
	Versions map[string]int64 `json:"versions"`
}

type bundleHolder struct {
	domain  func() configDomain
	replace func(raw []byte, next reward.Bundle) ([]byte, error)
}

func (h *Handler) bundleHolders() []bundleHolder {
	return []bundleHolder{
		{domain: func() configDomain { return h.missionDomain(false) }, replace: bundleInMissions},
		{domain: func() configDomain { return h.passDomain(false, true) }, replace: bundleInPass},
		{domain: func() configDomain { return h.catalogDomain(true) }, replace: h.bundleInShop},
		{domain: bundleLibraryDomainConfig, replace: bundleInLibrary},
	}
}

func bundleEditableIn(domain string) bool {
	return domain == mission.Domain || domain == pass.Domain || domain == shop.Domain || domain == bundleLibraryDomain
}

func bundleInMissions(raw []byte, next reward.Bundle) ([]byte, error) {
	catalog := mission.Default()
	if raw != nil {
		parsed, err := mission.Parse(raw)
		if err != nil {
			return nil, err
		}
		catalog = parsed
	}
	for i := range catalog.Missions {
		if bundle := catalog.Missions[i].Bundle; bundle != nil && bundle.ID == next.ID {
			catalog.Missions[i].Bundle = &next
		}
	}
	return json.Marshal(catalog)
}

func bundleInPass(raw []byte, next reward.Bundle) ([]byte, error) {
	config, err := pass.Parse(raw)
	if err != nil {
		return nil, err
	}
	for _, season := range config.Seasons {
		for _, levels := range []map[int]reward.Bundle{season.Free, season.Premium} {
			for level, bundle := range levels {
				if bundle.ID == next.ID {
					levels[level] = next
				}
			}
		}
	}
	return json.Marshal(config)
}

func (h *Handler) bundleInShop(raw []byte, next reward.Bundle) ([]byte, error) {
	catalog, err := shop.Parse(raw, h.config.Shop)
	if err != nil {
		return nil, err
	}
	for i := range catalog.Offers {
		if catalog.Offers[i].Bundle.ID == next.ID {
			catalog.Offers[i].Bundle = next
		}
	}
	if catalog.FirstPurchase != nil && catalog.FirstPurchase.ID == next.ID {
		catalog.FirstPurchase = &next
	}
	return json.Marshal(catalog)
}

func (h *Handler) editRewardBundle(w http.ResponseWriter, r *http.Request, p principal) {
	var request editBundleRequest
	if !decode(w, r, &request) {
		return
	}
	if err := h.noted(request.changeNote); err != nil {
		h.refuse(w, err)
		return
	}
	row, next, err := h.bundleToEdit(r.Context(), r.PathValue("id"), request)
	if err != nil {
		h.refuse(w, err)
		return
	}
	versions, err := h.publishRewardBundle(r.Context(), p, row, next, request.changeNote)
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, bundleEditView{Bundle: next, Versions: versions})
}

func (h *Handler) bundleToEdit(ctx context.Context, id string, request editBundleRequest) (bundleRow, reward.Bundle, error) {
	view, err := h.rewardBundles(ctx)
	if err != nil {
		return bundleRow{}, reward.Bundle{}, err
	}
	at := slices.IndexFunc(view.Bundles, func(row bundleRow) bool { return row.ID == id })
	if at < 0 {
		return bundleRow{}, reward.Bundle{}, errBundleUnknown
	}
	row := view.Bundles[at]
	if err := checkBundleEditable(row, request.Revision); err != nil {
		return bundleRow{}, reward.Bundle{}, err
	}
	next := reward.Bundle{ID: id, Revision: row.Revision + 1, Coins: request.Coins, Crystals: request.Crystals, Cosmetics: request.Cosmetics, Medals: request.Medals, PassXP: request.PassXP}
	before, err := bundleContent(row.Bundle)
	if err != nil {
		return bundleRow{}, reward.Bundle{}, err
	}
	after, err := bundleContent(next)
	if err != nil {
		return bundleRow{}, reward.Bundle{}, err
	}
	if before == after {
		return bundleRow{}, reward.Bundle{}, errBundleSame
	}
	return row, next, nil
}

func checkBundleEditable(row bundleRow, revision int) error {
	if row.Differs {
		return errBundleDiffers
	}
	var held []string
	for _, use := range row.UsedBy {
		if !bundleEditableIn(use.Domain) && !slices.Contains(held, use.Domain) {
			held = append(held, use.Domain)
		}
	}
	if len(held) > 0 {
		return fmt.Errorf("%w: %s", errBundleHeld, strings.Join(held, ", "))
	}
	if revision != row.Revision {
		return errBundleStale
	}
	return nil
}

func (h *Handler) publishRewardBundle(ctx context.Context, p principal, row bundleRow, next reward.Bundle, note changeNote) (map[string]int64, error) {
	versions := map[string]int64{}
	err := h.audited(ctx, p, func(tx pgx.Tx, _ time.Time) ([]audit.Entry, error) {
		entries, err := h.publishHolders(ctx, tx, p, row, next, note, versions)
		if err != nil {
			return nil, err
		}
		before, err := json.Marshal(row.Bundle)
		if err != nil {
			return nil, err
		}
		after, err := json.Marshal(next)
		if err != nil {
			return nil, err
		}
		return append(entries, audit.Entry{
			Action: "publish", Object: audit.Object{Type: rbac.RewardBundles, ID: next.ID}, Before: json.RawMessage(before), After: json.RawMessage(after),
			ConfigVersion: entries[len(entries)-1].ConfigVersion, Reason: note.Reason, Ref: note.Ref,
		}), nil
	})
	return versions, err
}

func (h *Handler) publishHolders(ctx context.Context, tx pgx.Tx, p principal, row bundleRow, next reward.Bundle, note changeNote, versions map[string]int64) ([]audit.Entry, error) {
	var entries []audit.Entry
	for _, holder := range h.bundleHolders() {
		domain := holder.domain()
		if !slices.ContainsFunc(row.UsedBy, func(use bundleUse) bool { return use.Domain == domain.name }) {
			continue
		}
		entry, err := h.publishHolder(ctx, tx, p, holder, domain, next, note)
		if err != nil {
			return nil, err
		}
		versions[domain.name] = entry.ConfigVersion
		entries = append(entries, entry)
	}
	return entries, nil
}

func (h *Handler) publishHolder(ctx context.Context, tx pgx.Tx, p principal, holder bundleHolder, domain configDomain, next reward.Bundle, note changeNote) (audit.Entry, error) {
	if err := h.allows(p, domain.object, rbac.Publish); err != nil {
		return audit.Entry{}, err
	}
	if err := lockDomain(ctx, tx, domain.name); err != nil {
		return audit.Entry{}, err
	}
	_, raw, err := latestSnapshot(ctx, tx, domain.name)
	if err != nil {
		return audit.Entry{}, err
	}
	replaced, err := holder.replace(raw, next)
	if err != nil {
		return audit.Entry{}, err
	}
	entry, _, err := publishIn(ctx, tx, domain, "publish", replaced, note)
	return entry, err
}
