# Ranking

Ratings, ranks, LP, placement, seasons, soft reset and leaderboards for ranked play. The server alone computes and stores everything here; the client only shows what the server sends and never sees MMR. Every name in `CamelCase` code font is admin config (ADR 0006, section 10). Every value shown is a proposed default that admins can change, not a constant.

Related: `docs/rules.md` R-43 to R-55 (match result, timer, AFK, leavers, ranked and custom play), `docs/penalties.md` (AFK and leaver offenses), ADR 0005 (storage), ADR 0007 (server time).

## 1. Scope

- Applies to ranked 1v1 and ranked 2v2 (R-53). Each mode has its own MMR, rank, LP, placement, season result and leaderboards. A result in one mode never changes the other.
- A ranked match is a match formed by the ranked matchmaking queue (#48, #118). In ranked 2v2 the partner is random (R-54), so there are no premade teams to correct for.
- Custom rooms (R-55) are unranked: nothing in this document is read or written by them.
- A match cancelled by the server (server fault, region outage) changes nothing for anyone.
- A match counts toward the season in which it started.
- Applying a match result is idempotent by match id: replaying the same match end never changes MMR or LP twice.
- Matchmaking quality (MMR window, wait expansion) is #48 and #118. Abuse detection (win-trading, boosting) is #121. Season rewards are #120.

## 2. Terms

| Term | Meaning |
|---|---|
| MMR | Hidden skill rating per player per mode, an integer. Carried across seasons. |
| Tier | Iron, Bronze, Silver, Gold, Platinum, Emerald, Diamond, Master, Grandmaster, Challenger, lowest to highest |
| Division | IV, III, II, I inside a tier, lowest to highest. Only the first 7 tiers have divisions. |
| Rank | A tier plus a division (Gold II), or a tier alone for Master and above |
| LP | Visible points inside a rank. Win adds, loss subtracts. |
| Rank index | Position of a rank: Iron IV = 0, Iron III = 1, ... Diamond I = 27, Master and above = 28 |
| Apex tiers | Master, Grandmaster, Challenger |
| Season | A period with its own ranks and leaderboards (section 7) |

Rounding everywhere in this document is to the nearest integer, halves away from zero.

## 3. Tiers and divisions

| Tier | Divisions | Rank index |
|---|---|---|
| Iron | IV, III, II, I | 0 to 3 |
| Bronze | IV, III, II, I | 4 to 7 |
| Silver | IV, III, II, I | 8 to 11 |
| Gold | IV, III, II, I | 12 to 15 |
| Platinum | IV, III, II, I | 16 to 19 |
| Emerald | IV, III, II, I | 20 to 23 |
| Diamond | IV, III, II, I | 24 to 27 |
| Master | none | 28 |
| Grandmaster | none | 28 |
| Challenger | none | 28 |

The tier list and the division count are fixed by this document; changing them is a spec change.

### 3.1 Divisions (Iron IV to Diamond I)

- LP runs from 0 to `DivisionLp` (default 100).
- Promotion: when LP reaches `DivisionLp` or more, the player moves up one division (Diamond I moves to Master) and keeps the excess: new LP = LP - `DivisionLp`.
- Demotion: when LP would go below 0, the player moves down one division with LP = `DemotionLp` (default 75).
- Iron IV is the floor: LP stops at 0.
- A match moves a player at most one division. Publish validation rejects `LpChangeMax` + `OffenseRatingPenalty` >= `DivisionLp`.

### 3.2 Apex tiers (Master and above)

- LP starts at the carried excess on promotion from Diamond I and has no upper bound.
- When LP would go below 0, the player drops to Diamond I with LP = `DemotionLp`, whatever their apex tier.
- Master is reached by LP. Grandmaster and Challenger are positions, recomputed per mode once a day at `ApexRecalcHourUtc` (default 00:00 UTC, server time) and once more right before the season-end snapshot:
  1. Order all apex players of the mode by LP, then by tie-breakers 3 to 5 of section 8.1.
  2. The first `ChallengerSlots` (default 50) players with LP >= `ChallengerMinLp` (default 500) are Challenger.
  3. The next `GrandmasterSlots` (default 200) players with LP >= `GrandmasterMinLp` (default 200) are Grandmaster.
  4. Everyone else at index 28 is Master.
- Between recalculations a player keeps the apex tier of the last recalculation, unless they drop out of the apex tiers. A new Master shows Master until the next recalculation.
- Slots are global per mode, not per country.

## 4. MMR and LP per match

### 4.1 Inputs

For each human player of a ranked match, with values taken before the match:

| Input | Meaning |
|---|---|
| `S` | 1 if the player's team won (R-46, R-47), else 0. A ranked match cannot end in a draw (R-47). |
| `R_team` | 1v1: the player's MMR. 2v2: the mean of both partners' MMR. |
| `R_opp` | 1v1: the opponent's MMR. 2v2: the mean of both opponents' MMR. |
| `K` | `MmrKProvisional` (default 40) for a placement match (section 5) or while the player has fewer than `MmrProvisionalMatches` (default 20) ranked matches in the mode; otherwise `MmrK` (default 20) |
| `RankMmr` | The MMR that the player's rank and LP stand for: `RankMmrBase` (default 800) + `RankMmrStep` (default 40) x (rank index + LP / `DivisionLp`) |

Score margin, hands won and win streaks do not change MMR or LP. Streaks are covered by the MMR gap: a player who keeps winning rises in MMR above `RankMmr` and gains more LP per win.

### 4.2 MMR

```
E      = 1 / (1 + 10 ^ ((R_opp - R_team) / MmrScale))
dMMR   = round(K x (S - E))
```

`MmrScale` default 400. New players start at `InitialMmr` (default 1200, which is `RankMmr` of Silver II at 0 LP).

### 4.3 LP

```
d      = MMR - RankMmr
win    : LP gain = clamp(round(LpBase x 2 x (1 - E) x clamp(1 + d / LpGapScale, LpFactorMin, LpFactorMax)), LpChangeMin, LpChangeMax)
loss   : LP loss = clamp(round(LpBase x 2 x E       x clamp(1 - d / LpGapScale, LpFactorMin, LpFactorMax)), LpChangeMin, LpChangeMax)
```

Defaults: `LpBase` 20, `LpGapScale` 200, `LpFactorMin` 0.5, `LpFactorMax` 1.5, `LpChangeMin` 5, `LpChangeMax` 40.

- `E` makes upsets worth more and expected wins worth less.
- `d` pulls the rank toward the MMR: a player ranked below their skill gains more and loses less, a player ranked above it gains less and loses more.
- An even match (`E` = 0.5) at `d` = 0 is +20 / -20.

### 4.4 Team results (2v2)

- Both partners get the same `S`, `E`, `R_team` and `R_opp`.
- Each partner's `dMMR` uses their own `K`, and each partner's LP uses their own `d`. Partners on the same team can therefore gain or lose different amounts.
- There is no per-player performance score. Points go to the team, not the player (R-25), so a team result is the only input.

### 4.5 AFK, leavers and their partners

Offenses and their detection are defined in `docs/penalties.md`; this section only maps its section 5 onto MMR and LP.

| Who | MMR | LP |
|---|---|---|
| Offender | Applied as a loss (`S` = 0), even if their team won | Normal loss + `OffenseRatingPenalty` |
| Partner of an offender (2v2), team lost | Loss reduced by `PartnerLossReductionPercent` | Loss after the clamp of section 4.3, reduced by `PartnerLossReductionPercent` |
| Partner of an offender (2v2), team won | Normal win | Normal win |
| Opponents | Normal | Normal |

The seat played by the bot after R-51 or R-52 keeps its human player's MMR in `R_team` or `R_opp`.

### 4.6 Worked examples

1v1. A has MMR 1300 at Silver I 40 LP; B has MMR 1250; K = 20.

| Step | Value |
|---|---|
| `E` for A | 1 / (1 + 10 ^ (-50 / 400)) = 0.5715 |
| `RankMmr` of A | 800 + 40 x (11 + 0.40) = 1256, so `d` = 44 |
| A wins | MMR +9 (1309); LP + round(20 x 2 x 0.4285 x 1.22) = +21, Silver I 61 LP |
| A loses | MMR -11 (1289); LP - round(20 x 2 x 0.5715 x 0.78) = -18, Silver I 22 LP |

2v2. Team A: P1 MMR 1400 at Gold II 50 LP (K 20), P2 MMR 1200 at Silver IV 20 LP (provisional, K 40). Team B mean MMR 1300. `R_team` = 1300, `E` = 0.5.

| Player | `RankMmr` | `d` | Team wins | Team loses |
|---|---|---|---|---|
| P1 | 1380 | 20 | MMR +10, LP +22 | MMR -10, LP -18 |
| P2 | 1128 | 72 | MMR +20, LP +27 | MMR -20, LP -13 |

Division moves: Gold I 90 LP wins +21: Platinum IV 11 LP. Gold III 10 LP loses -18: Gold IV 75 LP. Iron IV 10 LP loses -18: Iron IV 0 LP.

- Implemented in `server/internal/rating` (#112): sections 3.1, 3.2, 4 and the worked examples are table-driven tests; coefficients are the `rating` config snapshot domain with one `Config` per mode (`docs/ranking/examples/rating.json`), and every applied match records the version it used. Grandmaster and Challenger (#113) are recomputed per mode by `rating.Store.RecalculateApex` once a day at `ApexRecalcHourUtc` under an advisory lock, in the order of 8.1 without its apex part (LP, then the server time of the match that set the LP, ranked wins, account id); the season-end run is #115.

## 5. Placement

- The first `PlacementMatches` (default 5) ranked matches of a season in a mode are placement matches. Until they are done the player has no visible rank in that mode and is not on its leaderboards.
- MMR changes normally with the provisional `K` (section 4.1). LP is not tracked.
- Section 4.5 applies to MMR. LP penalties earned during placement (`OffenseRatingPenalty`) are summed and subtracted once placement ends, never below 0 LP and never causing a demotion.
- After the last placement match:
  1. MMR rank index = floor((MMR - `RankMmrBase`) / `RankMmrStep`), limited to 0 to 27.
  2. A returning player (had a final rank in the mode in the previous season) is limited to their reset rank (section 7.2) minus `PlacementMaxDropDivisions` (default 4) to reset rank plus `PlacementMaxClimbDivisions` (default 4).
  3. A new player (no final rank in the mode in the previous season) is limited to at most `NewPlayerPlacementCap` (default Gold IV).
  4. The result is never below Iron IV and never above Diamond I. LP starts at 0.
- A player who does not finish placement before the season ends has no final rank for that season.

- Implemented in `server/internal/rating` (#114): `Config.Place` sets the first rank of a season from the MMR after `PlacementMatches`, limited for a returning player to the reset rank of 7.2 minus `PlacementMaxDropDivisions` and plus `PlacementMaxClimbDivisions`, and for a new player to `NewPlayerPlacementCap`; LP starts at 0 after the summed placement penalties. The examples of 7.3 are tests.

## 6. No dodge

A ranked match has no accept or decline step (R-53), so there is no dodge and no dodge penalty. A player who has not loaded when the loading phase ends starts the match disconnected (R-57) and falls under the leave rules of `docs/penalties.md`: back within `ReconnectGraceSeconds` is no offense, later is a leave.

## 7. Seasons and soft reset

### 7.1 Season

- A season is created in the admin panel with a start and an end in server time (ADR 0007). Default length `SeasonLengthMonths` (default 3) calendar months. The next season starts at the end of the previous one, with no gap.
- Section 3 structure, `PlacementMatches`, the reset table and the apex slot rules are frozen while a season runs. MMR and LP coefficients (section 4) can change mid-season and apply to matches that start after the publish; each match records the config version it used (ADR 0006).
- Season end runs once per mode and is idempotent (#115): final apex recalculation, leaderboard snapshot, final ranks stored for rewards (#120), reset ranks computed for the next season.

- Implemented in `server/internal/rating` (#115): seasons are rows of `ranked_seasons` (`cmd/ranked-season` until #143), contiguous by construction; the end runs once per mode when the end time has passed (apex recalculation, `season_results` snapshot in the order of 8.1, reset ranks, the next season created with `SeasonLengthMonths` if none exists), a result of a closed season changes MMR only, and `state` is derived from the times and the per-mode close marks. Rank rewards and emblems from the snapshot are #120; the Redis leaderboards are #116.

### 7.2 Soft reset

Every player with a final rank gets a reset rank for the next season from `SeasonResetTable`, one row per final rank (Iron IV to Diamond I, Master, Grandmaster, Challenger). The reset rank is the starting point that placement (section 5) moves from. MMR is not reset.

The default table is generated from three values, and admins can then edit single rows:

```
old       = final rank index (apex tiers = 28)
reset     = min(ResetCeiling, max(old - 4 x ResetTierDrop, min(old, ResetFloor)))
```

Defaults: `ResetTierDrop` 1 (one tier down, same division), `ResetFloor` Bronze IV (nobody is reset below Bronze IV, and nobody at or below it is reset at all), `ResetCeiling` Diamond IV.

Publish validation: no row maps to a rank above its old rank, and a higher old rank never maps to a lower reset rank than a lower old rank.

### 7.3 Reset table with defaults

| Final rank | Reset rank | Placement range (section 5) |
|---|---|---|
| Iron IV | Iron IV | Iron IV to Bronze IV |
| Iron III | Iron III | Iron IV to Bronze III |
| Iron II | Iron II | Iron IV to Bronze II |
| Iron I | Iron I | Iron IV to Bronze I |
| Bronze IV | Bronze IV | Iron IV to Silver IV |
| Bronze III | Bronze IV | Iron IV to Silver IV |
| Bronze II | Bronze IV | Iron IV to Silver IV |
| Bronze I | Bronze IV | Iron IV to Silver IV |
| Silver IV | Bronze IV | Iron IV to Silver IV |
| Silver III | Bronze III | Iron III to Silver III |
| Silver II | Bronze II | Iron II to Silver II |
| Silver I | Bronze I | Iron I to Silver I |
| Gold IV | Silver IV | Bronze IV to Gold IV |
| Gold III | Silver III | Bronze III to Gold III |
| **Gold II** | **Silver II** | **Bronze II to Gold II** |
| Gold I | Silver I | Bronze I to Gold I |
| Platinum IV | Gold IV | Silver IV to Platinum IV |
| Platinum III | Gold III | Silver III to Platinum III |
| Platinum II | Gold II | Silver II to Platinum II |
| Platinum I | Gold I | Silver I to Platinum I |
| Emerald IV | Platinum IV | Gold IV to Emerald IV |
| Emerald III | Platinum III | Gold III to Emerald III |
| Emerald II | Platinum II | Gold II to Emerald II |
| Emerald I | Platinum I | Gold I to Emerald I |
| Diamond IV | Emerald IV | Platinum IV to Diamond IV |
| Diamond III | Emerald III | Platinum III to Diamond III |
| Diamond II | Emerald II | Platinum II to Diamond II |
| Diamond I | Emerald I | Platinum I to Diamond I |
| Master | Diamond IV | Emerald IV to Diamond I |
| Grandmaster | Diamond IV | Emerald IV to Diamond I |
| Challenger | Diamond IV | Emerald IV to Diamond I |

Gold II, step by step. Old index 14. Reset = min(24, max(14 - 4, min(14, 4))) = 10, Silver II. After 5 placement matches the MMR picks the rank inside Bronze II (index 6) to Gold II (index 14):

| MMR after placement | MMR rank index | Placed at |
|---|---|---|
| 1412 | floor(612 / 40) = 15 | Gold II (limited by the climb) |
| 1290 | floor(490 / 40) = 12 | Gold IV |
| 1100 | floor(300 / 40) = 7 | Bronze I |
| 1000 | floor(200 / 40) = 5 | Bronze II (limited by the drop) |

## 8. Leaderboards

Per mode and per season. Only players who finished placement in that season appear. PostgreSQL holds the ranks; leaderboards are read models (Redis sorted sets, ADR 0005) rebuilt from it.

### 8.1 Order

Positions are unique; ties are broken in this order:

1. Higher rank index, then, inside the apex tiers, Challenger before Grandmaster before Master.
2. Higher LP.
3. Earlier server time of the match that set the current LP (who got there first).
4. More ranked wins in the season and mode.
5. Lower account id.

Every local leaderboard uses the same order. Apex recalculation (section 3.2) uses it without the apex tier part of step 1.

### 8.2 Global and local

| Board | Who is on it |
|---|---|
| Global | Every placed player of the mode |
| Country | Players whose leaderboard country is that country |
| Region | Players whose leaderboard country belongs to that region. `LeaderboardRegions` maps each region to its countries; a country belongs to at most one region. |
| Friends | The global order filtered to the viewer and their friends (social epic) |

- A player's leaderboard country is their account country (#117: profile, IP fallback) at their first ranked match of the season in the mode played while the account has a country. It stays fixed for that season and mode, so moving country cannot be used to top a smaller board. Players without a country appear on the global and friends boards only.
- Each board shows the top `LeaderboardTopSize` (default 200) and the viewer's own position and neighbours, whatever their position.
- The season-end snapshot freezes every board of the season. Past seasons stay readable.
- Every row of every board opens that player's public profile (`docs/privacy.md` section 5): nickname, equipped cosmetics, rank and the emblem showcase (8.4). A player can browse the global board and any country or region board, not only their own.

- Implemented in `server/internal/leaderboard` (#116): the global board per season and mode as a Redis sorted set in the order of 8.1 (score: rank index, apex tier, LP; member: time the LP was set, wins, account id), updated by the rating store on every change and rebuilt from PostgreSQL when missing; `GET /leaderboard/{mode}` serves the top `LeaderboardTopSize` and the viewer's position with neighbours. Country and region boards (#117): the same sets per board (`country:<CC>`, `region:<id>` from `LeaderboardRegions`) fed by the same update and rebuild paths; the leaderboard country is frozen in `ratings.board_country` at the first ranked result played while the account has a country, snapshotted into `season_results` and cleared for the next season; the account country is set from the profile (`PUT /profile/country`, one change per `CountryChangeCooldown`, default 180 days) or adopted from the connection country at sign-in. Friends boards are #153.

### 8.3 Leaderboard emblems

Owner decision (2026-10-26). A season-end finish near the top of a board earns a permanent emblem.

- Brackets come from `EmblemBrackets` (default 1, 50, 100, 200): final position 1 earns Top 1, 2 to 50 Top 50, 51 to 100 Top 100, 101 to 200 Top 200. Only the best bracket per board is awarded.
- Emblems are awarded for the global board and for the player's country and region boards, per mode. Friends boards award nothing.
- An emblem records mode, season, board (global, country code or region id) and bracket. It is awarded once by the idempotent season-end job (#115) from the frozen snapshot, never from a live position, and is kept for the account's lifetime.
- Shadowbanned accounts are not on public boards (`docs/anti-cheat.md` section 3), so they earn no emblem. An emblem awarded through a position later found to be cheated is revoked by an admin with an audited reason (#209).
- Emblem art is one static asset per bracket and board scope (global, local), with the season and the country or region shown as text, not baked into art (#178).

### 8.4 Profile emblem showcase

- A player chooses up to `ProfileEmblemSlots` (default 3) of their own emblems and their order for their public profile. Slots may stay empty.
- The client sends only the chosen emblem ids. The server accepts an id only if the account owns that emblem and stores the showcase; other players always see the server's copy (`docs/anti-cheat.md` NT-9).
- A revoked emblem leaves the showcase at once.
- The medal collection screen lists emblems in their own section; emblems are not medals (`docs/pass-missions-medals.md` 5.5).

## 9. No bot fill in ranked play

Owner decision on #8 (2026-10-26).

- A ranked match starts only when every seat is held by a real player. The ranked queue never fills a seat with a bot, however long the wait.
- The server bot takes a seat in a ranked match only to replace a player during the match (R-50 to R-52). Its MMR and LP handling is section 4.5.
- Bot fill after a long wait (#48, config) applies to casual queues only (R-56). Nothing in this document is read or written by a casual match.

## 10. Config

Every key has one value per mode (1v1, 2v2); both start at the same default. Structural keys (section 7.1) are frozen while a season runs.

| Key | Default | Section |
|---|---|---|
| `DivisionLp` | 100 | 3.1 |
| `DemotionLp` | 75 | 3.1, 3.2 |
| `ApexRecalcHourUtc` | 0 | 3.2 |
| `ChallengerSlots` | 50 | 3.2 |
| `ChallengerMinLp` | 500 | 3.2 |
| `GrandmasterSlots` | 200 | 3.2 |
| `GrandmasterMinLp` | 200 | 3.2 |
| `InitialMmr` | 1200 | 4.2 |
| `MmrScale` | 400 | 4.2 |
| `MmrK` | 20 | 4.1 |
| `MmrKProvisional` | 40 | 4.1 |
| `MmrProvisionalMatches` | 20 | 4.1 |
| `RankMmrBase` | 800 | 4.1 |
| `RankMmrStep` | 40 | 4.1 |
| `LpBase` | 20 | 4.3 |
| `LpGapScale` | 200 | 4.3 |
| `LpFactorMin`, `LpFactorMax` | 0.5, 1.5 | 4.3 |
| `LpChangeMin`, `LpChangeMax` | 5, 40 | 4.3 |
| `PlacementMatches` | 5 | 5 |
| `PlacementMaxDropDivisions`, `PlacementMaxClimbDivisions` | 4, 4 | 5 |
| `NewPlayerPlacementCap` | Gold IV | 5 |
| `SeasonLengthMonths` | 3 | 7.1 |
| `ResetTierDrop` | 1 | 7.2 |
| `ResetFloor` | Bronze IV | 7.2 |
| `ResetCeiling` | Diamond IV | 7.2 |
| `SeasonResetTable` | generated, section 7.3 | 7.2 |
| `LeaderboardTopSize` | 200 | 8.2 |
| `LeaderboardRegions` | set by the owner before launch | 8.2 |
| `CountryChangeCooldown` | 180 days | 8.2 |
| `EmblemBrackets` | 1, 50, 100, 200 | 8.3 |
| `ProfileEmblemSlots` | 3 | 8.4 |

Defined in `docs/penalties.md` and only referenced here: `OffenseRatingPenalty`, `PartnerLossReductionPercent`.
