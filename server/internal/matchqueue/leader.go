package matchqueue

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Leader struct {
	client redis.UniversalClient
	key    string
	node   string
	ttl    time.Duration
}

func NewLeader(client redis.UniversalClient, prefix, region, node string, ttl time.Duration) *Leader {
	return &Leader{client: client, key: prefix + "matchmaking:leader:" + region, node: node, ttl: ttl}
}

var holdScript = redis.NewScript(`
local owner = redis.call("GET", KEYS[1])
if owner == false then
  redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2])
  return 1
end
if owner == ARGV[1] then
  redis.call("PEXPIRE", KEYS[1], ARGV[2])
  return 1
end
return 0
`)

var resignScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

func (l *Leader) Hold(ctx context.Context) (bool, error) {
	held, err := holdScript.Run(ctx, l.client, []string{l.key}, l.node, l.ttl.Milliseconds()).Int()
	return held == 1, err
}

func (l *Leader) Resign(ctx context.Context) error {
	return resignScript.Run(ctx, l.client, []string{l.key}, l.node).Err()
}

func Run(ctx context.Context, leader *Leader, matcher *Matcher, every time.Duration, now func() time.Time, place func(context.Context, Placement) error, failed func(error)) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	defer leader.Resign(context.WithoutCancel(ctx))
	for {
		if err := lead(ctx, leader, matcher, now(), place); err != nil && ctx.Err() == nil {
			failed(err)
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

func lead(ctx context.Context, leader *Leader, matcher *Matcher, now time.Time, place func(context.Context, Placement) error) error {
	held, err := leader.Hold(ctx)
	if err != nil || !held {
		return err
	}
	placements, err := matcher.Tick(ctx, now)
	for _, placement := range placements {
		if placeErr := place(ctx, placement); placeErr != nil {
			if returnErr := matcher.Return(ctx, placement); returnErr != nil {
				return returnErr
			}
			err = placeErr
		}
	}
	return err
}
