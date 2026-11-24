package matchreward

import (
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestTheExampleIsTheDefault(t *testing.T) {
	raw, err := os.ReadFile("../../../docs/economy/examples/match_rewards.json")
	if err != nil {
		t.Fatal(err)
	}
	config, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(config, Default()) {
		t.Fatalf("example %+v differs from the defaults", config)
	}
}

func TestMatchCoinsFollowTheTable(t *testing.T) {
	config := Default()
	cases := []struct {
		name  string
		match Match
		coins int64
		bonus int64
	}{
		{"ranked 1v1 win", Match{Kind: matchstore.KindRanked, Mode: rules.Mode1v1, Won: true}, 30, 100},
		{"ranked 1v1 loss", Match{Kind: matchstore.KindRanked, Mode: rules.Mode1v1}, 10, 0},
		{"ranked 2v2 win", Match{Kind: matchstore.KindRanked, Mode: rules.Mode2v2, Won: true}, 40, 100},
		{"ranked 2v2 loss", Match{Kind: matchstore.KindRanked, Mode: rules.Mode2v2}, 14, 0},
		{"casual 1v1 win", Match{Kind: matchstore.KindCasual, Mode: rules.Mode1v1, Won: true}, 18, 100},
		{"casual 2v2 loss", Match{Kind: matchstore.KindCasual, Mode: rules.Mode2v2}, 8, 0},
		{"casual with a bot seat rounds halves up", Match{Kind: matchstore.KindCasual, Mode: rules.Mode2v2, Won: true, BotSeat: true}, 12, 100},
		{"casual loss with a bot seat", Match{Kind: matchstore.KindCasual, Mode: rules.Mode1v1, BotSeat: true}, 3, 0},
		{"ranked ignores the bot seat flag", Match{Kind: matchstore.KindRanked, Mode: rules.Mode1v1, Won: true, BotSeat: true}, 30, 100},
		{"custom room", Match{Kind: matchstore.KindCustom, Mode: rules.Mode1v1, Won: true}, 0, 0},
		{"cancelled", Match{Kind: matchstore.KindRanked, Mode: rules.Mode1v1, Won: true, Cancelled: true}, 0, 0},
		{"offender", Match{Kind: matchstore.KindRanked, Mode: rules.Mode2v2, Won: true, Offender: true}, 0, 0},
	}
	for _, c := range cases {
		if got := config.MatchCoins(c.match); got != c.coins {
			t.Errorf("%s: %d coins, want %d", c.name, got, c.coins)
		}
		if got := config.FirstWinCoins(c.match); got != c.bonus {
			t.Errorf("%s: %d first win coins, want %d", c.name, got, c.bonus)
		}
	}
}

func TestTheDailyCapCutsTheMatchThatCrossesIt(t *testing.T) {
	cases := []struct{ coins, earned, limit, want int64 }{
		{30, 0, 400, 30},
		{30, 380, 400, 20},
		{30, 400, 400, 0},
		{30, 420, 400, 0},
	}
	for _, c := range cases {
		if got := capToRemainder(c.coins, c.earned, c.limit); got != c.want {
			t.Errorf("cap(%d, %d, %d) = %d, want %d", c.coins, c.earned, c.limit, got, c.want)
		}
	}
}

func TestPublishRulesRefuseBrokenConfigs(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Config)
	}{
		{"wrong schema", func(c *Config) { c.Schema = 2 }},
		{"negative win", func(c *Config) { c.Ranked1v1.Win = -1 }},
		{"loss above win", func(c *Config) { c.Casual2v2.Loss = 25 }},
		{"bot percent above 100", func(c *Config) { c.BotSeatRewardPercent = 101 }},
		{"negative cap", func(c *Config) { c.DailyMatchCoinCap = -1 }},
		{"negative first win", func(c *Config) { c.FirstWinBonusCoins = -1 }},
	}
	for _, c := range cases {
		config := Default()
		c.change(config)
		if err := config.Validate(); !errors.Is(err, ErrInvalidConfig) {
			t.Errorf("%s: Validate() = %v", c.name, err)
		}
	}
}
