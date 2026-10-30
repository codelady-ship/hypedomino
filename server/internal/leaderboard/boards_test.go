package leaderboard_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Alievs-corp/hypedomino/server/internal/cluster/redistest"
	"github.com/Alievs-corp/hypedomino/server/internal/leaderboard"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestMain(m *testing.M) {
	os.Exit(redistest.Main(m))
}

var now = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

type fakeRegions map[string][]string

func (f fakeRegions) Region(country string) (string, bool) {
	for region, countries := range f {
		for _, c := range countries {
			if c == country {
				return region, true
			}
		}
	}
	return "", false
}

func (f fakeRegions) KnownRegion(region string) bool {
	_, ok := f[region]
	return ok
}

var regions = fakeRegions{"caucasus": {"AZ", "GE"}, "eu": {"DE", "FR"}}

type fakeSource struct {
	entries map[string][]rating.Entry
	calls   int
	err     error
}

func (f *fakeSource) Placed(_ context.Context, season string, mode rules.Mode, visit func(rating.Entry) error) error {
	f.calls++
	if f.err != nil {
		return f.err
	}
	for _, e := range f.entries[season+"/"+mode.String()] {
		if err := visit(e); err != nil {
			return err
		}
	}
	return nil
}

func client(t *testing.T) redis.UniversalClient {
	t.Helper()
	options, err := redis.ParseURL(redistest.URL(t))
	if err != nil {
		t.Fatal(err)
	}
	c := redis.NewClient(options)
	t.Cleanup(func() { c.Close() })
	return c
}

func entry(account string, index, lp int, tier string, wins int, at time.Time) rating.Entry {
	if tier == "" {
		tier = (rating.Rank{Index: index}).Tier()
	}
	return rating.Entry{Account: account, Rank: rating.Rank{Index: index, LP: lp}, Tier: tier, Wins: wins, LPAt: at, Version: at.UnixMilli()}
}

func accounts(rows []leaderboard.Row) []string {
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, fmt.Sprintf("%d:%s", r.Position, r.Account))
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestTheGlobalBoardFollowsTheLeaderboardOrder(t *testing.T) {
	source := &fakeSource{entries: map[string][]rating.Entry{"season.s1/1v1": {
		entry("gm", 28, 400, rating.TierGrandmaster, 50, now),
		entry("ch", 28, 300, rating.TierChallenger, 50, now),
		entry("master", 28, 900, rating.TierMaster, 50, now),
		entry("gold-late", 14, 50, "", 10, now.Add(time.Hour)),
		entry("gold-early", 14, 50, "", 5, now),
		entry("gold-wins", 14, 50, "", 20, now.Add(time.Hour)),
		entry("gold-b", 14, 50, "", 10, now.Add(time.Hour)),
		entry("iron", 0, 0, "", 0, now),
	}}}
	prefix := fmt.Sprintf("lbtest%d", time.Now().UnixNano())
	boards := leaderboard.New(client(t), source, regions, leaderboard.Config{Prefix: prefix, TTL: time.Hour})
	top, err := boards.Top(context.Background(), "season.s1", rules.Mode1v1, leaderboard.BoardGlobal, 200)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"1:ch", "2:gm", "3:master", "4:gold-early", "5:gold-wins", "6:gold-b", "7:gold-late", "8:iron"}
	if got := accounts(top); !equal(got, want) {
		t.Fatalf("top %v, want %v", got, want)
	}
	if top[0].Tier != rating.TierChallenger || top[0].Rank.LP != 300 || top[2].Rank.String() != "Master" || top[3].Rank.String() != "Gold II" || top[3].Wins != 5 || top[7].Tier != "Iron" {
		t.Fatalf("decoded rows %+v", top)
	}
	if source.calls != 1 {
		t.Fatalf("rebuilds %d", source.calls)
	}
	around, listed, err := boards.Around(context.Background(), "season.s1", rules.Mode1v1, leaderboard.BoardGlobal, "gold-b", 2)
	if err != nil || !listed || !equal(accounts(around), []string{"4:gold-early", "5:gold-wins", "6:gold-b", "7:gold-late", "8:iron"}) {
		t.Fatalf("around %v %v %v", accounts(around), listed, err)
	}
	around, listed, err = boards.Around(context.Background(), "season.s1", rules.Mode1v1, leaderboard.BoardGlobal, "ch", 2)
	if err != nil || !listed || !equal(accounts(around), []string{"1:ch", "2:gm", "3:master"}) {
		t.Fatalf("around the top %v %v %v", accounts(around), listed, err)
	}
	if _, listed, err := boards.Around(context.Background(), "season.s1", rules.Mode1v1, leaderboard.BoardGlobal, "nobody", 2); err != nil || listed {
		t.Fatalf("unlisted %v %v", listed, err)
	}
	if err := boards.Update(context.Background(), "season.s1", rules.Mode1v1, []rating.Entry{entry("iron", 15, 10, "", 1, now.Add(2*time.Hour))}); err != nil {
		t.Fatal(err)
	}
	if err := boards.Update(context.Background(), "season.s1", rules.Mode1v1, []rating.Entry{entry("iron", 0, 0, "", 0, now)}); err != nil {
		t.Fatal(err)
	}
	top, _ = boards.Top(context.Background(), "season.s1", rules.Mode1v1, leaderboard.BoardGlobal, 4)
	if got := accounts(top); !equal(got, []string{"1:ch", "2:gm", "3:master", "4:iron"}) {
		t.Fatalf("after update %v", got)
	}
	if size, _ := boards.Size(context.Background(), "season.s1", rules.Mode1v1, leaderboard.BoardGlobal); size != 8 {
		t.Fatalf("size %d", size)
	}
	if err := boards.Remove(context.Background(), "season.s1", rules.Mode1v1, []rating.Entry{entry("iron", 0, 0, "", 0, now)}); err != nil {
		t.Fatal(err)
	}
	if size, _ := boards.Size(context.Background(), "season.s1", rules.Mode1v1, leaderboard.BoardGlobal); size != 7 {
		t.Fatalf("size after remove %d", size)
	}
	if _, err := boards.Top(context.Background(), "season.s1", rules.Mode1v1, "friends", 10); !errors.Is(err, leaderboard.ErrUnknownBoard) {
		t.Fatalf("unknown board: %v", err)
	}
	if top, err := boards.Top(context.Background(), "season.s1", rules.Mode2v2, leaderboard.BoardGlobal, 10); err != nil || len(top) != 0 {
		t.Fatalf("empty mode %v %v", top, err)
	}
	source.err = errors.New("database down")
	if _, err := boards.Top(context.Background(), "season.s2", rules.Mode1v1, leaderboard.BoardGlobal, 10); err == nil {
		t.Fatal("rebuild failure hidden")
	}
}

func TestALargeBoardAnswersFast(t *testing.T) {
	size := 20000
	source := &fakeSource{entries: map[string][]rating.Entry{}}
	entries := make([]rating.Entry, 0, size)
	for i := 0; i < size; i++ {
		entries = append(entries, entry(fmt.Sprintf("acc-%06d", i), i%29, (i*37)%100, "", i%50, now.Add(time.Duration(i)*time.Second)))
	}
	source.entries["season.big/2v2"] = entries
	prefix := fmt.Sprintf("lbbig%d", time.Now().UnixNano())
	boards := leaderboard.New(client(t), source, regions, leaderboard.Config{Prefix: prefix, TTL: time.Hour})
	count, err := boards.Rebuild(context.Background(), "season.big", rules.Mode2v2, leaderboard.BoardGlobal)
	if err != nil || count != size {
		t.Fatalf("rebuild %d %v", count, err)
	}
	started := time.Now()
	top, err := boards.Top(context.Background(), "season.big", rules.Mode2v2, leaderboard.BoardGlobal, 200)
	if err != nil || len(top) != 200 || top[0].Rank.Index != 28 {
		t.Fatalf("top %d %v", len(top), err)
	}
	_, listed, err := boards.Around(context.Background(), "season.big", rules.Mode2v2, leaderboard.BoardGlobal, "acc-010000", 2)
	elapsed := time.Since(started)
	if err != nil || !listed {
		t.Fatalf("around %v %v", listed, err)
	}
	t.Logf("top 200 and around in %s", elapsed)
	if os.Getenv(redistest.EnvURL) == "" && elapsed > 50*time.Millisecond {
		t.Errorf("queries took %s", elapsed)
	}
}

func BenchmarkTopOfAMillion(b *testing.B) {
	if testing.Short() {
		b.Skip("integration benchmark skipped under -short")
	}
	size := 1_000_000
	source := &fakeSource{entries: map[string][]rating.Entry{}}
	entries := make([]rating.Entry, 0, size)
	for i := 0; i < size; i++ {
		entries = append(entries, entry(fmt.Sprintf("acc-%07d", i), i%29, (i*37)%100, "", i%50, now.Add(time.Duration(i)*time.Second)))
	}
	source.entries["season.bench/1v1"] = entries
	options, err := redis.ParseURL(redistest.URL(b))
	if err != nil {
		b.Fatal(err)
	}
	c := redis.NewClient(options)
	defer c.Close()
	boards := leaderboard.New(c, source, regions, leaderboard.Config{Prefix: fmt.Sprintf("lbbench%d", time.Now().UnixNano()), TTL: time.Hour})
	if _, err := boards.Rebuild(context.Background(), "season.bench", rules.Mode1v1, leaderboard.BoardGlobal); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := boards.Top(context.Background(), "season.bench", rules.Mode1v1, leaderboard.BoardGlobal, 200); err != nil {
			b.Fatal(err)
		}
		if _, _, err := boards.Around(context.Background(), "season.bench", rules.Mode1v1, leaderboard.BoardGlobal, "acc-0500000", 2); err != nil {
			b.Fatal(err)
		}
	}
}

func national(account, country string, index, lp int, at time.Time) rating.Entry {
	e := entry(account, index, lp, "", 1, at)
	e.Country = country
	return e
}

func TestCountryAndRegionBoardsListOnlyTheirPlayers(t *testing.T) {
	source := &fakeSource{entries: map[string][]rating.Entry{"season.s1/1v1": {
		national("az-1", "AZ", 20, 10, now),
		national("ge-1", "GE", 18, 10, now),
		national("de-1", "DE", 16, 10, now),
		national("az-2", "AZ", 12, 10, now),
		national("stateless", "", 24, 10, now),
	}}}
	prefix := fmt.Sprintf("lbtest%d", time.Now().UnixNano())
	boards := leaderboard.New(client(t), source, regions, leaderboard.Config{Prefix: prefix, TTL: time.Hour})
	for board, want := range map[string][]string{
		leaderboard.BoardGlobal: {"1:stateless", "2:az-1", "3:ge-1", "4:de-1", "5:az-2"},
		"country:AZ":            {"1:az-1", "2:az-2"},
		"region:caucasus":       {"1:az-1", "2:ge-1", "3:az-2"},
		"region:eu":             {"1:de-1"},
		"country:GB":            {},
	} {
		top, err := boards.Top(context.Background(), "season.s1", rules.Mode1v1, board, 10)
		if err != nil || !equal(accounts(top), want) {
			t.Fatalf("%s: %v %v, want %v", board, accounts(top), err, want)
		}
	}
	if err := boards.Update(context.Background(), "season.s1", rules.Mode1v1, []rating.Entry{national("ge-1", "GE", 22, 10, now.Add(time.Hour))}); err != nil {
		t.Fatal(err)
	}
	for board, want := range map[string][]string{
		leaderboard.BoardGlobal: {"1:stateless", "2:ge-1", "3:az-1", "4:de-1", "5:az-2"},
		"country:GE":            {"1:ge-1"},
		"region:caucasus":       {"1:ge-1", "2:az-1", "3:az-2"},
		"region:eu":             {"1:de-1"},
	} {
		top, err := boards.Top(context.Background(), "season.s1", rules.Mode1v1, board, 10)
		if err != nil || !equal(accounts(top), want) {
			t.Fatalf("after update %s: %v %v, want %v", board, accounts(top), err, want)
		}
	}
	around, listed, err := boards.Around(context.Background(), "season.s1", rules.Mode1v1, "region:caucasus", "az-2", 1)
	if err != nil || !listed || !equal(accounts(around), []string{"2:az-1", "3:az-2"}) {
		t.Fatalf("around on region: %v %v %v", accounts(around), listed, err)
	}
	if _, listed, err := boards.Around(context.Background(), "season.s1", rules.Mode1v1, "country:AZ", "de-1", 1); err != nil || listed {
		t.Fatalf("foreigner on country board: %v %v", listed, err)
	}
	for _, board := range []string{"country:az", "country:AZE", "region:asia", "region:", "friends"} {
		if _, err := boards.Top(context.Background(), "season.s1", rules.Mode1v1, board, 10); !errors.Is(err, leaderboard.ErrUnknownBoard) {
			t.Errorf("%s: %v", board, err)
		}
	}
}
