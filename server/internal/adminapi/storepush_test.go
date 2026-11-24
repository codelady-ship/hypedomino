package adminapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/storeprice"
	"github.com/Alievs-corp/hypedomino/server/internal/storesync"
)

type fakeStore struct {
	pushed map[string]bool
	writes int
}

func (f *fakeStore) Sync(_ context.Context, targets []storeprice.Target, write bool) []storesync.Row {
	rows := []storesync.Row{}
	for _, target := range targets {
		key := target.Product + "/" + target.Country
		row := storesync.Row{Product: target.Product, Country: target.Country, TargetCents: target.Cents, Status: storesync.StatusChange}
		switch {
		case f.pushed[key]:
			row.Status = storesync.StatusUnchanged
		case write:
			f.pushed[key], row.Status = true, storesync.StatusWritten
			f.writes++
		}
		rows = append(rows, row)
	}
	return rows
}

func storeReport(t *testing.T, body map[string]any, store string) map[string]any {
	t.Helper()
	for _, report := range body["stores"].([]any) {
		if report.(map[string]any)["store"] == store {
			return report.(map[string]any)
		}
	}
	t.Fatalf("no %s report in %v", store, body)
	return nil
}

func TestStorePricesArePlannedByViewersAndPushedOnlyBySuperAdmins(t *testing.T) {
	play := &fakeStore{pushed: map[string]bool{}}
	a := newAPIWith(t, func(config *adminapi.Config) { config.Stores = storesync.Stores{GooglePlay: play} })
	a.create("leyla@example.com", rbac.LiveOps)
	a.create("vusal@example.com", rbac.Support)
	root, liveOps, support, content := a.signIn("root"), a.signIn("leyla"), a.signIn("vusal"), a.signIn("ayla")

	status, body := a.call("POST", "/admin/config/store-prices/sync/plan", support, nil)
	expect(t, status, body, http.StatusConflict, "not_published")
	status, body = a.call("PUT", "/admin/config/store-prices", liveOps, map[string]any{"catalog": exampleStorePrices(t), "reason": "launch"})
	expect(t, status, body, http.StatusOK, "")
	version := body["config_version"]
	targets := len(storeprice.Default().Targets())

	status, body = a.call("POST", "/admin/config/store-prices/sync/plan", content, nil)
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("POST", "/admin/config/store-prices/sync/plan", support, nil)
	expect(t, status, body, http.StatusOK, "")
	google, apple := storeReport(t, body, storesync.GooglePlay), storeReport(t, body, storesync.AppStore)
	if body["config_version"] != version || google["configured"] != true || len(google["rows"].([]any)) != targets || google["counts"].(map[string]any)["change"] != float64(targets) {
		t.Fatalf("plan %v", google["counts"])
	}
	if apple["configured"] != false || len(apple["rows"].([]any)) != 0 {
		t.Fatalf("App Store without credentials: %v", apple)
	}
	if play.writes != 0 {
		t.Fatalf("a plan wrote %d prices", play.writes)
	}

	status, body = a.call("POST", "/admin/config/store-prices/sync/apply", liveOps, map[string]any{"reason": "push"})
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("POST", "/admin/config/store-prices/sync/apply", support, map[string]any{"reason": "push"})
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("POST", "/admin/config/store-prices/sync/apply", root, map[string]any{"reason": " "})
	expect(t, status, body, http.StatusBadRequest, "reason_required")
	if play.writes != 0 {
		t.Fatalf("a refused apply wrote %d prices", play.writes)
	}

	status, body = a.call("POST", "/admin/config/store-prices/sync/apply", root, map[string]any{"reason": "launch prices", "ref": "#585"})
	expect(t, status, body, http.StatusOK, "")
	if counts := storeReport(t, body, storesync.GooglePlay)["counts"].(map[string]any); counts["written"] != float64(targets) || play.writes != targets {
		t.Fatalf("apply %v, %d writes", counts, play.writes)
	}

	status, body = a.call("POST", "/admin/config/store-prices/sync/apply", root, map[string]any{"reason": "again"})
	expect(t, status, body, http.StatusOK, "")
	if counts := storeReport(t, body, storesync.GooglePlay)["counts"].(map[string]any); counts["unchanged"] != float64(targets) || play.writes != targets {
		t.Fatalf("second apply %v, %d writes", counts, play.writes)
	}

	status, body = a.call("GET", "/admin/audit?object_type=store_products&action=push", root, nil)
	expect(t, status, body, http.StatusOK, "")
	entries := body["entries"].([]any)
	if len(entries) != 2 {
		t.Fatalf("one entry per configured store and run: %v", entries)
	}
	first := entries[1].(map[string]any)
	var counts map[string]int
	raw, _ := json.Marshal(first["after"])
	json.Unmarshal(raw, &counts)
	if first["object_id"] != storesync.GooglePlay || first["config_version"] != version || first["reason"] != "launch prices" || first["ref"] != "#585" || counts["written"] != targets {
		t.Fatalf("audit %v", first)
	}

	a.now = start.Add(11 * time.Minute)
	status, body = a.call("POST", "/admin/config/store-prices/sync/apply", root, map[string]any{"reason": "late"})
	expect(t, status, body, http.StatusForbidden, "step_up_required")
}
