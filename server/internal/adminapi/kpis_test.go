package adminapi_test

import (
	"net/http"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func TestKpisCoverWholeEconomyWeeksUpToToday(t *testing.T) {
	a := newAPI(t)
	a.create("orxan@example.com", rbac.Analyst)
	analyst := a.signIn("orxan")

	status, body := a.call("GET", "/admin/analytics/kpis", analyst, nil)
	if status != http.StatusOK || body["from"] != "2026-10-05" || body["to"] != "2026-11-01" || len(body["dailyCohorts"].([]any)) != 27 || len(body["weeklyCohorts"].([]any)) != 4 || len(body["matches"].([]any)) != 54 {
		t.Fatalf("default kpis %d %v", status, body)
	}
	status, body = a.call("GET", "/admin/analytics/kpis?weeks=1", analyst, nil)
	if status != http.StatusOK || body["from"] != "2026-10-26" || len(body["weeklyCohorts"].([]any)) != 1 {
		t.Fatalf("one week %d %v", status, body)
	}
	for _, weeks := range []string{"0", "14", "x"} {
		status, body = a.call("GET", "/admin/analytics/kpis?weeks="+weeks, analyst, nil)
		expect(t, status, body, http.StatusBadRequest, "invalid")
	}
}
