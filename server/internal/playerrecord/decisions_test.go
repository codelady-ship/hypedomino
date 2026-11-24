package playerrecord_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/playerrecord"
)

func TestDecisionsGroupTheReportsOneDecisionClosedAndAddClosedReviewsNewestFirst(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	at := time.Date(2026, 11, 3, 9, 0, 0, 0, time.UTC)
	ids := make([]string, 4)
	for i := range ids {
		if err := pool.QueryRow(ctx, `insert into accounts default values returning id::text`).Scan(&ids[i]); err != nil {
			t.Fatal(err)
		}
	}
	player, quiet, first, second := ids[0], ids[1], ids[2], ids[3]
	for _, report := range []struct {
		reporter, target, reason, state string
		decided                         time.Time
	}{
		{first, player, "offensive_nickname", "rejected", at},
		{second, player, "impersonation", "rejected", at},
		{first, player, "other", "upheld", at.Add(2 * time.Hour)},
		{second, player, "offensive_nickname", "open", time.Time{}},
		{player, first, "other", "rejected", at.Add(time.Hour)},
	} {
		if _, err := pool.Exec(ctx, `insert into player_reports (reporter, target, reason, name_text, state, created_at, updated_at, decided_at, decided_by)
			values ($1, $2, $3, 'Rude', $4::text, $5, $5, case when $4::text = 'open' then null else $6::timestamptz end, 'moderator')`,
			report.reporter, report.target, report.reason, report.state, at.Add(-time.Hour), report.decided); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `insert into risk_reviews (account_id, cause, opened_at, closed_at, decision, decided_by) values
		($1, 'hard_signal', $2, $3, 'confirm', 'moderator'), ($1, 'score', $2, null, null, null)`, player, at.Add(-2*time.Hour), at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	decisions, err := playerrecord.Decisions(ctx, pool, []string{player, quiet}, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions[quiet]) != 0 || len(decisions) != 1 {
		t.Fatalf("decisions %+v", decisions)
	}
	got := decisions[player]
	if len(got) != 3 {
		t.Fatalf("player decisions %+v", got)
	}
	upheld, review, rejected := got[0], got[1], got[2]
	if upheld.Queue != "reports" || upheld.Outcome != "upheld" || upheld.Reports != 1 || !slices.Equal(upheld.Reasons, []string{"other"}) || !upheld.DecidedAt.Equal(at.Add(2*time.Hour)) {
		t.Fatalf("upheld %+v", upheld)
	}
	if review.Queue != "reviews" || review.Outcome != "confirm" || review.Reports != 0 || !slices.Equal(review.Reasons, []string{"hard_signal"}) || !review.DecidedAt.Equal(at.Add(time.Hour)) {
		t.Fatalf("review %+v", review)
	}
	if rejected.Outcome != "rejected" || rejected.Reports != 2 || !slices.Equal(rejected.Reasons, []string{"impersonation", "offensive_nickname"}) || !rejected.DecidedAt.Equal(at) {
		t.Fatalf("rejected %+v", rejected)
	}

	newest, err := playerrecord.Decisions(ctx, pool, []string{player}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(newest[player]) != 1 || newest[player][0].Outcome != "upheld" {
		t.Fatalf("limited %+v", newest)
	}
}

func TestDecisionsListDecidedAppealsWithTheAppealedSanctionKind(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	at := time.Date(2026, 11, 22, 9, 0, 0, 0, time.UTC)
	var player string
	if err := pool.QueryRow(ctx, `insert into accounts default values returning id::text`).Scan(&player); err != nil {
		t.Fatal(err)
	}
	for _, appealed := range []struct {
		kind, state string
		decided     *time.Time
	}{
		{"ranked_ban", "upheld", &at},
		{"forced_rename", "lifted", ptr(at.Add(time.Hour))},
		{"temporary_ban", "open", nil},
	} {
		var sanctionID int64
		if err := pool.QueryRow(ctx, `insert into account_sanctions (account_id, kind, starts_at, ends_at)
			values ($1, $2::text, $3, case when $2::text = 'forced_rename' then null else $3::timestamptz + interval '72 hours' end) returning id`,
			player, appealed.kind, at.Add(-time.Hour)).Scan(&sanctionID); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `insert into appeals (account_id, sanction_id, text, state, created_at, decided_at, decided_by) values ($1, $2, 'not me', $3, $4, $5, 'moderator')`,
			player, sanctionID, appealed.state, at.Add(-time.Minute), appealed.decided); err != nil {
			t.Fatal(err)
		}
	}
	decisions, err := playerrecord.Decisions(ctx, pool, []string{player}, 50)
	if err != nil {
		t.Fatal(err)
	}
	got := decisions[player]
	if len(got) != 2 || got[0].Queue != "appeals" || got[0].Outcome != "lifted" || !slices.Equal(got[0].Reasons, []string{"forced_rename"}) ||
		got[1].Outcome != "upheld" || !slices.Equal(got[1].Reasons, []string{"ranked_ban"}) || !got[1].DecidedAt.Equal(at) {
		t.Fatalf("appeal decisions %+v", got)
	}
}

func ptr(at time.Time) *time.Time {
	return &at
}
