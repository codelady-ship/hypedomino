package integrity

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisNonceStore struct {
	client redis.UniversalClient
	prefix string
	clock  Clock
}

type storedNonce struct {
	Account   string  `json:"a"`
	Session   string  `json:"s"`
	Purpose   Purpose `json:"p"`
	ExpiresAt int64   `json:"e"`
}

func NewRedisNonceStore(client redis.UniversalClient, prefix string, clock Clock) *RedisNonceStore {
	return &RedisNonceStore{client: client, prefix: prefix + "integrity-nonce:", clock: clock}
}

func (s *RedisNonceStore) Put(ctx context.Context, nonce Nonce) error {
	ttl := nonce.ExpiresAt.Sub(s.clock.Now())
	if ttl <= 0 {
		return nil
	}
	value, err := json.Marshal(storedNonce{Account: nonce.Binding.Account, Session: nonce.Binding.Session, Purpose: nonce.Binding.Purpose, ExpiresAt: nonce.ExpiresAt.UnixMilli()})
	if err != nil {
		return err
	}
	stored, err := s.client.SetNX(ctx, s.prefix+nonce.Value, value, ttl).Result()
	if err != nil {
		return err
	}
	if !stored {
		return ErrNonceExists
	}
	return nil
}

func (s *RedisNonceStore) Take(ctx context.Context, value string) (Nonce, bool, error) {
	raw, err := s.client.GetDel(ctx, s.prefix+value).Bytes()
	if errors.Is(err, redis.Nil) {
		return Nonce{}, false, nil
	}
	if err != nil {
		return Nonce{}, false, err
	}
	var stored storedNonce
	if err := json.Unmarshal(raw, &stored); err != nil {
		return Nonce{}, false, err
	}
	binding := Binding{Account: stored.Account, Session: stored.Session, Purpose: stored.Purpose}
	return Nonce{Value: value, Binding: binding, ExpiresAt: time.UnixMilli(stored.ExpiresAt)}, true, nil
}
