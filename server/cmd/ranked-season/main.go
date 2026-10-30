package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/db"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
)

var errUsage = errors.New("usage: ranked-season create|extend|list [flags]")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Getenv, os.Stdout, time.Now); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if errors.Is(err, errUsage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, env func(string) string, out io.Writer, now func() time.Time) error {
	if len(args) == 0 {
		return errUsage
	}
	flags := flag.NewFlagSet("ranked-season "+args[0], flag.ContinueOnError)
	id := flags.String("id", "", "season id (season.<slug>)")
	nameKey := flags.String("name-key", "", "game string key of the season name")
	start := flags.String("start", "", "start, RFC 3339 UTC")
	end := flags.String("end", "", "end, RFC 3339 UTC")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	url := env("HD_DATABASE_URL")
	if url == "" {
		return errors.New("HD_DATABASE_URL is required")
	}
	pool, err := db.Open(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()
	store := rating.NewStore(pool, rating.NewWatcher(pool, slog.New(slog.NewTextHandler(io.Discard, nil))))
	switch args[0] {
	case "create":
		season := rating.Season{ID: *id, NameKey: *nameKey}
		if season.Start, err = time.Parse(time.RFC3339, *start); err != nil {
			return fmt.Errorf("-start: %w", err)
		}
		if season.End, err = time.Parse(time.RFC3339, *end); err != nil {
			return fmt.Errorf("-end: %w", err)
		}
		created, err := store.CreateSeason(ctx, season, now())
		if err != nil {
			return err
		}
		print(out, created, now())
	case "extend":
		at, err := time.Parse(time.RFC3339, *end)
		if err != nil {
			return fmt.Errorf("-end: %w", err)
		}
		season, err := store.ExtendSeason(ctx, *id, at)
		if err != nil {
			return err
		}
		print(out, season, now())
	case "list":
		seasons, err := store.Seasons(ctx)
		if err != nil {
			return err
		}
		for _, season := range seasons {
			print(out, season, now())
		}
	default:
		return errUsage
	}
	return nil
}

func print(out io.Writer, season rating.Season, now time.Time) {
	fmt.Fprintf(out, "%s %s %s %s %s\n", season.ID, season.State(now), season.Start.Format(time.RFC3339), season.End.Format(time.RFC3339), season.NameKey)
}
