package app

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestEveryRequestGetsACorrelationIdAndIsCounted(t *testing.T) {
	logs := &strings.Builder{}
	a := New(DefaultConfig(), NewLogger(logs, "info"), &seatHandler{})
	handler := a.Handler()

	kept := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set(RequestIDHeader, "client-req-0001")
	handler.ServeHTTP(kept, request)

	fresh := httptest.NewRecorder()
	unknown := httptest.NewRequest(http.MethodGet, "/nowhere", nil)
	unknown.Header.Set(RequestIDHeader, "bad id!")
	handler.ServeHTTP(fresh, unknown)

	if kept.Code != http.StatusOK || kept.Header().Get(RequestIDHeader) != "client-req-0001" {
		t.Fatalf("kept %d %q", kept.Code, kept.Header().Get(RequestIDHeader))
	}
	generated := fresh.Header().Get(RequestIDHeader)
	if fresh.Code != http.StatusNotFound || !regexp.MustCompile(`^[0-9a-f]{16}$`).MatchString(generated) {
		t.Fatalf("fresh %d %q", fresh.Code, generated)
	}
	for _, line := range []string{`"request_id":"client-req-0001","method":"GET","path":"/healthz","status":200`, `"request_id":"` + generated + `","method":"GET","path":"/nowhere","status":404`} {
		if !strings.Contains(logs.String(), line) {
			t.Fatalf("no %s in\n%s", line, logs.String())
		}
	}
	metrics := &strings.Builder{}
	a.responses.WriteTo(metrics)
	for _, sample := range []string{`hd_http_responses_total{class="2xx"} 1`, `hd_http_responses_total{class="4xx"} 1`, `hd_http_responses_total{class="5xx"} 0`} {
		if !strings.Contains(metrics.String(), sample) {
			t.Fatalf("no %s in\n%s", sample, metrics.String())
		}
	}
}
