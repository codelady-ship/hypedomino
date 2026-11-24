# Closed beta

How the game reaches its first real players before the soft launch (#196): the beta phases on the Google Play testing tracks and TestFlight, who the testers are and how they join, how they send feedback, which crash, retention and match signals are tracked and reported, how the launch defaults are tuned from beta data, what happens to beta progress and purchases, and the exit criteria that close #193. Epic 17 (#189).

**Status: draft, awaiting owner approval.** Every rule here is a proposal the owner approves by merging the pull request that adds or changes it. Decisions that only the owner can make are section 10, each with a proposed default; until the owner answers, a default is a plan, not a decision.

Owned elsewhere and only referenced here: release gate (`docs/roadmap.md` section 4), go/no-go item GN-6, health signals, roles, rollout and incidents (`docs/launch-runbook.md`), regression and device smoke (`docs/test-strategy.md`), personal data (`docs/privacy.md`), coin values and monitors (`docs/economy.md`, `docs/economy/balance.csv`), pass pacing (`docs/pass-missions-medals.md`), admin workflow, change sets and live controls (`docs/admin.md`), balance update process (`docs/live-ops-90-days.md` section 6), environments (`docs/environments.md`), integrity and receipts (`docs/anti-cheat.md`). No price, reward, duration, colour or rarity is defined here. Operational numbers marked *default* follow the `docs/launch-runbook.md` convention: starting values, changed by PR to this file. Game values stay admin config (ADR 0006); names in `CamelCase` code font are config keys. Roles are the roles of `docs/launch-runbook.md` section 1.

## 1. Phases

The beta runs in three phases. Each phase uses one Play testing track and one TestFlight group, and the same build reaches both stores.

| Id | Phase | Google Play | TestFlight | Environment | Testers | Size (*default*, OQ-2) | Length (*default*, OQ-3) |
|---|---|---|---|---|---|---|---|
| CB-1 | Internal | Internal testing track | Internal group (App Store Connect team members) | `staging` | Team only | Whole team, within the store limits below | At least 1 week |
| CB-2 | Closed, small | Closed testing track, tester list `beta-small` | External group `beta-small` | `production` | Team and invited players | Up to 100 | At least 2 weeks |
| CB-3 | Closed, wide | Closed testing track, tester list `beta-wide` (includes `beta-small`) | External group `beta-wide` | `production` | Team and invited players | Up to 1,000 | At least 3 weeks, then until section 7 is green |

- Why staging for CB-1: only team accounts play, so store sandbox purchases, drills and destructive tests run without touching player data (`docs/environments.md` section 1). Real players never play on `staging`, because it holds synthetic data and staff accounts only.
- Why production for CB-2 and CB-3: invited players are real people whose data falls under `docs/privacy.md`; that data lives only in `production` (E-1, E-2). Before the soft launch the production environment has no other players, so beta controls (section 6) apply to everyone in it.
- Why these sizes: CB-2 is small enough that the team reads every report; CB-3 is large enough to reach the session counts of BX-1 on each platform in about a week even though only testers with `diagnostics` consent send crash data (section 4.1), and to fill ranked queues in playtest windows (OQ-10).
- Why these lengths: CB-3 needs at least two full weeks after its first cohort to measure D7 for two weekly cohorts (BS-7).
- Store limits at the time of writing, checked in the consoles when a group is created: Play internal testing up to 100 testers; TestFlight internal testing up to 100 App Store Connect users; TestFlight external testing up to 10,000 testers; a TestFlight build expires 90 days after upload, so a CB-3 longer than that needs a newer build. The first build of a TestFlight external group goes through Beta App Review, and Play closed testing builds go through Play review; the release manager plans both into the phase start.
- If the Play developer account is a personal account, Play may require a closed test with a minimum number of opted-in testers for a minimum number of days before production access. The release manager checks the requirement in Play Console before CB-2 and, if it applies, CB-2 and CB-3 are sized and timed to satisfy it.
- A phase advances only by a product owner decision at the weekly beta review (section 4.3), when the next phase's entry criteria (1.1) hold, no `priority:p0` bug from the beta is open, and crash-free sessions of the current build are at least 99% (*default*) on each platform.

### 1.1 Entry criteria

Every item is green before the phase opens to testers, or waived by an owner comment on #193. Legal items are never waived on #193; they are resolved on #187 or the phase stays closed to the testers they concern.

| Id | Item | Phase | Owner | Evidence |
|---|---|---|---|---|
| CE-1 | Builds are produced and signed by the pipeline from a tag, with symbols and mapping files uploaded (GN-11) | CB-1 | Client lead | Pipeline run, #27, #182 |
| CE-2 | Crash reporting shows release health for the beta build; a test crash is visible (GN-13) | CB-1 | Client lead | Dashboard link, #182 |
| CE-3 | The build hash of every distributed beta build is on the environment's build allowlist; older beta builds can be blocked (#210) | CB-1 | Client lead, live-ops lead | Allowlist entries |
| CE-4 | In-game support form reachable from Settings > Support with the bug category (`docs/launch-runbook.md` 9.1); tickets reach the support tool | CB-1 | Support lead | Test ticket, #144 |
| CE-5 | Sandbox purchase on each store granted once after server validation on `staging` and restored (M3 exit criterion, GN-22 rehearsal) | CB-1 | Live-ops lead | Ledger entry on a staff test account, #106, #107 |
| CE-6 | Production environment, secrets, backups and restore exercised (GN-8, GN-15); dashboards and alerts live (GN-12) | CB-2 | Server lead | Same evidence as the runbook items, #24, #25, #28, #188 |
| CE-7 | First-launch flow, age gate and consent records as `docs/privacy.md` sections 6 and 7; no analytics or crash event before consent (L-12 to L-14) | CB-2 | Client lead | Test record, #73, #157, #181, #182 |
| CE-8 | In-app and web account deletion work (L-15, L-17, L-18) | CB-2 | Server lead | Test record, #63 |
| CE-9 | Privacy policy and terms published and linked in the app and in both testing listings, naming the controller (L-1, L-10, L-27) | CB-2 | Product owner | Links, #195 |
| CE-10 | Controller duties that apply before data of the first player is collected: AZ-2 registration if the controller is an Azerbaijani entity (L-29); VERBIS through a Turkish representative before testers in Turkey are invited (L-4); EU and UK representatives (L-3) | CB-2 | Product owner | #187 record. Without L-4, no tester from Turkey is invited |
| CE-11 | `docs/privacy.md` section 3 has rows for the tester data of section 2.3 (sign-up record, invitation lists, support tickets, store tester feedback) as P-2 requires | CB-2 | Product owner | Merged PR |
| CE-12 | Play Data safety form and Apple App Privacy answers submitted from `docs/privacy.md` section 10 (L-24, L-25) | CB-2 | Release manager | Console screenshots, #194 |
| CE-13 | Beta controls of section 6 in place in production: purchase kill switch state per OQ-5, beta ranked and pass seasons published | CB-2 | Live-ops lead | Live control state, config version |
| CE-14 | Load test passed for the CB-3 size with margin (#192 tooling, `docs/performance.md` section 6) | CB-3 | Server lead | Load run record |

## 2. Testers

### 2.1 Sources

| Source | Phases | Notes |
|---|---|---|
| Team | CB-1 to CB-3 | Everyone on the project, on their own devices. In CB-1 they sign in on `staging`; in CB-2 and CB-3 they play on `production` with ordinary player accounts (TP-6) |
| Invited players | CB-2, CB-3 | People who applied through the sign-up form (2.2), chosen to cover both platforms, the device tiers of `docs/performance.md` section 1 and the four launch languages (`docs/localization.md`) |

Selection favours device and language coverage over enthusiasm. A tester is selected only if their sign-up says they are 13 or older and their country is a store country of `docs/privacy.md` 9.3 (every country except Russia, OQ-1).

### 2.2 How testers join

1. The tester fills in the sign-up form (OQ-4): email address, platform, device model, country, preferred language, and a confirmation that they are 13 or older. Nothing else is asked.
2. The release manager adds selected testers to the phase's Play tester list and TestFlight group by email. TestFlight public links are not used, so every tester is selected (OQ-1).
3. The invitation carries the beta notice (2.3), the store opt-in link (Play) or the TestFlight invitation, and the external channel invite (section 3).
4. The tester installs from the store, opens the game and goes through the normal first-launch flow: age gate, privacy notice, consent choices and sign-in (`docs/privacy.md` 6.1). The in-game age gate is authoritative: a tester under `MinAccountAgeByCountry` gets no account, whatever the sign-up said.
5. The Play closed track's country list excludes Russia, matching the store countries (`docs/privacy.md` 9.3).

Leaving: a tester leaves the testing program in the store at any time and deletes their account in the game or on the web page like any player (`docs/privacy.md` 8.2).

### 2.3 Consent and data handling

| Id | Rule |
|---|---|
| TP-1 | Testers are players. Every rule of `docs/privacy.md` applies to them unchanged: 13 and older only (7.3), `analytics` and `diagnostics` off until the tester turns them on (6.2), data in Germany on Hetzner and crash reports in the Sentry EU region (section 2), the same rights and deletion (section 8) |
| TP-2 | Testers are asked to turn on `analytics` and `diagnostics`, with the reason (crash and retention data, section 4), but nothing in the beta depends on it: refusing changes nothing in gameplay, rewards or tester compensation (`docs/privacy.md` 6.2) |
| TP-3 | The beta notice in the invitation states: the build is a test build and may crash; what is measured and that it needs the tester's consent in the game; whether progress is reset at the end (OQ-6) and what testers receive (OQ-7); that purchases are disabled or test-only (OQ-5); how to leave and delete the account; where to send feedback. It is in the four launch languages and links the privacy policy (#195) |
| TP-4 | Tester data outside the game is kept to the minimum and has its own rows in `docs/privacy.md` section 3 before CB-2 (CE-11): the sign-up record, the invitation lists in the store consoles, support tickets with their text and screenshots, and feedback sent through TestFlight or the Play feedback channel. Proposed retention: sign-up records and invitation lists are deleted when the beta ends; tickets follow the support retention that row sets |
| TP-5 | The repository is public. No issue, pull request, comment or report contains a tester's name, email, nickname, account id, install id or screenshot with personal data. Beta bug issues reference the support ticket id; the ticket stays in the support tool |
| TP-6 | Team members in CB-2 and CB-3 play on ordinary player accounts, never on staff test accounts (`docs/admin.md` 6.3); admin work stays on the separate SSO admin accounts (`docs/admin.md` section 9). Beta play then goes through exactly the paths players use |

## 3. Feedback

| Id | Channel | Use | Lands in |
|---|---|---|---|
| FB-1 | In-game support form, Settings > Support, bug category | Primary bug and crash report path. Attaches player id, app version, build, platform, OS, device model, language and last match id (`docs/launch-runbook.md` 9.1) | Support tool (#144) |
| FB-2 | TestFlight feedback (screenshot and crash feedback from the TestFlight app) | iOS reports with a screenshot | App Store Connect |
| FB-3 | Play testing feedback channel (the feedback email or URL set on the testing track) | Android testers who prefer the store's channel | The support inbox (`docs/launch-runbook.md` section 10, owner input OI-13) |
| FB-4 | External community channel, private and invite-only, one space per launch language (OQ-4) | Discussion, playtest window announcements, polls, build notes. Bug reports posted there are moved to FB-1 by the support lead | Not a record; summarised in the weekly report |
| FB-5 | Crash groups and errors | Automatic, from testers with `diagnostics` consent | Sentry (#182) |

### 3.1 Triage

1. The support lead reads FB-1 to FB-4 every business day (*default*) and answers the tester within 2 business days (*default*).
2. A reproducible defect becomes a GitHub issue labelled `type:bug`, one `area:` label and a `priority:` label, with "Found in closed beta (#193)" and the build number in its body, written per the `CLAUDE.md` language rule. It references the support ticket id, never tester data (TP-5).
3. Every new Sentry crash group on a beta build gets an issue the same way; the issue links the Sentry group.
4. Priority: a crash on launch or in a match, data loss, a currency or item appearing or disappearing, or a security or privacy problem is `priority:p0`; a broken feature without a workaround is `priority:p1`.
5. An economy integrity case (currency or items outside legal play) is handled as an incident (`docs/launch-runbook.md` 6.5) even in the beta.
6. Each build's notes in FB-4 list the beta issues it fixes, by title, so testers can retest.

## 4. Crash and retention tracking

### 4.1 Signals

Measured per platform and per app version (version and build number) unless stated. "Consented" means only testers with the named consent purpose contribute; such signals are a sample of the testers, not all of them.

| Id | Signal | Definition | Source | Population | Use |
|---|---|---|---|---|---|
| BS-1 | Crash-free sessions | Sessions without a fatal error, native crashes included, divided by all sessions of the build | Sentry release health (#182) | `diagnostics` consented | BX-1 |
| BS-2 | Crash-free users | Testers without a fatal error on the build, divided by all testers on it | Sentry release health (#182) | `diagnostics` consented | Report |
| BS-3 | Android vitals | User-perceived crash rate and ANR rate | Play Console | Devices that share usage data with Google | BX-2 |
| BS-4 | TestFlight crashes | Crash reports and crash feedback | App Store Connect | TestFlight testers | Triage (3.1) |
| BS-5 | Active testers, matches per day | Daily active accounts; completed matches per daily active account, per mode | Server aggregates without ids (#28, #147, `docs/privacy.md` 6.4) | All testers | Report; BT inputs |
| BS-6 | D1 retention | Share of a day-0 cohort (first session on an economy day, server UTC) with a session on day 1 | #181 events, #147 cohorts | `analytics` consented | Report; BX-7 |
| BS-7 | D7 retention | As BS-6, session on day 7 | #181 events, #147 cohorts | `analytics` consented | Report; BX-7 |
| BS-8 | Match completion | Matches with a result row divided by matches started (dealt after the loading phase, R-57), per mode. A leaver does not lower it, since the bot finishes the seat (R-52) | #50, #28 | All | BX-3 |
| BS-9 | Disconnects | In-match disconnects per 100 matches, per platform and network type where known | #45, #28 | All | Report |
| BS-10 | Resume success | Resume attempts within the reconnect grace that restore the session, divided by all such attempts | #45, #87, #28 | All | BX-4 |
| BS-11 | Move round-trip | p99 move round-trip measured on the server, per region | #28 | All | BX-5 |
| BS-12 | Queue wait | p50 and p90 wait per queue and mode; share of casual matches started with a bot-filled seat | #48, #28 | All | BT-11 |
| BS-13 | Turn and loading behaviour | Turn timeouts per 100 turns; AFK rate (R-51); share of players not ready at the end of the loading phase (R-57) | #46, #47 | All | BT-10 |
| BS-14 | Economy monitors | Sink ratio, median balance, account daily earn, source mix (`docs/economy.md` 10) | #110, #147, computed from D-16 | All | BT-6, BT-7 |
| BS-15 | Economy integrity | Ledger reconciliation mismatches (#101); hard anti-cheat signals (`docs/anti-cheat.md` section 5) | #101, #202, #209 | All | BX-6 |
| BS-16 | Soft signal rate | Soft anti-cheat signals per 100 testers by signal; score-based sanctions applied to testers | #209 | All | BT-13; report to the security lead only (`docs/admin.md` 5.14) |
| BS-17 | Support themes | Tickets per category and per new symptom | #144 | Testers who write | Report |

- No client performance telemetry (frame time, cold start) is collected: it has no row in `docs/privacy.md` section 3 (P-2). Performance budgets stay the device matrix runs of #191 and #161.
- Retention and play-time figures come from consented testers only and from a self-selected group; they prove that the pipeline works and show large problems, they do not predict launch retention. Targets are set from soft launch data (`docs/live-ops-90-days.md` OD-4).

### 4.2 Weekly report

Posted by the analyst once per economy week, before the weekly beta review (4.3), from CB-1 to beta exit.

| Part | Content |
|---|---|
| Phase | Phase, active testers per platform, builds in use and their share |
| Stability | BS-1 to BS-4 per platform and build, week over week; top crash groups with their issues |
| Engagement | BS-5 to BS-7 per weekly cohort, with cohort sizes |
| Match health | BS-8 to BS-13 |
| Economy and progression | BS-14, BS-15 and the measures of section 5.1 against their reference values |
| Feedback | BS-17 themes, open beta bugs by priority |
| Exit | The section 7 table with the current value and state of every BX item |

The report holds aggregates only. Where it is stored is OQ-11; the section 7 table is also posted on #193 every week, without engagement or revenue figures.

### 4.3 Weekly beta review

Once per economy week. Product owner, release manager, QA lead, live-ops lead, analyst, support lead; security lead when BS-15 or BS-16 changed. Inputs: the weekly report. Output, recorded on #193: phase advance or hold, priorities of open beta bugs, balance proposals for section 5.2, the next playtest windows (OQ-10).

## 5. Balance tuning

Every number of `docs/economy.md` and `docs/pass-missions-medals.md` is a launch default "to be tuned with closed-beta data". This section says what is measured to tune them and how a change is made.

### 5.1 What is measured

Measures are taken in CB-2 and CB-3 on production; CB-1 data is not used, because staff play is not representative. Per-profile measures bucket testers by measured play hours per day around the `ModelProfileHours` profiles (light, regular, heavy), so the comparison is per profile, not a population mean that a group of enthusiasts would bias.

| Id | Measure | Compared with | Lever |
|---|---|---|---|
| BT-1 | Match length per mode, median and mean, first deal to result | `ModelMatchMinutes` (10, 14) | Model key: replaced in `docs/economy/balance.csv` |
| BT-2 | Time between matches: result screen to the next match's first deal, queue included | `ModelBetweenMatchMinutes` | Model key |
| BT-3 | Ranked win rate distribution per mode | `ModelWinRatePercent` | Model key; a wide spread is a matchmaking finding (BT-11) |
| BT-4 | Play hours per day per tester | `ModelProfileHours` | Model key |
| BT-5 | Mission completion per profile and per slot | `ModelProfileMissionPercent`; intended effort of `docs/pass-missions-medals.md` 4.4 | Model key; missions added, retired or replaced in a slot pool |
| BT-6 | Coins earned per day per profile, by ledger reason | "Earn per day by player profile" in `docs/economy.md` section 11 | `MatchCoins*`, `FirstWinBonusCoins`, `DailyLoginCoins`, `DailyMatchCoinCap`, mission slot coins within `DailyMissionCoinBudget` and `WeeklyMissionCoinBudget` |
| BT-7 | Sink ratio and median balance | `SinkRatioAlertPercent`, target 70% to 100%, `BalanceMedianAlertCoins` (`docs/economy.md` 10) | Rotation, featured and event offers; source keys of BT-6 |
| BT-8 | Pass XP per day per profile | XP per day of `docs/pass-missions-medals.md` 8.1 (615, 1229, 2102) | `PassMatchXp1v1`, `PassMatchXp2v2`, `PassDailyMatchXpCap`, `DailyMissionSlotXp`, `WeeklyMissionSlotXp` within their budgets; `PassLevels` and the XP curve for the launch season records |
| BT-9 | Medal earn share | Bands of `docs/pass-missions-medals.md` 5.3 | The medal's `rarityId`. A small beta shows only gross mis-tiering |
| BT-10 | Turn timeouts, AFK rate, loading-phase misses (BS-13) | `TurnSeconds`, `AfkConsecutiveTimeouts`, `MatchLoadingSeconds` (`docs/rules.md` section 13) | Those keys; a change to their default changes `docs/rules.md` first |
| BT-11 | Queue wait and casual bot fill (BS-12) | `CasualBotFillSeconds`, `MmrWindowStart`, `MmrWindowStep`, `MmrWindowStepSeconds`, `MmrWindowMax` (`docs/matchmaking.md`) | Those keys. Beta queues are thin, so waits are read only inside playtest windows |
| BT-12 | Account daily earn alerts on honest testers | `AccountDailyEarnAlertCoins` | That key, only if honest play crosses it |
| BT-13 | Soft anti-cheat signals on honest testers (BS-16) | Signal weights and score thresholds (`docs/anti-cheat.md` section 7) | `anticheat` domain, `super_admin` only; values never published in the repository or the report |

- A model value is replaced only from at least 500 completed matches per mode (*default*) of CB-2 and CB-3.
- The premium pass track cannot be measured while store purchases are off (OQ-5); its value stays as approved.
- Ranked distribution, placement and LP (`docs/ranking.md` section 4) are reported but not tuned from the beta: the population is too small to form a distribution.

### 5.2 How a change is made

| Value | Path |
|---|---|
| Model keys (`ModelMatchMinutes` and the others of `docs/economy.md` section 11) | PR to `docs/economy/balance.csv`, then regenerate the balance sheet and paste it (`docs/economy.md` 11); regenerate the pass sample if its inputs changed (`tools/pass-season`). `npm run test:tools` must pass |
| Config keys and admin objects | The process of `docs/live-ops-90-days.md` 6.2: analyst signal, live-ops proposal on an `area:live-ops` issue with old and new value and the rollback value, product owner approval, change set tried in `staging` on staff test accounts, promoted to `production` (`docs/admin.md` 3.4 to 3.6), verification at the next weekly review |

- A balance change never needs a client release (`docs/roadmap.md` M4 exit criterion). A change that seems to need one is a defect in what the client renders, filed as a bug.
- At most one balance change set per domain per economy week (*default*, as `docs/live-ops-90-days.md` 6.2), so its effect can be attributed.
- A change is proposed only when a measure differs from its reference by more than the tolerance of OQ-9.
- Values approved during the beta are written back to the repo defaults and their spec by PR before the soft launch (AP-3), with the owner's approval because every economy and pass value was owner-approved (`docs/economy.md` 13, `docs/pass-missions-medals.md` 11). The soft launch production config is built from those defaults.

## 6. Beta economy

| Id | Rule |
|---|---|
| EC-1 | The economy in the beta is the real one: the same server, ledger, reward bundles, caps and monitors as at launch (`docs/economy.md`). No beta-only grant path exists |
| EC-2 | Real-money purchases are granted only after server-side receipt validation (NT-5, `docs/economy.md` 7.5). Production rejects sandbox receipts (`docs/anti-cheat.md` V-3), so store test purchases are tested in CB-1 on `staging` with Play license testers and App Store sandbox accounts (CE-5) and never in production |
| EC-3 | In CB-2 and CB-3 the purchase kill switch is on (OQ-5): store purchases are disabled in the shop and no real money is taken from testers. The coin shop, pass levels for coins and the rename stay open, so coin sinks are measured (BT-7) |
| EC-4 | Support grants to testers follow `docs/admin.md` 6.2 with a reason; no tester receives coins or items to "test" a feature outside the normal sources. Testing a feature with granted items is done on staff test accounts in `staging` |
| EC-5 | The beta has its own ranked season and pass season records in production, which end at beta exit. The beta ranked season grants no rank rewards (as the soft launch season, `docs/live-ops-90-days.md` section 2). The launch season records are created separately and are not affected by beta records |

### 6.1 Beta exit and progress

Proposed default (OQ-6, OQ-7), applied after the exit is recorded and before the soft launch T-0:

1. Game state of every production account is reset: coins, inventory, pass and mission progress, medals, rank state per mode, level, XP and statistics. Accounts, nicknames, friends, settings and consent records stay.
2. The reset goes through the normal write paths: ledger entries with their own reason and idempotency key, inventory removal as for `disabled` items (`docs/cosmetics.md` section 6), progress and rank resets by their own services. No SQL and no migration changes a ledger, purchase or inventory row (AP-1, `docs/environments.md` M-8). The ledger reason and the job are added to `docs/economy.md` section 5 and get their own issue when the owner decides OQ-6.
3. Testers then receive the tester reward of OQ-7 through a `targeted` promo campaign (`docs/economy.md` section 8) whose account list holds every account that finished at least 10 matches (*default*) in CB-2 or CB-3. Eligibility uses match records only, never consent state (TP-2). The reward item lists the campaign as its `promo` source (`docs/cosmetics.md` V-15) and is retired after the campaign ends, so it stays unique to the beta.
4. Every beta build is removed from the production build allowlist and the minimum version is raised to the soft launch build (#210), so a kept or leaked beta build gets `UPDATE_REQUIRED` (ADR 0004).
5. The beta notice (TP-3) told testers about steps 1 and 3 before they joined, and a banner (#146) repeats it in the last beta week.

Wiping by recreating the production database is rejected: consent records must outlive the accounts they belong to (`docs/privacy.md` D-23), and the reset must leave an audit trail like any other economy write.

## 7. Exit criteria

The beta exits when every item holds on the exit build, the release candidate build of the soft launch, measured in CB-3. Each item is green or waived by an owner comment on #193 (`docs/roadmap.md` section 4).

| Id | Criterion | Threshold | Source | Owner |
|---|---|---|---|---|
| BX-1 | Crash-free sessions per platform on the exit build | At least 99.5% (#193 acceptance criterion; the level `docs/launch-runbook.md` 4.1 requires at launch), over at least 1,000 sessions per platform (*default*), so one crashed session moves the rate by at most 0.1 point | BS-1 | Client lead |
| BX-2 | Android vitals on the exit build | Crash rate and ANR rate below the Play Console bad behaviour thresholds (`docs/launch-runbook.md` 4.1) | BS-3 | Client lead |
| BX-3 | Match completion per mode | At least 99.5% (*default*): a match only fails to complete on a server or client fault, since leavers are replaced (R-52, R-57), so the bar equals the crash bar | BS-8 | Server lead |
| BX-4 | Resume success | At least 99% (*default*): a resume inside the grace with a valid token only fails on a defect | BS-10 | Server lead |
| BX-5 | Move round-trip | p99 under 100 ms in region (`docs/performance.md`, CLAUDE.md) for every week of CB-3 | BS-11 | Server lead |
| BX-6 | Economy integrity | Zero unexplained ledger reconciliation mismatches and zero unexplained hard signals over CB-2 and CB-3; no open economy integrity incident | BS-15 | Security lead |
| BX-7 | KPI pipeline | D1, D7, active testers and matches per day reported for every weekly cohort of CB-3, and #147 figures match raw events (#147 acceptance), so the soft launch KPI report (#196) can start on day one | BS-5 to BS-7 | Analyst |
| BX-8 | Beta bugs | No open `priority:p0` or `priority:p1` issue found in the beta (search "Found in closed beta (#193)") | GitHub | QA lead |
| BX-9 | Balance | Every BT measure reported against its reference; model keys replaced (5.2); values approved in the beta written back to the repo defaults | Section 5 | Live-ops lead |
| BX-10 | Testers' rights | Every deletion and access request from a tester answered within its deadline (`docs/privacy.md` 8.1) | Support tool | Support lead |
| BX-11 | Wrap-up | Section 6.1 done as the owner decided OQ-6 and OQ-7; sign-up records and invitation lists deleted (TP-4) | Section 6.1 | Release manager |

### 7.1 Exit record

The release manager posts the section 7 table on #193 with a link to the evidence of each item. The product owner records the exit decision as a comment on #193 and closes it. That closed issue, or an owner waiver on it, is the #193 part of GN-6 (`docs/launch-runbook.md` 3.1) and of release gate item 6 (`docs/roadmap.md` section 4).

## 8. Config

This document adds no config key. Used here and defined elsewhere: `ModelMatchMinutes`, `ModelBetweenMatchMinutes`, `ModelWinRatePercent`, `ModelProfileHours`, `ModelProfileMissionPercent` (model keys), `MatchCoins*`, `FirstWinBonusCoins`, `DailyLoginCoins`, `DailyMatchCoinCap`, `DailyMissionCoinBudget`, `WeeklyMissionCoinBudget`, `SinkRatioAlertPercent`, `BalanceMedianAlertCoins`, `AccountDailyEarnAlertCoins` (`docs/economy.md`), `PassMatchXp1v1`, `PassMatchXp2v2`, `PassDailyMatchXpCap`, `DailyMissionSlotXp`, `WeeklyMissionSlotXp`, `PassLevels` (`docs/pass-missions-medals.md`), `TurnSeconds`, `AfkConsecutiveTimeouts`, `MatchLoadingSeconds` (`docs/rules.md`), `CasualBotFillSeconds`, `MmrWindowStart`, `MmrWindowStep`, `MmrWindowStepSeconds`, `MmrWindowMax` (`docs/matchmaking.md`), `MinAccountAgeByCountry` (`docs/privacy.md`), the purchase kill switch, minimum version and build allowlist (#210).

## 9. Implementation and open items

| Issue | Needed for |
|---|---|
| #27, #182 | Signed beta builds with symbols (CE-1), release health (CE-2, BS-1, BS-2) |
| #210 | Build allowlist, purchase kill switch, minimum version (CE-3, EC-3, 6.1) |
| #144 | Support form, tickets, targeted promo lists (FB-1, 6.1) |
| #181, #147 | Analytics events and cohorts (BS-5 to BS-7, BX-7) |
| #28, #45, #48, #50, #87 | Server metrics for BS-8 to BS-12 |
| #110, #101, #202, #209 | Economy monitors and integrity (BS-14 to BS-16) |
| #63, #73, #157, #187, #195 | Privacy entry criteria CE-7 to CE-12 |
| #146 | Beta banners and playtest window announcements |

- `docs/privacy.md` has no rows yet for support tickets, the sign-up record, invitation lists and store tester feedback (CE-11, TP-4). They are added before CB-2.
- The reset job and its ledger reason (6.1 step 2) need an issue and a `docs/economy.md` change once the owner decides OQ-6.
- No issue owns the sign-up form or the external channel; they are set up by the release manager and the support lead once OQ-4 is decided.
- #193 stays open until the exit record (7.1); its acceptance criterion needs real CB-3 data.

## 10. Open questions

Proposed defaults, not decisions. The owner answers on #193; the answer is written into the section it changes and this table records it.

| Id | Question | Proposed default |
|---|---|---|
| OQ-1 | Beta countries and how testers are found | Invite-only by email in every store country except Russia (`docs/privacy.md` 9.3), recruiting players of the four launch languages; no TestFlight public link; no tester in Turkey before L-4 holds |
| OQ-2 | Tester count per phase | CB-1 the whole team; CB-2 up to 100; CB-3 up to 1,000 (section 1) |
| OQ-3 | Phase lengths | CB-1 at least 1 week, CB-2 at least 2 weeks, CB-3 at least 3 weeks and until section 7 is green |
| OQ-4 | Sign-up form and external channel | A sign-up form asking only the fields of 2.2; one private, invite-only community server with a channel per launch language, moderated by the support lead |
| OQ-5 | Real-money purchases in CB-2 and CB-3 | Off: purchase kill switch on in production for the whole beta; store purchases tested in CB-1 on `staging` with sandbox and license tester accounts only. Alternative: purchases on for Play closed testing with real money, which then makes OQ-6 a refund or re-grant question |
| OQ-6 | Beta progress at the soft launch | Full game-state reset of every account, accounts kept (6.1) |
| OQ-7 | Tester compensation | One exclusive cosmetic, proposed an `epic` frame, through a `targeted` promo campaign to accounts with at least 10 finished beta matches, never sold or granted again; no coins |
| OQ-8 | Confidentiality | No non-disclosure agreement; testers may talk about the beta; builds may not be redistributed, which the build allowlist enforces anyway |
| OQ-9 | Tolerance before a balance change | A change is proposed when a measure differs from its reference by more than 20%; smaller gaps wait for soft launch data |
| OQ-10 | Ranked play and playtest windows | Ranked open from CB-2 in the beta ranked season (EC-5); two announced playtest windows per week so queues fill without bots |
| OQ-11 | Where the weekly report is kept | In the team's private workspace, outside the public repository; #193 gets the section 7 table only |
