package shop

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Alievs-corp/hypedomino/server/internal/cosmetics"
	"github.com/Alievs-corp/hypedomino/server/internal/reward"
	"github.com/Alievs-corp/hypedomino/server/internal/wallet"
)

type ChestOpen struct {
	Item        string `json:"item"`
	Rarity      string `json:"rarity"`
	OpensToPity int    `json:"opens_to_pity"`
}

func pityIn(ctx context.Context, q wallet.Querier, account string) (int, error) {
	var pity int
	err := q.QueryRow(ctx, `select pity_after from chest_opens where account_id = $1 order by id desc limit 1`, account).Scan(&pity)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	return pity, err
}

func (s *Store) chestStatus(offer Offer, st standing) (status, error) {
	out := status{owned: []string{}, compensation: map[string]int64{}}
	for _, id := range offer.Bundle.Cosmetics {
		held, err := st.owns(id)
		if err != nil {
			return out, err
		}
		if held {
			out.owned = append(out.owned, id)
		}
	}
	unowned, err := chestPool(offer, s.inventory.Catalog(), st.owns)
	if err != nil {
		return out, err
	}
	if len(unowned) == 0 {
		out.refusal = ErrOwned
	}
	return out, nil
}

func (s *Store) open(ctx context.Context, account string, offer Offer, rules ChestRules, version int64, requestID string, now time.Time) (Purchase, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Purchase{}, err
	}
	defer tx.Rollback(ctx)
	if err := reward.LockIn(ctx, tx, account); err != nil {
		return Purchase{}, err
	}
	key := "chest:" + requestID
	earlier, err := reward.GetIn(ctx, tx, account, key)
	if err == nil {
		return s.reopen(ctx, tx, earlier, rules)
	}
	if !errors.Is(err, reward.ErrUnknownGrant) {
		return Purchase{}, err
	}
	pity, err := pityIn(ctx, tx, account)
	if err != nil {
		return Purchase{}, err
	}
	unowned, err := chestPool(offer, s.inventory.Catalog(), func(id string) (bool, error) { return s.inventory.OwnsIn(ctx, tx, account, id) })
	if err != nil {
		return Purchase{}, err
	}
	draw, ok := rules.Draw(unowned, pity, s.random)
	if !ok {
		return Purchase{}, ErrOwned
	}
	source := cosmetics.Source{Type: SourceChest, Ref: offer.ID}
	spend := wallet.Entry{Account: account, Currency: wallet.Coin, Amount: -offer.PriceCoins, Reason: wallet.ChestPurchase, Source: wallet.Source{Type: source.Type, Ref: source.Ref}, Key: "spend:" + requestID, ConfigVersion: int(version)}
	applied, err := s.wallet.ApplyIn(ctx, tx, spend, now)
	if err != nil {
		return Purchase{}, err
	}
	if applied.Repeated {
		return Purchase{}, ErrRequest
	}
	bundle := offer.Bundle
	bundle.Cosmetics = []string{draw.Item}
	report, err := s.engine.ApplyIn(ctx, tx, reward.Grant{Account: account, Bundle: bundle, Source: source, Reason: wallet.ChestPurchase, Key: key, ConfigVersion: int(version)}, now)
	if err != nil {
		return Purchase{}, err
	}
	if _, err := tx.Exec(ctx, `insert into chest_opens (account_id, offer_id, idempotency_key, roll, pity_before, pity_after, rarity, item_id, config_version, created_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`, account, offer.ID, key, draw.Roll, draw.PityBefore, draw.PityAfter, draw.Rarity, draw.Item, version, now); err != nil {
		return Purchase{}, err
	}
	chest := &ChestOpen{Item: draw.Item, Rarity: draw.Rarity, OpensToPity: rules.OpensToPity(draw.PityAfter)}
	return Purchase{Offer: offer.ID, Currency: wallet.Coin, Balance: applied.BalanceAfter, Grant: report, Chest: chest}, tx.Commit(ctx)
}

func (s *Store) reopen(ctx context.Context, tx pgx.Tx, earlier reward.Report, rules ChestRules) (Purchase, error) {
	chest := &ChestOpen{}
	var pity int
	if err := tx.QueryRow(ctx, `select item_id, rarity, pity_after from chest_opens where account_id = $1 and idempotency_key = $2`, earlier.Account, earlier.Key).Scan(&chest.Item, &chest.Rarity, &pity); err != nil {
		return Purchase{}, err
	}
	chest.OpensToPity = rules.OpensToPity(pity)
	balance, err := s.wallet.BalanceIn(ctx, tx, earlier.Account, wallet.Coin)
	if err != nil {
		return Purchase{}, err
	}
	return Purchase{Offer: earlier.Source.Ref, Currency: wallet.Coin, Balance: balance, Grant: earlier, Chest: chest, Repeated: true}, tx.Commit(ctx)
}
