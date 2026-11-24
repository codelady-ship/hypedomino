package friends_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cluster/redistest"
	"github.com/Alievs-corp/hypedomino/server/internal/db/dbtest"
	"github.com/Alievs-corp/hypedomino/server/internal/friends"
)

func TestMain(m *testing.M) {
	code := m.Run()
	redistest.Stop()
	dbtest.Stop()
	os.Exit(code)
}

func TestPresenceShowsConnectedAccountsUntilItExpires(t *testing.T) {
	client := redistest.Client(t)
	presence := friends.NewPresence(client, redistest.Prefix(t), time.Second)
	ctx := context.Background()
	if err := presence.Publish(ctx, map[string]bool{"a": false, "b": true}); err != nil {
		t.Fatal(err)
	}
	states, err := presence.States(ctx, []string{"a", "b", "c"})
	if err != nil || strings.Join(states, ",") != "online,in_match,offline" {
		t.Fatalf("states %v %v", states, err)
	}
	time.Sleep(1500 * time.Millisecond)
	if states, _ := presence.States(ctx, []string{"a", "b"}); strings.Join(states, ",") != "offline,offline" {
		t.Fatalf("after expiry %v", states)
	}
}
