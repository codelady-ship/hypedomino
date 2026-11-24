package loadbot

import (
	"fmt"
	"strings"
	"time"
)

type WindowReport struct {
	Index int
	Moves uint64
	P50   time.Duration
	P99   time.Duration
	Max   time.Duration
}

type Report struct {
	Duration          time.Duration
	Rooms             int
	PeakConnections   int64
	Matches           uint64
	Moves             uint64
	Mean              time.Duration
	P50               time.Duration
	P95               time.Duration
	P99               time.Duration
	Max               time.Duration
	SteadyMoves       uint64
	SteadyP99         time.Duration
	WorstWindowP99    time.Duration
	Windows           []WindowReport
	Rejected          uint64
	DialFailures      uint64
	ReadFailures      uint64
	Drops             uint64
	Reconnects        uint64
	StormAt           time.Duration
	StormDrops        uint64
	StormRecovered    uint64
	StormRecovery     time.Duration
	ErrorRate         float64
	BytesPerIntent    float64
	Node              []NodeSample
	HeapDriftPct      float64
	GoroutineDriftPct float64
	Budget            time.Duration
	Pass              bool
	Failures          []string
}

func (r *Runner) report() *Report {
	rep := &Report{
		Duration:        time.Since(r.start),
		Rooms:           r.config.Rooms,
		PeakConnections: r.counters.Peak.Load(),
		Matches:         r.counters.Matches.Load(),
		Moves:           r.overall.Count(),
		Mean:            r.overall.Mean(),
		P50:             r.overall.Percentile(50),
		P95:             r.overall.Percentile(95),
		P99:             r.overall.Percentile(99),
		Max:             r.overall.Max(),
		SteadyMoves:     r.steady.Count(),
		SteadyP99:       r.steady.Percentile(99),
		Rejected:        r.counters.Rejected.Load(),
		DialFailures:    r.counters.DialFailures.Load(),
		ReadFailures:    r.counters.ReadFailures.Load(),
		Drops:           r.counters.Drops.Load(),
		Reconnects:      r.counters.Reconnects.Load(),
		StormAt:         r.config.StormAt,
		StormDrops:      r.counters.StormDrops.Load(),
		StormRecovered:  r.counters.Recovered.Load(),
		StormRecovery:   time.Duration(r.recovery.Load()),
		Budget:          r.config.Budget,
	}
	warmupWindows := int(r.config.Warmup / r.config.Window)
	for index, slot := range r.windows.Slots() {
		if slot.Count() == 0 {
			continue
		}
		window := WindowReport{Index: index, Moves: slot.Count(), P50: slot.Percentile(50), P99: slot.Percentile(99), Max: slot.Max()}
		rep.Windows = append(rep.Windows, window)
		if index >= warmupWindows {
			rep.WorstWindowP99 = max(rep.WorstWindowP99, window.P99)
		}
	}
	failures := rep.Rejected + rep.DialFailures + rep.ReadFailures
	if attempts := rep.Moves + rep.Reconnects + uint64(rep.PeakConnections); attempts > 0 {
		rep.ErrorRate = float64(failures) / float64(attempts)
	}
	if sent := r.counters.Sent.Load(); sent > 0 {
		rep.BytesPerIntent = float64(r.counters.SentBytes.Load()) / float64(sent)
	}
	r.nodeMu.Lock()
	rep.Node = append([]NodeSample(nil), r.node...)
	r.nodeMu.Unlock()
	rep.HeapDriftPct = drift(rep.Node, r.config.Ramp+r.config.Warmup, heapOf)
	rep.GoroutineDriftPct = drift(rep.Node, r.config.Ramp+r.config.Warmup, goroutinesOf)
	rep.judge()
	return rep
}

func heapOf(sample NodeSample) uint64       { return sample.HeapBytes }
func goroutinesOf(sample NodeSample) uint64 { return sample.Goroutines }

func drift(samples []NodeSample, warmup time.Duration, value func(NodeSample) uint64) float64 {
	var steady []uint64
	for _, sample := range samples {
		if sample.At >= warmup && value(sample) > 0 {
			steady = append(steady, value(sample))
		}
	}
	if len(steady) < 2 {
		return 0
	}
	first, last := steady[0], steady[len(steady)-1]
	return (float64(last) - float64(first)) / float64(first) * 100
}

func (rep *Report) judge() {
	p99 := rep.P99
	if rep.SteadyMoves > 0 {
		p99 = rep.SteadyP99
	}
	if rep.Moves == 0 {
		rep.Failures = append(rep.Failures, "no move was measured")
	}
	if p99 >= rep.Budget {
		rep.Failures = append(rep.Failures, fmt.Sprintf("p99 %s is not under the budget %s", p99, rep.Budget))
	}
	if rep.WorstWindowP99 >= rep.Budget {
		rep.Failures = append(rep.Failures, fmt.Sprintf("worst window p99 %s is not under the budget %s", rep.WorstWindowP99, rep.Budget))
	}
	if rep.ErrorRate >= 0.001 {
		rep.Failures = append(rep.Failures, fmt.Sprintf("error rate %.3f%% is not under 0.1%%", rep.ErrorRate*100))
	}
	if rep.HeapDriftPct > 5 {
		rep.Failures = append(rep.Failures, fmt.Sprintf("heap grew %.1f%% during the steady state", rep.HeapDriftPct))
	}
	if rep.GoroutineDriftPct > 5 {
		rep.Failures = append(rep.Failures, fmt.Sprintf("goroutines grew %.1f%% during the steady state", rep.GoroutineDriftPct))
	}
	rep.Pass = len(rep.Failures) == 0
}

func (rep *Report) Text() string {
	b := &strings.Builder{}
	fmt.Fprintf(b, "duration %s, rooms %d, peak connections %d, matches finished %d\n", rep.Duration.Round(time.Second), rep.Rooms, rep.PeakConnections, rep.Matches)
	fmt.Fprintf(b, "moves %d: mean %s p50 %s p95 %s p99 %s max %s\n", rep.Moves, rep.Mean, rep.P50, rep.P95, rep.P99, rep.Max)
	fmt.Fprintf(b, "steady state: moves %d p99 %s, worst window p99 %s, budget %s\n", rep.SteadyMoves, rep.SteadyP99, rep.WorstWindowP99, rep.Budget)
	fmt.Fprintf(b, "drops %d reconnects %d rejected %d dial failures %d read failures %d, error rate %.4f%%\n", rep.Drops, rep.Reconnects, rep.Rejected, rep.DialFailures, rep.ReadFailures, rep.ErrorRate*100)
	fmt.Fprintf(b, "bytes per intent %.1f\n", rep.BytesPerIntent)
	if rep.StormAt > 0 {
		fmt.Fprintf(b, "storm at %s: dropped %d, back in their match %d, slowest return %s\n", rep.StormAt, rep.StormDrops, rep.StormRecovered, rep.StormRecovery.Round(time.Millisecond))
	}
	if len(rep.Node) > 0 {
		first, last := rep.Node[0], rep.Node[len(rep.Node)-1]
		fmt.Fprintf(b, "heap: first %d bytes, last %d bytes, steady drift %.1f%%\n", first.HeapBytes, last.HeapBytes, rep.HeapDriftPct)
		fmt.Fprintf(b, "goroutines: first %d, last %d, steady drift %.1f%%\n", first.Goroutines, last.Goroutines, rep.GoroutineDriftPct)
	}
	fmt.Fprintln(b, "window  moves  p50  p99  max")
	for _, window := range rep.Windows {
		fmt.Fprintf(b, "%6d %6d %4s %4s %4s\n", window.Index, window.Moves, window.P50, window.P99, window.Max)
	}
	if rep.Pass {
		fmt.Fprintln(b, "PASS")
	} else {
		fmt.Fprintln(b, "FAIL: "+strings.Join(rep.Failures, "; "))
	}
	return b.String()
}
