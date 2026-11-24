package adminapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func TestSupportMutesForTheConfiguredHoursAndUnmutes(t *testing.T) {
	a := newAPI(t)
	a.now = time.Now().UTC()
	a.create("orxan@example.com", rbac.Support)
	support, content, root := a.signIn("orxan"), a.signIn("ayla"), a.signIn("root")
	id := a.player("g-1", "kamal@mail.example", "Kamal")
	path := "/admin/players/" + id + "/mute"

	status, body := a.call("DELETE", path, support, map[string]any{"reason": "appeal"})
	expect(t, status, body, http.StatusConflict, "not_muted")
	status, body = a.call("POST", path, support, map[string]any{})
	expect(t, status, body, http.StatusBadRequest, "reason_required")
	status, body = a.call("POST", path, content, map[string]any{"reason": "emoji spam"})
	expect(t, status, body, http.StatusForbidden, "forbidden")

	status, body = a.call("POST", path, support, map[string]any{"reason": "emoji spam", "ref": "REP-3"})
	until, _ := time.Parse(time.RFC3339Nano, body["muted_until"].(string))
	if status != http.StatusOK || !until.Equal(a.now.Add(adminapi.DefaultMuteHours*time.Hour).Truncate(time.Microsecond)) {
		t.Fatalf("mute %d %v", status, body)
	}
	if _, body := a.call("GET", "/admin/players/"+id, support, nil); body["account"].(map[string]any)["muted_until"] == nil {
		t.Fatalf("record %v", body)
	}
	status, body = a.call("DELETE", path, support, map[string]any{"reason": "appeal upheld"})
	if status != http.StatusOK || body["muted_until"] != nil {
		t.Fatalf("unmute %d %v", status, body)
	}
	muted := a.entries(root, "object_type=players&action="+adminapi.MuteAction)
	unmuted := a.entries(root, "object_type=players&action="+adminapi.UnmuteAction)
	if len(muted) != 1 || muted[0]["ref"] != "REP-3" || len(unmuted) != 1 {
		t.Fatalf("entries %v %v", muted, unmuted)
	}
}
