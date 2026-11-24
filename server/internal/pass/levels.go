package pass

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const SourceLevelPurchase = "pass_level_purchase"

var (
	ErrLevelCount = errors.New("pass.level_count")
	ErrRequest    = errors.New("request.invalid")
	ErrNoWallet   = errors.New("pass: no wallet")
)

type Spender interface {
	ApplyIn(ctx context.Context, tx pgx.Tx, entry wallet.Entry, now time.Time) (wallet.Applied, error)
	BalanceIn(ctx context.Context, q wallet.Querier, account string, currency wallet.Currency) (int64, error)
}

type LevelPurchase struct {
	Season   string          `json:"season"`
	Count    int             `json:"count"`
	Currency wallet.Currency `json:"currency"`
	Price    int64           `json:"price"`
	Level    int             `json:"level"`
	XP       int64           `json:"xp"`
	Balance  int64           `json:"balance"`
	Repeated bool            `json:"repeated"`
}

func (s *Store) SetWallet(spender Spender) {
	s.wallet = spender
}

func LevelPurchaseXp(curve Curve, xp int64, count int) int64 {
	level := curve.LevelFor(xp)
	return curve.XpToReach(level+count) - curve.XpToReach(level)
}

func (s *Store) BuyLevels(ctx context.Context, account string, count int, currency wallet.Currency, requestID string, now time.Time) (LevelPurchase, error) {
	if requestID == "" || (currency != wallet.Coin && currency != wallet.Crystal) {
		return LevelPurchase{}, ErrRequest
	}
	if s.wallet == nil {
		return LevelPurchase{}, ErrNoWallet
	}
	current, version, err := s.current()
	if err != nil {
		return LevelPurchase{}, err
	}
	season, ok := current.Active(now)
	if !ok {
		return LevelPurchase{}, ErrNoSeason
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return LevelPurchase{}, err
	}
	defer tx.Rollback(ctx)
	if err := reward.LockIn(ctx, tx, account); err != nil {
		return LevelPurchase{}, err
	}
	progress, err := s.enterIn(ctx, tx, account, season, version, now)
	if err != nil {
		return LevelPurchase{}, err
	}
	key := "spend:" + requestID
	purchase := LevelPurchase{Season: season.ID, Count: count, Currency: currency, Price: int64(count) * current.LevelPrice(currency)}
	repeated, err := s.xpEventExists(ctx, tx, account, key)
	if err != nil {
		return LevelPurchase{}, err
	}
	if !repeated {
		level := season.Curve.LevelFor(progress.XP)
		if count < 1 || count > season.Curve.Levels-level {
			return LevelPurchase{}, ErrLevelCount
		}
		spend := wallet.Entry{Account: account, Currency: currency, Amount: -purchase.Price, Reason: wallet.PassLevelPurchase, Source: wallet.Source{Type: SourcePass, Ref: season.ID}, Key: key, ConfigVersion: int(version)}
		if _, err := s.wallet.ApplyIn(ctx, tx, spend, now); err != nil {
			return LevelPurchase{}, err
		}
		event := xpEvent{Amount: int(LevelPurchaseXp(season.Curve, progress.XP, count)), Source: cosmetics.Source{Type: SourceLevelPurchase, Ref: requestID}, Key: key}
		if _, err := s.addXpIn(ctx, tx, account, season, version, event, now); err != nil {
			return LevelPurchase{}, err
		}
	}
	if progress, err = progressIn(ctx, tx, account, season.ID); err != nil {
		return LevelPurchase{}, err
	}
	purchase.XP, purchase.Level, purchase.Repeated = progress.XP, season.Curve.LevelFor(progress.XP), repeated
	if purchase.Balance, err = s.wallet.BalanceIn(ctx, tx, account, currency); err != nil {
		return LevelPurchase{}, err
	}
	return purchase, tx.Commit(ctx)
}

func (s *Store) xpEventExists(ctx context.Context, tx pgx.Tx, account, key string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `select exists (select 1 from pass_xp_events where account_id = $1 and idempotency_key = $2)`, account, key).Scan(&exists)
	return exists, err
}
