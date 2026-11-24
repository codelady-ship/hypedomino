package sanction_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func TestOnlyTheSixCategoriesParse(t *testing.T) {
	cases := []struct {
		name   string
		reason pb.BanReason
	}{
		{"cheating", pb.BanReason_BAN_REASON_CHEATING},
		{"collusion", pb.BanReason_BAN_REASON_COLLUSION},
		{"afk_leaving", pb.BanReason_BAN_REASON_AFK_LEAVING},
		{"offensive_name", pb.BanReason_BAN_REASON_OFFENSIVE_NAME},
		{"payment_fraud", pb.BanReason_BAN_REASON_PAYMENT_FRAUD},
		{"other", pb.BanReason_BAN_REASON_OTHER},
	}
	for _, c := range cases {
		category, err := sanction.ParseCategory(c.name)
		if err != nil || string(category) != c.name || category.Reason() != c.reason {
			t.Fatalf("%s: %q %v %v", c.name, category, category.Reason(), err)
		}
	}
	for _, name := range []string{"", "Cheating", "speed_hack", " other"} {
		if _, err := sanction.ParseCategory(name); !errors.Is(err, sanction.ErrInvalidCategory) {
			t.Fatalf("%q parsed: %v", name, err)
		}
	}
	if got := sanction.Category("unknown").Reason(); got != pb.BanReason_BAN_REASON_OTHER {
		t.Fatalf("an unknown stored category reads as %v", got)
	}
}

func TestASanctionRunsUntilItEndsIsLiftedOrIsRenewed(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	var account string
	if err := pool.QueryRow(ctx, `insert into accounts default values returning id`).Scan(&account); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 22, 12, 0, 0, 0, time.UTC)
	ends := now.Add(72 * time.Hour)
	first, err := sanction.Impose(ctx, pool, account, sanction.RankedBan, sanction.Collusion, now, &ends)
	if err != nil || first.ID == 0 || first.Kind != sanction.RankedBan || first.Category != sanction.Collusion || !first.EndsAt.Equal(ends) || first.LiftedAt != nil {
		t.Fatalf("impose %+v %v", first, err)
	}
	checks := []struct {
		at      time.Time
		kinds   []sanction.Kind
		running bool
	}{
		{now, []sanction.Kind{sanction.RankedBan}, true},
		{ends.Add(-time.Microsecond), []sanction.Kind{sanction.RankedBan}, true},
		{ends, []sanction.Kind{sanction.RankedBan}, false},
		{now, []sanction.Kind{sanction.TemporaryBan, sanction.PermanentBan}, false},
	}
	for _, c := range checks {
		current, running, err := sanction.Current(ctx, pool, account, c.at, c.kinds...)
		if err != nil || running != c.running || (running && current.ID != first.ID) {
			t.Fatalf("at %v kinds %v: %+v %v %v", c.at, c.kinds, current, running, err)
		}
	}

	later := now.Add(time.Hour)
	renewedEnds := later.Add(72 * time.Hour)
	renewed, err := sanction.Impose(ctx, pool, account, sanction.RankedBan, sanction.Cheating, later, &renewedEnds)
	if err != nil || renewed.ID == first.ID {
		t.Fatalf("renew %+v %v", renewed, err)
	}
	if current, _, err := sanction.Current(ctx, pool, account, later, sanction.RankedBan); err != nil || current.ID != renewed.ID || current.Category != sanction.Cheating {
		t.Fatalf("after renewal %+v %v", current, err)
	}
	lifted, err := sanction.Lift(ctx, pool, account, sanction.RankedBan, later.Add(time.Minute))
	if err != nil || lifted != 1 {
		t.Fatalf("lift %d %v", lifted, err)
	}
	if _, running, err := sanction.Current(ctx, pool, account, later.Add(time.Minute), sanction.RankedBan); err != nil || running {
		t.Fatalf("after lift %v %v", running, err)
	}
	if lifted, err := sanction.Lift(ctx, pool, account, sanction.RankedBan, later.Add(time.Hour)); err != nil || lifted != 0 {
		t.Fatalf("second lift %d %v", lifted, err)
	}

	history, err := sanction.List(ctx, pool, account, 50)
	if err != nil || len(history) != 2 || history[0].ID != renewed.ID || history[0].LiftedAt == nil || history[1].ID != first.ID || !history[1].LiftedAt.Equal(later) {
		t.Fatalf("history %+v %v", history, err)
	}
	if err := sanction.RankedBanCheck(pool)(ctx, account, later.Add(time.Minute)); err != nil {
		t.Fatalf("check after lift %v", err)
	}
}

func TestARowWithoutACategoryReadsAsOther(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	var account string
	if err := pool.QueryRow(ctx, `insert into accounts default values returning id`).Scan(&account); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if _, err := pool.Exec(ctx, `insert into account_sanctions (account_id, kind, starts_at, ends_at) values ($1, 'ranked_ban', $2, $3)`, account, now, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	var banned *sanction.RankedBanError
	if err := sanction.RankedBanCheck(pool)(ctx, account, now); !errors.As(err, &banned) || banned.Sanction.Category != sanction.Other {
		t.Fatalf("check %v", err)
	}
	for _, statement := range []string{
		`insert into account_sanctions (account_id, kind, category, starts_at, ends_at) values ($1, 'ranked_ban', 'speed_hack', $2, $2::timestamptz + interval '1 hour')`,
		`insert into account_sanctions (account_id, kind, category, starts_at) values ($1, 'ranked_ban', 'other', $2)`,
		`insert into account_sanctions (account_id, kind, category, starts_at, ends_at) values ($1, 'permanent_ban', 'other', $2, $2::timestamptz + interval '1 hour')`,
	} {
		if _, err := pool.Exec(ctx, statement, account, now); err == nil {
			t.Fatalf("stored %s", statement)
		}
	}
}
