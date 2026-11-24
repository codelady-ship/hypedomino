package matchreward

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/snapshot"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const SourceMatch = "match"

var ErrNoConfig = errors.New("match_rewards.unavailable")

type Result struct {
	Coins    int64 `json:"coins"`
	FirstWin int64 `json:"first_win"`
}

type Store struct {
	pool   *pgxpool.Pool
	config *snapshot.Watcher[*Config]
	wallet *wallet.Store
}

func NewWatcher(pool *pgxpool.Pool, logger *slog.Logger) *snapshot.Watcher[*Config] {
	watcher := snapshot.New(pool, Domain, Parse, logger)
	watcher.Seed(Default(), 0)
	return watcher
}

func NewStore(pool *pgxpool.Pool, config *snapshot.Watcher[*Config], wallet *wallet.Store) *Store {
	return &Store{pool: pool, config: config, wallet: wallet}
}

func (s *Store) Award(ctx context.Context, match Match) (Result, error) {
	config, version, ok := s.config.Current()
	if !ok {
		return Result{}, ErrNoConfig
	}
	coins, bonus := config.MatchCoins(match), config.FirstWinCoins(match)
	if coins == 0 && bonus == 0 {
		return Result{}, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Result{}, err
	}
	defer tx.Rollback(ctx)
	if err := reward.LockIn(ctx, tx, match.Account); err != nil {
		return Result{}, err
	}
	day := EconomyDay(match.EndedAt)
	var earned int64
	if err := tx.QueryRow(ctx, `select coalesce(sum(amount), 0) from ledger_entries where account_id = $1 and reason = $2 and created_at >= $3 and created_at < $4 and idempotency_key <> $5`,
		match.Account, wallet.MatchReward, day, day.AddDate(0, 0, 1), "match:"+match.ID).Scan(&earned); err != nil {
		return Result{}, err
	}
	source := wallet.Source{Type: SourceMatch, Ref: match.ID}
	var result Result
	if amount := capToRemainder(coins, earned, config.DailyMatchCoinCap); amount > 0 {
		if result.Coins, err = s.credit(ctx, tx, wallet.Entry{Account: match.Account, Amount: amount, Reason: wallet.MatchReward, Source: source, Key: "match:" + match.ID, ConfigVersion: int(version)}, match.EndedAt); err != nil {
			return Result{}, err
		}
	}
	if bonus > 0 {
		if result.FirstWin, err = s.credit(ctx, tx, wallet.Entry{Account: match.Account, Amount: bonus, Reason: wallet.FirstWin, Source: source, Key: "first_win:" + day.Format(time.DateOnly), ConfigVersion: int(version)}, match.EndedAt); err != nil {
			return Result{}, err
		}
	}
	return result, tx.Commit(ctx)
}

func (s *Store) credit(ctx context.Context, tx pgx.Tx, entry wallet.Entry, now time.Time) (int64, error) {
	applied, err := s.wallet.ApplyIn(ctx, tx, entry, now)
	if err != nil || applied.Repeated {
		return 0, err
	}
	return applied.Amount, nil
}
