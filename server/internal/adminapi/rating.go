package adminapi

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/medal"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

type ratingRulesView struct {
	ServerTime     time.Time        `json:"server_time"`
	OpenSeason     string           `json:"open_season"`
	StructuralKeys []string         `json:"structural_keys"`
	Tiers          []string         `json:"tiers"`
	DivisionRanks  []string         `json:"division_ranks"`
	Defaults       *rating.Snapshot `json:"defaults"`
}

type ratingConfigView struct {
	catalogView
	Rules ratingRulesView `json:"rules"`
}

func (h *Handler) ratingConfig(w http.ResponseWriter, r *http.Request, p principal) {
	version, raw, err := latestSnapshot(r.Context(), h.pool, rating.Domain)
	if err != nil {
		h.refuse(w, err)
		return
	}
	now := h.now().UTC()
	open, err := rating.OpenSeason(r.Context(), h.pool, now)
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, ratingConfigView{catalogView: viewCatalog(version, raw), Rules: ratingRulesView{
		ServerTime:     now,
		OpenSeason:     open,
		StructuralKeys: rating.StructuralKeys(),
		Tiers:          rating.Tiers(),
		DivisionRanks:  rating.DivisionRanks(),
		Defaults:       rating.Default(),
	}})
}

func (h *Handler) ratingDomain() configDomain {
	return configDomain{name: rating.Domain, object: rbac.RankedSeasons,
		prepare: func(ctx context.Context, tx pgx.Tx, before, after []byte) ([]byte, error) {
			next, err := rating.Parse(after)
			if err != nil {
				return nil, err
			}
			offenses, err := publishedPenalties(ctx, tx)
			if err != nil {
				return nil, err
			}
			if err := next.CheckOffensePenalty(offenses.OffenseRatingPenalty); err != nil {
				return nil, err
			}
			published := rating.Default()
			if before != nil {
				if published, err = rating.Parse(before); err != nil {
					return nil, err
				}
			}
			open, err := rating.OpenSeason(ctx, tx, h.now())
			if err != nil {
				return nil, err
			}
			if open != "" {
				if err := rating.CheckFrozen(published, next); err != nil {
					return nil, err
				}
			}
			medals, err := publishedMedals(ctx, tx)
			if err != nil {
				return nil, err
			}
			return after, rankRewardSourcesListed(ctx, tx, next, medals)
		},
		accept: func(_, after []byte) error {
			_, err := rating.Parse(after)
			return err
		},
	}
}

func rankRewardSourcesListed(ctx context.Context, tx pgx.Tx, config *rating.Snapshot, medals *medal.Catalog) error {
	items, err := cosmetics.Published(ctx, tx)
	if err != nil {
		return err
	}
	return rankRewardSourcesIn(items, config, medals)
}

func rankRewardSourcesIn(items *cosmetics.Catalog, config *rating.Snapshot, medals *medal.Catalog) error {
	medalLists := func(id string, source cosmetics.Source) bool {
		found, ok := medals.Medal(id)
		return ok && found.Lists(source)
	}
	itemLists := func(id string, source cosmetics.Source) bool {
		found, ok := items.Item(id)
		return ok && found.Grantable(source)
	}
	return config.CheckSources(medalLists, itemLists)
}

func publishedRating(ctx context.Context, tx pgx.Tx) (*rating.Snapshot, error) {
	_, raw, err := latestSnapshot(ctx, tx, rating.Domain)
	if err != nil || raw == nil {
		return nil, err
	}
	return rating.Parse(raw)
}
