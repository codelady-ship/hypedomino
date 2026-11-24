package kpi_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/kpi"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

type fixture struct {
	t    *testing.T
	pool *pgxpool.Pool
}

func (f fixture) exec(sql string, args ...any) {
	f.t.Helper()
	if _, err := f.pool.Exec(context.Background(), sql, args...); err != nil {
		f.t.Fatal(err)
	}
}

func (f fixture) analyticsID() string {
	f.t.Helper()
	var id string
	err := f.pool.QueryRow(context.Background(), `with account as (insert into accounts default values returning id)
		insert into analytics_ids (analytics_id, account_id, first_seen_at) select gen_random_uuid(), id, now() from account returning analytics_id::text`).Scan(&id)
	if err != nil {
		f.t.Fatal(err)
	}
	return id
}

func (f fixture) event(analyticsID any, name string, received time.Time) {
	f.t.Helper()
	f.exec(`insert into analytics_events (analytics_id, name, label, value, client_time_ms, received_at) values ($1, $2, '', 0, 1, $3)`, analyticsID, name, received)
}

func (f fixture) match(mode string, ended time.Time, cancelled bool) {
	f.t.Helper()
	f.exec(`insert into matches (id, started_at, ended_at, mode, kind, config_version, seed, cancelled) values (gen_random_uuid(), $1, $2, $3, 'casual', 1, 1, $4)`,
		ended.Add(-10*time.Minute), ended, mode, cancelled)
}

func at(text string, hour int) time.Time {
	return day(text).Add(time.Duration(hour) * time.Hour)
}

func TestTheReportCountsCompletedMatchesAndRetentionFromRawEvents(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	f := fixture{t, pool}
	now := at("2026-11-21", 10)

	returnsNextDayAndAWeekLater := f.analyticsID()
	f.event(returnsNextDayAndAWeekLater, "app_open", at("2026-11-09", 23))
	f.event(returnsNextDayAndAWeekLater, "app_open", at("2026-11-09", 23))
	f.event(returnsNextDayAndAWeekLater, "app_open", at("2026-11-10", 0))
	f.event(returnsNextDayAndAWeekLater, "app_open", at("2026-11-16", 12))
	opensOnce := f.analyticsID()
	f.event(opensOnce, "app_open", at("2026-11-09", 8))
	f.event(opensOnce, "screen_view", at("2026-11-10", 8))
	returnsOnDayTwo := f.analyticsID()
	f.event(returnsOnDayTwo, "app_open", at("2026-11-12", 8))
	f.event(returnsOnDayTwo, "app_open", at("2026-11-14", 8))
	startedBeforeTheReport := f.analyticsID()
	f.event(startedBeforeTheReport, "app_open", at("2026-10-01", 8))
	f.event(startedBeforeTheReport, "app_open", at("2026-11-10", 8))
	startsToday := f.analyticsID()
	f.event(startsToday, "app_open", at("2026-11-21", 1))
	f.event(nil, "match_end", at("2026-11-09", 9))

	f.match("1v1", at("2026-11-20", 23), false)
	f.match("1v1", at("2026-11-20", 1), false)
	f.match("2v2", at("2026-11-20", 5), false)
	f.match("2v2", at("2026-11-20", 6), true)
	f.match("1v1", at("2026-11-21", 1), false)
	f.match("1v1", at("2026-11-08", 23), false)

	report, err := kpi.Build(context.Background(), pool, now, 2)
	if err != nil {
		t.Fatal(err)
	}
	if report.From != "2026-11-09" || report.To != "2026-11-21" {
		t.Fatalf("period %s to %s", report.From, report.To)
	}
	if len(report.Matches) != 24 || report.Matches[0] != (kpi.MatchDay{Day: "2026-11-09", Mode: "1v1"}) {
		t.Fatalf("match days %v", report.Matches)
	}
	last := report.Matches[22:]
	if last[0] != (kpi.MatchDay{Day: "2026-11-20", Mode: "1v1", Completed: 2}) || last[1] != (kpi.MatchDay{Day: "2026-11-20", Mode: "2v2", Completed: 1}) {
		t.Fatalf("completed matches on the last day %v", last)
	}

	cohorts := map[string]kpi.Cohort{}
	for _, cohort := range report.DailyCohorts {
		cohorts[cohort.Start] = cohort
	}
	if len(report.DailyCohorts) != 12 || cohorts["2026-11-10"].Size != 0 || cohorts["2026-11-21"].Start != "" {
		t.Fatalf("daily cohorts %v", report.DailyCohorts)
	}
	first, second := cohorts["2026-11-09"], cohorts["2026-11-12"]
	if first.Size != 2 || *first.D1 != 1 || *first.D7 != 1 || second.Size != 1 || *second.D1 != 0 || *second.D7 != 0 {
		t.Fatalf("cohorts %+v %+v", first, second)
	}
	week := report.WeeklyCohorts[0]
	if len(report.WeeklyCohorts) != 2 || week.Start != "2026-11-09" || week.Size != 3 || *week.D1 != 1 || week.D7 != nil {
		t.Fatalf("weekly cohorts %+v", report.WeeklyCohorts)
	}
}

func TestTheReportCoversOneToThirteenWeeks(t *testing.T) {
	for _, weeks := range []int{0, kpi.MaxWeeks + 1} {
		if _, err := kpi.Build(context.Background(), nil, time.Now(), weeks); !errors.Is(err, kpi.ErrInvalidWeeks) {
			t.Errorf("%d weeks: %v", weeks, err)
		}
	}
}
