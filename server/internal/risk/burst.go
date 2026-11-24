package risk

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type burstRule struct {
	kind      string
	countName string
	threshold int
	window    time.Duration
	weight    int
}

type Burst struct {
	store   *Store
	rule    burstRule
	clock   Clock
	logger  *slog.Logger
	mu      sync.Mutex
	recent  map[string][]time.Time
	sweepAt time.Time
}

func NewReplays(store *Store, config Config, clock Clock, logger *slog.Logger) *Burst {
	return newBurst(store, burstRule{kind: KindReplay, countName: "rejections", threshold: config.ReplayThreshold, window: config.ReplayWindow, weight: config.ReplayWeight}, clock, logger)
}

func NewMacFailures(store *Store, config Config, clock Clock, logger *slog.Logger) *Burst {
	return newBurst(store, burstRule{kind: KindMacFailure, countName: "failures", threshold: config.MacFailureThreshold, window: config.MacFailureWindow, weight: config.MacFailureWeight}, clock, logger)
}

func NewUnownedEquips(store *Store, config Config, clock Clock, logger *slog.Logger) *Burst {
	return newBurst(store, burstRule{kind: KindUnownedEquip, countName: "attempts", threshold: config.UnownedEquipThreshold, window: config.UnownedEquipWindow, weight: config.UnownedEquipWeight}, clock, logger)
}

func newBurst(store *Store, rule burstRule, clock Clock, logger *slog.Logger) *Burst {
	return &Burst{store: store, rule: rule, clock: clock, logger: logger, recent: map[string][]time.Time{}}
}

func (b *Burst) Observe(ctx context.Context, account string) {
	now := b.clock.Now().UTC()
	count, reached := b.count(account, now)
	if !reached {
		return
	}
	signal := Signal{
		Account: account, Kind: b.rule.kind, Weight: b.rule.weight, Dedupe: now.Truncate(b.rule.window).Format(time.RFC3339), At: now,
		Evidence: map[string]any{b.rule.countName: count, "window_seconds": int64(b.rule.window.Seconds())},
	}
	if _, err := b.store.Record(ctx, signal); err != nil {
		b.logger.Error("risk signal not recorded", "kind", b.rule.kind, "account", account, "error", err)
	}
}

func (b *Burst) count(account string, now time.Time) (int, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	since := now.Add(-b.rule.window)
	b.sweep(now, since)
	kept := within(b.recent[account], since)
	kept = append(kept, now)
	if len(kept) < b.rule.threshold {
		b.recent[account] = kept
		return len(kept), false
	}
	delete(b.recent, account)
	return len(kept), true
}

func (b *Burst) sweep(now, since time.Time) {
	if now.Before(b.sweepAt) {
		return
	}
	b.sweepAt = now.Add(b.rule.window)
	for account, times := range b.recent {
		if kept := within(times, since); len(kept) > 0 {
			b.recent[account] = kept
		} else {
			delete(b.recent, account)
		}
	}
}

func within(times []time.Time, since time.Time) []time.Time {
	for i, at := range times {
		if at.After(since) {
			return times[i:]
		}
	}
	return times[:0]
}

func (b *Burst) Tracked() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.recent)
}
