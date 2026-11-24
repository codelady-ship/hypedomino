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

	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestStandingOfShowsRankAndLpOnlyOncePlaced(t *testing.T) {
	config := rating.DefaultConfig()
	cases := []struct {
		name   string
		rating rating.Rating
		want   rating.ModeStanding
	}{
		{"no rating", rating.Rating{}, rating.ModeStanding{Mode: "1v1", PlacementMatches: 5, DivisionLP: 100}},
		{"in placement at a reset rank", rating.Rating{PlacementPlayed: 2, Wins: 1, Rank: rank(t, "Silver II", 0), Tier: "Silver"}, rating.ModeStanding{Mode: "1v1", PlacementPlayed: 2, PlacementMatches: 5, DivisionLP: 100, Wins: 1}},
		{"placed", rating.Rating{Placed: true, PlacementPlayed: 5, Wins: 9, Rank: rank(t, "Gold II", 42), Tier: "Gold"}, rating.ModeStanding{Mode: "1v1", Placed: true, PlacementPlayed: 5, PlacementMatches: 5, Rank: "Gold II", Tier: "Gold", LP: 42, DivisionLP: 100, Wins: 9}},
		{"apex", rating.Rating{Placed: true, PlacementPlayed: 5, Wins: 80, Rank: rating.Rank{Index: rating.ApexIndex, LP: 340}, Tier: rating.TierGrandmaster}, rating.ModeStanding{Mode: "1v1", Placed: true, PlacementPlayed: 5, PlacementMatches: 5, Rank: "Grandmaster", Tier: "Grandmaster", LP: 340, DivisionLP: 100, Wins: 80}},
	}
	for _, c := range cases {
		if got := rating.StandingOf(rules.Mode1v1, config, c.rating); got != c.want {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
}

type standingMailbox struct {
	rating.Mailbox
	standing rating.Standing
	err      error
	asked    time.Time
	account  string
}

func (m *standingMailbox) Standing(_ context.Context, account string, now time.Time) (rating.Standing, error) {
	m.account, m.asked = account, now
	return m.standing, m.err
}

func callStanding(t *testing.T, mailbox rating.Mailbox, access string) (int, string) {
	t.Helper()
	handler := rating.NewHandler(fakeVerifier{}, mailbox, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return now })
	request := httptest.NewRequest(http.MethodGet, "/ranked/me", nil)
	request.Header.Set("Authorization", "Bearer "+access)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder.Code, recorder.Body.String()
}

func TestTheStandingRouteAnswersTheViewersOwnRanks(t *testing.T) {
	mailbox := &standingMailbox{standing: rating.Standing{
		Season: rating.SeasonClock{ID: "season.test", NameKey: "season.test.name", StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour)},
		Modes:  []rating.ModeStanding{rating.StandingOf(rules.Mode1v1, rating.DefaultConfig(), rating.Rating{Placed: true, MMR: 1500, Rank: rank(t, "Gold II", 42), Tier: "Gold"})},
	}}
	status, body := callStanding(t, mailbox, "access:acc-1")
	if status != http.StatusOK || mailbox.account != "acc-1" || !mailbox.asked.Equal(now) {
		t.Fatalf("standing %d %s %s %v", status, body, mailbox.account, mailbox.asked)
	}
	if strings.Contains(strings.ToLower(body), "mmr") || strings.Contains(body, "1500") {
		t.Fatalf("the standing carries MMR: %s", body)
	}
	var reply struct {
		Season struct {
			ID      string `json:"id"`
			NameKey string `json:"name_key"`
			EndsAt  string `json:"ends_at"`
		} `json:"season"`
		Modes []map[string]any `json:"modes"`
	}
	if err := json.Unmarshal([]byte(body), &reply); err != nil {
		t.Fatal(err)
	}
	if reply.Season.ID != "season.test" || reply.Season.NameKey != "season.test.name" || reply.Season.EndsAt != now.Add(time.Hour).Format(time.RFC3339) {
		t.Fatalf("season %+v", reply.Season)
	}
	if len(reply.Modes) != 1 || reply.Modes[0]["rank"] != "Gold II" || reply.Modes[0]["lp"] != float64(42) || reply.Modes[0]["division_lp"] != float64(100) || reply.Modes[0]["placed"] != true {
		t.Fatalf("modes %v", reply.Modes)
	}
}

func TestTheStandingRouteRefusesWithoutASeasonOrAccess(t *testing.T) {
	if status, body := callStanding(t, &standingMailbox{err: rating.ErrNoSeason}, "access:acc-1"); status != http.StatusNotFound || !strings.Contains(body, `"rating.season_unknown"`) {
		t.Fatalf("no season %d %s", status, body)
	}
	if status, body := callStanding(t, &standingMailbox{err: errors.New("down")}, "access:acc-1"); status != http.StatusInternalServerError || !strings.Contains(body, `"internal"`) {
		t.Fatalf("failure %d %s", status, body)
	}
	if status, body := callStanding(t, &standingMailbox{}, "expired"); status != http.StatusUnauthorized || !strings.Contains(body, `"access_expired"`) {
		t.Fatalf("expired %d %s", status, body)
	}
	if status, body := callStanding(t, &standingMailbox{}, "nope"); status != http.StatusUnauthorized || !strings.Contains(body, `"access_rejected"`) {
		t.Fatalf("rejected %d %s", status, body)
	}
}

func TestTheStandingReadsTheCurrentSeasonRows(t *testing.T) {
	f := newFixture(t)
	placed, fresh, stale := f.account(t), f.account(t), f.account(t)
	f.set(t, placed, rules.Mode1v1, 1500, 50, rank(t, "Gold II", 42))
	if _, err := f.pool.Exec(context.Background(), `update ratings set wins = 7 where account_id = $1`, placed); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(context.Background(), `insert into ratings (account_id, mode, mmr, matches, wins, placement_played, placed, rank_index, lp, season_id, updated_at) values ($1, '2v2', 1300, 2, 1, 2, false, 10, 0, 'season.test', $2)`, placed, now); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(context.Background(), `insert into ranked_seasons (id, name_key, starts_at, ends_at, created_at) values ('season.old', 'season.old.name', $1, $2, $1)`, now.Add(-200*24*time.Hour), now.Add(-30*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(context.Background(), `insert into ratings (account_id, mode, mmr, matches, wins, placement_played, placed, rank_index, lp, season_id, updated_at) values ($1, '1v1', 1600, 60, 30, 5, true, 20, 55, 'season.old', $2)`, stale, now); err != nil {
		t.Fatal(err)
	}
	standing, err := f.store.Standing(context.Background(), placed, now)
	if err != nil {
		t.Fatal(err)
	}
	if standing.Season.ID != "season.test" || standing.Season.NameKey != "season.test.name" || !standing.Season.EndsAt.Equal(now.Add(60*24*time.Hour)) || len(standing.Modes) != 2 {
		t.Fatalf("standing %+v", standing)
	}
	want1v1 := rating.ModeStanding{Mode: "1v1", Placed: true, PlacementPlayed: 5, PlacementMatches: 5, Rank: "Gold II", Tier: "Gold", LP: 42, DivisionLP: 100, Wins: 7}
	want2v2 := rating.ModeStanding{Mode: "2v2", PlacementPlayed: 2, PlacementMatches: 5, DivisionLP: 100, Wins: 1}
	if standing.Modes[0] != want1v1 || standing.Modes[1] != want2v2 {
		t.Fatalf("modes %+v", standing.Modes)
	}
	for _, account := range []string{fresh, stale} {
		standing, err := f.store.Standing(context.Background(), account, now)
		if err != nil || standing.Modes[0] != (rating.ModeStanding{Mode: "1v1", PlacementMatches: 5, DivisionLP: 100}) {
			t.Fatalf("unplaced %s %+v %v", account, standing.Modes, err)
		}
	}
	if _, err := f.store.Standing(context.Background(), placed, now.Add(365*24*time.Hour)); !errors.Is(err, rating.ErrNoSeason) {
		t.Fatalf("no season %v", err)
	}
}
