package kpi_test

import (
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/kpi"
)

func day(text string) time.Time {
	parsed, err := time.Parse(time.DateOnly, text)
	if err != nil {
		panic(err)
	}
	return parsed
}

func TestTheEconomyWeekStartsOnMonday(t *testing.T) {
	for _, tc := range []struct{ day, start string }{
		{"2026-11-16", "2026-11-16"},
		{"2026-11-18", "2026-11-16"},
		{"2026-11-22", "2026-11-16"},
		{"2026-11-23", "2026-11-23"},
	} {
		if got := kpi.WeekStart(day(tc.day)).Format(time.DateOnly); got != tc.start {
			t.Errorf("%s starts its week on %s, want %s", tc.day, got, tc.start)
		}
	}
}

func retained(c kpi.Cohort) (d1, d7 any) {
	d1, d7 = "pending", "pending"
	if c.D1 != nil {
		d1 = *c.D1
	}
	if c.D7 != nil {
		d7 = *c.D7
	}
	return d1, d7
}

func TestDailyCohortsShowARetentionDayOnlyOnceItHasEnded(t *testing.T) {
	counts := map[string]kpi.CohortCount{
		"2026-11-02": {Size: 10, D1: 4, D7: 2},
		"2026-11-12": {Size: 5, D1: 3},
		"2026-11-19": {Size: 2},
	}
	cohorts := kpi.DailyCohorts(counts, day("2026-11-02"), day("2026-11-21"))
	if len(cohorts) != 19 || cohorts[0].Start != "2026-11-02" || cohorts[18].Start != "2026-11-20" {
		t.Fatalf("cohort days %v", cohorts)
	}
	for _, tc := range []struct {
		index  int
		size   int64
		d1, d7 any
	}{
		{0, 10, int64(4), int64(2)},
		{1, 0, int64(0), int64(0)},
		{10, 5, int64(3), int64(0)},
		{12, 0, int64(0), "pending"},
		{17, 2, int64(0), "pending"},
		{18, 0, "pending", "pending"},
	} {
		cohort := cohorts[tc.index]
		d1, d7 := retained(cohort)
		if cohort.Size != tc.size || d1 != tc.d1 || d7 != tc.d7 {
			t.Errorf("%s: size %d d1 %v d7 %v, want %d %v %v", cohort.Start, cohort.Size, d1, d7, tc.size, tc.d1, tc.d7)
		}
	}
}

func TestWeeklyCohortsSumTheirDaysAndWaitForTheLastOne(t *testing.T) {
	counts := map[string]kpi.CohortCount{
		"2026-11-02": {Size: 10, D1: 4, D7: 2},
		"2026-11-08": {Size: 6, D1: 1, D7: 1},
		"2026-11-09": {Size: 3, D1: 2, D7: 1},
		"2026-11-16": {Size: 2, D1: 1},
	}
	cohorts := kpi.WeeklyCohorts(counts, day("2026-11-02"), day("2026-11-17"))
	if len(cohorts) != 3 {
		t.Fatalf("weeks %v", cohorts)
	}
	for i, want := range []struct {
		start  string
		size   int64
		d1, d7 any
	}{
		{"2026-11-02", 16, int64(5), int64(3)},
		{"2026-11-09", 3, int64(2), "pending"},
		{"2026-11-16", 2, "pending", "pending"},
	} {
		d1, d7 := retained(cohorts[i])
		if cohorts[i].Start != want.start || cohorts[i].Size != want.size || d1 != want.d1 || d7 != want.d7 {
			t.Errorf("week %d: %+v d1 %v d7 %v, want %+v", i, cohorts[i], d1, d7, want)
		}
	}
}
