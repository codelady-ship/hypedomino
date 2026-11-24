package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/db"
	"github.com/Alievs-corp/hypedomino/server/internal/purchase"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func main() {
	flags := flag.NewFlagSet("coin-conversion", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: coin-conversion")
		fmt.Fprintln(os.Stderr, "converts every granted coin pack purchase to the crystals of the same tier, once (docs/economy.md 7.6)")
	}
	flags.Parse(os.Args[1:])
	url, ok := os.LookupEnv("HD_DATABASE_URL")
	if !ok || url == "" {
		fmt.Fprintln(os.Stderr, "HD_DATABASE_URL is not set")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	if err := run(ctx, url, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, url string, out io.Writer) error {
	pool, err := db.Open(ctx, url)
	if err != nil {
		return err
	}
	defer pool.Close()
	version, raw, err := snapshot.Latest(ctx, pool, shop.Domain)
	if err != nil {
		return fmt.Errorf("published shop catalog: %w", err)
	}
	catalog, err := shop.Parse(raw, shop.DefaultConfig())
	if err != nil {
		return fmt.Errorf("shop catalog version %d: %w", version, err)
	}
	converted, err := purchase.ConvertCoinPacks(ctx, pool, wallet.NewStore(pool), catalog, time.Now())
	for _, c := range converted {
		fmt.Fprintf(out, "%s %s:%s %s coins -%d crystals +%d\n", c.Account, c.Platform, c.TransactionID, c.Product, c.CoinsDebited, c.Crystals)
	}
	fmt.Fprintf(out, "%d purchase(s) converted\n", len(converted))
	return err
}
