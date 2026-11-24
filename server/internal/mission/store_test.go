package mission_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/analytics"
	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/mission"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

var now = time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)

var rankedWin = mission.Match{Kind: matchstore.KindRanked, Mode: rules.Mode1v1, Won: true, HandsWon: 3, BoardPoints: 40}

type fixture struct {
	pool   *pgxpool.Pool
	store  *mission.Store
	wallet *wallet.Store
}

func periodic(slug, missionType string, slot int, event string, target int64) mission.Mission {
	return mission.Mission{ID: "mission." + slug, Revision: 1, Type: missionType, NameKey: "mission." + slug + ".name", Event: event, Target: target, Slot: slot, Weight: 1, Availability: mission.Live}
}

func achievement(slug string, target, coins int64, next string) mission.Mission {
	return mission.Mission{ID: "mission." + slug, Revision: 1, Type: mission.Achievement, NameKey: "mission." + slug + ".name", Event: mission.MatchWon, Target: target, Next: next, Bundle: &reward.Bundle{ID: "bundle.mission_" + slug, Revision: 1, Coins: coins}, Availability: mission.Live}
}

func newFixture(t *testing.T, missions ...mission.Mission) *fixture {
	t.Helper()
	_, pool := dbtest.Fresh(t)
	catalog, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	walletStore := wallet.NewStore(pool)
	engine := reward.NewEngine(pool, walletStore, inventory.NewStore(pool, catalog, inventory.DefaultConfig()), nil, nil, reward.DefaultConfig())
	watcher := mission.NewWatcher(pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
	config := mission.Default()
	config.DailySlots = config.DailySlots[:1]
	config.WeeklySlots = config.WeeklySlots[:1]
	config.Missions = missions
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mission.Parse(raw); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot.Publish(context.Background(), pool, mission.Domain, raw); err != nil {
		t.Fatal(err)
	}
	if _, err := watcher.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	return &fixture{pool: pool, store: mission.NewStore(pool, watcher, engine), wallet: walletStore}
}

func (f *fixture) account(t *testing.T) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `insert into accounts (created_at) values ($1) returning id`, now.Add(-time.Hour)).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

func ids(board mission.Board) map[string]int64 {
	out := map[string]int64{}
	for _, a := range board.Missions {
		out[a.Mission.ID] = a.Progress
	}
	return out
}

func TestEachSlotIsDrawnOncePerPeriod(t *testing.T) {
	f := newFixture(t, periodic("daily_play_1", mission.Daily, 1, mission.MatchCompleted, 1), periodic("weekly_win_2", mission.Weekly, 1, mission.MatchWon, 2), achievement("win_5", 5, 100, ""))
	account := f.account(t)
	first, err := f.store.Board(context.Background(), account, now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.store.Board(context.Background(), account, now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Missions) != 3 || len(second.Missions) != 3 || first.Day != "2026-11-03" || first.Week != "2026-W45" {
		t.Fatalf("first %+v, second %+v", first, second)
	}
	if first.DailyRerolls != 1 || first.WeeklyRerolls != 1 {
		t.Fatalf("rerolls %d, %d", first.DailyRerolls, first.WeeklyRerolls)
	}
}

func TestAMatchCompletesMissionsAndGrantsThemOnce(t *testing.T) {
	f := newFixture(t, periodic("daily_play_1", mission.Daily, 1, mission.MatchCompleted, 1), periodic("weekly_win_2", mission.Weekly, 1, mission.MatchWon, 2), achievement("win_1", 1, 100, ""))
	account := f.account(t)
	completed, err := f.store.ApplyMatch(context.Background(), account, "m1", rankedWin, now)
	if err != nil || len(completed) != 2 {
		t.Fatalf("completed %v, %v", completed, err)
	}
	if balance, err := f.wallet.Balance(context.Background(), account, wallet.Coin); err != nil || balance != 130 {
		t.Fatalf("balance %d, %v, want 30 slot coins and 100 achievement coins", balance, err)
	}
	if completed, err := f.store.ApplyMatch(context.Background(), account, "m1", rankedWin, now); err != nil || len(completed) != 0 {
		t.Fatalf("replayed match completed %v, %v", completed, err)
	}
	board, err := f.store.Board(context.Background(), account, now)
	if err != nil || ids(board)["mission.weekly_win_2"] != 1 {
		t.Fatalf("weekly progress after a replay %v, %v", ids(board), err)
	}
	if completed, err := f.store.ApplyMatch(context.Background(), account, "m2", mission.Match{Kind: matchstore.KindCustom, Mode: rules.Mode1v1, Won: true}, now); err != nil || len(completed) != 0 {
		t.Fatalf("custom room completed %v, %v", completed, err)
	}
	if completed, err := f.store.ApplyMatch(context.Background(), account, "m3", rankedWin, now); err != nil || len(completed) != 1 || completed[0] != "mission.weekly_win_2" {
		t.Fatalf("second win completed %v, %v", completed, err)
	}
	var keys int
	if err := f.pool.QueryRow(context.Background(), `select count(*) from reward_grants where account_id = $1 and idempotency_key in ('mission:mission.daily_play_1:2026-11-03', 'mission:mission.weekly_win_2:2026-W45', 'mission:mission.win_1:once')`, account).Scan(&keys); err != nil || keys != 3 {
		t.Fatalf("grant keys %d, %v", keys, err)
	}
	var events, identified int
	if err := f.pool.QueryRow(context.Background(), `select count(*), count(*) filter (where analytics_id is not null or client_time_ms is not null or label <> '' or value <> 0 or received_at <> $2)
		from analytics_events where name = $1`, analytics.MissionComplete, now).Scan(&events, &identified); err != nil || events != 3 || identified != 0 {
		t.Fatalf("%d mission_complete events, %d with an id, a device time, a label, a value or another time, %v", events, identified, err)
	}
}

func TestABotTableFeedsDailyMissionsButNotAchievements(t *testing.T) {
	f := newFixture(t, periodic("daily_play_1", mission.Daily, 1, mission.MatchCompleted, 1), periodic("weekly_win_2", mission.Weekly, 1, mission.MatchWon, 2), achievement("win_1", 1, 100, ""))
	account := f.account(t)
	botTable := mission.Match{Kind: matchstore.KindCasual, Mode: rules.Mode1v1, Won: true, BotSeat: true}
	completed, err := f.store.ApplyMatch(context.Background(), account, "m1", botTable, now)
	if err != nil || len(completed) != 1 || completed[0] != "mission.daily_play_1" {
		t.Fatalf("completed %v, %v", completed, err)
	}
}

func TestTheNextDayAvoidsYesterdaysMission(t *testing.T) {
	f := newFixture(t, periodic("daily_play_1", mission.Daily, 1, mission.MatchCompleted, 1), periodic("daily_win_1", mission.Daily, 1, mission.MatchWon, 1), periodic("weekly_win_2", mission.Weekly, 1, mission.MatchWon, 2))
	account := f.account(t)
	today, err := f.store.Board(context.Background(), account, now)
	if err != nil {
		t.Fatal(err)
	}
	tomorrow, err := f.store.Board(context.Background(), account, now.AddDate(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	for id := range ids(today) {
		if _, again := ids(tomorrow)[id]; again && id != "mission.weekly_win_2" {
			t.Fatalf("%s drawn two days in a row", id)
		}
	}
}

func TestARerollReplacesOneMissionPerPeriod(t *testing.T) {
	f := newFixture(t, periodic("daily_play_1", mission.Daily, 1, mission.MatchCompleted, 2), periodic("daily_win_1", mission.Daily, 1, mission.MatchWon, 2), periodic("weekly_win_2", mission.Weekly, 1, mission.MatchWon, 2))
	account := f.account(t)
	board, err := f.store.Board(context.Background(), account, now)
	if err != nil {
		t.Fatal(err)
	}
	var daily string
	for _, a := range board.Missions {
		if a.Mission.Type == mission.Daily {
			daily = a.Mission.ID
		}
	}
	rerolled, err := f.store.Reroll(context.Background(), account, daily, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, kept := ids(rerolled)[daily]; kept || len(rerolled.Missions) != 2 || rerolled.DailyRerolls != 0 {
		t.Fatalf("after reroll %+v", rerolled)
	}
	for id := range ids(rerolled) {
		if id != "mission.weekly_win_2" {
			if _, err := f.store.Reroll(context.Background(), account, id, now); !errors.Is(err, mission.ErrRerollUsed) {
				t.Fatalf("second reroll = %v", err)
			}
		}
	}
	if _, err := f.store.Reroll(context.Background(), account, "mission.weekly_win_2", now); !errors.Is(err, mission.ErrNoReplacement) {
		t.Fatalf("weekly reroll without a replacement = %v", err)
	}
	if _, err := f.store.Reroll(context.Background(), account, "mission.nobody", now); !errors.Is(err, mission.ErrNotAssigned) {
		t.Fatalf("reroll of an unassigned mission = %v", err)
	}
}

func TestAChainCompletesEveryThresholdAlreadyPassed(t *testing.T) {
	f := newFixture(t, periodic("daily_play_1", mission.Daily, 1, mission.MatchCompleted, 5), periodic("weekly_win_2", mission.Weekly, 1, mission.MatchWon, 5), achievement("win_1", 1, 10, "mission.win_2"), achievement("win_2", 2, 20, ""))
	account := f.account(t)
	if _, err := f.store.ApplyMatch(context.Background(), account, "m1", rankedWin, now); err != nil {
		t.Fatal(err)
	}
	completed, err := f.store.ApplyMatch(context.Background(), account, "m2", rankedWin, now)
	if err != nil || len(completed) != 1 || completed[0] != "mission.win_2" {
		t.Fatalf("completed %v, %v", completed, err)
	}
	if balance, err := f.wallet.Balance(context.Background(), account, wallet.Coin); err != nil || balance != 30 {
		t.Fatalf("balance %d, %v", balance, err)
	}
}
