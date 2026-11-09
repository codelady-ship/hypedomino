package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/dailylogin"
	"github.com/Alievs-corp/hypedomino/server/internal/db"
	"github.com/Alievs-corp/hypedomino/server/internal/gamestrings"
	"github.com/Alievs-corp/hypedomino/server/internal/matchmaking"
	"github.com/Alievs-corp/hypedomino/server/internal/matchreward"
	"github.com/Alievs-corp/hypedomino/server/internal/medal"
	"github.com/Alievs-corp/hypedomino/server/internal/mission"
	"github.com/Alievs-corp/hypedomino/server/internal/pass"
	"github.com/Alievs-corp/hypedomino/server/internal/penalties"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/rulesconfig"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

func main() {
	flags := flag.NewFlagSet("config-publish", flag.ExitOnError)
	domain := flags.String("domain", shop.Domain, "config domain to publish")
	file := flags.String("file", "", "JSON file with the new snapshot")
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: config-publish -domain "+strings.Join(domains, "|")+" -file snapshot.json")
		flags.PrintDefaults()
	}
	flags.Parse(os.Args[1:])
	url, ok := os.LookupEnv("HD_DATABASE_URL")
	if !ok || url == "" || *file == "" {
		fmt.Fprintln(os.Stderr, "HD_DATABASE_URL and -file are required")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	if err := run(ctx, url, *domain, *file, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var domains = []string{cosmetics.Domain, dailylogin.Domain, gamestrings.Domain, matchmaking.Domain, matchreward.Domain, medal.Domain, mission.Domain, pass.Domain, penalties.Domain, rating.Domain, rulesconfig.Domain, shop.Domain}

var errDomain = errors.New("unknown config domain")

func run(ctx context.Context, url, domain, file string, out io.Writer) error {
	raw, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	if err := validate(domain, raw); err != nil {
		return err
	}
	pool, err := db.Open(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := checkAgainstPublished(ctx, pool, domain, raw); err != nil {
		return err
	}
	version, err := snapshot.Publish(ctx, pool, domain, raw)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s version %d\n", domain, version)
	return nil
}

func validate(domain string, raw []byte) error {
	switch domain {
	case dailylogin.Domain:
		_, err := dailylogin.Parse(raw)
		return err
	case penalties.Domain:
		_, err := penalties.Parse(raw)
		return err
	case matchmaking.Domain:
		_, err := matchmaking.Parse(raw)
		return err
	case matchreward.Domain:
		_, err := matchreward.Parse(raw)
		return err
	case medal.Domain:
		_, err := medal.Parse(raw)
		return err
	case mission.Domain:
		_, err := mission.Parse(raw)
		return err
	case pass.Domain:
		_, err := pass.Parse(raw)
		return err
	case shop.Domain:
		_, err := shop.Parse(raw, shop.DefaultConfig())
		return err
	case rulesconfig.Domain:
		_, err := rulesconfig.Parse(raw)
		return err
	case cosmetics.Domain:
		_, err := cosmetics.ParseManifest(raw)
		return err
	case gamestrings.Domain:
		_, err := gamestrings.ParseLocalization(raw)
		return err
	case rating.Domain:
		_, err := rating.Parse(raw)
		return err
	}
	return fmt.Errorf("%w: %s", errDomain, domain)
}

func checkAgainstPublished(ctx context.Context, pool *pgxpool.Pool, domain string, raw []byte) error {
	switch domain {
	case cosmetics.Domain:
		return followsPublished(ctx, pool, raw)
	case pass.Domain:
		return passFollowsPublished(ctx, pool, raw)
	case mission.Domain:
		return missionSourcesArePublished(ctx, pool, raw)
	case shop.Domain:
		return chestPoolsArePublished(ctx, pool, raw)
	case gamestrings.Domain:
		localization, err := gamestrings.ParseLocalization(raw)
		if err != nil {
			return err
		}
		_, err = gamestrings.NewStore(pool).Resolve(ctx, localization)
		return err
	}
	return nil
}

func chestPoolsArePublished(ctx context.Context, pool *pgxpool.Pool, raw []byte) error {
	catalog, err := shop.Parse(raw, shop.DefaultConfig())
	if err != nil {
		return err
	}
	items, err := cosmetics.Published(ctx, pool)
	if err != nil {
		return err
	}
	return catalog.CheckChestPools(items)
}

func missionSourcesArePublished(ctx context.Context, pool *pgxpool.Pool, raw []byte) error {
	catalog, err := mission.Parse(raw)
	if err != nil {
		return err
	}
	medals := medal.Default()
	_, published, err := snapshot.Latest(ctx, pool, medal.Domain)
	switch {
	case err == nil:
		if medals, err = medal.Parse(published); err != nil {
			return err
		}
	case !errors.Is(err, snapshot.ErrNone):
		return err
	}
	items, err := cosmetics.Published(ctx, pool)
	if err != nil {
		return err
	}
	return catalog.CheckSources(func(id string, source cosmetics.Source) bool {
		found, ok := medals.Medal(id)
		return ok && found.Lists(source)
	}, func(id string, source cosmetics.Source) bool {
		item, ok := items.Item(id)
		return ok && item.Grantable(source)
	})
}

func followsPublished(ctx context.Context, pool *pgxpool.Pool, raw []byte) error {
	_, published, err := snapshot.Latest(ctx, pool, cosmetics.Domain)
	if errors.Is(err, snapshot.ErrNone) {
		return nil
	}
	if err != nil {
		return err
	}
	previous, err := cosmetics.ParseManifest(published)
	if err != nil {
		return err
	}
	next, err := cosmetics.ParseManifest(raw)
	if err != nil {
		return err
	}
	return cosmetics.CheckSuccessor(previous, next)
}

func passFollowsPublished(ctx context.Context, pool *pgxpool.Pool, raw []byte) error {
	_, published, err := snapshot.Latest(ctx, pool, pass.Domain)
	if errors.Is(err, snapshot.ErrNone) {
		return nil
	}
	if err != nil {
		return err
	}
	previous, err := pass.Parse(published)
	if err != nil {
		return err
	}
	next, err := pass.Parse(raw)
	if err != nil {
		return err
	}
	return pass.CheckFrozen(previous, next, time.Now())
}
