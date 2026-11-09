package main

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Alievs-corp/hypedomino/server/internal/app"
	"github.com/Alievs-corp/hypedomino/server/internal/cluster"
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/matchqueue"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstart"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/secrand"
	"github.com/Alievs-corp/hypedomino/server/internal/session"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

var errNoHost = errors.New("matchmaking: no node can host a room")

type matchmakingDeps struct {
	pool       *pgxpool.Pool
	client     *redis.Client
	members    *cluster.Cluster
	sessions   *session.Manager
	rooms      *app.Rooms
	ratings    *rating.Store
	profiles   *profile.Store
	rankedOpen func() bool
}

func seeded() *rand.Rand {
	source := secrand.Source{}
	return rand.New(rand.NewPCG(source.Uint64(), source.Uint64()))
}

func startMatchmaking(ctx context.Context, config app.Config, logger *slog.Logger, deps matchmakingDeps) {
	configs := snapshot.New(deps.pool, matchmaking.Domain, matchmaking.Parse, logger)
	configs.Seed(matchmaking.DefaultConfig(), 0)
	go configs.Run(ctx, config.ConfigPoll)
	current := func() matchmaking.Config {
		value, _, _ := configs.Current()
		return value
	}
	prefix, node := config.Cluster.KeyPrefix, deps.members.NodeID()
	store := matchqueue.NewStore(deps.client, prefix)
	deps.sessions.Queue(matchqueue.NewGate(store, node, "", matchqueue.RatedEntry(deps.ratings), deps.rankedOpen, deps.members.Accepting, time.Now, logger))
	dispatcher := matchqueue.NewDispatcher(deps.client, prefix, node)
	host := matchstart.NewHost(deps.rooms, deps.sessions, dispatcher, store, matchstart.ProfileRoster(deps.pool, deps.profiles, time.Now), current, systemClock{}, seeded(), logger)
	go listen(ctx, dispatcher, host, deps.sessions, config.MatchmakingTick, logger)
	place := func(ctx context.Context, placement matchqueue.Placement) error {
		target, ok, err := deps.members.Least(ctx)
		switch {
		case err != nil:
			return err
		case !ok:
			return errNoHost
		}
		return dispatcher.Place(ctx, target.ID, placement)
	}
	leader := matchqueue.NewLeader(deps.client, prefix, "", node, config.MatchmakingTTL)
	matcher := matchqueue.NewMatcher(store, current, seeded())
	go matchqueue.Run(ctx, leader, matcher, config.MatchmakingTick, time.Now, place, func(err error) { logger.Error("matchmaking tick failed", "error", err) })
}

func listen(ctx context.Context, dispatcher *matchqueue.Dispatcher, host *matchstart.Host, sessions *session.Manager, retry time.Duration, logger *slog.Logger) {
	for ctx.Err() == nil {
		if err := dispatcher.Listen(ctx, host.Open, sessions.MatchFound); err != nil && ctx.Err() == nil {
			logger.Error("matchmaking listener failed", "error", err)
		}
		select {
		case <-time.After(retry):
		case <-ctx.Done():
		}
	}
}
