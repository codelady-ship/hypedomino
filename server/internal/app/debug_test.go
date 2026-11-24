package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/Alievs-corp/hypedomino/server/internal/botclient"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestMovesOfALiveMatchReachTheMetrics(t *testing.T) {
	r := start(t, DefaultConfig())
	r.openRoom(rules.Mode1v1)
	x, y := r.dial(rules.SeatX), r.dial(rules.SeatY)
	finished := make(chan error, 2)
	for _, c := range []*botclient.Client{x, y} {
		go func() {
			_, err := c.PlayUntil(botclient.Finished)
			finished <- err
		}()
	}
	for range 2 {
		if err := <-finished; err != nil {
			t.Fatal(err)
		}
	}
	w := httptest.NewRecorder()
	r.app.metricsHandler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	count := regexp.MustCompile(`(?m)^move_roundtrip_seconds_count (\d+)$`).FindStringSubmatch(w.Body.String())
	if w.Code != http.StatusOK || count == nil || count[1] == "0" {
		t.Fatalf("status %d metrics:\n%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `move_roundtrip_seconds_bucket{le="0.1"}`) {
		t.Fatal("no bucket at the 100 ms budget")
	}
	for _, gauge := range []string{`(?m)^go_heap_live_bytes [1-9]`, `(?m)^go_goroutines [1-9]`, `(?m)^hd_rooms \d+$`, `(?m)^hd_connections \d+$`, `(?m)^hd_http_responses_total\{class="1xx"\} [1-9]`, `(?m)^# TYPE hd_panics_total counter$`} {
		if !regexp.MustCompile(gauge).MatchString(w.Body.String()) {
			t.Fatalf("missing %s in\n%s", gauge, w.Body.String())
		}
	}
	if status, _ := r.get("/metrics"); status != http.StatusNotFound {
		t.Fatalf("metrics on the public listener: %d", status)
	}
}

func TestRunFailsOnABadSideListenerAddress(t *testing.T) {
	for _, bad := range []func(*Config){
		func(c *Config) { c.DebugAddr = "256.0.0.1:1" },
		func(c *Config) { c.PprofAddr = "256.0.0.1:1" },
	} {
		config := DefaultConfig()
		config.ListenAddr = "127.0.0.1:0"
		bad(&config)
		if err := New(config, NewLogger(&strings.Builder{}, "info"), &seatHandler{}).Run(t.Context()); err == nil {
			t.Fatalf("Run started without its side listener: debug %q pprof %q", config.DebugAddr, config.PprofAddr)
		}
	}
}

func TestProfilesAreServedOnlyByThePprofListener(t *testing.T) {
	config := DefaultConfig()
	config.DebugAddr = "127.0.0.1:0"
	config.PprofAddr = "127.0.0.1:0"
	logs := &strings.Builder{}
	stop, err := New(config, NewLogger(logs, "info"), &seatHandler{}).serveDebug()
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	addrs := listenerAddrs(t, logs.String())
	for _, c := range []struct {
		listener, path string
		want           int
	}{
		{"debug listening", "/metrics", http.StatusOK},
		{"debug listening", "/debug/pprof/heap", http.StatusNotFound},
		{"pprof listening", "/debug/pprof/heap", http.StatusOK},
		{"pprof listening", "/metrics", http.StatusNotFound},
	} {
		response, err := http.Get("http://" + addrs[c.listener] + c.path)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != c.want {
			t.Fatalf("%s %s: status %d, want %d", c.listener, c.path, response.StatusCode, c.want)
		}
	}
}

func listenerAddrs(t *testing.T, logs string) map[string]string {
	addrs := map[string]string{}
	for line := range strings.Lines(logs) {
		var entry struct{ Msg, Addr string }
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatal(err)
		}
		addrs[entry.Msg] = entry.Addr
	}
	return addrs
}
