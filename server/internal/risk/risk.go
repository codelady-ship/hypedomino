package risk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	KindClockSkew    = "clock_skew"
	KindFakeReceipt  = "fake_receipt"
	KindRefund       = "refund"
	KindReplay       = "replay_reject"
	KindMacFailure   = "mac_failure"
	KindUnownedEquip = "unowned_equip"
	KindCollusion    = "collusion"
	KindFarm         = "farm"
)

var ErrInvalidConfig = errors.New("risk: invalid config")

type Config struct {
	ClockSkewTolerance    time.Duration
	ClockSkewWeight       int
	FakeReceiptWeight     int
	RefundWeight          int
	ReplayThreshold       int
	ReplayWindow          time.Duration
	ReplayWeight          int
	MacFailureThreshold   int
	MacFailureWindow      time.Duration
	MacFailureWeight      int
	UnownedEquipThreshold int
	UnownedEquipWindow    time.Duration
	UnownedEquipWeight    int
	CollusionWeight       int
	Decay                 time.Duration
	ReviewScore           int
	ShadowbanScore        int
	BanWaveInterval       time.Duration
}

func DefaultConfig() Config {
	return Config{ClockSkewTolerance: 5 * time.Minute, ClockSkewWeight: 1, FakeReceiptWeight: 20, RefundWeight: 3, ReplayThreshold: 5, ReplayWindow: 10 * time.Minute, ReplayWeight: 5, MacFailureThreshold: 5, MacFailureWindow: 10 * time.Minute, MacFailureWeight: 5, UnownedEquipThreshold: 5, UnownedEquipWindow: 10 * time.Minute, UnownedEquipWeight: 5, CollusionWeight: 10, Decay: 30 * 24 * time.Hour, ReviewScore: 20, ShadowbanScore: 40, BanWaveInterval: 24 * time.Hour}
}

func (c Config) Validate() error {
	if c.ClockSkewTolerance <= 0 || c.ClockSkewWeight < 0 {
		return fmt.Errorf("%w: clock skew tolerance must be positive and its weight not negative", ErrInvalidConfig)
	}
	if c.ReplayThreshold < 1 || c.ReplayWindow <= 0 || c.ReplayWeight < 0 {
		return fmt.Errorf("%w: replay threshold must be at least 1, its window positive and its weight not negative", ErrInvalidConfig)
	}
	if c.MacFailureThreshold < 1 || c.MacFailureWindow <= 0 || c.MacFailureWeight < 0 {
		return fmt.Errorf("%w: mac failure threshold must be at least 1, its window positive and its weight not negative", ErrInvalidConfig)
	}
	if c.UnownedEquipThreshold < 1 || c.UnownedEquipWindow <= 0 || c.UnownedEquipWeight < 0 {
		return fmt.Errorf("%w: unowned equip threshold must be at least 1, its window positive and its weight not negative", ErrInvalidConfig)
	}
	if c.Decay <= 0 || c.BanWaveInterval <= 0 {
		return fmt.Errorf("%w: decay and ban wave interval must be positive", ErrInvalidConfig)
	}
	if c.ReviewScore < 1 || c.ShadowbanScore < c.ReviewScore {
		return fmt.Errorf("%w: review score must be at least 1 and the shadowban score not below it", ErrInvalidConfig)
	}
	if c.FakeReceiptWeight < 0 || c.RefundWeight < 0 || c.CollusionWeight < 0 {
		return fmt.Errorf("%w: fake receipt, refund and collusion weights must not be negative", ErrInvalidConfig)
	}
	return nil
}

type Signal struct {
	Account  string
	Kind     string
	Weight   int
	Dedupe   string
	Evidence map[string]any
	At       time.Time
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

type Executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

func (s *Store) Record(ctx context.Context, signal Signal) (bool, error) {
	return RecordIn(ctx, s.pool, signal)
}

func RecordIn(ctx context.Context, db Executor, signal Signal) (bool, error) {
	evidence, err := json.Marshal(signal.Evidence)
	if err != nil {
		return false, err
	}
	tag, err := db.Exec(ctx, `insert into risk_signals (account_id, kind, weight, dedupe_key, evidence, created_at) values ($1, $2, $3, $4, $5, $6)
		on conflict (account_id, kind, dedupe_key) do nothing`, signal.Account, signal.Kind, signal.Weight, signal.Dedupe, evidence, signal.At)
	if err != nil || tag.RowsAffected() == 0 {
		return false, err
	}
	if isHard(signal.Kind) {
		return true, openReview(ctx, db, signal.Account, CauseHardSignal, signal.At)
	}
	return true, nil
}

func (s *Store) Signals(ctx context.Context, account string) ([]Signal, error) {
	rows, err := s.pool.Query(ctx, `select account_id, kind, weight, dedupe_key, evidence, created_at from risk_signals where account_id = $1 order by created_at, id`, account)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Signal, error) {
		var signal Signal
		var evidence []byte
		if err := row.Scan(&signal.Account, &signal.Kind, &signal.Weight, &signal.Dedupe, &evidence, &signal.At); err != nil {
			return signal, err
		}
		return signal, json.Unmarshal(evidence, &signal.Evidence)
	})
}

type Clock interface {
	Now() time.Time
}

type ClockSkew struct {
	store  *Store
	config Config
	clock  Clock
	logger *slog.Logger
}

func NewClockSkew(store *Store, config Config, clock Clock, logger *slog.Logger) *ClockSkew {
	return &ClockSkew{store: store, config: config, clock: clock, logger: logger}
}

func (c *ClockSkew) Observe(ctx context.Context, account string, clientUnixMs uint64) {
	if clientUnixMs == 0 {
		return
	}
	now := c.clock.Now().UTC()
	client := time.UnixMilli(int64(clientUnixMs)).UTC()
	skew := client.Sub(now)
	if skew.Abs() <= c.config.ClockSkewTolerance {
		return
	}
	signal := Signal{
		Account: account, Kind: KindClockSkew, Weight: c.config.ClockSkewWeight, Dedupe: now.Format(time.DateOnly), At: now,
		Evidence: map[string]any{"client_time": client.Format(time.RFC3339Nano), "server_time": now.Format(time.RFC3339Nano), "skew_seconds": int64(skew.Seconds())},
	}
	if _, err := c.store.Record(ctx, signal); err != nil {
		c.logger.Error("clock skew signal not recorded", "account", account, "error", err)
	}
}
