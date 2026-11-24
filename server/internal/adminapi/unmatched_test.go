package adminapi_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/adminapi"
	"github.com/Alievs-corp/hypedomino/server/internal/ratelimit"
)

func unmatchedCall(t *testing.T, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	handler := adminapi.New(adminapi.Config{Limits: unthrottled}, identities{}, ratelimit.NewMemory(), nil, time.Now, slog.New(slog.NewTextHandler(io.Discard, nil)))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, path, strings.NewReader("{}")))
	return recorder
}

func expectHardenedJSON(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if recorder.Code != status || strings.TrimSpace(recorder.Body.String()) != `{"error":"`+code+`"}` {
		t.Fatalf("got %d %s, want %d %s", recorder.Code, recorder.Body.String(), status, code)
	}
	for header, want := range map[string]string{"Content-Type": "application/json", "Cache-Control": "no-store", "X-Content-Type-Options": "nosniff"} {
		if got := recorder.Header().Get(header); got != want {
			t.Errorf("%s %q, want %q", header, got, want)
		}
	}
}

func TestUnknownPathAnswersJSONNotFound(t *testing.T) {
	for _, path := range []string{"/", "/admin/unknown", "/admin/players/p1/unknown", "/.env"} {
		expectHardenedJSON(t, unmatchedCall(t, "GET", path), http.StatusNotFound, "not_found")
	}
}

func TestWrongMethodAnswersJSONMethodNotAllowed(t *testing.T) {
	recorder := unmatchedCall(t, "PATCH", "/admin/me")
	expectHardenedJSON(t, recorder, http.StatusMethodNotAllowed, "method_not_allowed")
	if allow := recorder.Header().Get("Allow"); !strings.Contains(allow, "GET") {
		t.Fatalf("Allow %q, want the methods of /admin/me", allow)
	}
}
