package metrics

import (
	"strings"
	"testing"
	"time"
)

func TestCounterWritesSortedSeries(t *testing.T) {
	c := NewCounter("hd_logins_total", "Sign-ins.", Labels{"provider", "result"})
	c.Inc(Labels{"google", "ok"})
	c.Inc(Labels{"apple", "token_rejected"})
	c.Inc(Labels{"google", "ok"})
	var out strings.Builder
	if _, err := c.WriteTo(&out); err != nil {
		t.Fatal(err)
	}
	want := `# HELP hd_logins_total Sign-ins.
# TYPE hd_logins_total counter
hd_logins_total{provider="apple",result="token_rejected"} 1
hd_logins_total{provider="google",result="ok"} 2
`
	if out.String() != want {
		t.Fatalf("got\n%s", out.String())
	}
}

func TestCounterWithoutLabelsWritesBareSeries(t *testing.T) {
	c := NewCounter("hd_left_total", "Left.", Labels{})
	c.Inc(Labels{})
	var out strings.Builder
	c.WriteTo(&out)
	if want := "# HELP hd_left_total Left.\n# TYPE hd_left_total counter\nhd_left_total 1\n"; out.String() != want {
		t.Fatalf("got %q", out.String())
	}
}

func TestHistogramsWriteOneHeaderAndLabeledSeries(t *testing.T) {
	h := NewHistograms("wait_seconds", "Wait.", Labels{"queue", "mode"}, []time.Duration{time.Second})
	h.Observe(Labels{"ranked", "1v1"}, 500*time.Millisecond)
	h.Observe(Labels{"casual", "2v2"}, 2*time.Second)
	var out strings.Builder
	if _, err := h.WriteTo(&out); err != nil {
		t.Fatal(err)
	}
	want := `# HELP wait_seconds Wait.
# TYPE wait_seconds histogram
wait_seconds_bucket{queue="casual",mode="2v2",le="1"} 0
wait_seconds_bucket{queue="casual",mode="2v2",le="+Inf"} 1
wait_seconds_sum{queue="casual",mode="2v2"} 2
wait_seconds_count{queue="casual",mode="2v2"} 1
wait_seconds_bucket{queue="ranked",mode="1v1",le="1"} 1
wait_seconds_bucket{queue="ranked",mode="1v1",le="+Inf"} 1
wait_seconds_sum{queue="ranked",mode="1v1"} 0.5
wait_seconds_count{queue="ranked",mode="1v1"} 1
`
	if out.String() != want {
		t.Fatalf("got\n%s", out.String())
	}
}

func TestLabeledObserveOfAKnownSeriesDoesNotAllocate(t *testing.T) {
	c := NewCounter("c", "c", Labels{"store", "result"})
	h := NewHistograms("h", "h", Labels{"store"}, []time.Duration{time.Second})
	c.Inc(Labels{"google", "valid"})
	h.Observe(Labels{"google"}, time.Millisecond)
	allocs := testing.AllocsPerRun(100, func() {
		c.Inc(Labels{"google", "valid"})
		h.Observe(Labels{"google"}, time.Millisecond)
	})
	if allocs != 0 {
		t.Fatalf("%v allocations", allocs)
	}
}

func BenchmarkCounterInc(b *testing.B) {
	c := NewCounter("c", "c", Labels{"provider", "result"})
	b.ReportAllocs()
	for b.Loop() {
		c.Inc(Labels{"google", "ok"})
	}
}

func BenchmarkHistogramsObserve(b *testing.B) {
	h := NewHistograms("h", "h", Labels{"queue", "mode"}, []time.Duration{time.Second, time.Minute})
	b.ReportAllocs()
	for b.Loop() {
		h.Observe(Labels{"ranked", "2v2"}, 3*time.Second)
	}
}
