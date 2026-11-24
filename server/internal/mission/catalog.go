package mission

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const (
	Domain = "missions"
	Schema = 1

	Daily       = "daily"
	Weekly      = "weekly"
	Achievement = "achievement"
	Event       = "event"

	Live    = "live"
	Retired = "retired"

	MatchCompleted = "match_completed"
	MatchWon       = "match_won"
	HandWon        = "hand_won"
	Domino         = "domino"
	BoardPoints    = "board_points"
	DoubleChain    = "double_chain"
)

var (
	ErrInvalidCatalog = errors.New("mission: invalid catalog")
	idPattern         = regexp.MustCompile(`^mission\.([a-z0-9_]+)$`)
	eventIDPattern    = regexp.MustCompile(`^[a-z0-9_.]+$`)
	Types             = []string{Daily, Weekly, Achievement, Event}
	ProgressEvents    = []string{MatchCompleted, MatchWon, HandWon, Domino, BoardPoints, DoubleChain}
	Queues            = []string{matchstore.KindRanked, matchstore.KindCasual}
)

type Filters struct {
	Queue string `json:"queue,omitempty"`
	Mode  string `json:"mode,omitempty"`
}

type Mission struct {
	ID           string         `json:"id"`
	Revision     int            `json:"revision"`
	Type         string         `json:"type"`
	NameKey      string         `json:"nameKey"`
	Event        string         `json:"event"`
	Target       int64          `json:"target"`
	Filters      Filters        `json:"filters"`
	Slot         int            `json:"slot,omitempty"`
	Weight       int            `json:"weight,omitempty"`
	Bundle       *reward.Bundle `json:"bundle,omitempty"`
	Next         string         `json:"next,omitempty"`
	EventID      string         `json:"eventId,omitempty"`
	Availability string         `json:"availability"`
}

type SlotReward struct {
	Coins int64 `json:"coins"`
	Xp    int   `json:"xp"`
}

type Catalog struct {
	Schema           int          `json:"schema"`
	DailySlots       []SlotReward `json:"dailySlots"`
	WeeklySlots      []SlotReward `json:"weeklySlots"`
	DailyCoinBudget  int64        `json:"dailyCoinBudget"`
	WeeklyCoinBudget int64        `json:"weeklyCoinBudget"`
	DailyXpBudget    int          `json:"dailyXpBudget"`
	WeeklyXpBudget   int          `json:"weeklyXpBudget"`
	DailyRerolls     int          `json:"dailyRerolls"`
	WeeklyRerolls    int          `json:"weeklyRerolls"`
	EventXpBudget    int          `json:"eventXpBudget"`
	Missions         []Mission    `json:"missions"`
}

func Default() *Catalog {
	return &Catalog{
		Schema:           Schema,
		DailySlots:       []SlotReward{{Coins: 30, Xp: 100}, {Coins: 40, Xp: 150}, {Coins: 50, Xp: 200}},
		WeeklySlots:      []SlotReward{{Coins: 150, Xp: 600}, {Coins: 150, Xp: 600}, {Coins: 150, Xp: 600}, {Coins: 150, Xp: 600}},
		DailyCoinBudget:  120,
		WeeklyCoinBudget: 600,
		DailyXpBudget:    450,
		WeeklyXpBudget:   2400,
		DailyRerolls:     1,
		WeeklyRerolls:    1,
		EventXpBudget:    3000,
	}
}

func Parse(raw []byte) (*Catalog, error) {
	catalog := Default()
	if err := json.Unmarshal(raw, catalog); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCatalog, err)
	}
	if err := catalog.Validate(); err != nil {
		return nil, err
	}
	return catalog, nil
}

func (c *Catalog) Mission(id string) (Mission, bool) {
	for _, mission := range c.Missions {
		if mission.ID == id {
			return mission, true
		}
	}
	return Mission{}, false
}

func (c *Catalog) Slots(missionType string) []SlotReward {
	if missionType == Weekly {
		return c.WeeklySlots
	}
	if missionType == Daily {
		return c.DailySlots
	}
	return nil
}

func (c *Catalog) Validate() error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidCatalog, fmt.Sprintf(format, args...))
	}
	if c.Schema != Schema {
		return fail("schema %d", c.Schema)
	}
	if c.DailyRerolls < 0 || c.WeeklyRerolls < 0 || c.EventXpBudget < 0 {
		return fail("rerolls and eventXpBudget must not be negative")
	}
	if err := checkSlots(Daily, c.DailySlots, c.DailyCoinBudget, c.DailyXpBudget); err != nil {
		return fail("%v", err)
	}
	if err := checkSlots(Weekly, c.WeeklySlots, c.WeeklyCoinBudget, c.WeeklyXpBudget); err != nil {
		return fail("%v", err)
	}
	seen := map[string]bool{}
	for _, mission := range c.Missions {
		if seen[mission.ID] {
			return fail("%s listed twice", mission.ID)
		}
		seen[mission.ID] = true
		if err := c.validateMission(mission); err != nil {
			return fail("%s: %v", mission.ID, err)
		}
	}
	if err := c.checkChains(); err != nil {
		return fail("%v", err)
	}
	if err := c.checkSlotPools(); err != nil {
		return fail("%v", err)
	}
	return c.checkEventBudgets(fail)
}

func checkSlots(missionType string, slots []SlotReward, coinBudget int64, xpBudget int) error {
	var coins int64
	var xp int
	for i, slot := range slots {
		if slot.Coins < 0 || slot.Xp < 0 {
			return fmt.Errorf("%s slot %d has negative coins or xp", missionType, i+1)
		}
		coins += slot.Coins
		xp += slot.Xp
	}
	if coins > coinBudget || xp > xpBudget {
		return fmt.Errorf("%s slots pay %d coins and %d xp, above the budgets of %d and %d", missionType, coins, xp, coinBudget, xpBudget)
	}
	return nil
}

func (c *Catalog) validateMission(mission Mission) error {
	match := idPattern.FindStringSubmatch(mission.ID)
	if match == nil {
		return errors.New("id must be mission.<slug>")
	}
	switch {
	case mission.Revision < 1:
		return errors.New("revision must be at least 1")
	case mission.NameKey != "mission."+match[1]+".name":
		return errors.New("nameKey must be mission.<slug>.name")
	case !slices.Contains(ProgressEvents, mission.Event):
		return fmt.Errorf("unknown event %q", mission.Event)
	case mission.Target < 1:
		return errors.New("target must be at least 1")
	case mission.Filters.Queue != "" && !slices.Contains(Queues, mission.Filters.Queue):
		return fmt.Errorf("unknown queue filter %q", mission.Filters.Queue)
	case mission.Filters.Mode != "" && !validMode(mission.Filters.Mode):
		return fmt.Errorf("unknown mode filter %q", mission.Filters.Mode)
	case mission.Availability != Live && mission.Availability != Retired:
		return fmt.Errorf("availability %q", mission.Availability)
	}
	switch mission.Type {
	case Daily, Weekly:
		return c.validatePeriodic(mission)
	case Achievement, Event:
		return validateRewarded(mission)
	}
	return fmt.Errorf("unknown type %q", mission.Type)
}

func validMode(text string) bool {
	_, err := rules.ParseMode(text)
	return err == nil
}

func (c *Catalog) validatePeriodic(mission Mission) error {
	switch {
	case mission.Slot < 1 || mission.Slot > len(c.Slots(mission.Type)):
		return fmt.Errorf("slot %d outside the %d %s slots", mission.Slot, len(c.Slots(mission.Type)), mission.Type)
	case mission.Weight < 1:
		return errors.New("weight must be at least 1")
	case mission.Bundle != nil || mission.Next != "" || mission.EventID != "":
		return fmt.Errorf("a %s mission takes no bundle, next or eventId", mission.Type)
	}
	return nil
}

func validateRewarded(mission Mission) error {
	switch {
	case mission.Slot != 0 || mission.Weight != 0:
		return fmt.Errorf("a %s mission takes no slot or weight", mission.Type)
	case mission.Bundle == nil:
		return fmt.Errorf("a %s mission needs a bundle", mission.Type)
	case mission.Type == Achievement && mission.Bundle.PassXP > 0:
		return errors.New("an achievement grants no pass xp")
	case mission.Type == Achievement && mission.EventID != "":
		return errors.New("an achievement takes no eventId")
	case mission.Type == Event && (mission.Next != "" || !eventIDPattern.MatchString(mission.EventID)):
		return errors.New("an event mission needs an eventId and takes no next")
	}
	return mission.Bundle.ValidateFor(wallet.Mission)
}

func (c *Catalog) checkChains() error {
	for _, mission := range c.Missions {
		visited := map[string]bool{mission.ID: true}
		for id := mission.Next; id != ""; {
			next, ok := c.Mission(id)
			if !ok || next.Type != Achievement {
				return fmt.Errorf("%s: next %s is not an achievement", mission.ID, id)
			}
			if visited[id] {
				return fmt.Errorf("%s: chain loops at %s", mission.ID, id)
			}
			visited[id] = true
			id = next.Next
		}
	}
	return nil
}

func (c *Catalog) checkSlotPools() error {
	for _, missionType := range []string{Daily, Weekly} {
		for slot := 1; slot <= len(c.Slots(missionType)); slot++ {
			if !slices.ContainsFunc(c.Missions, func(m Mission) bool {
				return m.Type == missionType && m.Slot == slot && m.Availability == Live
			}) {
				return fmt.Errorf("%s slot %d has no live mission", missionType, slot)
			}
		}
	}
	return nil
}

func (c *Catalog) checkEventBudgets(fail func(string, ...any) error) error {
	xp := map[string]int{}
	for _, mission := range c.Missions {
		if mission.Type == Event {
			xp[mission.EventID] += mission.Bundle.PassXP
		}
	}
	for eventID, total := range xp {
		if total > c.EventXpBudget {
			return fail("event %s missions grant %d pass xp, above %d", eventID, total, c.EventXpBudget)
		}
	}
	return nil
}

func (c *Catalog) CheckSources(medalLists, itemLists func(id string, source cosmetics.Source) bool) error {
	for _, mission := range c.Missions {
		if mission.Bundle == nil {
			continue
		}
		source := cosmetics.Source{Type: SourceMission, Ref: mission.ID}
		for _, grant := range []struct {
			ids   []string
			lists func(string, cosmetics.Source) bool
		}{{mission.Bundle.Medals, medalLists}, {mission.Bundle.Cosmetics, itemLists}} {
			for _, id := range grant.ids {
				if !grant.lists(id, source) {
					return fmt.Errorf("%w: %s grants %s, whose sources do not list it", ErrInvalidCatalog, mission.ID, id)
				}
			}
		}
	}
	return nil
}
