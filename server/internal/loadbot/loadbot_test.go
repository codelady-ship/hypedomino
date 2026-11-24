package loadbot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/app"
	"github.com/Alievs-corp/hypedomino/server/internal/bot"
	"github.com/Alievs-corp/hypedomino/server/internal/botclient"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestHistogramPercentilesAndWindows(t *testing.T) {
	h := &Histogram{}
	if h.Percentile(99) != 0 || h.Mean() != 0 || h.Max() != 0 {
		t.Fatal("empty histogram is zero")
	}
	for i := 1; i <= 100; i++ {
		h.Add(time.Duration(i) * time.Millisecond)
	}
	h.Add(-time.Millisecond)
	h.Add(time.Hour)
	cases := map[float64]time.Duration{50: 51 * time.Millisecond, 99: 100 * time.Millisecond, 100: time.Hour}
	for p, want := range cases {
		if got := h.Percentile(p); got != want {
			t.Errorf("p%v = %s, want %s", p, got, want)
		}
	}
	if h.Count() != 102 || h.Max() != time.Hour {
		t.Fatalf("count %d max %s", h.Count(), h.Max())
	}
	start := time.Now()
	w := NewWindows(start, time.Minute)
	w.At(start.Add(-time.Second)).Add(time.Millisecond)
	w.At(start.Add(90 * time.Second)).Add(2 * time.Millisecond)
	slots := w.Slots()
	if len(slots) != 2 || slots[0].Count() != 1 || slots[1].Count() != 1 {
		t.Fatalf("windows %d", len(slots))
	}
}

func TestTokenRoundTripAndRejections(t *testing.T) {
	token := Token(42, rules.Mode2v2, rules.SeatWest)
	c, err := parseToken(token)
	if err != nil || c.room != 42 || c.mode != rules.Mode2v2 || c.seat != rules.SeatWest {
		t.Fatalf("%s -> %+v, %v", token, c, err)
	}
	for _, bad := range []string{"", "1/2v2", "x/2v2/N", "1/3v3/N", "1/2v2/Q", "1/1v1/N", "1/2v2/X"} {
		if _, err := parseToken(bad); !errors.Is(err, ErrBadToken) {
			t.Errorf("%q accepted: %v", bad, err)
		}
	}
}

func startServer(t *testing.T, target int) (string, *Handler) {
	t.Helper()
	config := app.DefaultConfig()
	config.ListenAddr = "127.0.0.1:0"
	config.Gateway.IntentsPerSecond = 1000
	config.Gateway.IntentBurst = 1000
	config.DrainTimeout = 2 * time.Second
	rulesConfig := rules.DefaultConfig()
	rulesConfig.TargetScore = target
	handler := NewHandler(rulesConfig, bot.DefaultConfig())
	a := app.New(config, app.NewLogger(&bytes.Buffer{}, "error"), handler)
	handler.Bind(a.Rooms())
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- a.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	select {
	case addr := <-a.Addr():
		return "ws://" + addr + "/ws", handler
	case err := <-done:
		t.Fatal(err)
		return "", nil
	}
}

func TestHandlerOpensARoomPerTokenAndResumesASeat(t *testing.T) {
	target, handler := startServer(t, 365)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := botclient.Dial(ctx, target, botclient.Options{Token: "nonsense", AppVersion: "1.0.0"}); !errors.Is(err, botclient.ErrRefused) {
		t.Fatalf("bad token: %v", err)
	}
	x, err := botclient.Dial(ctx, target, botclient.Options{Token: Token(7, rules.Mode1v1, rules.SeatX), AppVersion: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	defer x.Drop()
	y, err := botclient.Dial(ctx, target, botclient.Options{Token: Token(7, rules.Mode1v1, rules.SeatY), AppVersion: "1.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if handler.Opened() != 1 {
		t.Fatalf("opened %d rooms", handler.Opened())
	}
	if _, err := y.View(); err != nil {
		t.Fatal(err)
	}
	y.Drop()
	again, err := botclient.Dial(ctx, target, botclient.Options{Token: Token(7, rules.Mode1v1, rules.SeatY), AppVersion: "1.0.0", LastAcked: y.Acked})
	if err != nil {
		t.Fatal(err)
	}
	defer again.Drop()
	view, err := again.View()
	if err != nil || view.GetSeat().String() != "SEAT_Y" {
		t.Fatalf("resumed seat view %v, %v", view.GetSeat(), err)
	}
	if handler.Opened() != 1 {
		t.Fatalf("a resume opened a room: %d", handler.Opened())
	}
}

func TestRunPlaysMatchesDropsConnectionsAndReports(t *testing.T) {
	if testing.Short() {
		t.Skip("load smoke needs a few seconds")
	}
	target, handler := startServer(t, 30)
	config := DefaultConfig()
	config.Target = target
	config.Rooms = 4
	config.Ramp = 0
	config.Hold = 8 * time.Second
	config.ThinkMin = 0
	config.ThinkMax = 20 * time.Millisecond
	config.ThinkFollow = time.Millisecond
	config.DropsPerMinute = 30
	config.Window = time.Second
	config.Warmup = 2 * time.Second
	config.ReportEvery = time.Second
	config.Budget = time.Second
	heap := uint64(0)
	config.Node = func() (NodeSample, error) {
		heap += 1 << 20
		return NodeSample{HeapBytes: heap, Goroutines: 100}, nil
	}
	logs := &bytes.Buffer{}
	report, err := New(config, app.NewLogger(logs, "debug")).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.ReadFailures+report.DialFailures > 0 {
		t.Log(logs.String()[:min(3000, logs.Len())])
	}
	t.Log("\n" + report.Text())
	if report.Moves == 0 || report.Matches == 0 || report.PeakConnections < 2 {
		t.Fatalf("moves %d matches %d peak %d", report.Moves, report.Matches, report.PeakConnections)
	}
	if report.Drops == 0 || report.Reconnects != report.Drops {
		t.Fatalf("drops %d reconnects %d", report.Drops, report.Reconnects)
	}
	if report.Rejected != 0 || report.DialFailures != 0 || report.ReadFailures != 0 {
		t.Fatalf("rejected %d dial %d read %d", report.Rejected, report.DialFailures, report.ReadFailures)
	}
	if report.P99 == 0 || report.P99 > report.Max || len(report.Windows) < 3 || report.SteadyMoves == 0 {
		t.Fatalf("latency summary %+v", report)
	}
	if report.BytesPerIntent <= 0 || report.BytesPerIntent > 64 {
		t.Fatalf("bytes per intent %.1f", report.BytesPerIntent)
	}
	if report.HeapDriftPct <= 5 || report.Pass || !strings.Contains(strings.Join(report.Failures, ";"), "heap grew") {
		t.Fatalf("a growing heap must fail the run: drift %.1f%% pass %v failures %v", report.HeapDriftPct, report.Pass, report.Failures)
	}
	if handler.Opened() < 4 {
		t.Fatalf("opened %d rooms", handler.Opened())
	}
	if report.GoroutineDriftPct != 0 || strings.Contains(strings.Join(report.Failures, ";"), "goroutines grew") {
		t.Fatalf("flat goroutines failed the run: drift %.1f%% failures %v", report.GoroutineDriftPct, report.Failures)
	}
	if !strings.Contains(logs.String(), `"msg":"load"`) || !strings.Contains(logs.String(), "heap_bytes") || !strings.Contains(logs.String(), "goroutines") {
		t.Fatal("progress log missing")
	}
}

func TestJudgeAppliesTheBudgetAndRates(t *testing.T) {
	rep := &Report{Moves: 10, P99: 50 * time.Millisecond, WorstWindowP99: 50 * time.Millisecond, Budget: 100 * time.Millisecond}
	rep.judge()
	if !rep.Pass {
		t.Fatalf("clean report failed: %v", rep.Failures)
	}
	rep = &Report{Moves: 10, SteadyMoves: 5, SteadyP99: 120 * time.Millisecond, WorstWindowP99: 130 * time.Millisecond, ErrorRate: 0.002, Budget: 100 * time.Millisecond}
	rep.judge()
	if rep.Pass || len(rep.Failures) != 3 {
		t.Fatalf("failures %v", rep.Failures)
	}
	rep = &Report{Budget: 100 * time.Millisecond}
	rep.judge()
	if rep.Pass || rep.Failures[0] != "no move was measured" {
		t.Fatalf("failures %v", rep.Failures)
	}
	samples := []NodeSample{{0, 100, 10}, {time.Minute, 200, 1000}, {2 * time.Minute, 210, 1100}}
	if got := drift(samples, time.Minute, heapOf); got != 5 {
		t.Fatalf("heap drift %.1f", got)
	}
	if got := drift(samples, time.Minute, goroutinesOf); got != 10 {
		t.Fatalf("goroutine drift %.1f", got)
	}
	if got := drift(samples[:1], 0, heapOf); got != 0 {
		t.Fatalf("one sample drift %.1f", got)
	}
	leaking := &Report{Moves: 10, P99: time.Millisecond, Budget: 100 * time.Millisecond, GoroutineDriftPct: 6}
	leaking.judge()
	if leaking.Pass || leaking.Failures[0] != "goroutines grew 6.0% during the steady state" {
		t.Fatalf("goroutine leak failures %v", leaking.Failures)
	}
	text := rep.Text()
	if !strings.Contains(text, "FAIL: no move was measured") {
		t.Fatal(text)
	}
}

func TestRunRequiresRoomsAndATarget(t *testing.T) {
	if _, err := New(Config{}, nil).Run(context.Background()); err == nil {
		t.Fatal("empty config accepted")
	}
}

func TestStormDropsEveryLiveConnectionOnceAndTimesTheReturn(t *testing.T) {
	if testing.Short() {
		t.Skip("load smoke needs a few seconds")
	}
	target, _ := startServer(t, 30)
	config := DefaultConfig()
	config.Target = target
	config.Rooms = 6
	config.TwoVersusTwoPct = 0
	config.Ramp = 0
	config.Hold = 4 * time.Second
	config.ThinkMin = 50 * time.Millisecond
	config.ThinkMax = 100 * time.Millisecond
	config.ThinkFollow = time.Millisecond
	config.DropsPerMinute = 0
	config.StormAt = time.Second
	config.Window = time.Second
	config.Warmup = 0
	config.Budget = time.Second
	report, err := New(config, app.NewLogger(&bytes.Buffer{}, "info")).Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Log("\n" + report.Text())
	if report.StormDrops < 6 || report.StormDrops > 12 || report.Drops != report.StormDrops || report.Reconnects != report.StormDrops {
		t.Fatalf("storm dropped %d, drops %d, reconnects %d; want every seat in a match once, a seat between matches has no connection", report.StormDrops, report.Drops, report.Reconnects)
	}
	if report.StormRecovered != report.StormDrops || report.StormRecovery <= 0 || report.StormRecovery > 2*time.Second {
		t.Fatalf("recovered %d of %d, slowest %s", report.StormRecovered, report.StormDrops, report.StormRecovery)
	}
	if report.DialFailures+report.ReadFailures+report.Rejected != 0 || !strings.Contains(report.Text(), fmt.Sprintf("storm at 1s: dropped %d", report.StormDrops)) {
		t.Fatalf("failures after the storm: %+v", report)
	}
}
