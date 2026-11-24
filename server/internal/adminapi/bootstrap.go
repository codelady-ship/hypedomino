package adminapi

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/adminaccount"
	"github.com/Alievs-corp/hypedomino/server/internal/audit"
	"github.com/Alievs-corp/hypedomino/server/internal/rbac"
)

var ErrBootstrapped = errors.New("adminapi: admin accounts already exist")

const BootstrapActor = "bootstrap"

func Bootstrap(ctx context.Context, pool *pgxpool.Pool, config Config, email, displayName string, now time.Time) (adminaccount.Account, error) {
	if !strings.HasSuffix(adminaccount.NormalizeEmail(email), "@"+strings.ToLower(config.GoogleDomain)) {
		return adminaccount.Account{}, errNotWorkspace
	}
	var created adminaccount.Account
	err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `lock table admin_accounts in exclusive mode`); err != nil {
			return err
		}
		var existing int
		if err := tx.QueryRow(ctx, `select count(*) from admin_accounts`).Scan(&existing); err != nil {
			return err
		}
		if existing > 0 {
			return ErrBootstrapped
		}
		var err error
		created, err = adminaccount.NewStore(pool).In(tx).Create(ctx, email, displayName, []rbac.Role{rbac.SuperAdmin}, now)
		if err != nil {
			return err
		}
		entry := accountEntry("create", changeNote{Reason: BootstrapActor}, adminaccount.Account{}, created)
		entry.Actor = audit.Actor{ID: BootstrapActor, Roles: []rbac.Role{rbac.SuperAdmin}}
		entry.Environment = config.Environment
		entry.Source = audit.Source{SessionID: BootstrapActor}
		_, err = audit.NewStore(pool).In(tx).Append(ctx, entry, now)
		return err
	})
	return created, err
}
