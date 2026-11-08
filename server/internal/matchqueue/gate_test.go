package matchqueue_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cluster/redistest"
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/matchqueue"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

func TestTheGateQueuesAnAccountUnlessRankedIsClosedOrTheNodeDrains(t *testing.T) {
	client, prefix := redistest.Client(t), redistest.Prefix(t)
	ctx := context.Background()
	store := matchqueue.NewStore(client, prefix)
	rankedOpen, accepting := true, true
	entry := func(ctx context.Context, account string, mode rules.Mode, queue matchmaking.Queue, at time.Time) (matchmaking.Entry, error) {
		return matchmaking.Entry{Mmr: 1300, SearchFrom: at.Add(30 * time.Second)}, nil
	}
	gate := matchqueue.NewGate(store, "node-a", "eu", entry, func() bool { return rankedOpen }, func() bool { return accepting }, func() time.Time { return now }, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ranked := &pb.JoinQueueIntent{Mode: pb.Mode_MODE_2V2, Queue: pb.QueueKind_QUEUE_KIND_RANKED}

	state := gate.Join(ctx, "kamal", "session-1", ranked)
	if state.GetStatus() != pb.QueueStatus_QUEUE_STATUS_SEARCHING || state.GetJoinedUnixMs() != uint64(now.UnixMilli()) || state.GetSearchStartsUnixMs() != uint64(now.Add(30*time.Second).UnixMilli()) {
		t.Fatalf("join %v", state)
	}
	tickets, _, err := store.Tickets(ctx)
	if got := tickets["kamal"]; err != nil || got.Node != "node-a" || got.Session != "session-1" || got.Entry.Mode != rules.Mode2v2 || got.Entry.Queue != matchmaking.QueueRanked || got.Entry.Region != "eu" || got.Entry.Mmr != 1300 {
		t.Fatalf("ticket %+v, %v", got, err)
	}
	if state := gate.Leave(ctx, "kamal"); state.GetStatus() != pb.QueueStatus_QUEUE_STATUS_LEFT {
		t.Fatalf("leave %v", state)
	}

	rankedOpen = false
	if state := gate.Join(ctx, "kamal", "session-1", ranked); state.GetRefusal() != pb.QueueRefusal_QUEUE_REFUSAL_RANKED_CLOSED {
		t.Fatalf("ranked closed %v", state)
	}
	if state := gate.Join(ctx, "kamal", "session-1", &pb.JoinQueueIntent{Mode: pb.Mode_MODE_1V1, Queue: pb.QueueKind_QUEUE_KIND_CASUAL}); state.GetStatus() != pb.QueueStatus_QUEUE_STATUS_SEARCHING {
		t.Fatalf("casual while ranked is closed %v", state)
	}
	accepting = false
	if state := gate.Join(ctx, "ayla", "session-2", &pb.JoinQueueIntent{Mode: pb.Mode_MODE_1V1, Queue: pb.QueueKind_QUEUE_KIND_CASUAL}); state.GetRefusal() != pb.QueueRefusal_QUEUE_REFUSAL_UNAVAILABLE {
		t.Fatalf("draining %v", state)
	}
	accepting = true
	if state := gate.Join(ctx, "ayla", "session-2", &pb.JoinQueueIntent{Queue: pb.QueueKind_QUEUE_KIND_CASUAL}); state.GetRefusal() != pb.QueueRefusal_QUEUE_REFUSAL_UNAVAILABLE {
		t.Fatalf("no mode %v", state)
	}
}
