package mission

import (
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestWhatCountsForEachMissionType(t *testing.T) {
	ranked := Match{Kind: matchstore.KindRanked, Mode: rules.Mode1v1}
	cases := []struct {
		name     string
		match    Match
		periodic bool
		lasting  bool
	}{
		{"ranked", ranked, true, true},
		{"casual", Match{Kind: matchstore.KindCasual, Mode: rules.Mode2v2}, true, true},
		{"casual with a bot seat", Match{Kind: matchstore.KindCasual, Mode: rules.Mode1v1, BotSeat: true}, true, false},
		{"ranked ignores the bot seat flag", Match{Kind: matchstore.KindRanked, Mode: rules.Mode1v1, BotSeat: true}, true, true},
		{"custom room", Match{Kind: matchstore.KindCustom, Mode: rules.Mode1v1}, false, false},
		{"cancelled", Match{Kind: matchstore.KindRanked, Mode: rules.Mode1v1, Cancelled: true}, false, false},
		{"offender", Match{Kind: matchstore.KindRanked, Mode: rules.Mode2v2, Offender: true}, false, false},
	}
	for _, c := range cases {
		for _, missionType := range []string{Daily, Weekly} {
			if got := c.match.CountsFor(missionType); got != c.periodic {
				t.Errorf("%s for %s: %v, want %v", c.name, missionType, got, c.periodic)
			}
		}
		for _, missionType := range []string{Achievement, Event} {
			if got := c.match.CountsFor(missionType); got != c.lasting {
				t.Errorf("%s for %s: %v, want %v", c.name, missionType, got, c.lasting)
			}
		}
	}
}

func TestProgressPerEventAndFilter(t *testing.T) {
	match := Match{Kind: matchstore.KindRanked, Mode: rules.Mode2v2, Won: true, HandsWon: 4, Dominoes: 2, BoardPoints: 85, DoubleChains: 1}
	cases := []struct {
		name    string
		mission Mission
		match   Match
		want    int64
	}{
		{"match completed", Mission{Type: Daily, Event: MatchCompleted}, match, 1},
		{"match won", Mission{Type: Daily, Event: MatchWon}, match, 1},
		{"match lost", Mission{Type: Daily, Event: MatchWon}, Match{Kind: matchstore.KindRanked, Mode: rules.Mode2v2}, 0},
		{"hands won", Mission{Type: Weekly, Event: HandWon}, match, 4},
		{"dominoes", Mission{Type: Achievement, Event: Domino}, match, 2},
		{"board points", Mission{Type: Achievement, Event: BoardPoints}, match, 85},
		{"double chains", Mission{Type: Event, Event: DoubleChain}, match, 1},
		{"queue filter fits", Mission{Type: Weekly, Event: MatchWon, Filters: Filters{Queue: matchstore.KindRanked}}, match, 1},
		{"queue filter misses", Mission{Type: Weekly, Event: MatchWon, Filters: Filters{Queue: matchstore.KindCasual}}, match, 0},
		{"mode filter fits", Mission{Type: Daily, Event: MatchCompleted, Filters: Filters{Mode: "2v2"}}, match, 1},
		{"mode filter misses", Mission{Type: Daily, Event: MatchCompleted, Filters: Filters{Mode: "1v1"}}, match, 0},
		{"achievement skips a bot table", Mission{Type: Achievement, Event: MatchCompleted}, Match{Kind: matchstore.KindCasual, Mode: rules.Mode1v1, BotSeat: true}, 0},
		{"daily counts a bot table", Mission{Type: Daily, Event: MatchCompleted}, Match{Kind: matchstore.KindCasual, Mode: rules.Mode1v1, BotSeat: true}, 1},
		{"custom room adds nothing", Mission{Type: Daily, Event: MatchCompleted}, Match{Kind: matchstore.KindCustom, Mode: rules.Mode1v1}, 0},
	}
	for _, c := range cases {
		if got := c.mission.Progress(c.match); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
}
