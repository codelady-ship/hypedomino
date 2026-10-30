package rating_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

var now = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

type fixture struct {
	pool    *pgxpool.Pool
	watcher *snapshot.Watcher[*rating.Snapshot]
	store   *rating.Store
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	_, pool := dbtest.Fresh(t)
	watcher := rating.NewWatcher(pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return &fixture{pool: pool, watcher: watcher, store: rating.NewStore(pool, watcher)}
}

func (f *fixture) account(t *testing.T) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `insert into accounts (created_at) values ($1) returning id`, now.Add(-time.Hour)).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func (f *fixture) set(t *testing.T, account string, mode rules.Mode, mmr, matches int, rank rating.Rank) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), `insert into ratings (account_id, mode, mmr, matches, placement_played, placed, rank_index, lp, updated_at) values ($1, $2, $3, $4, 5, true, $5, $6, $7)`, account, mode.String(), mmr, matches, rank.Index, rank.LP, now); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) rating(t *testing.T, account string, mode rules.Mode) rating.Rating {
	t.Helper()
	r, ok, err := f.store.Get(context.Background(), account, mode)
	if err != nil || !ok {
		t.Fatalf("rating of %s: %v %v", account, ok, err)
	}
	return r
}

func TestAResultIsAppliedOnceAndRecorded(t *testing.T) {
	f := newFixture(t)
	a, b := f.account(t), f.account(t)
	f.set(t, a, rules.Mode1v1, 1300, 30, rank(t, "Silver I", 40))
	f.set(t, b, rules.Mode1v1, 1250, 30, rank(t, "Silver I", 0))
	result := rating.Result{MatchID: "11111111-1111-1111-1111-111111111111", Mode: rules.Mode1v1, Teams: [2][]rating.Participant{{{Account: a}}, {{Account: b}}}, Winner: 0, At: now}
	applied, err := f.store.Apply(context.Background(), result)
	if err != nil || applied.Duplicate || applied.ConfigVersion != 0 {
		t.Fatalf("apply %+v %v", applied, err)
	}
	if applied.Changes[0][0].MMR != 1309 || applied.Changes[0][0].LPDelta != 21 {
		t.Fatalf("change %+v", applied.Changes[0][0])
	}
	again, err := f.store.Apply(context.Background(), result)
	if err != nil || !again.Duplicate {
		t.Fatalf("replay %+v %v", again, err)
	}
	ra, rb := f.rating(t, a, rules.Mode1v1), f.rating(t, b, rules.Mode1v1)
	if ra.MMR != 1309 || ra.Matches != 31 || ra.Rank.String() != "Silver I" || ra.Rank.LP != 61 {
		t.Errorf("winner %+v", ra)
	}
	if rb.MMR != 1241 || rb.Rank.String() != "Silver II" || rb.Rank.LP != 75 {
		t.Errorf("loser %+v", rb)
	}
	if _, ok, _ := f.store.Get(context.Background(), a, rules.Mode2v2); ok {
		t.Error("a 1v1 result touched the 2v2 rating")
	}
	var rows int
	if err := f.pool.QueryRow(context.Background(), `select count(*) from rating_changes where match_id = $1`, result.MatchID).Scan(&rows); err != nil || rows != 2 {
		t.Errorf("changes %d %v", rows, err)
	}
	bad := result
	bad.MatchID = "22222222-2222-2222-2222-222222222222"
	bad.Teams[1] = []rating.Participant{{Account: a}}
	if _, err := f.store.Apply(context.Background(), bad); !errors.Is(err, rating.ErrResult) {
		t.Errorf("duplicate account: %v", err)
	}
	bad.Mode = rules.Mode2v2
	bad.Teams[1] = []rating.Participant{{Account: b}}
	if _, err := f.store.Apply(context.Background(), bad); !errors.Is(err, rating.ErrResult) {
		t.Errorf("2v2 with one player per team: %v", err)
	}
}

func TestNewPlayersPlaceAfterTheirPlacementMatches(t *testing.T) {
	f := newFixture(t)
	a := f.account(t)
	opponents := make([]string, 5)
	for i := range opponents {
		opponents[i] = f.account(t)
		f.set(t, opponents[i], rules.Mode2v2, 1500, 40, rank(t, "Gold I", 0))
	}
	partner := f.account(t)
	f.set(t, partner, rules.Mode2v2, 1500, 40, rank(t, "Gold I", 0))
	for i, opponent := range opponents {
		second := opponents[(i+1)%len(opponents)]
		result := rating.Result{MatchID: "33333333-3333-3333-3333-33333333333" + string(rune('0'+i)), Mode: rules.Mode2v2, Teams: [2][]rating.Participant{{{Account: a}, {Account: partner}}, {{Account: opponent}, {Account: second}}}, Winner: 0, At: now.Add(time.Duration(i) * time.Minute)}
		applied, err := f.store.Apply(context.Background(), result)
		if err != nil {
			t.Fatal(err)
		}
		r := f.rating(t, a, rules.Mode2v2)
		if i < 4 && (r.Placed || r.PlacementPlayed != i+1 || r.Rank != (rating.Rank{}) || applied.Changes[0][0].K != 40) {
			t.Fatalf("match %d: %+v %+v", i, r, applied.Changes[0][0])
		}
	}
	r := f.rating(t, a, rules.Mode2v2)
	if !r.Placed || r.PlacementPlayed != 5 || r.Matches != 5 || r.MMR <= 1200 || r.Rank.String() != rating.DefaultConfig().Place(r.MMR, 0).String() || r.Rank.LP != 0 {
		t.Fatalf("placed %+v", r)
	}
	if r.Rank.Index > 12 {
		t.Fatalf("new player above the cap: %s", r.Rank)
	}
}

func TestAPublishedConfigChangesTheNextMatch(t *testing.T) {
	f := newFixture(t)
	a, b := f.account(t), f.account(t)
	f.set(t, a, rules.Mode1v1, 1300, 30, rank(t, "Silver I", 40))
	f.set(t, b, rules.Mode1v1, 1250, 30, rank(t, "Silver I", 0))
	version, err := snapshot.Publish(context.Background(), f.pool, rating.Domain, []byte(`{"schema":1,"modes":{"1v1":{"lpBase":40,"lpChangeMax":60},"2v2":{}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.watcher.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	applied, err := f.store.Apply(context.Background(), rating.Result{MatchID: "44444444-4444-4444-4444-444444444444", Mode: rules.Mode1v1, Teams: [2][]rating.Participant{{{Account: a}}, {{Account: b}}}, Winner: 0, At: now})
	if err != nil || applied.ConfigVersion != version || applied.Changes[0][0].LPDelta != 42 {
		t.Fatalf("applied %+v %v", applied, err)
	}
	var recorded int64
	if err := f.pool.QueryRow(context.Background(), `select config_version from rating_results where match_id = $1`, "44444444-4444-4444-4444-444444444444").Scan(&recorded); err != nil || recorded != version {
		t.Fatalf("recorded version %d %v", recorded, err)
	}
}
