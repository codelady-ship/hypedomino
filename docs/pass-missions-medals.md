# Pass, missions and medals

The seasonal pass with a free and a premium track, daily, weekly, achievement and event missions, and the medal collection. The server alone computes and stores everything here (ADR 0003, ADR 0007); the client sends intents and shows what the server sends (`docs/anti-cheat.md` NT-1 to NT-12).

**Status: approved by the owner on 2026-10-26 (section 11).** Every number is a launch default to be tuned with closed-beta data (#193). Values marked *economy* are defined in `docs/economy.md` and only repeated here. Every number is admin config (ADR 0006): read from the published config snapshot, recorded with the config version each match and grant used, and changed without a release. Names in `CamelCase` code font are config keys (section 9).

Related: `docs/economy.md` (coins, reward bundle, ledger, pass products), `docs/rarity.md` (tiers), `docs/cosmetics.md` (sources `pass` and `mission`), `docs/ranking.md` (seasons, leaderboard emblems), `docs/rules.md` R-53 to R-56 (ranked, custom, casual), `docs/penalties.md` sections 5 and 6, `docs/anti-cheat.md` (V-7, V-8, V-9, V-11), `docs/localization.md`, `docs/ui-motion.md`, `docs/privacy.md` (D-16, section 5). Implementation: Epic 11 #122 (section 10).

## 1. Principles

| Id | Rule |
|---|---|
| PM-1 | Server authority. XP, levels, mission progress, completions, medals and grants are computed and stored by the server from server-recorded matches. The client sends intents only: buy levels, reroll a mission, set the medal showcase, acknowledge a reveal (NT-1). |
| PM-2 | Server time. Season windows, mission periods, rerolls and expiries use server UTC and the economy day of `docs/economy.md` section 2 (NT-6, #208). The device clock only formats countdowns the server sent. |
| PM-3 | Grants are automatic. A reward is granted by the server the moment it is earned. Nothing waits for a claim, so nothing expires unclaimed (section 6). |
| PM-4 | Every grant is a reward bundle (`docs/economy.md` section 4) applied once under its idempotency key (`docs/economy.md` section 5). Replaying a match end, purchase or intent changes nothing (V-8). |
| PM-5 | Medals are earned by play only. No medal is on the premium track, in a shop offer, in a promo code or reachable by bought levels (section 5.2). |
| PM-6 | No randomness in what a reward is. Every pass level, mission slot and medal shows its exact reward before it is earned (EP-3). Mission selection is random per player (section 4.4), but every mission of a slot pays the same. |
| PM-7 | No pay-to-win and no pay-to-progress missions. A mission or medal condition never depends on a purchase, an owned item or an equipped item (EP-2). |

## 2. Pass season

### 2.1 Window

- A pass season is created in the admin panel (#140) with an id `season_<n>`, a start and an end in server UTC.
- Default window: the same start and end as the ranked season (`docs/ranking.md` 7.1, `SeasonLengthMonths` 3 calendar months, 89 to 92 days). One season boundary for the whole game keeps the theme, the rank rewards and the pass reset on one date, and matches the 90-day model of `docs/economy.md` section 11 (`ModelPassSeasonDays`).
- Pass seasons never overlap. A gap is allowed; during a gap pass XP is dropped and the grant records that (`docs/economy.md` section 4).
- Frozen while the season runs: `PassLevels`, the XP curve (2.2) and which bundle sits at which level of each track (2.4). The only change allowed to a running track is replacing a `disabled` cosmetic (`docs/cosmetics.md` section 6) with another item of the same type and rarity, with an audited reason. XP source values (2.3) may change mid-season and apply to matches that end after the publish.
- Every account is in the active season from its first session in it, at level 1 with 0 XP. Level and XP never carry over to the next season.

### 2.2 Levels and XP curve

`PassLevels` (100) levels. XP to go from level `L` to `L + 1`, for `L` = 1 to `PassLevels` - 1:

```
xpToNext(L) = min(PassXpBase + PassXpStep x (L - 1), PassXpMax)
```

Defaults `PassXpBase` 300, `PassXpStep` 50, `PassXpMax` 1000: early levels are fast, from level 15 on every level costs the same. The XP total to reach level `n` is the sum of `xpToNext` for levels 1 to `n` - 1:

| Levels | XP total to reach level `n` |
|---|---|
| `n` <= 15 | 300 x (`n` - 1) + 25 x (`n` - 1) x (`n` - 2) |
| `n` >= 15 | 8750 + 1000 x (`n` - 15) |

| Level | 2 | 5 | 10 | 15 | 20 | 30 | 50 | 70 | 90 | 100 |
|---|---|---|---|---|---|---|---|---|---|---|
| XP to next | 350 | 500 | 750 | 1000 | 1000 | 1000 | 1000 | 1000 | 1000 | - |
| XP total to reach | 300 | 1500 | 4500 | 8750 | 13750 | 23750 | 43750 | 63750 | 83750 | 93750 |

- The level is a pure function of the season XP total. The server stores the XP total and derives the level.
- XP earned at level `PassLevels` is recorded and has no effect. There are no levels or rewards past the last level.
- Pacing per player profile is in section 8.1.

### 2.3 XP sources

| Source | XP | Rule |
|---|---|---|
| Completed 1v1 match | `PassMatchXp1v1` win 100, loss 60 | Ranked or casual; section 3 |
| Completed 2v2 match | `PassMatchXp2v2` win 140, loss 80 | Ranked or casual; section 3. A 2v2 match is longer, so both modes give about 440 XP per hour (section 8.1) |
| Daily mission | Slot XP, `DailyMissionSlotXp` 100, 150, 200 | Section 4.4 |
| Weekly mission | Slot XP, `WeeklyMissionSlotXp` 600 each | Section 4.4 |
| Event mission | From the event's bundles, at most `EventMissionXpBudget` (3000) per event in total | Section 4.6 |
| Bundle `passXp` | As set on the bundle | Any other bundle the admin attaches XP to, for example a promo campaign |
| Premium plus | `PassPremiumPlusLevels` (20, *economy*) levels of XP | Section 2.5 |
| Level purchase | The XP of the bought levels | Section 2.6 |

- Match XP has no score-margin or length bonus, as with match coins (`docs/economy.md` 2.1).
- `PassDailyMatchXpCap` (2000, about 4.6 hours of play) limits match XP per account per economy day. A match that would cross it is cut to the remainder. Mission, event, purchase and bundle XP are outside the cap. The cap is independent of `DailyMatchCoinCap`: after the coin cap a match still gives XP until the XP cap.
- Achievement missions give no XP: a new account clearing many of them at once would distort the season curve.
- A match's XP goes to the season active when the match ends (the grant time of `docs/economy.md` section 4), even if it started in the previous season.

### 2.4 Tracks and rewards

- Each level has one free-track bundle and one premium-track bundle. A free-track level may be empty; a premium-track level may not.
- Coin budgets per season, *economy*: `PassFreeTrackCoins` 1500 coins on the free track, `PassPremiumTrackCrystals` 600 crystals on the premium track: the premium track is bought with real money, so it pays crystals, never coins (`docs/economy.md` EP-1). Admin publish rejects a season above either.
- Track bundles carry `coins` and `cosmetics`. They never carry `medals` (PM-5) and never `passXp` (a level that grants levels would make the curve meaningless).
- Track cosmetics list the source `pass` with ref `pass.<seasonId>.<track>.level_<n>`, for example `pass.season_1.premium.level_30` (`docs/cosmetics.md` section 5).
- Level 1 rewards are granted when the account enters the season (free) and when premium is unlocked (premium), so the premium track gives its headline item at once.
- Pass tracks carry no `mythic` item, so `mythic` stays the tier of time-limited offers (`docs/economy.md` 3.1; section 11 OQ-10).
- The sample season is section 8.2.

### 2.5 Premium and late purchase

Products, *economy* (`docs/economy.md` 7.2): `pass_premium` (`PassPremiumUsd` 4.99) unlocks the premium track of the current season; `pass_premium_plus` (`PassPremiumPlusUsd` 9.99) unlocks it and adds `PassPremiumPlusLevels` (20) levels. The premium track is never sold for coins.

- Purchase flow, receipt validation and binding to the season active at the store's purchase time: `docs/economy.md` 7.5 and #125. A second premium purchase in the same season is rejected before the store flow starts.
- **Late purchase.** When premium unlocks at level `L`, the server grants the premium bundle of every level 1 to `L` at once, then continues normally. Each level is its own grant with key `pass:<seasonId>:premium:<level>`, so an interrupted batch resumes and a repeated one grants nothing twice.
- `pass_premium_plus` adds the XP from the current XP total to the total of level `L + 20`, keeping progress inside the current level. Levels crossed grant both tracks.
- `pass_premium_plus` is offered only while the account's level is at most `PassLevels` - `PassPremiumPlusLevels` (80), so its levels are never wasted. Above that only `pass_premium` is offered. If the account levels up between the buy intent and validation, levels past `PassLevels` are paid out as `PassLevelCrystalPrice` (40, *economy*) crystals each in the purchase grant: a real-money product never pays coins (`docs/economy.md` EP-1).
- A validated purchase is always granted, even after the season ended (`docs/economy.md` 7.5 step 7). Its premium bundles for the levels reached are granted at validation.
- Refund (`docs/economy.md` section 9): the premium track and every premium reward granted are revoked. For `pass_premium_plus` the XP it added is also removed, the level is derived again, and both tracks' rewards of levels no longer reached are revoked the same way: their cosmetics are removed and their crystals debited, while free-track coins are kept, since a refund never touches coins (`docs/economy.md` section 9).

### 2.6 Level purchase

- Intent with a count `k`, a currency (`coin`, the default, or `crystal`) and a client request id. Price `k` x `PassLevelCoinPrice` (200, *economy*) coins or `k` x `PassLevelCrystalPrice` (40, *economy*) crystals, ledger reason `pass_level_purchase`, key `spend:<requestId>` (`docs/economy.md` 3.2, section 5).
- The server rejects `k` < 1, a count that would pass `PassLevels`, a balance below the price, and any purchase outside the season window.
- A bought level adds the XP from the current total to the total of level `L + k`, keeping progress inside the current level. Levels crossed grant the free track, and the premium track if unlocked.
- Coins and crystals are not refunded for levels, and a later premium unlock grants the premium rewards of bought levels as in 2.5.

### 2.7 Season end

At the season's end (server time):
1. XP stops counting for the season; level purchase and pass products close.
2. The idempotent season-end job (#123) checks, for every account with progress, that every level reached has its grants on every unlocked track, and grants any that are missing. This is the safety net behind PM-3; no reward is lost.
3. The season's progress stays readable in the pass history. Coins and items granted stay.

## 3. What counts

| Match | Pass XP | Daily and weekly missions | Achievement and event missions, medals |
|---|---|---|---|
| Ranked 1v1, 2v2 (R-53) | Full | Yes | Yes |
| Casual 1v1, 2v2 (R-56), all seats human at start | Full | Yes | Yes |
| Casual with a bot-filled seat at start (#48) | `BotSeatRewardPercent` (50, *economy*) of match XP, rounded, halves away from zero | Yes | No |
| Custom room (R-55) | None | No | No |
| Cancelled by the server | None | No | No |

- Only completed, server-recorded matches count. Progress is read from the match record, never from a client report.
- A bot replacing an AFK or leaving player during the match (R-50 to R-52) reduces nobody's XP or progress.
- AFK and leave offenders in ranked and casual matches get no pass XP and no mission or medal progress from that match; AFK returned keeps them (`docs/penalties.md` sections 5 and 6). Partners and opponents count normally.
- Custom rooms count for nothing because players pick the teams, so results can be arranged (as for coins, `docs/economy.md` 2.1).
- Bot-filled casual matches count for daily and weekly missions because their rewards are capped per period anyway (4.4), so a bot table only makes the same reward faster; they never count for achievements and medals, which mark play against people.

### 3.1 Anti-farm

| Control | Rule |
|---|---|
| Caps | `PassDailyMatchXpCap`; mission coin and XP budgets per period (4.4); a finite level table; idempotent keys per period (`docs/economy.md` section 5) |
| Queues | Custom rooms and cancelled matches count for nothing; bot-filled casual is reduced as above |
| Offenders | Ranked and casual: no XP or progress from the offending match (`docs/penalties.md` sections 5 and 6) |
| No transfers | XP, levels, missions and medals cannot be moved between accounts (EP-5), so farm accounts cannot feed a main account |
| Signals | Repeated meetings of the same accounts, losses to the same opponent and shared devices feed `CollusionWeight` (V-9, #183, #110). Mission completions and XP per account and day are reported to #110; the account daily earn alert of `docs/economy.md` section 10 already includes mission coins |
| Response | Confirmed farming or collusion: an admin removes the XP, levels, mission rewards and medals gained from the affected matches, with an audited reason (#209). Grants of levels no longer reached are revoked as in 2.5 |
| Time | Every window is server UTC; clock changes gain nothing (V-7) |

## 4. Missions

### 4.1 Types

| Type | Period | Rewards | Assigned |
|---|---|---|---|
| `daily` | Economy day (`docs/economy.md` section 2) | Slot coins and XP (4.4) | `DailyMissionSlots` (3) per player per day |
| `weekly` | Economy week: Monday at the economy day boundary to the next | Slot coins and XP (4.4) | `WeeklyMissionSlots` (4) per player per week |
| `achievement` | None; once per account | Bundle: coins, cosmetics, medals; no XP (2.3) | All live achievements, in chains |
| `event` | The live event's window (#146) | Bundle: coins, cosmetics, medals, XP | All missions of a running event |

### 4.2 Definition

Missions are created in admin without code changes (#127, #141).

| Field | Rule |
|---|---|
| `id` | `mission.<slug>`, slug `[a-z0-9_]+`. Stable, never reused |
| `revision` | Integer >= 1, increases on every published change |
| `type` | Section 4.1 |
| `nameKey` | `mission.<slug>.name`, an ICU message with the `{target}` placeholder and plural forms (`docs/localization.md` 4.2, 4.3) |
| `event` | A progress event of 4.3 |
| `target` | Integer >= 1: the count to reach |
| `filters` | Optional: `queue` (`ranked`, `casual`), `mode` (`1v1`, `2v2`). Empty means every counting match (section 3) |
| `slot` | `daily` and `weekly` only: the slot number (4.4) |
| `weight` | `daily` and `weekly` only: integer >= 1, selection weight within the slot |
| `bundle` | `achievement` and `event` only: the reward bundle id |
| `next` | `achievement` only, optional: the next mission of the chain |
| `eventId` | `event` only: the live event it belongs to |
| `availability` | `live` or `retired`. A retired mission is no longer assigned; progress already made is kept for achievements |

- A cosmetic or medal granted by a mission lists the source `mission` with the mission id (`docs/cosmetics.md` V-15).
- No condition can require a custom room, a purchase, or an owned or equipped item (PM-7).

### 4.3 Progress events

All come from the server's match record of a counting match (section 3), attributed to the player.

| Event | Adds per match |
|---|---|
| `match_completed` | 1 |
| `match_won` | 1 if the player's team won the match (R-46) |
| `hand_won` | Hands won by the player's team (R-35, R-37) |
| `domino` | Hands the player ended by placing their last tile (R-32) |
| `board_points` | Board points scored by the player's own placements (R-25 to R-31) |
| `double_chain` | Turns in which the player placed two or more doubles (R-31) |

Adding an event is a server change and a change to this table. Progress applies to the missions active when the match ends.

### 4.4 Daily and weekly slots, rotation and reroll

Each slot has a fixed reward; every mission in the slot's pool pays it, so selection and reroll never change what a player can earn.

| Slot | Coins | XP | Intended effort |
|---|---|---|---|
| Daily 1 | 30 | 100 | About 1 match |
| Daily 2 | 40 | 150 | About 3 matches |
| Daily 3 | 50 | 200 | About 5 matches (one hour) |
| Weekly 1 to 4 | 150 each | 600 each | About 15 matches each. One match counts for every mission it fits, so all four fit in a week of one hour a day (about 38 matches) |

- Keys: `DailyMissionSlotCoins` (30, 40, 50), `DailyMissionSlotXp` (100, 150, 200), `WeeklyMissionSlotCoins` (150 x 4), `WeeklyMissionSlotXp` (600 x 4).
- Budgets: slot coins sum to at most `DailyMissionCoinBudget` (120, *economy*) and `WeeklyMissionCoinBudget` (600, *economy*); slot XP sums to at most `DailyMissionXpBudget` (450) and `WeeklyMissionXpBudget` (2400). Admin publish rejects a slot set above a budget.
- Assignment: at the first moment the server needs the player's missions in a period (session start or match end), it draws one live mission per slot by weight with the server's cryptographically secure random source, avoiding the mission the slot had in the previous period when the pool has another. The draw is stored; it happens once per player and period.
- Expiry: at the end of the period unfinished missions and their progress end. Completed missions were already granted (PM-3).
- Reroll: `DailyMissionRerolls` (1) per economy day and `WeeklyMissionRerolls` (1) per week, free, by intent. It replaces one uncompleted mission with another live mission of the same slot that the player has not had in this period; progress on the replaced mission is lost. A reroll is rejected when the slot has no such mission. Rerolls are never sold and do not accumulate.
- Grant key: `mission:<missionId>:<period>`, the period being the economy day (`2026-11-03`) or ISO week (`2026-W45`) at 00:00 UTC, the same for every player (`docs/economy.md` 2, #131, #208).

### 4.5 Achievements

- One-time missions, permanent, completed once per account under key `mission:<missionId>:once`. Progress counts from the account's first match, across seasons.
- Chains: completing a mission shows its `next`. Progress of the whole chain runs from the start, so a player past several thresholds completes them one after another at once.
- Achievement coins are outside the steady-state budgets; admin shows the total coins of all live achievements at publish and #110 reports it (`docs/economy.md` 2.4).
- Achievements are the main medal source (section 5.2).

### 4.6 Event missions

- Belong to a live event in the live-ops calendar (#146) and run only inside its window, server UTC. Grant key `mission:<missionId>:<eventId>`.
- Pass XP from all missions of one event is at most `EventMissionXpBudget` (3000, three flat levels); admin publish rejects more. Event coins count in the economy monitors (`docs/economy.md` 2.7).
- An event medal is earned through an event mission (section 5.2).

## 5. Medals

### 5.1 Definition

Managed in admin (#141); art from #178.

| Field | Rule |
|---|---|
| `id` | `medal.<slug>`, slug `[a-z0-9_]+`. Stable, never reused |
| `revision` | Integer >= 1, increases on every published change |
| `nameKey`, `descriptionKey` | `medal.<slug>.name`, `medal.<slug>.description`. The description says how the medal is earned |
| `rarityId` | A tier of `docs/rarity.md` |
| `staticAddress` | `medal/<slug>/static`. Medals are static; the locked look is #178's |
| `sources` | At least 1: `mission` (achievement or event mission id) or `rank_reward` (#120 reward id). No other source type is valid (PM-5) |
| `availability` | `live`, `retired` or `disabled`, with the meanings of `docs/cosmetics.md` section 6 |

### 5.2 Conditions

- A medal is granted only through a reward bundle (`medals` field) of a listed source: an achievement or event mission completed by play (section 4), or a rank reward at season end (#120). Its condition is the source's condition; there is no second condition system.
- A grant from a record not listed in the medal's `sources` is refused (`docs/cosmetics.md` V-15 applies to medals). This is how a broken condition never grants a medal (#129).
- Pass tracks, shop offers and promo codes never carry medals, and bought levels or premium plus levels never advance a mission, so a medal cannot be bought.
- Each medal is held once. A bundle that grants an owned medal skips it, with no compensation (`docs/economy.md` 2.7).
- Medals are kept for the account's lifetime. An admin revokes one only with an audited reason, for example confirmed farming (3.1).

### 5.3 Rarity

The admin sets each medal's `rarityId` from the share of players active in the season who are expected to earn it:

| `rarityId` | Expected share of season-active players |
|---|---|
| `common` | 25% or more |
| `rare` | 10% to 25% |
| `epic` | 2% to 10% |
| `legendary` | Under 2% |
| `mythic` | Not used for medals at launch |

- The admin medal list shows each medal's measured earn share from #147 next to its rarity, so a mis-tiered medal is visible after closed beta (#193).
- Rarity marks (diamond, label, glow) follow `docs/rarity.md` 2.3 as for an item card.

### 5.4 Collection and profile

- Collection grid (#130): live medals and owned retired medals, grouped by source (achievements, events, rank rewards), then by rarity ordinal descending, then by id. Unowned retired and all `disabled` medals are not shown.
- Locked medal: the locked look from #178, its name, its description, and for a mission source the current progress and target from the server.
- Earned medal: its art, rarity marks and the server date earned.
- The collection header shows earned / shown medals. Emblems follow in their own section (5.5).
- Profile showcase: a player chooses up to `ProfileMedalSlots` (3) owned medals and their order, as an intent with medal ids. The server accepts only owned medals, stores the showcase, and other players see the server's copy (NT-9). A revoked medal leaves the showcase at once. The medal showcase sits next to the emblem showcase on the public profile (`docs/privacy.md` section 5).
- Medals have no effect in a match and are not shown at the table.

### 5.5 Emblems

- Leaderboard emblems (`docs/ranking.md` 8.3, 8.4) stay a ranking system: awarded by the season-end job from frozen boards, recorded with mode, season, board and bracket, revoked only by an admin, shown in their own profile showcase (`ProfileEmblemSlots`, 3). They have no rarity, no sources and no bundle, and are not medals.
- The medal collection screen shows emblems in their own section after the medals, newest season first, with the same art (#178). Emblems do not count toward the medal total.
- Reason: emblems are positional season records with board and bracket fields; making each board, bracket and season a catalogue medal would add thousands of entries for no player-visible gain. One screen, two systems.

## 6. Grants, claim and reveal

- **No claim step.** Pass levels, mission completions and medals are granted by the server in the transaction that records the earning event (match end, purchase, level purchase, season end). A reward is never lost to a missed claim, a period end or a season end; the season-end job (2.7) is the safety net for defects.
- Ordering at match end: match XP, then mission progress, then each completed mission's bundle, then each level crossed. Each step is idempotent on its own key.
- Match XP is applied by the game server's relay on the `xp` outbox topic (`HD_OUTBOX_POLL`, #440): `pass.Store.Publish` runs `AwardMatch` for every player of the `matchstore.Outcome`. Mission progress comes the same way from the `missions` topic: `mission.Store.Publish` runs `ApplyMatch` per player with the hands won, dominoes, board points and double chains the outcome carries. The two topics are relayed independently, so mission progress may land before the match XP; each step stays idempotent and the end state is the same. Matchmade rooms write the outcome when they end (`docs/matchmaking.md` section 4); hands won, dominoes, board points and double chains are counted from the replay timeline (`replay.Timeline.Stats`).
- The server keeps a list of grants the client has not presented yet and sends it with the state. The client presents them as reveals and sends an acknowledge intent with the last grant id shown. Acknowledging changes only what is still to be presented, never ownership.
- Notices are `key` plus typed parameters (`docs/localization.md` section 3); items, medals and missions are sent as ids.

### 6.1 Reward reveal

Uses existing tokens of `docs/ui-motion.md` and `docs/rarity.md`; no new duration or colour.

| Step | Presentation |
|---|---|
| Card | Panel enter (`docs/ui-motion.md` section 5: `moderate`, `enter`) |
| Item or medal | Pops in with `fast` and `overshoot`; rarity halo behind it, diamond and tier label as in `docs/rarity.md` 2.3 (reward reveal row), flow for tiers with flow |
| Coins and XP | Count up over `slow` with `standard`; the pass XP bar fills over `slow` |
| Level reached | Banner entrance with `emphasis` |

- Several grants queue one card each. A tap jumps the current card to its end state; a "collect all" button jumps the whole queue (`docs/ui-motion.md` section 6 rule 4).
- Reduce motion and effect levels follow `docs/ui-motion.md` (pops become `fast` fades, flow rests).
- Reveals run outside the match scene and add no particles. Adding any particle effect is a `docs/ui-motion.md` change with its budgets (section 7 there).
- Nothing in a reveal suggests chance: no chest imagery (chests exist only in the shop, `docs/economy.md` 3.4), reels, wheels or spinning reveals, and the reward shown is the one already granted (`docs/localization.md` section 6, EP-3).

## 7. Data and idempotency

| Record | Holds | Unique on |
|---|---|---|
| Pass season | Id, window, `PassLevels`, curve values, the level-to-bundle table per track, config version | Season id |
| Pass progress | Account, season, XP total, premium flag with its purchase ref | Account, season |
| XP event | Account, season, amount, source type, source ref, idempotency key, server time | Account, key |
| Mission assignment | Account, mission, period, slot, progress, completed time | Account, mission, period |
| Medal ownership | Account, medal, server time, source type and ref, grant key | Account, medal |

- XP keys reuse the keys of the grant that carries the XP: `match:<matchId>`, `mission:<missionId>:<period>`, `spend:<requestId>`, `purchase:<store>:<transactionId>`. Track grants use `pass:<seasonId>:<track>:<level>` (`docs/economy.md` section 5).
- Progress, missions and medals are personal data D-16 (`docs/privacy.md`), kept for the account's lifetime.

## 8. Sample season

Season `season_1`, 90 days, the defaults above. Item ids are illustrative; each item is created in the catalogue with the `pass` source of its level (2.4).

### 8.1 Pacing

Inputs from `docs/economy.md` section 11: ranked 1v1, 10-minute matches plus 1 minute between, 50% wins (436 XP per hour; 2v2 gives 440). The light profile completes half the missions, the others all.

| Profile | XP per day (matches + missions) | Level on day 30 | Level on day 60 | Level on day 90 | Day the last level is reached | Free-track coins |
|---|---|---|---|---|---|---|
| Light, 0.5 h/day | 218 + 396 = 615 | 24 | 43 | 61 | Not reached | 930 |
| Regular, 1 h/day | 436 + 793 = 1229 | 43 | 80 | 100 | 77 | 1500 |
| Heavy, 3 h/day | 1309 + 793 = 2102 | 69 | 100 | 100 | 45 | 1500 |

- The regular player finishes about two weeks before the end, so a few missed days do not cost the last level.
- The light player collects 62% of the free-track coins; the economy model assumes 50% (`ModelProfileMissionPercent`), about 2 coins a day more than the sheet.
- `pass_premium_plus` skips 20 flat levels, 20000 XP, about 16 days of regular play. Buying from level 61 to 100 costs 39 x 200 = 7800 coins.

### 8.2 Levels and rewards

Free track: 30 coins on every odd level (1500), a cosmetic on every tenth level. Premium track: 6 crystals on every level (600), a cosmetic on level 1 and every tenth level. Cosmetics at guide prices (`docs/economy.md` 3.1): free 11850 coins, premium 45000 coins. `node tools/pass-season/pass-season.mjs` checks the table against the curve (2.2) and the track budgets in `docs/economy/balance.csv` (2.4); CI runs it through `npm run test:tools`.

| Level | XP total to reach | Free | Premium |
|---|---|---|---|
| 1 | 0 | 30 coins | 6 crystals + `character.s1_navigator` (`epic`) |
| 2 | 300 | - | 6 crystals |
| 3 | 650 | 30 coins | 6 crystals |
| 4 | 1050 | - | 6 crystals |
| 5 | 1500 | 30 coins | 6 crystals |
| 6 | 2000 | - | 6 crystals |
| 7 | 2550 | 30 coins | 6 crystals |
| 8 | 3150 | - | 6 crystals |
| 9 | 3800 | 30 coins | 6 crystals |
| 10 | 4500 | `emoji.s1_wave` (`common`) | 6 crystals + `avatar.s1_pirate` (`rare`) |
| 11 | 5250 | 30 coins | 6 crystals |
| 12 | 6050 | - | 6 crystals |
| 13 | 6900 | 30 coins | 6 crystals |
| 14 | 7800 | - | 6 crystals |
| 15 | 8750 | 30 coins | 6 crystals |
| 16 | 9750 | - | 6 crystals |
| 17 | 10750 | 30 coins | 6 crystals |
| 18 | 11750 | - | 6 crystals |
| 19 | 12750 | 30 coins | 6 crystals |
| 20 | 13750 | `avatar.s1_sailor` (`common`) | 6 crystals + `emoji.s1_wink` (`rare`) |
| 21 | 14750 | 30 coins | 6 crystals |
| 22 | 15750 | - | 6 crystals |
| 23 | 16750 | 30 coins | 6 crystals |
| 24 | 17750 | - | 6 crystals |
| 25 | 18750 | 30 coins | 6 crystals |
| 26 | 19750 | - | 6 crystals |
| 27 | 20750 | 30 coins | 6 crystals |
| 28 | 21750 | - | 6 crystals |
| 29 | 22750 | 30 coins | 6 crystals |
| 30 | 23750 | `frame.s1_rope` (`common`) | 6 crystals + `frame.s1_wave` (`rare`) |
| 31 | 24750 | 30 coins | 6 crystals |
| 32 | 25750 | - | 6 crystals |
| 33 | 26750 | 30 coins | 6 crystals |
| 34 | 27750 | - | 6 crystals |
| 35 | 28750 | 30 coins | 6 crystals |
| 36 | 29750 | - | 6 crystals |
| 37 | 30750 | 30 coins | 6 crystals |
| 38 | 31750 | - | 6 crystals |
| 39 | 32750 | 30 coins | 6 crystals |
| 40 | 33750 | `emoji.s1_thumbs` (`rare`) | 6 crystals + `table.s1_reef` (`rare`) |
| 41 | 34750 | 30 coins | 6 crystals |
| 42 | 35750 | - | 6 crystals |
| 43 | 36750 | 30 coins | 6 crystals |
| 44 | 37750 | - | 6 crystals |
| 45 | 38750 | 30 coins | 6 crystals |
| 46 | 39750 | - | 6 crystals |
| 47 | 40750 | 30 coins | 6 crystals |
| 48 | 41750 | - | 6 crystals |
| 49 | 42750 | 30 coins | 6 crystals |
| 50 | 43750 | `avatar.s1_captain` (`rare`) | 6 crystals + `avatar.s1_admiral` (`epic`) |
| 51 | 44750 | 30 coins | 6 crystals |
| 52 | 45750 | - | 6 crystals |
| 53 | 46750 | 30 coins | 6 crystals |
| 54 | 47750 | - | 6 crystals |
| 55 | 48750 | 30 coins | 6 crystals |
| 56 | 49750 | - | 6 crystals |
| 57 | 50750 | 30 coins | 6 crystals |
| 58 | 51750 | - | 6 crystals |
| 59 | 52750 | 30 coins | 6 crystals |
| 60 | 53750 | `table.s1_harbor` (`common`) | 6 crystals + `emoji.s1_storm` (`epic`) |
| 61 | 54750 | 30 coins | 6 crystals |
| 62 | 55750 | - | 6 crystals |
| 63 | 56750 | 30 coins | 6 crystals |
| 64 | 57750 | - | 6 crystals |
| 65 | 58750 | 30 coins | 6 crystals |
| 66 | 59750 | - | 6 crystals |
| 67 | 60750 | 30 coins | 6 crystals |
| 68 | 61750 | - | 6 crystals |
| 69 | 62750 | 30 coins | 6 crystals |
| 70 | 63750 | `frame.s1_anchor` (`rare`) | 6 crystals + `frame.s1_kraken` (`epic`) |
| 71 | 64750 | 30 coins | 6 crystals |
| 72 | 65750 | - | 6 crystals |
| 73 | 66750 | 30 coins | 6 crystals |
| 74 | 67750 | - | 6 crystals |
| 75 | 68750 | 30 coins | 6 crystals |
| 76 | 69750 | - | 6 crystals |
| 77 | 70750 | 30 coins | 6 crystals |
| 78 | 71750 | - | 6 crystals |
| 79 | 72750 | 30 coins | 6 crystals |
| 80 | 73750 | `emoji.s1_cheer` (`rare`) | 6 crystals + `table.s1_deep_sea` (`epic`) |
| 81 | 74750 | 30 coins | 6 crystals |
| 82 | 75750 | - | 6 crystals |
| 83 | 76750 | 30 coins | 6 crystals |
| 84 | 77750 | - | 6 crystals |
| 85 | 78750 | 30 coins | 6 crystals |
| 86 | 79750 | - | 6 crystals |
| 87 | 80750 | 30 coins | 6 crystals |
| 88 | 81750 | - | 6 crystals |
| 89 | 82750 | 30 coins | 6 crystals |
| 90 | 83750 | `table.s1_lighthouse` (`rare`) | 6 crystals + `emoji.s1_treasure_map` (`epic`) |
| 91 | 84750 | 30 coins | 6 crystals |
| 92 | 85750 | - | 6 crystals |
| 93 | 86750 | 30 coins | 6 crystals |
| 94 | 87750 | - | 6 crystals |
| 95 | 88750 | 30 coins | 6 crystals |
| 96 | 89750 | - | 6 crystals |
| 97 | 90750 | 30 coins | 6 crystals |
| 98 | 91750 | - | 6 crystals |
| 99 | 92750 | 30 coins | 6 crystals |
| 100 | 93750 | `frame.s1_compass` (`epic`) | 6 crystals + `character.s1_sea_queen` (`legendary`) |

### 8.3 Missions

Daily pool (sample; real pools are larger):

| Id | Slot | Event, target | Filters |
|---|---|---|---|
| `mission.daily_play_1` | 1 | `match_completed` 1 | - |
| `mission.daily_points_30` | 1 | `board_points` 30 | - |
| `mission.daily_hands_3` | 1 | `hand_won` 3 | - |
| `mission.daily_play_3` | 2 | `match_completed` 3 | - |
| `mission.daily_win_1` | 2 | `match_won` 1 | - |
| `mission.daily_domino_2` | 2 | `domino` 2 | - |
| `mission.daily_win_3` | 3 | `match_won` 3 | - |
| `mission.daily_points_150` | 3 | `board_points` 150 | - |
| `mission.daily_play_2v2_4` | 3 | `match_completed` 4 | `mode` 2v2 |

Weekly pool:

| Id | Slot | Event, target | Filters |
|---|---|---|---|
| `mission.weekly_play_15` | 1 | `match_completed` 15 | - |
| `mission.weekly_hands_40` | 1 | `hand_won` 40 | - |
| `mission.weekly_win_8` | 2 | `match_won` 8 | - |
| `mission.weekly_ranked_win_5` | 2 | `match_won` 5 | `queue` ranked |
| `mission.weekly_points_600` | 3 | `board_points` 600 | - |
| `mission.weekly_domino_10` | 3 | `domino` 10 | - |
| `mission.weekly_chain_3` | 4 | `double_chain` 3 | - |
| `mission.weekly_win_2v2_5` | 4 | `match_won` 5 | `mode` 2v2 |

Achievements, one chain per line; together they pay 7900 coins once per account:

| Chain | Missions, target: reward |
|---|---|
| Wins (`match_won`) | `mission.win_5_matches` 5: 100 coins, `emoji.laugh_spin`; `mission.win_50` 50: 200 coins, `medal.wins_50`; `mission.win_250` 250: 500 coins, `medal.wins_250`; `mission.win_1000` 1000: 1000 coins, `medal.wins_1000`; `mission.win_5000` 5000: 2000 coins, `medal.wins_5000` |
| Dominoes (`domino`) | `mission.domino_100` 100: 200 coins, `medal.domino_100`; `mission.domino_1000` 1000: 1000 coins, `medal.domino_1000` |
| Double chains (`double_chain`) | `mission.chain_1` 1: 100 coins, `medal.chain_first`; `mission.chain_100` 100: 1000 coins, `medal.chain_100` |
| Board points (`board_points`) | `mission.points_10000` 10000: 300 coins, `medal.points_10k`; `mission.points_100000` 100000: 1500 coins, `medal.points_100k` |

### 8.4 Medals

| Medal | Source | `rarityId` |
|---|---|---|
| `medal.wins_50` | `mission.win_50` | `common` |
| `medal.wins_250` | `mission.win_250` | `rare` |
| `medal.wins_1000` | `mission.win_1000` | `epic` |
| `medal.wins_5000` | `mission.win_5000` | `legendary` |
| `medal.domino_100` | `mission.domino_100` | `common` |
| `medal.domino_1000` | `mission.domino_1000` | `epic` |
| `medal.chain_first` | `mission.chain_1` | `common` |
| `medal.chain_100` | `mission.chain_100` | `epic` |
| `medal.points_10k` | `mission.points_10000` | `rare` |
| `medal.points_100k` | `mission.points_100000` | `legendary` |
| `medal.season1_gold` | Rank reward of `season_1`, Gold (#120, `docs/economy.md` section 4 example) | Set by #120 |

Rarities are first estimates for 5.3, checked against measured earn shares after closed beta. The example catalog `docs/pass/examples/medals.json` leaves out `medal.season1_gold` until its rarity is set.

## 9. Config

| Key | Default | Defined in | Section |
|---|---|---|---|
| `PassLevels` | 100 | This document | 2.2 |
| `PassXpBase`, `PassXpStep`, `PassXpMax` | 300, 50, 1000 | This document | 2.2 |
| `PassMatchXp1v1` | 100 win, 60 loss | This document | 2.3 |
| `PassMatchXp2v2` | 140 win, 80 loss | This document | 2.3 |
| `PassDailyMatchXpCap` | 2000 | This document | 2.3 |
| `EventMissionXpBudget` | 3000 per event | This document | 2.3, 4.6 |
| `DailyMissionSlots`, `WeeklyMissionSlots` | 3, 4 | This document | 4.1 |
| `DailyMissionSlotCoins`, `DailyMissionSlotXp` | 30, 40, 50; 100, 150, 200 | This document | 4.4 |
| `WeeklyMissionSlotCoins`, `WeeklyMissionSlotXp` | 150 x 4; 600 x 4 | This document | 4.4 |
| `DailyMissionXpBudget`, `WeeklyMissionXpBudget` | 450, 2400 | This document | 4.4 |
| `DailyMissionRerolls`, `WeeklyMissionRerolls` | 1, 1 | This document | 4.4 |
| `ProfileMedalSlots` | 3 | This document | 5.4 |
| `ProfileEmblemSlots` | 3 | `docs/ranking.md` | 5.5 |
| Medal rarity shares | Section 5.3 | This document, guidance for admins | 5.3 |
| `PassFreeTrackCoins`, `PassPremiumTrackCrystals` | 1500, 600 | `docs/economy.md` | 2.4 |
| `PassLevelCoinPrice`, `PassLevelCrystalPrice` | 200, 40 | `docs/economy.md` | 2.5, 2.6 |
| `PassPremiumUsd`, `PassPremiumPlusUsd`, `PassPremiumPlusLevels` | 4.99, 9.99, 20 | `docs/economy.md` | 2.5 |
| `DailyMissionCoinBudget`, `WeeklyMissionCoinBudget` | 120, 600 | `docs/economy.md` | 4.4 |
| `BotSeatRewardPercent` | 50 | `docs/economy.md` | 3 |

Season windows are records, not keys: they default to the ranked season's window (2.1).

Pass seasons are published as the `pass` config domain: `docs/pass/examples/pass.json` is the sample season of section 8.2 in that format (schema 1, one bundle per level and track, the curve of 2.2 per season). The same domain carries the XP values of 2.3 that may change mid-season: `matchXp1v1` and `matchXp2v2` (`win`, `loss`), `dailyMatchXpCap` and `botSeatRewardPercent`, and the `levelCoinPrice` and `levelCrystalPrice` of 2.6 (`PassLevelCoinPrice`, `PassLevelCrystalPrice`). The track budgets are `freeTrackCoins` and `premiumTrackCrystals`; a free bundle grants no crystals and a premium bundle no coins.

### 9.1 Publish validation

Admin publish (#140, #141) rejects:

| Id | Rule |
|---|---|
| PV-1 | A free track whose coins exceed `PassFreeTrackCoins`, a premium track whose crystals exceed `PassPremiumTrackCrystals`, or a premium reward with coins |
| PV-2 | A premium level without a bundle, a level outside 1 to `PassLevels`, or a track bundle with `medals` or `passXp` |
| PV-3 | `PassLevels` <= `PassPremiumPlusLevels`, or a curve value < 1, or `PassXpMax` < `PassXpBase` |
| PV-4 | Overlapping pass seasons; a change to a frozen field of a running season (2.1) |
| PV-5 | Daily or weekly slot coins or XP above their budgets (4.4), or a slot without a live mission |
| PV-6 | Event missions of one event with more XP than `EventMissionXpBudget` |
| PV-7 | A mission with an unknown event or filter, a `bundle` on a daily or weekly mission, or slot fields on another type |
| PV-8 | A medal with a source type other than `mission` or `rank_reward`, a `rarityId` not in the rarity table, or a bundle granting a medal whose `sources` do not list the bundle's record |
| PV-9 | A cosmetic on a track, mission or event whose `sources` do not list it (`docs/cosmetics.md` V-15) |
| PV-10 | A `mythic` cosmetic on a pass track |

## 10. Implementation

| Issue | Covers |
|---|---|
| #123 | Pass seasons, levels, tracks, progress, season-end job (2.1, 2.2, 2.4, 2.7, 7) |
| #124 | XP engine: sources, daily cap, idempotent XP events, late premium grants (2.3, 2.5, 3) |
| #125 | Premium and premium plus purchase, level purchase (2.5, 2.6) |
| #126 | Pass UI and reveal (6.1) |
| #127 | Mission backend: definitions, progress events, slots, rotation, reroll (4) |
| #128 | Mission UI |
| #129 | Medal system: definition, grant, showcase (5) |
| #130 | Medal UI: collection, emblem section, profile (5.4, 5.5) |
| #131 | Daily login and the 00:00 UTC economy day (`docs/economy.md` 2.3) |
| #132 | Push notifications; this spec requires none |

Elsewhere: admin #138 (bundles), #140 (pass seasons), #141 (missions and medals); grants #102, ledger #101, purchases #106; rank rewards #120 and season end #115; live events #146; analytics #147; medal art #178; anti-farm #110, #183, #209, #211; server time #208.

## 11. Owner decisions

Decided by the owner on 2026-10-26. Values stay admin config and are tuned with closed-beta data (#193).

| Id | Question | Decision |
|---|---|---|
| OQ-1 | Approve the values of section 9 | As listed |
| OQ-2 | Pass season window | Same as the ranked season (2.1) |
| OQ-3 | Level count and pace | 100 levels; a regular player (1 h/day, all missions) finishes around day 77 of 90; a light player reaches level 61 (8.1) |
| OQ-4 | Casual pass XP | Same as ranked per match; half when a seat was bot-filled at start. Coins in casual stay at about 60% of ranked (`docs/economy.md` OQ-4) |
| OQ-5 | Claim model | Automatic grants, no claim step, nothing expires unclaimed (section 6) |
| OQ-6 | Bot-filled casual and missions | Counts for daily and weekly missions, not for achievements and medals (section 3) |
| OQ-7 | Medals and money | Medals only from play: achievements, event missions, rank rewards; never on the premium track or for sale (PM-5) |
| OQ-8 | Medals and emblems | Separate systems with their own profile showcase: `ProfileEmblemSlots` (3) emblems and `ProfileMedalSlots` (3) medals; emblems have their own section of the medal collection (5.4, 5.5) |
| OQ-9 | Catch-up for players who join late in a season | None at launch; level purchase and premium plus cover it |
| OQ-10 | `mythic` items on pass tracks | None; the sample tops out at one `legendary` per track season (8.2) |
