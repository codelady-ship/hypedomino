package loadbot

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestScrapeNodeReadsHeapAndGoroutinesFromTheMetrics(t *testing.T) {
	bodies := map[string]string{
		"/metrics": "# HELP go_heap_live_bytes Heap.\n# TYPE go_heap_live_bytes gauge\ngo_heap_live_bytes 123456789\n" +
			"move_roundtrip_seconds_count 7\ngo_goroutines 4321\n",
		"/partial":   "go_heap_live_bytes 1\n",
		"/malformed": "go_heap_live_bytes x\ngo_goroutines 1\n",
	}
	node := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := bodies[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(body))
	}))
	defer node.Close()
	sample, err := ScrapeNode(node.URL + "/metrics")()
	if err != nil || sample.HeapBytes != 123456789 || sample.Goroutines != 4321 {
		t.Fatalf("sample %+v, %v", sample, err)
	}
	for _, path := range []string{"/missing", "/partial", "/malformed"} {
		if sample, err := ScrapeNode(node.URL + path)(); err == nil {
			t.Fatalf("%s read as %+v", path, sample)
		}
	}
}
