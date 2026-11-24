package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Buckets interface {
	Take(ctx context.Context, key string, rule Rule, now time.Time) (time.Duration, error)
}

func nextFree(theoretical, now time.Time, rule Rule) (time.Time, time.Duration) {
	interval := rule.interval()
	start := now
	if theoretical.After(now) {
		start = theoretical
	}
	if wait := start.Sub(now) - interval*time.Duration(rule.Burst-1); wait > 0 {
		return theoretical, wait
	}
	return start.Add(interval), 0
}

type Memory struct {
	mu      sync.Mutex
	arrival map[string]time.Time
	sweepAt time.Time
}

func NewMemory() *Memory {
	return &Memory{arrival: map[string]time.Time{}}
}

func (m *Memory) Take(_ context.Context, key string, rule Rule, now time.Time) (time.Duration, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sweep(now)
	next, wait := nextFree(m.arrival[key], now, rule)
	m.arrival[key] = next
	return wait, nil
}

func (m *Memory) sweep(now time.Time) {
	if now.Before(m.sweepAt) {
		return
	}
	m.sweepAt = now.Add(time.Minute)
	for key, next := range m.arrival {
		if !next.After(now) {
			delete(m.arrival, key)
		}
	}
}

func (m *Memory) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.arrival)
}

var takeScript = redis.NewScript(`
local now = tonumber(ARGV[1])
local interval = tonumber(ARGV[2])
local burst = tonumber(ARGV[3])
local theoretical = tonumber(redis.call('GET', KEYS[1]) or '0')
local start = math.max(theoretical, now)
local wait = start - now - interval * (burst - 1)
if wait > 0 then
  return wait
end
redis.call('SET', KEYS[1], start + interval, 'PX', start + interval - now)
return 0
`)

type Redis struct {
	client redis.Scripter
	prefix string
}

func NewRedis(client redis.Scripter, prefix string) *Redis {
	return &Redis{client: client, prefix: prefix + "rate:"}
}

func (r *Redis) Take(ctx context.Context, key string, rule Rule, now time.Time) (time.Duration, error) {
	wait, err := takeScript.Run(ctx, r.client, []string{r.prefix + key}, now.UnixMilli(), rule.interval().Milliseconds(), rule.Burst).Int64()
	return time.Duration(wait) * time.Millisecond, err
}
