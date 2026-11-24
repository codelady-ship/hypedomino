package adminapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

func TestThePlayerRecordShowsThePassLevelAndXpOfTheActiveSeason(t *testing.T) {
	a := newAPI(t)
	a.create("orxan@example.com", rbac.Support)
	support := a.signIn("orxan")
	id := a.player("g-1", "kamal@mail.example", "Kamal")
	level := func() (any, any) {
		t.Helper()
		status, body := a.call("GET", "/admin/players/"+id, support, nil)
		if status != http.StatusOK {
			t.Fatalf("record %d %v", status, body)
		}
		profile := body["profile"].(map[string]any)
		return profile["level"], profile["xp"]
	}
	if got, xp := level(); got != float64(1) || xp != float64(0) {
		t.Fatalf("without a pass season level %v xp %v", got, xp)
	}

	ctx := context.Background()
	config := pass.Default()
	config.PremiumPlusLevels = 0
	season := pass.Season{
		ID: "season_1", Start: start.Add(-24 * time.Hour), End: start.Add(30 * 24 * time.Hour),
		Curve: pass.Curve{Levels: 3, XpBase: 100, XpStep: 1, XpMax: 100},
		Free:  map[int]reward.Bundle{}, Premium: map[int]reward.Bundle{},
	}
	for l := 1; l <= 3; l++ {
		season.Premium[l] = reward.Bundle{ID: "bundle.test_premium", Revision: l, Crystals: 1}
	}
	config.Seasons = []pass.Season{season}
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot.Publish(ctx, a.pool, pass.Domain, raw); err != nil {
		t.Fatal(err)
	}
	if got, xp := level(); got != float64(1) || xp != float64(0) {
		t.Fatalf("without progress level %v xp %v", got, xp)
	}
	total := season.Curve.XpToReach(3) + 5
	if _, err := a.pool.Exec(ctx, `insert into pass_progress (account_id, season_id, xp_total, entered_at) values ($1, 'season_1', $2, $3)`, id, total, start); err != nil {
		t.Fatal(err)
	}
	if got, xp := level(); got != float64(3) || xp != float64(total) {
		t.Fatalf("with progress level %v xp %v", got, xp)
	}
}
