package dailylogin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/reward"
)

const (
	Domain = "daily_login"
	Schema = 1
	Source = "daily_login"
)

var ErrInvalidCalendar = errors.New("dailylogin: invalid calendar")

type Day struct {
	Coins int64 `json:"coins"`
}

type Calendar struct {
	Schema int   `json:"schema"`
	Days   []Day `json:"days"`
}

func Default() *Calendar {
	coins := []int64{30, 30, 50, 50, 70, 70, 150}
	calendar := &Calendar{Schema: Schema}
	for _, amount := range coins {
		calendar.Days = append(calendar.Days, Day{Coins: amount})
	}
	return calendar
}

func Parse(raw []byte) (*Calendar, error) {
	calendar := &Calendar{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(calendar); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCalendar, err)
	}
	if err := calendar.Validate(); err != nil {
		return nil, err
	}
	return calendar, nil
}

func (c *Calendar) Validate() error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidCalendar, fmt.Sprintf(format, args...))
	}
	if c.Schema != Schema {
		return fail("schema %d", c.Schema)
	}
	if len(c.Days) != 7 && len(c.Days) != 30 {
		return fail("a calendar has 7 or 30 days, not %d", len(c.Days))
	}
	for i, day := range c.Days {
		if day.Coins < 1 {
			return fail("day %d grants %d coins, a daily login day grants coins only", i+1, day.Coins)
		}
	}
	return nil
}

func (c *Calendar) Reward(day int, version int64) reward.Bundle {
	return reward.Bundle{ID: fmt.Sprintf("bundle.daily_login_day_%d", day), Revision: int(max(1, version)), Coins: c.Days[day-1].Coins}
}

func (c *Calendar) Next(claims int64) int {
	return int(claims%int64(len(c.Days))) + 1
}

func EconomyDay(at time.Time) string {
	return at.UTC().Format(time.DateOnly)
}
