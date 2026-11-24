package pass

import (
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestMatchXpFollowsWhatCounts(t *testing.T) {
	config := Default()
	cases := []struct {
		name  string
		match Match
		xp    int
	}{
		{"ranked 1v1 win", Match{Mode: rules.Mode1v1, Kind: matchstore.KindRanked, Won: true}, 100},
		{"ranked 1v1 loss", Match{Mode: rules.Mode1v1, Kind: matchstore.KindRanked}, 60},
		{"casual 2v2 win", Match{Mode: rules.Mode2v2, Kind: matchstore.KindCasual, Won: true}, 140},
		{"casual 2v2 loss", Match{Mode: rules.Mode2v2, Kind: matchstore.KindCasual}, 80},
		{"casual with a bot seat", Match{Mode: rules.Mode1v1, Kind: matchstore.KindCasual, Won: true, BotSeat: true}, 50},
		{"ranked ignores the bot seat flag", Match{Mode: rules.Mode1v1, Kind: matchstore.KindRanked, Won: true, BotSeat: true}, 100},
		{"custom room", Match{Mode: rules.Mode1v1, Kind: matchstore.KindCustom, Won: true}, 0},
		{"cancelled", Match{Mode: rules.Mode1v1, Kind: matchstore.KindRanked, Won: true, Cancelled: true}, 0},
		{"offender", Match{Mode: rules.Mode2v2, Kind: matchstore.KindRanked, Offender: true}, 0},
	}
	for _, c := range cases {
		if got := config.MatchXp(c.match); got != c.xp {
			t.Errorf("%s: %d, want %d", c.name, got, c.xp)
		}
	}
}

func TestBotSeatPercentRoundsHalvesAwayFromZero(t *testing.T) {
	cases := []struct{ value, percent, want int }{
		{60, 50, 30},
		{75, 50, 38},
		{25, 10, 3},
		{24, 10, 2},
		{100, 0, 0},
		{100, 100, 100},
	}
	for _, c := range cases {
		if got := percentHalfAwayFromZero(c.value, c.percent); got != c.want {
			t.Errorf("%d%% of %d = %d, want %d", c.percent, c.value, got, c.want)
		}
	}
}

func TestTheDailyCapCutsTheMatchThatCrossesIt(t *testing.T) {
	cases := []struct{ xp, earned, limit, want int }{
		{100, 0, 2000, 100},
		{100, 1950, 2000, 50},
		{100, 2000, 2000, 0},
		{100, 2100, 2000, 0},
	}
	for _, c := range cases {
		if got := capToRemainder(c.xp, c.earned, c.limit); got != c.want {
			t.Errorf("cap(%d, %d, %d) = %d, want %d", c.xp, c.earned, c.limit, got, c.want)
		}
	}
}

func TestTheEconomyDayStartsAtMidnightUtc(t *testing.T) {
	at := time.Date(2026, 12, 10, 23, 30, 0, 0, time.FixedZone("AZT", 4*3600))
	if got := EconomyDay(at); !got.Equal(time.Date(2026, 12, 10, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("EconomyDay = %v", got)
	}
}
