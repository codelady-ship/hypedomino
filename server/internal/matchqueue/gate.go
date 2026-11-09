package matchqueue

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/penalties"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rules"
)

type EntryFunc func(ctx context.Context, account string, mode rules.Mode, queue matchmaking.Queue, now time.Time) (matchmaking.Entry, error)

type Gate struct {
	store      *Store
	node       string
	region     string
	entry      EntryFunc
	rankedOpen func() bool
	accepting  func() bool
	now        func() time.Time
	logger     *slog.Logger
}

func NewGate(store *Store, node, region string, entry EntryFunc, rankedOpen, accepting func() bool, now func() time.Time, logger *slog.Logger) *Gate {
	return &Gate{store: store, node: node, region: region, entry: entry, rankedOpen: rankedOpen, accepting: accepting, now: now, logger: logger}
}

func refused(intent *pb.JoinQueueIntent, refusal pb.QueueRefusal) *pb.QueueState {
	return &pb.QueueState{Status: pb.QueueStatus_QUEUE_STATUS_REFUSED, Mode: intent.GetMode(), Queue: intent.GetQueue(), Refusal: refusal}
}

func modeOf(mode pb.Mode) (rules.Mode, bool) {
	switch mode {
	case pb.Mode_MODE_1V1:
		return rules.Mode1v1, true
	case pb.Mode_MODE_2V2:
		return rules.Mode2v2, true
	}
	return 0, false
}

func queueOf(queue pb.QueueKind) (matchmaking.Queue, bool) {
	switch queue {
	case pb.QueueKind_QUEUE_KIND_RANKED:
		return matchmaking.QueueRanked, true
	case pb.QueueKind_QUEUE_KIND_CASUAL:
		return matchmaking.QueueCasual, true
	}
	return 0, false
}

func (g *Gate) Join(ctx context.Context, account, session string, intent *pb.JoinQueueIntent) *pb.QueueState {
	mode, modeOK := modeOf(intent.GetMode())
	queue, queueOK := queueOf(intent.GetQueue())
	switch {
	case !modeOK || !queueOK || !g.accepting():
		return refused(intent, pb.QueueRefusal_QUEUE_REFUSAL_UNAVAILABLE)
	case queue == matchmaking.QueueRanked && !g.rankedOpen():
		return refused(intent, pb.QueueRefusal_QUEUE_REFUSAL_RANKED_CLOSED)
	}
	now := g.now()
	entry, err := g.entry(ctx, account, mode, queue, now)
	var banned *BannedError
	if errors.As(err, &banned) {
		state := refused(intent, pb.QueueRefusal_QUEUE_REFUSAL_MATCHMAKING_BAN)
		if !banned.Until.IsZero() {
			state.RefusedUntilUnixMs = uint64(banned.Until.UnixMilli())
		}
		return state
	}
	if err != nil {
		g.logger.Error("queue entry failed", "account", account, "error", err)
		return refused(intent, pb.QueueRefusal_QUEUE_REFUSAL_UNAVAILABLE)
	}
	entry.Account, entry.Mode, entry.Queue, entry.Region, entry.EnqueuedAt = account, mode, queue, g.region, now
	if entry.SearchFrom.Before(now) {
		entry.SearchFrom = now
	}
	if err := g.store.Join(ctx, Ticket{Entry: entry, Node: g.node, Session: session}); err != nil {
		g.logger.Error("queue join failed", "account", account, "error", err)
		return refused(intent, pb.QueueRefusal_QUEUE_REFUSAL_UNAVAILABLE)
	}
	return &pb.QueueState{Status: pb.QueueStatus_QUEUE_STATUS_SEARCHING, Mode: intent.GetMode(), Queue: intent.GetQueue(), JoinedUnixMs: uint64(now.UnixMilli()), SearchStartsUnixMs: uint64(entry.SearchFrom.UnixMilli())}
}

func (g *Gate) Leave(ctx context.Context, account string) *pb.QueueState {
	if _, err := g.store.Leave(ctx, account); err != nil {
		g.logger.Error("queue leave failed", "account", account, "error", err)
	}
	return &pb.QueueState{Status: pb.QueueStatus_QUEUE_STATUS_LEFT}
}

func RatedEntry(ratings *rating.Store) EntryFunc {
	return func(ctx context.Context, account string, mode rules.Mode, queue matchmaking.Queue, now time.Time) (matchmaking.Entry, error) {
		rated, err := ratings.Queue(ctx, account, mode, now)
		if err != nil {
			return matchmaking.Entry{}, err
		}
		entry := matchmaking.Entry{Mmr: rated.Mmr, Rank: rated.Rank, Placed: rated.Placed}
		if queue == matchmaking.QueueRanked {
			entry.Avoid = rated.Avoid
		}
		return entry, nil
	}
}

type BannedError struct {
	Until time.Time
}

func (e *BannedError) Error() string {
	return "matchqueue: matchmaking ban"
}

func PenalizedEntry(records func(ctx context.Context, account string) (penalties.Record, error), next EntryFunc) EntryFunc {
	return func(ctx context.Context, account string, mode rules.Mode, queue matchmaking.Queue, now time.Time) (matchmaking.Entry, error) {
		record, err := records(ctx, account)
		if err != nil {
			return matchmaking.Entry{}, err
		}
		if until, banned := record.Banned(now); banned {
			return matchmaking.Entry{}, &BannedError{Until: until}
		}
		entry, err := next(ctx, account, mode, queue, now)
		if err != nil {
			return matchmaking.Entry{}, err
		}
		if delayed := now.Add(record.QueueDelay(queue)); delayed.After(entry.SearchFrom) {
			entry.SearchFrom = delayed
		}
		return entry, nil
	}
}
