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

const urlLifetime = 15 * time.Minute

func main() {
	flags := flag.NewFlagSet("content-publish", flag.ExitOnError)
	dir := flags.String("dir", "", "remote folder of an Addressables build, client/ServerData/<platform>")
	environment := flags.String("environment", "", "dev, staging or production")
	platform := flags.String("platform", "Android", "Android or iOS")
	version := flags.String("catalog-version", "", "catalog version of the build")
	dryRun := flags.Bool("dry-run", false, "print the upload plan and send nothing")
	flags.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: content-publish -dir folder -environment env -catalog-version version [-platform Android|iOS] [-dry-run]")
		flags.PrintDefaults()
	}
	flags.Parse(os.Args[1:])
	if *dir == "" {
		fmt.Fprintln(os.Stderr, "-dir is required")
		os.Exit(2)
	}
	target := contentpublish.Target{Environment: *environment, Platform: *platform, CatalogVersion: *version, StoreURL: os.Getenv("HD_CONTENT_STORE_URL")}
	if err := run(context.Background(), target, *dir, *dryRun, os.Getenv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, target contentpublish.Target, dir string, dryRun bool, env func(string) string, out io.Writer) error {
	uploads, err := contentpublish.Plan(dir, target)
	if err != nil {
		return err
	}
	for _, upload := range uploads {
		fmt.Fprintf(out, "%s  %s\n", upload.Key, upload.CacheControl)
	}
	if dryRun {
		return nil
	}
	store, err := objectstore.New(objectstore.Config{
		BaseURL:   target.StoreURL,
		Region:    env("HD_CONTENT_STORE_REGION"),
		AccessKey: env("HD_CONTENT_STORE_ACCESS_KEY"),
		SecretKey: env("HD_CONTENT_STORE_SECRET_KEY"),
		URLTTL:    urlLifetime,
	})
	if err != nil {
		return err
	}
	sent, err := contentpublish.Publish(ctx, store, dir, uploads, time.Now())
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "published %d of %d files; activate %s with HD_CATALOG_VERSION\n", sent, len(uploads), target.CatalogVersion)
	return nil
}
