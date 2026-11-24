package adminapi_test

import (
	"net/http"
	"testing"
)

func TestTheMatchmakingConfigIsCheckedBeforeItIsPublished(t *testing.T) {
	a := newAPI(t)
	root := a.signIn("root")
	path := "/admin/config/matchmaking"

	status, body := a.call("PUT", path, root, map[string]any{"catalog": map[string]any{"casualBotLevel": "brutal"}, "reason": "tune"})
	expect(t, status, body, http.StatusBadRequest, "invalid")
	status, body = a.call("PUT", path, root, map[string]any{"catalog": map[string]any{"casualBotLevel": "easy", "casualBotFillSeconds": 20}, "reason": "tune"})
	if status != http.StatusOK || body["config_version"] != float64(1) {
		t.Fatalf("publish %d %v", status, body)
	}
	status, body = a.call("GET", path, root, nil)
	if catalog := body["catalog"].(map[string]any); status != http.StatusOK || catalog["casualBotLevel"] != "easy" || catalog["casualBotFillSeconds"] != float64(20) {
		t.Fatalf("read %d %v", status, body)
	}
}
