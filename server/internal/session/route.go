package session

import (
	"context"
	"errors"

	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type RouteOwner interface {
	Elsewhere(ctx context.Context, token string) (bool, error)
}

func (m *Manager) routedElsewhere(ctx context.Context, token string) (string, bool) {
	owner, ok := m.router.(RouteOwner)
	if !ok {
		return "", false
	}
	key := token
	if m.auth != nil {
		_, id, err := m.auth.Authenticate(token)
		if err != nil || id == "" {
			return "", false
		}
		key = id
	}
	if m.seatedHere(key) {
		return "", false
	}
	elsewhere, err := owner.Elsewhere(ctx, key)
	return key, err == nil && elsewhere
}

func (m *Manager) seatedHere(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[key]
	return ok && s.room != nil
}

func (m *Manager) Place(id, account string, r *room.Room, seat rules.Seat) error {
	m.mu.Lock()
	m.ensure(id, account)
	m.mu.Unlock()
	return m.Seat(id, r, seat)
}

func (m *Manager) Renew(ctx context.Context) error {
	if m.router == nil {
		return nil
	}
	var failed error
	for _, token := range m.routedTokens() {
		failed = errors.Join(failed, m.router.Claim(ctx, token))
		if !m.stillRouted(token) {
			m.release(token)
		}
	}
	return failed
}

func (m *Manager) routedTokens() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var tokens []string
	for token, s := range m.sessions {
		if s.routed {
			tokens = append(tokens, token)
		}
	}
	return tokens
}

func (m *Manager) stillRouted(token string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[token]
	return ok && s.routed
}
