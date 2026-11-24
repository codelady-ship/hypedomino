package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/contentpublish"
	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
)

const (
	urlLifetime  = 15 * time.Minute
	defaultGrace = 24 * time.Hour
)

func main() {
	flags := flag.NewFlagSet("content-gc", flag.ExitOnError)
	environment := flags.String("environment", "", "dev, staging or production")
	platform := flags.String("platform", "Android", "Android or iOS")
	active := flags.String("active-catalog-version", os.Getenv("HD_CATALOG_VERSION"), "catalog version the server announces now, defaults to HD_CATALOG_VERSION")
	minimum := flags.String("min-app-version", os.Getenv("HD_MIN_APP_VERSION"), "oldest supported app version, defaults to HD_MIN_APP_VERSION")
	grace := flags.Duration("grace", defaultGrace, "an unreferenced bundle younger than this is kept")
	confirmed := flags.Bool("delete", false, "delete the unreferenced bundles; without it nothing is deleted")
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: content-gc -environment env -active-catalog-version version -min-app-version version [-platform Android|iOS] [-grace 24h] [-delete]")
		flags.PrintDefaults()
	}
	flags.Parse(os.Args[1:])
	retention := contentpublish.Retention{
		Target:        contentpublish.Target{Environment: *environment, Platform: *platform, CatalogVersion: *active, StoreURL: os.Getenv("HD_CONTENT_STORE_URL")},
		MinAppVersion: *minimum,
		Grace:         *grace,
	}
	if err := retention.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	store, err := objectstore.New(objectstore.Config{
		BaseURL:   retention.Target.StoreURL,
		Region:    os.Getenv("HD_CONTENT_STORE_REGION"),
		AccessKey: os.Getenv("HD_CONTENT_STORE_ACCESS_KEY"),
		SecretKey: os.Getenv("HD_CONTENT_STORE_SECRET_KEY"),
		URLTTL:    urlLifetime,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if err := run(context.Background(), store, retention, *confirmed, os.Stdout, time.Now()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, bucket contentpublish.Bucket, retention contentpublish.Retention, confirmed bool, out io.Writer, now time.Time) error {
	sweep, err := contentpublish.PlanSweep(ctx, bucket, retention, now)
	if err != nil {
		return err
	}
	report(out, sweep)
	if !confirmed {
		fmt.Fprintf(out, "dry run: %d bundles would be deleted; run again with -delete to delete them\n", len(sweep.Unreferenced))
		return nil
	}
	deleted, err := contentpublish.DeleteUnreferenced(ctx, bucket, sweep, now)
	fmt.Fprintf(out, "deleted %d of %d unreferenced bundles\n", deleted, len(sweep.Unreferenced))
	return err
}

func report(out io.Writer, sweep contentpublish.Sweep) {
	supported := 0
	for _, catalog := range sweep.Catalogs {
		state := "unsupported"
		if catalog.Supported {
			state = "in scope"
			supported++
		}
		fmt.Fprintf(out, "catalog %s  %s: %s\n", state, catalog.Key, catalog.Reason)
	}
	lines(out, "keep referenced", sweep.Referenced)
	lines(out, "keep young", sweep.Young)
	lines(out, "missing", sweep.Missing)
	lines(out, "unreferenced", sweep.Unreferenced)
	fmt.Fprintf(out, "catalogs: %d in scope, %d unsupported; bundles: %d referenced, %d young, %d unreferenced, %d referenced but missing\n",
		supported, len(sweep.Catalogs)-supported, len(sweep.Referenced), len(sweep.Young), len(sweep.Unreferenced), len(sweep.Missing))
}

func lines(out io.Writer, label string, keys []string) {
	for _, key := range keys {
		fmt.Fprintf(out, "%s  %s\n", label, key)
	}
}
