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
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

func main() {
	flags := flag.NewFlagSet("wallet-audit", flag.ExitOnError)
	fix := flags.Bool("fix", false, "set every mismatching stored balance to the sum of its ledger entries")
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: wallet-audit [-fix]")
		flags.PrintDefaults()
	}
	flags.Parse(os.Args[1:])
	url, ok := os.LookupEnv("HD_DATABASE_URL")
	if !ok || url == "" {
		fmt.Fprintln(os.Stderr, "HD_DATABASE_URL is not set")
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	mismatches, err := run(ctx, url, *fix, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if mismatches > 0 && !*fix {
		os.Exit(3)
	}
}

func run(ctx context.Context, url string, fix bool, out io.Writer) (int, error) {
	pool, err := db.Open(ctx, url)
	if err != nil {
		return 0, err
	}
	defer pool.Close()
	store := wallet.NewStore(pool)
	mismatches, err := store.Audit(ctx)
	if err != nil {
		return 0, err
	}
	for _, m := range mismatches {
		fmt.Fprintf(out, "%s %s stored %d ledger %d\n", m.Account, m.Currency, m.Stored, m.Computed)
		if !fix {
			continue
		}
		if _, _, err := store.Reconcile(ctx, m.Account, m.Currency, time.Now()); err != nil {
			return len(mismatches), err
		}
		fmt.Fprintf(out, "%s %s reconciled\n", m.Account, m.Currency)
	}
	fmt.Fprintf(out, "%d wallet(s) mismatched\n", len(mismatches))
	return len(mismatches), nil
}
