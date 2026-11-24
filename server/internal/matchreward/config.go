package matchreward

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

const (
	Domain = "match_rewards"
	Schema = 1
)

var ErrInvalidConfig = errors.New("matchreward: invalid config")

type Coins struct {
	Win  int64 `json:"win"`
	Loss int64 `json:"loss"`
}

type Config struct {
	Schema               int   `json:"schema"`
	Ranked1v1            Coins `json:"ranked1v1"`
	Ranked2v2            Coins `json:"ranked2v2"`
	Casual1v1            Coins `json:"casual1v1"`
	Casual2v2            Coins `json:"casual2v2"`
	BotSeatRewardPercent int64 `json:"botSeatRewardPercent"`
	DailyMatchCoinCap    int64 `json:"dailyMatchCoinCap"`
	FirstWinBonusCoins   int64 `json:"firstWinBonusCoins"`
}

type Match struct {
	ID        string
	Account   string
	Mode      rules.Mode
	Kind      string
	Won       bool
	BotSeat   bool
	Offender  bool
	Cancelled bool
	EndedAt   time.Time
}

func Default() *Config {
	return &Config{
		Schema:               Schema,
		Ranked1v1:            Coins{Win: 30, Loss: 10},
		Ranked2v2:            Coins{Win: 40, Loss: 14},
		Casual1v1:            Coins{Win: 18, Loss: 6},
		Casual2v2:            Coins{Win: 24, Loss: 8},
		BotSeatRewardPercent: 50,
		DailyMatchCoinCap:    400,
		FirstWinBonusCoins:   100,
	}
}

func Parse(raw []byte) (*Config, error) {
	config := Default()
	if err := json.Unmarshal(raw, config); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	if err := config.Validate(); err != nil {
		return nil, err
	}
	return config, nil
}

func (c *Config) Validate() error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidConfig, fmt.Sprintf(format, args...))
	}
	if c.Schema != Schema {
		return fail("schema %d", c.Schema)
	}
	for name, coins := range map[string]Coins{"ranked1v1": c.Ranked1v1, "ranked2v2": c.Ranked2v2, "casual1v1": c.Casual1v1, "casual2v2": c.Casual2v2} {
		if coins.Win < 0 || coins.Loss < 0 || coins.Loss > coins.Win {
			return fail("%s must pay a win at least as much as a loss, never negative", name)
		}
	}
	if c.BotSeatRewardPercent < 0 || c.BotSeatRewardPercent > 100 {
		return fail("botSeatRewardPercent must be between 0 and 100")
	}
	if c.DailyMatchCoinCap < 0 || c.FirstWinBonusCoins < 0 {
		return fail("dailyMatchCoinCap and firstWinBonusCoins must not be negative")
	}
	return nil
}

func counts(match Match) bool {
	return !match.Cancelled && !match.Offender && (match.Kind == matchstore.KindRanked || match.Kind == matchstore.KindCasual)
}

func (c *Config) MatchCoins(match Match) int64 {
	if !counts(match) {
		return 0
	}
	table := map[string]map[rules.Mode]Coins{
		matchstore.KindRanked: {rules.Mode1v1: c.Ranked1v1, rules.Mode2v2: c.Ranked2v2},
		matchstore.KindCasual: {rules.Mode1v1: c.Casual1v1, rules.Mode2v2: c.Casual2v2},
	}[match.Kind][match.Mode]
	coins := table.Loss
	if match.Won {
		coins = table.Win
	}
	if match.Kind == matchstore.KindCasual && match.BotSeat {
		return (coins*c.BotSeatRewardPercent*2 + 100) / 200
	}
	return coins
}

func (c *Config) FirstWinCoins(match Match) int64 {
	if !counts(match) || !match.Won {
		return 0
	}
	return c.FirstWinBonusCoins
}

func capToRemainder(coins, earnedToday, limit int64) int64 {
	return max(0, min(coins, limit-earnedToday))
}

func EconomyDay(at time.Time) time.Time {
	utc := at.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}
