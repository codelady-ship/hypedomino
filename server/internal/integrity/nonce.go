package integrity

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

const nonceBytes = 32

var (
	ErrInvalidBinding = errors.New("integrity: invalid nonce binding")
	ErrNonceExists    = errors.New("integrity: nonce already stored")
)

type Nonce struct {
	Value     string
	Binding   Binding
	ExpiresAt time.Time
}

type NonceStore interface {
	Put(ctx context.Context, nonce Nonce) error
	Take(ctx context.Context, value string) (Nonce, bool, error)
}

type NonceConfig struct {
	TTL time.Duration
}

type Nonces struct {
	config NonceConfig
	store  NonceStore
	clock  Clock
	random io.Reader
}

func NewNonces(config NonceConfig, store NonceStore, clock Clock, random io.Reader) (*Nonces, error) {
	switch {
	case config.TTL <= 0:
		return nil, errors.New("integrity: nonce TTL must be positive")
	case store == nil || clock == nil || random == nil:
		return nil, errors.New("integrity: nonce store, clock and random source are required")
	}
	return &Nonces{config: config, store: store, clock: clock, random: random}, nil
}

func (n *Nonces) Issue(ctx context.Context, binding Binding) (Nonce, error) {
	if binding.Account == "" || !binding.Purpose.valid() {
		return Nonce{}, ErrInvalidBinding
	}
	var raw [nonceBytes]byte
	if _, err := io.ReadFull(n.random, raw[:]); err != nil {
		return Nonce{}, fmt.Errorf("integrity: read nonce: %w", err)
	}
	nonce := Nonce{
		Value:     base64.RawURLEncoding.EncodeToString(raw[:]),
		Binding:   binding,
		ExpiresAt: n.clock.Now().Add(n.config.TTL),
	}
	if err := n.store.Put(ctx, nonce); err != nil {
		return Nonce{}, err
	}
	return nonce, nil
}

func (n *Nonces) redeem(ctx context.Context, value string, binding Binding) (Reason, error) {
	if value == "" {
		return ReasonNonceMissing, nil
	}
	nonce, found, err := n.store.Take(ctx, value)
	if err != nil {
		return 0, err
	}
	switch {
	case !found:
		return ReasonNonceUnknown, nil
	case !n.clock.Now().Before(nonce.ExpiresAt):
		return ReasonNonceExpired, nil
	case nonce.Binding != binding:
		return ReasonNonceBindingMismatch, nil
	}
	return 0, nil
}

type MemoryNonceStore struct {
	clock  Clock
	mu     sync.Mutex
	nonces map[string]Nonce
	queue  []Nonce
}

func NewMemoryNonceStore(clock Clock) *MemoryNonceStore {
	return &MemoryNonceStore{clock: clock, nonces: map[string]Nonce{}}
}

func (s *MemoryNonceStore) Put(_ context.Context, nonce Nonce) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dropExpired(s.clock.Now())
	if _, exists := s.nonces[nonce.Value]; exists {
		return ErrNonceExists
	}
	s.nonces[nonce.Value] = nonce
	s.queue = append(s.queue, nonce)
	return nil
}

func (s *MemoryNonceStore) Take(_ context.Context, value string) (Nonce, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	nonce, found := s.nonces[value]
	delete(s.nonces, value)
	return nonce, found, nil
}

func (s *MemoryNonceStore) dropExpired(now time.Time) {
	for len(s.queue) > 0 && !now.Before(s.queue[0].ExpiresAt) {
		oldest := s.queue[0]
		if stored, found := s.nonces[oldest.Value]; found && stored.ExpiresAt.Equal(oldest.ExpiresAt) {
			delete(s.nonces, oldest.Value)
		}
		s.queue = s.queue[1:]
	}
}
