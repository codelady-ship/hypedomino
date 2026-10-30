package profile_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
)

var start = time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

func newAccount(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `insert into accounts default values returning id`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestTheFirstNicknameCreatesTheProfileAndIsUniqueOnItsFoldedForm(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := profile.NewStore(pool, profile.DefaultConfig())
	ctx := context.Background()
	ayla, other := newAccount(t, pool), newAccount(t, pool)
	if _, found, err := store.Get(ctx, ayla); err != nil || found {
		t.Fatalf("profile before setup: %v %v", found, err)
	}
	p, err := store.SetNickname(ctx, ayla, "  Ayşe​ Han ", start)
	if err != nil || p.Account != ayla || p.Nickname != "Ayşe Han" || p.Level != 1 || p.XP != 0 || !p.RenamedAt.IsZero() || p.Stats.Games != 0 {
		t.Fatalf("setup: %+v %v", p, err)
	}
	if _, err := store.SetNickname(ctx, other, "AYSE HAN", start); !errors.Is(err, profile.ErrTaken) {
		t.Fatalf("folded duplicate: %v", err)
	}
	if _, err := store.SetNickname(ctx, other, "Ayşe Han", start); !errors.Is(err, profile.ErrTaken) {
		t.Fatalf("exact duplicate: %v", err)
	}
	if _, err := store.SetNickname(ctx, other, "Ayla!", start); !errors.Is(err, profile.ErrInvalidCharacters) {
		t.Fatalf("invalid nickname: %v", err)
	}
	if _, found, err := store.Get(ctx, other); err != nil || found {
		t.Fatalf("a refused setup created a profile: %v %v", found, err)
	}
	same, err := store.SetNickname(ctx, ayla, "Ayşe Han", start.Add(time.Hour))
	if err != nil || !same.RenamedAt.IsZero() {
		t.Fatalf("setting the same nickname again: %+v %v", same, err)
	}
}

func TestRenamesKeepHistoryHoldTheOldNameAndCoolDown(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	config := profile.DefaultConfig()
	config.NicknameRenameCooldown = 48 * time.Hour
	config.NicknameRelease = 24 * time.Hour
	store := profile.NewStore(pool, config)
	ctx := context.Background()
	ayla, other := newAccount(t, pool), newAccount(t, pool)
	if _, err := store.SetNickname(ctx, ayla, "Ayla", start); err != nil {
		t.Fatal(err)
	}
	renamed, err := store.SetNickname(ctx, ayla, "Ayla_07", start.Add(time.Hour))
	if err != nil || renamed.Nickname != "Ayla_07" || !renamed.RenamedAt.Equal(start.Add(time.Hour)) {
		t.Fatalf("free rename: %+v %v", renamed, err)
	}
	if available, cooling := store.RenameAvailableAt(renamed); !cooling || !available.Equal(start.Add(49*time.Hour)) {
		t.Fatalf("rename available at %v %v", available, cooling)
	}
	if _, err := store.SetNickname(ctx, ayla, "Ayla_08", start.Add(2*time.Hour)); !errors.Is(err, profile.ErrCooldown) || !strings.Contains(err.Error(), "2026-11-01T13:00:00Z") {
		t.Fatalf("rename inside the cooldown: %v", err)
	}
	if _, err := store.SetNickname(ctx, other, "ayla", start.Add(2*time.Hour)); !errors.Is(err, profile.ErrTaken) {
		t.Fatalf("released name inside the hold: %v", err)
	}
	if _, err := store.SetNickname(ctx, other, "ayla", start.Add(26*time.Hour)); err != nil {
		t.Fatalf("released name after the hold: %v", err)
	}
	if _, err := store.SetNickname(ctx, ayla, "Ayla_08", start.Add(50*time.Hour)); err != nil {
		t.Fatalf("rename after the cooldown: %v", err)
	}
	var history int
	if err := pool.QueryRow(ctx, `select count(*) from nickname_history where account_id = $1`, ayla).Scan(&history); err != nil || history != 2 {
		t.Fatalf("history rows %d %v", history, err)
	}
}

func TestConcurrentSetupsOfTheSameNicknameLetOneWin(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := profile.NewStore(pool, profile.DefaultConfig())
	ctx := context.Background()
	accounts := make([]string, 6)
	for index := range accounts {
		accounts[index] = newAccount(t, pool)
	}
	var wg sync.WaitGroup
	errs := make([]error, len(accounts))
	for index, id := range accounts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, errs[index] = store.SetNickname(ctx, id, "Champion", start)
		}()
	}
	wg.Wait()
	winners := 0
	for _, err := range errs {
		switch {
		case err == nil:
			winners++
		case !errors.Is(err, profile.ErrTaken):
			t.Fatalf("unexpected error %v", err)
		}
	}
	if winners != 1 {
		t.Fatalf("%d accounts hold the same nickname", winners)
	}
}

func TestSuggestionsAreFreeAndMatchesUpdateTheStatistics(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	store := profile.NewStore(pool, profile.DefaultConfig())
	ctx := context.Background()
	suggestion, err := store.Suggest(ctx, start)
	if err != nil || !strings.HasPrefix(suggestion, "Player") || len(suggestion) != 12 {
		t.Fatalf("suggestion %q %v", suggestion, err)
	}
	ayla := newAccount(t, pool)
	if err := store.RecordMatch(ctx, ayla, "1v1", true, start); !errors.Is(err, profile.ErrUnknownAccount) {
		t.Fatalf("recording before setup: %v", err)
	}
	if _, err := store.SetNickname(ctx, ayla, suggestion, start); err != nil {
		t.Fatal(err)
	}
	for _, result := range []struct {
		mode string
		won  bool
	}{{"1v1", true}, {"1v1", true}, {"2v2", true}, {"1v1", false}, {"2v2", true}} {
		if err := store.RecordMatch(ctx, ayla, result.mode, result.won, start); err != nil {
			t.Fatal(err)
		}
	}
	p, _, err := store.Get(ctx, ayla)
	if err != nil {
		t.Fatal(err)
	}
	want := profile.Stats{Games: 5, Wins: 4, Losses: 1, Streak: 1, BestStreak: 3, Modes: map[string]profile.ModeStats{"1v1": {Games: 3, Wins: 2, Losses: 1}, "2v2": {Games: 2, Wins: 2}}}
	if p.Stats.Games != want.Games || p.Stats.Wins != want.Wins || p.Stats.Losses != want.Losses || p.Stats.Streak != want.Streak || p.Stats.BestStreak != want.BestStreak || p.Stats.Modes["1v1"] != want.Modes["1v1"] || p.Stats.Modes["2v2"] != want.Modes["2v2"] {
		t.Fatalf("stats %+v", p.Stats)
	}
	if p.Stats.WinRatePercent() != 80 {
		t.Fatalf("win rate %d", p.Stats.WinRatePercent())
	}
	if exists, err := store.Exists(ctx, ayla); err != nil || !exists {
		t.Fatalf("exists %v %v", exists, err)
	}
	if exists, err := store.Exists(ctx, "00000000-0000-0000-0000-000000000000"); err != nil || exists {
		t.Fatalf("unknown account exists %v %v", exists, err)
	}
}
