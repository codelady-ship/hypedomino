package matchqueue_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cluster/redistest"
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/matchqueue"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/penalties"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
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
	queueMetrics := matchqueue.NewMetrics()
	gate.Measure(queueMetrics)
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
	gate.Leave(ctx, "kamal")
	var written strings.Builder
	queueMetrics.WriteTo(&written)
	if !strings.Contains(written.String(), "\nhd_matchmaking_left_total 1\n") {
		t.Fatalf("a leave counted once, a second leave of no ticket not at all:\n%s", written.String())
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

func TestAPenaltyRecordRefusesABannedAccountAndDelaysTheSearch(t *testing.T) {
	client, prefix := redistest.Client(t), redistest.Prefix(t)
	ctx := context.Background()
	config := penalties.DefaultConfig()
	var record penalties.Record
	leave := rules.PresenceReport{Left: true, ByBot: true}
	record.MatchEnd(config, matchmaking.QueueRanked, "m1", rules.SeatX, leave, 0, now.Add(-time.Hour))
	records := func(context.Context, string) (penalties.Record, error) { return record, nil }
	plain := func(context.Context, string, rules.Mode, matchmaking.Queue, time.Time) (matchmaking.Entry, error) {
		return matchmaking.Entry{}, nil
	}
	gate := matchqueue.NewGate(matchqueue.NewStore(client, prefix), "node-a", "eu", matchqueue.PenalizedEntry(records, plain), func() bool { return true }, func() bool { return true }, func() time.Time { return now }, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ranked := &pb.JoinQueueIntent{Mode: pb.Mode_MODE_1V1, Queue: pb.QueueKind_QUEUE_KIND_RANKED}
	delay := time.Duration(config.QueueDelaySeconds) * time.Second
	if state := gate.Join(ctx, "kamal", "s1", ranked); state.GetStatus() != pb.QueueStatus_QUEUE_STATUS_SEARCHING || state.GetSearchStartsUnixMs() != uint64(now.Add(delay).UnixMilli()) {
		t.Fatalf("delayed join %v", state)
	}
	record.MatchEnd(config, matchmaking.QueueRanked, "m2", rules.SeatX, leave, 0, now.Add(-time.Minute))
	state := gate.Join(ctx, "kamal", "s1", &pb.JoinQueueIntent{Mode: pb.Mode_MODE_1V1, Queue: pb.QueueKind_QUEUE_KIND_CASUAL})
	until, _ := record.Banned(now)
	if state.GetRefusal() != pb.QueueRefusal_QUEUE_REFUSAL_MATCHMAKING_BAN || state.GetRefusedUntilUnixMs() != uint64(until.UnixMilli()) || state.GetBanReason() != pb.BanReason_BAN_REASON_AFK_LEAVING {
		t.Fatalf("banned join %v", state)
	}
}

func rankedBanUntil(ends time.Time, category sanction.Category) func(context.Context, string, time.Time) error {
	return func(_ context.Context, _ string, at time.Time) error {
		if !at.Before(ends) {
			return nil
		}
		return &sanction.RankedBanError{Sanction: sanction.Sanction{ID: 7, Kind: sanction.RankedBan, Category: category, EndsAt: &ends}}
	}
}

func TestARankedBanClosesTheRankedQueueOnlyUntilItEnds(t *testing.T) {
	client, prefix := redistest.Client(t), redistest.Prefix(t)
	ctx := context.Background()
	ends := now.Add(time.Hour)
	clock := now
	plain := func(context.Context, string, rules.Mode, matchmaking.Queue, time.Time) (matchmaking.Entry, error) {
		return matchmaking.Entry{}, nil
	}
	gate := matchqueue.NewGate(matchqueue.NewStore(client, prefix), "node-a", "eu", matchqueue.RankedBanned(rankedBanUntil(ends, sanction.Collusion), plain), func() bool { return true }, func() bool { return true }, func() time.Time { return clock }, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ranked := &pb.JoinQueueIntent{Mode: pb.Mode_MODE_1V1, Queue: pb.QueueKind_QUEUE_KIND_RANKED}
	casual := &pb.JoinQueueIntent{Mode: pb.Mode_MODE_1V1, Queue: pb.QueueKind_QUEUE_KIND_CASUAL}
	state := gate.Join(ctx, "kamal", "s1", ranked)
	if state.GetRefusal() != pb.QueueRefusal_QUEUE_REFUSAL_RANKED_BAN || state.GetRefusedUntilUnixMs() != uint64(ends.UnixMilli()) || state.GetBanReason() != pb.BanReason_BAN_REASON_COLLUSION {
		t.Fatalf("ranked during a ranked ban %v", state)
	}
	if state := gate.Join(ctx, "kamal", "s1", casual); state.GetStatus() != pb.QueueStatus_QUEUE_STATUS_SEARCHING {
		t.Fatalf("casual during a ranked ban %v", state)
	}
	gate.Leave(ctx, "kamal")
	clock = ends
	if state := gate.Join(ctx, "kamal", "s1", ranked); state.GetStatus() != pb.QueueStatus_QUEUE_STATUS_SEARCHING {
		t.Fatalf("ranked after the ranked ban ended %v", state)
	}
}

func TestRankedBannedChecksOnlyTheRankedQueue(t *testing.T) {
	ends := now.Add(time.Hour)
	plain := func(context.Context, string, rules.Mode, matchmaking.Queue, time.Time) (matchmaking.Entry, error) {
		return matchmaking.Entry{Mmr: 1200}, nil
	}
	entry := matchqueue.RankedBanned(rankedBanUntil(ends, sanction.Cheating), plain)
	cases := []struct {
		queue  matchmaking.Queue
		at     time.Time
		banned bool
	}{
		{matchmaking.QueueRanked, now, true},
		{matchmaking.QueueCasual, now, false},
		{matchmaking.QueueRanked, ends.Add(-time.Microsecond), true},
		{matchmaking.QueueRanked, ends, false},
		{matchmaking.QueueCasual, ends, false},
	}
	for _, c := range cases {
		got, err := entry(context.Background(), "kamal", rules.Mode1v1, c.queue, c.at)
		var ranked *sanction.RankedBanError
		if errors.As(err, &ranked) != c.banned || (!c.banned && (err != nil || got.Mmr != 1200)) {
			t.Fatalf("queue %v at %v: %+v, %v", c.queue, c.at, got, err)
		}
	}
}

func TestTheGateNamesTheRankedBanAndItsReason(t *testing.T) {
	ends := now.Add(time.Hour)
	plain := func(context.Context, string, rules.Mode, matchmaking.Queue, time.Time) (matchmaking.Entry, error) {
		return matchmaking.Entry{}, nil
	}
	for _, category := range sanction.Categories {
		gate := matchqueue.NewGate(nil, "node-a", "eu", matchqueue.RankedBanned(rankedBanUntil(ends, category), plain), func() bool { return true }, func() bool { return true }, func() time.Time { return now }, slog.New(slog.NewTextHandler(io.Discard, nil)))
		state := gate.Join(context.Background(), "kamal", "s1", &pb.JoinQueueIntent{Mode: pb.Mode_MODE_2V2, Queue: pb.QueueKind_QUEUE_KIND_RANKED})
		if state.GetStatus() != pb.QueueStatus_QUEUE_STATUS_REFUSED || state.GetRefusal() != pb.QueueRefusal_QUEUE_REFUSAL_RANKED_BAN || state.GetRefusedUntilUnixMs() != uint64(ends.UnixMilli()) || state.GetBanReason() != category.Reason() || category.Reason() == pb.BanReason_BAN_REASON_UNSPECIFIED {
			t.Fatalf("%s: %v", category, state)
		}
	}
}

func TestABanRefusalCarriesAnAppealGrantForItsSanction(t *testing.T) {
	ends := now.Add(time.Hour)
	plain := func(context.Context, string, rules.Mode, matchmaking.Queue, time.Time) (matchmaking.Entry, error) {
		return matchmaking.Entry{}, nil
	}
	penaltyBan := func(id int64) matchqueue.EntryFunc {
		return func(context.Context, string, rules.Mode, matchmaking.Queue, time.Time) (matchmaking.Entry, error) {
			return matchmaking.Entry{}, &matchqueue.BannedError{Until: ends, SanctionID: id}
		}
	}
	grant := func(account string, sanctionID int64) *pb.AppealGrant {
		return &pb.AppealGrant{SanctionId: uint64(sanctionID), Token: "appeal:" + account}
	}
	cases := []struct {
		name    string
		entry   matchqueue.EntryFunc
		grant   matchqueue.AppealFunc
		refusal pb.QueueRefusal
		want    uint64
	}{
		{"ranked ban", matchqueue.RankedBanned(rankedBanUntil(ends, sanction.Cheating), plain), grant, pb.QueueRefusal_QUEUE_REFUSAL_RANKED_BAN, 7},
		{"penalty ladder ban", penaltyBan(12), grant, pb.QueueRefusal_QUEUE_REFUSAL_MATCHMAKING_BAN, 12},
		{"penalty ban recorded before sanctions", penaltyBan(0), grant, pb.QueueRefusal_QUEUE_REFUSAL_MATCHMAKING_BAN, 0},
		{"no appeals wired", penaltyBan(12), nil, pb.QueueRefusal_QUEUE_REFUSAL_MATCHMAKING_BAN, 0},
	}
	for _, c := range cases {
		gate := matchqueue.NewGate(nil, "node-a", "eu", c.entry, func() bool { return true }, func() bool { return true }, func() time.Time { return now }, slog.New(slog.NewTextHandler(io.Discard, nil)))
		gate.GrantAppeals(c.grant)
		state := gate.Join(context.Background(), "kamal", "s1", &pb.JoinQueueIntent{Mode: pb.Mode_MODE_1V1, Queue: pb.QueueKind_QUEUE_KIND_RANKED})
		if state.GetRefusal() != c.refusal || state.GetAppeal().GetSanctionId() != c.want || (c.want != 0 && state.GetAppeal().GetToken() != "appeal:kamal") || (c.want == 0 && state.GetAppeal() != nil) {
			t.Fatalf("%s: %v", c.name, state)
		}
	}
}
