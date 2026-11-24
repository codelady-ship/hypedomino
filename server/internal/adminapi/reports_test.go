package adminapi_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/report"
)

func (a *api) fileReport(reporter, target, reason string, at time.Time) {
	a.t.Helper()
	if _, err := a.pool.Exec(context.Background(), `insert into player_reports (reporter, target, reason, name_text, created_at, updated_at) values ($1, $2, $3, 'Rude', $4, $4)`, reporter, target, reason, at); err != nil {
		a.t.Fatal(err)
	}
}

func listedTargets(a *api, token, query string) []string {
	a.t.Helper()
	status, body := a.call("GET", "/admin/reports"+query, token, nil)
	if status != http.StatusOK {
		a.t.Fatalf("reports %d %v", status, body)
	}
	var targets []string
	for _, target := range body["targets"].([]any) {
		targets = append(targets, target.(map[string]any)["account_id"].(string))
	}
	return targets
}

func TestTheReportQueuePagesFiltersByReasonAndSearchesByAccount(t *testing.T) {
	a := newAPI(t)
	a.now = time.Now().UTC()
	a.create("orxan@example.com", rbac.Support)
	support := a.signIn("orxan")
	reporter := a.player("g-0", "reporter@mail.example", "Reporter")
	oldest := a.player("g-1", "oldest@mail.example", "Oldest")
	middle := a.player("g-2", "middle@mail.example", "Middle")
	newest := a.player("g-3", "newest@mail.example", "Newest")
	a.fileReport(reporter, oldest, report.ReasonNickname, a.now.Add(-3*time.Hour))
	a.fileReport(reporter, middle, report.ReasonImpersonation, a.now.Add(-2*time.Hour))
	a.fileReport(reporter, newest, report.ReasonNickname, a.now.Add(-time.Hour))

	page := listedTargets(a, support, "?limit=2")
	if strings.Join(page, ",") != oldest+","+middle {
		t.Fatalf("page %v", page)
	}
	status, body := a.call("POST", "/admin/players/"+middle+"/reports/reject", support, map[string]any{"reason": "name is fine"})
	expect(t, status, body, http.StatusOK, "")
	if rest := listedTargets(a, support, "?limit=2&after="+middle); strings.Join(rest, ",") != newest {
		t.Fatalf("after a decided target %v", rest)
	}
	if named := listedTargets(a, support, "?reason="+report.ReasonNickname); strings.Join(named, ",") != oldest+","+newest {
		t.Fatalf("named %v", named)
	}
	if found := listedTargets(a, support, "?account="+newest[:8]); strings.Join(found, ",") != newest {
		t.Fatalf("found %v", found)
	}
	status, body = a.call("GET", "/admin/reports?after=nope", support, nil)
	expect(t, status, body, http.StatusBadRequest, "invalid")
}

func TestSupportUpholdsOrRejectsTheOpenReportsOfAPlayer(t *testing.T) {
	a := newAPI(t)
	a.now = time.Now().UTC()
	a.create("orxan@example.com", rbac.Support)
	support, content, root := a.signIn("orxan"), a.signIn("ayla"), a.signIn("root")
	rude := a.player("g-1", "rude@mail.example", "Rude")
	calm := a.player("g-2", "calm@mail.example", "Calm")
	first := a.player("g-3", "first@mail.example", "First")
	second := a.player("g-4", "second@mail.example", "Second")
	a.fileReport(first, rude, report.ReasonNickname, a.now.Add(-2*time.Hour))
	a.fileReport(second, rude, report.ReasonImpersonation, a.now.Add(-time.Hour))
	a.fileReport(first, calm, report.ReasonOther, a.now.Add(-30*time.Minute))

	status, body := a.call("GET", "/admin/reports", content, nil)
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("GET", "/admin/reports", support, nil)
	targets := body["targets"].([]any)
	if status != http.StatusOK || len(targets) != 2 || targets[0].(map[string]any)["account_id"] != rude || len(targets[0].(map[string]any)["reports"].([]any)) != 2 {
		t.Fatalf("queue %d %v", status, body)
	}

	status, body = a.call("POST", "/admin/players/"+rude+"/reports/reject", support, map[string]any{})
	expect(t, status, body, http.StatusBadRequest, "reason_required")
	status, body = a.call("POST", "/admin/players/"+rude+"/reports/reject", support, map[string]any{"reason": "name is fine"})
	if status != http.StatusOK || body["closed"] != float64(2) || body["state"] != report.StateRejected {
		t.Fatalf("reject %d %v", status, body)
	}
	status, body = a.call("POST", "/admin/players/"+rude+"/reports/uphold", support, map[string]any{"reason": "again"})
	expect(t, status, body, http.StatusConflict, "no_open_reports")
	status, body = a.call("POST", "/admin/players/"+calm+"/reports/uphold", support, map[string]any{"reason": "confirmed"})
	if status != http.StatusOK || body["closed"] != float64(1) {
		t.Fatalf("uphold %d %v", status, body)
	}
	if _, body := a.call("GET", "/admin/reports", support, nil); len(body["targets"].([]any)) != 0 {
		t.Fatalf("queue after decisions %v", body)
	}
	rejected := a.entries(root, "object_type=players&action="+adminapi.RejectReportsAction)
	upheld := a.entries(root, "object_type=players&action="+adminapi.UpholdReportsAction)
	if len(rejected) != 1 || len(upheld) != 1 {
		t.Fatalf("entries %v %v", rejected, upheld)
	}
}

func TestTheReportQueueShowsTheEarlierDecisionsOnEachTarget(t *testing.T) {
	a := newAPI(t)
	a.now = time.Now().UTC()
	a.create("orxan@example.com", rbac.Support)
	support := a.signIn("orxan")
	rude := a.player("g-1", "rude@mail.example", "Rude")
	fresh := a.player("g-2", "fresh@mail.example", "Fresh")
	first := a.player("g-3", "first@mail.example", "First")
	second := a.player("g-4", "second@mail.example", "Second")
	a.fileReport(first, rude, report.ReasonNickname, a.now.Add(-3*time.Hour))
	a.fileReport(second, rude, report.ReasonImpersonation, a.now.Add(-3*time.Hour))
	status, body := a.call("POST", "/admin/players/"+rude+"/reports/reject", support, map[string]any{"reason": "name is fine"})
	expect(t, status, body, http.StatusOK, "")
	a.fileReport(first, rude, report.ReasonNickname, a.now.Add(-time.Hour))
	a.fileReport(first, fresh, report.ReasonOther, a.now.Add(-30*time.Minute))

	status, body = a.call("GET", "/admin/reports", support, nil)
	if status != http.StatusOK || len(body["targets"].([]any)) != 2 {
		t.Fatalf("queue %d %v", status, body)
	}
	earlier := body["targets"].([]any)[0].(map[string]any)
	decisions := earlier["decisions"].([]any)
	if earlier["account_id"] != rude || len(earlier["reports"].([]any)) != 1 || len(decisions) != 1 {
		t.Fatalf("earlier target %v", earlier)
	}
	decision := decisions[0].(map[string]any)
	if decision["queue"] != "reports" || decision["outcome"] != report.StateRejected || decision["reports"] != float64(2) || len(decision["reasons"].([]any)) != 2 || decision["decided_at"] == nil {
		t.Fatalf("decision %v", decision)
	}
	if _, leaked := decision["decided_by"]; leaked {
		t.Fatalf("the decider left the audit log %v", decision)
	}
	if none := body["targets"].([]any)[1].(map[string]any)["decisions"].([]any); len(none) != 0 {
		t.Fatalf("a first report has decisions %v", none)
	}
}
