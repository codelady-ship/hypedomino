package adminapi_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/contentpublish"
	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

type contentStore struct {
	files   map[string]objectstore.Object
	puts    int
	offline bool
}

func (c *contentStore) PutObject(_ context.Context, key string, object objectstore.Object, _ time.Time) error {
	if c.offline {
		return objectstore.ErrUnavailable
	}
	c.files[key] = object
	c.puts++
	return nil
}

var winKey = map[string]any{"note": "Match end title.", "max_length": 12}

func (a *api) raw(method, path, token, body string) *httptest.ResponseRecorder {
	a.t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	a.handler.ServeHTTP(recorder, request)
	return recorder
}

func (a *api) entries(token, query string) []map[string]any {
	a.t.Helper()
	status, body := a.call("GET", "/admin/audit?"+query, token, nil)
	if status != http.StatusOK {
		a.t.Fatalf("audit %d %v", status, body)
	}
	var entries []map[string]any
	for _, entry := range body["entries"].([]any) {
		entries = append(entries, entry.(map[string]any))
	}
	return entries
}

func TestStringRoutesFollowTheRoleMatrix(t *testing.T) {
	a := newAPI(t)
	a.create("leyla@example.com", rbac.LiveOps)
	a.create("orxan@example.com", rbac.Support)
	content, liveOps, support := a.signIn("ayla"), a.signIn("leyla"), a.signIn("orxan")
	routes := []struct {
		name                      string
		method, path              string
		body                      any
		content, liveOps, support int
	}{
		{"list", "GET", "/admin/strings", nil, 200, 200, 200},
		{"coverage", "GET", "/admin/strings/coverage", nil, 200, 200, 200},
		{"unapproved", "POST", "/admin/strings/unapproved", map[string]any{"keys": []string{"match.result.win"}}, 200, 200, 200},
		{"create a key", "PUT", "/admin/strings/match.result.win", winKey, 200, 200, 403},
		{"edit a key", "PUT", "/admin/strings/match.result.win", map[string]any{"note": "Title.", "max_length": 12}, 200, 200, 403},
		{"save a draft", "PUT", "/admin/strings/match.result.win/values/en", map[string]any{"text": "Victory!", "status": "draft"}, 200, 200, 403},
		{"approve", "PUT", "/admin/strings/match.result.win/values/en", map[string]any{"text": "Victory!", "status": "approved"}, 200, 403, 403},
		{"export", "GET", "/admin/strings/export?language=en&format=json", nil, 200, 200, 200},
		{"bundles", "GET", "/admin/strings/bundles?language=en", nil, 200, 200, 200},
		{"publish", "POST", "/admin/strings/bundles/en", map[string]any{}, 200, 403, 403},
		{"deprecate", "PUT", "/admin/strings/match.result.win", map[string]any{"note": "Title.", "max_length": 12, "deprecated": true}, 200, 403, 403},
	}
	for _, route := range routes {
		for _, caller := range []struct {
			token string
			want  int
		}{{support, route.support}, {liveOps, route.liveOps}, {content, route.content}} {
			if status, body := a.call(route.method, route.path, caller.token, route.body); status != caller.want {
				t.Fatalf("%s: got %d %v, want %d", route.name, status, body, caller.want)
			}
		}
	}
	a.create("vusal@example.com", rbac.Analyst)
	status, body := a.call("GET", "/admin/strings", a.signIn("vusal"), nil)
	expect(t, status, body, http.StatusForbidden, "forbidden")
}

func listedKeys(a *api, token, query string) string {
	a.t.Helper()
	status, body := a.call("GET", "/admin/strings?"+query, token, nil)
	if status != http.StatusOK {
		a.t.Fatalf("strings %q: %d %v", query, status, body)
	}
	var keys []string
	for _, entry := range body["strings"].([]any) {
		keys = append(keys, entry.(map[string]any)["key"].(string))
	}
	return strings.Join(keys, ",")
}

func TestStringsPageAndCoverageIsItsOwnReport(t *testing.T) {
	a := newAPI(t)
	content := a.signIn("ayla")
	a.approveEnglishWin(content)
	a.call("PUT", "/admin/strings/match.result.lose", content, winKey)

	if first, rest := listedKeys(a, content, "limit=1"), listedKeys(a, content, "limit=1&after=match.result.lose"); first != "match.result.lose" || rest != "match.result.win" {
		t.Fatalf("pages %q %q", first, rest)
	}
	if found := listedKeys(a, content, "q=WIN"); found != "match.result.win" {
		t.Fatalf("search %q", found)
	}
	if missing := listedKeys(a, content, "missing=en"); missing != "match.result.lose" {
		t.Fatalf("missing in en %q", missing)
	}
	status, body := a.call("GET", "/admin/strings/coverage", content, nil)
	english := body["coverage"].([]any)[1].(map[string]any)
	if status != http.StatusOK || english["language"] != "en" || english["keys"] != float64(2) || english["approved"] != float64(1) || english["percent"] != float64(50) {
		t.Fatalf("coverage %d %v", status, body)
	}
	for _, path := range []string{"/admin/strings?missing=de", "/admin/strings?limit=x", "/admin/strings/bundles", "/admin/strings/bundles?language=de", "/admin/strings/bundles?language=en&before=x"} {
		status, body := a.call("GET", path, content, nil)
		expect(t, status, body, http.StatusBadRequest, "invalid")
	}
}

func unapprovedKeys(a *api, token string, body any) string {
	a.t.Helper()
	status, reply := a.call("POST", "/admin/strings/unapproved", token, body)
	if status != http.StatusOK {
		a.t.Fatalf("unapproved %v: %d %v", body, status, reply)
	}
	var keys []string
	for _, key := range reply["keys"].([]any) {
		keys = append(keys, key.(string))
	}
	return strings.Join(keys, ",")
}

func TestUnapprovedListsTheKeysWithoutAnApprovedEnglishValueInTheirOrder(t *testing.T) {
	a := newAPI(t)
	content := a.signIn("ayla")
	a.approveEnglishWin(content)
	a.call("PUT", "/admin/strings/match.result.lose", content, winKey)
	a.call("PUT", "/admin/strings/match.result.lose/values/en", content, map[string]any{"text": "Defeat", "status": "draft"})
	a.call("PUT", "/admin/strings/match.result.draw", content, winKey)
	a.call("PUT", "/admin/strings/match.result.draw/values/az", content, map[string]any{"text": "Heç-heçə", "status": "approved"})

	keys := []string{"mission.unknown.name", "match.result.win", "match.result.lose", "match.result.draw"}
	if got := unapprovedKeys(a, content, map[string]any{"keys": keys}); got != "mission.unknown.name,match.result.lose,match.result.draw" {
		t.Fatalf("unapproved %q", got)
	}
	if got := unapprovedKeys(a, content, map[string]any{"keys": []string{"match.result.win"}}); got != "" {
		t.Fatalf("approved key listed %q", got)
	}
	if got := unapprovedKeys(a, content, map[string]any{}); got != "" {
		t.Fatalf("no keys %q", got)
	}
	status, body := a.call("POST", "/admin/strings/unapproved", content, []string{"match.result.win"})
	expect(t, status, body, http.StatusBadRequest, "malformed")
}

func TestKeyAndValueChangesAreAudited(t *testing.T) {
	a := newAPI(t)
	content := a.signIn("ayla")
	status, body := a.call("PUT", "/admin/strings/match.result.win", content, winKey)
	expect(t, status, body, http.StatusOK, "")
	a.call("PUT", "/admin/strings/match.result.win", content, map[string]any{"note": "Title.", "max_length": 10, "reason": "shorter"})
	a.call("PUT", "/admin/strings/match.result.win/values/az", content, map[string]any{"text": "Qələbə", "status": "draft"})
	status, body = a.call("PUT", "/admin/strings/match.result.win/values/az", content, map[string]any{"text": "Qələbə!", "status": "approved", "ref": "#214"})
	if status != http.StatusOK || body["status"] != "approved" {
		t.Fatalf("approve %d %v", status, body)
	}
	a.call("PUT", "/admin/strings/match.result.win", content, map[string]any{"note": "Title.", "max_length": 10, "deprecated": true})

	status, body = a.call("GET", "/admin/strings", content, nil)
	listed := body["strings"].([]any)[0].(map[string]any)
	if status != http.StatusOK || listed["deprecated"] != true || listed["values"].(map[string]any)["az"].(map[string]any)["text"] != "Qələbə!" {
		t.Fatalf("list %d %v", status, body)
	}
	entries := a.entries(content, "object_type=game_strings&object_id=match.result.win")
	actions := []string{}
	for _, entry := range entries {
		actions = append(actions, entry["action"].(string))
	}
	if strings.Join(actions, ",") != "retire,approve,edit,edit,create" {
		t.Fatalf("actions %v", actions)
	}
	approve, draft, edit := entries[1], entries[2], entries[3]
	if approve["ref"] != "#214" || approve["before"].(map[string]any)["status"] != "draft" || approve["after"].(map[string]any)["text"] != "Qələbə!" {
		t.Fatalf("approve entry %v", approve)
	}
	if draft["before"].(map[string]any)["status"] != "missing" || draft["after"].(map[string]any)["language"] != "az" {
		t.Fatalf("draft entry %v", draft)
	}
	if edit["reason"] != "shorter" || edit["before"].(map[string]any)["max_length"] != float64(12) || edit["after"].(map[string]any)["max_length"] != float64(10) {
		t.Fatalf("edit entry %v", edit)
	}
}

func TestRefusedStringChangesLeaveNoEntry(t *testing.T) {
	a := newAPI(t)
	content := a.signIn("ayla")
	a.call("PUT", "/admin/strings/match.result.win", content, winKey)
	tests := map[string]struct {
		method, path string
		body         any
		status       int
		error        string
	}{
		"bad key":                 {"PUT", "/admin/strings/Win", winKey, http.StatusBadRequest, "invalid"},
		"too long":                {"PUT", "/admin/strings/match.result.win/values/en", map[string]any{"text": "A very long victory title", "status": "approved"}, http.StatusBadRequest, "invalid"},
		"broken message":          {"PUT", "/admin/strings/match.result.win/values/en", map[string]any{"text": "Win {", "status": "approved"}, http.StatusBadRequest, "invalid"},
		"unknown language":        {"PUT", "/admin/strings/match.result.win/values/de", map[string]any{"text": "Sieg", "status": "draft"}, http.StatusBadRequest, "invalid"},
		"missing status":          {"PUT", "/admin/strings/match.result.win/values/en", map[string]any{"text": "Win"}, http.StatusBadRequest, "invalid"},
		"unknown key":             {"PUT", "/admin/strings/match.result.lose/values/en", map[string]any{"text": "Lost", "status": "draft"}, http.StatusNotFound, "not_found"},
		"unknown language bundle": {"POST", "/admin/strings/bundles/de", map[string]any{}, http.StatusBadRequest, "invalid"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			status, body := a.call(test.method, test.path, content, test.body)
			expect(t, status, body, test.status, test.error)
			if test.status == http.StatusBadRequest && body["detail"] == "" {
				t.Fatalf("no detail in %v", body)
			}
		})
	}
	if entries := a.entries(content, "object_type=game_strings"); len(entries) != 1 {
		t.Fatalf("entries %v", entries)
	}
}

func TestImportStoresDraftsWithAnEntryPerKey(t *testing.T) {
	a := newAPI(t)
	content := a.signIn("ayla")
	a.call("PUT", "/admin/strings/match.result.win", content, winKey)
	a.call("PUT", "/admin/strings/match.result.lose", content, map[string]any{"note": "Match end title.", "max_length": 12})
	a.call("PUT", "/admin/strings/match.result.win/values/tr", content, map[string]any{"text": "Zafer", "status": "approved"})

	export := a.raw("GET", "/admin/strings/export?language=tr&format=csv", content, "")
	if export.Code != http.StatusOK || export.Header().Get("Content-Type") != "text/csv; charset=utf-8" ||
		export.Header().Get("Content-Disposition") != `attachment; filename="strings-tr.csv"` || !strings.Contains(export.Body.String(), "Zafer") {
		t.Fatalf("export %d %v %s", export.Code, export.Header(), export.Body)
	}
	blocked := a.raw("POST", "/admin/strings/import?language=tr&format=json", content,
		`{"language":"tr","rows":[{"key":"match.result.win","text":"Zafer {"},{"key":"match.result.draw","text":"Berabere"}]}`)
	if blocked.Code != http.StatusUnprocessableEntity || !strings.Contains(blocked.Body.String(), `"key":"match.result.draw"`) {
		t.Fatalf("blocked import %d %s", blocked.Code, blocked.Body)
	}
	before := len(a.entries(content, "object_type=game_strings"))
	imported := a.raw("POST", "/admin/strings/import?language=tr&format=json", content,
		`{"language":"tr","rows":[{"key":"match.result.win","text":"Kazandın!"},{"key":"match.result.lose","text":"Kaybettin"}]}`)
	if imported.Code != http.StatusOK || imported.Body.String() != "{\"changed\":2}\n" {
		t.Fatalf("import %d %s", imported.Code, imported.Body)
	}
	entries := a.entries(content, "object_type=game_strings")
	if len(entries) != before+2 {
		t.Fatalf("import entries %v", entries)
	}
	for _, entry := range entries[:2] {
		after := entry["after"].(map[string]any)
		if entry["action"] != "edit" || entry["reason"] != "import" || after["status"] != "draft" || after["language"] != "tr" {
			t.Fatalf("import entry %v", entry)
		}
		if entry["object_id"] == "match.result.win" && entry["before"].(map[string]any)["status"] != "approved" {
			t.Fatalf("win before %v", entry)
		}
	}
	if bad := a.raw("POST", "/admin/strings/import?language=tr&format=xml", content, "{}"); bad.Code != http.StatusBadRequest {
		t.Fatalf("unknown format %d %s", bad.Code, bad.Body)
	}
}

func TestPublishingIsAuditedAndBlockedByFailingValues(t *testing.T) {
	a := newAPI(t)
	content := a.signIn("ayla")
	a.call("PUT", "/admin/strings/match.result.win", content, winKey)
	a.call("PUT", "/admin/strings/match.result.win/values/en", content, map[string]any{"text": "Victory!", "status": "approved"})
	status, body := a.call("POST", "/admin/strings/bundles/en", content, map[string]any{"reason": "launch"})
	if status != http.StatusOK || body["version"] != float64(1) || body["hash"] == "" {
		t.Fatalf("publish %d %v", status, body)
	}
	a.call("PUT", "/admin/strings/match.result.win", content, map[string]any{"note": "Title.", "max_length": 3})
	status, body = a.call("POST", "/admin/strings/bundles/en", content, map[string]any{})
	if status != http.StatusUnprocessableEntity || body["error"] != "blocked" || body["problems"].([]any)[0].(map[string]any)["key"] != "match.result.win" {
		t.Fatalf("blocked publish %d %v", status, body)
	}
	status, body = a.call("GET", "/admin/strings/bundles?language=en", content, nil)
	bundles := body["bundles"].([]any)
	if status != http.StatusOK || len(bundles) != 1 || bundles[0].(map[string]any)["by"] == "" {
		t.Fatalf("bundles %d %v", status, body)
	}
	entries := a.entries(content, "object_type=game_strings&action=publish")
	if len(entries) != 1 || entries[0]["object_id"] != "bundle:en" || entries[0]["reason"] != "launch" || entries[0]["after"].(map[string]any)["version"] != float64(1) {
		t.Fatalf("publish entries %v", entries)
	}
}

func (a *api) approveEnglishWin(token string) {
	a.t.Helper()
	a.call("PUT", "/admin/strings/match.result.win", token, winKey)
	a.call("PUT", "/admin/strings/match.result.win/values/en", token, map[string]any{"text": "Victory!", "status": "approved"})
}

func TestPublishingUploadsTheBundleToTheContentStoreUnderItsHash(t *testing.T) {
	a := newAPI(t)
	content := a.signIn("ayla")
	a.approveEnglishWin(content)
	status, body := a.call("POST", "/admin/strings/bundles/en", content, map[string]any{})
	if status != http.StatusOK {
		t.Fatalf("publish %d %v", status, body)
	}
	hash := body["hash"].(string)
	file, ok := a.content.files["strings/en/"+hash+".json"]
	if !ok || len(a.content.files) != 1 {
		t.Fatalf("uploaded %v", a.content.files)
	}
	sum := sha256.Sum256(file.Body)
	if hex.EncodeToString(sum[:]) != hash || file.ContentType != "application/json" || file.CacheControl != contentpublish.ImmutableCacheControl {
		t.Fatalf("file %s %s %s", file.ContentType, file.CacheControl, file.Body)
	}
	var bundle struct {
		Language string            `json:"language"`
		Strings  map[string]string `json:"strings"`
	}
	if err := json.Unmarshal(file.Body, &bundle); err != nil || bundle.Language != "en" || !reflect.DeepEqual(bundle.Strings, map[string]string{"match.result.win": "Victory!"}) {
		t.Fatalf("bundle %+v %v", bundle, err)
	}

	status, body = a.call("POST", "/admin/strings/bundles/en", content, map[string]any{})
	if status != http.StatusOK || body["version"] != float64(1) || body["hash"] != hash || a.content.puts != 2 || len(a.content.files) != 1 {
		t.Fatalf("unchanged republish %d %v, %d puts", status, body, a.content.puts)
	}
}

func TestABundleTheContentStoreRefusesIsNotPublished(t *testing.T) {
	cases := map[string]struct {
		adjust func(*adminapi.Config, *contentStore)
		want   string
	}{
		"no content store":      {func(c *adminapi.Config, _ *contentStore) { c.Content = nil }, "content_store_off"},
		"content store offline": {func(_ *adminapi.Config, store *contentStore) { store.offline = true }, "content_store_unavailable"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			store := &contentStore{files: map[string]objectstore.Object{}}
			a := newAPIWith(t, func(config *adminapi.Config) {
				config.Content = store
				c.adjust(config, store)
			})
			content := a.signIn("ayla")
			a.approveEnglishWin(content)
			status, body := a.call("POST", "/admin/strings/bundles/en", content, map[string]any{})
			if status != http.StatusServiceUnavailable || body["error"] != c.want {
				t.Fatalf("publish %d %v", status, body)
			}
			if _, body := a.call("GET", "/admin/strings/bundles?language=en", content, nil); len(body["bundles"].([]any)) != 0 {
				t.Fatalf("bundles %v", body)
			}
			if entries := a.entries(content, "object_type=game_strings&action=publish"); len(entries) != 0 {
				t.Fatalf("publish entries %v", entries)
			}
		})
	}
}
