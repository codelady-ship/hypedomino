package panics

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
)

const euTestDSN = "https://public@o1.ingest.de.sentry.io/1"

func shipTo(t *testing.T) *sentry.MockTransport {
	t.Helper()
	transport := &sentry.MockTransport{}
	flush, err := Ship(euTestDSN, "test", transport)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		flush()
		sentry.CurrentHub().BindClient(nil)
	})
	return transport
}

func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &logs
}

func exposed(t *testing.T) string {
	t.Helper()
	var out strings.Builder
	if err := WriteTo(&out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestSuperviseRestartsAPanickingJobAndCountsIt(t *testing.T) {
	logs := captureLogs(t)
	ctx, cancel := context.WithCancel(t.Context())
	runs := 0
	Supervise(ctx, "test job", time.Millisecond, func(ctx context.Context, every time.Duration) {
		runs++
		if runs < 3 {
			panic("job broke")
		}
		cancel()
	})
	if runs != 3 {
		t.Fatalf("runs %d, want 3", runs)
	}
	if !strings.Contains(exposed(t), `hd_panics_total{where="test job"} 2`) {
		t.Fatalf("metrics:\n%s", exposed(t))
	}
	for _, field := range []string{`"level":"ERROR"`, `"where":"test job"`, `"panic":"job broke"`, `"stack":"goroutine `} {
		if !strings.Contains(logs.String(), field) {
			t.Fatalf("missing %s in %s", field, logs.String())
		}
	}
}

func TestSuperviseStopsWhenTheJobReturns(t *testing.T) {
	runs := 0
	Supervise(t.Context(), "returning job", time.Millisecond, func(context.Context, time.Duration) { runs++ })
	if runs != 1 || strings.Contains(exposed(t), "returning job") {
		t.Fatalf("runs %d metrics:\n%s", runs, exposed(t))
	}
}

func TestCaughtIgnoresNoPanic(t *testing.T) {
	if Caught("calm", nil) || strings.Contains(exposed(t), "calm") {
		t.Fatal("counted a nil recover")
	}
}

func TestCaughtShipsThePanicWithItsSiteAndStack(t *testing.T) {
	captureLogs(t)
	transport := shipTo(t)
	func() {
		defer func() { Caught("shipped job", recover(), "match", "m-1") }()
		panic("shipped broke")
	}()
	func() {
		defer func() { Caught("shipped job", recover()) }()
		var broken map[string]int
		broken["write"] = 1
	}()
	events := transport.Events()
	if len(events) != 2 {
		t.Fatalf("events %d, want 2", len(events))
	}
	for _, event := range events {
		if event.Tags["where"] != "shipped job" || event.Environment != "test" || event.Level != sentry.LevelFatal {
			t.Fatalf("tags %v environment %q level %q", event.Tags, event.Environment, event.Level)
		}
		if _, leaked := event.Tags["match"]; leaked || event.User.ID != "" || event.User.IPAddress != "" {
			t.Fatalf("tags %v user %+v", event.Tags, event.User)
		}
	}
	if events[0].Message != "shipped broke" || len(events[0].Threads) != 1 || events[0].Threads[0].Stacktrace == nil {
		t.Fatalf("message %q threads %+v", events[0].Message, events[0].Threads)
	}
	exception := events[1].Exception
	if len(exception) == 0 || !strings.Contains(exception[len(exception)-1].Value, "nil map") || exception[len(exception)-1].Stacktrace == nil {
		t.Fatalf("exception %+v", exception)
	}
}

func TestShipWithoutDSNSendsNothing(t *testing.T) {
	captureLogs(t)
	flush, err := Ship("", "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	flush()
	if sentry.CurrentHub().Client().IsEnabled() {
		t.Fatal("bound a client without a DSN")
	}
	if !Caught("unshipped", "quiet") {
		t.Fatal("did not catch")
	}
}

func TestCheckDSNAcceptsOnlyTheEURegion(t *testing.T) {
	for _, tc := range []struct {
		dsn  string
		want error
	}{
		{euTestDSN, nil},
		{"https://public@o1.ingest.sentry.io/1", ErrDSNOutsideEU},
		{"https://public@o1.ingest.us.sentry.io/1", ErrDSNOutsideEU},
		{"https://public@sentry.example.com/1", ErrDSNOutsideEU},
		{"https://o1.ingest.de.sentry.io/1", ErrDSNMalformed},
		{"secret-looking not a dsn %zz", ErrDSNMalformed},
	} {
		if err := CheckDSN(tc.dsn); !errors.Is(err, tc.want) {
			t.Fatalf("%s: %v, want %v", tc.dsn, err, tc.want)
		}
	}
	if _, err := Ship("https://public@o1.ingest.us.sentry.io/1", "test", &sentry.MockTransport{}); !errors.Is(err, ErrDSNOutsideEU) {
		t.Fatalf("shipped outside the EU: %v", err)
	}
}
