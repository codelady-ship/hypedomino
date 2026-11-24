# Launch content gate

The minimum catalogue a launch needs, what "ready" means for every piece of it, and the admin content gate report that checks it. The report is the evidence for release gate 3 (`docs/roadmap.md` section 4) and go/no-go item GN-3 (`docs/launch-runbook.md` section 3.1).

**Status: proposal.** The minimum counts in section 2 are the owner's (#198). Every other rule here is a proposal the owner approves by merging the pull request that adds or changes it; open owner decisions are section 7. Every minimum is admin config (ADR 0006); names in `CamelCase` code font are config keys (section 5).

Owned elsewhere and only referenced here: item shape, sources, availability and validation V-1 to V-19 (`docs/cosmetics.md`), rarity tiers and status colours (`docs/rarity.md`), pass seasons, missions and medals with PV-1 to PV-10 (`docs/pass-missions-medals.md`), ranked seasons (`docs/ranking.md`), reward bundles and promo codes (`docs/economy.md`), launch languages and string approval (`docs/localization.md`), admin objects, roles, publish and audit (`docs/admin.md`). No price, duration, colour or reward value is defined here.

## 1. Scope

| In | Out (owner) |
|---|---|
| Cosmetic and medal counts, item readiness, pass season 1 with its pass products, ranked season 1, mission pools, promo code templates | Crystal packs, the starter offer and the licensed test purchase (GN-22); performance budgets (gate 2); UI, banner and FAQ strings (GN-18, GN-26); kill switch and maintenance states (GN-17); leaderboard emblem art, needed only at the first season end (#178, `docs/ranking.md` 8.3) |

`launchAt` is the planned T-0 (`docs/launch-runbook.md` section 4) in server UTC, entered when the report is run. At the first launch, pass season 1 is `season_1` and ranked season 1 is the first ranked season.

## 2. Minimum catalogue

| Id | Category | Counts | Key | Default |
|---|---|---|---|---|
| MC-1 | Removed 2026-11-12 (characters; the game has no characters) | | | |
| MC-2 | Animated avatars | Ready `avatar` items with `animated` true | `ContentGateMinAvatarsAnimated` | 30 |
| MC-3 | Static avatars | Ready `avatar` items with `animated` false | `ContentGateMinAvatarsStatic` | 60 |
| MC-4 | Frames | Ready `frame` items | `ContentGateMinFrames` | 20 |
| MC-5 | Tables | Ready `table` items | `ContentGateMinTables` | 15 |
| MC-6 | Emoji | Ready `emoji` items | `ContentGateMinEmoji` | 40 |
| MC-7 | Medals | Ready medals | `ContentGateMinMedals` | 40 |
| MC-8 | Removed 2026-11-12 (lobby actions; the `action` type is removed) | | | |

- An item counts once, in the category of its `type` and `animated` flag, whatever its sources: `default` items (static, `docs/cosmetics.md` V-19), pass track items, shop, mission, promo, rank reward and event items all count.
- Only items in the active production manifest or progression domain count. A draft, a scheduled change set and a `retired` or `disabled` item do not.
- An animated item whose animation fails a readiness rule does not count as a static item instead; it is not ready (CG-8).

### 2.1 Rarity distribution

Owner decision OD-1: MC-2 to MC-5 each need at least one ready item of every tier from `common` to `legendary` (`docs/rarity.md` 1). `mythic` is not required, since it is sold only in time-limited offers. Emoji and medals have no per-tier minimum. Beyond that the report shows the count per category and `rarityId` as information. Constraints that already hold through other specs:

| Constraint | Source |
|---|---|
| Every type has at least one `live` `default` item, and every `default` item is static | `docs/cosmetics.md` V-11, V-19 |
| No `mythic` item on a pass track | `docs/pass-missions-medals.md` PV-10 |
| `mythic` items are sold only in time-limited offers | `docs/economy.md` 3.1 |
| No `mythic` medal at launch | `docs/pass-missions-medals.md` 5.3 |
| The starter offer needs a `rare` avatar and a `rare` frame; the first-purchase bonus an `epic` frame | `docs/economy.md` 7.3 |

## 3. Ready

A thing is ready only when every rule of its table holds in production at report time. "Four languages" means an `approved` value in `az`, `en`, `tr` and `ru`, present in the active bundle version of each language (`docs/localization.md` 2.1, 2.2). This is stricter than publish (V-16 needs only `en`): at launch no player meets a fallback string.

### 3.1 Cosmetic item

| Id | Rule | Source |
|---|---|---|
| CR-1 | In the active production manifest with `availability` `live` | `docs/cosmetics.md` 6, 7.3 |
| CR-2 | V-1 to V-19 pass when the report runs them again on current data. This includes `culturalReview` `approved`, `artSource` recorded, every source resolving to an existing record (V-15) | `docs/cosmetics.md` 9 |
| CR-3 | `rarityId` resolves in the active rarity table, so the item has its rarity marks and status colour (the tier's `fill`; the manifest has no colour field) | `docs/rarity.md` 1, 7; V-14 |
| CR-4 | `staticAddress` (the static fallback) and, if `animated`, every animation address (`animationAddress`, every layer) resolve to an asset in the Addressables catalog version active in production | `docs/cosmetics.md` 8, V-2, V-17; ADR 0002 |
| CR-5 | Every asset of the item passes its server checks (format, size, atlas, frame count, static present) | `docs/admin.md` 5.2, #93, #135 |
| CR-6 | `nameKey` in four languages | `docs/localization.md` 2.1 |

### 3.2 Medal

| Id | Rule | Source |
|---|---|---|
| MR-1 | Published in the active `progression` domain with `availability` `live` | `docs/pass-missions-medals.md` 5.1 |
| MR-2 | PV-8 passes when run again; `rarityId` resolves in the active rarity table and is not `mythic` | `docs/pass-missions-medals.md` 5.3, 9.1 |
| MR-3 | Every source resolves: a `live` achievement mission, an event mission of a published event, or a rank reward of a ranked season | `docs/pass-missions-medals.md` 5.1, 5.2 |
| MR-4 | `staticAddress` resolves in the active Addressables catalog version and its asset passes its server checks | `docs/admin.md` 5.2, 5.9 |
| MR-5 | `nameKey` and `descriptionKey` in four languages | `docs/pass-missions-medals.md` 5.1 |

### 3.3 Mission

Missions are the templates the server draws per-player assignments from (`docs/pass-missions-medals.md` 4.4); there is no separate template object.

| Id | Rule | Source |
|---|---|---|
| MS-1 | Every `live` mission is published in the active `progression` domain, and PV-5 and PV-7 pass when run again | `docs/pass-missions-medals.md` 4.2, 9.1 |
| MS-2 | Every daily slot 1 to `DailyMissionSlots` has at least max(2, `DailyMissionRerolls` + 1) `live` missions, and every weekly slot 1 to `WeeklyMissionSlots` at least max(2, `WeeklyMissionRerolls` + 1). Two let the draw avoid the previous period's mission; rerolls + 1 let every reroll find a mission not had in the period | `docs/pass-missions-medals.md` 4.4 |
| MS-3 | `nameKey` in four languages | `docs/pass-missions-medals.md` 4.2 |
| MS-4 | An achievement or event mission's `bundle` resolves; every cosmetic in it is ready (3.1) and every medal ready (3.2) | `docs/economy.md` 4 |

### 3.4 Pass season 1

The pass season whose window contains `launchAt`.

| Id | Rule | Source |
|---|---|---|
| PS-1 | Published in the active `progression` domain; PV-1 to PV-10 pass when run again | `docs/pass-missions-medals.md` 9.1 |
| PS-2 | Window equals the window of ranked season 1 (3.5) | `docs/pass-missions-medals.md` 2.1, OQ-2 |
| PS-3 | Every level 1 to `PassLevels` has a premium bundle; every cosmetic on either track is ready (3.1) | `docs/pass-missions-medals.md` 2.4 |
| PS-4 | `pass_premium` and `pass_premium_plus` are bound to the season and their `storeStatus` is active in both stores | `docs/admin.md` 5.6, 5.7 |
| PS-5 | `nameKey` in four languages | `docs/admin.md` 5.7 |

### 3.5 Ranked season 1

The ranked season whose window contains `launchAt`.

| Id | Rule | Source |
|---|---|---|
| RS-1 | `state` is `live`, or `scheduled` with `start` at or before `launchAt`; `end` is after `launchAt` | `docs/admin.md` 5.12 |
| RS-2 | The `ranking` publish checks pass when run again, for both modes | `docs/admin.md` 3.3, `docs/ranking.md` 3.1, 7.1, 7.2 |
| RS-3 | `rankRewards` has a bundle for every mode and every final tier; every cosmetic and medal in those bundles is ready | `docs/ranking.md` 2, `docs/economy.md` 2.6, #120 |
| RS-4 | `LeaderboardRegions` is published | `docs/ranking.md` 10 |
| RS-5 | `nameKey` in four languages | `docs/admin.md` 5.12 |

### 3.6 Promo code template

A template is a promo code record with `template` true (`docs/admin.md` 5.11): it stays a draft, never redeems, and is cloned to issue codes. Cloning adds the window, cap, code and, for `targeted`, the account list, and then follows `docs/economy.md` section 8, including second-admin approval above `PromoCoinsApprovalThreshold`, with a `mythic` item or with `passPremium`. A template does not hold `passPremium`; it is chosen when the code is issued.

| Id | Rule | Source |
|---|---|---|
| PT-1 | The three launch templates of the table below exist | `docs/economy.md` 8 |
| PT-2 | `campaignId` is set; `bundle` resolves and is not a store product | `docs/economy.md` 8, `docs/admin.md` 5.11 |
| PT-3 | Every cosmetic in the bundle is ready and lists the template's campaign as a `promo` source | `docs/cosmetics.md` 5, V-15 |

Launch templates (owner decision OD-2):

| Template | Type | Bundle | Cloned |
|---|---|---|---|
| Welcome | `multi_use` | One launch cosmetic and a coin amount at or below `PromoCoinsApprovalThreshold` | Once per launch campaign |
| Creator | `multi_use` | One cosmetic exclusive to the creator's campaign | Once per creator, each clone with its own `campaignId`, so redemptions are reported per creator |
| Compensation | `single_use` | Set at cloning by the product owner's compensation decision (`docs/launch-runbook.md` 6.5) | Per incident, as a generated batch |

The compensation template's bundle is filled at cloning, so PT-2 and PT-3 check it on the clone, not on the template.

## 4. Content gate report

Admin screen A-41 (`docs/admin.md` section 10). Computed by the admin API from published data only (AP-2); the browser shows the result.

### 4.1 Inputs

| Input | Source |
|---|---|
| Environment | `production`; a `staging` run is a rehearsal and never evidence |
| `launchAt` | Entered by the runner, UTC |
| `configVersion` and every domain revision | The active snapshot (`docs/admin.md` 3.1, 5.17) |
| `manifestVersion` | The `cosmetics` domain of that snapshot |
| Addressables catalog version | The catalog version active in production (ADR 0002, #26) |
| String bundle version per language | The `localization` domain of that snapshot |
| Promo templates | Promo code records (not snapshot content, `docs/admin.md` 3.1) at run time |
| Thresholds | Section 5 keys of the active snapshot and their repo defaults |

### 4.2 Checks

| Id | Check | Pass when |
|---|---|---|
| CG-2 to CG-7 | MC-2 to MC-7 | Count of ready items at or above the key's value; for MC-2 to MC-5 also at least one ready item per tier `common` to `legendary` (2.1). CG-1 was removed with MC-1 |
| CG-8 | Every `live` cosmetic and medal is ready | No `live` item fails a CR or MR rule. Failures list the item id and rule id |
| CG-9 | Missions | MS-1 to MS-4 hold |
| CG-10 | Pass season 1 | A pass season contains `launchAt` and PS-1 to PS-5 hold |
| CG-11 | Ranked season 1 | A ranked season contains `launchAt` and RS-1 to RS-5 hold |
| CG-12 | Promo templates | PT-1 to PT-3 hold for every template counted by PT-1 |
| CG-13 | Thresholds | No section 5 key is below its repo default |

- Each check is `pass` or `fail` and lists every failing object with the rule id it broke. No check is skipped: missing data is a `fail`.
- Information only, never pass or fail: counts per category and `rarityId` (2.1), `live` items not ready per category, drafts of each category.

### 4.3 Verdict and record

- The verdict is `green` when every check passes, otherwise `red`.
- A run stores an immutable report record: id, environment, `launchAt`, run time (server UTC), runner, every version of 4.1, every threshold with its value, repo default and the `configVersion` that set it, every check with its result and failures, the verdict. Records are never edited or deleted; a JSON export is attached to the release issue.
- A report is stale once any version of 4.1 it read is no longer active. The screen marks it stale; a stale report is not evidence.
- GN-3 evidence: a `green`, non-stale `production` report run after the config freeze starts (T-3d) and still not stale at the go/no-go meeting. A content or config publish after the run makes it stale, so the report is run again.

### 4.4 Access, override and audit

| Item | Rule |
|---|---|
| Run | `super_admin`, `content`, `live_ops` (`docs/admin.md` 2.3). Running reads only and needs no approval |
| View | Every role. The report holds no personal data |
| Audit | Each run is an audit entry: action `create`, object `content_gate_report` with the report id, the `configVersion` read (`docs/admin.md` section 7) |
| Thresholds | Changed only by a `liveops` config publish (`docs/admin.md` section 3), which is audited and, during the config freeze, approved by the release manager. Lowering one below its repo default is allowed but fails CG-13, so editing a number never turns the report green; the repo default changes by PR to this document |
| Override | No admin action marks a check passed or the verdict `green`. A `red` report is accepted only by a product owner waiver: a comment on #198 naming the report id and each waived check id (`docs/roadmap.md` section 4, `docs/launch-runbook.md` section 3). The release manager links the report and the waiver at GN-3 |

## 5. Config

`liveops` domain (`docs/admin.md` 3.1, 5.14), admin API only, never sent to the client. Defaults are the launch values; integers >= 0.

| Key | Default | Section |
|---|---|---|
| `ContentGateMinAvatarsAnimated` | 30 | 2 |
| `ContentGateMinAvatarsStatic` | 60 | 2 |
| `ContentGateMinFrames` | 20 | 2 |
| `ContentGateMinTables` | 15 | 2 |
| `ContentGateMinEmoji` | 40 | 2 |
| `ContentGateMinMedals` | 40 | 2 |

Implemented in `server/internal/liveops` (#145): the admin API edits, publishes and rolls back these keys at `/config/liveops` (`docs/admin.md` 9.1), and `config-publish -domain liveops` checks a snapshot the same way. A snapshot with an unknown key, a value that is not an integer or a value below 0 is refused; a key it leaves out keeps its default.

Used here and defined elsewhere: `DailyMissionSlots`, `WeeklyMissionSlots`, `DailyMissionRerolls`, `WeeklyMissionRerolls`, `PassLevels` (`docs/pass-missions-medals.md`), `PromoCoinsApprovalThreshold` (`docs/economy.md`), `LeaderboardRegions` (`docs/ranking.md`).

## 6. Implementation and open items

- No sub-issue of Epic 12 (#133) owns the report yet; it needs one. It reuses the publish validators of `docs/admin.md` 3.3 rather than a second copy of the rules, and builds on #134 (roles, audit), #135 (assets, catalog version), #136 and #137 (catalogue), #140, #141, #142, #143, #145.
- Its tests: for each CG check a seeded production-like snapshot that fails it by one missing condition and one that passes; a lowered threshold gives `red`; a publish after a run makes the run stale.
- The medal definition has no cultural review field, unlike cosmetics (`docs/cosmetics.md` 3), while `docs/localization.md` 6.1 requires the review for every asset. MR rules cannot check it until #141 adds the field.
- #198 is closed when a `green` production report (or a waived `red` one) is linked at GN-3.

## 7. Owner decisions

Decided by the owner on 2026-10-26.

| Id | Question | Decision |
|---|---|---|
| OD-1 | Per-rarity minimums per category | MC-2 to MC-5 (MC-1 removed 2026-11-12): at least one ready item per tier `common` to `legendary` (2.1) |
| OD-2 | Promo templates at launch | Welcome, creator and compensation templates (3.6) |
