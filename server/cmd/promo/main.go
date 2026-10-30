package main

import (
	"context"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/db"
	"github.com/Alievs-corp/hypedomino/server/internal/inventory"
	"github.com/Alievs-corp/hypedomino/server/internal/promo"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

var errUsage = errors.New("usage: promo create|batch|approve|disable [flags]")

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Getenv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		if errors.Is(err, errUsage) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}

type command struct {
	flags    *flag.FlagSet
	id       string
	campaign string
	kind     string
	bundle   string
	start    string
	end      string
	cap      int
	code     string
	accounts string
	creator  string
	count    int
	approver string
}

func newCommand(name string) *command {
	c := &command{flags: flag.NewFlagSet("promo "+name, flag.ContinueOnError)}
	c.flags.StringVar(&c.id, "id", "", "promo id (promo.<slug>)")
	c.flags.StringVar(&c.campaign, "campaign", "", "campaign id (campaign.<slug>)")
	c.flags.StringVar(&c.kind, "kind", promo.MultiUse, "single_use, multi_use or targeted")
	c.flags.StringVar(&c.bundle, "bundle", "", "path of the bundle JSON")
	c.flags.StringVar(&c.start, "start", "", "window start, RFC 3339 UTC")
	c.flags.StringVar(&c.end, "end", "", "window end, RFC 3339 UTC")
	c.flags.IntVar(&c.cap, "cap", 0, "maximum redemptions of a multi_use code")
	c.flags.StringVar(&c.code, "code", "", "admin-chosen code, multi_use only")
	c.flags.StringVar(&c.accounts, "accounts", "", "comma-separated account ids of a targeted code")
	c.flags.StringVar(&c.creator, "creator", "", "admin creating the code")
	c.flags.IntVar(&c.count, "count", 0, "number of single_use codes of a batch")
	c.flags.StringVar(&c.approver, "approver", "", "admin approving the code")
	return c
}

func (c *command) definition() (promo.Definition, error) {
	d := promo.Definition{ID: c.id, Campaign: c.campaign, Kind: c.kind, Cap: c.cap, Code: c.code, Creator: c.creator}
	if c.accounts != "" {
		d.Accounts = strings.Split(c.accounts, ",")
	}
	raw, err := os.ReadFile(c.bundle)
	if err != nil {
		return d, err
	}
	if d.Bundle, err = reward.ParseBundle(raw); err != nil {
		return d, err
	}
	if d.Start, err = time.Parse(time.RFC3339, c.start); err != nil {
		return d, fmt.Errorf("-start: %w", err)
	}
	if d.End, err = time.Parse(time.RFC3339, c.end); err != nil {
		return d, fmt.Errorf("-end: %w", err)
	}
	return d, nil
}

func run(ctx context.Context, args []string, env func(string) string, out io.Writer) error {
	if len(args) == 0 {
		return errUsage
	}
	c := newCommand(args[0])
	if err := c.flags.Parse(args[1:]); err != nil {
		return err
	}
	store, release, err := open(ctx, env)
	if err != nil {
		return err
	}
	defer release()
	now := time.Now()
	switch args[0] {
	case "create":
		d, err := c.definition()
		if err != nil {
			return err
		}
		created, err := store.Create(ctx, d, now)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s %s code %s\n", created.Code.ID, created.Code.State, created.Plaintext)
	case "batch":
		d, err := c.definition()
		if err != nil {
			return err
		}
		d.Kind = promo.SingleUse
		created, err := store.Batch(ctx, d, c.count, now)
		if err != nil {
			return err
		}
		writer := csv.NewWriter(out)
		writer.Write([]string{"promo_id", "code", "state"})
		for _, code := range created {
			writer.Write([]string{code.Code.ID, code.Plaintext, code.Code.State})
		}
		writer.Flush()
		return writer.Error()
	case "approve":
		code, err := store.Approve(ctx, c.id, c.approver, now)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s %s approved by %s\n", code.ID, code.State, code.Approver)
	case "disable":
		code, err := store.Disable(ctx, c.id, now)
		if err != nil {
			return err
		}
		fmt.Fprintf(out, "%s %s\n", code.ID, code.State)
	default:
		return errUsage
	}
	return nil
}

func open(ctx context.Context, env func(string) string) (*promo.Store, func(), error) {
	url, key := env("HD_DATABASE_URL"), env("HD_PROMO_KEY")
	if url == "" || key == "" {
		return nil, nil, errors.New("HD_DATABASE_URL and HD_PROMO_KEY are required")
	}
	hasher, err := promo.NewHasher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("HD_PROMO_KEY: %w", err)
	}
	catalog := cosmetics.Builtin()
	if path := env("HD_COSMETICS_MANIFEST"); path != "" {
		if catalog, err = cosmetics.Load(path); err != nil {
			return nil, nil, fmt.Errorf("HD_COSMETICS_MANIFEST: %w", err)
		}
	}
	pool, err := db.Open(ctx, url)
	if err != nil {
		return nil, nil, err
	}
	engine := reward.NewEngine(pool, wallet.NewStore(pool), inventory.NewStore(pool, catalog, inventory.DefaultConfig()), nil, nil, reward.DefaultConfig())
	return promo.NewStore(pool, engine, catalog, hasher, promo.DefaultConfig()), pool.Close, nil
}
