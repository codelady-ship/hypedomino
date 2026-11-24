package panics

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
)

const euIngestHostSuffix = ".ingest.de.sentry.io"

const shipFlushTimeout = 2 * time.Second

var (
	ErrDSNMalformed = errors.New("not a Sentry DSN")
	ErrDSNOutsideEU = errors.New("not a Sentry EU data region DSN, the host must end in " + euIngestHostSuffix)
)

var counts = struct {
	sync.Mutex
	by map[string]uint64
}{by: map[string]uint64{}}

func Caught(where string, value any, attrs ...any) bool {
	if value == nil {
		return false
	}
	counts.Lock()
	counts.by[where]++
	counts.Unlock()
	slog.Error("panic recovered", append([]any{"where", where, "panic", fmt.Sprint(value), "stack", string(debug.Stack())}, attrs...)...)
	ship(where, value)
	return true
}

func ship(where string, value any) {
	hub := sentry.CurrentHub().Clone()
	hub.Scope().SetTag("where", where)
	hub.Recover(value)
}

func CheckDSN(raw string) error {
	dsn, err := sentry.NewDsn(raw)
	if err != nil {
		return ErrDSNMalformed
	}
	if !strings.HasSuffix(dsn.GetHost(), euIngestHostSuffix) {
		return ErrDSNOutsideEU
	}
	return nil
}

func Ship(dsn, environment string, transport sentry.Transport) (func(), error) {
	if dsn == "" {
		return func() {}, nil
	}
	if err := CheckDSN(dsn); err != nil {
		return nil, err
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      environment,
		Transport:        transport,
		SendDefaultPII:   false,
		AttachStacktrace: true,
	})
	if err != nil {
		return nil, err
	}
	return func() { sentry.Flush(shipFlushTimeout) }, nil
}

func Supervise(ctx context.Context, where string, every time.Duration, run func(context.Context, time.Duration)) {
	for !returned(ctx, where, every, run) {
		select {
		case <-ctx.Done():
			return
		case <-time.After(every):
		}
	}
}

func returned(ctx context.Context, where string, every time.Duration, run func(context.Context, time.Duration)) bool {
	defer func() { Caught(where, recover()) }()
	run(ctx, every)
	return true
}

func WriteTo(w io.Writer) error {
	counts.Lock()
	defer counts.Unlock()
	if _, err := fmt.Fprint(w, "# HELP hd_panics_total Panics recovered in long-lived goroutines by site.\n# TYPE hd_panics_total counter\n"); err != nil {
		return err
	}
	sites := make([]string, 0, len(counts.by))
	for where := range counts.by {
		sites = append(sites, where)
	}
	slices.Sort(sites)
	for _, where := range sites {
		if _, err := fmt.Fprintf(w, "hd_panics_total{where=%q} %d\n", where, counts.by[where]); err != nil {
			return err
		}
	}
	return nil
}
