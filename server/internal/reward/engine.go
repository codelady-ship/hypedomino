package reward

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const Topic = "grants"

var (
	ErrInvalidGrant  = errors.New("reward: invalid grant")
	ErrUnknownGrant  = errors.New("reward: unknown grant")
	ErrAlreadyRevoke = errors.New("reward: grant already revoked")
)

type Config struct {
	DuplicateCompensationPercent int
	GuidePriceCoins              map[string]int64
	GuidePriceTypePercent        map[string]int
}

func DefaultConfig() Config {
	return Config{
		DuplicateCompensationPercent: 20,
		GuidePriceCoins:              map[string]int64{"common": 400, "rare": 1500, "epic": 3500, "legendary": 12000, "mythic": 30000},
		GuidePriceTypePercent:        map[string]int{cosmetics.TypeAvatar: 100, cosmetics.TypeCharacter: 150, cosmetics.TypeFrame: 100, cosmetics.TypeTable: 150, cosmetics.TypeEmoji: 50, cosmetics.TypeAction: 50},
	}
}

func (c Config) GuidePrice(item cosmetics.Item) int64 {
	return c.GuidePriceCoins[item.Rarity] * int64(c.GuidePriceTypePercent[item.Type]) / 100
}

func (c Config) Compensation(item cosmetics.Item) int64 {
	return c.GuidePrice(item) * int64(c.DuplicateCompensationPercent) / 100
}

type Medals interface {
	GrantIn(ctx context.Context, tx pgx.Tx, account, medal string, source cosmetics.Source, now time.Time) (bool, error)
	RevokeIn(ctx context.Context, tx pgx.Tx, account, medal string, now time.Time) error
}

type Pass interface {
	AddIn(ctx context.Context, tx pgx.Tx, account string, xp int, now time.Time) (bool, error)
}

type Grant struct {
	Account       string
	Bundle        Bundle
	Source        cosmetics.Source
	Reason        wallet.Reason
	Key           string
	ConfigVersion int
}

type Report struct {
	ID            int64            `json:"id"`
	Account       string           `json:"account_id"`
	BundleID      string           `json:"bundle_id"`
	Revision      int              `json:"revision"`
	Source        cosmetics.Source `json:"source"`
	Reason        wallet.Reason    `json:"reason"`
	Key           string           `json:"key"`
	ConfigVersion int              `json:"config_version"`
	Coins         int64            `json:"coins"`
	Cosmetics     []string         `json:"cosmetics"`
	Duplicates    map[string]int64 `json:"duplicates"`
	Medals        []string         `json:"medals"`
	MedalsSkipped []string         `json:"medals_skipped"`
	PassXP        int              `json:"pass_xp"`
	PassXPDropped int              `json:"pass_xp_dropped"`
	At            time.Time        `json:"at"`
	Repeated      bool             `json:"-"`
	RevokedAt     time.Time        `json:"-"`
	RevokeKey     string           `json:"-"`
}

func (r Report) Revoked() bool {
	return !r.RevokedAt.IsZero()
}

type Engine struct {
	pool      *pgxpool.Pool
	wallet    *wallet.Store
	inventory *inventory.Store
	medals    Medals
	pass      Pass
	config    Config
}

func NewEngine(pool *pgxpool.Pool, wallet *wallet.Store, inventory *inventory.Store, medals Medals, pass Pass, config Config) *Engine {
	return &Engine{pool: pool, wallet: wallet, inventory: inventory, medals: medals, pass: pass, config: config}
}

func (g Grant) validate() error {
	if err := g.Bundle.Validate(); err != nil {
		return err
	}
	if g.Account == "" || g.Key == "" || g.Source.Type == "" || g.Source.Type == cosmetics.SourceDefault {
		return fmt.Errorf("%w: account, key and a non-default source are required", ErrInvalidGrant)
	}
	probe := wallet.Entry{Account: g.Account, Amount: 1, Reason: g.Reason, Source: wallet.Source{Type: g.Source.Type, Ref: g.Source.Ref}, Key: g.Key}
	if err := probe.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidGrant, err)
	}
	return nil
}

func (e *Engine) Apply(ctx context.Context, grant Grant, now time.Time) (Report, error) {
	if err := grant.validate(); err != nil {
		return Report{}, err
	}
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return Report{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtext($1))`, grant.Account); err != nil {
		return Report{}, err
	}
	earlier, err := load(ctx, tx, grant.Account, grant.Key)
	if err == nil {
		earlier.Repeated = true
		return earlier, tx.Commit(ctx)
	}
	if !errors.Is(err, ErrUnknownGrant) {
		return Report{}, err
	}
	report := Report{Account: grant.Account, BundleID: grant.Bundle.ID, Revision: grant.Bundle.Revision, Source: grant.Source, Reason: grant.Reason, Key: grant.Key, ConfigVersion: grant.ConfigVersion, Cosmetics: []string{}, Duplicates: map[string]int64{}, Medals: []string{}, MedalsSkipped: []string{}, At: now}
	source := wallet.Source{Type: grant.Source.Type, Ref: grant.Source.Ref}
	if grant.Bundle.Coins > 0 {
		if _, err := e.wallet.ApplyIn(ctx, tx, wallet.Entry{Account: grant.Account, Amount: grant.Bundle.Coins, Reason: grant.Reason, Source: source, Key: grant.Key, ConfigVersion: grant.ConfigVersion}, now); err != nil {
			return Report{}, err
		}
		report.Coins = grant.Bundle.Coins
	}
	for _, id := range grant.Bundle.Cosmetics {
		granted, err := e.inventory.GrantIn(ctx, tx, grant.Account, id, grant.Source, now)
		if err != nil {
			return Report{}, err
		}
		if granted {
			report.Cosmetics = append(report.Cosmetics, id)
			continue
		}
		item, _ := e.inventory.Catalog().Item(id)
		compensation := e.config.Compensation(item)
		report.Duplicates[id] = compensation
		if compensation == 0 {
			continue
		}
		if _, err := e.wallet.ApplyIn(ctx, tx, wallet.Entry{Account: grant.Account, Amount: compensation, Reason: wallet.DuplicateCompensation, Source: source, Key: grant.Key + ":dup:" + id, ConfigVersion: grant.ConfigVersion}, now); err != nil {
			return Report{}, err
		}
	}
	for _, id := range grant.Bundle.Medals {
		granted := false
		if e.medals != nil {
			if granted, err = e.medals.GrantIn(ctx, tx, grant.Account, id, grant.Source, now); err != nil {
				return Report{}, err
			}
		}
		if granted {
			report.Medals = append(report.Medals, id)
		} else {
			report.MedalsSkipped = append(report.MedalsSkipped, id)
		}
	}
	if grant.Bundle.PassXP > 0 {
		applied := false
		if e.pass != nil {
			if applied, err = e.pass.AddIn(ctx, tx, grant.Account, grant.Bundle.PassXP, now); err != nil {
				return Report{}, err
			}
		}
		if applied {
			report.PassXP = grant.Bundle.PassXP
		} else {
			report.PassXPDropped = grant.Bundle.PassXP
		}
	}
	if err := e.record(ctx, tx, &report); err != nil {
		return Report{}, err
	}
	return report, tx.Commit(ctx)
}

func (e *Engine) record(ctx context.Context, tx pgx.Tx, report *Report) error {
	encoded, err := json.Marshal(report)
	if err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `insert into reward_grants (account_id, bundle_id, revision, source_type, source_ref, idempotency_key, config_version, report, created_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9) returning id`,
		report.Account, report.BundleID, report.Revision, report.Source.Type, report.Source.Ref, report.Key, report.ConfigVersion, encoded, report.At).Scan(&report.ID); err != nil {
		return err
	}
	encoded, err = json.Marshal(report)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `update reward_grants set report = $2 where id = $1`, report.ID, encoded); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into outbox (topic, key, payload) values ($1, $2, $3) on conflict (topic, key) do nothing`, Topic, report.Account+":"+report.Key, encoded)
	return err
}

func load(ctx context.Context, tx pgx.Tx, account, key string) (Report, error) {
	var encoded []byte
	var revokedAt *time.Time
	var revokeKey *string
	err := tx.QueryRow(ctx, `select report, revoked_at, revoke_key from reward_grants where account_id = $1 and idempotency_key = $2 for update`, account, key).Scan(&encoded, &revokedAt, &revokeKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return Report{}, ErrUnknownGrant
	}
	if err != nil {
		return Report{}, err
	}
	var report Report
	if err := json.Unmarshal(encoded, &report); err != nil {
		return Report{}, err
	}
	report.At = report.At.UTC()
	if revokedAt != nil {
		report.RevokedAt = revokedAt.UTC()
	}
	if revokeKey != nil {
		report.RevokeKey = *revokeKey
	}
	return report, nil
}

func (e *Engine) Get(ctx context.Context, account, key string) (Report, error) {
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return Report{}, err
	}
	defer tx.Rollback(ctx)
	return load(ctx, tx, account, key)
}

func (e *Engine) Revoke(ctx context.Context, account, key string, reason wallet.Reason, revokeKey string, now time.Time) (Report, error) {
	tx, err := e.pool.Begin(ctx)
	if err != nil {
		return Report{}, err
	}
	defer tx.Rollback(ctx)
	report, err := load(ctx, tx, account, key)
	if err != nil {
		return Report{}, err
	}
	if report.Revoked() {
		return report, ErrAlreadyRevoke
	}
	coins := report.Coins
	for _, compensation := range report.Duplicates {
		coins += compensation
	}
	if coins > 0 {
		if _, err := e.wallet.ApplyIn(ctx, tx, wallet.Entry{Account: account, Amount: -coins, Reason: reason, Source: wallet.Source{Type: report.Source.Type, Ref: report.Source.Ref}, Key: revokeKey, ConfigVersion: report.ConfigVersion}, now); err != nil {
			return Report{}, err
		}
	}
	for _, id := range report.Cosmetics {
		if err := e.inventory.RevokeIn(ctx, tx, account, id, now); err != nil && !errors.Is(err, inventory.ErrNotOwned) {
			return Report{}, err
		}
	}
	if e.medals != nil {
		for _, id := range report.Medals {
			if err := e.medals.RevokeIn(ctx, tx, account, id, now); err != nil {
				return Report{}, err
			}
		}
	}
	if _, err := tx.Exec(ctx, `update reward_grants set revoked_at = $2, revoke_key = $3 where id = $1`, report.ID, now, revokeKey); err != nil {
		return Report{}, err
	}
	report.RevokedAt, report.RevokeKey = now, revokeKey
	return report, tx.Commit(ctx)
}
