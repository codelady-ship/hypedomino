package adminapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/adminaccount"
	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/adminsession"
	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
	"github.com/Alievs-corp/hypedomino/server/internal/ratelimit"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
)

var (
	start       = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	unthrottled = adminapi.Limits{Account: ratelimit.Rule{PerMinute: 60000, Burst: 10000}, SignIn: ratelimit.Rule{PerMinute: 60000, Burst: 10000}}
)

func TestMain(m *testing.M) {
	os.Exit(dbtest.Main(m))
}

type identities map[string]idtoken.Identity

func (identities) IssueNonce(context.Context) (string, error) { return "nonce", nil }

func (i identities) Verify(_ context.Context, provider idtoken.Provider, token, nonce string) (idtoken.Identity, error) {
	identity, ok := i[token]
	if !ok || provider != idtoken.ProviderGoogle || nonce != "nonce" {
		return idtoken.Identity{}, idtoken.ErrSignature
	}
	return identity, nil
}

func company(email string) idtoken.Identity {
	return idtoken.Identity{Provider: idtoken.ProviderGoogle, Subject: email, Email: email, EmailVerified: true, HostedDomain: "example.com"}
}

type api struct {
	t        *testing.T
	pool     *pgxpool.Pool
	handler  http.Handler
	accounts *adminaccount.Store
	content  *contentStore
	now      time.Time
}

func newAPI(t *testing.T) *api {
	t.Helper()
	return newAPIWith(t, func(*adminapi.Config) {})
}

func newAPIWith(t *testing.T, adjust func(*adminapi.Config)) *api {
	t.Helper()
	_, pool := dbtest.Fresh(t)
	verifier := identities{
		"root":       company("root@example.com"),
		"ayla":       company("ayla@example.com"),
		"vusal":      company("vusal@example.com"),
		"nobody":     company("nobody@example.com"),
		"leyla":      company("leyla@example.com"),
		"orxan":      company("orxan@example.com"),
		"unverified": {Email: "ayla@example.com", HostedDomain: "example.com"},
		"personal":   {Email: "ayla@example.com", EmailVerified: true},
		"elsewhere":  {Email: "ayla@other.com", EmailVerified: true, HostedDomain: "other.com"},
		"mixed":      {Email: "ayla@other.com", EmailVerified: true, HostedDomain: "example.com"},
	}
	a := &api{t: t, pool: pool, accounts: adminaccount.NewStore(pool), content: &contentStore{files: map[string]objectstore.Object{}}, now: start}
	config := adminapi.Config{Environment: "staging", GoogleDomain: "Example.com", Session: adminsession.DefaultConfig(), ApprovalTTL: 24 * time.Hour, Shop: shop.DefaultConfig(), GrantApprovalCoins: adminapi.DefaultGrantApprovalCoins, TempBan: 72 * time.Hour, RankedBan: adminapi.DefaultRankedBanHours * time.Hour, Mute: adminapi.DefaultMuteHours * time.Hour, Limits: unthrottled, Content: a.content}
	adjust(&config)
	a.handler = adminapi.New(config, verifier, ratelimit.NewMemory(), pool, func() time.Time { return a.now }, slog.New(slog.NewTextHandler(io.Discard, nil)))
	a.create("root@example.com", rbac.SuperAdmin)
	a.create("ayla@example.com", rbac.Content)
	return a
}

func (a *api) create(email string, roles ...rbac.Role) adminaccount.Account {
	a.t.Helper()
	account, err := a.accounts.Create(context.Background(), email, email, roles, start.Add(-time.Hour))
	if err != nil {
		a.t.Fatal(err)
	}
	return account
}

func (a *api) call(method, path, token string, body any) (int, map[string]any) {
	a.t.Helper()
	var payload io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		payload = bytes.NewReader(raw)
	}
	request := httptest.NewRequest(method, path, payload)
	request.RemoteAddr = "10.0.0.5:4431"
	request.Header.Set("X-Forwarded-For", "198.51.100.7")
	request.Header.Set("User-Agent", "admin-web")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	a.handler.ServeHTTP(recorder, request)
	var decoded map[string]any
	json.Unmarshal(recorder.Body.Bytes(), &decoded)
	return recorder.Code, decoded
}

func (a *api) signIn(idToken string) string {
	a.t.Helper()
	status, body := a.call("POST", "/admin/session", "", map[string]string{"id_token": idToken, "nonce": "nonce"})
	if status != http.StatusOK {
		a.t.Fatalf("sign in %s: %d %v", idToken, status, body)
	}
	return body["token"].(string)
}

func expect(t *testing.T, status int, body map[string]any, wantStatus int, wantError string) {
	t.Helper()
	if status != wantStatus || wantError != "" && body["error"] != wantError {
		t.Fatalf("got %d %v, want %d %q", status, body, wantStatus, wantError)
	}
}

func refusedKeys(body map[string]any, want ...string) bool {
	keys, _ := body["keys"].([]any)
	if len(keys) != len(want) {
		return false
	}
	for i, key := range keys {
		if key != want[i] {
			return false
		}
	}
	return true
}

func TestOnlyActiveCompanyAccountsSignIn(t *testing.T) {
	a := newAPI(t)
	disabled := a.create("vusal@example.com", rbac.Support)
	if _, err := a.accounts.Disable(context.Background(), disabled.ID, start); err != nil {
		t.Fatal(err)
	}
	tests := map[string]struct {
		token  string
		status int
		error  string
	}{
		"bad token":        {"forged", http.StatusUnauthorized, "token_rejected"},
		"unverified email": {"unverified", http.StatusForbidden, "not_company_account"},
		"personal account": {"personal", http.StatusForbidden, "not_company_account"},
		"other workspace":  {"elsewhere", http.StatusForbidden, "not_company_account"},
		"email outside it": {"mixed", http.StatusForbidden, "not_company_account"},
		"disabled account": {"vusal", http.StatusForbidden, "account_refused"},
		"no admin account": {"nobody", http.StatusForbidden, "account_refused"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			status, body := a.call("POST", "/admin/session", "", map[string]string{"id_token": test.token, "nonce": "nonce"})
			expect(t, status, body, test.status, test.error)
		})
	}
	status, body := a.call("POST", "/admin/session", "", map[string]string{"id_token": "ayla", "nonce": "nonce"})
	expect(t, status, body, http.StatusOK, "")
	session := body["session"].(map[string]any)
	if body["token"] == "" || session["step_up_until"] != "2026-11-01T09:10:00Z" || session["idle_until"] != "2026-11-01T09:30:00Z" || session["expires_at"] != "2026-11-01T21:00:00Z" {
		t.Fatalf("sign in %v", body)
	}
	account := body["account"].(map[string]any)
	if account["email"] != "ayla@example.com" || account["last_login_at"] != "2026-11-01T09:00:00Z" {
		t.Fatalf("account %v", account)
	}
}

func TestMeListsThePermissionsOfTheRoles(t *testing.T) {
	a := newAPI(t)
	status, body := a.call("GET", "/admin/me", a.signIn("ayla"), nil)
	expect(t, status, body, http.StatusOK, "")
	permissions := body["permissions"].(map[string]any)
	if _, ok := permissions[string(rbac.AdminAccounts)]; ok {
		t.Fatalf("content sees admin accounts: %v", permissions)
	}
	strings := permissions[string(rbac.GameStrings)].([]any)
	if len(strings) != 6 || !slices.Contains(strings, any("publish")) {
		t.Fatalf("content on game strings: %v", strings)
	}
}

func TestEveryRouteRefusesAMissingSessionAndARoleWithoutTheAction(t *testing.T) {
	a := newAPI(t)
	content := a.signIn("ayla")
	root := a.signIn("root")
	target := a.create("vusal@example.com", rbac.Support)
	routes := []struct {
		method, path string
		body         any
		content      int
	}{
		{"GET", "/admin/me", nil, http.StatusOK},
		{"PUT", "/admin/me/locale", map[string]any{"locale": "az"}, http.StatusOK},
		{"GET", "/admin/audit", nil, http.StatusOK},
		{"POST", "/admin/audit/export", map[string]any{"reason": "regulator request"}, http.StatusForbidden},
		{"GET", "/admin/accounts", nil, http.StatusForbidden},
		{"POST", "/admin/accounts", map[string]any{"email": "new@example.com", "display_name": "New", "roles": []string{"analyst"}}, http.StatusForbidden},
		{"PUT", "/admin/accounts/" + target.ID + "/roles", map[string]any{"roles": []string{"analyst"}}, http.StatusForbidden},
		{"POST", "/admin/accounts/" + target.ID + "/disable", map[string]any{}, http.StatusForbidden},
		{"GET", "/admin/live-controls/break-glass", nil, http.StatusForbidden},
		{"GET", "/admin/config/version", nil, http.StatusOK},
		{"POST", "/admin/session/step-up", map[string]any{"id_token": "ayla", "nonce": "nonce"}, http.StatusOK},
		{"DELETE", "/admin/session", nil, http.StatusNoContent},
	}
	for _, route := range routes {
		for _, token := range []string{"", "made-up", root + "x"} {
			status, body := a.call(route.method, route.path, token, route.body)
			expect(t, status, body, http.StatusUnauthorized, "session_rejected")
		}
		status, body := a.call(route.method, route.path, content, route.body)
		if status != route.content {
			t.Fatalf("content %s %s: %d %v", route.method, route.path, status, body)
		}
	}
	status, body := a.call("GET", "/admin/me", content, nil)
	expect(t, status, body, http.StatusUnauthorized, "session_rejected")
	status, body = a.call("GET", "/admin/accounts", root, nil)
	if status != http.StatusOK || len(body["accounts"].([]any)) != 3 {
		t.Fatalf("root lists accounts: %d %v", status, body)
	}
}

func TestAccountChangesAreAuditedWithTheirSource(t *testing.T) {
	a := newAPI(t)
	root := a.signIn("root")
	status, created := a.call("POST", "/admin/accounts", root, map[string]any{
		"email": " Vusal@Example.com ", "display_name": "Vusal", "roles": []string{"support"}, "ref": "#134",
	})
	expect(t, status, created, http.StatusCreated, "")
	id := created["id"].(string)
	a.now = start.Add(time.Minute)
	status, body := a.call("PUT", "/admin/accounts/"+id+"/roles", root, map[string]any{"roles": []string{"support", "analyst"}, "reason": "covers reports"})
	if status != http.StatusOK || len(body["roles"].([]any)) != 2 {
		t.Fatalf("set roles %d %v", status, body)
	}
	vusal := a.signIn("vusal")
	a.now = start.Add(2 * time.Minute)
	status, body = a.call("POST", "/admin/accounts/"+id+"/disable", root, map[string]any{"reason": "left the team"})
	if status != http.StatusOK || body["state"] != "disabled" {
		t.Fatalf("disable %d %v", status, body)
	}
	status, body = a.call("GET", "/admin/me", vusal, nil)
	expect(t, status, body, http.StatusUnauthorized, "session_rejected")

	status, body = a.call("GET", "/admin/audit?object_type=admin_accounts&object_id="+id, root, nil)
	expect(t, status, body, http.StatusOK, "")
	entries := body["entries"].([]any)
	if len(entries) != 3 {
		t.Fatalf("entries %v", entries)
	}
	retire, edit, create := entries[0].(map[string]any), entries[1].(map[string]any), entries[2].(map[string]any)
	if create["action"] != "create" || create["before"] != nil || create["ref"] != "#134" ||
		create["after"].(map[string]any)["email"] != "vusal@example.com" {
		t.Fatalf("create entry %v", create)
	}
	if edit["action"] != "edit" || edit["reason"] != "covers reports" || len(edit["before"].(map[string]any)["roles"].([]any)) != 1 ||
		len(edit["after"].(map[string]any)["roles"].([]any)) != 2 {
		t.Fatalf("edit entry %v", edit)
	}
	if retire["action"] != "retire" || retire["before"].(map[string]any)["state"] != "active" || retire["after"].(map[string]any)["state"] != "disabled" {
		t.Fatalf("retire entry %v", retire)
	}
	for _, entry := range entries {
		e := entry.(map[string]any)
		if e["environment"] != "staging" || e["source_ip"] != "198.51.100.7" || e["user_agent"] != "admin-web" || e["session_id"] == "" ||
			e["actor_roles"].([]any)[0] != "super_admin" {
			t.Fatalf("entry source %v", e)
		}
	}
	status, body = a.call("GET", "/admin/audit", a.signIn("ayla"), nil)
	if status != http.StatusOK || len(body["entries"].([]any)) != 0 {
		t.Fatalf("content sees account entries: %d %v", status, body)
	}
}

func TestGuardedAccountChangesAreRefusedWithoutAnEntry(t *testing.T) {
	a := newAPI(t)
	root := a.signIn("root")
	self, err := a.accounts.ByEmail(context.Background(), "root@example.com")
	if err != nil {
		t.Fatal(err)
	}
	ayla, err := a.accounts.ByEmail(context.Background(), "ayla@example.com")
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]struct {
		method, path string
		body         any
		status       int
		error        string
	}{
		"create a super admin":   {"POST", "/admin/accounts", map[string]any{"email": "x@example.com", "display_name": "X", "roles": []string{"super_admin"}}, http.StatusConflict, "grant_by_approval"},
		"grant without a reason": {"PUT", "/admin/accounts/" + ayla.ID + "/roles", map[string]any{"roles": []string{"content", "super_admin"}}, http.StatusBadRequest, "reason_required"},
		"grant an unknown role":  {"PUT", "/admin/accounts/" + ayla.ID + "/roles", map[string]any{"roles": []string{"super_admin", "owner"}, "reason": "x"}, http.StatusBadRequest, "invalid"},
		"change own roles":       {"PUT", "/admin/accounts/" + self.ID + "/roles", map[string]any{"roles": []string{"analyst"}}, http.StatusConflict, "own_account"},
		"disable own account":    {"POST", "/admin/accounts/" + self.ID + "/disable", map[string]any{}, http.StatusConflict, "own_account"},
		"unknown role":           {"PUT", "/admin/accounts/" + ayla.ID + "/roles", map[string]any{"roles": []string{"owner"}}, http.StatusBadRequest, "invalid"},
		"no roles":               {"PUT", "/admin/accounts/" + ayla.ID + "/roles", map[string]any{"roles": []string{}}, http.StatusBadRequest, "invalid"},
		"taken email":            {"POST", "/admin/accounts", map[string]any{"email": "AYLA@example.com", "display_name": "A", "roles": []string{"analyst"}}, http.StatusConflict, "email_taken"},
		"unknown account":        {"POST", "/admin/accounts/00000000-0000-0000-0000-000000000000/disable", map[string]any{}, http.StatusNotFound, "not_found"},
		"malformed account id":   {"PUT", "/admin/accounts/nope/roles", map[string]any{"roles": []string{"analyst"}}, http.StatusNotFound, "not_found"},
		"malformed body":         {"POST", "/admin/accounts", "[", http.StatusBadRequest, "malformed"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			status, body := a.call(test.method, test.path, root, test.body)
			expect(t, status, body, test.status, test.error)
		})
	}
	status, body := a.call("GET", "/admin/audit", root, nil)
	if status != http.StatusOK || len(body["entries"].([]any)) != 0 {
		t.Fatalf("refused changes left entries: %d %v", status, body)
	}
}

func TestSuperAdminExportsEveryMatchingEntryAndTheExportIsAudited(t *testing.T) {
	a := newAPI(t)
	store := audit.NewStore(a.pool)
	for i := range audit.MaxLimit + 3 {
		entry := audit.Entry{
			Actor: audit.Actor{ID: "seed", Roles: []rbac.Role{rbac.Content}}, Environment: "staging", Action: "edit",
			Object: audit.Object{Type: rbac.CosmeticItems, ID: fmt.Sprintf("avatar.fox%d", i)}, Source: audit.Source{SessionID: "seed"},
		}
		if _, err := store.Append(context.Background(), entry, start); err != nil {
			t.Fatal(err)
		}
	}
	root := a.signIn("root")
	status, body := a.call("POST", "/admin/audit/export?object_type=cosmetic_items", root, map[string]any{})
	expect(t, status, body, http.StatusBadRequest, "reason_required")
	status, body = a.call("POST", "/admin/audit/export?from=yesterday", root, map[string]any{"reason": "regulator request"})
	expect(t, status, body, http.StatusBadRequest, "invalid")
	status, body = a.call("POST", "/admin/audit/export?object_type=cosmetic_items", root, map[string]any{"reason": "regulator request", "ref": "#134"})
	expect(t, status, body, http.StatusOK, "")
	if entries := body["entries"].([]any); len(entries) != audit.MaxLimit+3 {
		t.Fatalf("exported %d entries", len(entries))
	}
	status, body = a.call("GET", "/admin/audit?action=export", root, nil)
	expect(t, status, body, http.StatusOK, "")
	entries := body["entries"].([]any)
	record := entries[0].(map[string]any)
	after := record["after"].(map[string]any)
	if len(entries) != 1 || record["object_type"] != "audit_log" || record["reason"] != "regulator request" || record["ref"] != "#134" ||
		after["filter"] != "object_type=cosmetic_items" || after["entries"] != float64(audit.MaxLimit+3) {
		t.Fatalf("export entry %v", entries)
	}
	a.now = start.Add(15 * time.Minute)
	status, body = a.call("POST", "/admin/audit/export", root, map[string]any{"reason": "regulator request"})
	expect(t, status, body, http.StatusForbidden, "step_up_required")
}

func TestAuditSearchRejectsMalformedFilters(t *testing.T) {
	a := newAPI(t)
	root := a.signIn("root")
	for _, query := range []string{"from=yesterday", "to=2026-11-01", "before_id=-1", "limit=many"} {
		status, body := a.call("GET", "/admin/audit?"+query, root, nil)
		expect(t, status, body, http.StatusBadRequest, "invalid")
	}
}

func TestStepUpNeedsTheSameAccountAndRenewsTheWindow(t *testing.T) {
	a := newAPI(t)
	token := a.signIn("ayla")
	a.now = start.Add(25 * time.Minute)
	status, body := a.call("POST", "/admin/session/step-up", token, map[string]string{"id_token": "root", "nonce": "nonce"})
	expect(t, status, body, http.StatusForbidden, "different_account")
	status, body = a.call("POST", "/admin/session/step-up", token, map[string]string{"id_token": "personal", "nonce": "nonce"})
	expect(t, status, body, http.StatusForbidden, "not_company_account")
	status, body = a.call("POST", "/admin/session/step-up", token, map[string]string{"id_token": "ayla", "nonce": "nonce"})
	if status != http.StatusOK || body["authenticated_at"] != "2026-11-01T09:25:00Z" || body["step_up_until"] != "2026-11-01T09:35:00Z" {
		t.Fatalf("step up %d %v", status, body)
	}
}

func TestAnIdleSessionIsRefused(t *testing.T) {
	a := newAPI(t)
	token := a.signIn("ayla")
	a.now = start.Add(29 * time.Minute)
	status, body := a.call("GET", "/admin/me", token, nil)
	expect(t, status, body, http.StatusOK, "")
	a.now = start.Add(59 * time.Minute)
	status, body = a.call("GET", "/admin/me", token, nil)
	expect(t, status, body, http.StatusUnauthorized, "session_rejected")
}

func TestAnAdminKeepsThePanelLanguageOnTheAccount(t *testing.T) {
	a := newAPI(t)
	token := a.signIn("ayla")
	if _, body := a.call("GET", "/admin/me", token, nil); body["account"].(map[string]any)["locale"] != nil {
		t.Fatalf("a new account has a language: %v", body["account"])
	}
	status, body := a.call("PUT", "/admin/me/locale", token, map[string]any{"locale": "tr"})
	expect(t, status, body, http.StatusBadRequest, "unknown_locale")
	status, body = a.call("PUT", "/admin/me/locale", token, map[string]any{"locale": "en"})
	if status != http.StatusOK || body["locale"] != "en" {
		t.Fatalf("choose %d %v", status, body)
	}
	if _, body := a.call("GET", "/admin/me", a.signIn("ayla"), nil); body["account"].(map[string]any)["locale"] != "en" {
		t.Fatalf("the language did not stay: %v", body["account"])
	}
}
