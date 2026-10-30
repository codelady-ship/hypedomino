package rating_test

import (
	"errors"
	"math"
	"math/rand"
	"reflect"
	"sort"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/rating"
)

func rank(t *testing.T, name string, lp int) rating.Rank {
	t.Helper()
	index, err := rating.ParseRankIndex(name)
	if err != nil {
		t.Fatal(err)
	}
	return rating.Rank{Index: index, LP: lp}
}

func TestRanksParsePrintAndMove(t *testing.T) {
	c := rating.DefaultConfig()
	for name, index := range map[string]int{"Iron IV": 0, "iron i": 3, "Silver I": 11, "Gold II": 14, "Diamond I": 27, "Master": 28} {
		got, err := rating.ParseRankIndex(name)
		if err != nil || got != index {
			t.Errorf("ParseRankIndex(%q) = %d, %v", name, got, err)
		}
	}
	for _, name := range []string{"", "Gold", "Gold V", "Wood IV", "Grandmaster"} {
		if _, err := rating.ParseRankIndex(name); !errors.Is(err, rating.ErrRank) {
			t.Errorf("ParseRankIndex(%q) = %v", name, err)
		}
	}
	if (rating.Rank{Index: 14}).String() != "Gold II" || (rating.Rank{Index: 28}).String() != "Master" || (rating.Rank{Index: 28}).Division() != "" {
		t.Error("rank names")
	}
	for _, m := range []struct {
		from  string
		lp    int
		delta int
		to    string
		toLP  int
	}{
		{"Gold I", 90, 21, "Platinum IV", 11},
		{"Gold III", 10, -18, "Gold IV", 75},
		{"Iron IV", 10, -18, "Iron IV", 0},
		{"Silver I", 40, 21, "Silver I", 61},
		{"Diamond I", 95, 40, "Master", 35},
		{"Master", 10, -20, "Diamond I", 75},
		{"Master", 10, 40, "Master", 50},
		{"Master", 300, 40, "Master", 340},
	} {
		got := c.Move(rank(t, m.from, m.lp), m.delta)
		if got.String() != m.to || got.LP != m.toLP {
			t.Errorf("%s %d LP %+d = %s %d LP, want %s %d LP", m.from, m.lp, m.delta, got, got.LP, m.to, m.toLP)
		}
	}
	if c.RankMmr(rank(t, "Silver I", 40)) != 1256 || c.MmrRankIndex(1412) != 15 || c.MmrRankIndex(100) != 0 || c.MmrRankIndex(5000) != 27 {
		t.Error("rank mmr mapping")
	}
}

func TestTheWorkedExamplesOfTheSpec(t *testing.T) {
	c := rating.DefaultConfig()
	a := rating.Player{Account: "a", MMR: 1300, Matches: 30, Rank: rank(t, "Silver I", 40)}
	b := rating.Player{Account: "b", MMR: 1250, Matches: 30, Rank: rank(t, "Silver I", 0)}
	changes, err := c.Resolve(rating.Match{Teams: [2][]rating.Player{{a}, {b}}, Winner: 0})
	if err != nil {
		t.Fatal(err)
	}
	win := changes[0][0]
	if math.Abs(win.Expected-0.5715) > 0.0001 || win.MMR != 1309 || win.LPDelta != 21 || win.Rank.String() != "Silver I" || win.Rank.LP != 61 || win.K != 20 {
		t.Errorf("A wins: %+v", win)
	}
	changes, _ = c.Resolve(rating.Match{Teams: [2][]rating.Player{{a}, {b}}, Winner: 1})
	loss := changes[0][0]
	if loss.MMR != 1289 || loss.LPDelta != -18 || loss.Rank.LP != 22 || loss.Won {
		t.Errorf("A loses: %+v", loss)
	}
	p1 := rating.Player{Account: "p1", MMR: 1400, Matches: 30, Rank: rank(t, "Gold II", 50)}
	p2 := rating.Player{Account: "p2", MMR: 1200, Matches: 3, Rank: rank(t, "Silver IV", 20)}
	o1 := rating.Player{Account: "o1", MMR: 1350, Matches: 30, Rank: rank(t, "Gold IV", 0)}
	o2 := rating.Player{Account: "o2", MMR: 1250, Matches: 30, Rank: rank(t, "Silver II", 0)}
	changes, _ = c.Resolve(rating.Match{Teams: [2][]rating.Player{{p1, p2}, {o1, o2}}, Winner: 0})
	if changes[0][0].Expected != 0.5 || changes[0][0].MMR != 1410 || changes[0][0].LPDelta != 22 || changes[0][1].MMR != 1220 || changes[0][1].LPDelta != 27 || changes[0][1].K != 40 {
		t.Errorf("team wins: %+v %+v", changes[0][0], changes[0][1])
	}
	changes, _ = c.Resolve(rating.Match{Teams: [2][]rating.Player{{p1, p2}, {o1, o2}}, Winner: 1})
	if changes[0][0].MMR != 1390 || changes[0][0].LPDelta != -18 || changes[0][1].MMR != 1180 || changes[0][1].LPDelta != -13 {
		t.Errorf("team loses: %+v %+v", changes[0][0], changes[0][1])
	}
	if _, err := c.Resolve(rating.Match{Teams: [2][]rating.Player{{p1}, {o1, o2}}, Winner: 0}); !errors.Is(err, rating.ErrMatch) {
		t.Errorf("uneven teams: %v", err)
	}
	if _, err := c.Resolve(rating.Match{Teams: [2][]rating.Player{{p1}, {o1}}, Winner: 2}); !errors.Is(err, rating.ErrMatch) {
		t.Errorf("bad winner: %v", err)
	}
}

func TestOffendersLoseAndPartnersLoseLess(t *testing.T) {
	c := rating.DefaultConfig()
	p1 := rating.Player{Account: "p1", MMR: 1400, Matches: 30, Rank: rank(t, "Gold II", 50), Offender: true}
	p2 := rating.Player{Account: "p2", MMR: 1200, Matches: 3, Rank: rank(t, "Silver IV", 20)}
	o1 := rating.Player{Account: "o1", MMR: 1350, Matches: 30, Rank: rank(t, "Gold IV", 0)}
	o2 := rating.Player{Account: "o2", MMR: 1250, Matches: 30, Rank: rank(t, "Silver II", 0)}
	changes, _ := c.Resolve(rating.Match{Teams: [2][]rating.Player{{p1, p2}, {o1, o2}}, Winner: 0})
	if changes[0][0].Won || changes[0][0].MMR != 1390 || changes[0][0].LPDelta != -28 {
		t.Errorf("winning offender: %+v", changes[0][0])
	}
	if !changes[0][1].Won || changes[0][1].MMR != 1220 || changes[0][1].LPDelta != 27 {
		t.Errorf("partner of a winning offender: %+v", changes[0][1])
	}
	changes, _ = c.Resolve(rating.Match{Teams: [2][]rating.Player{{p1, p2}, {o1, o2}}, Winner: 1})
	if changes[0][0].MMR != 1390 || changes[0][0].LPDelta != -28 {
		t.Errorf("losing offender: %+v", changes[0][0])
	}
	if changes[0][1].MMR != 1190 || changes[0][1].LPDelta != -7 {
		t.Errorf("partner of a losing offender: %+v", changes[0][1])
	}
	if changes[1][0].MMR != 1360 || changes[1][0].LPDelta <= 0 {
		t.Errorf("opponent: %+v", changes[1][0])
	}
	placing := rating.Player{Account: "n", MMR: 1200, Placement: true, Offender: true}
	changes, _ = c.Resolve(rating.Match{Teams: [2][]rating.Player{{placing}, {o1}}, Winner: 0})
	if changes[0][0].K != 40 || changes[0][0].LPDelta != 0 || changes[0][0].Penalty != 10 || changes[0][0].Rank != placing.Rank || changes[0][0].MMR >= 1200 {
		t.Errorf("placement offender: %+v", changes[0][0])
	}
	if c.Place(1412, 0).String() != "Gold IV" || c.Place(1100, 0).String() != "Bronze I" || c.Place(1100, 10).LP != 0 || c.Place(500, 0).Index != 0 {
		t.Error("placement ranks")
	}
}

func TestConfigValidationAndParsing(t *testing.T) {
	if err := rating.DefaultConfig().Validate(); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*rating.Config){
		"division":  func(c *rating.Config) { c.DivisionLp = 0 },
		"demotion":  func(c *rating.Config) { c.DemotionLp = 100 },
		"scale":     func(c *rating.Config) { c.MmrScale = 0 },
		"step":      func(c *rating.Config) { c.RankMmrStep = 0 },
		"factors":   func(c *rating.Config) { c.LpFactorMin = 2 },
		"changes":   func(c *rating.Config) { c.LpChangeMin = 50 },
		"one move":  func(c *rating.Config) { c.LpChangeMax = 90 },
		"percent":   func(c *rating.Config) { c.PartnerLossReductionPercent = 101 },
		"placement": func(c *rating.Config) { c.PlacementMatches = 0 },
		"cap":       func(c *rating.Config) { c.NewPlayerPlacementCap = "Master" },
		"hour":      func(c *rating.Config) { c.ApexRecalcHourUtc = 24 },
		"slots":     func(c *rating.Config) { c.ChallengerSlots = -1 },
	} {
		c := rating.DefaultConfig()
		mutate(&c)
		if err := c.Validate(); !errors.Is(err, rating.ErrInvalidConfig) {
			t.Errorf("%s: %v", name, err)
		}
	}
	snapshot, err := rating.Parse([]byte(`{"schema":1,"modes":{"2v2":{"lpBase":25}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(snapshot.Modes.OneVsOne, rating.DefaultConfig()) || snapshot.Modes.TwoVsTwo.LpBase != 25 || snapshot.Modes.TwoVsTwo.LpChangeMax != 40 {
		t.Errorf("overlay %+v", snapshot.Modes)
	}
	for _, raw := range []string{`{`, `{"schema":2}`, `{"schema":1,"modes":{"1v1":{"divisionLp":0}}}`} {
		if _, err := rating.Parse([]byte(raw)); !errors.Is(err, rating.ErrInvalidConfig) {
			t.Errorf("%s: %v", raw, err)
		}
	}
}

type simulated struct {
	skill  float64
	player rating.Player
}

func TestASimulatedLadderMatchesTheExpectedCurve(t *testing.T) {
	c := rating.DefaultConfig()
	random := rand.New(rand.NewSource(7))
	const players, rounds = 1000, 80
	ladder := make([]*simulated, players)
	for i := range ladder {
		ladder[i] = &simulated{skill: 1200 + 300*random.NormFloat64(), player: rating.Player{Account: "p", MMR: c.InitialMmr, Placement: true}}
	}
	played := make([]int, players)
	for round := 0; round < rounds; round++ {
		order := random.Perm(players)
		sort.Slice(order, func(i, j int) bool { return ladder[order[i]].player.MMR < ladder[order[j]].player.MMR })
		for i := 0; i+1 < len(order); i += 2 {
			a, b := ladder[order[i]], ladder[order[i+1]]
			winner := 1
			if random.Float64() < rating.Expected(a.skill, b.skill, c.MmrScale) {
				winner = 0
			}
			changes, err := c.Resolve(rating.Match{Teams: [2][]rating.Player{{a.player}, {b.player}}, Winner: winner})
			if err != nil {
				t.Fatal(err)
			}
			for side, s := range []*simulated{a, b} {
				change := changes[side][0]
				s.player.MMR, s.player.Matches, s.player.Rank = change.MMR, s.player.Matches+1, change.Rank
				played[order[i+side]]++
				if s.player.Placement && s.player.Matches >= c.PlacementMatches {
					s.player.Placement = false
					s.player.Rank = c.Place(s.player.MMR, 0)
				}
			}
		}
	}
	tierShare := map[string]float64{}
	var skillMean, rankMean, cov, skillVar, rankVar float64
	for _, s := range ladder {
		tierShare[s.player.Rank.Tier()] += 1 / float64(players)
		skillMean += s.skill / players
		rankMean += float64(s.player.Rank.Index) / players
	}
	for _, s := range ladder {
		ds, dr := s.skill-skillMean, float64(s.player.Rank.Index)-rankMean
		cov += ds * dr
		skillVar += ds * ds
		rankVar += dr * dr
	}
	correlation := cov / math.Sqrt(skillVar*rankVar)
	t.Logf("tiers %v correlation %.3f", tierShare, correlation)
	if correlation < 0.85 {
		t.Errorf("rank follows skill weakly: %.3f", correlation)
	}
	middle := tierShare["Silver"] + tierShare["Gold"] + tierShare["Platinum"]
	if middle < 0.5 || tierShare["Iron"] > 0.15 || tierShare["Diamond"]+tierShare[rating.TierMaster] > 0.12 {
		t.Errorf("distribution is not a bell: %v", tierShare)
	}
	for _, s := range ladder {
		if s.player.Rank.LP < 0 || (!s.player.Rank.Apex() && s.player.Rank.LP >= c.DivisionLp) {
			t.Fatalf("rank out of range: %+v", s.player.Rank)
		}
	}
}

func TestTheResetTableFollowsTheSpecAndItsOverrides(t *testing.T) {
	c := rating.DefaultConfig()
	for old, want := range map[string]string{"Iron II": "Iron II", "Bronze IV": "Bronze IV", "Bronze III": "Bronze IV", "Silver IV": "Bronze IV", "Silver III": "Bronze III", "Gold II": "Silver II", "Diamond I": "Emerald I", "Master": "Diamond IV"} {
		if got := (rating.Rank{Index: c.ResetIndex(rank(t, old, 0).Index)}).String(); got != want {
			t.Errorf("%s resets to %s, want %s", old, got, want)
		}
	}
	c.SeasonResetTable = map[string]string{"Gold II": "Silver III", "Master": "Diamond I"}
	if c.ResetIndex(14) != 9 || c.ResetIndex(28) != 27 {
		t.Errorf("overrides ignored: %d %d", c.ResetIndex(14), c.ResetIndex(28))
	}
	for name, table := range map[string]map[string]string{
		"above itself":  {"Silver IV": "Gold IV"},
		"not monotone":  {"Gold II": "Iron IV"},
		"into the apex": {"Master": "Master"},
		"bad rank":      {"Wood IV": "Iron IV"},
	} {
		c.SeasonResetTable = table
		if err := c.Validate(); !errors.Is(err, rating.ErrInvalidConfig) {
			t.Errorf("%s: %v", name, err)
		}
	}
	c.SeasonResetTable = nil
	c.ResetFloor, c.ResetCeiling = "Diamond IV", "Bronze IV"
	if err := c.Validate(); !errors.Is(err, rating.ErrInvalidConfig) {
		t.Errorf("floor above ceiling: %v", err)
	}
}
