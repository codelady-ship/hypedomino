package rating_test

import (
	"context"
	"errors"
	"fmt"
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
	f := &fixture{pool: pool, watcher: watcher, store: rating.NewStore(pool, watcher)}
	if _, err := f.store.CreateSeason(context.Background(), rating.Season{ID: "season.test", NameKey: "season.test.name", Start: now.Add(-30 * 24 * time.Hour), End: now.Add(60 * 24 * time.Hour)}, now.Add(-31*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	return f
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
	if !r.Placed || r.PlacementPlayed != 5 || r.Matches != 5 || r.MMR <= 1200 || r.Rank.String() != rating.DefaultConfig().Place(r.MMR, 0, nil).String() || r.Rank.LP != 0 {
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

func (f *fixture) apex(t *testing.T, lp, wins int, at time.Time) string {
	t.Helper()
	account := f.account(t)
	if _, err := f.pool.Exec(context.Background(), `insert into ratings (account_id, mode, mmr, matches, wins, placement_played, placed, rank_index, lp, lp_at, updated_at) values ($1, '1v1', 2000, 100, $2, 5, true, 28, $3, $4, $4)`, account, wins, lp, at); err != nil {
		t.Fatal(err)
	}
	return account
}

func TestApexTiersAreRecalculatedInLeaderboardOrder(t *testing.T) {
	f := newFixture(t)
	if _, err := snapshot.Publish(context.Background(), f.pool, rating.Domain, []byte(`{"schema":1,"modes":{"1v1":{"challengerSlots":2,"challengerMinLp":300,"grandmasterSlots":2,"grandmasterMinLp":100},"2v2":{}}}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.watcher.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	top := f.apex(t, 500, 10, now)
	early := f.apex(t, 400, 10, now.Add(-time.Hour))
	late := f.apex(t, 400, 50, now)
	mid := f.apex(t, 250, 10, now)
	low := f.apex(t, 150, 10, now)
	floor := f.apex(t, 10, 10, now)
	diamond := f.account(t)
	f.set(t, diamond, rules.Mode1v1, 1900, 100, rank(t, "Diamond I", 90))
	result, err := f.store.RecalculateApex(context.Background(), rules.Mode1v1)
	if err != nil || result.Skipped || result.Challenger != 2 || result.Grandmaster != 2 || result.Master != 2 {
		t.Fatalf("recalculated %+v %v", result, err)
	}
	want := map[string]string{top: rating.TierChallenger, early: rating.TierChallenger, late: rating.TierGrandmaster, mid: rating.TierGrandmaster, low: rating.TierMaster, floor: rating.TierMaster, diamond: "Diamond"}
	for account, tier := range want {
		if got := f.rating(t, account, rules.Mode1v1).Tier; got != tier {
			t.Errorf("%s: tier %s, want %s", account, got, tier)
		}
	}
	again, err := f.store.RecalculateApex(context.Background(), rules.Mode1v1)
	if err != nil || again != result {
		t.Fatalf("second run %+v %v", again, err)
	}
	if other, err := f.store.RecalculateApex(context.Background(), rules.Mode2v2); err != nil || other != (rating.Recalculated{}) {
		t.Fatalf("empty mode %+v %v", other, err)
	}
	applied, err := f.store.Apply(context.Background(), rating.Result{MatchID: "55555555-5555-5555-5555-555555555555", Mode: rules.Mode1v1, Teams: [2][]rating.Participant{{{Account: diamond}}, {{Account: floor}}}, Winner: 0, At: now.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	promoted, demoted := f.rating(t, diamond, rules.Mode1v1), f.rating(t, floor, rules.Mode1v1)
	if !promoted.Rank.Apex() || promoted.Tier != rating.TierMaster || promoted.Wins != 1 || !promoted.LPAt.Equal(now.Add(time.Hour)) {
		t.Errorf("promoted %+v %+v", promoted, applied.Changes[0][0])
	}
	if demoted.Rank.String() != "Diamond I" || demoted.Tier != "Diamond" || demoted.Wins != 10 || !demoted.LPAt.Equal(now.Add(time.Hour)) {
		t.Errorf("demoted %+v %+v", demoted, applied.Changes[1][0])
	}
	kept := f.rating(t, top, rules.Mode1v1)
	if kept.Tier != rating.TierChallenger || !kept.LPAt.Equal(now) {
		t.Errorf("untouched %+v", kept)
	}
}

func TestTheNextApexRunIsTheComingHour(t *testing.T) {
	at := time.Date(2026, 10, 30, 12, 30, 0, 0, time.UTC)
	for hour, want := range map[int]time.Time{0: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), 12: time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC), 13: time.Date(2026, 10, 30, 13, 0, 0, 0, time.UTC)} {
		if got := rating.NextApexRun(at, hour); !got.Equal(want) {
			t.Errorf("hour %d: %s, want %s", hour, got, want)
		}
	}
}

func TestAReturningPlayerPlacesInsideTheResetRange(t *testing.T) {
	f := newFixture(t)
	returning, opponent := f.account(t), f.account(t)
	if _, err := f.pool.Exec(context.Background(), `insert into ratings (account_id, mode, mmr, matches, placement_played, placed, rank_index, lp, reset_index, season_id, updated_at) values ($1, '1v1', 1000, 60, 4, false, 10, 0, 10, 'season.test', $2)`, returning, now); err != nil {
		t.Fatal(err)
	}
	f.set(t, opponent, rules.Mode1v1, 1000, 60, rank(t, "Bronze I", 0))
	applied, err := f.store.Apply(context.Background(), rating.Result{MatchID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", Mode: rules.Mode1v1, Teams: [2][]rating.Participant{{{Account: returning}}, {{Account: opponent}}}, Winner: 1, At: now})
	if err != nil {
		t.Fatal(err)
	}
	r := f.rating(t, returning, rules.Mode1v1)
	if !r.Placed || r.MMR != 980 || r.Rank.String() != "Bronze II" || r.Rank.LP != 0 || applied.Changes[0][0].Rank.String() != "Bronze II" {
		t.Fatalf("returning player placed %+v %+v", r, applied.Changes[0][0])
	}
}

type fakeBoards struct {
	updates []string
}

func (f *fakeBoards) Update(_ context.Context, season string, mode rules.Mode, entries []rating.Entry) error {
	for _, e := range entries {
		f.updates = append(f.updates, fmt.Sprintf("%s %s %s %s %d", season, mode, e.Account, e.Rank, e.Wins))
	}
	return nil
}

func TestPlacedPlayersReachTheBoardsAfterEveryChange(t *testing.T) {
	f := newFixture(t)
	boards := &fakeBoards{}
	f.store.SetBoards(boards, slog.New(slog.NewTextHandler(io.Discard, nil)))
	a, b, fresh := f.account(t), f.account(t), f.account(t)
	f.set(t, a, rules.Mode1v1, 1300, 30, rank(t, "Silver I", 40))
	f.set(t, b, rules.Mode1v1, 1250, 30, rank(t, "Silver I", 0))
	if _, err := f.store.Apply(context.Background(), rating.Result{MatchID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", Mode: rules.Mode1v1, Teams: [2][]rating.Participant{{{Account: a}}, {{Account: b}}}, Winner: 0, At: now}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Apply(context.Background(), rating.Result{MatchID: "cccccccc-cccc-cccc-cccc-cccccccccccc", Mode: rules.Mode1v1, Teams: [2][]rating.Participant{{{Account: fresh}}, {{Account: b}}}, Winner: 1, At: now.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	want := []string{"season.test 1v1 " + a + " Silver I 1", "season.test 1v1 " + b + " Silver II 0", "season.test 1v1 " + b + " Silver II 1"}
	if len(boards.updates) != len(want) {
		t.Fatalf("updates %v", boards.updates)
	}
	for i := range want {
		if boards.updates[i] != want[i] {
			t.Errorf("update %d: %s, want %s", i, boards.updates[i], want[i])
		}
	}
	var entries []rating.Entry
	if err := f.store.Placed(context.Background(), "season.test", rules.Mode1v1, func(e rating.Entry) error {
		entries = append(entries, e)
		return nil
	}); err != nil || len(entries) != 2 {
		t.Fatalf("placed %v %v", entries, err)
	}
	if err := f.store.Placed(context.Background(), "season.missing", rules.Mode1v1, func(rating.Entry) error { return nil }); !errors.Is(err, rating.ErrSeasonUnknown) {
		t.Fatalf("unknown season: %v", err)
	}
	end := now.Add(60 * 24 * time.Hour)
	if _, err := f.store.CloseSeason(context.Background(), "season.test", rules.Mode1v1, end); err != nil {
		t.Fatal(err)
	}
	var frozen []rating.Entry
	if err := f.store.Placed(context.Background(), "season.test", rules.Mode1v1, func(e rating.Entry) error {
		frozen = append(frozen, e)
		return nil
	}); err != nil || len(frozen) != 2 || frozen[0].Account != a || frozen[0].Rank.String() != "Silver I" || !frozen[0].LPAt.Before(frozen[1].LPAt) {
		t.Fatalf("frozen %+v %v", frozen, err)
	}
}

func (f *fixture) setCountry(t *testing.T, account, code string) {
	t.Helper()
	if _, err := f.pool.Exec(context.Background(), `update accounts set country = $2 where id = $1`, account, code); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) countries(t *testing.T, season string) map[string]string {
	t.Helper()
	out := map[string]string{}
	if err := f.store.Placed(context.Background(), season, rules.Mode1v1, func(e rating.Entry) error {
		out[e.Account] = e.Country
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestTheBoardCountryIsFixedForTheSeason(t *testing.T) {
	f := newFixture(t)
	boards := &fakeBoards{}
	f.store.SetBoards(boards, slog.New(slog.NewTextHandler(io.Discard, nil)))
	az, stateless := f.account(t), f.account(t)
	f.setCountry(t, az, "AZ")
	f.set(t, az, rules.Mode1v1, 1300, 30, rank(t, "Silver I", 40))
	f.set(t, stateless, rules.Mode1v1, 1250, 30, rank(t, "Silver I", 0))
	play := func(id string, at time.Time) {
		t.Helper()
		if _, err := f.store.Apply(context.Background(), rating.Result{MatchID: id, Mode: rules.Mode1v1, Teams: [2][]rating.Participant{{{Account: az}}, {{Account: stateless}}}, Winner: 0, At: at}); err != nil {
			t.Fatal(err)
		}
	}
	play("dddddddd-dddd-dddd-dddd-dddddddddddd", now)
	if got := f.countries(t, "season.test"); got[az] != "AZ" || got[stateless] != "" {
		t.Fatalf("first match countries %v", got)
	}
	f.setCountry(t, az, "DE")
	f.setCountry(t, stateless, "GE")
	play("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee", now.Add(time.Minute))
	if got := f.countries(t, "season.test"); got[az] != "AZ" || got[stateless] != "GE" {
		t.Fatalf("open season countries %v", got)
	}
	if r := f.rating(t, az, rules.Mode1v1); r.Country != "AZ" {
		t.Fatalf("rating country %q", r.Country)
	}
	end := now.Add(60 * 24 * time.Hour)
	if _, err := f.store.CloseSeason(context.Background(), "season.test", rules.Mode1v1, end); err != nil {
		t.Fatal(err)
	}
	if got := f.countries(t, "season.test"); got[az] != "AZ" || got[stateless] != "GE" {
		t.Fatalf("frozen countries %v", got)
	}
	for i := 0; i < 5; i++ {
		play(fmt.Sprintf("ffffffff-ffff-ffff-ffff-ffffffffff%02d", i), end.Add(time.Duration(i+1)*time.Minute))
	}
	if got := f.countries(t, end.Format("season.2006_01")); got[az] != "DE" || got[stateless] != "GE" {
		t.Fatalf("next season countries %v", got)
	}
}
