package adminapi_test

import (
	"net/http"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func calendarOf(coins ...int) map[string]any {
	days := make([]any, 0, len(coins))
	for _, amount := range coins {
		days = append(days, map[string]any{"coins": amount})
	}
	return map[string]any{"schema": 1, "days": days}
}

func TestTheDailyLoginCalendarIsPublishedCheckedAndRolledBack(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	liveOps, content := a.signIn("leyla"), a.signIn("ayla")
	week := calendarOf(30, 30, 50, 50, 70, 70, 150)

	status, body := a.call("GET", "/admin/config/daily-login", content, nil)
	expect(t, status, body, http.StatusOK, "")
	if body["catalog"] != nil || len(body["rules"].(map[string]any)["default"].(map[string]any)["days"].([]any)) != 7 {
		t.Fatalf("empty %v", body)
	}

	status, body = a.call("PUT", "/admin/config/daily-login", content, map[string]any{"catalog": week, "reason": "launch"})
	expect(t, status, body, http.StatusForbidden, "forbidden")
	for _, refused := range []map[string]any{calendarOf(30, 30, 50), calendarOf(30, 30, 50, 50, 70, 70, 0)} {
		status, body = a.call("PUT", "/admin/config/daily-login", liveOps, map[string]any{"catalog": refused, "reason": "bad"})
		expect(t, status, body, http.StatusBadRequest, "invalid")
	}
	medal := calendarOf(30, 30, 50, 50, 70, 70, 150)
	medal["days"].([]any)[0].(map[string]any)["medals"] = []any{"medal.first"}
	status, body = a.call("PUT", "/admin/config/daily-login", liveOps, map[string]any{"catalog": medal, "reason": "bad"})
	expect(t, status, body, http.StatusBadRequest, "invalid")

	status, body = a.call("PUT", "/admin/config/daily-login", liveOps, map[string]any{"catalog": week, "reason": "launch"})
	expect(t, status, body, http.StatusOK, "")
	first := body["config_version"]
	status, body = a.call("PUT", "/admin/config/daily-login", liveOps, map[string]any{"catalog": calendarOf(40, 30, 50, 50, 70, 70, 150), "reason": "retune"})
	expect(t, status, body, http.StatusOK, "")

	day := bundleRows(t, a, liveOps, "bundle.daily_login_day_1")
	if len(day) != 1 || day[0]["coins"].(float64) != 40 {
		t.Fatalf("the published day is the one listed: %v", day)
	}
	status, body = a.call("GET", "/admin/config/daily-login/history", content, nil)
	expect(t, status, body, http.StatusOK, "")
	if len(body["versions"].([]any)) != 2 {
		t.Fatalf("history %v", body)
	}

	status, body = a.call("POST", "/admin/config/daily-login/rollback", liveOps, map[string]any{"version": first, "reason": "back"})
	expect(t, status, body, http.StatusOK, "")
	day = bundleRows(t, a, liveOps, "bundle.daily_login_day_1")
	if len(day) != 1 || day[0]["coins"].(float64) != 30 {
		t.Fatalf("rolled back %v", day)
	}
}
