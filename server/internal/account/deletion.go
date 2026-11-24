package account

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
)

var ErrNoDeletionPending = errors.New("account: no deletion pending")

type Deletion struct {
	Account   string
	Tombstone string
	At        time.Time
}

type DeletionLog interface {
	Append(ctx context.Context, d Deletion) error
}

type erasure int

const (
	eraseRows erasure = iota + 1
	moveToTombstone
)

var erasures = map[string]erasure{
	"auth_identities":      eraseRows,
	"auth_sessions":        eraseRows,
	"app_attest_keys":      eraseRows,
	"push_devices":         eraseRows,
	"push_settings":        eraseRows,
	"profiles":             eraseRows,
	"nickname_history":     eraseRows,
	"inventory_items":      eraseRows,
	"wallets":              eraseRows,
	"ledger_entries":       eraseRows,
	"reward_grants":        eraseRows,
	"chest_opens":          eraseRows,
	"promo_targets":        eraseRows,
	"promo_redemptions":    eraseRows,
	"promo_attempts":       eraseRows,
	"pass_progress":        eraseRows,
	"pass_xp_events":       eraseRows,
	"medal_ownership":      eraseRows,
	"medal_showcases":      eraseRows,
	"mission_assignments":  eraseRows,
	"mission_matches":      eraseRows,
	"daily_logins":         eraseRows,
	"friendships":          eraseRows,
	"friend_requests":      eraseRows,
	"blocks":               eraseRows,
	"friend_codes":         eraseRows,
	"ratings":              eraseRows,
	"analytics_ids":        eraseRows,
	"appeals":              eraseRows,
	"match_participants":   moveToTombstone,
	"rating_changes":       moveToTombstone,
	"season_results":       moveToTombstone,
	"season_rewards":       moveToTombstone,
	"leaderboard_emblems":  moveToTombstone,
	"leaderboard_removals": moveToTombstone,
	"emblem_showcases":     moveToTombstone,
	"rating_signals":       moveToTombstone,
	"purchases":            moveToTombstone,
	"penalty_records":      moveToTombstone,
	"risk_signals":         moveToTombstone,
	"risk_reviews":         moveToTombstone,
	"risk_actions":         moveToTombstone,
	"account_sanctions":    moveToTombstone,
	"player_reports":       moveToTombstone,
	"consent_records":      moveToTombstone,
	"data_exports":         moveToTombstone,
}

type reference struct {
	table  string
	column string
}

func (s *Store) Lookup(ctx context.Context, identity idtoken.Identity) (Account, error) {
	a, found, err := s.find(ctx, identity)
	if err != nil {
		return Account{}, err
	}
	if !found {
		return Account{}, ErrUnknown
	}
	return a, admitted(a, time.Now())
}

func (s *Store) RequestDeletion(ctx context.Context, id string, now time.Time) (time.Time, error) {
	var requested time.Time
	err := s.pool.QueryRow(ctx, `with requested as (update accounts set state = $2, deletion_requested_at = $3
		where id = $1 and state = $4 and (banned_until is null or banned_until <= $3) returning id, deletion_requested_at),
		unregistered as (delete from push_devices where account_id in (select id from requested))
		select deletion_requested_at from requested`,
		id, StateDeleting, now.UTC(), StateActive).Scan(&requested)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, ErrBanned
	}
	return requested.UTC(), err
}

func (s *Store) CancelDeletion(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `update accounts set state = $2, deletion_requested_at = null where id = $1 and state = $3`, id, StateActive, StateDeleting)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoDeletionPending
	}
	return nil
}

func (s *Store) DeleteDue(ctx context.Context, log DeletionLog, requestedBefore, now time.Time) (int, error) {
	rows, err := s.pool.Query(ctx, `select id::text from accounts where state = $1 and deletion_requested_at <= $2 order by deletion_requested_at`, StateDeleting, requestedBefore.UTC())
	if err != nil {
		return 0, err
	}
	due, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return 0, err
	}
	var eraseErr error
	for _, id := range due {
		if err := s.eraseDue(ctx, id, requestedBefore, now); err != nil {
			eraseErr = fmt.Errorf("erase %s: %w", id, err)
			break
		}
	}
	logged, err := s.LogDeletions(ctx, log)
	return logged, errors.Join(eraseErr, err)
}

func (s *Store) LogDeletions(ctx context.Context, log DeletionLog) (int, error) {
	pending, err := s.UnloggedDeletions(ctx)
	if err != nil {
		return 0, err
	}
	for logged, d := range pending {
		if err := log.Append(ctx, d); err != nil {
			return logged, fmt.Errorf("log deletion of %s: %w", d.Account, err)
		}
		if _, err := s.pool.Exec(ctx, `delete from deletion_log_pending where account_id = $1`, d.Account); err != nil {
			return logged, err
		}
	}
	return len(pending), nil
}

func (s *Store) UnloggedDeletions(ctx context.Context) ([]Deletion, error) {
	rows, err := s.pool.Query(ctx, `select account_id::text, tombstone_id::text, deleted_at from deletion_log_pending order by deleted_at, account_id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Deletion, error) {
		var d Deletion
		err := row.Scan(&d.Account, &d.Tombstone, &d.At)
		d.At = d.At.UTC()
		return d, err
	})
}

func (s *Store) Reapply(ctx context.Context, d Deletion) (bool, error) {
	if d.Account == "" || d.Tombstone == "" {
		return false, fmt.Errorf("account: deletion %+v has no account or tombstone id", d)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	var present bool
	err = tx.QueryRow(ctx, `select true from accounts where id = $1 and state <> $2 for update`, d.Account, StateDeleted).Scan(&present)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := erase(ctx, tx, d); err != nil {
		return false, err
	}
	return true, tx.Commit(ctx)
}

func (s *Store) eraseDue(ctx context.Context, id string, requestedBefore, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var due bool
	err = tx.QueryRow(ctx, `select state = $2 and deletion_requested_at <= $3 from accounts where id = $1 for update`, id, StateDeleting, requestedBefore.UTC()).Scan(&due)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !due) {
		return nil
	}
	if err != nil {
		return err
	}
	d, err := erase(ctx, tx, Deletion{Account: id, At: now})
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `insert into deletion_log_pending (account_id, tombstone_id, deleted_at) values ($1, $2, $3)`, d.Account, d.Tombstone, d.At); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func erase(ctx context.Context, tx pgx.Tx, d Deletion) (Deletion, error) {
	references, err := accountReferences(ctx, tx)
	if err != nil {
		return Deletion{}, err
	}
	d.At = d.At.UTC().Truncate(time.Microsecond)
	if err := tx.QueryRow(ctx, `insert into accounts (id, state, created_at, deleted_at) values (coalesce(nullif($1, '')::uuid, gen_random_uuid()), $2, $3, $3) returning id::text`,
		d.Tombstone, StateDeleted, d.At).Scan(&d.Tombstone); err != nil {
		return Deletion{}, err
	}
	if err := holdNickname(ctx, tx, d.Account, d.Tombstone, d.At); err != nil {
		return Deletion{}, err
	}
	for _, r := range references {
		if err := apply(ctx, tx, r, d.Account, d.Tombstone); err != nil {
			return Deletion{}, err
		}
	}
	if _, err := tx.Exec(ctx, `delete from accounts where id = $1`, d.Account); err != nil {
		return Deletion{}, err
	}
	return d, nil
}

func holdNickname(ctx context.Context, tx pgx.Tx, id, tombstone string, now time.Time) error {
	_, err := tx.Exec(ctx, `insert into nickname_history (account_id, nickname, nickname_key, replaced_at)
		select $2, nickname, nickname_key, $3 from profiles where account_id = $1`, id, tombstone, now.UTC())
	return err
}

func apply(ctx context.Context, tx pgx.Tx, r reference, id, tombstone string) error {
	table, column := pgx.Identifier{r.table}.Sanitize(), pgx.Identifier{r.column}.Sanitize()
	var err error
	switch erasures[r.table] {
	case eraseRows:
		_, err = tx.Exec(ctx, `delete from `+table+` where `+column+` = $1`, id)
	case moveToTombstone:
		_, err = tx.Exec(ctx, `update `+table+` set `+column+` = $2 where `+column+` = $1`, id, tombstone)
	default:
		err = fmt.Errorf("table %s references accounts and has no erasure rule", r.table)
	}
	return err
}

func accountReferences(ctx context.Context, tx pgx.Tx) ([]reference, error) {
	rows, err := tx.Query(ctx, `select c.conrelid::regclass::text, a.attname::text from pg_constraint c
		join pg_attribute a on a.attrelid = c.conrelid and a.attnum = any(c.conkey)
		where c.contype = 'f' and c.confrelid = 'accounts'::regclass and c.conparentid = 0
		order by 1, 2`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (reference, error) {
		var r reference
		err := row.Scan(&r.table, &r.column)
		return r, err
	})
}
