package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/Alievs-corp/hypedomino/server/internal/db"
	"github.com/Alievs-corp/hypedomino/server/internal/rating"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
)

func main() {
	flags := flag.NewFlagSet("config-publish", flag.ExitOnError)
	domain := flags.String("domain", shop.Domain, "config domain to publish")
	file := flags.String("file", "", "JSON file with the new snapshot")
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: config-publish -domain shop|rating -file shop.json")
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
	version, err := snapshot.Publish(ctx, pool, domain, raw)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "%s version %d\n", domain, version)
	return nil
}

func validate(domain string, raw []byte) error {
	switch domain {
	case shop.Domain:
		_, err := shop.Parse(raw, shop.DefaultConfig())
		return err
	case rating.Domain:
		_, err := rating.Parse(raw)
		return err
	}
	return fmt.Errorf("%w: %s", errDomain, domain)
}
