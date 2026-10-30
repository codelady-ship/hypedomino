package purchase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const (
	Topic             = "purchases"
	SourceFirst       = "first_purchase"
	RecordPending     = "pending"
	RecordGranted     = "granted"
	RecordRefunded    = "refunded"
	RecordCancelled   = "cancelled"
	EventGranted      = "granted"
	EventRefunded     = "refunded"
	EventReversed     = "reversed"
	EventAccountMatch = "account_mismatch"
	EventReuse        = "reuse"
)

var (
	ErrPlatform  = errors.New("purchase.platform_unavailable")
	ErrProduct   = errors.New("receipt.product")
	ErrAccount   = errors.New("receipt.account")
	ErrReused    = errors.New("receipt.reused")
	ErrCancelled = errors.New("receipt.cancelled")
	ErrBlocked   = errors.New("purchase.blocked")
)

type Config struct {
	RefundsBeforePurchaseBlock int
	RefundWindowDays           int
}

func DefaultConfig() Config {
	return Config{RefundsBeforePurchaseBlock: 3, RefundWindowDays: 365}
}

func (c Config) Validate() error {
	if c.RefundsBeforePurchaseBlock < 1 || c.RefundWindowDays < 1 {
		return errors.New("purchase: refunds before block and refund window must be at least 1")
	}
	return nil
}

type Store struct {
	pool       *pgxpool.Pool
	wallet     *wallet.Store
	engine     *reward.Engine
	offers     *snapshot.Watcher[*shop.Catalog]
	validators map[Platform]Validator
	config     Config
	logger     *slog.Logger
}

func NewStore(pool *pgxpool.Pool, wallet *wallet.Store, engine *reward.Engine, offers *snapshot.Watcher[*shop.Catalog], validators map[Platform]Validator, config Config, logger *slog.Logger) *Store {
	return &Store{pool: pool, wallet: wallet, engine: engine, offers: offers, validators: validators, config: config, logger: logger}
}

type Result struct {
	Platform      Platform       `json:"platform"`
	TransactionID string         `json:"transaction_id"`
	Offer         string         `json:"offer"`
	State         string         `json:"state"`
	Repeated      bool           `json:"repeated"`
	Balance       int64          `json:"balance"`
	Grant         *reward.Report `json:"grant,omitempty"`
	Bonus         *reward.Report `json:"bonus,omitempty"`
	Finished      bool           `json:"finished"`
}

type record struct {
	ID            int64
	Platform      Platform
	TransactionID string
	Account       string
	Offer         string
	Product       string
	State         string
	GrantKey      string
	BonusKey      string
}

const recordColumns = `id, platform, transaction_id, account_id, offer_id, product_id, state, grant_key, bonus_key`

func scanRecord(row pgx.Row) (record, error) {
	var r record
	err := row.Scan(&r.ID, &r.Platform, &r.TransactionID, &r.Account, &r.Offer, &r.Product, &r.State, &r.GrantKey, &r.BonusKey)
	return r, err
}

func (s *Store) Complete(ctx context.Context, account, offerID string, platform Platform, token string, now time.Time) (Result, error) {
	validator, ok := s.validators[platform]
	if !ok {
		return Result{}, ErrPlatform
	}
	catalog, version, offer, err := s.offer(offerID)
	if err != nil {
		return Result{}, err
	}
	blocked, err := s.Blocked(ctx, account, now)
	if err != nil {
		return Result{}, err
	}
	if blocked {
		return Result{}, ErrBlocked
	}
	receipt, err := validator.Verify(ctx, offer.ProductID, token)
	if err != nil {
		return Result{}, err
	}
	switch {
	case receipt.ProductID != offer.ProductID:
		return Result{}, fmt.Errorf("%w: %s is not sold by %s", ErrProduct, receipt.ProductID, offer.ID)
	case receipt.AccountToken != shop.AccountToken(account):
		s.evidence(ctx, account, receipt, EventAccountMatch)
		return Result{}, ErrAccount
	case receipt.State == StateCancelled:
		return Result{}, ErrCancelled
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback(ctx)
	if err := reward.LockIn(ctx, tx, account); err != nil {
		return Result{}, err
	}
	existing, err := s.lookup(ctx, tx, platform, receipt.TransactionID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Result{}, err
	}
	found := err == nil
	if found && existing.Account != account {
		s.evidence(ctx, account, receipt, EventReuse)
		return Result{}, ErrReused
	}
	if found && existing.State != RecordPending {
		result, err := s.load(ctx, tx, existing)
		if err != nil {
			return Result{}, err
		}
		result.Finished = s.finish(ctx, validator, receipt)
		return result, tx.Commit(ctx)
	}
	if !found {
		existing, err = s.insert(ctx, tx, account, offer, receipt, now)
		if err != nil {
			return Result{}, err
		}
	}
	if receipt.State == StatePending {
		return Result{Platform: platform, TransactionID: receipt.TransactionID, Offer: offer.ID, State: RecordPending}, tx.Commit(ctx)
	}
	result, err := s.grant(ctx, tx, existing, catalog, version, offer, now)
	if err != nil {
		return Result{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Result{}, err
	}
	result.Finished = s.finish(ctx, validator, receipt)
	return result, nil
}

func (s *Store) offer(offerID string) (*shop.Catalog, int64, shop.Offer, error) {
	catalog, version, ok := s.offers.Current()
	if !ok {
		return nil, 0, shop.Offer{}, shop.ErrNoCatalog
	}
	offer, ok := catalog.Offer(offerID)
	switch {
	case !ok:
		return nil, 0, shop.Offer{}, shop.ErrUnknownOffer
	case offer.ProductID == "":
		return nil, 0, shop.Offer{}, shop.ErrCoinsOnly
	case offer.Kind == shop.KindPass || offer.Kind == shop.KindPassLevels:
		return nil, 0, shop.Offer{}, shop.ErrUnsupported
	}
	return catalog, version, offer, nil
}

func (s *Store) lookup(ctx context.Context, tx pgx.Tx, platform Platform, transaction string) (record, error) {
	return scanRecord(tx.QueryRow(ctx, `select `+recordColumns+` from purchases where platform = $1 and transaction_id = $2 for update`, platform, transaction))
}

func (s *Store) insert(ctx context.Context, tx pgx.Tx, account string, offer shop.Offer, receipt Receipt, now time.Time) (record, error) {
	return scanRecord(tx.QueryRow(ctx, `insert into purchases (platform, transaction_id, order_id, account_id, offer_id, product_id, environment, state, purchased_at, created_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) returning `+recordColumns,
		receipt.Platform, receipt.TransactionID, receipt.OrderID, account, offer.ID, offer.ProductID, receipt.Environment, RecordPending, receipt.PurchasedAt, now))
}

func (s *Store) grant(ctx context.Context, tx pgx.Tx, r record, catalog *shop.Catalog, version int64, offer shop.Offer, now time.Time) (Result, error) {
	key := "purchase:" + string(r.Platform) + ":" + r.TransactionID
	source := cosmetics.Source{Type: shop.SourceType, Ref: offer.ID}
	report, err := s.engine.ApplyIn(ctx, tx, reward.Grant{Account: r.Account, Bundle: offer.Bundle, Source: source, Reason: wallet.Purchase, Key: key, ConfigVersion: int(version)}, now)
	if err != nil {
		return Result{}, err
	}
	result := Result{Platform: r.Platform, TransactionID: r.TransactionID, Offer: offer.ID, State: RecordGranted, Grant: &report}
	bonusKey := ""
	if first, err := s.first(ctx, tx, r); err != nil {
		return Result{}, err
	} else if first && catalog.FirstPurchase != nil {
		bonusKey = key + ":first"
		bonusSource := cosmetics.Source{Type: SourceFirst, Ref: catalog.FirstPurchase.ID}
		bonus, err := s.engine.ApplyIn(ctx, tx, reward.Grant{Account: r.Account, Bundle: *catalog.FirstPurchase, Source: bonusSource, Reason: wallet.FirstPurchaseBonus, Key: bonusKey, ConfigVersion: int(version)}, now)
		if err != nil {
			return Result{}, err
		}
		result.Bonus = &bonus
	}
	if _, err := tx.Exec(ctx, `update purchases set state = $2, grant_key = $3, bonus_key = $4, granted_at = $5, refunded_at = null where id = $1`, r.ID, RecordGranted, key, bonusKey, now); err != nil {
		return Result{}, err
	}
	balance, err := s.wallet.BalanceIn(ctx, tx, r.Account)
	if err != nil {
		return Result{}, err
	}
	result.Balance = balance
	return result, s.publish(ctx, tx, r, EventGranted, now)
}

func (s *Store) first(ctx context.Context, tx pgx.Tx, r record) (bool, error) {
	var earlier int
	err := tx.QueryRow(ctx, `select count(*) from purchases where account_id = $1 and id <> $2 and state in ($3, $4)`, r.Account, r.ID, RecordGranted, RecordRefunded).Scan(&earlier)
	return earlier == 0, err
}

func (s *Store) load(ctx context.Context, tx pgx.Tx, r record) (Result, error) {
	result := Result{Platform: r.Platform, TransactionID: r.TransactionID, Offer: r.Offer, State: r.State, Repeated: true}
	if r.GrantKey != "" {
		report, err := reward.GetIn(ctx, tx, r.Account, r.GrantKey)
		if err != nil {
			return Result{}, err
		}
		result.Grant = &report
	}
	if r.BonusKey != "" {
		bonus, err := reward.GetIn(ctx, tx, r.Account, r.BonusKey)
		if err != nil {
			return Result{}, err
		}
		result.Bonus = &bonus
	}
	balance, err := s.wallet.BalanceIn(ctx, tx, r.Account)
	if err != nil {
		return Result{}, err
	}
	result.Balance = balance
	return result, nil
}

func (s *Store) finish(ctx context.Context, validator Validator, receipt Receipt) bool {
	if err := validator.Finish(ctx, receipt); err != nil {
		s.logger.Warn("purchase not finished with the store", "platform", receipt.Platform, "error", err)
		return false
	}
	return true
}

func (s *Store) Blocked(ctx context.Context, account string, now time.Time) (bool, error) {
	var refunds int
	since := now.Add(-time.Duration(s.config.RefundWindowDays) * 24 * time.Hour)
	if err := s.pool.QueryRow(ctx, `select count(*) from purchases where account_id = $1 and state = $2 and refunded_at >= $3`, account, RecordRefunded, since).Scan(&refunds); err != nil {
		return false, err
	}
	return refunds >= s.config.RefundsBeforePurchaseBlock, nil
}

func (s *Store) Notice(ctx context.Context, platform Platform, notice Notice, now time.Time) (bool, error) {
	if notice.ID == "" {
		return false, ErrMalformed
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `insert into purchase_notices (platform, notice_id, kind, received_at) values ($1, $2, $3, $4) on conflict do nothing`, platform, notice.ID, notice.Kind, now)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	handled, err := s.apply(ctx, tx, platform, notice, now)
	if err != nil {
		return false, err
	}
	return handled, tx.Commit(ctx)
}

func (s *Store) apply(ctx context.Context, tx pgx.Tx, platform Platform, notice Notice, now time.Time) (bool, error) {
	if notice.Kind == KindOther || notice.TransactionID == "" {
		return false, nil
	}
	r, err := s.lookup(ctx, tx, platform, notice.TransactionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := reward.LockIn(ctx, tx, r.Account); err != nil {
		return false, err
	}
	switch {
	case notice.Kind == KindPurchased && r.State == RecordPending:
		return s.settle(ctx, tx, platform, r, now)
	case notice.Kind == KindRefund && r.State == RecordGranted:
		return true, s.refund(ctx, tx, r, notice, now)
	case notice.Kind == KindReversal && r.State == RecordRefunded:
		return true, s.reverse(ctx, tx, r, notice, now)
	}
	return false, nil
}

func (s *Store) settle(ctx context.Context, tx pgx.Tx, platform Platform, r record, now time.Time) (bool, error) {
	validator, ok := s.validators[platform]
	if !ok {
		return false, ErrPlatform
	}
	receipt, err := validator.Verify(ctx, r.Product, r.TransactionID)
	if err != nil {
		return false, err
	}
	switch receipt.State {
	case StateCancelled:
		_, err := tx.Exec(ctx, `update purchases set state = $2 where id = $1`, r.ID, RecordCancelled)
		return true, err
	case StatePending:
		return false, nil
	}
	catalog, version, offer, err := s.offer(r.Offer)
	if err != nil {
		return false, err
	}
	if _, err := s.grant(ctx, tx, r, catalog, version, offer, now); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) refund(ctx context.Context, tx pgx.Tx, r record, notice Notice, now time.Time) error {
	revokeKey := "refund:" + string(r.Platform) + ":" + notice.ID
	for _, key := range []struct{ grant, revoke string }{{r.GrantKey, revokeKey}, {r.BonusKey, revokeKey + ":first"}} {
		if key.grant == "" {
			continue
		}
		if _, err := s.engine.RevokeIn(ctx, tx, r.Account, key.grant, wallet.RefundRevoke, key.revoke, now); err != nil && !errors.Is(err, reward.ErrAlreadyRevoke) {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `update purchases set state = $2, refunded_at = $3 where id = $1`, r.ID, RecordRefunded, now); err != nil {
		return err
	}
	return s.publish(ctx, tx, r, EventRefunded, now)
}

func (s *Store) reverse(ctx context.Context, tx pgx.Tx, r record, notice Notice, now time.Time) error {
	_, version, offer, err := s.offer(r.Offer)
	if err != nil {
		return err
	}
	key := "purchase:" + string(r.Platform) + ":" + r.TransactionID + ":reversal:" + notice.ID
	source := cosmetics.Source{Type: shop.SourceType, Ref: offer.ID}
	if _, err := s.engine.ApplyIn(ctx, tx, reward.Grant{Account: r.Account, Bundle: offer.Bundle, Source: source, Reason: wallet.RefundReversal, Key: key, ConfigVersion: int(version)}, now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `update purchases set state = $2, grant_key = $3, bonus_key = '', granted_at = $4, refunded_at = null where id = $1`, r.ID, RecordGranted, key, now); err != nil {
		return err
	}
	return s.publish(ctx, tx, r, EventReversed, now)
}

type event struct {
	Event         string    `json:"event"`
	Account       string    `json:"account_id"`
	Platform      Platform  `json:"platform"`
	TransactionID string    `json:"transaction_id"`
	Offer         string    `json:"offer"`
	At            time.Time `json:"at"`
}

func (s *Store) publish(ctx context.Context, tx pgx.Tx, r record, name string, now time.Time) error {
	encoded, err := json.Marshal(event{Event: name, Account: r.Account, Platform: r.Platform, TransactionID: r.TransactionID, Offer: r.Offer, At: now})
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into outbox (topic, key, payload) values ($1, $2, $3) on conflict (topic, key) do nothing`, Topic, string(r.Platform)+":"+r.TransactionID+":"+name+":"+now.UTC().Format(time.RFC3339Nano), encoded)
	return err
}

func (s *Store) evidence(ctx context.Context, account string, receipt Receipt, name string) {
	s.logger.Warn("purchase evidence", "event", name, "account", account, "platform", receipt.Platform, "product", receipt.ProductID)
	encoded, err := json.Marshal(event{Event: name, Account: account, Platform: receipt.Platform, TransactionID: receipt.TransactionID, At: receipt.PurchasedAt})
	if err != nil {
		return
	}
	if _, err := s.pool.Exec(ctx, `insert into outbox (topic, key, payload) values ($1, $2, $3) on conflict (topic, key) do nothing`, Topic, string(receipt.Platform)+":"+receipt.TransactionID+":"+name+":"+account, encoded); err != nil {
		s.logger.Error("purchase evidence not recorded", "error", err)
	}
}
