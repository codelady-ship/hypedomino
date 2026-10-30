package rating_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

type rewardFixture struct {
	*fixture
	wallet    *wallet.Store
	inventory *inventory.Store
}

func newRewardFixture(t *testing.T) *rewardFixture {
	t.Helper()
	f := newFixture(t)
	catalog, err := cosmetics.Load("../../../docs/cosmetics/examples/catalog.json")
	if err != nil {
		t.Fatal(err)
	}
	r := &rewardFixture{fixture: f, wallet: wallet.NewStore(f.pool), inventory: inventory.NewStore(f.pool, catalog, inventory.DefaultConfig())}
	f.store.SetRewards(reward.NewEngine(f.pool, r.wallet, r.inventory, nil, nil, reward.DefaultConfig()))
	if _, err := snapshot.Publish(context.Background(), f.pool, rating.Domain, []byte(`{"schema":1,"leaderboardRegions":{"caucasus":["AZ","GE"]},"emblemBrackets":[1,2,100],"modes":{"1v1":{"rankRewards":{"Gold":{"id":"bundle.rank_season_1_gold","revision":1,"cosmetics":["frame.season_1_top"]}}},"2v2":{}}}`)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.watcher.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	return r
}

func (f *rewardFixture) balance(t *testing.T, account string) int64 {
	t.Helper()
	balance, err := f.wallet.Balance(context.Background(), account)
	if err != nil {
		t.Fatal(err)
	}
	return balance
}

func (f *rewardFixture) grants(t *testing.T, account string) int {
	t.Helper()
	var count int
	if err := f.pool.QueryRow(context.Background(), `select count(*) from reward_grants where account_id = $1`, account).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func emblemNames(emblems []rating.Emblem) string {
	names := make([]string, 0, len(emblems))
	for _, e := range emblems {
		names = append(names, e.Season+"/"+e.Mode+"/"+e.String())
	}
	return strings.Join(names, " ")
}

func TestSeasonEndRewardsAndEmblemsAreGrantedOnce(t *testing.T) {
	f := newRewardFixture(t)
	end := now.Add(60 * 24 * time.Hour)
	challenger := f.apex(t, 600, 40, now)
	gold, gold2, silver := f.account(t), f.account(t), f.account(t)
	f.setCountry(t, gold, "AZ")
	f.setCountry(t, gold2, "GE")
	f.setCountry(t, silver, "AZ")
	f.set(t, gold, rules.Mode1v1, 1500, 50, rank(t, "Gold II", 50))
	f.set(t, gold2, rules.Mode1v1, 1450, 50, rank(t, "Gold IV", 20))
	f.set(t, silver, rules.Mode1v1, 1300, 50, rank(t, "Silver I", 0))
	for _, account := range []string{gold, gold2, silver} {
		if _, err := f.pool.Exec(context.Background(), `update ratings set board_country = (select country from accounts where id = $1) where account_id = $1`, account); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.store.DistributeRewards(context.Background(), "season.test", rules.Mode1v1, end); !errors.Is(err, rating.ErrSeasonOpen) {
		t.Fatalf("rewards before the close: %v", err)
	}
	if _, err := f.store.CloseSeason(context.Background(), "season.test", rules.Mode1v1, end); err != nil {
		t.Fatal(err)
	}
	distributed, err := f.store.DistributeRewards(context.Background(), "season.test", rules.Mode1v1, end)
	if err != nil || distributed.Rewarded != 4 || distributed.Skipped {
		t.Fatalf("distributed %+v %v", distributed, err)
	}
	for account, want := range map[string]int64{challenger: 5000, gold: 700, gold2: 700, silver: 400} {
		if got := f.balance(t, account); got != want {
			t.Errorf("balance %d, want %d", got, want)
		}
	}
	if owns, err := f.inventory.Owns(context.Background(), gold, "frame.season_1_top"); err != nil || !owns {
		t.Fatalf("gold frame %v %v", owns, err)
	}
	if owns, _ := f.inventory.Owns(context.Background(), silver, "frame.season_1_top"); owns {
		t.Fatal("silver got the gold frame")
	}
	emblems, err := f.store.Emblems(context.Background(), gold)
	if err != nil || emblemNames(emblems) != "season.test/1v1/country:AZ:1 season.test/1v1/global:2 season.test/1v1/region:caucasus:1" {
		t.Fatalf("gold emblems %s %v", emblemNames(emblems), err)
	}
	if emblems, _ := f.store.Emblems(context.Background(), silver); emblemNames(emblems) != "season.test/1v1/country:AZ:2 season.test/1v1/global:100 season.test/1v1/region:caucasus:100" {
		t.Fatalf("silver emblems %s", emblemNames(emblems))
	}
	if emblems, _ := f.store.Emblems(context.Background(), challenger); emblemNames(emblems) != "season.test/1v1/global:1" {
		t.Fatalf("challenger emblems %s", emblemNames(emblems))
	}
	again, err := f.store.DistributeRewards(context.Background(), "season.test", rules.Mode1v1, end.Add(time.Hour))
	if err != nil || !again.Skipped || f.balance(t, gold) != 700 || f.grants(t, gold) != 1 {
		t.Fatalf("second run %+v %v, balance %d, grants %d", again, err, f.balance(t, gold), f.grants(t, gold))
	}
	if _, err := f.pool.Exec(context.Background(), `update ranked_seasons set rewarded_1v1_at = null where id = 'season.test'`); err != nil {
		t.Fatal(err)
	}
	if rerun, err := f.store.DistributeRewards(context.Background(), "season.test", rules.Mode1v1, end.Add(time.Hour)); err != nil || rerun.Rewarded != 0 || f.balance(t, gold) != 700 {
		t.Fatalf("rerun after a lost mark %+v %v", rerun, err)
	}
	rewards, err := f.store.Rewards(context.Background(), gold)
	if err != nil || len(rewards) != 1 || rewards[0].Coins != 700 || rewards[0].BundleID != "bundle.rank_season_1_gold" || rewards[0].Tier != "Gold" || rewards[0].Seen || strings.Join(rewards[0].Emblems, " ") != "global:2 country:AZ:1 region:caucasus:1" {
		t.Fatalf("gold rewards %+v %v", rewards, err)
	}
	if seen, err := f.store.MarkRewardsSeen(context.Background(), gold, end); err != nil || seen != 1 {
		t.Fatalf("seen %d %v", seen, err)
	}
	if seen, _ := f.store.MarkRewardsSeen(context.Background(), gold, end); seen != 0 {
		t.Fatalf("seen twice %d", seen)
	}
	if rewards, _ := f.store.Rewards(context.Background(), gold); !rewards[0].Seen {
		t.Fatal("reward still unseen")
	}
	if due, err := f.store.DistributeDue(context.Background(), end); err != nil || len(due) != 0 {
		t.Fatalf("due after everything %+v %v", due, err)
	}
}

func TestASeasonWithoutRewardsGrantsNothing(t *testing.T) {
	f := newRewardFixture(t)
	end := now.Add(60 * 24 * time.Hour)
	gold := f.account(t)
	f.set(t, gold, rules.Mode1v1, 1500, 50, rank(t, "Gold II", 50))
	if _, err := f.pool.Exec(context.Background(), `update ranked_seasons set rewards = false where id = 'season.test'`); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []rules.Mode{rules.Mode1v1, rules.Mode2v2} {
		if _, err := f.store.CloseSeason(context.Background(), "season.test", mode, end); err != nil {
			t.Fatal(err)
		}
	}
	due, err := f.store.DistributeDue(context.Background(), end)
	if err != nil || len(due) != 2 || due[0].Rewarded != 0 || due[1].Rewarded != 0 {
		t.Fatalf("due %+v %v", due, err)
	}
	if f.balance(t, gold) != 0 || f.grants(t, gold) != 0 {
		t.Fatalf("beta season paid out: balance %d", f.balance(t, gold))
	}
	seasons, err := f.store.Seasons(context.Background())
	if err != nil || !seasons[0].NoRewards || !seasons[0].RewardedFor(rules.Mode1v1) || !seasons[0].RewardedFor(rules.Mode2v2) || seasons[1].NoRewards {
		t.Fatalf("seasons %+v %v", seasons, err)
	}
	if _, err := f.store.CreateSeason(context.Background(), rating.Season{ID: "season.beta", Start: seasons[1].End, End: seasons[1].End.AddDate(0, 1, 0), NoRewards: true}, end); err != nil {
		t.Fatal(err)
	}
	if seasons, _ := f.store.Seasons(context.Background()); !seasons[2].NoRewards {
		t.Fatalf("created without rewards %+v", seasons[2])
	}
}

type fakeVerifier struct{}

func (fakeVerifier) Verify(access string) (token.Claims, error) {
	switch {
	case access == "expired":
		return token.Claims{}, token.ErrExpired
	case strings.HasPrefix(access, "access:"):
		return token.Claims{Account: strings.TrimPrefix(access, "access:")}, nil
	}
	return token.Claims{}, token.ErrMalformed
}

func TestTheRankedRoutesServeRewardsAndEmblems(t *testing.T) {
	f := newRewardFixture(t)
	end := now.Add(60 * 24 * time.Hour)
	gold := f.account(t)
	f.set(t, gold, rules.Mode1v1, 1500, 50, rank(t, "Gold II", 50))
	if _, err := f.store.CloseSeason(context.Background(), "season.test", rules.Mode1v1, end); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.DistributeRewards(context.Background(), "season.test", rules.Mode1v1, end); err != nil {
		t.Fatal(err)
	}
	handler := rating.NewHandler(fakeVerifier{}, f.store, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return end })
	call := func(method, path, access string) (int, map[string]any) {
		t.Helper()
		request := httptest.NewRequest(method, path, nil)
		request.Header.Set("Authorization", "Bearer "+access)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		var body map[string]any
		json.Unmarshal(recorder.Body.Bytes(), &body)
		return recorder.Code, body
	}
	status, body := call(http.MethodGet, "/ranked/rewards", "access:"+gold)
	rewards, _ := body["rewards"].([]any)
	if status != http.StatusOK || body["unseen"] != float64(1) || len(rewards) != 1 {
		t.Fatalf("rewards %d %v", status, body)
	}
	first := rewards[0].(map[string]any)
	if first["coins"] != float64(700) || first["tier"] != "Gold" || first["bundle_id"] != "bundle.rank_season_1_gold" || first["seen"] != false || first["season"] != "season.test" {
		t.Fatalf("reward row %v", first)
	}
	if status, body := call(http.MethodPost, "/ranked/rewards/seen", "access:"+gold); status != http.StatusOK || body["seen"] != float64(1) {
		t.Fatalf("seen %d %v", status, body)
	}
	if status, body := call(http.MethodGet, "/ranked/rewards", "access:"+gold); status != http.StatusOK || body["unseen"] != float64(0) {
		t.Fatalf("after seen %d %v", status, body)
	}
	status, body = call(http.MethodGet, "/ranked/emblems", "access:"+gold)
	emblems, _ := body["emblems"].([]any)
	if status != http.StatusOK || len(emblems) != 1 || emblems[0].(map[string]any)["board"] != "global" || emblems[0].(map[string]any)["bracket"] != float64(1) {
		t.Fatalf("emblems %d %v", status, body)
	}
	if status, body := call(http.MethodGet, "/ranked/rewards", "access:"+f.account(t)); status != http.StatusOK || body["unseen"] != float64(0) || len(body["rewards"].([]any)) != 0 {
		t.Fatalf("empty mailbox %d %v", status, body)
	}
	if status, body := call(http.MethodGet, "/ranked/rewards", "expired"); status != http.StatusUnauthorized || body["error"] != "access_expired" {
		t.Fatalf("expired %d %v", status, body)
	}
}
