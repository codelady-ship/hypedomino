package adminapi_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

func TestSupportForcesARenameThatThePlayerMustComplete(t *testing.T) {
	a := newAPI(t)
	a.create("orxan@example.com", rbac.Support)
	support, content, root := a.signIn("orxan"), a.signIn("ayla"), a.signIn("root")
	id := a.player("g-1", "kamal@mail.example", "Rude_Name")
	path := "/admin/players/" + id + "/force-rename"

	status, body := a.call("POST", path, support, map[string]any{})
	expect(t, status, body, http.StatusBadRequest, "reason_required")
	status, body = a.call("POST", path, content, map[string]any{"reason": "report"})
	expect(t, status, body, http.StatusForbidden, "forbidden")
	for range 2 {
		status, body = a.call("POST", path, support, map[string]any{"reason": "confirmed report", "ref": "REP-3"})
		if status != http.StatusOK || body["nickname"] != "Rude_Name" || body["rename_required"] != true {
			t.Fatalf("force %d %v", status, body)
		}
	}
	status, body = a.call("GET", "/admin/players/"+id, support, nil)
	if status != http.StatusOK || body["profile"].(map[string]any)["rename_required"] != true {
		t.Fatalf("record %d %v", status, body)
	}
	forced, running, err := sanction.Current(context.Background(), a.pool, id, a.now, sanction.ForcedRename)
	if err != nil || !running || forced.Category != sanction.OffensiveName || forced.EndsAt != nil {
		t.Fatalf("forced rename sanction %+v %v %v", forced, running, err)
	}
	renamed, err := profile.NewStore(a.pool, profile.DefaultConfig()).SetNickname(context.Background(), id, "Kind_Name", start)
	if err != nil || renamed.RenameRequired {
		t.Fatalf("player rename %+v %v", renamed, err)
	}
	if _, running, err := sanction.Current(context.Background(), a.pool, id, a.now.Add(time.Second), sanction.ForcedRename); err != nil || running {
		t.Fatalf("the forced rename still runs after the rename %v %v", running, err)
	}
	if history, err := sanction.List(context.Background(), a.pool, id, 50); err != nil || len(history) != 1 {
		t.Fatalf("one sanction for two force calls %+v %v", history, err)
	}

	created, _, err := account.New(a.pool).SignIn(context.Background(), idtoken.Identity{Provider: idtoken.ProviderGoogle, Subject: "g-2"})
	if err != nil {
		t.Fatal(err)
	}
	status, body = a.call("POST", "/admin/players/"+created.ID+"/force-rename", support, map[string]any{"reason": "report"})
	expect(t, status, body, http.StatusConflict, "no_profile")

	entries := a.entries(root, "object_type=players&action=force_rename")
	if len(entries) != 1 || entries[0]["ref"] != "REP-3" || entries[0]["before"].(map[string]any)["rename_required"] != false ||
		entries[0]["after"].(map[string]any)["nickname"] != "Rude_Name" {
		t.Fatalf("entries %v", entries)
	}
}
