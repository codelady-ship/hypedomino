package rating_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
)

func TestStructuralKeysAreFrozenAndCoefficientsAreNot(t *testing.T) {
	published := rating.Default()
	next := rating.Default()
	next.Modes.OneVsOne.MmrK, next.Modes.TwoVsTwo.LpBase, next.LeaderboardTopSize = 30, 25, 300
	if err := rating.CheckFrozen(published, next); err != nil {
		t.Fatalf("coefficients: %v", err)
	}
	same := rating.Default()
	same.Modes.OneVsOne.SeasonResetTable = map[string]string{"Gold I": "Silver I"}
	if err := rating.CheckFrozen(published, same); err != nil {
		t.Fatalf("a row equal to the generated one: %v", err)
	}
	for name, edit := range map[string]func(*rating.Snapshot){
		"divisionLp":       func(s *rating.Snapshot) { s.Modes.OneVsOne.DivisionLp = 120 },
		"placementMatches": func(s *rating.Snapshot) { s.Modes.TwoVsTwo.PlacementMatches = 6 },
		"challengerSlots":  func(s *rating.Snapshot) { s.Modes.OneVsOne.ChallengerSlots = 60 },
		"resetFloor":       func(s *rating.Snapshot) { s.Modes.TwoVsTwo.ResetFloor = "Silver IV" },
		"seasonResetTable": func(s *rating.Snapshot) { s.Modes.OneVsOne.SeasonResetTable = map[string]string{"Gold I": "Gold I"} },
	} {
		changed := rating.Default()
		edit(changed)
		err := rating.CheckFrozen(published, changed)
		if !errors.Is(err, rating.ErrFrozen) || !strings.Contains(err.Error(), name) {
			t.Fatalf("%s: %v", name, err)
		}
	}
}

func TestRankRewardItemsMustListTheirBundle(t *testing.T) {
	snapshot := rating.Default()
	snapshot.Modes.TwoVsTwo.RankRewards = map[string]reward.Bundle{"Gold": {ID: "bundle.rank_gold", Revision: 1, Medals: []string{"medal.gold"}, Cosmetics: []string{"frame.gold"}}}
	listed := map[string]cosmetics.Source{"medal.gold": {Type: "rank_reward", Ref: "bundle.rank_gold"}, "frame.gold": {Type: "rank_reward", Ref: "bundle.rank_gold"}}
	lists := func(id string, source cosmetics.Source) bool { return listed[id] == source }
	if err := snapshot.CheckSources(lists, lists); err != nil {
		t.Fatalf("listed: %v", err)
	}
	listed["frame.gold"] = cosmetics.Source{Type: "rank_reward", Ref: "bundle.rank_silver"}
	err := snapshot.CheckSources(lists, lists)
	if !errors.Is(err, rating.ErrInvalidConfig) || !strings.Contains(err.Error(), "2v2 Gold grants frame.gold") {
		t.Fatalf("unlisted: %v", err)
	}
	if got := rating.Tiers(); len(got) != 10 || got[0] != "Iron" || got[9] != "Challenger" {
		t.Fatalf("tiers %v", got)
	}
	if got := rating.DivisionRanks(); len(got) != 28 || got[0] != "Iron IV" || got[27] != "Diamond I" {
		t.Fatalf("ranks %v", got)
	}
}
