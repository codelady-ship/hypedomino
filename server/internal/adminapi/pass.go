package adminapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

var errSeasonRemoved = errors.New("adminapi: a pass season that has started stays in the config")

type passRulesView struct {
	ServerTime time.Time `json:"server_time"`
}

type passConfigView struct {
	catalogView
	Rules passRulesView `json:"rules"`
}

func (h *Handler) passConfig(w http.ResponseWriter, r *http.Request, p principal) {
	version, raw, err := latestSnapshot(r.Context(), h.pool, pass.Domain)
	if err != nil {
		h.refuse(w, err)
		return
	}
	reply(w, http.StatusOK, passConfigView{catalogView: viewCatalog(version, raw), Rules: passRulesView{ServerTime: h.now().UTC()}})
}

func (h *Handler) passDomain(rollback, noted bool) configDomain {
	return configDomain{name: pass.Domain, object: rbac.PassSeasons,
		prepare: func(ctx context.Context, tx pgx.Tx, before, after []byte) ([]byte, error) {
			kept, err := keepStartedSeasons(before, after, h.now(), rollback)
			if err != nil {
				return nil, err
			}
			next, err := pass.Parse(kept)
			if err != nil {
				return nil, err
			}
			items, err := cosmetics.Published(ctx, tx)
			if err != nil {
				return nil, err
			}
			if err := h.frozenSeasonsKept(before, next, items, noted); err != nil {
				return nil, err
			}
			return kept, trackSourcesIn(items, next)
		},
		accept: func(_, after []byte) error {
			_, err := pass.Parse(after)
			return err
		},
	}
}

func (h *Handler) frozenSeasonsKept(before []byte, next *pass.Snapshot, items *cosmetics.Catalog, noted bool) error {
	if before == nil {
		return nil
	}
	published, err := pass.Parse(before)
	if err != nil {
		return err
	}
	replaced, err := pass.CheckFrozen(published, next, h.now(), disabledItemReplaces(items))
	if err == nil && replaced && !noted {
		return errReasonRequired
	}
	return err
}

func disabledItemReplaces(items *cosmetics.Catalog) pass.Replaces {
	return func(disabled, replacement string) bool {
		old, known := items.Item(disabled)
		next, found := items.Item(replacement)
		return known && found && old.Availability == cosmetics.Disabled && next.Availability == cosmetics.Live && next.Type == old.Type && next.Rarity == old.Rarity
	}
}

func keepStartedSeasons(before, after []byte, now time.Time, rollback bool) ([]byte, error) {
	if before == nil {
		return after, nil
	}
	next, current := pass.Default(), pass.Default()
	if err := json.Unmarshal(after, next); err != nil {
		return nil, fmt.Errorf("%w: %v", pass.ErrInvalidConfig, err)
	}
	if err := json.Unmarshal(before, current); err != nil {
		return nil, err
	}
	changed := false
	for _, season := range current.Seasons {
		if _, listed := next.Season(season.ID); listed || now.Before(season.Start) {
			continue
		}
		if !rollback {
			return nil, fmt.Errorf("%w: %s", errSeasonRemoved, season.ID)
		}
		next.Seasons = append(next.Seasons, season)
		changed = true
	}
	if !changed {
		return after, nil
	}
	return json.Marshal(next)
}

func trackSourcesIn(items *cosmetics.Catalog, config *pass.Snapshot) error {
	grantable := func(id string, source cosmetics.Source) bool {
		item, ok := items.Item(id)
		return ok && item.Grantable(source)
	}
	mythic := func(id string) bool {
		item, ok := items.Item(id)
		return ok && item.Rarity == mythicRarity
	}
	return config.CheckSources(grantable, mythic)
}
