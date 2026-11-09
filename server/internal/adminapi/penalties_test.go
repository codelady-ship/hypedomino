package adminapi_test

import (
	"net/http"
	"testing"
)

func TestThePenaltiesConfigIsCheckedBeforeItIsPublished(t *testing.T) {
	a := newAPI(t)
	root := a.signIn("root")
	path := "/admin/config/penalties"

	status, body := a.call("PUT", path, root, map[string]any{"catalog": map[string]any{"banLevel": 1}, "reason": "tune"})
	expect(t, status, body, http.StatusBadRequest, "invalid")
	status, body = a.call("PUT", path, root, map[string]any{"catalog": map[string]any{"banHours": 48}, "reason": "tune"})
	if status != http.StatusOK || body["config_version"] != float64(1) {
		t.Fatalf("publish %d %v", status, body)
	}
	status, body = a.call("GET", path, root, nil)
	if catalog := body["catalog"].(map[string]any); status != http.StatusOK || catalog["banHours"] != float64(48) {
		t.Fatalf("read %d %v", status, body)
	}
}
