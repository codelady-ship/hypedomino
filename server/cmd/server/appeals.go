package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/app"
	"github.com/Alievs-corp/hypedomino/server/internal/appeal"
	"github.com/Alievs-corp/hypedomino/server/internal/auth"
	"github.com/Alievs-corp/hypedomino/server/internal/liveops"
	pb "github.com/Alievs-corp/hypedomino/server/internal/pb/hypedomino/v1"
	"github.com/Alievs-corp/hypedomino/server/internal/profile"
	"github.com/Alievs-corp/hypedomino/server/internal/sanction"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/token"
)

func enableAppeals(a *app.App, pool *pgxpool.Pool, tokens *token.Service, values *snapshot.Watcher[liveops.Values], signIn *auth.Handler, profiles *profile.Handler, logger *slog.Logger) func(account string, sanctionID int64) *pb.AppealGrant {
	settings := func() liveops.Appeals {
		current, _, _ := values.Current()
		return current.Appeals()
	}
	grants := appeal.NewGranter(tokens, settings)
	signIn.EnableAppeals(grants)
	a.Handle("POST /appeals", appeal.NewHandler(tokens, appeal.StoreSubmitter(pool), settings, logger, time.Now))
	profiles.SetRenameAppeals(func(ctx context.Context, account string) (any, error) {
		grant, err := grants.Running(ctx, pool, account, sanction.ForcedRename, time.Now())
		if grant == nil {
			return nil, err
		}
		return grant, err
	})
	return func(account string, sanctionID int64) *pb.AppealGrant {
		grant, err := grants.Grant(account, sanctionID)
		if err != nil {
			logger.Error("appeal grant failed", "account", account, "error", err)
		}
		return grant.Proto()
	}
}
