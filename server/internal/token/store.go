package token

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInvalidFamily   = errors.New("token: account and device are required")
	ErrUnknownRefresh  = errors.New("token: unknown refresh token")
	ErrRevoked         = errors.New("token: session revoked")
	ErrRefreshExpired  = errors.New("token: refresh token expired")
	ErrReused          = errors.New("token: refresh token reused, session revoked")
	ErrDeviceMismatch  = errors.New("token: refresh token presented by another device, session revoked")
	errMalformedFamily = errors.New("token: malformed session id")
)

const (
	ReasonLogout    = "logout"
	ReasonLogoutAll = "logout_all"
	ReasonReuse     = "reuse"
	ReasonDevice    = "device"
	ReasonExpired   = "expired"
	ReasonBanned    = "banned"
)

const (
	familyBytes = 16
	secretBytes = 32
)

type Family struct {
	ID         string
	Account    string
	Device     string
	Generation int
	ExpiresAt  time.Time
}

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Open(ctx context.Context, account, device string, now time.Time, ttl time.Duration) (Family, string, error) {
	if account == "" || device == "" {
		return Family{}, "", ErrInvalidFamily
	}
	secret, hash := newSecret()
	f := Family{Account: account, Device: device, Generation: 1, ExpiresAt: now.Add(ttl)}
	err := s.pool.QueryRow(ctx, `insert into auth_sessions (account_id, device_id, refresh_hash, created_at, last_seen_at, expires_at)
		values ($1, $2, $3, $4, $4, $5) returning id`, account, device, hash, now, f.ExpiresAt).Scan(&f.ID)
	if err != nil {
		return Family{}, "", err
	}
	refresh, err := encodeRefresh(f.ID, secret)
	return f, refresh, err
}

func (s *Store) Rotate(ctx context.Context, refresh, device string, now time.Time, ttl, grace time.Duration) (Family, string, error) {
	id, secret, err := decodeRefresh(refresh)
	if err != nil {
		return Family{}, "", ErrUnknownRefresh
	}
	presented := hashOf(secret)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Family{}, "", err
	}
	defer tx.Rollback(ctx)
	f := Family{ID: id}
	var current, previous []byte
	var rotatedAt, revokedAt *time.Time
	err = tx.QueryRow(ctx, `select account_id, device_id, refresh_hash, previous_hash, rotated_at, generation, expires_at, revoked_at
		from auth_sessions where id = $1 for update`, id).Scan(&f.Account, &f.Device, &current, &previous, &rotatedAt, &f.Generation, &f.ExpiresAt, &revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Family{}, "", ErrUnknownRefresh
	}
	if err != nil {
		return Family{}, "", err
	}
	retry := previous != nil && rotatedAt != nil && bytes.Equal(presented, previous) && now.Sub(*rotatedAt) < grace
	switch {
	case revokedAt != nil:
		return Family{}, "", ErrRevoked
	case !now.Before(f.ExpiresAt):
		return Family{}, "", revoke(ctx, tx, id, now, ReasonExpired, ErrRefreshExpired)
	case device != f.Device:
		return Family{}, "", revoke(ctx, tx, id, now, ReasonDevice, ErrDeviceMismatch)
	case !bytes.Equal(presented, current) && !retry:
		return Family{}, "", revoke(ctx, tx, id, now, ReasonReuse, ErrReused)
	}
	next, hash := newSecret()
	f.Generation++
	f.ExpiresAt = now.Add(ttl)
	if _, err := tx.Exec(ctx, `update auth_sessions set refresh_hash = $2, previous_hash = $3, rotated_at = $4, last_seen_at = $4, generation = $5, expires_at = $6
		where id = $1`, id, hash, presented, now, f.Generation, f.ExpiresAt); err != nil {
		return Family{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return Family{}, "", err
	}
	token, err := encodeRefresh(id, next)
	return f, token, err
}

func revoke(ctx context.Context, tx pgx.Tx, id string, now time.Time, reason string, cause error) error {
	if _, err := tx.Exec(ctx, `update auth_sessions set revoked_at = $2, revoke_reason = $3 where id = $1`, id, now, reason); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return cause
}

func (s *Store) Revoke(ctx context.Context, refresh string, now time.Time) error {
	id, secret, err := decodeRefresh(refresh)
	if err != nil {
		return nil
	}
	_, err = s.pool.Exec(ctx, `update auth_sessions set revoked_at = $2, revoke_reason = $3
		where id = $1 and revoked_at is null and (refresh_hash = $4 or previous_hash = $4)`, id, now, ReasonLogout, hashOf(secret))
	return err
}

func (s *Store) RevokeAccount(ctx context.Context, account string, now time.Time) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	revoked, err := RevokeAccountIn(ctx, tx, account, ReasonLogoutAll, now)
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `delete from push_devices where account_id = $1`, account); err != nil {
		return 0, err
	}
	return revoked, tx.Commit(ctx)
}

type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func RevokeAccountIn(ctx context.Context, db Execer, account, reason string, now time.Time) (int, error) {
	tag, err := db.Exec(ctx, `update auth_sessions set revoked_at = $2, revoke_reason = $3 where account_id = $1 and revoked_at is null`, account, now, reason)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func newSecret() ([]byte, []byte) {
	secret := make([]byte, secretBytes)
	rand.Read(secret)
	return secret, hashOf(secret)
}

func hashOf(secret []byte) []byte {
	digest := sha256.Sum256(secret)
	return digest[:]
}

func encodeRefresh(id string, secret []byte) (string, error) {
	raw, err := hex.DecodeString(strings.ReplaceAll(id, "-", ""))
	if err != nil || len(raw) != familyBytes {
		return "", errMalformedFamily
	}
	return base64.RawURLEncoding.EncodeToString(append(raw, secret...)), nil
}

func decodeRefresh(refresh string) (string, []byte, error) {
	raw, err := base64.RawURLEncoding.Strict().DecodeString(refresh)
	if err != nil || len(raw) != familyBytes+secretBytes {
		return "", nil, ErrUnknownRefresh
	}
	id := hex.EncodeToString(raw[:familyBytes])
	return id[:8] + "-" + id[8:12] + "-" + id[12:16] + "-" + id[16:20] + "-" + id[20:], raw[familyBytes:], nil
}
