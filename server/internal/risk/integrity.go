package risk

import (
	"context"
	"log/slog"
	"time"
)

type Integrity struct {
	store  *Store
	clock  Clock
	logger *slog.Logger
}

func NewIntegrity(store *Store, clock Clock, logger *slog.Logger) *Integrity {
	return &Integrity{store: store, clock: clock, logger: logger}
}

func (i *Integrity) RecordIntegrity(ctx context.Context, account, purpose, kind string, weight int, reasons []string) {
	now := i.clock.Now().UTC()
	signal := Signal{
		Account: account, Kind: kind, Weight: weight, Dedupe: purpose + ":" + now.Format(time.DateOnly), At: now,
		Evidence: map[string]any{"purpose": purpose, "reasons": reasons},
	}
	if _, err := i.store.Record(ctx, signal); err != nil {
		i.logger.Error("integrity signal not recorded", "account", account, "kind", kind, "error", err)
	}
}
