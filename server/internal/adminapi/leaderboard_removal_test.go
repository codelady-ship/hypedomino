package adminapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func TestSupportRemovesAPlayerFromOneLeaderboardWithAnAuditedReason(t *testing.T) {
	a := newAPI(t)
	a.now = time.Now().UTC()
	a.create("orxan@example.com", rbac.Support)
	support, content, root := a.signIn("orxan"), a.signIn("ayla"), a.signIn("root")
	id := a.player("g-1", "kamal@mail.example", "Kamal")
	if _, err := a.pool.Exec(context.Background(), `insert into ranked_seasons (id, name_key, starts_at, ends_at, rewards, created_at) values ('season.one', 'season.one.name', $1, $2, true, $1)`, a.now.Add(-time.Hour), a.now.Add(48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := a.pool.Exec(context.Background(), `insert into ratings (account_id, mode, season_id, mmr, matches, placement_played, placed, rank_index, lp, board_country, updated_at) values ($1, '1v1', 'season.one', 1500, 30, 5, true, 10, 40, 'AZ', $2)`, id, a.now); err != nil {
		t.Fatal(err)
	}
	var dropped []rating.BoardRemoval
	a.handler.(*adminapi.Handler).OnLeaderboardRemoval(func(_ context.Context, removal rating.BoardRemoval) error {
		dropped = append(dropped, removal)
		return nil
	})
	path := "/admin/players/" + id + "/leaderboard-removal"
	request := map[string]any{"season": "season.one", "mode": "1v1", "reason": "offensive nickname on the top 10", "ref": "TCK-4"}

	status, body := a.call("POST", path, support, map[string]any{"season": "season.one", "mode": "1v1"})
	expect(t, status, body, http.StatusBadRequest, "reason_required")
	status, body = a.call("POST", path, content, request)
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("POST", path, support, map[string]any{"season": "season.one", "mode": "3v3", "reason": "x"})
	expect(t, status, body, http.StatusBadRequest, "invalid")
	status, body = a.call("POST", path, support, map[string]any{"season": "season.one", "mode": "2v2", "reason": "x"})
	expect(t, status, body, http.StatusNotFound, "not_on_board")

	status, body = a.call("POST", path, support, request)
	if status != http.StatusOK || body["season"] != "season.one" || body["mode"] != "1v1" || body["account_id"] != id || body["country"] != "AZ" {
		t.Fatalf("removal %d %v", status, body)
	}
	if len(dropped) != 1 || dropped[0].Account != id || dropped[0].Country != "AZ" {
		t.Fatalf("dropped %+v", dropped)
	}
	status, body = a.call("POST", path, support, request)
	expect(t, status, body, http.StatusConflict, "already_removed")

	entries := a.entries(root, "object_type=players&action="+adminapi.RemoveFromLeaderboardAction)
	if len(entries) != 1 || entries[0]["object_id"] != id || entries[0]["reason"] != "offensive nickname on the top 10" || entries[0]["ref"] != "TCK-4" {
		t.Fatalf("entries %v", entries)
	}
}
