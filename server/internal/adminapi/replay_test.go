package adminapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/replay"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

const replayedMatch = "5f0c7a52-3d1e-4c8b-9a61-2b7e4d9f0a13"

func openingMoves(t *testing.T, seed uint64) []byte {
	t.Helper()
	recorder := replay.NewRecorder(rules.Mode1v1, rules.DefaultConfig(), seed, [32]byte{})
	opening, forced := recorder.ForcedOpening()
	if !forced {
		t.Fatal("the first hand has no forced opening")
	}
	if _, err := recorder.Open(opening.Seat, opening.Tile); err != nil {
		t.Fatal(err)
	}
	var buffer [64]rules.Move
	for range 3 {
		seat := recorder.Current()
		moves := recorder.LegalMoves(seat, buffer[:0])
		if len(moves) == 0 {
			if _, err := recorder.Draw(seat); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if _, err := recorder.Place(seat, moves[0].Tile, moves[0].End); err != nil {
			t.Fatal(err)
		}
		if recorder.Current() == seat && recorder.Turn().Placements() > 0 {
			if err := recorder.EndTurn(seat); err != nil {
				t.Fatal(err)
			}
		}
	}
	record := recorder.Record()
	raw, err := record.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func (a *api) storeMatch(id string, accountID string, raw []byte) {
	a.t.Helper()
	ctx := context.Background()
	started := start.Add(-time.Hour)
	if _, err := a.pool.Exec(ctx, `insert into matches (id, started_at, ended_at, mode, kind, config_version, seed, hands_played, score_a, score_b, winner_team, replay)
		values ($1, $2, $3, '1v1', 'ranked', 4, 11, 1, 150, 90, 0, $4)`, id, started, started.Add(20*time.Minute), raw); err != nil {
		a.t.Fatal(err)
	}
	if _, err := a.pool.Exec(ctx, `insert into match_participants (match_id, started_at, seat, team, account_id, bot, result) values
		($1, $2, 4, 0, $3, false, 'win'), ($1, $2, 5, 1, null, true, 'loss')`, id, started, accountID); err != nil {
		a.t.Fatal(err)
	}
}

func TestSupportOpensAMatchReplayAndTheViewIsAudited(t *testing.T) {
	a := newAPI(t)
	a.create("orxan@example.com", rbac.Support)
	support, content, root := a.signIn("orxan"), a.signIn("ayla"), a.signIn("root")
	id := a.player("g-1", "kamal@mail.example", "Kamal")
	a.storeMatch(replayedMatch, id, openingMoves(t, 11))
	path := "/admin/matches/" + replayedMatch + "/replay"

	status, body := a.call("GET", path, content, nil)
	expect(t, status, body, http.StatusForbidden, "forbidden")
	for _, missing := range []string{"not-a-uuid", "00000000-0000-0000-0000-000000000000"} {
		status, body = a.call("GET", "/admin/matches/"+missing+"/replay", support, nil)
		expect(t, status, body, http.StatusNotFound, "not_found")
	}

	status, body = a.call("GET", path, support, nil)
	if status != http.StatusOK || body["mode"] != "1v1" || body["winner_team"] != "A" || body["score_a"] != float64(150) {
		t.Fatalf("replay %d %v", status, body)
	}
	participants := body["participants"].([]any)
	human, bot := participants[0].(map[string]any), participants[1].(map[string]any)
	if len(participants) != 2 || human["seat"] != "X" || human["account_id"] != id || human["nickname"] != "Kamal" || bot["bot"] != true || bot["account_id"] != nil {
		t.Fatalf("participants %v", participants)
	}
	timeline := body["timeline"].(map[string]any)
	hands := timeline["hands"].([]any)
	first := hands[0].(map[string]any)
	steps := first["steps"].([]any)
	if timeline["problem"] != nil || len(hands) != 1 || len(first["dealt"].(map[string]any)) != 2 || first["outcome"] != nil || len(steps) < 4 || steps[0].(map[string]any)["kind"] != "open" {
		t.Fatalf("timeline %v", timeline)
	}

	if _, err := a.pool.Exec(context.Background(), `update matches set replay = $2 where id = $1`, replayedMatch, []byte("HDR broken")); err != nil {
		t.Fatal(err)
	}
	status, body = a.call("GET", path, support, nil)
	if timeline := body["timeline"].(map[string]any); status != http.StatusOK || timeline["problem"] == nil || len(timeline["hands"].([]any)) != 0 {
		t.Fatalf("broken %d %v", status, body)
	}
	if _, err := a.pool.Exec(context.Background(), `update matches set replay = null where id = $1`, replayedMatch); err != nil {
		t.Fatal(err)
	}
	status, body = a.call("GET", path, support, nil)
	if status != http.StatusOK || body["timeline"] != nil {
		t.Fatalf("no replay %d %v", status, body)
	}

	entries := a.entries(root, "object_type=players&object_id="+replayedMatch+"&action=view_replay")
	if len(entries) != 3 {
		t.Fatalf("entries %v", entries)
	}
}
