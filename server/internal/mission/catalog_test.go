package mission

import (
	"errors"
	"os"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/medal"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
)

func example(t *testing.T) *Catalog {
	t.Helper()
	raw, err := os.ReadFile("../../../docs/pass/examples/missions.json")
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}

func eventMission(slug, eventID string, xp int) Mission {
	return Mission{ID: "mission." + slug, Revision: 1, Type: Event, NameKey: "mission." + slug + ".name", Event: MatchWon, Target: 1, EventID: eventID, Bundle: &reward.Bundle{ID: "bundle.event_" + slug, Revision: 1, PassXP: xp}, Availability: Live}
}

func TestTheExampleHoldsTheSamplePools(t *testing.T) {
	catalog := example(t)
	counts := map[string]int{}
	var achievementCoins int64
	for _, mission := range catalog.Missions {
		counts[mission.Type]++
		if mission.Type == Achievement {
			achievementCoins += mission.Bundle.Coins
		}
	}
	if counts[Daily] != 9 || counts[Weekly] != 8 || counts[Achievement] != 11 || achievementCoins != 7900 {
		t.Fatalf("counts %v, achievement coins %d", counts, achievementCoins)
	}
	win, ok := catalog.Mission("mission.win_50")
	if !ok || win.Next != "mission.win_250" || win.Bundle.Medals[0] != "medal.wins_50" {
		t.Fatalf("mission.win_50 = %+v", win)
	}
}

func TestPublishRulesRefuseBrokenMissions(t *testing.T) {
	cases := []struct {
		name   string
		change func(*Catalog)
	}{
		{"bad id", func(c *Catalog) { c.Missions[0].ID = "quest.daily_play_1" }},
		{"revision zero", func(c *Catalog) { c.Missions[0].Revision = 0 }},
		{"name key of another slug", func(c *Catalog) { c.Missions[0].NameKey = "mission.other.name" }},
		{"unknown type", func(c *Catalog) { c.Missions[0].Type = "monthly" }},
		{"unknown availability", func(c *Catalog) { c.Missions[0].Availability = "hidden" }},
		{"target zero", func(c *Catalog) { c.Missions[0].Target = 0 }},
		{"PV-7 unknown event", func(c *Catalog) { c.Missions[0].Event = "tile_bought" }},
		{"PV-7 unknown queue filter", func(c *Catalog) { c.Missions[0].Filters.Queue = "custom" }},
		{"PV-7 unknown mode filter", func(c *Catalog) { c.Missions[0].Filters.Mode = "3v3" }},
		{"PV-7 bundle on a daily mission", func(c *Catalog) {
			c.Missions[0].Bundle = &reward.Bundle{ID: "bundle.x", Revision: 1, Coins: 10}
		}},
		{"PV-7 slot on an achievement", func(c *Catalog) { c.Missions[len(c.Missions)-1].Slot = 1 }},
		{"daily slot outside the slots", func(c *Catalog) { c.Missions[0].Slot = 4 }},
		{"daily weight zero", func(c *Catalog) { c.Missions[0].Weight = 0 }},
		{"achievement without a bundle", func(c *Catalog) { c.Missions[len(c.Missions)-1].Bundle = nil }},
		{"achievement with pass xp", func(c *Catalog) { c.Missions[len(c.Missions)-1].Bundle.PassXP = 100 }},
		{"achievement with crystals", func(c *Catalog) { c.Missions[len(c.Missions)-1].Bundle.Crystals = 10 }},
		{"next is not a mission", func(c *Catalog) { c.Missions[len(c.Missions)-1].Next = "mission.nobody" }},
		{"next is a daily mission", func(c *Catalog) { c.Missions[len(c.Missions)-1].Next = "mission.daily_play_1" }},
		{"chain loops", func(c *Catalog) { c.Missions[len(c.Missions)-1].Next = "mission.points_10000" }},
		{"PV-5 daily coins above the budget", func(c *Catalog) { c.DailySlots[2].Coins = 51 }},
		{"PV-5 weekly xp above the budget", func(c *Catalog) { c.WeeklySlots[0].Xp = 601 }},
		{"PV-5 slot without a live mission", func(c *Catalog) {
			for i := range c.Missions {
				if c.Missions[i].Type == Weekly && c.Missions[i].Slot == 4 {
					c.Missions[i].Availability = Retired
				}
			}
		}},
		{"PV-6 event xp above the budget", func(c *Catalog) {
			c.Missions = append(c.Missions, eventMission("cup_a", "event.cup", 2000), eventMission("cup_b", "event.cup", 1001))
		}},
		{"event mission without an event id", func(c *Catalog) { c.Missions = append(c.Missions, eventMission("cup_a", "", 100)) }},
		{"mission listed twice", func(c *Catalog) { c.Missions = append(c.Missions, c.Missions[0]) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			catalog := example(t)
			c.change(catalog)
			if err := catalog.Validate(); !errors.Is(err, ErrInvalidCatalog) {
				t.Fatalf("Validate() = %v", err)
			}
		})
	}
}

func TestEventMissionsWithinTheBudgetPublish(t *testing.T) {
	catalog := example(t)
	catalog.Missions = append(catalog.Missions, eventMission("cup_a", "event.cup", 2000), eventMission("cup_b", "event.cup", 1000), eventMission("fair_a", "event.fair", 3000))
	if err := catalog.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestAMissionMedalOrCosmeticMustListTheMission(t *testing.T) {
	raw, err := os.ReadFile("../../../docs/pass/examples/medals.json")
	if err != nil {
		t.Fatal(err)
	}
	medals, err := medal.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	lists := func(id string, source cosmetics.Source) bool {
		found, ok := medals.Medal(id)
		return ok && found.Lists(source)
	}
	items, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	itemLists := func(id string, source cosmetics.Source) bool {
		item, ok := items.Item(id)
		return ok && item.Grantable(source)
	}
	catalog := example(t)
	if err := catalog.CheckSources(lists, itemLists); err != nil {
		t.Fatalf("example catalogs disagree: %v", err)
	}
	cases := map[string]func(*Catalog){
		"medal of another mission": func(c *Catalog) {
			win, _ := c.Mission("mission.win_250")
			win.Bundle.Medals = []string{"medal.wins_50"}
		},
		"medal outside the catalog": func(c *Catalog) {
			win, _ := c.Mission("mission.win_250")
			win.Bundle.Medals = []string{"medal.nobody"}
		},
		"cosmetic of another mission": func(c *Catalog) {
			win, _ := c.Mission("mission.win_250")
			win.Bundle.Cosmetics = []string{"emoji.laugh_spin"}
		},
		"cosmetic outside the catalog": func(c *Catalog) {
			win, _ := c.Mission("mission.win_5_matches")
			win.Bundle.Cosmetics = []string{"emoji.nobody"}
		},
	}
	for name, change := range cases {
		catalog := example(t)
		change(catalog)
		if err := catalog.CheckSources(lists, itemLists); !errors.Is(err, ErrInvalidCatalog) {
			t.Errorf("%s: CheckSources() = %v", name, err)
		}
	}
}
