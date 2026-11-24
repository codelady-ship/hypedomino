package adminapi_test

import (
	"net/http"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func TestModerationListsArePublishedBySupportAndValidated(t *testing.T) {
	a := newAPI(t)
	a.create("orxan@example.com", rbac.Support)
	content, support := a.signIn("ayla"), a.signIn("orxan")
	path := "/admin/config/moderation"
	lists := map[string]any{"words": map[string]any{"en": []any{map[string]any{"entry": "badword", "match": "substring"}}}, "reserved": []string{"admin"}}

	status, body := a.call("GET", path, content, nil)
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("PUT", path, content, map[string]any{"catalog": lists, "reason": "launch"})
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("PUT", path, support, map[string]any{"catalog": map[string]any{"words": map[string]any{"de": []any{}}}, "reason": "launch"})
	expect(t, status, body, http.StatusBadRequest, "invalid")
	status, body = a.call("PUT", path, support, map[string]any{"catalog": map[string]any{"words": map[string]any{"en": []any{map[string]any{"entry": "x", "match": "prefix"}}}}, "reason": "launch"})
	expect(t, status, body, http.StatusBadRequest, "invalid")

	status, body = a.call("PUT", path, support, map[string]any{"catalog": lists, "reason": "launch"})
	if status != http.StatusOK || body["config_version"] != float64(1) {
		t.Fatalf("publish %d %v", status, body)
	}
	status, body = a.call("GET", path, support, nil)
	if reserved := body["catalog"].(map[string]any)["reserved"].([]any); status != http.StatusOK || len(reserved) != 1 || reserved[0] != "admin" {
		t.Fatalf("read %d %v", status, body)
	}
	status, body = a.call("GET", path+"/history", support, nil)
	if versions := body["versions"].([]any); status != http.StatusOK || len(versions) != 1 {
		t.Fatalf("history %d %v", status, body)
	}
}
