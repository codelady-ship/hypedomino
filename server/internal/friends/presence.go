package friends

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	PresenceOnline  = "online"
	PresenceInMatch = "in_match"
	PresenceOffline = "offline"
)

type Presence struct {
	client redis.UniversalClient
	prefix string
	ttl    time.Duration
}

func NewPresence(client redis.UniversalClient, prefix string, ttl time.Duration) *Presence {
	return &Presence{client: client, prefix: prefix + "presence:", ttl: ttl}
}

func (p *Presence) Publish(ctx context.Context, connected map[string]bool) error {
	if len(connected) == 0 {
		return nil
	}
	pipe := p.client.Pipeline()
	for account, inMatch := range connected {
		state := PresenceOnline
		if inMatch {
			state = PresenceInMatch
		}
		pipe.Set(ctx, p.prefix+account, state, p.ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (p *Presence) States(ctx context.Context, accounts []string) ([]string, error) {
	states := make([]string, len(accounts))
	if len(accounts) == 0 {
		return states, nil
	}
	keys := make([]string, len(accounts))
	for i, account := range accounts {
		keys[i] = p.prefix + account
	}
	values, err := p.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	for i, value := range values {
		states[i] = PresenceOffline
		if state, ok := value.(string); ok {
			states[i] = state
		}
	}
	return states, nil
}

func (p *Presence) Run(ctx context.Context, interval time.Duration, connected func() map[string]bool, logger *slog.Logger) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := p.Publish(ctx, connected()); err != nil && ctx.Err() == nil {
			logger.Warn("presence not published", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
