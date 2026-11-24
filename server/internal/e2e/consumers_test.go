package e2e

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/matchreward"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/mission"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

var consumerTopics = []string{matchstore.TopicEconomy, matchstore.TopicXp, matchstore.TopicMissions, matchstore.TopicRating}

func assertConsumersApplyTheMatchOnce(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id string, now time.Time) {
	t.Helper()
	passConfig, missionConfig := seedConsumerConfigs(t, ctx, pool, now)
	runConsumers(t, ctx, pool)
	winner, loser := matchAccount(t, ctx, pool, id, matchstore.ResultWin), matchAccount(t, ctx, pool, id, matchstore.ResultLoss)
	coins := matchreward.Default()
	assertSum(t, ctx, pool, "match coins", `select coalesce(sum(amount), 0) from ledger_entries where account_id = $1 and source_type = 'match' and source_ref = $2`, winner, loser, id, coins.Ranked1v1.Win+coins.FirstWinBonusCoins, coins.Ranked1v1.Loss)
	assertSum(t, ctx, pool, "match xp", `select coalesce(sum(e.amount), 0) from pass_xp_events e join pass_progress p using (account_id, season_id) where e.account_id = $1 and e.match_xp and e.source_ref = $2`, winner, loser, id, int64(passConfig.MatchXp1v1.Win), int64(passConfig.MatchXp1v1.Loss))
	slot := missionConfig.DailySlots[0].Coins
	assertSum(t, ctx, pool, "mission coins", `select coalesce(sum(amount), 0) from ledger_entries where account_id = $1 and source_type = 'mission' and exists (select 1 from mission_matches where account_id = $1 and match_id = $2)`, winner, loser, id, slot, slot)
	assertSum(t, ctx, pool, "weekly win progress", `select coalesce(sum(progress), 0) from mission_assignments a join mission_matches m using (account_id) where a.account_id = $1 and m.match_id = $2 and a.mission_id = 'mission.weekly_win_9'`, winner, loser, id, 1, 0)
	assertSum(t, ctx, pool, "rated games", `select coalesce(sum(r.matches), 0) from ratings r join rating_changes c using (account_id) where r.account_id = $1 and c.match_id = $2 and r.mode = '1v1'`, winner, loser, id, 1, 1)
	assertSum(t, ctx, pool, "mmr gain", `select case when mmr_after > mmr_before then 1 else 0 end from rating_changes where account_id = $1 and match_id = $2`, winner, loser, id, 1, 0)

	before := consumerEffects(t, ctx, pool)
	if _, err := pool.Exec(ctx, `update outbox set published_at = null where key = $1`, "match:"+id); err != nil {
		t.Fatal(err)
	}
	runConsumers(t, ctx, pool)
	if after := consumerEffects(t, ctx, pool); after != before {
		t.Fatalf("a replayed outbox changed the consumer effects\nbefore %s\nafter  %s", before, after)
	}
}

func seedConsumerConfigs(t *testing.T, ctx context.Context, pool *pgxpool.Pool, now time.Time) (*pass.Snapshot, *mission.Catalog) {
	t.Helper()
	passConfig := pass.Default()
	passConfig.Seasons = []pass.Season{passSeason(now)}
	missionConfig := mission.Default()
	missionConfig.DailySlots = missionConfig.DailySlots[:1]
	missionConfig.WeeklySlots = missionConfig.WeeklySlots[:1]
	missionConfig.Missions = []mission.Mission{periodicMission("daily_play_1", mission.Daily, mission.MatchCompleted, 1), periodicMission("weekly_win_9", mission.Weekly, mission.MatchWon, 9)}
	publishConfig(t, ctx, pool, pass.Domain, passConfig)
	publishConfig(t, ctx, pool, mission.Domain, missionConfig)
	season := rating.Season{ID: "season.e2e", NameKey: "season.e2e.name", Start: now.Add(-30 * 24 * time.Hour), End: now.Add(60 * 24 * time.Hour)}
	if _, err := rating.NewStore(pool, rating.NewWatcher(pool, discardLogger())).CreateSeason(ctx, season, now.Add(-31*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	return passConfig, missionConfig
}

func passSeason(now time.Time) pass.Season {
	season := pass.Season{ID: "season_1", Start: now.Add(-10 * 24 * time.Hour), End: now.Add(80 * 24 * time.Hour), Curve: pass.Curve{Levels: 30, XpBase: 1000, XpStep: 1, XpMax: 1000}, Free: map[int]reward.Bundle{}, Premium: map[int]reward.Bundle{}}
	for level := 1; level <= season.Curve.Levels; level++ {
		season.Premium[level] = reward.Bundle{ID: "bundle.e2e_premium", Revision: level, Crystals: 2}
	}
	return season
}

func periodicMission(slug, missionType, event string, target int64) mission.Mission {
	return mission.Mission{ID: "mission." + slug, Revision: 1, Type: missionType, NameKey: "mission." + slug + ".name", Event: event, Target: target, Slot: 1, Weight: 1, Availability: mission.Live}
}

func publishConfig(t *testing.T, ctx context.Context, pool *pgxpool.Pool, domain string, config any) {
	t.Helper()
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot.Publish(ctx, pool, domain, raw); err != nil {
		t.Fatal(err)
	}
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func runConsumers(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	catalog, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	walletStore := wallet.NewStore(pool)
	passConfigs := pass.NewWatcher(pool, discardLogger())
	missionCatalog := mission.NewWatcher(pool, discardLogger())
	for _, load := range []func(context.Context) (bool, error){passConfigs.Load, missionCatalog.Load} {
		if _, err := load(ctx); err != nil {
			t.Fatal(err)
		}
	}
	passes := pass.NewStore(pool, passConfigs)
	engine := reward.NewEngine(pool, walletStore, inventory.NewStore(pool, catalog, inventory.DefaultConfig()), nil, passes, reward.DefaultConfig())
	passes.SetRewards(engine)
	passes.SetWallet(walletStore)
	consumers := map[string]matchstore.Publisher{
		matchstore.TopicEconomy:  matchreward.NewStore(pool, matchreward.NewWatcher(pool, discardLogger()), walletStore),
		matchstore.TopicXp:       passes,
		matchstore.TopicMissions: mission.NewStore(pool, missionCatalog, engine),
		matchstore.TopicRating:   rating.NewStore(pool, rating.NewWatcher(pool, discardLogger())),
	}
	for _, topic := range consumerTopics {
		if published, err := matchstore.NewRelay(matchstore.New(pool), consumers[topic], topic, 0, time.Now).Once(ctx); err != nil || published != 1 {
			t.Fatalf("%s relay published %d, %v", topic, published, err)
		}
	}
}

func matchAccount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, id, result string) string {
	t.Helper()
	var account string
	if err := pool.QueryRow(ctx, `select account_id from match_participants where match_id = $1 and result = $2`, id, result).Scan(&account); err != nil {
		t.Fatal(err)
	}
	return account
}

func assertSum(t *testing.T, ctx context.Context, pool *pgxpool.Pool, what, query, winner, loser, id string, wantWinner, wantLoser int64) {
	t.Helper()
	for account, want := range map[string]int64{winner: wantWinner, loser: wantLoser} {
		var got int64
		if err := pool.QueryRow(ctx, query, account, id).Scan(&got); err != nil || got != want {
			t.Fatalf("%s of %s: %d, %v, want %d", what, account, got, err, want)
		}
	}
}

func consumerEffects(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()
	var effects string
	err := pool.QueryRow(ctx, `select json_build_array(
		(select json_agg(l order by l.id) from (select id, account_id, amount, reason, source_type, source_ref, balance_after from ledger_entries) l),
		(select json_agg(p order by p.account_id) from (select account_id, season_id, xp_total from pass_progress) p),
		(select json_agg(e order by e.account_id, e.idempotency_key) from (select account_id, idempotency_key, amount from pass_xp_events) e),
		(select json_agg(a order by a.account_id, a.mission_id) from (select account_id, mission_id, period, progress, completed_at from mission_assignments) a),
		(select json_agg(r order by r.account_id, r.mode) from (select account_id, mode, mmr, matches, placement_played, placed, rank_index, lp from ratings) r),
		(select count(*) from rating_changes)
	)::text`).Scan(&effects)
	if err != nil {
		t.Fatal(err)
	}
	return effects
}
