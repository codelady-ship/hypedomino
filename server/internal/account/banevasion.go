package account

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
)

const MinBanEvasionKeyBytes = 32

var (
	ErrBanEvasionKey               = errors.New("account: ban-evasion key must be at least 32 bytes")
	ErrPermanentBanEvasionDisabled = errors.New("account: a permanent ban needs a positive ban-evasion retention")
)

type IdentityHasher struct {
	key []byte
}

func NewIdentityHasher(key string) (IdentityHasher, error) {
	if len(key) < MinBanEvasionKeyBytes {
		return IdentityHasher{}, ErrBanEvasionKey
	}
	return IdentityHasher{key: []byte(key)}, nil
}

func (h IdentityHasher) Hash(provider, subject string) []byte {
	mac := hmac.New(sha256.New, h.key)
	mac.Write([]byte(provider))
	mac.Write([]byte{0})
	mac.Write([]byte(subject))
	return mac.Sum(nil)
}

type BanEvasion struct {
	Hasher             IdentityHasher
	PermanentRetention time.Duration
}

type evadedBan struct {
	category sanction.Category
	endsAt   *time.Time
}

func (s *Store) EnableBanEvasion(evasion BanEvasion) {
	s.evasion = &evasion
}

func (s *Store) PurgeBanEvasion(ctx context.Context, now time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, `delete from ban_evasion_hashes where expires_at <= $1`, now.UTC())
	return tag.RowsAffected(), err
}

func (s *Store) recordBanEvasion(ctx context.Context, tx pgx.Tx, d Deletion) error {
	if s.evasion == nil {
		return nil
	}
	ban, banned, err := runningBan(ctx, tx, d.Account, d.At)
	if err != nil || !banned {
		return err
	}
	expires, err := s.evasion.expiry(ban, d.At)
	if err != nil {
		return err
	}
	hashes, err := s.identityHashes(ctx, tx, d.Account)
	if err != nil {
		return err
	}
	for _, hash := range hashes {
		if _, err := tx.Exec(ctx, `insert into ban_evasion_hashes (identity_hash, category, ban_ends_at, expires_at) values ($1, $2, $3, $4)
			on conflict (identity_hash) do update set category = excluded.category, ban_ends_at = excluded.ban_ends_at, expires_at = excluded.expires_at
			where excluded.expires_at > ban_evasion_hashes.expires_at`, hash, string(ban.category), ban.endsAt, expires); err != nil {
			return err
		}
	}
	return nil
}

func (b *BanEvasion) expiry(ban evadedBan, deletedAt time.Time) (time.Time, error) {
	if ban.endsAt != nil {
		return ban.endsAt.UTC(), nil
	}
	if b.PermanentRetention <= 0 {
		return time.Time{}, ErrPermanentBanEvasionDisabled
	}
	return deletedAt.Add(b.PermanentRetention).UTC(), nil
}

func runningBan(ctx context.Context, tx pgx.Tx, id string, at time.Time) (evadedBan, bool, error) {
	var ban evadedBan
	err := tx.QueryRow(ctx, `select category, ends_at from account_sanctions where account_id = $1 and kind in ($3, $4)
		and lifted_at is null and (ends_at is null or ends_at > $2) order by ends_at desc nulls first limit 1`,
		id, at.UTC(), string(sanction.TemporaryBan), string(sanction.PermanentBan)).Scan(&ban.category, &ban.endsAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return evadedBan{}, false, nil
	}
	return ban, err == nil, err
}

func (s *Store) identityHashes(ctx context.Context, tx pgx.Tx, id string) ([][]byte, error) {
	rows, err := tx.Query(ctx, `select provider, subject from auth_identities where account_id = $1`, id)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) ([]byte, error) {
		var provider, subject string
		err := row.Scan(&provider, &subject)
		return s.evasion.Hasher.Hash(provider, subject), err
	})
}

func (s *Store) inheritBan(ctx context.Context, tx pgx.Tx, a *Account, provider, subject string, now time.Time) error {
	if s.evasion == nil {
		return nil
	}
	var ban evadedBan
	err := tx.QueryRow(ctx, `select category, ban_ends_at from ban_evasion_hashes where identity_hash = $1 and expires_at > $2`,
		s.evasion.Hasher.Hash(provider, subject), now.UTC()).Scan(&ban.category, &ban.endsAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	kind, update, value := sanction.TemporaryBan, `update accounts set banned_until = $2 where id = $1 returning state, banned_until`, any(ban.endsAt)
	if ban.endsAt == nil {
		kind, update, value = sanction.PermanentBan, `update accounts set state = $2 where id = $1 returning state, banned_until`, StateBanned
	}
	if err := tx.QueryRow(ctx, update, a.ID, value).Scan(&a.State, &a.BannedUntil); err != nil {
		return err
	}
	imposed, err := sanction.Impose(ctx, tx, a.ID, kind, ban.category, now.UTC(), ban.endsAt)
	if err != nil {
		return err
	}
	a.BanCategory, a.BanSanction = string(imposed.Category), imposed.ID
	return nil
}
