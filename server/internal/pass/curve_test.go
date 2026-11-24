package pass

import "testing"

var defaultCurve = Curve{Levels: 100, XpBase: 300, XpStep: 50, XpMax: 1000}

func TestTheDefaultCurveMatchesTheSpecTable(t *testing.T) {
	cases := []struct {
		level   int
		toNext  int
		toReach int64
	}{
		{1, 300, 0},
		{2, 350, 300},
		{5, 500, 1500},
		{10, 750, 4500},
		{15, 1000, 8750},
		{20, 1000, 13750},
		{30, 1000, 23750},
		{50, 1000, 43750},
		{70, 1000, 63750},
		{90, 1000, 83750},
		{100, 0, 93750},
	}
	for _, c := range cases {
		if got := defaultCurve.XpToNext(c.level); got != c.toNext {
			t.Errorf("XpToNext(%d) = %d, want %d", c.level, got, c.toNext)
		}
		if got := defaultCurve.XpToReach(c.level); got != c.toReach {
			t.Errorf("XpToReach(%d) = %d, want %d", c.level, got, c.toReach)
		}
	}
}

func TestTheLevelIsAPureFunctionOfTheXpTotal(t *testing.T) {
	cases := []struct {
		xp    int64
		level int
	}{
		{0, 1},
		{299, 1},
		{300, 2},
		{649, 2},
		{650, 3},
		{8750, 15},
		{93749, 99},
		{93750, 100},
		{500000, 100},
	}
	for _, c := range cases {
		if got := defaultCurve.LevelFor(c.xp); got != c.level {
			t.Errorf("LevelFor(%d) = %d, want %d", c.xp, got, c.level)
		}
	}
}

func TestEveryLevelStartsAtItsXpTotal(t *testing.T) {
	for level := 1; level <= defaultCurve.Levels; level++ {
		if got := defaultCurve.LevelFor(defaultCurve.XpToReach(level)); got != level {
			t.Fatalf("LevelFor(XpToReach(%d)) = %d", level, got)
		}
	}
}

func TestABoughtLevelAddsTheXpOfTheLevelsSkipped(t *testing.T) {
	cases := []struct {
		xp    int64
		count int
		want  int64
	}{
		{0, 1, 300},
		{120, 1, 300},
		{120, 3, 1050},
		{8750, 5, 5000},
	}
	for _, c := range cases {
		if got := LevelPurchaseXp(defaultCurve, c.xp, c.count); got != c.want {
			t.Errorf("LevelPurchaseXp(%d, %d) = %d, want %d", c.xp, c.count, got, c.want)
		}
	}
}
