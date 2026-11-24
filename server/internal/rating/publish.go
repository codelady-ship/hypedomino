package rating

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
)

var ErrFrozen = errors.New("rating: structural keys are frozen while a season runs")

func Tiers() []string {
	return append(slices.Clone(tiers[:]), TierMaster, TierGrandmaster, TierChallenger)
}

func DivisionRanks() []string {
	ranks := make([]string, 0, MaxDivisionIndex+1)
	for index := 0; index <= MaxDivisionIndex; index++ {
		ranks = append(ranks, Rank{Index: index}.String())
	}
	return ranks
}

func StructuralKeys() []string {
	return []string{"divisionLp", "demotionLp", "apexRecalcHourUtc", "challengerSlots", "challengerMinLp", "grandmasterSlots", "grandmasterMinLp", "placementMatches", "resetTierDrop", "resetFloor", "resetCeiling", "seasonResetTable"}
}

func (c Config) structure() []any {
	table, _ := c.ResetTable()
	return []any{c.DivisionLp, c.DemotionLp, c.ApexRecalcHourUtc, c.ChallengerSlots, c.ChallengerMinLp, c.GrandmasterSlots, c.GrandmasterMinLp, c.PlacementMatches, c.ResetTierDrop, c.ResetFloor, c.ResetCeiling, table}
}

func CheckFrozen(published, next *Snapshot) error {
	keys := StructuralKeys()
	for _, mode := range modes {
		before, _ := published.Mode(mode)
		after, _ := next.Mode(mode)
		was, is := before.structure(), after.structure()
		var changed []string
		for i := range was {
			if was[i] != is[i] {
				changed = append(changed, keys[i])
			}
		}
		if len(changed) > 0 {
			return fmt.Errorf("%w: %s %s", ErrFrozen, mode, strings.Join(changed, ", "))
		}
	}
	return nil
}

func (s *Snapshot) CheckSources(medalLists, itemLists func(id string, source cosmetics.Source) bool) error {
	for _, mode := range modes {
		config, _ := s.Mode(mode)
		for _, tier := range Tiers() {
			bundle, ok := config.RankRewards[tier]
			if !ok {
				continue
			}
			source := cosmetics.Source{Type: SourceRankReward, Ref: bundle.ID}
			for _, grant := range []struct {
				ids   []string
				lists func(string, cosmetics.Source) bool
			}{{bundle.Medals, medalLists}, {bundle.Cosmetics, itemLists}} {
				for _, id := range grant.ids {
					if !grant.lists(id, source) {
						return fmt.Errorf("%w: %s %s grants %s, whose sources do not list rank_reward %s", ErrInvalidConfig, mode, tier, id, bundle.ID)
					}
				}
			}
		}
	}
	return nil
}

func OpenSeason(ctx context.Context, q querier, at time.Time) (string, error) {
	var id string
	err := q.QueryRow(ctx, `select id from ranked_seasons where starts_at <= $1 and (closed_1v1_at is null or closed_2v2_at is null) order by starts_at limit 1`, at).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return id, err
}
