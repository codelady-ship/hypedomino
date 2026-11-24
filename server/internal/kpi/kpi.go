package kpi

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/matchreward"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

const (
	SessionEvent = "app_open"
	DefaultWeeks = 4
	MaxWeeks     = 13

	day  = 24 * time.Hour
	week = 7 * day
)

var (
	ErrInvalidWeeks = errors.New("kpi: weeks out of range")

	Modes = []string{rules.Mode1v1.String(), rules.Mode2v2.String()}
)

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type MatchDay struct {
	Day       string `json:"day"`
	Mode      string `json:"mode"`
	Completed int64  `json:"completed"`
}

type Cohort struct {
	Start string `json:"start"`
	Size  int64  `json:"size"`
	D1    *int64 `json:"d1"`
	D7    *int64 `json:"d7"`
}

type Report struct {
	From          string     `json:"from"`
	To            string     `json:"to"`
	Matches       []MatchDay `json:"matches"`
	DailyCohorts  []Cohort   `json:"dailyCohorts"`
	WeeklyCohorts []Cohort   `json:"weeklyCohorts"`
}

func Build(ctx context.Context, db Querier, now time.Time, weeks int) (Report, error) {
	if weeks < 1 || weeks > MaxWeeks {
		return Report{}, ErrInvalidWeeks
	}
	today := matchreward.EconomyDay(now)
	from := WeekStart(today).Add(-time.Duration(weeks-1) * week)
	report := Report{From: date(from), To: date(today)}
	matches, err := completedMatches(ctx, db, from, today)
	if err != nil {
		return report, err
	}
	report.Matches = matches
	counts, err := cohortCounts(ctx, db, from, today)
	if err != nil {
		return report, err
	}
	report.DailyCohorts = DailyCohorts(counts, from, today)
	report.WeeklyCohorts = WeeklyCohorts(counts, from, today)
	return report, nil
}

func WeekStart(economyDay time.Time) time.Time {
	sinceMonday := (int(economyDay.Weekday()) + 6) % 7
	return economyDay.Add(-time.Duration(sinceMonday) * day)
}

type CohortCount struct {
	Size, D1, D7 int64
}

func DailyCohorts(counts map[string]CohortCount, from, today time.Time) []Cohort {
	cohorts := []Cohort{}
	for start := from; start.Before(today); start = start.Add(day) {
		count := counts[date(start)]
		cohorts = append(cohorts, Cohort{
			Start: date(start),
			Size:  count.Size,
			D1:    measured(count.D1, start.Add(day), today),
			D7:    measured(count.D7, start.Add(7*day), today),
		})
	}
	return cohorts
}

func WeeklyCohorts(counts map[string]CohortCount, from, today time.Time) []Cohort {
	cohorts := []Cohort{}
	for start := from; start.Before(today); start = start.Add(week) {
		var total CohortCount
		for member := start; member.Before(start.Add(week)); member = member.Add(day) {
			count := counts[date(member)]
			total.Size += count.Size
			total.D1 += count.D1
			total.D7 += count.D7
		}
		lastDay := start.Add(week - day)
		cohorts = append(cohorts, Cohort{
			Start: date(start),
			Size:  total.Size,
			D1:    measured(total.D1, lastDay.Add(day), today),
			D7:    measured(total.D7, lastDay.Add(7*day), today),
		})
	}
	return cohorts
}

func measured(retained int64, returnDay, today time.Time) *int64 {
	if !returnDay.Before(today) {
		return nil
	}
	return &retained
}

func completedMatches(ctx context.Context, db Querier, from, today time.Time) ([]MatchDay, error) {
	rows, err := db.Query(ctx, `select (ended_at at time zone 'UTC')::date, mode, count(*) from matches
		where ended_at >= $1 and ended_at < $2 and not cancelled group by 1, 2`, from, today)
	if err != nil {
		return nil, err
	}
	completed := map[[2]string]int64{}
	var ended time.Time
	var mode string
	var count int64
	if _, err := pgx.ForEachRow(rows, []any{&ended, &mode, &count}, func() error {
		completed[[2]string{date(ended), mode}] = count
		return nil
	}); err != nil {
		return nil, err
	}
	matches := []MatchDay{}
	for start := from; start.Before(today); start = start.Add(day) {
		for _, mode := range Modes {
			matches = append(matches, MatchDay{Day: date(start), Mode: mode, Completed: completed[[2]string{date(start), mode}]})
		}
	}
	return matches, nil
}

func cohortCounts(ctx context.Context, db Querier, from, today time.Time) (map[string]CohortCount, error) {
	rows, err := db.Query(ctx, `with sessions as (
			select distinct analytics_id, (received_at at time zone 'UTC')::date as day from analytics_events
			where name = $3 and analytics_id is not null),
		firsts as (select analytics_id, min(day) as day from sessions group by analytics_id)
		select f.day, count(*), count(d1.analytics_id), count(d7.analytics_id) from firsts f
		left join sessions d1 on d1.analytics_id = f.analytics_id and d1.day = f.day + 1
		left join sessions d7 on d7.analytics_id = f.analytics_id and d7.day = f.day + 7
		where f.day >= $1::date and f.day < $2::date group by f.day`, from, today, SessionEvent)
	if err != nil {
		return nil, err
	}
	counts := map[string]CohortCount{}
	var start time.Time
	var count CohortCount
	_, err = pgx.ForEachRow(rows, []any{&start, &count.Size, &count.D1, &count.D7}, func() error {
		counts[date(start)] = count
		return nil
	})
	return counts, err
}

func date(t time.Time) string {
	return t.UTC().Format(time.DateOnly)
}
