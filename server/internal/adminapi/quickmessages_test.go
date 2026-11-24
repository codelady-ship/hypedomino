package adminapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

func (a *api) translate(token, key string, languages ...string) {
	a.t.Helper()
	if status, body := a.call("PUT", "/admin/strings/"+key, token, map[string]any{"note": "Quick message.", "max_length": 24}); status != http.StatusOK {
		a.t.Fatalf("entry %s: %d %v", key, status, body)
	}
	for _, language := range languages {
		if status, body := a.call("PUT", "/admin/strings/"+key+"/values/"+language, token, map[string]any{"text": "Hello", "status": "approved"}); status != http.StatusOK {
			a.t.Fatalf("value %s %s: %d %v", key, language, status, body)
		}
	}
}

func TestQuickMessagesArePublishedOnlyWithEveryLanguageApproved(t *testing.T) {
	a := newAPI(t)
	a.create("orxan@example.com", rbac.Support)
	content, support := a.signIn("ayla"), a.signIn("orxan")
	a.translate(content, "quick.hello", "az", "en", "tr", "ru")
	a.translate(content, "quick.bye", "en")
	path := "/admin/config/quick-messages"

	status, body := a.call("PUT", path, support, map[string]any{"catalog": map[string]any{"keys": []string{"quick.hello"}}, "reason": "launch"})
	expect(t, status, body, http.StatusForbidden, "forbidden")
	status, body = a.call("PUT", path, content, map[string]any{"catalog": map[string]any{"keys": []string{"quick.hello", "quick.bye"}}, "reason": "launch"})
	expect(t, status, body, http.StatusBadRequest, "invalid")
	if detail, _ := body["detail"].(string); !strings.Contains(detail, "quick.bye") || strings.Contains(detail, "quick.hello") {
		t.Fatalf("detail %v", body)
	}
	status, body = a.call("PUT", path, content, map[string]any{"catalog": map[string]any{"keys": []string{"match.result.win"}}, "reason": "launch"})
	expect(t, status, body, http.StatusBadRequest, "invalid")

	status, body = a.call("PUT", path, content, map[string]any{"catalog": map[string]any{"keys": []string{"quick.hello"}}, "reason": "launch"})
	if status != http.StatusOK || body["config_version"] != float64(1) {
		t.Fatalf("publish %d %v", status, body)
	}
	status, body = a.call("GET", path, support, nil)
	if keys := body["catalog"].(map[string]any)["keys"].([]any); status != http.StatusOK || len(keys) != 1 || keys[0] != "quick.hello" {
		t.Fatalf("read %d %v", status, body)
	}
}
