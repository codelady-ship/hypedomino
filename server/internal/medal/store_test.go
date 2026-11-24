package medal_test

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

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/medal"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

var now = time.Date(2026, 12, 10, 12, 0, 0, 0, time.UTC)

var (
	winsSource = cosmetics.Source{Type: medal.SourceMission, Ref: "mission.win_50"}
	goldSource = cosmetics.Source{Type: medal.SourceRankReward, Ref: "bundle.rank_test_gold"}
)

type fixture struct {
	pool   *pgxpool.Pool
	store  *medal.Store
	engine *reward.Engine
}

func testMedal(slug, availability string, source cosmetics.Source) medal.Medal {
	return medal.Medal{ID: "medal." + slug, Revision: 1, NameKey: "medal." + slug + ".name", DescriptionKey: "medal." + slug + ".description", RarityID: "rare", StaticAddress: "medal/" + slug + "/static", Sources: []cosmetics.Source{source}, Availability: availability}
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	_, pool := dbtest.Fresh(t)
	catalog, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	watcher := medal.NewWatcher(pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
	f := &fixture{pool: pool, store: medal.NewStore(pool, watcher)}
	f.engine = reward.NewEngine(pool, wallet.NewStore(pool), inventory.NewStore(pool, catalog, inventory.DefaultConfig()), f.store, nil, reward.DefaultConfig())
	medals := medal.Default()
	medals.Medals = []medal.Medal{
		testMedal("wins_50", "live", winsSource),
		testMedal("rank_gold", "live", goldSource),
		testMedal("domino_100", "live", cosmetics.Source{Type: medal.SourceMission, Ref: "mission.domino_100"}),
		testMedal("chain_first", "live", cosmetics.Source{Type: medal.SourceMission, Ref: "mission.chain_1"}),
		testMedal("old_event", "retired", cosmetics.Source{Type: medal.SourceMission, Ref: "mission.old_event"}),
		testMedal("broken", "disabled", cosmetics.Source{Type: medal.SourceMission, Ref: "mission.broken"}),
	}
	raw, err := json.Marshal(medals)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := medal.Parse(raw); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot.Publish(context.Background(), pool, medal.Domain, raw); err != nil {
		t.Fatal(err)
	}
	if _, err := watcher.Load(context.Background()); err != nil {
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

func (f *fixture) grant(account, key string, source cosmetics.Source, medals ...string) (reward.Report, error) {
	return f.engine.Apply(context.Background(), reward.Grant{Account: account, Bundle: reward.Bundle{ID: "bundle.test_medal", Revision: 1, Medals: medals}, Source: source, Reason: wallet.Mission, Key: key}, now)
}

func TestAMedalIsGrantedOnceByAListedSource(t *testing.T) {
	f := newFixture(t)
	account := f.account(t)
	report, err := f.grant(account, "mission:win_50:1", winsSource, "medal.wins_50")
	if err != nil || len(report.Medals) != 1 {
		t.Fatalf("first grant %+v, %v", report, err)
	}
	report, err = f.grant(account, "mission:win_50:2", winsSource, "medal.wins_50")
	if err != nil || len(report.Medals) != 0 || len(report.MedalsSkipped) != 1 {
		t.Fatalf("second grant %+v, %v", report, err)
	}
}

func TestABrokenConditionNeverGrantsAMedal(t *testing.T) {
	f := newFixture(t)
	account := f.account(t)
	cases := []struct {
		name   string
		source cosmetics.Source
		medal  string
		want   error
	}{
		{"another mission", cosmetics.Source{Type: medal.SourceMission, Ref: "mission.win_5"}, "medal.wins_50", medal.ErrSourceNotListed},
		{"a promo code", cosmetics.Source{Type: "promo", Ref: "campaign.test"}, "medal.wins_50", medal.ErrSourceNotListed},
		{"a retired medal", cosmetics.Source{Type: medal.SourceMission, Ref: "mission.old_event"}, "medal.old_event", medal.ErrSourceNotListed},
		{"a disabled medal", cosmetics.Source{Type: medal.SourceMission, Ref: "mission.broken"}, "medal.broken", medal.ErrSourceNotListed},
		{"an unknown medal", winsSource, "medal.nobody", medal.ErrUnknownMedal},
	}
	for _, c := range cases {
		if _, err := f.grant(account, "key:"+c.name, c.source, c.medal); !errors.Is(err, c.want) {
			t.Fatalf("%s: %v, want %v", c.name, err, c.want)
		}
	}
	collection, err := f.store.Collection(context.Background(), account)
	if err != nil || len(collection.Owned) != 0 {
		t.Fatalf("owned after refusals %+v, %v", collection.Owned, err)
	}
	var grants int
	if err := f.pool.QueryRow(context.Background(), `select count(*) from reward_grants where account_id = $1`, account).Scan(&grants); err != nil || grants != 0 {
		t.Fatalf("reward grants after refusals = %d, %v", grants, err)
	}
}

func TestTheShowcaseHoldsOwnedMedalsAndLosesARevokedOne(t *testing.T) {
	f := newFixture(t)
	account := f.account(t)
	if _, err := f.grant(account, "mission:win_50", winsSource, "medal.wins_50"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.grant(account, "rank:gold", goldSource, "medal.rank_gold"); err != nil {
		t.Fatal(err)
	}
	showcase, err := f.store.SetShowcase(context.Background(), account, []string{"medal.rank_gold", "medal.wins_50"}, now)
	if err != nil || len(showcase) != 2 || showcase[0] != "medal.rank_gold" {
		t.Fatalf("SetShowcase = %v, %v", showcase, err)
	}
	refusals := []struct {
		medals []string
		want   error
	}{
		{[]string{"medal.domino_100"}, medal.ErrNotOwned},
		{[]string{"medal.wins_50", "medal.wins_50"}, medal.ErrShowcase},
		{[]string{"medal.wins_50", "medal.rank_gold", "medal.domino_100", "medal.chain_first"}, medal.ErrShowcase},
	}
	for _, r := range refusals {
		if _, err := f.store.SetShowcase(context.Background(), account, r.medals, now); !errors.Is(err, r.want) {
			t.Fatalf("SetShowcase(%v) = %v", r.medals, err)
		}
	}
	if _, err := f.engine.Revoke(context.Background(), account, "rank:gold", wallet.AdminAdjust, "revoke:gold", now); err != nil {
		t.Fatal(err)
	}
	collection, err := f.store.Collection(context.Background(), account)
	if err != nil {
		t.Fatal(err)
	}
	if len(collection.Showcase) != 1 || collection.Showcase[0] != "medal.wins_50" || len(collection.Owned) != 1 {
		t.Fatalf("collection after revoke %+v", collection)
	}
	if len(collection.Medals) != 4 {
		t.Fatalf("collection shows %d medals, want the 4 live ones", len(collection.Medals))
	}
	f.store.SetMissionTypes(func(id string) string {
		if id == "mission.win_50" {
			return "achievement"
		}
		return ""
	})
	collection, err = f.store.Collection(context.Background(), account)
	if err != nil || collection.Medals[0].Group != medal.GroupAchievement || collection.Medals[1].Group != medal.GroupRankReward || collection.Medals[2].Group != "" {
		t.Fatalf("collection groups %+v %v", collection.Medals, err)
	}
	if public, err := f.store.Public(context.Background(), account); err != nil || len(public) != 1 || public[0].ID != "medal.wins_50" || public[0].RarityID == "" {
		t.Fatalf("public showcase %+v %v", public, err)
	}
}
