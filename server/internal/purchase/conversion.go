package purchase

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/shop"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

const SourceConversion = "coin_pack_conversion"

var (
	ErrNoCrystalPack = errors.New("purchase: no crystal pack for a coin pack tier")
	coinPackProduct  = regexp.MustCompile(`^coins_tier([1-9][0-9]*)$`)
)

type Conversion struct {
	Account       string
	Platform      Platform
	TransactionID string
	Product       string
	CoinsDebited  int64
	Crystals      int64
}

type conversionPlan struct {
	purchase record
	crystals int64
}

func ConvertCoinPacks(ctx context.Context, pool *pgxpool.Pool, wallets *wallet.Store, catalog *shop.Catalog, now time.Time) ([]Conversion, error) {
	plans, err := planConversions(ctx, pool, catalog)
	if err != nil {
		return nil, err
	}
	var converted []Conversion
	for _, plan := range plans {
		conversion, done, err := convert(ctx, pool, wallets, plan, now)
		if err != nil {
			return converted, err
		}
		if done {
			converted = append(converted, conversion)
		}
	}
	return converted, nil
}

func planConversions(ctx context.Context, pool *pgxpool.Pool, catalog *shop.Catalog) ([]conversionPlan, error) {
	rows, err := pool.Query(ctx, `select `+recordColumns+` from purchases where state = $1 and product_id like 'coins\_tier%' order by id`, RecordGranted)
	if err != nil {
		return nil, err
	}
	purchases, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (record, error) { return scanRecord(row) })
	if err != nil {
		return nil, err
	}
	plans := make([]conversionPlan, 0, len(purchases))
	for _, p := range purchases {
		crystals, err := crystalPackFor(catalog, p.Product)
		if err != nil {
			return nil, err
		}
		plans = append(plans, conversionPlan{purchase: p, crystals: crystals})
	}
	return plans, nil
}

func crystalPackFor(catalog *shop.Catalog, product string) (int64, error) {
	tier := coinPackProduct.FindStringSubmatch(product)
	if tier == nil {
		return 0, fmt.Errorf("%w: %s", ErrNoCrystalPack, product)
	}
	for _, offer := range catalog.Offers {
		if offer.Kind == shop.KindCrystals && offer.ProductID == "crystals_tier"+tier[1] {
			return offer.Bundle.Crystals, nil
		}
	}
	return 0, fmt.Errorf("%w: %s", ErrNoCrystalPack, product)
}

func convert(ctx context.Context, pool *pgxpool.Pool, wallets *wallet.Store, plan conversionPlan, now time.Time) (Conversion, bool, error) {
	p := plan.purchase
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Conversion{}, false, err
	}
	defer tx.Rollback(ctx)
	if err := reward.LockIn(ctx, tx, p.Account); err != nil {
		return Conversion{}, false, err
	}
	key := "conversion:" + string(p.Platform) + ":" + p.TransactionID
	var converted bool
	if err := tx.QueryRow(ctx, `select exists (select 1 from ledger_entries where account_id = $1 and currency = $2 and idempotency_key = $3)`, p.Account, wallet.Crystal, key).Scan(&converted); err != nil || converted {
		return Conversion{}, false, err
	}
	granted, err := grantedCoins(ctx, tx, p)
	if err != nil {
		return Conversion{}, false, err
	}
	balance, err := wallets.BalanceIn(ctx, tx, p.Account, wallet.Coin)
	if err != nil {
		return Conversion{}, false, err
	}
	conversion := Conversion{Account: p.Account, Platform: p.Platform, TransactionID: p.TransactionID, Product: p.Product, CoinsDebited: max(0, min(balance, granted)), Crystals: plan.crystals}
	source := wallet.Source{Type: SourceConversion, Ref: p.TransactionID}
	if conversion.CoinsDebited > 0 {
		if _, err := wallets.ApplyIn(ctx, tx, wallet.Entry{Account: p.Account, Currency: wallet.Coin, Amount: -conversion.CoinsDebited, Reason: wallet.CoinPackConversion, Source: source, Key: key}, now); err != nil {
			return Conversion{}, false, err
		}
	}
	if _, err := wallets.ApplyIn(ctx, tx, wallet.Entry{Account: p.Account, Currency: wallet.Crystal, Amount: conversion.Crystals, Reason: wallet.CoinPackConversion, Source: source, Key: key}, now); err != nil {
		return Conversion{}, false, err
	}
	if _, err := tx.Exec(ctx, `update reward_grants set report = report || jsonb_build_object('coins', 0, 'crystals', $3::bigint) where account_id = $1 and idempotency_key = $2`, p.Account, p.GrantKey, conversion.Crystals); err != nil {
		return Conversion{}, false, err
	}
	return conversion, true, tx.Commit(ctx)
}

func grantedCoins(ctx context.Context, tx pgx.Tx, p record) (int64, error) {
	var coins int64
	err := tx.QueryRow(ctx, `select amount from ledger_entries where account_id = $1 and currency = $2 and reason = $3 and idempotency_key = $4`, p.Account, wallet.Coin, wallet.Purchase, purchaseKey(p)).Scan(&coins)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return coins, err
}
