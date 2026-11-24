package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestHistogramWritesCumulativeBuckets(t *testing.T) {
	h := NewHistogram("move_roundtrip_seconds", "Move intent receive to broadcast.", []time.Duration{10 * time.Millisecond, 100 * time.Millisecond})
	for _, elapsed := range []time.Duration{5 * time.Millisecond, 10 * time.Millisecond, 50 * time.Millisecond, 2 * time.Second} {
		h.Observe(elapsed)
	}
	var out strings.Builder
	if _, err := h.WriteTo(&out); err != nil {
		t.Fatal(err)
	}
	want := `# HELP move_roundtrip_seconds Move intent receive to broadcast.
# TYPE move_roundtrip_seconds histogram
move_roundtrip_seconds_bucket{le="0.01"} 2
move_roundtrip_seconds_bucket{le="0.1"} 3
move_roundtrip_seconds_bucket{le="+Inf"} 4
move_roundtrip_seconds_sum 2.065
move_roundtrip_seconds_count 4
`
	if out.String() != want {
		t.Fatalf("got\n%s", out.String())
	}
}

func TestHistogramObserveDoesNotAllocate(t *testing.T) {
	h := NewHistogram("x", "x", []time.Duration{time.Millisecond, time.Second})
	if allocs := testing.AllocsPerRun(100, func() { h.Observe(3 * time.Millisecond) }); allocs != 0 {
		t.Fatalf("%v allocations", allocs)
	}
}

func TestGaugeWritesLargeValuesWithoutExponent(t *testing.T) {
	var out strings.Builder
	if err := WriteGauge(&out, "go_heap_live_bytes", "Live heap.", 40566560); err != nil {
		t.Fatal(err)
	}
	if want := "# HELP go_heap_live_bytes Live heap.\n# TYPE go_heap_live_bytes gauge\ngo_heap_live_bytes 40566560\n"; out.String() != want {
		t.Fatalf("got %q", out.String())
	}
}
