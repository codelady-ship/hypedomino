package snapshot_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
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
