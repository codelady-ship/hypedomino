package wallet

import (
	"context"
	"errors"
	"fmt"
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
)

var signs = map[Reason]int{
	MatchReward: 1, FirstWin: 1, DailyLogin: 1, Mission: 1, PassReward: 1, RankReward: 1, EventReward: 1,
	DuplicateCompensation: 1, Promo: 1, Purchase: 1, FirstPurchaseBonus: 1, RefundReversal: 1,
	ShopPurchase: -1, PassLevelPurchase: -1, Rename: -1, RefundRevoke: -1,
	AdminAdjust: 0,
}

func Known(reason Reason) bool {
	_, known := signs[reason]
	return known
}

var (
	ErrInvalid       = errors.New("wallet: invalid entry")
	ErrUnknownReason = errors.New("wallet: unknown reason")
	ErrWrongSign     = errors.New("wallet: amount sign does not match the reason")
	ErrInsufficient  = errors.New("wallet: insufficient balance")
	ErrPurchaseReuse = errors.New("wallet: purchase already granted to another account")
)

type Source struct {
	Type string
	Ref  string
}

type Entry struct {
	Account       string
	Amount        int64
	Reason        Reason
	Source        Source
	Key           string
	ConfigVersion int
}

type Applied struct {
	ID            int64
	Account       string
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
	Stored   int64
	Computed int64
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (e Entry) Validate() error {
	sign, known := signs[e.Reason]
	switch {
	case e.Account == "" || e.Key == "" || e.Source.Type == "" || e.Source.Ref == "":
		return fmt.Errorf("%w: account, key and source are required", ErrInvalid)
	case e.Amount == 0:
		return fmt.Errorf("%w: amount is zero", ErrInvalid)
	case !known:
		return fmt.Errorf("%w: %q", ErrUnknownReason, e.Reason)
	case sign > 0 && e.Amount < 0, sign < 0 && e.Amount > 0:
		return fmt.Errorf("%w: %s with %d", ErrWrongSign, e.Reason, e.Amount)
	}
	return nil
}

const appliedColumns = `id, account_id, amount, reason, source_type, source_ref, idempotency_key, config_version, balance_after, created_at`

func scan(row pgx.Row) (Applied, error) {
	var a Applied
	if err := row.Scan(&a.ID, &a.Account, &a.Amount, &a.Reason, &a.Source.Type, &a.Source.Ref, &a.Key, &a.ConfigVersion, &a.BalanceAfter, &a.At); err != nil {
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
	if _, err := tx.Exec(ctx, `insert into wallets (account_id, updated_at) values ($1, $2) on conflict do nothing`, entry.Account, now); err != nil {
		return Applied{}, err
	}
	var balance int64
	if err := tx.QueryRow(ctx, `select balance from wallets where account_id = $1 for update`, entry.Account).Scan(&balance); err != nil {
		return Applied{}, err
	}
	earlier, err := scan(tx.QueryRow(ctx, `select `+appliedColumns+` from ledger_entries where account_id = $1 and idempotency_key = $2`, entry.Account, entry.Key))
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
	applied, err := scan(tx.QueryRow(ctx, `insert into ledger_entries (account_id, amount, reason, source_type, source_ref, idempotency_key, config_version, balance_after, created_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9) returning `+appliedColumns,
		entry.Account, entry.Amount, string(entry.Reason), entry.Source.Type, entry.Source.Ref, entry.Key, entry.ConfigVersion, after, now))
	if isUniqueViolation(err, "ledger_entries_purchase_idx") {
		return Applied{}, ErrPurchaseReuse
	}
	if err != nil {
		return Applied{}, err
	}
	if _, err := tx.Exec(ctx, `update wallets set balance = $2, updated_at = $3 where account_id = $1`, entry.Account, after, now); err != nil {
		return Applied{}, err
	}
	return applied, nil
}

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (s *Store) Balance(ctx context.Context, account string) (int64, error) {
	return s.BalanceIn(ctx, s.pool, account)
}

func (s *Store) CountIn(ctx context.Context, q Querier, account string, reason Reason) (int, error) {
	var count int
	err := q.QueryRow(ctx, `select count(*) from ledger_entries where account_id = $1 and reason = $2`, account, reason).Scan(&count)
	return count, err
}

func (s *Store) BalanceIn(ctx context.Context, q Querier, account string) (int64, error) {
	var balance int64
	err := q.QueryRow(ctx, `select balance from wallets where account_id = $1`, account).Scan(&balance)
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

const mismatchQuery = `select w.account_id, w.balance, coalesce(sum(l.amount), 0)
	from wallets w left join ledger_entries l on l.account_id = w.account_id
	group by w.account_id, w.balance having w.balance <> coalesce(sum(l.amount), 0)`

func (s *Store) Audit(ctx context.Context) ([]Mismatch, error) {
	rows, err := s.pool.Query(ctx, mismatchQuery+` order by w.account_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var mismatches []Mismatch
	for rows.Next() {
		var m Mismatch
		if err := rows.Scan(&m.Account, &m.Stored, &m.Computed); err != nil {
			return nil, err
		}
		mismatches = append(mismatches, m)
	}
	return mismatches, rows.Err()
}

func (s *Store) Reconcile(ctx context.Context, account string, now time.Time) (Mismatch, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Mismatch{}, false, err
	}
	defer tx.Rollback(ctx)
	m := Mismatch{Account: account}
	if err := tx.QueryRow(ctx, `select balance from wallets where account_id = $1 for update`, account).Scan(&m.Stored); err != nil {
		return Mismatch{}, false, err
	}
	if err := tx.QueryRow(ctx, `select coalesce(sum(amount), 0) from ledger_entries where account_id = $1`, account).Scan(&m.Computed); err != nil {
		return Mismatch{}, false, err
	}
	if m.Stored == m.Computed {
		return m, false, nil
	}
	if _, err := tx.Exec(ctx, `update wallets set balance = $2, updated_at = $3 where account_id = $1`, account, m.Computed, now); err != nil {
		return Mismatch{}, false, err
	}
	return m, true, tx.Commit(ctx)
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == constraint
}
