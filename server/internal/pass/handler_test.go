package pass

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTheViewShowsTheLevelAndTheXpOfTheNextOne(t *testing.T) {
	season := Season{ID: "season_1", Start: time.Unix(0, 0), End: time.Unix(1, 0), Curve: defaultCurve}
	cases := []struct {
		xp      int64
		level   int
		levelXP int64
		next    int64
	}{
		{0, 1, 0, 300},
		{700, 3, 650, 1050},
		{93750, 100, 93750, -1},
	}
	for _, c := range cases {
		view := NewView(Default(), season, Progress{XP: c.xp}, 4)
		if view.Level != c.level || view.LevelXP != c.levelXP || view.ConfigVersion != 4 || view.Levels != 100 {
			t.Fatalf("xp %d: %+v", c.xp, view)
		}
		switch {
		case c.next < 0 && view.NextLevelXP != nil:
			t.Fatalf("xp %d: next level xp %d at the last level", c.xp, *view.NextLevelXP)
		case c.next >= 0 && (view.NextLevelXP == nil || *view.NextLevelXP != c.next):
			t.Fatalf("xp %d: next level xp %v, want %d", c.xp, view.NextLevelXP, c.next)
		}
	}
}

func TestTheViewCarriesTheConfiguredLevelPrices(t *testing.T) {
	season := Season{ID: "season_1", Start: time.Unix(0, 0), End: time.Unix(1, 0), Curve: defaultCurve}
	config := Default()
	config.LevelCoinPrice, config.LevelCrystalPrice = 250, 45
	view := NewView(config, season, Progress{}, 4)
	if view.LevelCoinPrice != 250 || view.LevelCrystalPrice != 45 {
		t.Fatalf("prices %d coins, %d crystals", view.LevelCoinPrice, view.LevelCrystalPrice)
	}
	raw, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var reply map[string]any
	if err := json.Unmarshal(raw, &reply); err != nil {
		t.Fatal(err)
	}
	if reply["level_coin_price"] != 250.0 || reply["level_crystal_price"] != 45.0 {
		t.Fatalf("reply %s", raw)
	}
}
