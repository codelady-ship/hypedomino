package adminsession

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRejected = errors.New("adminsession: unknown, revoked, idle or expired session")

type Config struct {
	Idle   time.Duration
	Max    time.Duration
	StepUp time.Duration
}

func DefaultConfig() Config {
	return Config{Idle: 30 * time.Minute, Max: 12 * time.Hour, StepUp: 10 * time.Minute}
}

type Session struct {
	ID              string
	AccountID       string
	CreatedAt       time.Time
	AuthenticatedAt time.Time
	LastSeenAt      time.Time
	ExpiresAt       time.Time
}

type Store struct {
	pool   *pgxpool.Pool
	config Config
}

func NewStore(pool *pgxpool.Pool, config Config) *Store {
	return &Store{pool: pool, config: config}
}

const columns = `id, account_id, created_at, authenticated_at, last_seen_at, expires_at`

func hash(token string) []byte {
	digest := sha256.Sum256([]byte(token))
	return digest[:]
}

func scan(row pgx.Row) (Session, error) {
	var s Session
	err := row.Scan(&s.ID, &s.AccountID, &s.CreatedAt, &s.AuthenticatedAt, &s.LastSeenAt, &s.ExpiresAt)
	var pgErr *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) || errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return Session{}, ErrRejected
	}
	if err != nil {
		return Session{}, err
	}
	s.CreatedAt, s.AuthenticatedAt = s.CreatedAt.UTC(), s.AuthenticatedAt.UTC()
	s.LastSeenAt, s.ExpiresAt = s.LastSeenAt.UTC(), s.ExpiresAt.UTC()
	return s, nil
}

func (s *Store) Open(ctx context.Context, accountID string, now time.Time) (string, Session, error) {
	token := rand.Text()
	now = now.UTC()
	session, err := scan(s.pool.QueryRow(ctx, `insert into admin_sessions
		(account_id, token_hash, created_at, authenticated_at, last_seen_at, expires_at)
		values ($1, $2, $3, $3, $3, $4) returning `+columns, accountID, hash(token), now, now.Add(s.config.Max)))
	if err != nil {
		return "", Session{}, err
	}
	return token, session, nil
}

func (s *Store) Authenticate(ctx context.Context, token string, now time.Time) (Session, error) {
	if token == "" {
		return Session{}, ErrRejected
	}
	now = now.UTC()
	return scan(s.pool.QueryRow(ctx, `update admin_sessions set last_seen_at = $2
		where token_hash = $1 and revoked_at is null and expires_at > $2 and last_seen_at > $3
		returning `+columns, hash(token), now, now.Add(-s.config.Idle)))
}

func (s *Store) StepUp(ctx context.Context, id string, now time.Time) (Session, error) {
	now = now.UTC()
	return scan(s.pool.QueryRow(ctx, `update admin_sessions set authenticated_at = $2, last_seen_at = $2
		where id = $1 and revoked_at is null and expires_at > $2
		returning `+columns, id, now))
}

func (s *Store) Revoke(ctx context.Context, id string, now time.Time) error {
	_, err := s.pool.Exec(ctx, `update admin_sessions set revoked_at = coalesce(revoked_at, $2) where id = $1`, id, now.UTC())
	return err
}
