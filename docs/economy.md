# Economy

The two currencies, coin and crystal: where they come from, where they go, the chests, the reward bundle every grant uses, the shop, real-money products, promo codes, refunds and inflation controls. The server alone computes and stores everything here (ADR 0003, ADR 0007); the client sends intents and shows what the server sends (`docs/anti-cheat.md` NT-1 to NT-12).

**Status: approved by the owner on 2026-10-26 (section 13).** Every number in this document and in [`docs/economy/balance.csv`](economy/balance.csv) is a launch default to be tuned with closed-beta data. Every one is admin config (ADR 0006): read from the published config snapshot, recorded with the config version each match and grant used, and changed without a release. Names in `CamelCase` code font are config keys. The CSV is the single list of their values; section 11 is generated from it. **Two currencies and chests: owner decision of 2026-11-01 (OQ-11 to OQ-14).** The server keeps a coin and a crystal balance per account (#355) and sells crystal packs, crystal prices and crystal pass levels (#356); coin packs bought before the switch are converted once (7.6). Chests are #357.

Related: ADR 0005 (ledger, unique transaction ids), ADR 0006 (admin config), `docs/anti-cheat.md` (V-1, V-3, V-7, V-8, V-11), `docs/cosmetics.md` (acquisition sources), `docs/ranking.md` (final ranks), `docs/penalties.md` (offender rewards), `docs/localization.md` (gambling imagery, age rating, rename), `docs/privacy.md` (D-16, D-17), `docs/launch-runbook.md` (kill switches).

## 1. Principles

| Id | Rule |
|---|---|
| EP-1 | Two currencies. The **coin** (`coin`) is earned by play and never sold. The **crystal** (`crystal`) comes only from real-money purchases and what they unlock (crystal packs, offers, the premium pass track). One integer balance per currency per account. Neither converts into the other, and nothing bought with real money or crystals grants coins. |
| EP-2 | No pay-to-win. Coins, crystals and real money buy cosmetics, chests (coins only), pass levels, the premium pass track and a nickname rename, nothing else. Nothing for sale changes rating, MMR, LP, placement, matchmaking, turn time, penalties, hidden information or any rule in `docs/rules.md`. No rank protection, queue-delay skip or penalty removal is sold. |
| EP-3 | No paid randomised items. Random contents exist only in chests (3.4), bought with coins only. Coins cannot be bought (EP-1), so a chest is never paid for with real money, directly or through crystals. A chest shows its pool and the odds of every rarity before purchase. Every other offer and reward shows exactly what it grants. This keeps the IARC lowest tier, PEGI 3 and App Store 4+ targets and the gambling imagery rule (`docs/localization.md` sections 6 and 7). |
| EP-4 | No wagers. No coin entry fee for a match, room or event whose prize is coins or items. Coins are never staked on a result. |
| EP-5 | No transfers. Coins, crystals and items cannot be traded, gifted or moved between accounts. Neither currency has a cash value, can be withdrawn or expires. |
| EP-6 | Server authority. Every price, amount, eligibility and window is the server's (NT-1, NT-2). Every balance change is a ledger entry (NT-11, section 5). Every window uses server time (NT-6, #208). |
| EP-7 | Store-only payments. Real money goes through Google Play Billing and the App Store only, granted after server-side receipt validation (NT-5, #106). No own payment path and no own currency conversion (`docs/privacy.md` 7.4). |
| EP-8 | One price for everyone. No personalised prices. Regional differences come only from the stores' pricing (section 7.4). |

## 2. Sources

| Source | Section | Ledger reason | Details owned by |
|---|---|---|---|
| Match reward | 2.1 | `match_reward` | #103 |
| First win of the day | 2.2 | `first_win` | #103 |
| Daily login | 2.3 | `daily_login` | #131 |
| Missions | 2.4 | `mission` | #9, #127 |
| Pass tracks | 2.5 | `pass_reward` | #9, #123, #124 |
| Rank rewards | 2.6 | `rank_reward` | #8, #120 |
| Live events | 2.7 | `event_reward` | #146 |
| Duplicate compensation | 2.7 | `duplicate_compensation` | #102 |
| Promo codes | 8 | `promo` | #108 |
| Purchase | 7 | `purchase`, `first_purchase_bonus` | #106 |

Every source grants coins except the purchase, the first-purchase bonus and the premium pass track, which grant crystals (EP-1).

**Economy day.** Every daily limit (first win, daily login, daily missions, `DailyMatchCoinCap`) uses one day boundary computed on the server from server time (#208): 00:00 UTC, the same for every player. There is no per-player reset time zone (owner decision, 2026-11-01, #131).

### 2.1 Match rewards

Coins per completed match:

| Mode | Key | Win | Loss |
|---|---|---|---|
| Ranked 1v1 | `MatchCoinsRanked1v1` | 30 | 10 |
| Ranked 2v2 | `MatchCoinsRanked2v2` | 40 | 14 |
| Casual 1v1 | `MatchCoinsCasual1v1` | 18 | 6 |
| Casual 2v2 | `MatchCoinsCasual2v2` | 24 | 8 |

- There is no draw: a match cannot end level (R-47).
- A 2v2 match is longer, so it pays more per match; per hour both ranked modes earn the same within one coin (section 11).
- Casual is the unranked matchmaking queue (R-56). Casual earns about 60% of ranked per hour because it may fill seats with bots (#48) and is the easier farm target.
- A casual match that started with a bot-filled seat pays `BotSeatRewardPercent` (50) of the table value, rounded to the nearest integer, halves away from zero. A bot that replaces an AFK or leaving player during a match (R-50 to R-52) reduces nobody's reward.
- Custom rooms (R-55) pay no coins: players pick the teams, so wins can be arranged. They give no pass XP and advance no missions or medals (`docs/pass-missions-medals.md` section 3).
- AFK and leave offenders in ranked and casual matches get no coins, except AFK returned (`docs/penalties.md` sections 5 and 6). Partners and opponents get the normal reward.
- A match cancelled by the server pays nothing.
- No bonus for score margin or match length: margin rewards running up the score, and length is already priced per mode.
- `DailyMatchCoinCap` (400) limits match coins per account per economy day. A reward that would cross the cap is cut to the remainder; after the cap a match pays 0 coins and still gives pass XP and mission progress (`docs/pass-missions-medals.md` 2.3). The first-win bonus is outside the cap.

### 2.2 First win of the day

`FirstWinBonusCoins` (100) on the first won ranked or casual match of the economy day. Custom rooms do not count.

- Implemented in `server/internal/matchreward` (#103): the `match_rewards` config domain (`docs/economy/examples/match_rewards.json`) holds the table of 2.1, `BotSeatRewardPercent`, `DailyMatchCoinCap` and `FirstWinBonusCoins`. `Award` credits `match_reward` under key `match:<matchId>`, cut to what is left of the cap from that economy day's `match_reward` entries, and `first_win` under key `first_win:<economyDay>`, outside the cap. The game server runs a relay on the `economy` outbox topic (`HD_OUTBOX_POLL`, #440): each row carries a `matchstore.Outcome` (match, mode, kind, bot seat, cancelled, end time and per player the account, win and offender flag) and `Award` runs for every player; a redelivered row pays nothing twice. Matchmade rooms write the outcome when they end (`docs/matchmaking.md` section 4); the offender flag stays false until penalty records are stored.

### 2.3 Daily login

- A 7-day cycle. Claiming day n grants entry n of `DailyLoginCoins` (30, 30, 50, 50, 70, 70, 150; 450 per cycle). After day 7 the cycle starts again at day 1.
- One claim per economy day, by intent; the server decides the day (V-7).
- A missed day pauses the cycle; it does not reset it. No streak to lose.
- Every day grants coins only (owner decision, 2026-11-01, #131): no pass XP, cosmetic or medal. A 30-day calendar (#131) is not proposed for launch.
- Implemented in `server/internal/dailylogin` (#131): the `daily_login` config domain (`docs/economy/examples/daily_login.json`) holds 7 or 30 days of `coins`; publish refuses any other field. A claim grants the next day of the cycle through the reward engine, reason `daily_login`, key `daily_login:<economyDay>`, and counts claims rather than dates, so a missed day pauses the cycle.

### 2.4 Missions

Mission types, goals and the split of rewards: `docs/pass-missions-medals.md` section 4. Their coins come from fixed budgets: `DailyMissionCoinBudget` (120) per economy day for all daily missions together, `WeeklyMissionCoinBudget` (600) per week for all weekly missions. Admin publish rejects a mission set above its budget. One-time achievement missions are outside the steady-state budget; #110 reports their total.

### 2.5 Pass

Season length, levels and the XP curve: `docs/pass-missions-medals.md` section 2. Coin budget per pass season: `PassFreeTrackCoins` (1500) on the free track, `PassPremiumTrackCrystals` (600) on the premium track, which is bought with real money and so pays crystals, never coins (EP-1). Admin publish rejects a season above its budget. The premium track returns more crystals than its price buys at the reference rate (section 11).

### 2.6 Rank rewards

At season end every placed player gets coins for their final rank in each mode, once per season and mode (`docs/ranking.md` 7.1, #120). `RankRewardCoins` by final tier: Iron 100, Bronze 200, Silver 400, Gold 700, Platinum 1000, Emerald 1500, Diamond 2000, Master 3000, Grandmaster 4000, Challenger 5000. Cosmetic rank rewards are bundles in #120's table. No final rank in a mode, no reward for that mode.

- Implemented in `server/internal/rating` (#120): `RankRewardCoins` per tier and `RankRewards` (a bundle per mode and final tier) are the `rating` config; the season reward run grants both as one reward engine grant with key `rank:<seasonId>:<mode>`, so a player is paid once per season and mode whatever happens to the job in between.

### 2.7 Events and duplicates

- A live event grants reward bundles from its own config. Its coins count in the section 10 monitors. Event cosmetics are sold for coins in the event shop section (6.1).
- Duplicate compensation (#102): when a bundle grants a cosmetic the account already owns, the item is not granted again and the account gets `DuplicateCompensationPercent` (20) of that item's guide price (3.1) in coins. A medal already owned is skipped. Shop offers never sell an owned item (6.2), so duplicates only come from rewards and bundles.

## 3. Sinks

| Sink | Section | Ledger reason |
|---|---|---|
| Cosmetics (shop and event shop) | 3.1, 6 | `shop_purchase` |
| Pass levels | 3.2 | `pass_level_purchase` |
| Nickname rename | 3.3 | `rename` |
| Chests | 3.4 | `chest_purchase` |

Cosmetics and pass levels are paid in coins or crystals, as the offer states. Chests and rename are paid in coins only. Every spend is an intent carrying a client request id (section 5). The server takes the price from its own config and checks ownership and balance. A spend above the balance is rejected and counted (V-1).

### 3.1 Cosmetic guide prices

A shop offer holds its own price (`docs/cosmetics.md` section 5). The guide price is the default the admin offer form fills in and the base of duplicate compensation:

```
guide price = GuidePriceCoins[rarity] x GuidePriceTypePercent[type] / 100
```

- `GuidePriceCoins` per rarity id of #5 in order `common`, `rare`, `epic`, `legendary`, `mythic`: 400, 1500, 3500, 12000, 30000. Rarities are referenced by id only.
- `GuidePriceCrystals` per rarity, same order: 80, 300, 700, 2400, 6000, the crystal price the offer form fills in for an offer paid in crystals. Duplicate compensation is always in coins from `GuidePriceCoins`.
- `GuidePriceTypePercent` per type in order `avatar`, `character`, `frame`, `table`, `emoji`, `action`: 100, 150, 100, 150, 50, 50. A character carries four clips and a table fills the match scene; an emoji is a small, frequent item, and a lobby action is one clip seen only by friends in the lobby (owner decision 2026-10-28: action prices follow rarity like every other type and stay admin config).
- `mythic` items are sold only in time-limited offers (rotation, featured, event), never as a permanent offer.
- The full table and the play time each price takes are in section 11.

### 3.2 Pass levels

`PassLevelCoinPrice` (200) or `PassLevelCrystalPrice` (40) per level, bought by intent with a count, up to the season's last level. The premium track is a real-money product (7.2), not sold for coins: a coin price would let a completed premium track pay for the next one indefinitely.

### 3.3 Rename

`RenameCoinPrice` (1000) for each rename after the free one at setup. Cooldown and forced renames: `docs/localization.md` 5.6. A forced rename is free.

### 3.4 Chests

- A chest is a shop offer of kind `chest` in the Chests section (6.1), priced in coins only: `ChestCoinPrice` (500). Publish rejects a chest priced in crystals or as a store product, and a chest granted by a bundle, a promo code or a purchase.
- Opening a chest grants one cosmetic drawn by the server from the chest's pool. A pool holds `common` and `rare` items only: `rare` is the middle tier a chest reaches (owner decision, 2026-11-01). `epic`, `legendary` and `mythic` items are never in a pool, so they keep their shop and reward value.
- Odds: `ChestRarityPercent` (85, 15) for `common` and `rare`. Within the drawn rarity every unowned pool item has the same chance.
- Only unowned items are drawn. If no unowned item of the drawn rarity is left, the other rarity is drawn; a chest whose pool is fully owned is not sold (6.2). A chest never yields a duplicate.
- Pity: `ChestRarePityCount` (10). The tenth open in a row without a `rare` item gives a `rare` one. The counter is per account across all chests and resets on every `rare` result.
- One transaction debits the price (`chest_purchase`, key `spend:<requestId>`), draws from the server's cryptographically secure random source and grants the item through the reward engine (source `chest`, ref the chest offer id, key `chest:<requestId>`). Each open records the roll, the pity counter, the result and the config version, for support and abuse checks (#211).
- Before purchase the client shows the pool, the odds of each rarity and the opens left to the pity, all as the server sends them. The reveal shows the granted item without casino cues (`docs/localization.md` section 6): no reels, wheels, slot sounds or near-miss effects.
- Every pool item lists a `chest` source with the chest offer id (`docs/cosmetics.md` section 5).
- Expected value of an avatar or frame chest at guide prices: 0.85 x 400 + 0.15 x 1500 = 565 coins against a price of 500. The player gives up choosing the item and gets a little more value for it.
- Implemented in `server/internal/shop` (#357). A chest offer's `bundle.cosmetics` is its pool; the grant carries the same bundle id with the drawn item only. `ChestRarityPercent` and `ChestRarePityCount` are the catalog's `chests.rarityPercent` and `chests.rarePityCount` (`docs/economy/examples/shop.json`); a catalog with a chest offer and without them is refused. Only `live` pool items are drawn. `config-publish` refuses a chest whose pool item is unknown, not `common` or `rare`, or does not list the chest's `chest` source. A chest has no `limitPerAccount`. `POST /shop/buy` opens a chest and answers the drawn item, its rarity and the opens left to the pity; `GET /shop` shows a chest offer's pool as `cosmetics`, the odds as `odds` (`[{"rarity","percent"}]`, `common` first) and `opens_to_pity`. Opens are recorded in `chest_opens`. A request id already spent by another action is refused (`request.invalid`).

## 4. Reward bundle

Every grant is a reward bundle applied by one engine (#102): match rewards, first win, daily login days, missions, pass levels, rank rewards, events, promo codes, shop offers and purchases.

| Field | Type | Rule |
|---|---|---|
| `id` | string | `bundle.<slug>`, slug `[a-z0-9_]+`. Stable, never reused |
| `revision` | integer >= 1 | Increases by 1 on every published change. A grant records the revision it applied |
| `coins` | integer >= 0 | Coins added. Never in a bundle granted by a purchase, the first-purchase bonus or the premium pass track (EP-1) |
| `crystals` | integer >= 0 | Crystals added. Only in a bundle granted by a purchase, the first-purchase bonus or the premium pass track |
| `cosmetics` | array of cosmetic ids | Unique. Each item's `sources` lists the source that grants this bundle (`docs/cosmetics.md` V-15) |
| `medals` | array of medal ids | Unique. Medal ids from #9 and #129 |
| `passXp` | integer >= 0 | XP for the pass season active at grant time. With no active season it is dropped, and the grant records that |

At least one field among `coins`, `crystals`, `cosmetics`, `medals` and `passXp` is non-empty or non-zero. Admin publish rejects a bundle whose currency does not match its source.

```json
{
  "id": "bundle.season1_rank_gold",
  "revision": 1,
  "coins": 700,
  "cosmetics": ["frame.season1_gold"],
  "medals": ["medal.season1_gold"],
  "passXp": 0
}
```

- One database transaction writes the ledger entry, inventory rows, medal rows and pass XP, or none of them (ADR 0005).
- Match rewards and first win are computed from mode, result and cap, not stored; the engine builds a bundle with `coins` and `passXp` and applies it the same way.
- Every grant records its source type, source ref, bundle id and revision, and the config version. Idempotency follows section 5.
- Implemented in `server/internal/reward` (#102): `Engine.Apply` is the one path, idempotent per account and key, atomic over the ledger, inventory, medal and pass XP hooks, recorded in `reward_grants` and published to the outbox; `Engine.Revoke` reverses a grant for refunds (section 9) and support (`docs/admin.md` 6.2). Coins and crystals each go to their own balance under the grant key; `Bundle.ValidateFor(reason)` refuses a bundle whose currency its source may not move, and the mission, promo, rank reward and shop publishes use it; a pass publish refuses crystals on the free track and coins on the premium track. A revoke or reinstate moves only the currencies its reason moves, so a refund takes crystals and items back and never touches coins (section 9).

## 5. Ledger

- Append-only (ADR 0005). The balance is the sum of entries; the stored balance is reconciled against it (#101).
- Each currency has its own balance and entries. An entry holds the account, the currency (`coin` or `crystal`), a signed amount, the reason, source type and ref, idempotency key, config version, server time and the balance after.
- The idempotency key is unique per account. Repeating a write returns the first result and changes nothing (V-8).

| Reason | Sign | Idempotency key |
|---|---|---|
| `match_reward` | + | `match:<matchId>` |
| `first_win` | + | `first_win:<economyDay>` |
| `daily_login` | + | `daily_login:<economyDay>` |
| `mission` | + | `mission:<missionId>:<period>` |
| `pass_reward` | + | `pass:<seasonId>:<track>:<level>` |
| `rank_reward` | + | `rank:<seasonId>:<mode>` |
| `event_reward` | + | `event:<eventId>:<rewardId>` |
| `duplicate_compensation` | + | Key of the causing grant plus `:dup:<cosmeticId>` |
| `promo` | + | `promo:<codeId>` |
| `purchase` | + | `purchase:<store>:<transactionId>`, also unique across accounts |
| `first_purchase_bonus` | + | `first_purchase` |
| `shop_purchase`, `pass_level_purchase`, `rename`, `chest_purchase` | - | `spend:<requestId>` |
| `refund_revoke` | - | `refund:<store>:<transactionId>` |
| `refund_reversal` | + | `refund_reversal:<store>:<transactionId>` |
| `admin_adjust` | +/- | `admin:<auditId>`, with a mandatory reason (#144) |
| `coin_pack_conversion` | +/- | `conversion:<store>:<transactionId>`, one time (7.6) |

- `purchase`, `first_purchase_bonus`, `refund_revoke` and `refund_reversal` move crystals; `pass_reward` moves coins on the free track and crystals on the premium track; `shop_purchase` and `pass_level_purchase` move the currency of the price; `admin_adjust` moves either; `coin_pack_conversion` debits coins and credits crystals. Every other reason moves coins.
- `<requestId>` is generated by the client per spend intent and scoped to the account by the server, so a reused id only repeats the earlier answer.
- Spends of one account are serialised; no spend or `admin_adjust` makes the balance negative (#101). The single exception is `refund_revoke` (section 9).
- Implemented in `server/internal/wallet` (#101, #355): `Apply` is the only writer, idempotent per account, currency and key, serialised per account and currency, refusing a currency its reason does not move (`ErrWrongCurrency`; `purchase`, `first_purchase_bonus`, `refund_revoke` and `refund_reversal` move crystals only, #356), refusing any result below zero; `Audit` and `Reconcile` (`cmd/wallet-audit`) compare and restore each stored balance against the sum of its currency's entries. A `refund_revoke` entry is the only one that may take the balance below zero (#106); every other debit is refused while it is negative.

## 6. Shop

### 6.1 Sections

| Section | Content | Paid with |
|---|---|---|
| Featured | Admin-picked offers with a window | Crystals or store product |
| Daily rotation | `ShopDailyRotationSlots` (4) cosmetic offers | Coins or crystals, per offer |
| Weekly rotation | `ShopWeeklyRotationSlots` (2) cosmetic offers | Coins or crystals, per offer |
| Event | Event cosmetics while the event runs | Coins or crystals, per offer |
| Chests | Chest offers (3.4) | Coins |
| Pass | Premium track products and pass levels | Store product, coins, crystals |
| Crystals | Crystal packs | Store product |

- Rotations are the same for every player. The daily rotation changes at `ShopRotationResetHourUtc` (0) UTC, the weekly one at that hour on Monday. The admin schedules rotations ahead in the live-ops calendar (#146); a period without a schedule repeats the previous rotation. Rotation contents are scheduled, never drawn at random per player.
- Shop changes take effect on publish without a server restart (#104): the publish writes the catalog as a versioned snapshot (`config_snapshots`, domain `shop`; `docs/economy/examples/shop.json` is the format) and every node picks the newest version up within `ConfigPoll` (10 s). An invalid catalog is refused at publish and, should one reach the table, ignored by the server. The snapshot version is the config version of every purchase it priced.
- The server answers the sections above for one account: every offer with its price, contents, window end, the purchases of the account against the limit, the owned items with their compensation and whether the account may buy it now. The client renders that and sends intents only.

### 6.2 Offer

| Field | Rule |
|---|---|
| `id` | `offer.<slug>`. Cosmetic `shop` sources reference it |
| `kind` | `cosmetic`, `bundle`, `chest`, `crystals`, `pass`, `pass_levels` |
| `price` | Exactly one of: coins (integer > 0), crystals (integer > 0), a store product id (section 7). A `chest` takes coins only, a `crystals` pack a store product only |
| `bundle` | The reward bundle it grants |
| `window` | Start and optional end, server UTC |
| `limitPerAccount` | Purchases per account; 1 for cosmetics and one-time offers, none for crystal packs and chests |
| `eligibility` | Optional: hours since account creation, first purchase only |

- A cosmetic offer is not sold to an account that owns the item; the intent is rejected.
- A bundle offer is sold while at least one of its cosmetics is not owned, or while it grants a currency or pass XP besides them. Owned items get duplicate compensation (2.7), and the shop shows that before purchase.
- The client shows the server's price and never computes one.

### 6.3 Discounts

- A discount is a separate offer with its own price and window, never a percentage the client applies.
- A struck-through earlier price is shown only if it is the lowest price of the same content in the last `ReferencePriceDays` (30) days. Admin publish rejects one that is not.
- A countdown shows only the real end of the offer's window.

## 7. Real-money products

USD reference price = the App Store price point and the Google Play default price of the product. Every product is consumable. Product ids name a tier, never an amount, so amounts are retuned in config without a new store product. Ownership lives on the server, so restore on a new device is sign-in (#107).

### 7.1 Crystal packs

Products `crystals_tier1` to `crystals_tier5`. USD prices `CrystalPackUsd` (0.99, 4.99, 9.99, 19.99, 49.99) and crystals `CrystalPackCrystals` (100, 550, 1200, 2600, 7000). The bonus over `CrystalsPerUsdReference` (100 crystals per USD) grows from 1% to 40% with pack size (section 11). There are no coin packs (EP-1).

### 7.2 Pass products

- `pass_premium`, `PassPremiumUsd` (4.99): unlocks the premium track of the current pass season.
- `pass_premium_plus`, `PassPremiumPlusUsd` (9.99): the premium track plus `PassPremiumPlusLevels` (20) levels.
- The server binds the product to the pass season active at the store's purchase time. A second premium purchase in the same season is rejected before the store flow starts (#125). Premium rewards of levels already reached unlock at once (`docs/pass-missions-medals.md` 2.5).

### 7.3 Starter and first-purchase offers

- `offer_starter`, `StarterOfferUsd` (1.99): `StarterOfferCrystals` (300) plus one `rare` avatar and one `rare` frame chosen by the admin. Offered for `StarterOfferWindowHours` (168) from account creation, server time; once per account. Worth 900 crystals at crystal guide prices, about 4.5 times its price at the reference rate.
- First-purchase bonus: the first validated real-money purchase of an account grants `bundle.first_purchase` once: `FirstPurchaseBonusCrystals` (100) plus one `epic` frame. Any product counts, including the starter offer.
- A refund revokes the bonus with its purchase (section 9). A later purchase does not become "first" again.

### 7.4 Regional pricing

- Local prices come only from the stores: App Store price equalisation from the base storefront, Google Play pricing templates and automatic conversion. A market-specific override is made in the store console, noted on the admin offer, and needs owner approval.
- The game holds no exchange rates and computes no local price. The client shows the localized price string from the billing SDK.
- The crystal amount of a product is the same in every market.
- Taxes are the stores' as merchant of record (`docs/privacy.md` section 2).
- Where a store cannot charge in a country, the shop hides real-money products there. No other payment path is added (EP-7).

### 7.5 Purchase flow

1. The client sends a buy intent for an offer id. The server checks the window, eligibility, limit, the purchase kill switch (#210) and attestation (V-2), and returns the store product id and the account-bound token (`obfuscatedAccountId` on Android, `appAccountToken` on iOS).
2. The client runs the store purchase.
3. The client sends the purchase token or signed transaction. The server validates it with the store API (#106): signature, production environment, product id mapped to the offer, account token match, transaction id not used before (V-3).
4. The server records the purchase and grants the offer's bundle in one transaction, idempotent on the transaction id.
5. Only after that commit is the purchase acknowledged and consumed (Google Play) or finished (App Store). A purchase the server never granted is refunded by the store.
6. Pending purchases (deferred payment, parental approval) are granted only when the store's server notification reports them purchased (#106).
7. A validated purchase is always granted, even if the offer window closed or the purchase kill switch went on after payment (`docs/launch-runbook.md` 5.5). The grant uses the offer's revision at validation time.

- Implemented in `server/internal/purchase` (#106): step 1 is `/shop/begin` (#104), steps 3 to 7 are `/shop/complete` and the store notification endpoints; the client side of step 2 and the attestation check of step 1 are V-2; the kill switch of step 1 is `docs/anti-cheat.md` 5.2. An offer has exactly one of `priceCoins`, `priceCrystals` and `productId`; `POST /shop/buy` spends the offer's currency and answers it with the balance after; a store-product bundle grants no coins, an in-game priced one no currency, and a `crystals` pack only crystals (#356).

### 7.6 Coin pack conversion

Coin packs (`coins_tier1` to `coins_tier5`) were sold before the switch to crystals (#356). A one-time step converts each granted coin pack purchase:

- The purchase grants the crystals of the crystal pack of the same tier (`coins_tierN` becomes `CrystalPackCrystals` of `crystals_tierN`, read from the published shop catalog).
- The coins that purchase granted are debited, but never below zero: an account that spent them keeps what it bought and loses only the coins it still has.
- Both entries carry reason `coin_pack_conversion` and key `conversion:<store>:<transactionId>`; the grant is recorded as a crystal grant, so a later refund takes the crystals back (section 9).
- Refunded and pending purchases are skipped. Running the step again converts nothing twice.
- Starter offers, the first-purchase bonus and premium track coins are not converted.

## 8. Promo codes

| Type | Who can redeem | Cap |
|---|---|---|
| `single_use` | Any account; generated in batches | 1 redemption in total |
| `multi_use` | Any account, once per account | Required maximum redemptions |
| `targeted` | Listed accounts only, once each | Number of listed accounts |

A code has an id, a campaign id (what cosmetic `promo` sources reference), a bundle id, start and end in server UTC, its cap, its creator and its approver.

- Case-insensitive; letters and digits without look-alikes (`0`, `O`, `1`, `I`, `L`). Generated codes are `PromoCodeLength` (12) characters from a cryptographic random source. Admin-chosen codes are allowed for `multi_use` only. Codes are stored as keyed hashes; the admin sees a generated code once, at creation.
- Every code ends at most `PromoMaxValidityDays` (90) after its start.
- A code granting more than `PromoCoinsApprovalThreshold` (2000) coins, or any `mythic` item, is active only after a second admin approves it (ADR 0006: admin compromise is economy compromise).
- Redemption is one transaction: the redemption row, unique per code and account, the cap check and the grant. Parallel redemptions never exceed the cap or redeem twice (#108).
- Abuse limits: `PromoAttemptsPerHour` (10) attempts per account; `PromoFailedAttemptsBeforeLock` (5) failed attempts lock redemption for `PromoLockMinutes` (60). The same limits apply per device and IP (#184). Locks and redemption spikes are signals for #209 (V-11): each lock records a `promo_lock` risk signal of weight `PromoLockSignalWeight` (5) with the failed count as evidence, and a `multi_use` code whose redemptions within `PromoSpikeWindowMinutes` (60) reach `PromoSpikePercent` (50) of its cap is marked once with `spike_at`, shown on the code in the admin API, and logged as `promo redemption spike` for alerting (#28). A marked code keeps redeeming until an admin disables it. `single_use` and `targeted` codes are never marked.
- A leaked code is disabled in admin (`docs/launch-runbook.md` 5.5). Rewards already granted stay.
- A code grants coins, cosmetics, medals and pass XP, never crystals or a chest. It never grants a store product or an item whose `sources` lack its campaign. Codes are never sold.
- Implemented in `server/internal/promo` (#108): `cmd/promo` and the admin API (`docs/admin.md` 9.1, #142) create, batch, approve and disable codes; `POST /promo/redeem` redeems them. The per-account attempt limits and lock are enforced; the per-device and per-IP limits are #184.

## 9. Refunds and chargebacks

Notices: Google Play real-time developer notifications and the Voided Purchases API; App Store Server Notifications `REFUND` and `REFUND_REVERSED` (#106, #204). Processing is idempotent per notice.

1. Find the purchase by store transaction id.
2. Revoke what it granted through the ledger: a `refund_revoke` entry for its crystals; its cosmetics removed from inventory and unequipped as for `disabled` items (`docs/cosmetics.md` section 6); for a pass product, the premium track and every premium reward claimed (crystals debited, cosmetics removed). A starter or first-purchase bonus tied to it is revoked the same way.
3. Items bought with crystals are kept. If the crystals are already spent, the crystal balance goes negative and crystal spends are blocked until purchases bring it back to zero or above. Coins are never touched. This is the only way a balance goes negative.
4. Flag the account with `RefundWeight` (3, `docs/anti-cheat.md` section 5 and V-3). `RefundsBeforePurchaseBlock` (3) refunds within `RefundWindowDays` (365) block further purchases and send the account to review (#209).
5. `REFUND_REVERSED` re-grants the same contents through a `refund_reversal` entry.

- Implemented in `server/internal/purchase` (#106): Google voided purchase notifications and App Store `REFUND` and `REFUND_REVERSED`, each processed once; the purchase block of step 4 is enforced by the server and the refund is published on the `purchases` outbox topic for the risk score of #209.
- Money is refunded by the stores only, never by support. An admin may clear a negative balance with an `admin_adjust` entry and a reason.
- Deleting an account refunds nothing (`docs/privacy.md` 8.2).

## 10. Inflation controls

The monitors below are for coins. Crystals are created only by validated purchases, so crystal sales and spends are revenue analytics (#147), not inflation controls.

Caps: `DailyMatchCoinCap` (2.1), mission and pass budgets (2.4, 2.5), once-per-period idempotency keys (section 5). No transfers (EP-5): a farm account cannot move coins to a main account.

Monitors (#110, shown in the admin analytics #147):

| Monitor | Definition | Alert |
|---|---|---|
| Sink ratio | Coins spent / coins created, accounts active in the last 7 days, rolling 7 days | Outside `SinkRatioAlertPercent` (60% to 120%) |
| Account daily earn | `match_reward` + `first_win` + `daily_login` + `mission` of one account in one economy day | Above `AccountDailyEarnAlertCoins` (1500): alert and a farm signal to #209 |
| Median balance | Median balance of accounts active in the last 7 days | Above `BalanceMedianAlertCoins` (30000, one `mythic` avatar) |
| Source mix | Coins created per reason per day | Report only |

- Target sink ratio: 70% to 100%. Below 60% coins pile up with nothing worth buying; above 120% players are drawing savings down faster than they earn.
- The account daily earn alert sits above the legitimate maximum of its reasons with the proposed values (400 + 100 + 150 + 120 + 600 = 1370), so crossing it means a defect or an exploit.
- #110 compares the monitors before and after every economy config publish.

## 11. Balance sheet

Inputs are [`docs/economy/balance.csv`](economy/balance.csv): one row per key, columns `key,value,unit,kind,section`, lists separated by `;`. `kind` is `config` (admin config read by the server) or `model` (an assumption used only for this sheet):

| Model key | Value | Meaning |
|---|---|---|
| `ModelMatchMinutes` | 10, 14 | Average 1v1 and 2v2 match to 365. Estimate; replace with closed-beta telemetry (#193) |
| `ModelBetweenMatchMinutes` | 1 | Queue and result screens between matches |
| `ModelWinRatePercent` | 50 | Matchmaking aims at even matches |
| `ModelPassSeasonDays` | 90 | Same window as a ranked season (`docs/ranking.md` 7.1, `docs/pass-missions-medals.md` 2.1) |
| `ModelProfileHours` | 0.5, 1, 3 | Hours of ranked 1v1 per day for light, regular and heavy players |
| `ModelProfileMissionPercent` | 50, 100, 100 | Share of mission and free-track coins each profile collects |
| `CrystalsPerUsdReference` | 100 | Reference rate the pack bonuses and offer values are measured against |

Every profile is assumed to claim the daily login and one first win each day. The tables below are the output of `node tools/economy-balance/economy-balance.mjs --print`; `npm run test:tools` fails when they no longer match the CSV or when the CSV has a key this document does not name. To change a value: edit the CSV, regenerate, paste.

#### Earn rate per mode

| Mode | Win | Loss | Matches per hour | Coins per hour | Hours to daily cap |
|---|---|---|---|---|---|
| Ranked 1v1 | 30 | 10 | 5.45 | 109 | 3.7 |
| Ranked 2v2 | 40 | 14 | 4.00 | 108 | 3.7 |
| Casual 1v1 | 18 | 6 | 5.45 | 65 | 6.1 |
| Casual 2v2 | 24 | 8 | 4.00 | 64 | 6.3 |

#### Earn per day by player profile

| Source | 0.5 h/day | 1 h/day | 3 h/day |
|---|---|---|---|
| Matches (ranked 1v1) | 55 | 109 | 327 |
| First win of the day | 100 | 100 | 100 |
| Daily login (7-day average) | 64 | 64 | 64 |
| Missions (daily + weekly / 7) | 103 | 206 | 206 |
| Pass free track | 8 | 17 | 17 |
| **Total per day** | **330** | **496** | **714** |

#### Crystal packs

| Product | USD reference | Crystals | Bonus over reference rate | Crystals per USD |
|---|---|---|---|---|
| crystals_tier1 | 0.99 | 100 | 1% | 101 |
| crystals_tier2 | 4.99 | 550 | 10% | 110 |
| crystals_tier3 | 9.99 | 1200 | 20% | 120 |
| crystals_tier4 | 19.99 | 2600 | 30% | 130 |
| crystals_tier5 | 49.99 | 7000 | 40% | 140 |

#### Guide prices by rarity and type (coins)

| Rarity | `avatar` | `character` | `frame` | `table` | `emoji` | `action` |
|---|---|---|---|---|---|---|
| `common` | 400 | 600 | 400 | 600 | 200 | 200 |
| `rare` | 1500 | 2250 | 1500 | 2250 | 750 | 750 |
| `epic` | 3500 | 5250 | 3500 | 5250 | 1750 | 1750 |
| `legendary` | 12000 | 18000 | 12000 | 18000 | 6000 | 6000 |
| `mythic` | 30000 | 45000 | 30000 | 45000 | 15000 | 15000 |

#### Price against earn rate

| Rarity | Guide price in coins (avatar, frame) | Days at 0.5 h/day | Days at 1 h/day | Days at 3 h/day | Guide price in crystals | USD for the crystals (best to smallest pack) |
|---|---|---|---|---|---|---|
| `common` | 400 | 1.2 | 0.8 | 0.6 | 80 | 0.57 to 0.79 |
| `rare` | 1500 | 4.5 | 3.0 | 2.1 | 300 | 2.14 to 2.97 |
| `epic` | 3500 | 10.6 | 7.1 | 4.9 | 700 | 5.00 to 6.93 |
| `legendary` | 12000 | 36.4 | 24.2 | 16.8 | 2400 | 17.14 to 23.76 |
| `mythic` | 30000 | 90.9 | 60.5 | 42.0 | 6000 | 42.85 to 59.40 |

#### Pass and offer value

| Product | USD reference | Crystal value (crystals and levels, items excluded) | USD at reference rate | Value multiple |
|---|---|---|---|---|
| pass_premium | 4.99 | 600 | 6.00 | 1.20 |
| pass_premium_plus | 9.99 | 1400 | 14.00 | 1.40 |
| offer_starter | 1.99 | 300 | 3.00 | 1.51 |

Reading the sheet:
- A regular player (1 h/day) affords a `rare` avatar every 3 days, an `epic` in a week, a `legendary` in about 3.5 weeks and a `mythic` in about two months.
- One hour of ranked play earns about 109 coins, a fifth of a chest.
- A light player gets about two thirds of a regular player's coins for half the play time, because login, first win and missions do not scale with hours.
- The heavy player reaches `DailyMatchCoinCap` after 3.7 hours of ranked play.

## 12. Config

Every key and its proposed value is in `docs/economy/balance.csv`. Keys of kind `config` are admin config (ADR 0006); `model` keys are never read by the server. Used here and defined elsewhere: `RefundWeight` (`docs/anti-cheat.md`), `NicknameRenameCooldownDays` (`docs/localization.md`), the purchase kill switch (#210).

## 13. Owner decisions

Decided by the owner on 2026-10-26. Values stay admin config and are tuned with closed-beta data (#193).

| Id | Question | Decision |
|---|---|---|
| OQ-1 | Approve the values in `docs/economy/balance.csv` | As listed |
| OQ-2 | Paid randomised items | None (EP-3). Random contents only in chests bought with coins (OQ-12) |
| OQ-3 | What coins, crystals and money may buy | Cosmetics, chests (coins only), pass levels, premium track, rename only (EP-2) |
| OQ-4 | Casual queue rewards | About 60% of ranked per hour; half when a seat was bot-filled at start; custom rooms pay nothing |
| OQ-5 | Premium pass for coins | No, real money only (3.2) |
| OQ-6 | `mythic` items in the shop | Yes, time-limited offers only (3.1) |
| OQ-7 | Starter offer and first-purchase bonus | Both (7.3) |
| OQ-8 | Market-specific price overrides, for example a lower Turkey price | None at launch; store equalisation only (7.4) |
| OQ-9 | Answer App Store consumption requests during refund review (sends play data to Apple) | Not at launch; sending it later needs a consent purpose first (`docs/privacy.md` 12.2, OQ-13) |
| OQ-10 | Missed daily login day | Pauses the cycle, no reset (2.3) |
| OQ-11 | Currencies (2026-11-01) | Two: the coin, earned by play and never sold, and the crystal, from real money only; no conversion either way (EP-1) |
| OQ-12 | Chests (2026-11-01) | Bought with coins only; `common` and `rare` items, `rare` being the middle tier; odds and pity shown before purchase (3.4) |
| OQ-13 | Currency names (2026-11-01) | Coin (AZ Sikkə, TR Coin, RU Монета) and Crystal (AZ, TR Kristal, RU Кристалл). Not "gem": its Azerbaijani and Turkish word (`daş`, `taş`) is also the domino tile |
| OQ-14 | Daily login reward and economy day (2026-11-01) | Coins only; 00:00 UTC for every player, no per-player reset time zone (2.3) |
