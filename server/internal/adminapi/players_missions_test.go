package adminapi_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/mission"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

func TestThePlayerRecordShowsMissionAssignmentsWithTheirPublishedTarget(t *testing.T) {
	a := newAPI(t)
	a.create("orxan@example.com", rbac.Support)
	support := a.signIn("orxan")
	id := a.player("g-1", "kamal@mail.example", "Kamal")
	missions := func() []any {
		t.Helper()
		status, body := a.call("GET", "/admin/players/"+id, support, nil)
		if status != http.StatusOK {
			t.Fatalf("record %d %v", status, body)
		}
		return body["missions"].([]any)
	}
	if got := missions(); len(got) != 0 {
		t.Fatalf("without assignments %v", got)
	}

	ctx := context.Background()
	assigned := start.Add(time.Hour)
	if _, err := a.pool.Exec(ctx, `insert into mission_assignments (account_id, mission_id, period, mission_type, slot, progress, assigned_at, completed_at)
		values ($1, 'mission.daily_points_30', '2026-11-01', 'daily', 1, 30, $2, $2)`, id, assigned); err != nil {
		t.Fatal(err)
	}
	if _, err := a.pool.Exec(ctx, `insert into mission_assignments (account_id, mission_id, period, mission_type, slot, progress, assigned_at, replaced_at)
		values ($1, 'mission.dropped', '2026-11-01', 'daily', 2, 4, $2, $3)`, id, start, assigned); err != nil {
		t.Fatal(err)
	}
	rows := missions()
	if len(rows) != 2 {
		t.Fatalf("assignments %v", rows)
	}
	first, second := rows[0].(map[string]any), rows[1].(map[string]any)
	if first["mission_id"] != "mission.daily_points_30" || first["type"] != "daily" || first["slot"] != float64(1) || first["period"] != "2026-11-01" ||
		first["progress"] != float64(30) || first["target"] != float64(0) || first["completed_at"] == nil || first["replaced_at"] != nil {
		t.Fatalf("newest assignment without a catalog %v", first)
	}
	if second["mission_id"] != "mission.dropped" || second["replaced_at"] == nil || second["completed_at"] != nil {
		t.Fatalf("rerolled assignment %v", second)
	}

	raw, err := os.ReadFile("../../../docs/pass/examples/missions.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := snapshot.Publish(ctx, a.pool, mission.Domain, raw); err != nil {
		t.Fatal(err)
	}
	rows = missions()
	if rows[0].(map[string]any)["target"] != float64(30) || rows[1].(map[string]any)["target"] != float64(0) {
		t.Fatalf("targets from the published catalog %v", rows)
	}
}
