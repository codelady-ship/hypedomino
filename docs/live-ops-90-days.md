# First 90 days of live operations

The calendar, season 1 to 2 transition, balance update process and team rhythms from the global launch to the end of the first season. This document schedules values and procedures that other specs define; it defines no price, reward, duration, colour or rarity of its own. Every entry is an admin record or a scheduled change set (`docs/admin.md`), so nothing here needs a client release (`docs/roadmap.md` M4 exit criterion).

Owned elsewhere and only referenced: launch timeline, incidents, on-call and rollback (`docs/launch-runbook.md`), ranked seasons and reset (`docs/ranking.md`), pass seasons, missions and medals (`docs/pass-missions-medals.md`), coins, shop, monitors (`docs/economy.md`), admin workflow and objects (`docs/admin.md`), content rules (`docs/cosmetics.md`, `docs/localization.md`). Operational numbers marked *default* follow the `docs/launch-runbook.md` convention: starting values, changed by PR to this file. Owner decisions are section 9.

## 1. Principles

| Id | Rule |
|---|---|
| LO-1 | Every calendar entry is a published admin object with its own window, or a change set scheduled with an activation time (`docs/admin.md` 3.5). There is no calendar data outside those objects. |
| LO-2 | All times are server UTC (ADR 0007). Daily entries follow the economy day, weekly entries the economy week starting Monday (`docs/economy.md` section 2, `docs/pass-missions-medals.md` 4.1). |
| LO-3 | Values come from the specs and their admin config. A plan entry that needs a value no spec defines waits for an owner decision (section 9). |
| LO-4 | Calendar content ships through the admin publish and the Addressables catalog (ADR 0002). A client release is never on the critical path of an event or a season. |
| LO-5 | During the launch config freeze (T-3d to T+72h) and the season boundary window (5.2) production publishes go through the release manager, except incident mitigation (`docs/launch-runbook.md` section 4). |

## 2. Time base

| Mark | Meaning |
|---|---|
| Day 0 | T-0 of the global launch (`docs/launch-runbook.md` section 4) |
| W0 | From Day 0 to the first Monday economy-week boundary; W1, W2, ... are the following economy weeks |
| B | End of season 1 and start of season 2: one moment for ranked and pass seasons (`docs/pass-missions-medals.md` 2.1) |

- Season 1 starts at Day 0 (OD-1). With `SeasonLengthMonths` 3 (`docs/ranking.md` 7.1), B falls on day 89 to 92, so the 90 days cover exactly season 1 and its transition. B may move later while season 1 runs, never earlier (`docs/admin.md` 5.12).
- The economy day starts at 00:00 UTC for every player (`docs/economy.md` 2).
- Soft launch (#196) progress does not carry over (OD-1). At Day 0 every account's MMR returns to `InitialMmr` and it plays placement again in both modes, as a new player (`docs/ranking.md` 4.2, 5), and its soft launch pass season ends with the normal season-end job, so purchased levels keep their earned rewards (`docs/pass-missions-medals.md` 2.7). The soft launch ranked season grants no rank rewards. Coins, owned cosmetics and medals are kept.

## 3. Recurring cadence

Everything below runs from published config without a manual step, except where a column says who schedules it.

| Period | What changes | Config | Scheduled by |
|---|---|---|---|
| Economy day | Daily missions (`DailyMissionSlots` 3), one reroll, daily login day, first win of the day, match coin and XP caps reset | `docs/pass-missions-medals.md` 4.4, `docs/economy.md` 2.1 to 2.3 | Automatic |
| Daily at `ShopRotationResetHourUtc` (0) UTC | Daily shop rotation, `ShopDailyRotationSlots` (4) offers | `docs/economy.md` 6.1 | Live-ops lead, a week ahead (7.3) |
| Daily at `ApexRecalcHourUtc` (0) UTC | Grandmaster and Challenger recalculated per mode | `docs/ranking.md` 3.2 | Automatic |
| Economy week (Monday) | Weekly missions (`WeeklyMissionSlots` 4), one reroll; weekly shop rotation, `ShopWeeklyRotationSlots` (2) offers | `docs/pass-missions-medals.md` 4.4, `docs/economy.md` 6.1 | Live-ops lead, a week ahead (7.3) |
| Every second economy week | One live event (OD-2): window Friday 00:00 UTC to Monday 00:00 UTC, event missions, event shop offers, one themed cosmetic set (OD-3) | `docs/admin.md` 5.13, `docs/pass-missions-medals.md` 4.6 | Live-ops lead, two weeks ahead (7.3) |
| Season (B) | Ranked season end and soft reset, pass season rollover | Section 5 | Live-ops lead, before B (5.1) |

- A rotation period without a schedule repeats the previous one (`docs/economy.md` 6.1). A missed rotation entry is therefore a content gap, not an outage.
- New cosmetics reach players only through a source record: a rotation or featured offer, an event, a mission, a pass level or a rank reward (`docs/cosmetics.md` section 5). New items arrive as one themed set of 6 to 8 items per event plus the pass season tracks (OD-3).
- Event reward bundles carry no coins (OD-2): they grant the event's cosmetics and medals, and the event shop sells the set for coins, so events add sinks, not sources (`docs/economy.md` 2.7, 10). A regional holiday (`docs/localization.md` 6) replaces the next regular event; it does not add one.
- An event window does not cross B, so all of an event's pass XP (at most `EventMissionXpBudget` 3000) lands in one pass season.

## 4. Calendar, Day 0 to B + 72h

| When | Entries | Notes |
|---|---|---|
| Before T-3d | Ranked season 1 (start Day 0, end B) with its `resetTable` and `rankRewards` per mode and tier (#120); pass season `season_1` with the same window and its tracks (`docs/pass-missions-medals.md` 8.2); missions and medals; rotations for W0 and W1; launch banners | Everything of a ranked season is editable only until its start (`docs/admin.md` 5.12), so season 1 rank rewards and reset table are final at Day 0 |
| Day 0 | Season 1 starts. Every account enters `season_1` at level 1 at its first session (`docs/pass-missions-medals.md` 2.1) | Promo codes starting on Day 0 end by day 90 at the latest (`PromoMaxValidityDays` 90) |
| Day 0 to T+72h | Launch window: config freeze, hourly then daily health reviews, launch exit at T+72h | `docs/launch-runbook.md` section 4; no balance publish except incident mitigation |
| From W1 | Weekly review each economy week (7.3); it schedules the following week's rotations and event slot | |
| Day 7 | First full rolling 7-day sink ratio (`docs/economy.md` section 10) | The first weekly review after Day 7 is the first that may decide an economy change |
| Day 14 | End of the on-call schedule required by GN-25 | Extended before it runs out, through B + 72h (7.1) |
| Day 30 | Pass pacing checkpoint 1: sample levels light 24, regular 43, heavy 69. Season 2 theme chosen (OD-8) | `docs/pass-missions-medals.md` 8.1 |
| Day 45 | Heavy profile reaches level 100 | Same; pass rewards end for that group, shop and events remain |
| Day 60 | Pass pacing checkpoint 2: light 43, regular 80, heavy 100 | Same |
| Day 75 | Season 2 pass track items and season 2 rank reward items published in admin (OD-8) | Leaves the last weekly reviews before B for strings and the staging run (5.1) |
| Day 77 | Regular profile reaches level 100 | Same |
| Weekly review before B | Season 2 change sets approved and scheduled (5.1); season-end banners and push campaigns scheduled | Last review with time to fix a rejected publish |
| B - 1d to B + 72h | Season boundary window (5.2) | *default*, mirrors the launch window of `docs/launch-runbook.md` section 7 |
| B (day 89 to 92) | Season 1 ends, season 2 starts (5.2) | |
| After B + 72h | Season review (7.3) | Plans the next 90 days |

## 5. Season 1 to 2 transition

The behaviour at B is defined by `docs/ranking.md` 7 and 8.3, `docs/pass-missions-medals.md` 2.1 and 2.7, and `docs/economy.md` 2.6; this section only orders the work.

### 5.1 Before B

| Item | Admin object | Done when |
|---|---|---|
| Ranked season 2: start = B, config per mode, `resetTable`, `rankRewards` | `docs/admin.md` 5.12 (A-19) | Published; validation "seasons contiguous" passes |
| Pass season `season_2`: window = ranked season 2, tracks within `PassFreeTrackCoins` and `PassPremiumTrackCrystals`, products bound | `docs/admin.md` 5.7 (A-16) | Published before its start (PV-1 to PV-10) |
| Season 2 track cosmetics with source `pass.season_2.<track>.level_<n>`; season 1 rank reward items with their `rank_reward` source | `docs/admin.md` 5.1, 5.2 | Cultural review approved, catalog build contains them (V-15, V-17) |
| Strings for season 2 names, items, banners, push in `az`, `en`, `tr`, `ru` | `docs/admin.md` 5.15 | Approved in all four languages (`docs/localization.md` 2.1) |
| Structural changes collected during season 1: section 3 structure, `PlacementMatches`, reset table, apex slot rules, `PassLevels`, XP curve, track tables | Season 2 records only | Frozen fields of a running season are never edited (`docs/ranking.md` 7.1, PV-4) |
| Staging run: season 2 records active and a season-end run for both modes and the pass on staff test accounts | `docs/admin.md` 3.4, 6.3 | Rank rewards, emblems, reset ranks and missing-grant repair observed (#115, #123) |

- Which record's `resetTable` drives the reset at B, season 1's or season 2's, is not stated in `docs/ranking.md` or `docs/admin.md`. Until it is, season 2's table is created as a copy of season 1's so both readings give the same result.

### 5.2 At B

| Order | Event | Defined in |
|---|---|---|
| 1 | Pass: XP stops for `season_1`; level purchase and pass products close | `docs/pass-missions-medals.md` 2.7 |
| 2 | Ranked, per mode, idempotent: final apex recalculation, leaderboard snapshot, final ranks, reset ranks for season 2 | `docs/ranking.md` 3.2, 7.1 |
| 3 | Rank rewards per mode from `RankRewardCoins` and the #120 bundles, key `rank:<seasonId>:<mode>`; leaderboard emblems from the frozen snapshot | `docs/economy.md` 2.6, `docs/ranking.md` 8.3 |
| 4 | Pass season-end job grants any missing level rewards | `docs/pass-missions-medals.md` 2.7 |
| 5 | Season 2 runs: placement from the reset rank, MMR carried; pass at level 1 with 0 XP, level 1 rewards on entry | `docs/ranking.md` 5, 7.2; `docs/pass-missions-medals.md` 2.1, 2.4 |

- A ranked match counts toward the season it started in (`docs/ranking.md` 1); its pass XP goes to the season active when it ends (`docs/pass-missions-medals.md` 2.3).
- A pass purchase validated after B is still granted on the season bound at the store's purchase time (`docs/pass-missions-medals.md` 2.5).
- Boundary window B - 1d to B + 72h (*default*): primary and secondary on-call both on shift, no server deploy, client rollout step or unrelated config publish. The season-end jobs' progress is watched in admin (`docs/admin.md` 5.12); a failure is an incident (`docs/launch-runbook.md` section 6).
- Player notice: banner and push from the live-ops calendar (`docs/admin.md` 5.13); promotional sends only with `promotional_push` consent (`docs/privacy.md` 6.2).

## 6. Balance updates

### 6.1 Signals and levers

A lever is a config key or admin object that already exists. A signal without a threshold in a spec has its threshold in section 9.

| Signal | Source | Threshold | Levers | Not a lever |
|---|---|---|---|---|
| Sink ratio | `docs/economy.md` 10, #110 | Outside `SinkRatioAlertPercent` (60% to 120%); target 70% to 100% | Below: more sinks through rotation, featured and event offers. Above: source keys `MatchCoins*`, `FirstWinBonusCoins`, `DailyLoginCoins`, `DailyMatchCoinCap`, mission slot coins within `DailyMissionCoinBudget` and `WeeklyMissionCoinBudget` | Price of an offer on sale; a discount is a new offer (`docs/economy.md` 6.3) |
| Median balance | `docs/economy.md` 10 | Above `BalanceMedianAlertCoins` (30000) | Time-limited `mythic` offers (`docs/economy.md` 3.1, OQ-6), event shop offers | |
| Account daily earn | `docs/economy.md` 10 | Above `AccountDailyEarnAlertCoins` (1500) | None: a defect or exploit, handled as an economy integrity incident (`docs/launch-runbook.md` 6.5) and #209 | Any balance key |
| Pass pacing | #147, checkpoints of section 4 | Measured level per profile against `docs/pass-missions-medals.md` 8.1; tolerance OD-5 | `PassMatchXp1v1`, `PassMatchXp2v2`, `PassDailyMatchXpCap`, `DailyMissionSlotXp`, `WeeklyMissionSlotXp` within their budgets, mid-season | `PassLevels`, curve and tracks until season 2; catch-up offers (OQ-9) |
| Mission effort | #127, #147 | Completion against the intended effort of `docs/pass-missions-medals.md` 4.4; tolerance OD-5 | Mission targets: add, retire or replace missions in a slot pool | Slot rewards (fixed per slot) |
| Medal earn share | #147 | Outside the band of `docs/pass-missions-medals.md` 5.3 | The medal's `rarityId` | Condition changes on an earned medal |
| Ranked distribution, placement, leaves, queue wait | #147, #28, #48 | OD-6 | `docs/ranking.md` section 4 coefficients mid-season; structural keys only through season 2 (5.1) | Bot fill in ranked (`docs/ranking.md` 9) |
| D1, D7, ARPDAU, matches per day | #196 weekly KPI report | Targets of #196 (OD-4) | None directly; they set the weekly review's priorities | |
| Launch health signals | `docs/launch-runbook.md` 4.1 | As defined there | None: an incident (`docs/launch-runbook.md` section 6) | Any balance key |

### 6.2 Process

| Step | Who | Rule |
|---|---|---|
| 1. Signal | Analyst | Posts the weekly KPI and economy report before the weekly review (#196, #110, #147) |
| 2. Proposal | Live-ops lead | Issue labelled `area:live-ops`: signal, keys with old and new value, the admin economy impact preview (`docs/admin.md` 3.4), the signal value that triggers a rollback |
| 3. Decision | Product owner | Approves on the issue. Every value of `docs/economy.md` 13 and `docs/pass-missions-medals.md` 11 was owner-approved, so changing one is too |
| 4. Change set | Live-ops lead | Built and tried in staging on staff test accounts, promoted to production (`docs/admin.md` 3.4, 3.6, 6.3) |
| 5. Publish | Approver per `docs/admin.md` 3.5 and 8 | Publish now or schedule an activation time; in a freeze window the release manager approves (LO-5) |
| 6. Verify | Analyst | #110 compares the monitors before and after the publish (`docs/economy.md` 10); result at the next weekly review |

- At most one balance change set per domain per economy week (*default*), so the before and after comparison can attribute the effect.
- Values still in force at the season review are written back to the repo defaults and their spec by PR (`docs/economy/balance.csv`, AP-3).

### 6.3 Rollback

- A rollback is a change set restoring the domain from an earlier snapshot (`docs/admin.md` 3.7); `live_ops` may roll back its own domains. Target under 5 minutes (`docs/launch-runbook.md` 5.4).
- Triggered when the rollback value named in the proposal is reached, or at once when the change causes an incident.
- A rollback never reverses grants, purchases, ledger entries or progress. Correcting players is a support action and compensation is a product owner decision (`docs/launch-runbook.md` 6.5).

## 7. Team rhythms

### 7.1 On-call

`docs/launch-runbook.md` section 7 applies throughout: weekly primary and secondary, handoff with a written note, escalation path. Both are on shift in the launch window and in the boundary window (5.2). GN-25 publishes the schedule only to T+14d; before Day 14 it is extended through B + 72h.

### 7.2 Release train

| Step | Rule |
|---|---|
| Cadence | Server weekly on Tuesday; client every second Tuesday (OD-7, *default*). Tuesday keeps deploys away from the Monday economy boundary and the weekend event window |
| Candidate | Tagged from `main`; regression RG-1 to RG-13 (`docs/test-strategy.md` 8) |
| Server | Canary and drain (`docs/launch-runbook.md` 5.1); server before client |
| Client | Play staged rollout and App Store phased release (`docs/launch-runbook.md` 5.2); risky features behind a flag (#145) |
| Blackout | No train step in the launch window or the boundary window (5.2) |
| Hotfix | `docs/launch-runbook.md` 5.4, outside the cadence |

### 7.3 Reviews

| Review | When | Who | Inputs | Output |
|---|---|---|---|---|
| Daily health | T+24h, T+48h | Product owner | `docs/launch-runbook.md` 4.1, tickets, store reviews | Continue, hold or widen (runbook section 4) |
| Weekly live-ops | Once per economy week, before the Monday boundary, from W1 | Product owner, live-ops lead, analyst, release manager, support lead | Weekly KPI report (#196), economy monitors, pacing from Day 30, support themes, postmortem actions, the next week in the calendar (A-21) | Balance decisions (6.2); the next week's rotations, event and banners approved and scheduled; release train go or hold. Recorded on a weekly `area:live-ops` issue |
| Postmortem | After a SEV1 or SEV2 | Incident commander | `docs/launch-runbook.md` 6.4 | Action items as issues |
| Season | After B + 72h | Product owner, all leads | Pacing against 8.1, sink ratio, medal shares, rank distribution, season-end job records | Values written back to repo defaults (6.2); next season plan |

## 8. Entering the calendar in admin

Every entry is created on its own screen and shows in the calendar view A-21 (`docs/admin.md` 5.13, #146).

| Entry | Object | Screen | Time field |
|---|---|---|---|
| Ranked season and rank rewards | `docs/admin.md` 5.12, bundles 5.4 | A-19, A-11 | `start`, `end`; rewards granted by the season-end job |
| Pass season | 5.7 | A-16 | `window` |
| Live event, its missions and shop offers | 5.13, 5.8, 5.5 | A-22, A-17, A-12 | Event `window`; missions run inside it |
| Daily and weekly rotation | 5.5 | A-13 | Period start |
| Featured offer | 5.5 | A-12 | `window` |
| Banner | 5.13 | A-23 | `window` |
| Push campaign | 5.13 | A-24 | Send time |
| Promo campaign | 5.11 | A-15 | `window`, at most `PromoMaxValidityDays` |
| Config value change | 5.14 in a change set | A-26, A-2, A-3 | Scheduled activation time (3.5) |
| Planned maintenance | Live control (section 4) | A-25, A-23 | Switched at the time by `live_ops`; announced ahead by a scheduled banner and the status page (`docs/launch-runbook.md` 8.1) |

Windows inside published objects need no scheduled publish; they run on server time (`docs/admin.md` 3.5).

## 9. Owner decisions

Decided by the owner on 2026-10-26.

| Id | Question | Decision |
|---|---|---|
| OD-1 | Season 1 start and soft launch progress | Season 1 starts at Day 0. Soft launch rank and pass progress reset; coins, cosmetics and medals stay (section 2) |
| OD-2 | Live event cadence, length and rewards | One event every second economy week, Friday to Monday (3 days). No coins in event bundles; cosmetics and medals only, event set sold for coins. Theme chosen at the weekly review two weeks ahead |
| OD-3 | New cosmetics after launch | One themed set of 6 to 8 items per event (about 45 items in 90 days) plus each season's pass tracks |
| OD-4 | Targets for D1, D7, ARPDAU and matches per day | Set by the owner from the first two weeks of soft launch data (#196); until then the report runs without thresholds |
| OD-5 | Tolerance for pass pacing and mission completion | Same as OD-4; no pacing change without an owner call until set |
| OD-6 | Targets for rank distribution, placement, leaves, queue wait | Same as OD-4; no coefficient change without an owner call until set |
| OD-7 | Release train cadence | Server weekly, client every two weeks, both on Tuesday; hotfix outside the train (7.2) |
| OD-8 | Season 2 content | Theme on day 30; pass track items and rank reward items published by day 75 (section 4) |

## 10. Implementation

| Issue | Needed for |
|---|---|
| #146 | Calendar view A-21, events, banners, push, maintenance |
| #139, #140, #141, #143 | Rotations, pass seasons, missions and medals, ranked seasons |
| #115, #120, #123 | Season-end jobs and rank rewards (5.2) |
| #145 | Scheduled config changes and flags |
| #110, #147, #196 | Monitors, dashboards, weekly KPI report (6.1) |
| #28 | Alerts, pager, status page (section 7) |
