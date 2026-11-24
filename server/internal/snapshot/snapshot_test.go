package snapshot_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

type settings struct {
	Slots int `json:"slots"`
}

func parse(raw []byte) (settings, error) {
	var s settings
	if err := json.Unmarshal(raw, &s); err != nil {
		return settings{}, err
	}
	if s.Slots < 1 {
		return settings{}, errors.New("slots")
	}
	return s, nil
}

func TestTheWatcherAppliesEveryNewPublishedVersion(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	w := snapshot.New(pool, "shop", parse, logger)
	if _, _, loaded := w.Current(); loaded {
		t.Fatal("loaded before any publish")
	}
	if _, err := w.Load(ctx); !errors.Is(err, snapshot.ErrNone) {
		t.Fatalf("load without a publish: %v", err)
	}
	path := filepath.Join(t.TempDir(), "shop.json")
	if err := os.WriteFile(path, []byte(`{"slots":2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := w.SeedFile(path); err != nil {
		t.Fatal(err)
	}
	if value, version, loaded := w.Current(); !loaded || version != 0 || value.Slots != 2 {
		t.Fatalf("seeded %+v %d %v", value, version, loaded)
	}
	first, err := snapshot.Publish(ctx, pool, "shop", []byte(`{"slots":4}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot.Publish(ctx, pool, "cosmetics", []byte(`{"slots":99}`)); err != nil {
		t.Fatal(err)
	}
	if changed, err := w.Load(ctx); err != nil || !changed {
		t.Fatalf("first load %v %v", changed, err)
	}
	if value, version, _ := w.Current(); version != first || value.Slots != 4 {
		t.Fatalf("after the first load %+v %d", value, version)
	}
	if changed, err := w.Load(ctx); err != nil || changed {
		t.Fatalf("load without a new version %v %v", changed, err)
	}
	if _, err := snapshot.Publish(ctx, pool, "shop", []byte(`{"slots":0}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Load(ctx); err == nil {
		t.Fatal("an invalid snapshot was applied")
	}
	if value, version, _ := w.Current(); version != first || value.Slots != 4 {
		t.Fatalf("invalid snapshot replaced the current one %+v %d", value, version)
	}
	second, err := snapshot.Publish(ctx, pool, "shop", []byte(`{"slots":6}`))
	if err != nil {
		t.Fatal(err)
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		w.Run(runCtx, 50*time.Millisecond)
		close(done)
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, version, _ := w.Current(); version == second {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the runner did not apply the new version")
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	<-done
	if err := w.SeedFile(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("a missing file seeded")
	}
}

func TestNewestIsTheLatestPublishOfAnyDomain(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	if _, err := snapshot.Newest(ctx, pool); !errors.Is(err, snapshot.ErrNone) {
		t.Fatalf("newest before any publish: %v", err)
	}
	_, _ = snapshot.Publish(ctx, pool, "rules", []byte(`{}`))
	latest, _ := snapshot.Publish(ctx, pool, "shop", []byte(`{}`))
	got, err := snapshot.Newest(ctx, pool)
	if err != nil || got.Version != latest || got.Domain != "shop" || got.At.IsZero() {
		t.Fatalf("newest %+v %v", got, err)
	}
}

func TestVersionAndHistoryReadOneDomain(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	first, _ := snapshot.Publish(ctx, pool, "rules", []byte(`{"TurnSeconds": 20}`))
	other, _ := snapshot.Publish(ctx, pool, "shop", []byte(`{}`))
	second, _ := snapshot.Publish(ctx, pool, "rules", []byte(`{"TurnSeconds": 30}`))
	got, err := snapshot.Version(ctx, pool, "rules", first)
	if err != nil || got.Version != first || string(got.Content) != `{"TurnSeconds": 20}` || got.At.IsZero() {
		t.Fatalf("version %+v %v", got, err)
	}
	if _, err := snapshot.Version(ctx, pool, "rules", other); !errors.Is(err, snapshot.ErrNone) {
		t.Fatalf("version of another domain: %v", err)
	}
	history, err := snapshot.History(ctx, pool, "rules", 10)
	if err != nil || len(history) != 2 || history[0].Version != second || history[1].Version != first {
		t.Fatalf("history %+v %v", history, err)
	}
	if limited, err := snapshot.History(ctx, pool, "rules", 1); err != nil || len(limited) != 1 || limited[0].Version != second {
		t.Fatalf("limited %+v %v", limited, err)
	}
}

func TestTheVersionsInEffectWithinDaysStartWithTheOneInEffectAtTheWindowStart(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	publishedDaysAgo := func(domain string, days int) int64 {
		version, err := snapshot.Publish(ctx, pool, domain, []byte(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `update config_snapshots set published_at = now() - make_interval(days => $2) where version = $1`, version, days); err != nil {
			t.Fatal(err)
		}
		return version
	}
	oldest := publishedDaysAgo("shop", 40)
	inEffectAtStart := publishedDaysAgo("shop", 35)
	publishedDaysAgo("rules", 20)
	recent := publishedDaysAgo("shop", 10)
	latest := publishedDaysAgo("shop", 0)
	cases := []struct {
		days int
		want []int64
	}{
		{30, []int64{latest, recent, inEffectAtStart}},
		{5, []int64{latest, recent}},
		{60, []int64{latest, recent, inEffectAtStart, oldest}},
	}
	for _, tc := range cases {
		got, err := snapshot.InEffectWithinDays(ctx, pool, "shop", tc.days)
		if err != nil {
			t.Fatal(err)
		}
		versions := make([]int64, 0, len(got))
		for _, published := range got {
			versions = append(versions, published.Version)
		}
		if !slices.Equal(versions, tc.want) {
			t.Fatalf("%d days: %v, want %v", tc.days, versions, tc.want)
		}
	}
}

func TestTheWatcherReportsEveryAppliedVersion(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := snapshot.New(pool, "shop", parse, slog.New(slog.NewTextHandler(io.Discard, nil)))
	applied := make(chan int64, 2)
	w.OnApplied(func() {
		_, version, _ := w.Current()
		applied <- version
	})
	go w.Run(ctx, 10*time.Millisecond)
	version, err := snapshot.Publish(ctx, pool, "shop", []byte(`{"slots":2}`))
	if err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-applied:
		if got != version {
			t.Fatalf("applied %d, want %d", got, version)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no applied report")
	}
}

func TestTheWatcherIsSyncedOnceTheDatabaseAnswered(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	w := snapshot.New(pool, "shop", parse, logger)
	w.Seed(settings{Slots: 1}, 0)
	if w.Synced() {
		t.Fatal("a seeded watcher counted as synced before it asked the database")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := w.Load(cancelled); err == nil || w.Synced() {
		t.Fatalf("a failed query synced the watcher: %v", err)
	}
	if _, err := w.Load(context.Background()); !errors.Is(err, snapshot.ErrNone) || !w.Synced() {
		t.Fatalf("no published version must still sync the watcher: %v %v", err, w.Synced())
	}
}
