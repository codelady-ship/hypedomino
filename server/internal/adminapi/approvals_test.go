package adminapi_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminaccount"
	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/adminsession"
	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func (a *api) proposeGrant(token, accountID string) string {
	a.t.Helper()
	status, body := a.call("PUT", "/admin/accounts/"+accountID+"/roles", token, map[string]any{"roles": []string{"content", "super_admin"}, "reason": "on-call cover"})
	if status != http.StatusAccepted || body["state"] != "pending" || body["kind"] != "super_admin_grant" || body["object_id"] != accountID {
		a.t.Fatalf("propose %d %v", status, body)
	}
	return strconv.FormatFloat(body["id"].(float64), 'f', 0, 64)
}

func (a *api) rolesOf(token, accountID string) []any {
	a.t.Helper()
	_, body := a.call("GET", "/admin/accounts", token, nil)
	for _, account := range body["accounts"].([]any) {
		if view := account.(map[string]any); view["id"] == accountID {
			return view["roles"].([]any)
		}
	}
	a.t.Fatalf("account %s not listed", accountID)
	return nil
}

func TestASuperAdminGrantTakesASecondSuperAdmin(t *testing.T) {
	a := newAPI(t)
	a.create("vusal@example.com", rbac.SuperAdmin)
	ayla, _ := a.accounts.ByEmail(context.Background(), "ayla@example.com")
	root, second, content := a.signIn("root"), a.signIn("vusal"), a.signIn("ayla")

	id := a.proposeGrant(root, ayla.ID)
	if roles := a.rolesOf(root, ayla.ID); !reflect.DeepEqual(roles, []any{"content"}) {
		t.Fatalf("roles changed before approval: %v", roles)
	}
	for token, want := range map[string]int{root: 1, second: 1, content: 0} {
		status, body := a.call("GET", "/admin/approvals", token, nil)
		if status != http.StatusOK || len(body["proposals"].([]any)) != want {
			t.Fatalf("pending list %d %v, want %d", status, body, want)
		}
	}
	status, body := a.call("POST", "/admin/approvals/"+id+"/approve", root, map[string]any{"reason": "checked"})
	expect(t, status, body, http.StatusForbidden, "self_decision")
	status, body = a.call("POST", "/admin/approvals/"+id+"/approve", content, map[string]any{"reason": "checked"})
	expect(t, status, body, http.StatusForbidden, "forbidden")

	a.now = start.Add(11 * time.Minute)
	refusals := []struct {
		decision string
		body     map[string]any
		want     int
		code     string
	}{
		{"approve", map[string]any{"ref": "OPS-3"}, http.StatusBadRequest, "reason_required"},
		{"reject", map[string]any{"reason": "  "}, http.StatusBadRequest, "reason_required"},
		{"approve", map[string]any{"reason": "checked"}, http.StatusForbidden, "step_up_required"},
		{"reject", map[string]any{"reason": "not needed"}, http.StatusForbidden, "step_up_required"},
	}
	for _, refusal := range refusals {
		status, body = a.call("POST", "/admin/approvals/"+id+"/"+refusal.decision, second, refusal.body)
		expect(t, status, body, refusal.want, refusal.code)
	}
	a.call("POST", "/admin/session/step-up", second, map[string]string{"id_token": "vusal", "nonce": "nonce"})
	status, body = a.call("POST", "/admin/approvals/"+id+"/approve", second, map[string]any{"reason": "checked", "ref": "OPS-3"})
	if status != http.StatusOK || body["state"] != "approved" || body["decider_id"] == "" {
		t.Fatalf("approve %d %v", status, body)
	}
	if roles := a.rolesOf(second, ayla.ID); !reflect.DeepEqual(roles, []any{"content", "super_admin"}) {
		t.Fatalf("roles after approval: %v", roles)
	}
	status, body = a.call("POST", "/admin/approvals/"+id+"/approve", second, map[string]any{"reason": "checked"})
	expect(t, status, body, http.StatusConflict, "decided")

	entries := a.entries(second, "object_type=admin_accounts&object_id="+ayla.ID)
	if len(entries) != 3 {
		t.Fatalf("entries %v", entries)
	}
	applied, decision, submitted := entries[0], entries[1], entries[2]
	if submitted["action"] != "submit" || submitted["reason"] != "on-call cover" || len(submitted["after"].(map[string]any)["roles"].([]any)) != 2 {
		t.Fatalf("submit entry %v", submitted)
	}
	if decision["action"] != "approve" || decision["ref"] != "OPS-3" || decision["actor_id"] == submitted["actor_id"] {
		t.Fatalf("approve entry %v", decision)
	}
	if applied["action"] != "edit" || applied["approval_id"] != decision["id"] || applied["reason"] != "on-call cover" ||
		len(applied["before"].(map[string]any)["roles"].([]any)) != 1 {
		t.Fatalf("edit entry %v", applied)
	}
}

func TestAGrantProposalCanBeRejectedOrExpire(t *testing.T) {
	a := newAPI(t)
	a.create("vusal@example.com", rbac.SuperAdmin)
	ayla, _ := a.accounts.ByEmail(context.Background(), "ayla@example.com")
	root, second := a.signIn("root"), a.signIn("vusal")

	rejected := a.proposeGrant(root, ayla.ID)
	status, body := a.call("POST", "/admin/approvals/"+rejected+"/reject", second, map[string]any{"reason": "not needed"})
	if status != http.StatusOK || body["state"] != "rejected" {
		t.Fatalf("reject %d %v", status, body)
	}
	expired := a.proposeGrant(root, ayla.ID)
	a.now = start.Add(25 * time.Hour)
	late := a.signIn("vusal")
	status, body = a.call("POST", "/admin/approvals/"+expired+"/approve", late, map[string]any{"reason": "checked"})
	expect(t, status, body, http.StatusConflict, "expired")
	status, body = a.call("GET", "/admin/approvals", late, nil)
	if status != http.StatusOK || len(body["proposals"].([]any)) != 0 {
		t.Fatalf("pending after expiry %d %v", status, body)
	}
	for _, id := range []string{"999", "nope"} {
		status, body = a.call("POST", "/admin/approvals/"+id+"/approve", late, map[string]any{"reason": "checked"})
		expect(t, status, body, http.StatusNotFound, "not_found")
	}
	if roles := a.rolesOf(late, ayla.ID); !reflect.DeepEqual(roles, []any{"content"}) {
		t.Fatalf("roles changed: %v", roles)
	}
	rejections := a.entries(late, "object_type=admin_accounts&action=reject")
	if len(rejections) != 1 || rejections[0]["reason"] != "not needed" {
		t.Fatalf("reject entries %v", rejections)
	}
}

func TestAGrantProposalNeedsAFreshSignIn(t *testing.T) {
	a := newAPI(t)
	ayla, _ := a.accounts.ByEmail(context.Background(), "ayla@example.com")
	root := a.signIn("root")
	a.now = start.Add(11 * time.Minute)
	status, body := a.call("PUT", "/admin/accounts/"+ayla.ID+"/roles", root, map[string]any{"roles": []string{"super_admin"}, "reason": "cover"})
	expect(t, status, body, http.StatusForbidden, "step_up_required")
	if entries := a.entries(root, "object_type=admin_accounts"); len(entries) != 0 {
		t.Fatalf("entries %v", entries)
	}
}

func TestBootstrapCreatesTheFirstSuperAdminOnlyOnce(t *testing.T) {
	_, pool := dbtest.Fresh(t)
	ctx := context.Background()
	config := adminapi.Config{Environment: "production", GoogleDomain: "example.com", Session: adminsession.DefaultConfig()}
	if _, err := adminapi.Bootstrap(ctx, pool, config, "first@gmail.com", "First", start); err == nil {
		t.Fatal("a personal email was bootstrapped")
	}
	created, err := adminapi.Bootstrap(ctx, pool, config, " First@Example.com ", "First", start)
	if err != nil || created.Email != "first@example.com" || !reflect.DeepEqual(created.Roles, []rbac.Role{rbac.SuperAdmin}) {
		t.Fatalf("bootstrap %+v %v", created, err)
	}
	if _, err := adminapi.Bootstrap(ctx, pool, config, "second@example.com", "Second", start); !errors.Is(err, adminapi.ErrBootstrapped) {
		t.Fatalf("second bootstrap: %v", err)
	}
	accounts, err := adminaccount.NewStore(pool).List(ctx)
	if err != nil || len(accounts) != 1 {
		t.Fatalf("accounts %+v %v", accounts, err)
	}
	entries, err := audit.NewStore(pool).Search(ctx, []rbac.Role{rbac.SuperAdmin}, audit.Filter{Object: audit.Object{Type: rbac.AdminAccounts}})
	if err != nil || len(entries) != 1 || entries[0].Actor.ID != adminapi.BootstrapActor || entries[0].Action != "create" || entries[0].Environment != "production" {
		t.Fatalf("entries %+v %v", entries, err)
	}
}
