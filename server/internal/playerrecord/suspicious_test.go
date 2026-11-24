package playerrecord_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/playerrecord"
)

func TestMatchesWithAColludingCounterpartAreMarked(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	at := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	accounts := make([]string, 3)
	for i := range accounts {
		if err := pool.QueryRow(ctx, `insert into accounts default values returning id::text`).Scan(&accounts[i]); err != nil {
			t.Fatal(err)
		}
	}
	player, feeder, stranger := accounts[0], accounts[1], accounts[2]
	matches := []struct {
		id       string
		at       time.Time
		opponent string
	}{
		{"21111111-1111-1111-1111-111111111111", at, feeder},
		{"31111111-1111-1111-1111-111111111111", at.Add(time.Minute), stranger},
		{"41111111-1111-1111-1111-111111111111", at.Add(-200 * time.Hour), feeder},
	}
	for _, m := range matches {
		if _, err := pool.Exec(ctx, `insert into matches (id, started_at, mode, kind, config_version, seed, score_a, score_b, winner_team) values ($1, $2, '1v1', 'ranked', 1, 1, 101, 0, 0)`, m.id, m.at); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `insert into match_participants (match_id, started_at, seat, team, account_id, result) values ($1, $2, 0, 0, $3, 'win'), ($1, $2, 1, 1, $4, 'loss')`, m.id, m.at, player, m.opponent); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `insert into rating_signals (account_id, kind, counterpart, mode, matches, wins, offenses, window_hours, first_seen, updated_at)
		values ($1, 'feeding', $2, '1v1', 6, 6, 0, 168, $4, $4), ($1, 'pair_frequency', $3, '1v1', 6, 3, 0, 168, $4, $4)`, player, feeder, stranger, at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	record, err := playerrecord.Load(ctx, pool, player, false)
	if err != nil {
		t.Fatal(err)
	}
	marks := map[string]string{}
	for _, m := range record.Matches {
		marks[m.MatchID] = strings.Join(m.Suspicious, ",")
	}
	if marks[matches[0].id] != "feeding" || marks[matches[1].id] != "" || marks[matches[2].id] != "" || len(marks) != 3 {
		t.Fatalf("marks %v", marks)
	}
	if len(record.PairSignals) != 2 || record.PairSignals[0].UpdatedAt.IsZero() {
		t.Fatalf("pair signals %+v", record.PairSignals)
	}

	suspicious, err := playerrecord.LoadSuspiciousMatches(ctx, pool, playerrecord.SuspiciousQuery{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if len(suspicious) != 1 || suspicious[0].MatchID != matches[0].id || strings.Join(suspicious[0].Signals, ",") != "feeding" || !slices.Equal(suspicious[0].Accounts, slices.Sorted(slices.Values([]string{player, feeder}))) {
		t.Fatalf("suspicious matches %+v", suspicious)
	}

	later := "51111111-1111-1111-1111-111111111111"
	if _, err := pool.Exec(ctx, `insert into matches (id, started_at, mode, kind, config_version, seed, score_a, score_b, winner_team) values ($1, $2, '1v1', 'ranked', 1, 1, 101, 0, 0)`, later, at.Add(30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `insert into match_participants (match_id, started_at, seat, team, account_id, result) values ($1, $2, 0, 0, $3, 'win'), ($1, $2, 1, 1, $4, 'loss')`, later, at.Add(30*time.Minute), player, feeder); err != nil {
		t.Fatal(err)
	}
	listed := func(q playerrecord.SuspiciousQuery) string {
		t.Helper()
		found, err := playerrecord.LoadSuspiciousMatches(ctx, pool, q)
		if err != nil {
			t.Fatal(err)
		}
		ids := make([]string, 0, len(found))
		for _, m := range found {
			ids = append(ids, m.MatchID)
		}
		return strings.Join(ids, ",")
	}
	if page, rest := listed(playerrecord.SuspiciousQuery{Limit: 1}), listed(playerrecord.SuspiciousQuery{After: later, Limit: 1}); page != later || rest != matches[0].id {
		t.Fatalf("pages %q %q", page, rest)
	}
	if fed, traded := listed(playerrecord.SuspiciousQuery{Signal: "feeding", Limit: 50}), listed(playerrecord.SuspiciousQuery{Signal: "win_trade", Limit: 50}); fed != later+","+matches[0].id || traded != "" {
		t.Fatalf("by signal %q %q", fed, traded)
	}
	if byFeeder, byStranger := listed(playerrecord.SuspiciousQuery{Account: feeder[:8], Limit: 50}), listed(playerrecord.SuspiciousQuery{Account: stranger, Limit: 50}); byFeeder != later+","+matches[0].id || byStranger != "" {
		t.Fatalf("by account %q %q", byFeeder, byStranger)
	}
}
