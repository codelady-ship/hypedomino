package db_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/db"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func TestMigrationFilesAreNumberedAndReversible(t *testing.T) {
	migrations, err := db.ListMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(migrations) == 0 || migrations[0].Version != 1 || migrations[0].Name != "00001_core.sql" {
		t.Fatalf("migrations %+v", migrations)
	}
	for i := 1; i < len(migrations); i++ {
		if migrations[i].Version <= migrations[i-1].Version {
			t.Fatalf("versions out of order: %+v", migrations)
		}
	}
}

func tables(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	rows, err := pool.Query(context.Background(), "select tablename from pg_tables where schemaname = 'public' order by tablename")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	return names
}

func has(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}

const head = 11

func TestUpFromEmptyToHeadDownToZeroAndUpAgain(t *testing.T) {
	url := dbtest.Empty(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	migrator, err := db.NewMigrator(url)
	if err != nil {
		t.Fatal(err)
	}
	defer migrator.Close()
	if version, err := migrator.Version(ctx); err != nil || version != 0 {
		t.Fatalf("empty database version %d, %v", version, err)
	}
	version, err := migrator.Up(ctx)
	if err != nil || version != head {
		t.Fatalf("up: version %d, %v", version, err)
	}
	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	for _, want := range []string{"accounts", "auth_identities", "profiles", "inventory_items", "wallets", "ledger_entries", "matches", "match_participants", "matches_default", "match_participants_default", "match_events", "match_events_default", "outbox", "reward_grants", "config_snapshots", "purchases", "purchase_notices", "promo_codes", "promo_targets", "promo_redemptions", "promo_attempts", "ratings", "rating_results", "rating_changes", "goose_db_version"} {
		if !has(tables(t, pool), want) {
			t.Fatalf("table %s missing after up: %v", want, tables(t, pool))
		}
	}
	if version, err := migrator.Up(ctx); err != nil || version != head {
		t.Fatalf("second up: version %d, %v", version, err)
	}
	statuses, err := migrator.Status(ctx)
	if err != nil || len(statuses) != head || !statuses[head-1].Applied || statuses[head-1].Version != head {
		t.Fatalf("status %+v, %v", statuses, err)
	}
	if version, err := migrator.DownTo(ctx, 0); err != nil || version != 0 {
		t.Fatalf("down: version %d, %v", version, err)
	}
	if names := tables(t, pool); len(names) != 1 || names[0] != "goose_db_version" {
		t.Fatalf("tables left after down: %v", names)
	}
	if version, err := migrator.Up(ctx); err != nil || version != head {
		t.Fatalf("up after down: version %d, %v", version, err)
	}
}

func TestConcurrentMigratorsShareTheSessionLock(t *testing.T) {
	url := dbtest.Empty(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	var wg sync.WaitGroup
	errs := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			migrator, err := db.NewMigrator(url)
			if err != nil {
				errs <- err
				return
			}
			defer migrator.Close()
			if _, err := migrator.Up(ctx); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

func unique(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

func check(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23514"
}

func account(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), "insert into accounts default values returning id").Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestAccountIdentityProfileAndInventoryConstraints(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	a, b := account(t, pool), account(t, pool)
	if _, err := pool.Exec(ctx, "insert into auth_identities (provider, subject, account_id, email) values ('google', 'sub-1', $1, 'a@example.com')", a); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "insert into auth_identities (provider, subject, account_id) values ('google', 'sub-1', $1)", b); !unique(err) {
		t.Fatalf("a provider subject bound twice: %v", err)
	}
	if _, err := pool.Exec(ctx, "insert into auth_identities (provider, subject, account_id) values ('facebook', 'sub-2', $1)", b); !check(err) {
		t.Fatalf("unknown provider accepted: %v", err)
	}
	if _, err := pool.Exec(ctx, "insert into profiles (account_id, nickname, nickname_key) values ($1, 'Domino', 'domino')", a); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "insert into profiles (account_id, nickname, nickname_key) values ($1, 'DOMINO', 'domino')", b); !unique(err) {
		t.Fatalf("nickname key unique: %v", err)
	}
	if _, err := pool.Exec(ctx, "insert into profiles (account_id, nickname, nickname_key, level) values ($1, 'other', 'other', 0)", b); !check(err) {
		t.Fatalf("level zero accepted: %v", err)
	}
	if _, err := pool.Exec(ctx, "insert into inventory_items (account_id, item_id, source_type, source_ref) values ($1, 'frame.gold', 'pass_reward', 'pass:1:free:3')", a); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "insert into inventory_items (account_id, item_id, source_type, source_ref) values ($1, 'frame.gold', 'shop_purchase', 'spend:2')", a); !unique(err) {
		t.Fatalf("an item owned twice: %v", err)
	}
	if _, err := pool.Exec(ctx, "insert into accounts (state) values ('frozen')"); !check(err) {
		t.Fatalf("unknown account state accepted: %v", err)
	}
}

func TestWalletAndLedgerConstraints(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	a, b := account(t, pool), account(t, pool)
	if _, err := pool.Exec(ctx, "insert into wallets (account_id, balance) values ($1, -1)", a); err != nil {
		t.Fatalf("a refund may leave a negative balance (migration 00008): %v", err)
	}
	if _, err := pool.Exec(ctx, "insert into wallets (account_id) values ($1)", b); err != nil {
		t.Fatal(err)
	}
	entry := "insert into ledger_entries (account_id, amount, reason, source_type, source_ref, idempotency_key, config_version, balance_after) values ($1, $2, $3, 'match', 'm1', $4, 1, $5)"
	if _, err := pool.Exec(ctx, entry, a, 50, "match_reward", "match:m1", 50); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, entry, a, 50, "match_reward", "match:m1", 100); !unique(err) {
		t.Fatalf("a repeated idempotency key wrote twice: %v", err)
	}
	if _, err := pool.Exec(ctx, entry, b, 50, "match_reward", "match:m1", 50); err != nil {
		t.Fatalf("the same key on another account is another grant: %v", err)
	}
	if _, err := pool.Exec(ctx, entry, a, 500, "purchase", "purchase:play:tx-1", 550); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, entry, b, 500, "purchase", "purchase:play:tx-1", 550); !unique(err) {
		t.Fatalf("a purchase transaction credited to two accounts: %v", err)
	}
	if _, err := pool.Exec(ctx, entry, a, -2000, "refund_revoke", "refund:play:n1", -1450); err != nil {
		t.Fatalf("a refund revoke below zero refused (migration 00008): %v", err)
	}
}

func partitionOf(t *testing.T, pool *pgxpool.Pool, id string) string {
	t.Helper()
	var name string
	if err := pool.QueryRow(context.Background(), "select tableoid::regclass::text from matches where id = $1", id).Scan(&name); err != nil {
		t.Fatal(err)
	}
	return name
}

func TestMatchesArePartitionedByMonth(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	now := time.Now().UTC()
	thisMonth := now.Format("200601")
	insert := "insert into matches (id, started_at, mode, kind, config_version, seed) values (gen_random_uuid(), $1, '2v2', 'ranked', 1, 7) returning id"
	var current, future string
	if err := pool.QueryRow(ctx, insert, now).Scan(&current); err != nil {
		t.Fatal(err)
	}
	if got := partitionOf(t, pool, current); got != "matches_"+thisMonth {
		t.Fatalf("this month's match landed in %s", got)
	}
	if err := pool.QueryRow(ctx, insert, now.AddDate(5, 0, 0)).Scan(&future); err != nil {
		t.Fatal(err)
	}
	if got := partitionOf(t, pool, future); got != "matches_default" {
		t.Fatalf("a match outside the created months landed in %s", got)
	}
	if _, err := pool.Exec(ctx, "insert into match_participants (match_id, started_at, seat, team, bot) values ($1, $2, 0, 0, true)", current, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "insert into match_participants (match_id, started_at, seat, team, bot) values (gen_random_uuid(), $1, 0, 0, true)", now); err == nil {
		t.Fatal("a participant without a match was accepted")
	}
	var created int
	if err := pool.QueryRow(ctx, "select match_partitions_ensure(date '2031-01-15', 2)").Scan(&created); err != nil || created != 6 {
		t.Fatalf("ensure created %d, %v", created, err)
	}
	if err := pool.QueryRow(ctx, "select match_partitions_ensure(date '2031-01-15', 2)").Scan(&created); err != nil || created != 0 {
		t.Fatalf("ensure is not idempotent: %d, %v", created, err)
	}
	names := tables(t, pool)
	for _, want := range []string{"matches_203101", "matches_203102", "match_participants_203101", "match_participants_203102", "match_events_203101", "match_events_203102", "matches_" + thisMonth, "match_participants_" + thisMonth} {
		if !has(names, want) {
			t.Fatalf("partition %s missing: %v", want, names)
		}
	}
}
