package wallet

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Reason string

const (
	MatchReward           Reason = "match_reward"
	FirstWin              Reason = "first_win"
	DailyLogin            Reason = "daily_login"
	Mission               Reason = "mission"
	PassReward            Reason = "pass_reward"
	RankReward            Reason = "rank_reward"
	EventReward           Reason = "event_reward"
	DuplicateCompensation Reason = "duplicate_compensation"
	Promo                 Reason = "promo"
	Purchase              Reason = "purchase"
	FirstPurchaseBonus    Reason = "first_purchase_bonus"
	ShopPurchase          Reason = "shop_purchase"
	PassLevelPurchase     Reason = "pass_level_purchase"
	Rename                Reason = "rename"
	RefundRevoke          Reason = "refund_revoke"
	RefundReversal        Reason = "refund_reversal"
	AdminAdjust           Reason = "admin_adjust"
	CoinPackConversion    Reason = "coin_pack_conversion"
	ChestPurchase         Reason = "chest_purchase"
)

type Currency string

const (
	Coin    Currency = "coin"
	Crystal Currency = "crystal"
)

var Currencies = []Currency{Coin, Crystal}

type rule struct {
	sign       int
	currencies []Currency
}

var (
	coinOnly    = []Currency{Coin}
	crystalOnly = []Currency{Crystal}
	either      = []Currency{Coin, Crystal}
)

var rules = map[Reason]rule{
	MatchReward: {1, coinOnly}, FirstWin: {1, coinOnly}, DailyLogin: {1, coinOnly}, Mission: {1, coinOnly},
	RankReward: {1, coinOnly}, EventReward: {1, coinOnly}, DuplicateCompensation: {1, coinOnly}, Promo: {1, coinOnly},
	PassReward: {1, either}, Purchase: {1, crystalOnly}, FirstPurchaseBonus: {1, crystalOnly}, RefundReversal: {1, crystalOnly},
	ShopPurchase: {-1, either}, PassLevelPurchase: {-1, either}, Rename: {-1, coinOnly}, ChestPurchase: {-1, coinOnly}, RefundRevoke: {-1, crystalOnly},
	AdminAdjust: {0, either}, CoinPackConversion: {0, either},
}

func Known(reason Reason) bool {
	_, known := rules[reason]
	return known
}

func Moves(reason Reason, currency Currency) bool {
	return slices.Contains(rules[reason].currencies, currency)
}

var (
	ErrInvalid       = errors.New("wallet: invalid entry")
	ErrUnknownReason = errors.New("wallet: unknown reason")
	ErrWrongSign     = errors.New("wallet: amount sign does not match the reason")
	ErrWrongCurrency = errors.New("wallet: currency does not match the reason")
	ErrInsufficient  = errors.New("wallet: insufficient balance")
	ErrPurchaseReuse = errors.New("wallet: purchase already granted to another account")
)

type Source struct {
	Type string
	Ref  string
}

type Entry struct {
	Account       string
	Currency      Currency
	Amount        int64
	Reason        Reason
	Source        Source
	Key           string
	ConfigVersion int
}

type Applied struct {
	ID            int64
	Account       string
	Currency      Currency
	Amount        int64
	Reason        Reason
	Source        Source
	Key           string
	ConfigVersion int
	BalanceAfter  int64
	At            time.Time
	Repeated      bool
}

type Mismatch struct {
	Account  string
	Currency Currency
	Stored   int64
	Computed int64
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (e Entry) withCurrency() Entry {
	if e.Currency == "" && Known(e.Reason) {
		e.Currency = rules[e.Reason].currencies[0]
	}
	return e
}

func (e Entry) Validate() error {
	r, known := rules[e.Reason]
	e = e.withCurrency()
	switch {
	case e.Account == "" || e.Key == "" || e.Source.Type == "" || e.Source.Ref == "":
		return fmt.Errorf("%w: account, key and source are required", ErrInvalid)
	case e.Amount == 0:
		return fmt.Errorf("%w: amount is zero", ErrInvalid)
	case !known:
		return fmt.Errorf("%w: %q", ErrUnknownReason, e.Reason)
	case r.sign > 0 && e.Amount < 0, r.sign < 0 && e.Amount > 0:
		return fmt.Errorf("%w: %s with %d", ErrWrongSign, e.Reason, e.Amount)
	case !Moves(e.Reason, e.Currency):
		return fmt.Errorf("%w: %s in %q", ErrWrongCurrency, e.Reason, e.Currency)
	}
	return nil
}

const appliedColumns = `id, account_id, currency, amount, reason, source_type, source_ref, idempotency_key, config_version, balance_after, created_at`

func scan(row pgx.Row) (Applied, error) {
	var a Applied
	if err := row.Scan(&a.ID, &a.Account, &a.Currency, &a.Amount, &a.Reason, &a.Source.Type, &a.Source.Ref, &a.Key, &a.ConfigVersion, &a.BalanceAfter, &a.At); err != nil {
		return Applied{}, err
	}
	a.At = a.At.UTC()
	return a, nil
}

func (s *Store) Apply(ctx context.Context, entry Entry, now time.Time) (Applied, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Applied{}, err
	}
	defer tx.Rollback(ctx)
	applied, err := s.ApplyIn(ctx, tx, entry, now)
	if err != nil {
		return Applied{}, err
	}
	return applied, tx.Commit(ctx)
}

func (s *Store) ApplyIn(ctx context.Context, tx pgx.Tx, entry Entry, now time.Time) (Applied, error) {
	if err := entry.Validate(); err != nil {
		return Applied{}, err
	}
	entry = entry.withCurrency()
	if _, err := tx.Exec(ctx, `insert into wallets (account_id, currency, updated_at) values ($1, $2, $3) on conflict do nothing`, entry.Account, entry.Currency, now); err != nil {
		return Applied{}, err
	}
	var balance int64
	if err := tx.QueryRow(ctx, `select balance from wallets where account_id = $1 and currency = $2 for update`, entry.Account, entry.Currency).Scan(&balance); err != nil {
		return Applied{}, err
	}
	earlier, err := scan(tx.QueryRow(ctx, `select `+appliedColumns+` from ledger_entries where account_id = $1 and currency = $2 and idempotency_key = $3`, entry.Account, entry.Currency, entry.Key))
	if err == nil {
		earlier.Repeated = true
		return earlier, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Applied{}, err
	}
	after := balance + entry.Amount
	if after < 0 && entry.Amount < 0 && entry.Reason != RefundRevoke {
		return Applied{}, fmt.Errorf("%w: %d available, %d needed", ErrInsufficient, balance, -entry.Amount)
	}
	applied, err := scan(tx.QueryRow(ctx, `insert into ledger_entries (account_id, currency, amount, reason, source_type, source_ref, idempotency_key, config_version, balance_after, created_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) returning `+appliedColumns,
		entry.Account, entry.Currency, entry.Amount, string(entry.Reason), entry.Source.Type, entry.Source.Ref, entry.Key, entry.ConfigVersion, after, now))
	if isUniqueViolation(err, "ledger_entries_purchase_idx") {
		return Applied{}, ErrPurchaseReuse
	}
	if err != nil {
		return Applied{}, err
	}
	if _, err := tx.Exec(ctx, `update wallets set balance = $3, updated_at = $4 where account_id = $1 and currency = $2`, entry.Account, entry.Currency, after, now); err != nil {
		return Applied{}, err
	}
	return applied, nil
}

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (s *Store) Balance(ctx context.Context, account string, currency Currency) (int64, error) {
	return s.BalanceIn(ctx, s.pool, account, currency)
}

func (s *Store) CountIn(ctx context.Context, q Querier, account string, reason Reason) (int, error) {
	var count int
	err := q.QueryRow(ctx, `select count(*) from ledger_entries where account_id = $1 and reason = $2`, account, reason).Scan(&count)
	return count, err
}

func (s *Store) BalanceIn(ctx context.Context, q Querier, account string, currency Currency) (int64, error) {
	var balance int64
	err := q.QueryRow(ctx, `select balance from wallets where account_id = $1 and currency = $2`, account, currency).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return balance, err
}

func (s *Store) History(ctx context.Context, account string, limit int) ([]Applied, error) {
	rows, err := s.pool.Query(ctx, `select `+appliedColumns+` from ledger_entries where account_id = $1 order by id desc limit $2`, account, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []Applied
	for rows.Next() {
		entry, err := scan(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

const mismatchQuery = `select w.account_id, w.currency, w.balance, coalesce(sum(l.amount), 0)
	from wallets w left join ledger_entries l on l.account_id = w.account_id and l.currency = w.currency
	group by w.account_id, w.currency, w.balance having w.balance <> coalesce(sum(l.amount), 0)`

func (s *Store) Audit(ctx context.Context) ([]Mismatch, error) {
	rows, err := s.pool.Query(ctx, mismatchQuery+` order by w.account_id, w.currency`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var mismatches []Mismatch
	for rows.Next() {
		var m Mismatch
		if err := rows.Scan(&m.Account, &m.Currency, &m.Stored, &m.Computed); err != nil {
			return nil, err
		}
		mismatches = append(mismatches, m)
	}
	return mismatches, rows.Err()
}

func (s *Store) Reconcile(ctx context.Context, account string, currency Currency, now time.Time) (Mismatch, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Mismatch{}, false, err
	}
	defer tx.Rollback(ctx)
	m := Mismatch{Account: account, Currency: currency}
	if err := tx.QueryRow(ctx, `select balance from wallets where account_id = $1 and currency = $2 for update`, account, currency).Scan(&m.Stored); err != nil {
		return Mismatch{}, false, err
	}
	if err := tx.QueryRow(ctx, `select coalesce(sum(amount), 0) from ledger_entries where account_id = $1 and currency = $2`, account, currency).Scan(&m.Computed); err != nil {
		return Mismatch{}, false, err
	}
	if m.Stored == m.Computed {
		return m, false, nil
	}
	if _, err := tx.Exec(ctx, `update wallets set balance = $3, updated_at = $4 where account_id = $1 and currency = $2`, account, currency, m.Computed, now); err != nil {
		return Mismatch{}, false, err
	}
	return m, true, tx.Commit(ctx)
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}
