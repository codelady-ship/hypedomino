package adminapi

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/dailylogin"
	"github.com/Alievs-corp/hypedomino/server/internal/mission"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
)

const (
	bundleUsePromo         = "promo"
	bundleUsePromoTemplate = "promo_template"
	firstPurchaseObject    = "first_purchase"
)

type bundleUse struct {
	Domain string `json:"domain"`
	Object string `json:"object"`
	Detail string `json:"detail,omitempty"`
}

type bundleRow struct {
	reward.Bundle
	GuideValue int64       `json:"guideValue"`
	UsedBy     []bundleUse `json:"usedBy"`
	Differs    bool        `json:"differs"`
	Editable   bool        `json:"editable"`
}

type rewardBundlesView struct {
	Bundles []bundleRow `json:"bundles"`
}

type bundleListQuery struct {
	Text      string
	Domain    string
	Differing bool
	After     string
	Limit     int
}

type bundleIndex map[string]*bundleRow

func (i bundleIndex) add(bundle reward.Bundle, use bundleUse) error {
	if bundle.ID == "" {
		return nil
	}
	content, err := bundleContent(bundle)
	if err != nil {
		return err
	}
	row, known := i[content]
	if !known {
		row = &bundleRow{Bundle: bundle}
		i[content] = row
	}
	row.Revision = max(row.Revision, bundle.Revision)
	if !slices.Contains(row.UsedBy, use) {
		row.UsedBy = append(row.UsedBy, use)
	}
	return nil
}

func (i bundleIndex) rows(items *cosmetics.Catalog, prices reward.Config) []bundleRow {
	holders := map[string]int{}
	for _, row := range i {
		holders[row.ID]++
	}
	rows := make([]bundleRow, 0, len(i))
	for _, row := range i {
		row.GuideValue = guideValue(row.Bundle, items, prices)
		row.Differs = holders[row.ID] > 1
		row.Editable = !row.Differs && !slices.ContainsFunc(row.UsedBy, func(use bundleUse) bool { return !bundleEditableIn(use.Domain) })
		rows = append(rows, *row)
	}
	slices.SortFunc(rows, func(a, b bundleRow) int {
		return cmp.Or(cmp.Compare(a.ID, b.ID), cmp.Compare(a.UsedBy[0].Object, b.UsedBy[0].Object))
	})
	return rows
}

func bundleContent(bundle reward.Bundle) (string, error) {
	bundle.Revision = 0
	bundle.Cosmetics = slices.Sorted(slices.Values(bundle.Cosmetics))
	bundle.Medals = slices.Sorted(slices.Values(bundle.Medals))
	raw, err := json.Marshal(bundle)
	return string(raw), err
}

func guideValue(bundle reward.Bundle, items *cosmetics.Catalog, prices reward.Config) int64 {
	value := bundle.Coins
	for _, id := range bundle.Cosmetics {
		if item, ok := items.Item(id); ok {
			value += prices.GuidePrice(item)
		}
	}
	return value
}

func (h *Handler) listRewardBundles(w http.ResponseWriter, r *http.Request, _ principal) {
	query, err := rewardBundleQuery(r)
	if err != nil {
		h.refuse(w, err)
		return
	}
	view, err := h.rewardBundles(r.Context())
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, rewardBundlesView{Bundles: query.page(view.Bundles)})
}

func rewardBundleQuery(r *http.Request) (bundleListQuery, error) {
	query := r.URL.Query()
	limit, err := queuePageLimit(query.Get("limit"))
	differing := query.Get("differing")
	if err != nil || (differing != "" && differing != "true") {
		return bundleListQuery{}, errMalformed
	}
	text := strings.ToLower(strings.TrimSpace(query.Get("q")))
	return bundleListQuery{Text: text, Domain: query.Get("domain"), Differing: differing == "true", After: query.Get("after"), Limit: limit}, nil
}

func (q bundleListQuery) page(rows []bundleRow) []bundleRow {
	page := []bundleRow{}
	for _, row := range rows {
		if row.ID <= q.After || !q.keeps(row) {
			continue
		}
		if len(page) >= q.Limit && row.ID != page[len(page)-1].ID {
			break
		}
		page = append(page, row)
	}
	return page
}

func (q bundleListQuery) keeps(row bundleRow) bool {
	usedInDomain := slices.ContainsFunc(row.UsedBy, func(use bundleUse) bool { return use.Domain == q.Domain })
	matches := slices.ContainsFunc(row.searchable(), func(value string) bool { return strings.Contains(strings.ToLower(value), q.Text) })
	return (!q.Differing || row.Differs) && (q.Domain == "" || usedInDomain) && matches
}

func (r bundleRow) searchable() []string {
	values := append([]string{r.ID}, r.Cosmetics...)
	values = append(values, r.Medals...)
	for _, use := range r.UsedBy {
		values = append(values, use.Object)
	}
	return values
}

func (h *Handler) rewardBundles(ctx context.Context) (rewardBundlesView, error) {
	index := bundleIndex{}
	collectors := []func(context.Context, bundleIndex) error{h.shopBundles, h.passBundles, h.missionBundles, h.ratingBundles, h.dailyLoginBundles, h.libraryBundles}
	for _, collect := range collectors {
		if err := collect(ctx, index); err != nil {
			return rewardBundlesView{}, err
		}
	}
	if err := h.promoBundles(ctx, index); err != nil {
		return rewardBundlesView{}, err
	}
	items, err := cosmetics.Published(ctx, h.pool)
	if err != nil {
		return rewardBundlesView{}, err
	}
	return rewardBundlesView{Bundles: index.rows(items, reward.DefaultConfig())}, nil
}

func (h *Handler) shopBundles(ctx context.Context, index bundleIndex) error {
	_, raw, err := latestSnapshot(ctx, h.pool, shop.Domain)
	if err != nil || raw == nil {
		return err
	}
	catalog, err := shop.Parse(raw, h.config.Shop)
	if err != nil {
		return err
	}
	for _, offer := range catalog.Offers {
		if err := index.add(offer.Bundle, bundleUse{Domain: shop.Domain, Object: offer.ID}); err != nil {
			return err
		}
	}
	if catalog.FirstPurchase == nil {
		return nil
	}
	return index.add(*catalog.FirstPurchase, bundleUse{Domain: shop.Domain, Object: firstPurchaseObject})
}

func (h *Handler) passBundles(ctx context.Context, index bundleIndex) error {
	_, raw, err := latestSnapshot(ctx, h.pool, pass.Domain)
	if err != nil || raw == nil {
		return err
	}
	config, err := pass.Parse(raw)
	if err != nil {
		return err
	}
	for _, season := range config.Seasons {
		for _, track := range []string{pass.Free, pass.Premium} {
			levels := season.Track(track)
			for _, level := range slices.Sorted(maps.Keys(levels)) {
				use := bundleUse{Domain: pass.Domain, Object: season.ID, Detail: fmt.Sprintf("%s.%d", track, level)}
				if err := index.add(levels[level], use); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (h *Handler) missionBundles(ctx context.Context, index bundleIndex) error {
	_, raw, err := latestSnapshot(ctx, h.pool, mission.Domain)
	if err != nil {
		return err
	}
	catalog := mission.Default()
	if raw != nil {
		if catalog, err = mission.Parse(raw); err != nil {
			return err
		}
	}
	for _, listed := range catalog.Missions {
		if listed.Bundle == nil {
			continue
		}
		if err := index.add(*listed.Bundle, bundleUse{Domain: mission.Domain, Object: listed.ID}); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) ratingBundles(ctx context.Context, index bundleIndex) error {
	version, raw, err := latestSnapshot(ctx, h.pool, rating.Domain)
	if err != nil {
		return err
	}
	snapshot := rating.Default()
	if raw != nil {
		if snapshot, err = rating.Parse(raw); err != nil {
			return err
		}
	}
	if err := addRankBundles(index, snapshot, snapshot.Modes.OneVsOne, "1v1", version); err != nil {
		return err
	}
	return addRankBundles(index, snapshot, snapshot.Modes.TwoVsTwo, "2v2", version)
}

func addRankBundles(index bundleIndex, snapshot *rating.Snapshot, config rating.Config, mode string, version int64) error {
	tiers := slices.Collect(maps.Keys(snapshot.RankRewardCoins))
	tiers = slices.AppendSeq(tiers, maps.Keys(config.RankRewards))
	slices.Sort(tiers)
	for _, tier := range slices.Compact(tiers) {
		bundle, grants := rating.RankBundle(config, snapshot, tier, version)
		if !grants {
			continue
		}
		if err := index.add(bundle, bundleUse{Domain: rating.Domain, Object: mode, Detail: tier}); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) dailyLoginBundles(ctx context.Context, index bundleIndex) error {
	version, raw, err := latestSnapshot(ctx, h.pool, dailylogin.Domain)
	if err != nil || raw == nil {
		return err
	}
	calendar, err := dailylogin.Parse(raw)
	if err != nil {
		return err
	}
	for day := 1; day <= len(calendar.Days); day++ {
		use := bundleUse{Domain: dailylogin.Domain, Object: fmt.Sprintf("day.%d", day)}
		if err := index.add(calendar.Reward(day, version), use); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) promoBundles(ctx context.Context, index bundleIndex) error {
	if h.promos == nil {
		return nil
	}
	held, err := h.promos.CampaignBundles(ctx)
	if err != nil {
		return err
	}
	for _, code := range held {
		if err := index.add(code.Bundle, bundleUse{Domain: bundleUsePromo, Object: code.Campaign}); err != nil {
			return err
		}
	}
	templates, err := h.promos.Templates(ctx)
	if err != nil {
		return err
	}
	for _, template := range templates {
		if template.Bundle == nil {
			continue
		}
		if err := index.add(*template.Bundle, bundleUse{Domain: bundleUsePromoTemplate, Object: template.ID}); err != nil {
			return err
		}
	}
	return nil
}
