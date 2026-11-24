package attestkey

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/integrity"
)

var (
	ErrKeyExists    = integrity.ErrKeyExists
	ErrKeyNotFound  = integrity.ErrKeyNotFound
	ErrStaleCounter = integrity.ErrStaleCounter
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Save(ctx context.Context, account string, key integrity.AttestedKey, at time.Time) error {
	_, err := s.pool.Exec(ctx, `insert into app_attest_keys (key_id, account_id, public_key, counter, receipt, development, attested_at) values ($1, $2, $3, $4, $5, $6, $7)`,
		key.KeyID, account, key.PublicKey, int64(key.Counter), key.Receipt, key.Development, at)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrKeyExists
	}
	return err
}

func (s *Store) Key(ctx context.Context, account string, keyID []byte) (integrity.AttestedKey, error) {
	key := integrity.AttestedKey{KeyID: keyID}
	var counter int64
	err := s.pool.QueryRow(ctx, `select public_key, counter, receipt, development from app_attest_keys where key_id = $1 and account_id = $2`, keyID, account).
		Scan(&key.PublicKey, &counter, &key.Receipt, &key.Development)
	if errors.Is(err, pgx.ErrNoRows) {
		return integrity.AttestedKey{}, ErrKeyNotFound
	}
	key.Counter = uint32(counter)
	return key, err
}

func (s *Store) Advance(ctx context.Context, account string, keyID []byte, counter uint32) error {
	tag, err := s.pool.Exec(ctx, `update app_attest_keys set counter = $3 where key_id = $1 and account_id = $2 and counter < $3`, keyID, account, int64(counter))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrStaleCounter
	}
	return nil
}
