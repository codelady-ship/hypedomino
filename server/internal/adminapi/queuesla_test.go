package adminapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/report"
	"github.com/Alievs-corp/hypedomino/server/internal/risk"
)

type listedDeadline struct {
	due     time.Time
	overdue bool
}

func listedDeadlines(a *api, token, path, list string) map[string]listedDeadline {
	a.t.Helper()
	status, body := a.call("GET", path, token, nil)
	if status != http.StatusOK {
		a.t.Fatalf("%s %d %v", path, status, body)
	}
	deadlines := map[string]listedDeadline{}
	for _, entry := range body[list].([]any) {
		fields := entry.(map[string]any)
		due, err := time.Parse(time.RFC3339Nano, fields["due_at"].(string))
		if err != nil {
			a.t.Fatal(err)
		}
		deadlines[fields["account_id"].(string)] = listedDeadline{due: due, overdue: fields["overdue"].(bool)}
	}
	return deadlines
}

func publishLiveops(a *api, token string, values map[string]int) {
	a.t.Helper()
	status, body := a.call("PUT", "/admin/config/liveops", token, map[string]any{"values": values, "reason": "queue sla"})
	if status != http.StatusOK {
		a.t.Fatalf("publish %d %v", status, body)
	}
}

func TestEachReviewIsDueTheReviewSlaAfterItOpened(t *testing.T) {
	a := newAPI(t)
	a.now = time.Now().UTC().Truncate(time.Second)
	a.create("orxan@example.com", rbac.Support)
	a.create("leyla@example.com", rbac.LiveOps)
	support, liveOps := a.signIn("orxan"), a.signIn("leyla")
	late := a.player("g-1", "late@mail.example", "Late")
	exact := a.player("g-2", "exact@mail.example", "Exact")
	fresh := a.player("g-3", "fresh@mail.example", "Fresh")
	a.openReview(late, risk.CauseScore, a.now.Add(-25*time.Hour))
	a.openReview(exact, risk.CauseScore, a.now.Add(-24*time.Hour))
	a.openReview(fresh, risk.CauseHardSignal, a.now.Add(-2*time.Hour))

	deadlines := listedDeadlines(a, support, "/admin/reviews", "reviews")
	want := map[string]listedDeadline{
		late:  {due: a.now.Add(-time.Hour), overdue: true},
		exact: {due: a.now, overdue: false},
		fresh: {due: a.now.Add(22 * time.Hour), overdue: false},
	}
	for account, deadline := range want {
		if got := deadlines[account]; !got.due.Equal(deadline.due) || got.overdue != deadline.overdue {
			t.Errorf("default sla %s: %+v, want %+v", account, got, deadline)
		}
	}

	publishLiveops(a, liveOps, map[string]int{"ReviewSlaHours": 1})
	deadlines = listedDeadlines(a, support, "/admin/reviews", "reviews")
	if got := deadlines[fresh]; !got.due.Equal(a.now.Add(-time.Hour)) || !got.overdue {
		t.Errorf("published sla: %+v", got)
	}
}

func TestEachReportedPlayerIsDueTheReportSlaAfterTheOldestOpenReport(t *testing.T) {
	a := newAPI(t)
	a.now = time.Now().UTC().Truncate(time.Second)
	a.create("orxan@example.com", rbac.Support)
	a.create("leyla@example.com", rbac.LiveOps)
	support, liveOps := a.signIn("orxan"), a.signIn("leyla")
	reporter := a.player("g-0", "reporter@mail.example", "Reporter")
	second := a.player("g-4", "second@mail.example", "Second")
	late := a.player("g-1", "late@mail.example", "Late")
	exact := a.player("g-2", "exact@mail.example", "Exact")
	fresh := a.player("g-3", "fresh@mail.example", "Fresh")
	a.fileReport(second, late, report.ReasonOther, a.now.Add(-time.Hour))
	a.fileReport(reporter, late, report.ReasonNickname, a.now.Add(-50*time.Hour))
	a.fileReport(reporter, exact, report.ReasonNickname, a.now.Add(-48*time.Hour))
	a.fileReport(reporter, fresh, report.ReasonImpersonation, a.now.Add(-47*time.Hour))

	deadlines := listedDeadlines(a, support, "/admin/reports", "targets")
	want := map[string]listedDeadline{
		late:  {due: a.now.Add(-2 * time.Hour), overdue: true},
		exact: {due: a.now, overdue: false},
		fresh: {due: a.now.Add(time.Hour), overdue: false},
	}
	for account, deadline := range want {
		if got := deadlines[account]; !got.due.Equal(deadline.due) || got.overdue != deadline.overdue {
			t.Errorf("default sla %s: %+v, want %+v", account, got, deadline)
		}
	}

	publishLiveops(a, liveOps, map[string]int{"ReportSlaHours": 24, "ReviewSlaHours": 720, "AppealSlaHours": 1})
	deadlines = listedDeadlines(a, support, "/admin/reports", "targets")
	if got := deadlines[fresh]; !got.due.Equal(a.now.Add(-23*time.Hour)) || !got.overdue {
		t.Errorf("published sla: %+v", got)
	}
}
