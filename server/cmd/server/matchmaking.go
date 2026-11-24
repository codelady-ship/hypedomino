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
	"github.com/Alievs-corp/hypedomino/server/internal/custom"
	"github.com/Alievs-corp/hypedomino/server/internal/friends"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/lobbynet"
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/matchqueue"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstart"
	"github.com/Alievs-corp/hypedomino/server/internal/matchstore"
	"github.com/Alievs-corp/hypedomino/server/internal/panics"
	"github.com/Alievs-corp/hypedomino/server/internal/penalties"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/room"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
	"github.com/Alievs-corp/hypedomino/server/internal/secrand"
	"github.com/Alievs-corp/hypedomino/server/internal/session"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

var errNoHost = errors.New("matchmaking: no node can host a room")

type matchmakingDeps struct {
	metrics    *matchqueue.Metrics
	await      func(func() bool)
	pool       *pgxpool.Pool
	client     *redis.Client
	members    *cluster.Cluster
	sessions   *session.Manager
	rooms      *app.Rooms
	ratings    *rating.Store
	profiles   *profile.Store
	inventory  *inventory.Store
	penalties  *penalties.Store
	friends    *friends.Store
	emoji      room.EmojiLimit
	appeal     matchqueue.AppealFunc
	rankedOpen func() bool
}

func seeded() *rand.Rand {
	source := secrand.Source{}
	return rand.New(rand.NewPCG(source.Uint64(), source.Uint64()))
}

func startMatchmaking(ctx context.Context, config app.Config, logger *slog.Logger, deps matchmakingDeps) {
	configs := snapshot.New(deps.pool, matchmaking.Domain, matchmaking.Parse, logger)
	configs.Seed(matchmaking.DefaultConfig(), 0)
	watchConfig(ctx, deps.await, "matchmaking", config.ConfigPoll, configs)
	current := func() matchmaking.Config {
		value, _, _ := configs.Current()
		return value
	}
	prefix, node := config.Cluster.KeyPrefix, deps.members.NodeID()
	store := matchqueue.NewStore(deps.client, prefix)
	penaltyRecords := deps.penalties
	rankedBan := sanction.RankedBanCheck(deps.pool)
	gate := matchqueue.NewGate(store, node, "", matchqueue.PenalizedEntry(penaltyRecords.Record, matchqueue.RankedBanned(rankedBan, matchqueue.RatedEntry(deps.ratings))), deps.rankedOpen, deps.members.Accepting, time.Now, logger)
	gate.Measure(deps.metrics)
	gate.GrantAppeals(deps.appeal)
	deps.sessions.Queue(gate)
	dispatcher := matchqueue.NewDispatcher(deps.client, prefix, node)
	muted := matchstart.Muted(deps.pool, time.Now)
	roster := matchstart.ProfileRoster(deps.profiles, deps.inventory.Freeze, deps.ratings.Label, deps.ratings.Showcase, muted)
	host := matchstart.NewHost(deps.rooms, deps.sessions, dispatcher, store, matchstore.New(deps.pool), penaltyRecords, roster, current, systemClock{}, seeded(), logger)
	lobby := custom.NewLobby(secrand.Source{}, matchstart.LobbyOpener(deps.rooms, roster, current, systemClock{}, secrand.Source{}))
	lobby.Screen(deps.friends)
	lobby.LimitEmoji(deps.emoji)
	lobbies := lobbynet.New(deps.client, deps.members, prefix, config.Relay.DialTimeout)
	lobby.Route(lobbies)
	deps.sessions.Host(lobby, matchstart.LobbyMember(deps.inventory.Freeze, muted), deps.friends)
	deps.sessions.Reach(lobbies)
	deps.sessions.Restrict(rankedBan)
	go panics.Supervise(ctx, "lobby listener", config.MatchmakingTick, func(ctx context.Context, retry time.Duration) {
		listenLobbies(ctx, lobbies, deps.sessions, retry, logger)
	})
	go panics.Supervise(ctx, "route renewal", config.Cluster.RouteTTL/2, func(ctx context.Context, every time.Duration) {
		renewRoutes(ctx, every, logger, lobby.Renew, deps.sessions.Renew)
	})
	go panics.Supervise(ctx, "matchmaking listener", config.MatchmakingTick, func(ctx context.Context, retry time.Duration) {
		listen(ctx, dispatcher, host, deps.sessions, retry, logger)
	})
	place := func(ctx context.Context, placement matchqueue.Placement) error {
		target, ok, err := deps.members.Least(ctx)
		switch {
		case err != nil:
			return err
		case !ok:
			return errNoHost
		}
		if err := dispatcher.Place(ctx, target.ID, placement); err != nil {
			return err
		}
		deps.metrics.Placed(placement, time.Now())
		return nil
	}
	leader := matchqueue.NewLeader(deps.client, prefix, "", node, config.MatchmakingTTL)
	matcher := matchqueue.NewMatcher(store, current, seeded())
	go panics.Supervise(ctx, "matchmaking tick", config.MatchmakingTick, func(ctx context.Context, tick time.Duration) {
		matchqueue.Run(ctx, leader, matcher, tick, time.Now, place, func(err error) { logger.Error("matchmaking tick failed", "error", err) })
	})
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

func listenLobbies(ctx context.Context, lobbies *lobbynet.Net, sessions *session.Manager, retry time.Duration, logger *slog.Logger) {
	for ctx.Err() == nil {
		if err := lobbies.Listen(ctx, sessions.Adopt, sessions.Invited); err != nil && ctx.Err() == nil {
			logger.Error("lobby listener failed", "error", err)
		}
		select {
		case <-time.After(retry):
		case <-ctx.Done():
		}
	}
}

func renewRoutes(ctx context.Context, every time.Duration, logger *slog.Logger, renewals ...func(context.Context) error) {
	for {
		select {
		case <-time.After(every):
		case <-ctx.Done():
			return
		}
		for _, renew := range renewals {
			if err := renew(ctx); err != nil && ctx.Err() == nil {
				logger.Error("route renewal failed", "error", err)
			}
		}
	}
}

func penaltyStore(ctx context.Context, config app.Config, pool *pgxpool.Pool, logger *slog.Logger, await func(func() bool)) *penalties.Store {
	configs := snapshot.New(pool, penalties.Domain, penalties.Parse, logger)
	configs.Seed(penalties.DefaultConfig(), 0)
	watchConfig(ctx, await, "penalties", config.ConfigPoll, configs)
	return penalties.NewStore(pool, func() (penalties.Config, int) {
		value, version, _ := configs.Current()
		return value, int(version)
	})
}

type configWatcher interface {
	Run(ctx context.Context, every time.Duration)
	Synced() bool
}

func watchConfig(ctx context.Context, await func(func() bool), name string, every time.Duration, watcher configWatcher) {
	await(watcher.Synced)
	go panics.Supervise(ctx, "config "+name, every, watcher.Run)
}
