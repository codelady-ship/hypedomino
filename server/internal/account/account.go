package account

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

const (
	StateActive   = "active"
	StateDeleting = "deleting"
	StateDeleted  = "deleted"
	StateBanned   = "banned"
)

var (
	ErrBanned          = errors.New("account: banned")
	ErrDeleted         = errors.New("account: deleted")
	ErrDeletionPending = errors.New("account: deletion pending")
	ErrInvalidIdentity = errors.New("account: identity has no provider or subject")
)

type Account struct {
	ID          string
	State       string
	Country     string
	Email       string
	CreatedAt   time.Time
	BannedUntil *time.Time
	BanCategory string
	BanSanction int64
	DeletionAt  *time.Time
}

type BannedError struct {
	Account    string
	Category   string
	SanctionID int64
	Until      *time.Time
}

func (e *BannedError) Error() string {
	return ErrBanned.Error()
}

func (e *BannedError) Is(target error) bool {
	return target == ErrBanned
}

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) SignIn(ctx context.Context, identity idtoken.Identity) (Account, bool, error) {
	if identity.Provider.String() == "invalid" || identity.Subject == "" {
		return Account{}, false, ErrInvalidIdentity
	}
	for {
		a, found, err := s.find(ctx, identity)
		if err != nil {
			return Account{}, false, err
		}
		if found {
			return a, false, admitted(a, time.Now())
		}
		a, created, err := s.create(ctx, identity)
		if err != nil {
			return Account{}, false, err
		}
		if created {
			return a, true, nil
		}
	}
}

func (s *Store) find(ctx context.Context, identity idtoken.Identity) (Account, bool, error) {
	var a Account
	err := s.pool.QueryRow(ctx, `update auth_identities i set email = coalesce(nullif($3, ''), i.email)
		from accounts a left join lateral (select s.id, s.category from account_sanctions s where s.account_id = a.id and s.kind in ('temporary_ban', 'permanent_ban')
				and s.lifted_at is null and (s.ends_at is null or s.ends_at > now()) order by s.starts_at desc, s.id desc limit 1) ban on true
		where a.id = i.account_id and i.provider = $1 and i.subject = $2
		returning a.id, a.state, coalesce(a.country, ''), coalesce(i.email, ''), a.created_at, a.banned_until, a.deletion_requested_at,
			coalesce(ban.category, ''), coalesce(ban.id, 0)`,
		identity.Provider.String(), identity.Subject, identity.Email).Scan(&a.ID, &a.State, &a.Country, &a.Email, &a.CreatedAt, &a.BannedUntil, &a.DeletionAt, &a.BanCategory, &a.BanSanction)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, false, nil
	}
	return a, err == nil, err
}

func (s *Store) create(ctx context.Context, identity idtoken.Identity) (Account, bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Account{}, false, err
	}
	defer tx.Rollback(ctx)
	a := Account{Email: identity.Email}
	if err := tx.QueryRow(ctx, `insert into accounts default values returning id, state, created_at`).Scan(&a.ID, &a.State, &a.CreatedAt); err != nil {
		return Account{}, false, err
	}
	_, err = tx.Exec(ctx, `insert into auth_identities (provider, subject, account_id, email) values ($1, $2, $3, nullif($4, ''))`,
		identity.Provider.String(), identity.Subject, a.ID, identity.Email)
	if isUniqueViolation(err) {
		return Account{}, false, nil
	}
	if err != nil {
		return Account{}, false, err
	}
	return a, true, tx.Commit(ctx)
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func (a Account) banned(until *time.Time) error {
	category := a.BanCategory
	if category == "" {
		category = string(sanction.Other)
	}
	return &BannedError{Account: a.ID, Category: category, SanctionID: a.BanSanction, Until: until}
}

func admitted(a Account, now time.Time) error {
	switch {
	case a.State == StateBanned:
		return a.banned(nil)
	case a.State == StateDeleted:
		return ErrDeleted
	case a.State == StateDeleting:
		return ErrDeletionPending
	case a.BannedUntil != nil && now.Before(*a.BannedUntil):
		until := a.BannedUntil.UTC()
		return a.banned(&until)
	}
	return nil
}
