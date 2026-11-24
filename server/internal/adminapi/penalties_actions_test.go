package adminapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/penalties"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestSupportLiftsAPenaltyAndRemovesAnOffenseWithAReason(t *testing.T) {
	a := newAPI(t)
	a.now = time.Now().UTC()
	a.create("orxan@example.com", rbac.Support)
	support, root := a.signIn("orxan"), a.signIn("root")
	id := a.player("g-1", "kamal@mail.example", "Kamal")
	base := "/admin/players/" + id + "/penalties/"

	status, body := a.call("POST", base+"lift", support, map[string]any{"reason": "appeal"})
	expect(t, status, body, http.StatusConflict, "nothing_to_lift")

	store := penalties.NewStore(a.pool, func() (penalties.Config, int) { return penalties.DefaultConfig(), 0 })
	leave := rules.PresenceReport{Left: true, ByBot: true}
	for _, match := range []string{"m1", "m2"} {
		if _, err := store.MatchEnd(context.Background(), id, penalties.DefaultConfig(), matchmaking.QueueRanked, match, rules.SeatX, leave, 0, a.now.Add(-time.Minute)); err != nil {
			t.Fatal(err)
		}
	}

	status, body = a.call("GET", "/admin/players/"+id+"/penalties", support, nil)
	if ranked := body["ranked"].(map[string]any); status != http.StatusOK || ranked["step"] != "ban" || len(ranked["offenses"].([]any)) != 2 {
		t.Fatalf("view %d %v", status, body)
	}
	status, body = a.call("POST", base+"remove", support, map[string]any{"queue": "ranked", "match_id": "m9", "reason": "network outage"})
	expect(t, status, body, http.StatusNotFound, "unknown_offense")
	status, body = a.call("POST", base+"remove", support, map[string]any{"queue": "arena", "match_id": "m1", "reason": "network outage"})
	expect(t, status, body, http.StatusBadRequest, "invalid")
	status, body = a.call("POST", base+"lift", support, map[string]any{})
	expect(t, status, body, http.StatusBadRequest, "reason_required")

	status, body = a.call("POST", base+"lift", support, map[string]any{"reason": "appeal upheld", "ref": "APP-1"})
	if status != http.StatusOK {
		t.Fatalf("lift %d %v", status, body)
	}
	status, body = a.call("POST", base+"remove", support, map[string]any{"queue": "ranked", "match_id": "m1", "reason": "network outage"})
	if status != http.StatusOK {
		t.Fatalf("remove %d %v", status, body)
	}
	record, err := store.Record(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if _, banned := record.Banned(a.now); banned || len(record.Ranked.Offenses) != 1 || record.Ranked.Offenses[0].MatchID != "m2" {
		t.Fatalf("record %+v", record.Ranked)
	}
	lifted := a.entries(root, "object_type=players&action="+adminapi.LiftPenaltyAction)
	removed := a.entries(root, "object_type=players&action="+adminapi.RemoveOffenseAction)
	if len(lifted) != 1 || lifted[0]["ref"] != "APP-1" || len(removed) != 1 {
		t.Fatalf("entries %v %v", lifted, removed)
	}
}
