package adminapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func TestSupportShadowbansAndLiftsWithoutNotifyingThePlayer(t *testing.T) {
	a := newAPI(t)
	a.now = time.Now().UTC()
	a.create("orxan@example.com", rbac.Support)
	support, content, root := a.signIn("orxan"), a.signIn("ayla"), a.signIn("root")
	id := a.player("g-1", "kamal@mail.example", "Kamal")
	var hidden []string
	a.handler.(*adminapi.Handler).OnShadowban(func(_ context.Context, account string) error {
		hidden = append(hidden, account)
		return nil
	})
	path := "/admin/players/" + id + "/shadowban"

	status, body := a.call("DELETE", path, support, map[string]any{"reason": "appeal"})
	expect(t, status, body, http.StatusConflict, "not_shadowbanned")
	status, body = a.call("POST", path, support, map[string]any{})
	expect(t, status, body, http.StatusBadRequest, "reason_required")
	status, body = a.call("POST", path, content, map[string]any{"reason": "collusion"})
	expect(t, status, body, http.StatusForbidden, "forbidden")

	status, body = a.call("POST", path, support, map[string]any{"reason": "collusion", "ref": "REV-9"})
	if status != http.StatusOK || body["shadowbanned_at"] == nil {
		t.Fatalf("shadowban %d %v", status, body)
	}
	if !a.shadowbanned(id) || len(hidden) != 1 || hidden[0] != id {
		t.Fatalf("shadowbanned %v hidden %v", a.shadowbanned(id), hidden)
	}
	if err := a.playerSignIn("g-1"); err != nil {
		t.Fatalf("a shadowban blocked sign-in: %v", err)
	}
	status, body = a.call("POST", path, support, map[string]any{"reason": "again"})
	expect(t, status, body, http.StatusConflict, "shadowbanned")
	if _, body := a.call("GET", "/admin/players/"+id, support, nil); body["account"].(map[string]any)["shadowbanned_at"] == nil {
		t.Fatalf("record %v", body)
	}

	status, body = a.call("DELETE", path, support, map[string]any{"reason": "appeal upheld"})
	if status != http.StatusOK || body["shadowbanned_at"] != nil || a.shadowbanned(id) {
		t.Fatalf("lift %d %v", status, body)
	}
	banned := a.entries(root, "object_type=players&action="+adminapi.ShadowbanAction)
	lifted := a.entries(root, "object_type=players&action="+adminapi.LiftShadowbanAction)
	if len(banned) != 1 || banned[0]["ref"] != "REV-9" || len(lifted) != 1 || lifted[0]["reason"] != "appeal upheld" {
		t.Fatalf("entries %v %v", banned, lifted)
	}
}
