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
	"time"

	"github.com/Alievs-corp/hypedomino/server/internal/account"
	"github.com/Alievs-corp/hypedomino/server/internal/analytics"
	"github.com/Alievs-corp/hypedomino/server/internal/db"
	"github.com/Alievs-corp/hypedomino/server/internal/deletionlog"
	"github.com/Alievs-corp/hypedomino/server/internal/objectstore"
)

const (
	urlLifetime               = 15 * time.Minute
	defaultGrace              = 30 * 24 * time.Hour
	defaultRetention          = 35 * 24 * time.Hour
	defaultAnalyticsRetention = 395 * 24 * time.Hour
)

var errUsage = errors.New("account-deletion")

type settings struct {
	databaseURL        string
	grace              time.Duration
	retention          time.Duration
	analyticsRetention time.Duration
	store              objectstore.Config
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	err := dispatch(ctx, os.Args[1:], os.Getenv, os.Stdout, time.Now)
	if errors.Is(err, errUsage) {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "usage: account-deletion run | account-deletion replay -since <RFC 3339 time>")
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func dispatch(ctx context.Context, args []string, getenv func(string) string, out io.Writer, now func() time.Time) error {
	if len(args) == 0 {
		return fmt.Errorf("%w: no command", errUsage)
	}
	command, args := args[0], args[1:]
	var since time.Time
	switch command {
	case "run":
		if len(args) != 0 {
			return fmt.Errorf("%w: run takes no arguments", errUsage)
		}
	case "replay":
		parsed, err := parseSince(args)
		if err != nil {
			return err
		}
		since = parsed
	default:
		return fmt.Errorf("%w: unknown command %q", errUsage, command)
	}
	config, err := settingsFrom(getenv)
	if err != nil {
		return err
	}
	store, err := objectstore.New(config.store)
	if err != nil {
		return fmt.Errorf("%w: HD_DELETION_LOG_*: %v", errUsage, err)
	}
	pool, err := db.Open(ctx, config.databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	accounts, log := account.New(pool), deletionlog.New(store, now)
	if command == "replay" {
		return replay(ctx, accounts, log, since, out)
	}
	return run(ctx, accounts, log, analytics.NewStore(pool), config, now(), out)
}

func parseSince(args []string) (time.Time, error) {
	flags := flag.NewFlagSet("replay", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	since := flags.String("since", "", "replay deletions logged at or after this time")
	if err := flags.Parse(args); err != nil {
		return time.Time{}, fmt.Errorf("%w: %v", errUsage, err)
	}
	parsed, err := time.Parse(time.RFC3339Nano, *since)
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: -since %q is not an RFC 3339 time", errUsage, *since)
	}
	return parsed.UTC(), nil
}

func settingsFrom(getenv func(string) string) (settings, error) {
	config := settings{
		databaseURL:        getenv("HD_DATABASE_URL"),
		grace:              defaultGrace,
		retention:          defaultRetention,
		analyticsRetention: defaultAnalyticsRetention,
		store: objectstore.Config{
			BaseURL:   getenv("HD_DELETION_LOG_URL"),
			Region:    getenv("HD_DELETION_LOG_REGION"),
			AccessKey: getenv("HD_DELETION_LOG_ACCESS_KEY"),
			SecretKey: getenv("HD_DELETION_LOG_SECRET_KEY"),
			URLTTL:    urlLifetime,
		},
	}
	if config.databaseURL == "" {
		return settings{}, fmt.Errorf("%w: HD_DATABASE_URL is not set", errUsage)
	}
	if config.store.BaseURL == "" {
		return settings{}, fmt.Errorf("%w: HD_DELETION_LOG_URL is not set; a deletion is complete only once it is logged", errUsage)
	}
	for name, target := range map[string]*time.Duration{"HD_ACCOUNT_DELETION_GRACE": &config.grace, "HD_DELETION_LOG_RETENTION": &config.retention, "HD_ANALYTICS_RAW_RETENTION": &config.analyticsRetention} {
		if err := positiveDuration(getenv(name), target); err != nil {
			return settings{}, fmt.Errorf("%w: %s: %v", errUsage, name, err)
		}
	}
	return config, nil
}

func positiveDuration(value string, target *time.Duration) error {
	if value == "" {
		return nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return err
	}
	if parsed <= 0 {
		return fmt.Errorf("%s is not positive", value)
	}
	*target = parsed
	return nil
}

func run(ctx context.Context, accounts *account.Store, log *deletionlog.Log, events *analytics.Store, config settings, now time.Time, out io.Writer) error {
	return errors.Join(erase(ctx, accounts, log, config, now, out), purgeAnalytics(ctx, events, config, now, out))
}

func erase(ctx context.Context, accounts *account.Store, log *deletionlog.Log, config settings, now time.Time, out io.Writer) error {
	completed, err := accounts.DeleteDue(ctx, log, now.Add(-config.grace), now)
	fmt.Fprintf(out, "%d deletion(s) erased and logged\n", completed)
	if err != nil {
		return err
	}
	pruned, err := log.Prune(ctx, now.Add(-config.retention))
	fmt.Fprintf(out, "%d log entries older than %s pruned\n", pruned, config.retention)
	return err
}

func purgeAnalytics(ctx context.Context, events *analytics.Store, config settings, now time.Time, out io.Writer) error {
	deleted, err := events.PurgeRaw(ctx, now.Add(-config.analyticsRetention))
	fmt.Fprintf(out, "%d raw analytics events older than %s deleted\n", deleted, config.analyticsRetention)
	return err
}

func replay(ctx context.Context, accounts *account.Store, log *deletionlog.Log, since time.Time, out io.Writer) error {
	deletions, err := log.Since(ctx, since)
	if err != nil {
		return err
	}
	reapplied := 0
	for _, d := range deletions {
		done, err := accounts.Reapply(ctx, d)
		if err != nil {
			return fmt.Errorf("re-apply the deletion of %s: %w", d.Account, err)
		}
		if done {
			reapplied++
			fmt.Fprintf(out, "re-applied %s as tombstone %s, deleted %s\n", d.Account, d.Tombstone, d.At.Format(time.RFC3339Nano))
		}
	}
	fmt.Fprintf(out, "%d of %d logged deletion(s) since %s re-applied, the rest were already absent\n", reapplied, len(deletions), since.Format(time.RFC3339Nano))
	return nil
}
