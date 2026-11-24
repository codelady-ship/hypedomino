package adminapi

import (
	"cmp"
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
)

const (
	CalendarRankedSeason = "ranked_season"
	CalendarPassSeason   = "pass_season"
	CalendarShopOffer    = "shop_offer"
	CalendarPromo        = "promo_campaign"
	CalendarRotation     = "shop_rotation"
)

type calendarEntry struct {
	Kind  string     `json:"kind"`
	ID    string     `json:"id"`
	Start time.Time  `json:"start"`
	End   *time.Time `json:"end"`
}

type calendarView struct {
	ServerTime time.Time       `json:"server_time"`
	Entries    []calendarEntry `json:"entries"`
}

type calendarSource func(ctx context.Context) ([]calendarEntry, error)

func (h *Handler) calendar(w http.ResponseWriter, r *http.Request, p principal) {
	sources := []calendarSource{h.rankedEntries, h.passEntries, h.shopEntries}
	if h.allows(p, rbac.PromoCodes, rbac.View) == nil {
		sources = append(sources, h.promoEntries)
	}
	now := h.now().UTC()
	view := calendarView{ServerTime: now, Entries: []calendarEntry{}}
	for _, source := range sources {
		entries, err := source(r.Context())
		if err != nil {
			h.refuse(w, err)
			return
		}
		for _, entry := range entries {
			if entry.End == nil || entry.End.After(now) {
				view.Entries = append(view.Entries, entry)
			}
		}
	}
	slices.SortFunc(view.Entries, func(a, b calendarEntry) int {
		return cmp.Or(a.Start.Compare(b.Start), cmp.Compare(a.Kind, b.Kind), cmp.Compare(a.ID, b.ID))
	})
	reply(w, http.StatusOK, view)
}

func window(kind, id string, start, end time.Time) calendarEntry {
	entry := calendarEntry{Kind: kind, ID: id, Start: start.UTC()}
	if !end.IsZero() {
		end = end.UTC()
		entry.End = &end
	}
	return entry
}

func (h *Handler) rankedEntries(ctx context.Context) ([]calendarEntry, error) {
	seasons, err := rating.ListSeasons(ctx, h.pool)
	entries := make([]calendarEntry, 0, len(seasons))
	for _, season := range seasons {
		entries = append(entries, window(CalendarRankedSeason, season.ID, season.Start, season.End))
	}
	return entries, err
}

func (h *Handler) passEntries(ctx context.Context) ([]calendarEntry, error) {
	var published struct {
		Seasons []struct {
			ID    string    `json:"id"`
			Start time.Time `json:"start"`
			End   time.Time `json:"end"`
		} `json:"seasons"`
	}
	if err := h.publishedInto(ctx, pass.Domain, &published); err != nil {
		return nil, err
	}
	entries := make([]calendarEntry, 0, len(published.Seasons))
	for _, season := range published.Seasons {
		entries = append(entries, window(CalendarPassSeason, season.ID, season.Start, season.End))
	}
	return entries, nil
}

func (h *Handler) shopEntries(ctx context.Context) ([]calendarEntry, error) {
	var published struct {
		Offers []struct {
			ID      string      `json:"id"`
			Section string      `json:"section"`
			Window  shop.Window `json:"window"`
		} `json:"offers"`
		Rotations []shop.Rotation `json:"rotations"`
	}
	if err := h.publishedInto(ctx, shop.Domain, &published); err != nil {
		return nil, err
	}
	var entries []calendarEntry
	for _, offer := range published.Offers {
		if offer.Section == shop.SectionFeatured || offer.Section == shop.SectionEvent {
			entries = append(entries, window(CalendarShopOffer, offer.ID, offer.Window.Start, offer.Window.End))
		}
	}
	for _, rotation := range published.Rotations {
		start, end := h.config.Shop.Period(rotation.Period, rotation.Start)
		entries = append(entries, window(CalendarRotation, rotation.Period, start, end))
	}
	return entries, nil
}

func (h *Handler) promoEntries(ctx context.Context) ([]calendarEntry, error) {
	rows, err := h.pool.Query(ctx, `select campaign_id, min(starts_at), max(ends_at) from promo_codes where state <> 'disabled' group by campaign_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []calendarEntry
	for rows.Next() {
		var id string
		var start, end time.Time
		if err := rows.Scan(&id, &start, &end); err != nil {
			return nil, err
		}
		entries = append(entries, window(CalendarPromo, id, start, end))
	}
	return entries, rows.Err()
}

func (h *Handler) publishedInto(ctx context.Context, domain string, into any) error {
	_, raw, err := latestSnapshot(ctx, h.pool, domain)
	if err != nil || raw == nil {
		return err
	}
	return json.Unmarshal(raw, into)
}
