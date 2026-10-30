package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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
	"github.com/Alievs-corp/hypedomino/server/internal/promo"
	"github.com/Alievs-corp/hypedomino/server/internal/purchase"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
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
	ratingConfigs := rating.NewWatcher(pool, logger)
	go ratingConfigs.Run(ctx, config.ConfigPoll)
	ratings := rating.NewStore(pool, ratingConfigs)
	go ratings.RunApexRecalc(ctx, logger, time.Now)
	go ratings.RunSeasonEnd(ctx, logger, config.SeasonPoll, time.Now)
	shopStore := shop.NewStore(pool, walletStore, inventoryStore, engine, offers, config.Shop)
	store := shop.NewHandler(tokens, shopStore, logger, time.Now)
	a.Handle("/shop", store)
	a.Handle("/shop/", store)
	validators, google, apple, err := stores(config)
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	purchases := purchase.NewStore(pool, walletStore, engine, offers, validators, config.Purchase, logger)
	shopStore.Guard(purchases)
	receipts := purchase.NewHandler(tokens, purchases, google, apple, config.PushSecret, logger, time.Now)
	a.Handle("POST /shop/complete", receipts)
	a.Handle("POST /store/", receipts)
	promos, err := promos(pool, engine, catalog, config)
	if err != nil {
		pool.Close()
		return nil, nil, err
	}
	a.Handle("POST /promo/redeem", promo.NewHandler(tokens, promos, logger, time.Now))
	return a, pool.Close, nil
}

func stores(config app.Config) (map[purchase.Platform]purchase.Validator, purchase.Notifier, purchase.Notifier, error) {
	validators := map[purchase.Platform]purchase.Validator{}
	var google, apple purchase.Notifier
	if config.Play.PackageName != "" {
		play, err := purchase.NewGoogle(config.Play)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("HD_PLAY_SERVICE_ACCOUNT: %w", err)
		}
		validators[purchase.PlatformGoogle], google = play, play
	}
	if config.Apple.BundleID != "" {
		ios, err := purchase.NewApple(config.Apple)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("HD_APPLE_BUNDLE_ID: %w", err)
		}
		validators[purchase.PlatformApple], apple = ios, ios
	}
	return validators, google, apple, nil
}

func promos(pool *pgxpool.Pool, engine *reward.Engine, catalog *cosmetics.Catalog, config app.Config) (promo.Promos, error) {
	if config.PromoKey == "" {
		return nil, nil
	}
	hasher, err := promo.NewHasher(config.PromoKey)
	if err != nil {
		return nil, fmt.Errorf("HD_PROMO_KEY: %w", err)
	}
	return promo.NewStore(pool, engine, catalog, hasher, config.Promo), nil
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
