package appeal_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/appeal"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

func newAccount(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `insert into accounts default values returning id::text`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func impose(t *testing.T, pool *pgxpool.Pool, account string, kind sanction.Kind, ends *time.Time) int64 {
	t.Helper()
	imposed, err := sanction.Impose(context.Background(), pool, account, kind, sanction.Collusion, now.Add(-time.Hour), ends)
	if err != nil {
		t.Fatal(err)
	}
	return imposed.ID
}

func TestASanctionTakesOneAppealAndARejectedOneIsNotSentAgain(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	player, other := newAccount(t, pool), newAccount(t, pool)
	ends := now.Add(72 * time.Hour)
	ranked := impose(t, pool, player, sanction.RankedBan, &ends)
	rename := impose(t, pool, player, sanction.ForcedRename, nil)
	foreign := impose(t, pool, other, sanction.PermanentBan, nil)

	filed, err := appeal.Submit(ctx, pool, player, ranked, "it was my brother", now)
	if err != nil || filed.ID == 0 || filed.State != appeal.StateOpen || filed.AccountID != player || filed.SanctionID != ranked || !filed.CreatedAt.Equal(now) {
		t.Fatalf("submit %+v %v", filed, err)
	}
	if _, err := appeal.Submit(ctx, pool, player, ranked, "again", now.Add(time.Minute)); !errors.Is(err, appeal.ErrOpen) {
		t.Fatalf("second appeal while open %v", err)
	}
	if _, err := appeal.Submit(ctx, pool, player, foreign, "not mine", now); !errors.Is(err, sanction.ErrUnknown) {
		t.Fatalf("an appeal of another account's sanction %v", err)
	}
	if _, err := appeal.Submit(ctx, pool, player, 1<<40, "missing", now); !errors.Is(err, sanction.ErrUnknown) {
		t.Fatalf("an appeal of a missing sanction %v", err)
	}

	open, err := appeal.Open(ctx, pool, appeal.Query{Limit: 50})
	if err != nil || len(open) != 1 || open[0].ID != filed.ID || open[0].Sanction.Kind != sanction.RankedBan || open[0].Sanction.Category != sanction.Collusion {
		t.Fatalf("open %+v %v", open, err)
	}
	account, err := appeal.AccountOf(ctx, pool, filed.ID)
	if err != nil || account != player {
		t.Fatalf("account of %q %v", account, err)
	}
	decided, appealed, err := appeal.Decide(ctx, pool, filed.ID, appeal.StateUpheld, "moderator", now.Add(time.Hour))
	if err != nil || decided.State != appeal.StateUpheld || decided.DecidedAt == nil || appealed.ID != ranked {
		t.Fatalf("decide %+v %+v %v", decided, appealed, err)
	}
	if _, _, err := appeal.Decide(ctx, pool, filed.ID, appeal.StateLifted, "moderator", now.Add(time.Hour)); !errors.Is(err, appeal.ErrNoAppeal) {
		t.Fatalf("a second decision %v", err)
	}
	if _, err := appeal.AccountOf(ctx, pool, filed.ID); !errors.Is(err, appeal.ErrNoAppeal) {
		t.Fatalf("account of a decided appeal %v", err)
	}
	if _, _, err := appeal.Decide(ctx, pool, filed.ID, "maybe", "moderator", now); !errors.Is(err, appeal.ErrDecision) {
		t.Fatalf("an unknown decision %v", err)
	}
	if _, err := appeal.Submit(ctx, pool, player, ranked, "please", now.Add(2*time.Hour)); !errors.Is(err, appeal.ErrRejected) {
		t.Fatalf("an appeal after the rejection %v", err)
	}

	renameAppeal, err := appeal.Submit(ctx, pool, player, rename, "the name is my own", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := appeal.Decide(ctx, pool, renameAppeal.ID, appeal.StateLifted, "moderator", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := appeal.Submit(ctx, pool, player, rename, "again", now.Add(2*time.Hour)); !errors.Is(err, appeal.ErrEnded) {
		t.Fatalf("an appeal after the lift %v", err)
	}
	if open, err := appeal.Open(ctx, pool, appeal.Query{Limit: 50}); err != nil || len(open) != 0 {
		t.Fatalf("open after the decisions %+v %v", open, err)
	}
}

func TestAnEndedSanctionCannotBeAppealed(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	player := newAccount(t, pool)
	ended := now.Add(-time.Minute)
	expired := impose(t, pool, player, sanction.TemporaryBan, &ended)
	if _, err := appeal.Submit(ctx, pool, player, expired, "too late", now); !errors.Is(err, appeal.ErrEnded) {
		t.Fatalf("an appeal of an expired ban %v", err)
	}
	ends := now.Add(time.Hour)
	lifted := impose(t, pool, player, sanction.RankedBan, &ends)
	if _, err := sanction.Lift(ctx, pool, player, sanction.RankedBan, now.Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := appeal.Submit(ctx, pool, player, lifted, "already lifted", now); !errors.Is(err, appeal.ErrEnded) {
		t.Fatalf("an appeal of a lifted ban %v", err)
	}
}
