package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/app"
	"github.com/Alievs-corp/hypedomino/server/internal/auth"
	"github.com/Alievs-corp/hypedomino/server/internal/cluster"
	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/db"
	"github.com/Alievs-corp/hypedomino/server/internal/gateway"
	"github.com/Alievs-corp/hypedomino/server/internal/idtoken"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/secrand"
	"github.com/Alievs-corp/hypedomino/server/internal/session"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

type systemClock struct{}

func (systemClock) Now() time.Time                         { return time.Now() }
func (systemClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

func main() {
	config, err := app.ConfigFromEnv(os.LookupEnv)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	logger := app.NewLogger(os.Stdout, config.LogLevel)
	sessions := session.New(config.Session, systemClock{}, secrand.Source{})
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	a, release, err := build(ctx, config, logger, sessions)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	defer release()
	if err := a.Run(ctx); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func build(ctx context.Context, config app.Config, logger *slog.Logger, sessions *session.Manager) (*app.App, func(), error) {
	var client *redis.Client
	if config.Redis != "" {
		options, err := redis.ParseURL(config.Redis)
		if err != nil {
			return nil, nil, fmt.Errorf("HD_REDIS_URL: %w", err)
		}
		client = redis.NewClient(options)
	}
	a, err := node(config, logger, sessions, client)
	if err != nil {
		return nil, nil, err
	}
	if config.Database == "" {
		return a, func() {}, nil
	}
	pool, err := db.Open(ctx, config.Database)
	if err != nil {
		return nil, nil, fmt.Errorf("HD_DATABASE_URL: %w", err)
	}
	var nonces idtoken.NonceStore
	if client != nil {
		nonces = auth.NewRedisNonces(client, config.Cluster.KeyPrefix)
	}
	verifier := idtoken.New(idtoken.Options{Config: config.IDToken, Nonces: nonces})
	tokens, err := token.NewService(config.Token, token.NewStore(pool), time.Now)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("HD_TOKEN_KEYS: %w", err)
	}
	sessions.Authenticate(tokens)
	a.Handle("POST /auth/", auth.New(verifier, account.New(pool), tokens, sessions, logger))
	a.Handle("/profile/", profile.NewHandler(tokens, profile.NewStore(pool, config.Profile), logger, time.Now))
	catalog, err := loadCatalog(config.Cosmetics)
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	inventoryStore := inventory.NewStore(pool, catalog, config.Inventory)
	items := inventory.NewHandler(tokens, inventoryStore, logger, time.Now)
	a.Handle("/inventory", items)
	a.Handle("/inventory/", items)
	walletStore := wallet.NewStore(pool)
	a.Handle("/wallet", wallet.NewHandler(tokens, walletStore, logger))
	engine := reward.NewEngine(pool, walletStore, inventoryStore, nil, nil, config.Reward)
	offers := snapshot.New(pool, shop.Domain, shop.Parser(config.Shop), logger)
	if config.ShopCatalog != "" {
		if err := offers.SeedFile(config.ShopCatalog); err != nil {
			pool.Close()
			return nil, nil, fmt.Errorf("HD_SHOP_CATALOG: %w", err)
		}
	}
	go offers.Run(ctx, config.ConfigPoll)
	store := shop.NewHandler(tokens, shop.NewStore(pool, walletStore, inventoryStore, engine, offers, config.Shop), logger, time.Now)
	a.Handle("/shop", store)
	a.Handle("/shop/", store)
	return a, pool.Close, nil
}

func loadCatalog(path string) (*cosmetics.Catalog, error) {
	if path == "" {
		return cosmetics.Builtin(), nil
	}
	return cosmetics.Load(path)
}

func node(config app.Config, logger *slog.Logger, sessions *session.Manager, client *redis.Client) (*app.App, error) {
	if client == nil {
		return app.New(config, logger, sessions), nil
	}
	members, err := cluster.New(config.Cluster, client, logger)
	if err != nil {
		return nil, err
	}
	sessions.Route(members)
	var handler gateway.Handler = cluster.NewRelay(members, sessions, config.Relay)
	a := app.New(config, logger, handler)
	a.Join(members)
	return a, nil
}
