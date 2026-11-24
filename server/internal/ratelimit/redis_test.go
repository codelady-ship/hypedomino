package ratelimit

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cluster/redistest"
)

func TestMain(m *testing.M) {
	os.Exit(redistest.Main(m))
}

func TestRedisBucketsAreSharedByEveryNodeAndExpire(t *testing.T) {
	client := redistest.Client(t)
	ctx := context.Background()
	first, second := NewRedis(client, "hd:test184:"), NewRedis(client, "hd:test184:")
	rule := Rule{PerMinute: 60, Burst: 2}
	now := time.Now().Truncate(time.Millisecond)
	for i, step := range []struct {
		node *Redis
		at   time.Duration
		wait time.Duration
	}{
		{first, 0, 0}, {second, 0, 0}, {first, 0, time.Second}, {second, 400 * time.Millisecond, 600 * time.Millisecond}, {first, time.Second, 0},
	} {
		wait, err := step.node.Take(ctx, "spend:a", rule, now.Add(step.at))
		if err != nil || wait != step.wait {
			t.Fatalf("step %d: wait %v, want %v: %v", i, wait, step.wait, err)
		}
	}
	ttl, err := client.PTTL(ctx, "hd:test184:rate:spend:a").Result()
	if err != nil || ttl <= 0 || ttl > 2*time.Second {
		t.Fatalf("ttl %v: %v", ttl, err)
	}
}
