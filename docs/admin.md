# Admin panel

Roles and permissions, every object the admin panel manages with its fields and lifecycle, the draft, preview, publish and schedule workflow over versioned config snapshots, validation, rollback, live controls, player tools, the audit log, the two-person rule, security, and the screen list. Architecture is ADR 0006: a separate Next.js app that talks only to the admin API; the admin API validates and writes config, the game server reads published snapshots.

Owned elsewhere and only referenced here: item shape (`docs/cosmetics.md`, `docs/cosmetics/schema.json`), rarity (`docs/rarity.md`), economy values and rules (`docs/economy.md`, `docs/economy/balance.csv`), ranking (`docs/ranking.md`), pass, missions and medals (`docs/pass-missions-medals.md`), penalties (`docs/penalties.md`), anti-cheat responses (`docs/anti-cheat.md`), personal data and its access (`docs/privacy.md`), strings and moderation lists (`docs/localization.md`), kill switch procedures (`docs/launch-runbook.md`), the post-launch live-ops calendar (`docs/live-ops-90-days.md`). No colour, price, duration or reward value is defined here. Names in `CamelCase` code font are config keys; section 11 lists the keys this document adds. Implementation is Epic 12 (#133); section 12 maps sections to its sub-issues.

## 1. Principles

| Id | Rule |
|---|---|
| AP-1 | The admin panel writes only through the admin API (ADR 0006). No admin, operator or tool edits game data in PostgreSQL or Redis directly. Ledger mistakes are corrected with ledger entries (`docs/economy.md` section 5), never with SQL or a restore (`docs/launch-runbook.md` 5.4). |
| AP-2 | The server is authoritative here too. Every permission, validation, derived value (guide price, bundle value, budget totals) and timestamp is decided by the admin API. Checks in the browser are convenience only. |
| AP-3 | Every tunable number is a config value with a schema in the repo (ADR 0006: repo files hold defaults and schema). The panel edits values; adding a key, a type or a field is a code change. |
| AP-4 | Config changes reach players only through a published, immutable config snapshot (section 3). Matches record the snapshot version they started with and finish on it. |
| AP-5 | Published things are never deleted. Items, bundles, offers, keys and ids are retired or disabled and their ids stay reserved (`docs/cosmetics.md` section 6). Only a draft that was never published can be deleted. |
| AP-6 | Every write and every view of a player record is in the audit log (section 7). Sensitive actions carry a mandatory reason. |
| AP-7 | Least privilege. A role sees only the screens and data of section 2; a request outside its permissions is rejected by the admin API, not only hidden in the UI. Content and live-ops roles see no personal data (`docs/privacy.md` section 3). |
| AP-8 | All times are server UTC (ADR 0007). The panel shows and takes times in UTC and labels them so. |

## 2. Roles and permissions

### 2.1 Roles

| Role | Does | Personal data (`docs/privacy.md` section 3) | Usually held by (`docs/launch-runbook.md` section 1) |
|---|---|---|---|
| `super_admin` | Admin accounts and roles, retention and privacy config, anti-cheat config, audit log, every permission of the other roles | Every row any admin role can see, plus D-2, D-27, D-28 | Product owner, security lead |
| `content` | Cosmetics, assets, rarity, medals, emblem art, game strings and their translations, cultural review | None | Art and content team |
| `live_ops` | Economy config, reward bundles, shop, chests, crystal packs, pass, missions, daily login, promo codes, ranked seasons and ranking config, penalty and rules config, events, banners, push campaigns, feature flags, live controls | None | Live-ops lead, release manager, client lead for version gates |
| `support` | Player lookup and support actions, sanctions, review, report and appeal queues, moderation lists, rights request records, targeted promo account lists | D-1, D-3, D-5, D-6, D-8 to D-10, D-12, D-13, D-15 to D-21, D-23 | Support lead, support agents, moderators |
| `analyst` | Dashboards, KPI and economy monitors, aggregate exports, pseudonymous analytics events | D-24 only; every other number as an aggregate without ids | Analyst |

- Roles are granted per environment (staging, production) to named accounts. One account may hold several roles; permissions add up.
- Infrastructure access (`eng` in `docs/privacy.md`) is not an admin role. The panel never shows IP addresses (D-7), server logs (D-26), crash reports (D-25) or backups (D-30).
- Runbook roles map to admin roles through the accounts that hold them. A live-ops lead who corrects a player's ledger (`docs/launch-runbook.md` 9.2) needs the `support` role for that.

### 2.2 Actions

| Code | Action |
|---|---|
| V | View the object and its history |
| C | Create a draft (including clone) |
| E | Edit a draft |
| P | Publish or schedule a change set holding the object, and approve one (section 3.5) |
| R | Retire or disable a published object |
| D | Delete a draft that was never published |
| S | The sensitive actions named in the cell; each needs a reason (section 7) and some need a second admin (section 8) |

### 2.3 Permission matrix

An empty cell means no access: the screen is not in the navigation and the admin API rejects the request.

| Object (section) | `super_admin` | `content` | `live_ops` | `support` | `analyst` |
|---|---|---|---|---|---|
| Cosmetic items (5.1) | VCEPRD, S: cultural review | VCEPRD, S: cultural review | V; R and P for an incident switch-off | V | V |
| Assets (5.2) | VCED | VCED | V | | |
| Rarity table (5.3) | VCEP | VCEP | V | V | V |
| Reward bundles (5.4) | VCEPRD | V | VCEPRD | V | V |
| Shop offers and rotations (5.5) | VCEPRD | V | VCEPRD | V | V |
| Store products and crystal packs (5.6) | VCEPR | | VCEPR | V | V |
| Pass seasons (5.7) | VCEPRD | V | VCEPRD | V | V |
| Missions (5.8) | VCEPRD | V | VCEPRD | V | V |
| Medals (5.9) | VCEPRD | VCEPRD | V | V | V |
| Leaderboard emblems (5.10) | VCEPR | VCEPR | V | V | V |
| Promo codes (5.11) | VCEPRD, S: approve | | VCEPRD, S: approve, disable | V, S: targeted account list | V (counts) |
| Ranked seasons (5.12) | VCEPRD | | VCEPRD | V | V |
| Live events, banners, push campaigns (5.13) | VCEPRD | V | VCEPRD | V | V |
| Config values: economy, rules, penalties, ranking, effects, flags (5.14) | VCEP | V | VCEP | V | V |
| Config values: anti-cheat (5.14) | VCEP | | | | |
| Config values: retention and privacy (5.14) | VCEP | | | | |
| Config values: moderation and nickname (5.14) | VCEP | | | VCEP | |
| Game strings (5.15) | VCEPRD | VCEPRD | VCE | V | |
| Word filter and reserved names (5.16) | VCEPRD | | | VCEPRD | |
| Live controls (section 4) | V, S: switch | | V, S: switch | V | V |
| Config snapshots and change sets (5.17) | V, S: rollback | V, S: rollback (own domains) | V, S: rollback (own domains) | V | V |
| Players (section 6) | V, S: all of section 6 | | | V, S: all of section 6 | |
| Review, report and appeal queues (6.4) | V, S | | | V, S | |
| Rights request records (6.5) | V, E | | | VCE | |
| Content gate reports (`docs/launch-content-gate.md` 4.4); C is running a report | VC | VC | VC | V | V |
| Dashboards and aggregate exports (section 10, A-1) | V | V | V | V | V |
| Analytics events with analytics id (D-24) | V | | | | V |
| Audit log (section 7) | V | V (entries on objects it can view) | V (same) | V (same, no player entries) | |
| Admin accounts and roles (5.18) | VCER, S: grant `super_admin` | | | | |

- A change set is published only by an account that holds P on every object in it.
- The matrix is enforced by the admin API per request. #134's acceptance test checks that an account without the role can neither see nor perform each action.
- The matrix lives in `server/internal/rbac`. Its test reads this table, so a change to a cell fails CI until the code matches. A new qualifier or `S` phrase needs a named action there.

## 3. Workflow

### 3.1 Domains and snapshots

Published config is a sequence of immutable snapshots per environment. A snapshot holds every domain below at one revision; `configVersion` increases by 1 on every publish.

| Domain | Content | Delivered to the client |
|---|---|---|
| `cosmetics` | Cosmetics manifest (`manifestVersion`, `docs/cosmetics.md` 7.3); `LobbyActionSlots` (`docs/friend-lobby.md` section 5) | Manifest file on the CDN; slot count with client config |
| `rarity` | Rarity table (`docs/rarity.md` section 7) | With client config |
| `economy` | `docs/economy/balance.csv` keys of kind `config`, reward bundles, shop offers and rotations, store product mapping, daily login | Shop and reward views from the server |
| `progression` | Pass seasons, missions, medals | From the server |
| `ranking` | `docs/ranking.md` section 10 keys per mode, ranked seasons, reset tables, rank reward mapping, emblem definitions | From the server |
| `rules` | `TurnSeconds`, `AfkConsecutiveTimeouts`, `MatchLoadingSeconds` (`docs/rules.md` section 13). The other `rules.Config` fields change only with `docs/rules.md` and code | Rule values from the server |
| `matchmaking` | `docs/matchmaking.md` section 3, including the bot level of ranked and casual matches | From the server |
| `penalties` | `docs/penalties.md` section 8 | From the server |
| `anticheat` | `docs/anti-cheat.md` section 7 except live controls | Never |
| `privacy` | `docs/privacy.md` section 4 and section 11 | Policy and age values only |
| `moderation` | Word filter, reserved names, confusables table version, nickname and report keys (`docs/localization.md` section 5) | Never |
| `localization` | Active bundle version per language, `LocaleFallbackChains` | Bundle files on the CDN |
| `liveops` | Live events, banners, push campaign definitions, feature flags, effect level and motion config (`docs/ui-motion.md` sections 7, 8), content gate minimums (`docs/launch-content-gate.md` section 5) | With client config, except the content gate minimums |

- The game server loads a snapshot, validates it again (`docs/cosmetics.md` section 9) and swaps it in without a restart. A server that rejects a snapshot keeps its current one and alerts.
- A snapshot records its parent version, its change set, author, approver, activation time and the revision of every domain. Files it references on the CDN (manifest, string bundles) are immutable and content-hashed (ADR 0002).
- Live controls (section 4), promo codes and player records are not part of a config rollback: they change immediately and a rollback never touches them. The controls of `docs/anti-cheat.md` 5.2 are kept in their own `live_controls` snapshot domain, which has no rollback route.

### 3.2 States

```
draft -> submitted -> approved -> [scheduled] -> active -> superseded
```

| State | Meaning |
|---|---|
| `draft` | A change set: one or more object drafts, editable by their authors. Validation runs on every save and shows errors inline |
| `submitted` | Frozen and validated; waiting for approval. Any edit returns it to `draft` |
| `approved` | Approved per section 3.5: by its author, or by a second admin for a section 8 action. Publish now goes straight to `active` |
| `scheduled` | Approved with a future activation time |
| `active` | Its snapshot is the current one |
| `superseded` | A later snapshot is active |
| `rejected`, `cancelled` | Closed without activation, with a reason |
| `failed` | Validation failed at activation; nothing changed |

### 3.3 Draft and validation

- An object draft records the published revision it was based on. If that object is published again before this change set, publish is rejected as stale and the author rebases the draft on the new revision.
- Two open change sets may hold the same object; the first to activate wins and the other becomes stale.
- Validation runs at save, at submit, at approval and again at activation against the snapshot that will be its parent. An invalid change set is never published (ADR 0006).

| Domain | Checks at publish | Reference |
|---|---|---|
| `cosmetics` | Schema and V-1 to V-11, V-19 for the whole manifest; V-12 to V-18 across manifests and other admin data | `docs/cosmetics.md` section 9; `tools/cosmetics-schema` runs the single-manifest rules on repo examples |
| `rarity` | Ids, ordinals 1 to N, unique pips, glow order, contrast on all ground layers, no removed id still referenced | `docs/rarity.md` section 7; `tools/design-tokens` runs the same rules on the defaults |
| `economy` | Offer price is coins or a store product, never both; `mythic` only in time-limited offers; struck-through price is the lowest of `ReferencePriceDays`; bundle has a non-empty field; daily and weekly mission sets within budget | `docs/economy.md` 3.1, 4, 6.2, 6.3, 2.4; `tools/economy-balance` regenerates the balance sheet from the CSV defaults |
| `progression` | PV-1 to PV-10: free track coins within `PassFreeTrackCoins`, premium track crystals within `PassPremiumTrackCrystals`; every premium level has a bundle; no `medals`, `passXp` or `mythic` item on a track; no overlapping pass seasons and no change to a frozen field of a running one; mission slots within their coin and XP budgets; event mission XP within `EventMissionXpBudget`; medal sources only `mission` or `rank_reward` | `docs/pass-missions-medals.md` 9.1, `docs/economy.md` 2.5; `tools/pass-season` checks the sample season against the curve and the track budgets |
| `ranking` | `LpChangeMax` + `OffenseRatingPenalty` below `DivisionLp`; reset table monotone and never above the old rank; structural keys unchanged while a season runs; seasons contiguous | `docs/ranking.md` 3.1, 7.1, 7.2 |
| `localization` | Only approved values in a bundle; placeholders match the key's declaration; plural categories complete per language; value within the key's max length; `en` complete; `en` last in every fallback chain | `docs/localization.md` 1.2, 2.1, 4.1 to 4.3 |
| All | Every reference resolves (bundle, offer, item, medal, mission, product); every record that grants an item is listed in the item's `sources`; every value within its schema type and bounds; every `nameKey` has an approved `en` value | `docs/cosmetics.md` V-15, V-16, AP-3 |

The admin API implements these rules in Go. The tools in `tools/` check the repo defaults with the same rules; both run the same test vectors so they do not drift.

### 3.4 Preview

| Preview | What it shows |
|---|---|
| Item render | The item as the client draws it: static and animated, at the low, medium and high effect levels, with its rarity diamond and halo (`docs/rarity.md` 2.3), and with reduce motion on (#136) |
| Text | Every localized field in `az`, `en`, `tr`, `ru` with the expansion budget of `docs/localization.md` 4.1 marked |
| Diff | Field-level before and after for every object in the change set, and every object that references a changed one (a changed bundle lists every offer, mission, pass level and code that grants it, #138) |
| Economy impact | For changes to `economy` or `progression`: budget totals and the guide value of each changed bundle, computed by the admin API |
| Staging | The change set applied to staging, played with staff test accounts. Production never serves an unpublished change set |

### 3.5 Approval and publish

- Production: the author approves their own change set; validation, the audit log and rollback are the controls (owner decision O-1). Only the high-risk actions of section 8 need a second admin holding the same permission. Staging: the author approves.
- During the launch config freeze (`docs/launch-runbook.md` section 4) the approver is the release manager.
- Publish now activates at approval. Schedule sets an activation time in UTC; the change set appears in the live-ops calendar (#146) and can be cancelled until it activates. Files for the CDN are uploaded at approval, so activation only switches the active version.
- Time windows inside content (offer windows, event windows, season start and end) need no scheduled publish: they are part of the published object and run on server time (`docs/economy.md` EP-6).

### 3.6 Promotion between environments

A change set is built and tested in staging, then exported and imported into production as a new change set with the same content. Production validates it again and needs its own approval. Ids are the same in both environments.

### 3.7 Rollback

- A rollback picks an earlier snapshot and one or more domains and creates a change set with that content. It is validated and approved like any change set and publishes a new `configVersion`; history is never rewritten (ADR 0006).
- Matches in progress finish on their recorded version (`docs/launch-runbook.md` 5.4, target under 5 minutes).
- Cosmetics: items added after the target version stay in the manifest (V-12) with their current content unless the change set also disables them; changed items get a new revision with the earlier content (`docs/cosmetics.md` 7.1).
- Localization: switching the active bundle version is a rollback of the `localization` domain (`docs/localization.md` 2.2).
- A rollback never reverses grants, purchases, ledger entries or player progress. Corrections to players are support actions (section 6).

## 4. Live controls

Controls that must act within minutes and must not be undone by a config rollback. Each change is audited with a reason and, in production, follows section 8.

| Control | Effect | Defined in |
|---|---|---|
| Purchase kill switch | Store purchases disabled; validated receipts still granted | `docs/launch-runbook.md` 5.5, #210 |
| Ranked kill switch | Ranked queues closed; matches in progress finish | #210 |
| Minimum version, build allowlist | Older or unlisted builds get `UPDATE_REQUIRED` | #210, ADR 0004 |
| Maintenance mode | Maintenance screen with its end time; no new matches; players in a match finish it (`docs/anti-cheat.md` 5.2) | #146 |
| Promo code disable | One code stops redeeming; granted rewards stay | `docs/economy.md` section 8 |

## 5. Managed objects

Each table lists the fields the admin edits. Fields marked "server" are set by the admin API and shown read-only. Every object also carries `createdBy`, `createdAt`, `updatedBy`, `updatedAt` and its published revision. Ids are set at creation and immutable (AP-5). Unless a spec fixes another form (store products, rarity tiers, string keys), an id is `<prefix>.<slug>` with slug `[a-z0-9_]+`.

### 5.1 Cosmetic items: avatar, character, frame, table, emoji, action

One form engine driven by `docs/cosmetics/schema.json` (#136) with type fields per type (#137).

| Field | Types | Admin rule |
|---|---|---|
| `id` | All | `<type>.<slug>`, set at create |
| `type` | All | Chosen at create, immutable |
| `revision` | All | Server: +1 on every published change |
| `nameKey` | All | Server: `cosmetic.<id>.name`; the value is edited as a game string (5.15) |
| `rarityId` | All | Picked from the published rarity table |
| `animated` | All | Toggle; switches which type fields are required |
| `staticAddress` | All | Server: derived from the id; the upload must exist (5.2) |
| `availability` | All | `live`, `retired`, `disabled` (`docs/cosmetics.md` section 6) |
| `sources` | All | Picker of `default` and existing offers, pass rewards, missions, campaigns, season rewards, events. A form that grants an item offers to add its source to the item in the same change set |
| `artSource` | All | `handmade` or `ai_assisted` |
| `culturalReview` | All | Set to `approved` only by the cultural review action of an account other than the asset's uploader (`docs/localization.md` 6.1 step 3) |
| `animationAddress` | `avatar`, `frame`, `emoji`, `action` | Server: derived; upload required when animated, always for an action |
| `clips` | `character` | `idle`, `win`, `lose`, `reaction` uploads, all four when animated |
| `layers` | `table` | Ordered list of name and upload with an animated flag, at most `TableMaxLayers` |
| `group` | `emoji` | Emoji group id (#150) |

Lifecycle: create and edit as drafts; publish adds the item to the next manifest; retire or disable instead of delete. Bulk edit of availability and rarity over a filtered list (#136). Disabling an item for a content violation may be done by `live_ops` during an incident.

### 5.2 Assets

| Field | Rule |
|---|---|
| `address` | Server: from item id and slot (`docs/cosmetics.md` 8.1) |
| `slot` | `static`, `anim`, `clip/<name>`, `layer/<name>` |
| `file` | Uploaded through a signed URL to object storage (#135) |
| `format`, `width`, `height`, `bytes`, `frames` | Server: measured and checked against the limits of #93 and #135 |
| `contentHash` | Server |
| `checks` | Server: format, size, atlas, frame count, static present (V-2) |
| `catalogVersion` | Server: the Addressables catalog build that contains it (V-17) |

Lifecycle: upload, check, include in a catalog build, publish with the item. An asset of a published item is never deleted; a new upload makes a new item revision.

Built so far (#135): the `static` slot only, through `/assets`. `POST /admin/assets/uploads` records the row and returns a signed `PUT` URL; `POST /admin/assets/{id}/check` reads the object back and records `format`, `width`, `height`, `bytes`, `frames` and `contentHash`, with `format` (a PNG) and `alpha` (straight alpha, `docs/asset-workflow.md` 5.3) as the checks that can refuse. A rejected row stays listed; a new upload makes a new row. Every step is audited; a signed URL is never stored. Other slots answer `asset_slot_unsupported`; their format is proposed in ADR 0012 and their upload is not built. Size, atlas and frame limits are measured where possible but refuse nothing until O-5 and O-6 of `docs/asset-workflow.md` set them. `catalogVersion`, the catalog build and publishing with the item are not built, so an uploaded file does not reach players yet.

### 5.3 Rarity table

Fields and rules are `docs/rarity.md` section 7: tier id, `ordinal`, `nameKey`, `fill`, `on`, `text`, `pips`, `glow`, `flow`, `diamondMinSize`, `flowPeriodMs`. Values come from `docs/design-tokens.json` as defaults. A tier is never removed while an item references it (V-14). Changing ordinals changes table selection for matches started after the publish.

### 5.4 Reward bundles

| Field | Rule |
|---|---|
| `id` | `bundle.<slug>` |
| `revision` | Server: +1 on every published change; each grant records it |
| `coins` | Integer >= 0 |
| `cosmetics` | Unique item ids; each item lists the granting source (V-15) |
| `medals` | Unique medal ids |
| `passXp` | Integer >= 0 |
| `usedBy` | Server: every offer, mission, pass level, daily login day, season reward, event and promo campaign that grants it |
| `guideValue` | Server: coins plus guide prices of the cosmetics (`docs/economy.md` 3.1) |

Shape and grant rules: `docs/economy.md` section 4. Editing a bundle in use shows `usedBy` and needs confirmation (#138 acceptance). A bundle in use cannot be retired.

Today a bundle is stored inside each object that grants it, not in a catalog of its own. `GET /admin/reward-bundles` derives the list from the published shop, pass, missions, rating and daily login config and from promo codes and templates. The same id holding different content is shown as two rows marked `differs`. Events are not listed because they have no published config yet. `PUT /admin/reward-bundles/{id}` writes the new content to every holder and publishes each affected domain in one transaction, so all of them change or none do. Only bundles held entirely by `shop`, `pass` and `missions` can be edited this way. Rank rewards are excluded because their coins come from `RankRewardCoins`, daily login days because they are coins in a calendar, and promo codes because a code keeps the bundle it was created with. An id with `differs` is refused until its holders agree. Each edit raises the bundle's `revision` above its current one, and the domains keep their own history and rollback.

### 5.5 Shop offers and rotations

| Field | Rule |
|---|---|
| `id` | `offer.<slug>` |
| `kind` | `cosmetic`, `bundle`, `chest`, `crystals`, `pass`, `pass_levels` |
| `section` | `featured`, `daily_rotation`, `weekly_rotation`, `event`, `chests`, `pass`, `crystals` (`docs/economy.md` 6.1) |
| `price` | Exactly one of `priceCoins` (integer > 0), `priceCrystals` (integer > 0) or a store `productId` (`docs/economy.md` 6.2) |
| `bundle` | Bundle id |
| `window` | Start and optional end, UTC |
| `limitPerAccount` | Integer or none |
| `eligibility` | Optional hours since account creation, first purchase only |
| `nameKey` | Game string for the offer title |
| `referencePrice` | Optional struck-through price in the currency of the price (`referencePriceCoins` or `referencePriceCrystals`), validated by `docs/economy.md` 6.3 |
| `regionalOverrideNote` | Text; a store-console override needs owner approval (`docs/economy.md` 7.4) |
| `pool` | Chests only: the `common` and `rare` items of the chest, stored as its bundle cosmetics; the editor offers only published `common` and `rare` items (`docs/economy.md` 3.4) |
| `chests` | Catalog-wide: `rarityPercent` (`ChestRarityPercent`) and `rarePityCount` (`ChestRarePityCount`) |

A rotation is a schedule entry: period (`daily` or `weekly`), period start in UTC, and an ordered list of offer ids, `ShopDailyRotationSlots` or `ShopWeeklyRotationSlots` long. Rotations are planned ahead in the calendar; an unscheduled period repeats the previous one (`docs/economy.md` 6.1). Offers and rotations are the same for every player (EP-8): a variant test may change presentation only, never price, content, window or eligibility (owner decision O-3).

### 5.6 Store products and crystal packs

| Field | Rule |
|---|---|
| `productId` | `crystals_tier1` to `crystals_tier5`, `pass_premium`, `pass_premium_plus`, `offer_starter` (`docs/economy.md` section 7). Created in both store consoles first |
| `crystals` | For crystal packs, entry of `CrystalPackCrystals` |
| `usdReference` | The store price point, recorded for checks; the price itself is set only in the store consoles |
| `offer` | The offer that sells it |
| `storeStatus` | Server: product found and active in each store |

The admin never sets a local price (`docs/economy.md` 7.4). Store products must match admin config before launch (`docs/launch-runbook.md` GN-22).

### 5.7 Pass seasons

Rules: `docs/pass-missions-medals.md` section 2.

| Field | Rule |
|---|---|
| `id` | `season_<n>` |
| `nameKey` | Game string |
| `window` | Start and end, server UTC; defaults to the ranked season's window; never overlaps another pass season |
| `levels`, `curve` | Server: `PassLevels`, `PassXpBase`, `PassXpStep`, `PassXpMax` of the published config when the season is created |
| `xpTable` | Server: XP to next level and XP total per level from the curve (`docs/pass-missions-medals.md` 2.2) |
| `freeTrack`, `premiumTrack` | Per level 1 to `levels`: a bundle id; a free level may be empty, a premium level may not. Track bundles carry `coins` and `cosmetics` only, never `medals`, `passXp` or a `mythic` item; each cosmetic lists the source `pass.<seasonId>.<track>.level_<n>` |
| `products` | `pass_premium`, `pass_premium_plus` bound to the season (`docs/economy.md` 7.2) |
| `budget` | Server: free track coins against `PassFreeTrackCoins`, premium track crystals against `PassPremiumTrackCrystals` |

Lifecycle: create or clone a past season (#140), publish before its start. While it runs, `levels`, `curve` and the level-to-bundle tables are frozen; the only track change is replacing a `disabled` cosmetic with an item of the same type and rarity, with an audited reason. XP source values (`docs/pass-missions-medals.md` 2.3) are config keys (5.14) and may change at any time.

### 5.8 Missions and daily login

Rules: `docs/pass-missions-medals.md` section 4.

| Field | Rule |
|---|---|
| `id` | `mission.<slug>`, never reused |
| `revision` | Server: +1 on every published change |
| `type` | `daily`, `weekly`, `achievement`, `event` |
| `nameKey` | Server: `mission.<slug>.name`, an ICU message with `{target}` and plural forms; the value is edited as a game string (5.15) |
| `event`, `target`, `filters` | A progress event from the server's list (`docs/pass-missions-medals.md` 4.3), a target >= 1, optional `queue` and `mode` filters; built without code (#141) |
| `slot`, `weight` | `daily` and `weekly` only: slot number and selection weight >= 1. The slot fixes the reward: `DailyMissionSlotCoins`, `DailyMissionSlotXp`, `WeeklyMissionSlotCoins`, `WeeklyMissionSlotXp` |
| `bundle` | `achievement` and `event` only: reward bundle id |
| `next` | `achievement` only, optional: the next mission of the chain |
| `eventId` | `event` only: the live event (5.13) |
| `availability` | `live` or `retired` |

The daily login calendar is edited at `/daily-login` (`docs/economy.md` 2.3): the coins of each existing day, under the missions permissions. Until one is published the server has no calendar and refuses claims.

Rotation and reroll are automatic per player and slot; there is no per-mission schedule. Which matches count, including that custom rooms never do: `docs/pass-missions-medals.md` section 3. Daily login: one entry per day of the 7-day cycle, coins from `DailyLoginCoins` only (`docs/economy.md` 2.3). Before publish a mission is tried on a staff test account in staging (#141).

### 5.9 Medals

Rules: `docs/pass-missions-medals.md` section 5.

| Field | Rule |
|---|---|
| `id` | `medal.<slug>`, never reused |
| `revision` | Server: +1 on every published change |
| `nameKey`, `descriptionKey` | Server: `medal.<slug>.name`, `medal.<slug>.description`; the values are edited as game strings (5.15); the description says how the medal is earned |
| `rarityId` | Picked from the published rarity table, guided by the expected earn share (`docs/pass-missions-medals.md` 5.3); the list shows the measured share from #147 |
| `staticAddress` | Server: `medal/<slug>/static`; the upload must exist (5.2) |
| `sources` | At least one: `mission` (achievement or event mission id) or `rank_reward` (#120 reward id) |
| `availability` | `live`, `retired`, `disabled` (`docs/cosmetics.md` section 6) |

Medals are granted only through a bundle of a listed source (`docs/economy.md` section 4); pass tracks, shop offers and promo codes never carry one. Grant rules are #129.

### 5.10 Leaderboard emblems

| Field | Rule |
|---|---|
| `id` | `emblem.<scope>_top<bracket>`, scope `global` or `local` |
| `scope` | `global`, `local` (country and region) |
| `bracket` | An entry of `EmblemBrackets` |
| `nameKey` | Game string; season and board are text parameters, not art (`docs/ranking.md` 8.3) |
| `staticAddress` | One static asset per scope and bracket (#178) |

Awards come only from the season-end job. Revoking an awarded emblem is a support action (6.2).

### 5.11 Promo codes

| Field | Rule |
|---|---|
| `id` | `promo.<slug>` |
| `campaignId` | `campaign.<slug>`, what cosmetic `promo` sources reference |
| `type` | `single_use`, `multi_use`, `targeted` |
| `bundle` | Bundle id; never a store product |
| `window` | Start and end, UTC, at most `PromoMaxValidityDays` |
| `cap` | Required for `multi_use`; server for the other types |
| `code` | Generated (`PromoCodeLength`, cryptographic source) or admin-chosen for `multi_use` only; stored as a keyed hash |
| `accounts` | `targeted` only: account ids, entered by `support` after the code is created; the code is created without them and redeems for nobody until they are listed |
| `template` | True for a template (`docs/launch-content-gate.md` 3.6): it stays `draft`, cannot be submitted or redeemed, and has no window, cap, code or accounts. Cloning it creates a record with `template` false |
| `state` | Server: `draft`, `pending_approval`, `active`, `disabled`, `expired` |
| `creator`, `approver` | Server |
| `redemptions` | Server: count; per-account list visible to `support` only |

Rules are `docs/economy.md` section 8. Batch generation creates `single_use` codes and offers their plaintext once, as a CSV download that is never stored (#142: 10,000 codes in under 10 seconds). A code above `PromoCoinsApprovalThreshold` or with a `mythic` item is active only after a second admin approves it. Disable is immediate (section 4).

### 5.12 Ranked seasons

| Field | Rule |
|---|---|
| `id` | `season.<slug>` |
| `nameKey` | Game string |
| `start`, `end` | UTC; the next season starts at this season's end (`docs/ranking.md` 7.1) |
| `config` | `docs/ranking.md` section 10 keys, one value per mode |
| `resetTable` | `SeasonResetTable` rows, generated from `ResetTierDrop`, `ResetFloor`, `ResetCeiling` and editable per row |
| `rankRewards` | Per mode and final tier: the reward bundle (#120), published as `rankRewards` of the mode in the `rating` config; coins from `RankRewardCoins` |
| `state` | Server: `draft`, `scheduled`, `live`, `ending`, `closed` |

Everything is editable until start (#143). While live, only the non-structural keys of `docs/ranking.md` 7.1 change, and the end may move later, never earlier; the next season moves with it. `LeaderboardRegions` and `LeaderboardTopSize` are ranking config (5.14). Season end runs once per mode (#115); the admin sees its progress and cannot run it twice.

### 5.13 Live events, banners and push campaigns

| Object | Fields |
|---|---|
| Live event | `id` (`event.<slug>`), `nameKey`, `window`, reward bundles, event shop offers (`docs/economy.md` 2.7) |
| Banner | `id` (`banner.<slug>`), `textKey`, `window`, `placement`, audience filter by platform, app version and language |
| Push campaign | `id` (`push.<slug>`), `titleKey`, `bodyKey`, send time UTC, category (`gameplay`, `promotional`), audience filter by country, language and platform |

- Promotional sends reach only accounts with `promotional_push` consent (`docs/privacy.md` 6.2). Live-ops sees audience counts, never recipients (AP-7).
- Every text is a game string key in all four languages before publish.
- Events, banners, rotations, scheduled change sets and season boundaries share one calendar view (#146).

Calendar (A-21). The calendar holds no data of its own: an entry exists because a published or scheduled object has a window or an activation time, and it links to that object and its change set. A planned entry is entered by creating the object on its own screen with its window, or by scheduling a change set (3.5).

| Entry | Object | Screen |
|---|---|---|
| Ranked season start and end | 5.12 | A-19 |
| Pass season window | 5.7 | A-16 |
| Live event window, with its event missions and event shop offers | 5.13, 5.8, 5.5 | A-22, A-17, A-12 |
| Daily and weekly rotation period | 5.5 | A-13 |
| Featured offer window | 5.5 | A-12 |
| Banner window, push send time | 5.13 | A-23, A-24 |
| Promo code window | 5.11 | A-15 |
| Scheduled change set activation | 5.17 | A-2, A-3 |

Built so far (#146): the calendar reads ranked seasons, pass seasons from the published `pass` snapshot, featured and event offers from the published `shop` snapshot and promo campaigns, and lists the ones not yet ended; the panel is `/calendar`, each row linking to its screen. Rotation periods, live events, banners, push send times and scheduled change sets join it when they are built. Every role with view rights on live events sees it; an `analyst` also does, as it views every listed source, but sees promo campaigns only with view rights on promo codes.

The plan for the first 90 days after launch, entered this way, is `docs/live-ops-90-days.md`.

### 5.14 Config values and feature flags

Remote config (#145) is one editor over every key of the domains in 3.1. The `rules` domain keys and their bounds are in `docs/rules.md` section 13 (O-7).

| Field | Rule |
|---|---|
| `key` | From the repo schema; never created in the panel (AP-3) |
| `domain` | Section 3.1 |
| `type`, `unit`, `min`, `max` | From the schema |
| `perMode` | True for ranking keys (one value for 1v1, one for 2v2) |
| `value` | Edited |
| `default` | Repo default, with a link to its spec section |
| `frozenWhileSeason` | True for `docs/ranking.md` 7.1 structural keys; `PassLevels` and the pass XP curve are frozen per pass season by its record (5.7) |

Feature flags are boolean or enum keys of the `liveops` domain with an optional audience filter by platform and app version. Who edits which domain is the matrix in 2.3; anti-cheat values are hidden from every role but `super_admin` because they describe detection (`docs/privacy.md` 8.3).

### 5.15 Game strings

Fields and rules are `docs/localization.md` section 2.1 (#214).

| Field | Rule |
|---|---|
| `key` | `area.screen.item`, never renamed or reused |
| `note` | Translator context: where it shows, what each placeholder is |
| `maxLength` | Per key |
| `placeholders` | Declared names and types |
| `values` | Per language: text and status `missing`, `draft`, `approved` |
| `deprecated` | Marked when retired; removed once no supported client uses it |

- Editor with languages side by side, a missing-value report per language with percentage and key list, CSV and JSON import and export for external translators (#214).
- Setting a value to `approved` is the native-speaker review step (`docs/localization.md` 6.1 step 2) and is done by `content`. Publishing builds one bundle per language (5.17).

### 5.16 Word filter and reserved names

Owned by `support` as moderators (`docs/localization.md` 5.3): entries per list (`az`, `en`, `tr`, `ru`, language-independent) with match type and category, `ReservedNames`, and the confusables table version. Each publish increases the list version and triggers lazy revalidation of stored names at login (`docs/localization.md` 5.6). Entries are visible only to `support` and `super_admin`.

### 5.17 Config snapshots and change sets

| Field | Rule |
|---|---|
| `configVersion` | Server: integer, +1 per publish per environment |
| `environment` | `staging` or `production` |
| `parentVersion` | Server |
| `domains` | Server: revision of every domain; `manifestVersion`; bundle version per language |
| `changeSet` | Id, title, author, objects with before and after |
| `approver` | Server; never the author in production |
| `activationTime` | Approval time or the scheduled UTC time |
| `rollbackOf` | Optional: the snapshot whose content it restores |
| `state` | Section 3.2 |
| `validationReport` | Server: rules run and results |

Snapshots are read-only. Change sets are created, edited, submitted, approved, scheduled and cancelled as section 3 describes.

### 5.18 Admin accounts

| Field | Rule |
|---|---|
| `email` | Company identity-provider account; no personal accounts |
| `displayName` | Shown in audit entries |
| `roles` | Per environment |
| `state` | `active`, `disabled` |
| `lastLogin` | Server |

Created, changed and disabled by `super_admin` only. Granting `super_admin` needs a second `super_admin` (section 8). An account unused for `AdminInactiveDisableDays` is disabled. The role list is reviewed at every go/no-go (`docs/launch-runbook.md` GN-20).

The table is `admin_accounts` (`server/internal/adminaccount`). Each environment has its own database (`docs/environments.md` E-1), so an account row and its roles exist per environment. The email is stored trimmed and lower-case and is unique; sign-in is refused for a disabled or unknown email and records `lastLogin` only on success. An account never signed in counts as inactive from its creation.

## 6. Player tools

### 6.1 Lookup

- Search by account id, nickname (prefix) or email (exact match only), `support` and `super_admin` (#144). Search by auth identity is `super_admin` only (D-2).
- The player record shows profile, nickname history, devices (latest per install), country, inventory, ledger, purchases and refunds, pass and mission progress, rank state per mode, matches, offenses, sanctions, risk signals by category, reports, social graph and consent records, each limited to the rows of 2.1. Nothing is editable in place.
- Replay view of a finished match (D-13) for disputes and collusion review.
- Opening a player record, a replay or a ledger is a view entry in the audit log.

### 6.2 Actions

Every action goes through the same server path a game event would use, so it obeys the economy, ranking and penalty rules.

| Action | Effect | Two-person (section 8) |
|---|---|---|
| Grant | Applies a reward bundle through the bundle engine (#102) under a source that every cosmetic and medal of the bundle already lists (`docs/cosmetics.md` 5), such as the event or campaign whose reward the player missed; ledger reason `admin_adjust` for coins | Above `AdminGrantApprovalCoins` coins or with a `mythic` item |
| Adjust coins | `admin_adjust` entry, positive or negative; never makes the balance negative; a positive entry may clear a refund debt (`docs/economy.md` sections 5, 9) | Same threshold on the absolute amount |
| Temporary ban | No login for `TempBanHours` (`docs/anti-cheat.md` section 3) | No |
| Shadowban, lift shadowban | `docs/anti-cheat.md` section 3 | No |
| Permanent ban, lift permanent ban | From the review queue only (`docs/anti-cheat.md` section 3) | Yes |
| Remove offense, lift penalty | `docs/penalties.md` section 7 | No |
| Revert rating changes | Reverses MMR and LP of named matches after confirmed collusion (`docs/anti-cheat.md` V-9) | No |
| Revoke emblem | `docs/ranking.md` 8.3; leaves the showcase at once | No |
| Force rename | `docs/localization.md` 5.6 | No |
| Edit targeted promo list | 5.11 | No |
| Mute | For `ModerationMuteHours`, the player cannot send in-match emoji or quick messages (#151, #152); play is unaffected (owner decision O-4) | - |

- Every action needs a reason. Players see the effect through server notices (`docs/localization.md` section 3), never the reason text.
- Support cannot refund money (stores only), cannot delete an account (the player does, `docs/privacy.md` 8.2) and cannot export a player's data (the player does, 8.3).

### 6.3 Test accounts

Staff test accounts are flagged at creation by `super_admin`, hold no player data, and are what `content` and `live_ops` use to try content in staging. Grants to test accounts follow the same rules and audit.

### 6.4 Queues

| Queue | Fed by | Decisions |
|---|---|---|
| Review | Risk score at `ReviewScore`, hard signals, penalty step 4 (`docs/anti-cheat.md` section 3, `docs/penalties.md` section 4) | Clear, shadowban, temporary ban, propose permanent ban, revert rating changes |
| Reports | Player reports (`docs/localization.md` 5.8) | Approve the name, force a rename, add a filter entry |
| Appeals | Player appeals (#186, #209) | Uphold or reverse with a reason; the reversal undoes the sanction |

Each entry shows its age, the evidence the support role may see and the account's earlier decisions.

### 6.5 Rights requests

Records of objections and other requests handled through support (`docs/privacy.md` 8.1): account, type, received, deadline (30 days), state, outcome. The list shows open requests by age (L-21).

## 7. Audit log

| Field | Content |
|---|---|
| `id` | Unique, increasing |
| `time` | Server UTC |
| `actor` | Admin account and the roles it used |
| `environment` | `staging` or `production` |
| `action` | `create`, `edit`, `submit`, `approve`, `reject`, `publish`, `schedule`, `cancel`, `rollback`, `retire`, `delete`, `view`, a section 4 control or a section 6.2 action |
| `object` | Type and id; for player actions the account id |
| `before`, `after` | Field values of the affected record; secrets (promo plaintext, tokens) never |
| `reason` | Text; mandatory for every `S` action of 2.3, every section 4 and 6.2 action and every rollback |
| `approval` | The second admin's entry id where section 8 applies |
| `ref` | Optional incident, ticket or issue id |
| `changeSet`, `configVersion` | Where relevant |
| `source` | Admin session id, source IP and user agent of the admin |

- Append-only: the admin API's database role can insert and read audit rows, never update or delete them. Entries expire after `AdminAuditRetentionDays` (`docs/privacy.md` D-27).
- The table is `admin_audit_entries` (`server/internal/audit`). A trigger rejects every `update`, `delete` and `truncate` for every role, the owner included. The retention purge is not built yet; it has to run through a reviewed path that removes only expired rows, not by dropping the trigger.
- Entries about a player are visible to `super_admin` only (D-27). Other roles see entries about objects they can view. An entry is about a player when its object is a player, a review, report or appeal queue item, or a rights request record (`players`, `queues`, `rights_requests` in `server/internal/rbac`); the admin API filters them in the query, so a search never returns rows the role may not see.
- Searchable by actor, object, action, time range and reason text; exportable by `super_admin`.

## 8. Two-person rule

Owner decision (2026-10-26, O-1): the rule covers the high-risk actions below; ordinary production publishes and rollbacks are made by one author and rely on validation, the audit log and rollback. A high-risk action is first proposed, then confirmed by a second admin holding the same permission. Self-approval is impossible; a proposal expires after `AdminApprovalTtlHours`.

| Action | Proposer | Approver |
|---|---|---|
| Grant or adjust above `AdminGrantApprovalCoins`, or with a `mythic` item | `support` | Another `support` or `super_admin` |
| Promo code above `PromoCoinsApprovalThreshold` or with a `mythic` item | `live_ops` | Another `live_ops` or `super_admin` (`docs/economy.md` section 8) |
| Permanent ban and its lift | `support` | Another `support` or `super_admin` |
| Section 4 control in production, except promo code disable | `live_ops` or `super_admin` | Another `live_ops` or `super_admin` |
| Granting `super_admin` | `super_admin` | Another `super_admin` |

Proposals are stored in the `admin_approvals` table (`server/internal/approval`, migration `00012_admin_approvals`).
- **Proposing.** A proposal records the kind of action and its object. The object type comes from the kind: players, promo codes, live controls or admin accounts. It also records the change as JSON, a mandatory reason, the proposer and their roles, and an expiry of `AdminApprovalTtlHours` after the proposal.
- **Deciding.** A proposal can be approved or rejected only by an admin who holds an approver role from the table above and is not the proposer. The database checks this as well.
- **Expiry and finality.** A proposal can be decided only while it is pending and before it expires. A decision is final. Of several parallel decisions, exactly one wins.
- **Pending list.** The pending list holds only proposals that are still undecided and not expired.
- **Carrying out.** Running the approved action and writing its audit entries is up to each action's handler. The admin API does this for granting `super_admin` and for coin adjustments (9.1). The other kinds follow with their tools: bundle grants in #144; permanent bans from the review queue are carried out (`docs/anti-cheat.md` 5.3, #209), promo codes in #142. Live controls are carried out for the controls of `docs/anti-cheat.md` 5.2 (#210), maintenance mode included (#146). A live control approval is refused with `stale_proposal` when the controls changed after it was proposed, so an old proposal cannot undo a newer switch; reject it and propose again. Break-glass is not built yet.

Break-glass: during a declared SEV1 or SEV2 incident (`docs/launch-runbook.md` 6.1) one `live_ops` or `super_admin` may switch a section 4 control alone, citing the incident id. A second admin confirms afterwards; unconfirmed break-glass actions stay listed on the dashboard. The 5-minute targets of `docs/launch-runbook.md` GN-17 and 5.4 include the approval.

## 9. Security

| Item | Rule |
|---|---|
| Login | SSO through the company identity provider (Google Workspace, #134). No local passwords and no shared accounts |
| Second factor | Enforced by the identity provider for every admin; phishing-resistant keys (security key or passkey) for `super_admin` (O-5) |
| Network | The panel is reachable only through an identity-aware access proxy in front of SSO with 2FA; no public endpoint, no VPN (owner decision O-2) |
| Session | Server-side session, cookie `HttpOnly`, `Secure`, `SameSite=Strict`. Idle timeout `AdminSessionIdleMinutes`, absolute limit `AdminSessionMaxHours` |
| Step-up | Every `S` action and every approval needs a sign-in no older than `AdminStepUpMinutes` |
| Admin API | Separate host and credentials, never player tokens (ADR 0006). The browser talks only to the admin web server; the admin API authorises every request from the admin's identity and roles |
| Data access | The admin API's database role writes config, audit and admin tables and calls the ledger and sanction services; it has no direct write on ledger, inventory or rank tables |
| Web hardening | Cross-site request protection (`SameSite=Strict` session cookie, every change through a server action that checks `Origin`, no change on `GET`), strict CSP with a per-request nonce, rate limits per account, server-side pagination (#148, `admin/README.md`) |
| Secrets | From the environment only (ADR 0006). Promo plaintext is shown once and never logged |
| Compromise | Disable affected admin accounts, rotate secrets, review the audit log since the first suspect entry (`docs/launch-runbook.md` 6.5) |

The keys of section 11 are deployment config, changed by pull request, not in the panel: an admin who could edit them could weaken the controls that limit admins.

### 9.1 Admin API

`server/cmd/admin-api` is the admin API (`server/internal/adminapi`), a separate binary on its own host. It reads the environment of `docs/environments.md` 2.2. Every route answers JSON with `Cache-Control: no-store`, and every route except `/admin/nonce` and `/admin/session` takes the session token as `Authorization: Bearer`.

| Route | Who | Does |
|---|---|---|
| `POST /admin/nonce` | anyone | A single-use nonce for the Google sign-in |
| `POST /admin/session` `{"id_token","nonce"}` | anyone | Sign-in: answers `{"token","session","account"}` |
| `POST /admin/session/step-up` `{"id_token","nonce"}` | signed in | A new Google sign-in of the same account renews the step-up time |
| `DELETE /admin/session` | signed in | Sign-out: revokes the session |
| `GET /admin/me` | signed in | The account, the session and `permissions`: every object with the actions the roles grant (2.3), for the navigation |
| `GET /admin/audit` | V on the audit log | Search: `actor`, `object_type`, `object_id`, `action`, `from`, `to` (RFC 3339), `reason`, `before_id`, `limit`, newest first (section 7) |
| `GET /admin/accounts` | V on admin accounts | Every account |
| `POST /admin/accounts` `{"email","display_name","roles"}` | C on admin accounts | Creates an account |
| `PUT /admin/accounts/{id}/roles` `{"roles"}` | E on admin accounts; adding `super_admin` also S: grant `super_admin` | Replaces its roles, or proposes the grant (202 with the proposal) |
| `POST /admin/accounts/{id}/disable` | R on admin accounts | Disables it |
| `GET /admin/config/daily-login` | V on missions | The published calendar, its `config_version` (0 and `null` before the first publish) and `rules.default` |
| `PUT /admin/config/daily-login` `{"catalog","reason"}` | P on missions | Publishes the next `daily_login` snapshot through `dailylogin.Parse`: 7 or 30 days, each with whole coins from 1 and no other field |
| `GET /admin/config/daily-login/history` | V on missions | The last 50 `daily_login` snapshots, newest first |
| `POST /admin/config/daily-login/rollback` `{"version","reason"}` | P on missions and S: rollback on snapshots | Publishes the content of an earlier calendar as the next version |
| `GET /admin/config/rules` | V on config values | Every key of the `rules` domain with type, unit, bounds, default and current value, and the `config_version` (0 before the first publish) |
| `PUT /admin/config/rules` `{"values"}` | P on config values | Publishes the next `rules` snapshot. A key left out keeps its default |
| `GET /admin/config/rules/history` | V on config values | The last 50 `rules` snapshots, newest first |
| `POST /admin/config/rules/rollback` `{"version"}` | P on config values and S: rollback on snapshots | Publishes the content of an earlier `rules` snapshot as the next version |
| `GET`, `PUT /admin/config/matchmaking`, `GET /admin/config/matchmaking/history`, `POST /admin/config/matchmaking/rollback` | V or P on config values; rollback also S | The `matchmaking` domain (`docs/matchmaking.md` section 3) as a whole JSON object; keys left out keep their default; an unknown key or a value out of range gets 400 `invalid` |
| `GET`, `PUT /admin/config/penalties`, `GET /admin/config/penalties/history`, `POST /admin/config/penalties/rollback` | V or P on config values; rollback also S | The `penalties` domain (`docs/penalties.md` section 8) as a whole JSON object; keys left out keep their default; an unknown key or a broken ladder gets 400 `invalid` |
| `GET /admin/config/shop` | V on shop offers | The published shop catalog (offers and rotations of 5.5), its `config_version`, and the server's `rotation` rules (`daily_slots`, `weekly_slots`, `reset_hour_utc`); `catalog` is null before the first publish |
| `PUT /admin/config/shop` `{"catalog"}` | P on shop offers | Publishes the catalog as the next `shop` snapshot. A chest pool item must be a published `common` or `rare` item listing the chest as its `chest` source (V-15) |
| `GET /admin/cosmetics/items` | V on cosmetic items | The published cosmetic items (`id`, `type`, `rarity`, `availability`, `sources`), the built-in ones before the first publish; the chest pool editor picks from them |
| `GET /admin/config/shop/history` | V on shop offers | The last 50 `shop` snapshots, newest first |
| `POST /admin/config/shop/rollback` `{"version"}` | P on shop offers and S: rollback on snapshots | Publishes the content of an earlier `shop` snapshot as the next version |
| `GET /admin/config/cosmetics` | V on cosmetic items | The published manifest (5.1), its `config_version`, and `rules`: `types`, `rarities`, `availability`, `source_types`, `art_sources` and the `builtin` manifest; `catalog` is null before the first publish |
| `PUT /admin/config/cosmetics` `{"catalog"}` | P on cosmetic items | Publishes the items as the next manifest (Cosmetics manifest note) |
| `GET /admin/config/cosmetics/history` | V on cosmetic items | The last 50 manifests, newest first |
| `POST /admin/config/cosmetics/rollback` `{"version"}` | P on cosmetic items and S: rollback on snapshots | Republishes the content of an earlier manifest under new revisions (`docs/cosmetics.md` 9) |
| `GET /admin/config/missions` | V on missions | The published mission catalog (5.8), its `config_version`, and `rules`: the mission `types`, progress `events`, `queues` and `modes` the condition builder offers; `catalog` is null before the first publish |
| `PUT /admin/config/missions` `{"catalog"}` | P on missions | Publishes the catalog as the next `missions` snapshot, with the server's `nameKey` and `revision` (Missions and medals note) |
| `GET /admin/config/missions/history` | V on missions | The last 50 `missions` snapshots, newest first |
| `POST /admin/config/missions/rollback` `{"version"}` | P on missions and S: rollback on snapshots | Publishes the content of an earlier `missions` snapshot as the next version |
| `GET /admin/config/medals` | V on medals | The published medal catalog (5.9), its `config_version`, and `rules`: `rarities` in order, `source_types` and `availability` values |
| `PUT /admin/config/medals` `{"catalog"}` | P on medals | Publishes the catalog as the next `medals` snapshot, with the server's keys, `staticAddress` and `revision` |
| `GET /admin/config/medals/history` | V on medals | The last 50 `medals` snapshots, newest first |
| `POST /admin/config/medals/rollback` `{"version"}` | P on medals and S: rollback on snapshots | Publishes the content of an earlier `medals` snapshot as the next version |
| `GET /admin/config/rating` | V on ranked seasons | The published ranking config (5.12, `docs/ranking.md` section 10), its `config_version`, and `rules`: `server_time`, `open_season` (the season started and not closed in both modes, or empty), `structural_keys`, `tiers`, `division_ranks` and the server `defaults`; `catalog` is null before the first publish |
| `PUT /admin/config/rating` `{"catalog"}` | P on ranked seasons | Publishes the config as the next `rating` snapshot (Ranking config note) |
| `GET /admin/config/rating/history` | V on ranked seasons | The last 50 `rating` snapshots, newest first |
| `POST /admin/config/rating/rollback` `{"version"}` | P on ranked seasons and S: rollback on snapshots | Publishes the content of an earlier `rating` snapshot as the next version, under the same checks |
| `GET /admin/ranked/seasons` | V on ranked seasons | `server_time`, `next_start` (the end of the last season, or null) and every season in start order with its `state` (`scheduled`, `live`, `ending`, `closed`) |
| `GET /admin/config/quick-messages`, `PUT` `{"catalog","reason"}`, `GET .../history`, `POST .../rollback` | V, P on game strings | The `quick_messages` catalog `{"keys"}`: published only when every key is `quick.<slug>`, listed once and approved in every game language (`docs/localization.md` 5.7, #152) |
| `GET /admin/calendar` | V on live events | `server_time` and the running and planned entries (`kind`, `id`, `start`, `end` or null when open-ended) in start order: `ranked_season`, `pass_season`, `shop_offer` (featured and event sections) and `promo_campaign` (earliest start and latest end of its codes that are not disabled; only with V on promo codes) (5.13) |
| `POST /admin/ranked/seasons` `{"id", "start", "end", "no_rewards", "reason"}` | C on ranked seasons | Schedules a season (Ranked seasons note); 201 with the season |
| `PATCH /admin/ranked/seasons/{id}` `{"start", "end", "no_rewards", "reason"}` | E on ranked seasons | Reschedules a season that has not started |
| `POST /admin/ranked/seasons/{id}/extend` `{"end", "reason"}` | E on ranked seasons | Moves the end of the live season later |
| `DELETE /admin/ranked/seasons/{id}` `{"reason"}` | D on ranked seasons | Removes the last season if it has not started; 204 |
| `GET /admin/config/pass` | V on pass seasons | The published pass config (5.7), its `config_version`, and `rules.server_time` so the editor knows which season runs; `catalog` is null before the first publish |
| `PUT /admin/config/pass` `{"catalog"}` | P on pass seasons | Publishes the config as the next `pass` snapshot (Pass seasons note) |
| `GET /admin/config/pass/history` | V on pass seasons | The last 50 `pass` snapshots, newest first |
| `POST /admin/config/pass/rollback` `{"version"}` | P on pass seasons and S: rollback on snapshots | Publishes the content of an earlier `pass` snapshot as the next version |
| `GET /admin/players?q=` | V on players | With `provider` and `subject` instead of `q`, the account of that auth identity, for `super_admin` only (D-2). Otherwise search (6.1): an account id, an email (exact, any case) or a nickname prefix (folded as for uniqueness, `docs/localization.md` 5.2); at most 20, by nickname |
| `GET /admin/players/{id}` | V on players | The player record of 6.1 that exists so far: account (state, country, emails; auth identities for `super_admin` only, D-2), profile, nickname history, inventory with revoked items, balance, and the newest 50 ledger entries, reward grants, purchases and matches |
| `POST /admin/players/{id}/coins` `{"amount","request_id"}` | S on players: adjust coins | An `admin_adjust` ledger entry, positive or negative (6.2); above `AdminGrantApprovalCoins` it answers 202 with a `player_grant` proposal instead (section 8) |
| `POST /admin/players/{id}/temporary-ban` | S on players: temporary ban | Bans sign-in for `TempBanHours` (`HD_TEMP_BAN_HOURS`) from now and revokes every refresh session of the account (6.2) |
| `DELETE /admin/players/{id}/temporary-ban` | S on players: lift a ban | Ends a running temporary ban |
| `POST /admin/players/{id}/mute` | S on players: mute | Mutes the player for `ModerationMuteHours` (`HD_MUTE_HOURS`) from now (6.2) |
| `DELETE /admin/players/{id}/mute` | S on players: lift a mute | Ends a running mute |
| `POST /admin/players/{id}/force-rename` | S on players: force rename | Requires the player to pick a new nickname (`docs/localization.md` 5.6) |
| `GET /admin/reward-bundles` | V on reward bundles | Every bundle in published config with `guideValue`, `usedBy` and `differs`; `promo_truncated` when only the newest `MaxPromoListed` codes were read (5.4) |
| `PUT /admin/reward-bundles/{id}` `{"revision","coins","crystals","cosmetics","medals","passXp","reason"}` | P on reward bundles and on each domain holding it | Publishes the new content to every holder in one transaction; `editable` on the list says which bundles allow it. `stale_bundle` when `revision` is not the current one, `bundle_not_editable` for a differing id or another holder (5.4) |
| `POST /admin/players/{id}/grant` `{"bundle","source","request_id"}` | S on players: grant | Applies the bundle through the reward engine (6.2); above `AdminGrantApprovalCoins` coins or with a `mythic` item it answers 202 with a `player_grant` proposal instead (section 8) |
| `GET /admin/matches/{id}/replay` | V on players | The match (mode, kind, window, scores, winner, config version), its seats with account, nickname, bot and result, and the replay record (D-13) played back hand by hand: the tiles dealt to each seat, every intent with its tile, end, points or drawn tile, and each hand's outcome; `timeline` is null when no record was stored |
| `GET /admin/promo-codes?campaign=` | V on promo codes | Usage per campaign (`campaigns`: codes, active, pending, disabled, cap, redemptions, earliest start, latest end) and the newest 200 codes (`codes`), of one campaign when `campaign` is given; never a code's plaintext |
| `POST /admin/promo-codes` `{"id","campaign","kind","bundle","start","end","cap","code","accounts"}` | C on promo codes | Creates one code (5.11) and answers 201 with the code and its `plaintext`, shown this once; a code that needs a second admin also carries `approval_id` |
| `POST /admin/promo-codes/batch` `{...,"count"}` | C on promo codes | Creates `count` (1 to 10,000) `single_use` codes `<id>_<n>` and answers 201 with a CSV of `promo_id,code`, never stored; headers `Hd-Promo-State` and, when a second admin must approve, `Hd-Approval-Id` |
| `GET /admin/promo-codes/{id}` | V on promo codes | The code; for `support` (S: targeted account list) also `personal`: the listed accounts and the redemptions (`account_id`, `redeemed_at`) |
| `PUT /admin/promo-codes/{id}/accounts` `{"accounts"}` | S on promo codes: targeted account list | Replaces the account list of a `targeted` code; its cap becomes the number listed |
| `POST /admin/promo-codes/{id}/disable` | S on promo codes: disable | Stops the code redeeming at once; granted rewards stay (section 4) |
| `POST /admin/promo-batches/{id}/disable` | S on promo codes: disable | Disables every code of the batch |
| `GET /admin/promo-templates` | V on promo codes | The promo templates (`docs/launch-content-gate.md` 3.6), by id; each is a `draft` with `template` true and holds only `id`, `campaign`, `kind` and an optional `bundle` |
| `POST /admin/promo-templates` `{"id","campaign","kind","bundle"}` | C on promo codes | Saves a new template and answers 201. It has no window, cap, code or accounts, never redeems and is not listed with the codes |
| `PUT /admin/promo-templates/{id}` `{"campaign","kind","bundle"}` | E on promo codes | Replaces the template's fields; its creator stays |
| `GET /admin/approvals` | signed in | Pending proposals (section 8) the admin proposed or may decide |
| `POST /admin/approvals/{id}/approve` | an approver role of section 8 | Approves and carries out the proposal |
| `POST /admin/approvals/{id}/reject` | an approver role of section 8 | Rejects it |
| `GET /admin/strings` | V on game strings | Every key with its values per language, and the missing-value report per language (`coverage`: keys, approved, percent, missing keys) |
| `PUT /admin/strings/{key}` `{"note","max_length","placeholders","deprecated"}` | C for a new key, E for an existing one, R to deprecate it | Saves the key |
| `PUT /admin/strings/{key}/values/{language}` `{"text","status"}` | E for `draft`, E and P for `approved` | Saves one value |
| `GET /admin/strings/export?language=&format=` | V on game strings | The translator file of `docs/localization.md` 2.1, `csv` or `json` |
| `POST /admin/strings/import?language=&format=` | E on game strings | Imports a translator file of at most 4 MiB |
| `GET /admin/strings/active` | V on game strings | The active bundle version per language and the fallback chains, with their `config_version` (0 before the first publish) |
| `PUT /admin/strings/active` `{"bundles","fallback_chains"}` | P on game strings; a rollback also S: rollback on snapshots | Publishes the next `localization` snapshot (`docs/localization.md` 2.2) |
| `GET /admin/strings/bundles` | V on game strings | Every published bundle version, newest first per language |
| `POST /admin/strings/bundles/{language}` | P on game strings | Publishes the language as its next bundle version |

- **Sign-in.** The Google ID token must have a trusted audience and the nonce, a verified email, the company Workspace as hosted domain (`hd`) and an email in that domain. The email must belong to an active account (5.18). Unknown and disabled accounts get the same answer, `account_refused`.
- **Session.** The token is random and only its SHA-256 is stored (`admin_sessions`, migration `00013_admin_sessions`). A request is refused with `session_rejected` when the token is unknown or revoked, when the last request was `AdminSessionIdleMinutes` ago or more, when the session is `AdminSessionMaxHours` old, or when the account has been disabled since. Each answer gives `idle_until`, `expires_at` and `step_up_until`, which is the last sign-in plus `AdminStepUpMinutes`.
- **Rate limits (#148).** `POST /admin/nonce` and `POST /admin/session` share one bucket per source IP (`AdminRateSignInPerMinute`, burst `AdminRateSignInBurst`), the same address the audit entry records. Every signed-in route takes one bucket per account after the session check (`AdminRateAccountPerMinute`, burst `AdminRateAccountBurst`), so an unknown token gets `session_rejected` and spends nothing. A refused request gets 429 `rate_limited` with `Retry-After` in whole seconds. The buckets live in Redis when `HD_REDIS_URL` is set, so every instance counts together, and in memory otherwise. When Redis fails the request passes and a warning is logged, as on the game server.
- **Roles.** Every route checks the matrix of 2.3 with the roles the account has at that moment, so a role change counts from the next request. A missing action gets 403 `forbidden`.
- **Audit.** A change and its audit entry are written in one transaction, so an entry exists exactly when the change does. The entry carries the actor with their roles, the environment of `HD_ADMIN_ENVIRONMENT`, the session id, the user agent and the source IP. Before and after hold `email`, `display_name`, `roles` and `state`. A write may carry `reason` and `ref`. The source IP is the last address of the last `X-Forwarded-For` header, else the connection address. The admin web server therefore sets that header to the browser address it sees.
- **Game strings.** Approving a value is the review step of 5.15, so it needs P besides E: `content` and `super_admin` approve, `live_ops` saves drafts. The audit object is the key: `create`, `edit` and `retire` hold the key fields, and a value change is `edit`, or `approve` for an approved value, with language, text and status. An import writes one `edit` entry per changed key with reason `import`. A publish is `publish` on `bundle:<language>` with the version and hash. A blocked import or publish answers 422 `blocked` with `problems` (`key`, `error`) and stores nothing.
- **Active versions.** The request is checked like `cmd/config-publish`: every named version is a published bundle and the `en` bundle has every live key. A new snapshot is published under a lock on the domain, so its audit `before` is the snapshot it replaces. It is a `rollback` when a language gets a lower version than the active one or loses its bundle, and a `publish` otherwise. A rollback is an `S` action: it needs rollback on snapshots (`super_admin`, or `content` for this domain, which it owns), a reason and a sign-in no older than `AdminStepUpMinutes`. The audit object is `snapshots` `localization`, with both snapshots and the new `configVersion`.
- **Rules config (#145).** Values are checked against the bounds of `docs/rules.md` section 13 before publishing, so an unknown key, a value that is not an integer or one out of bounds gets 400 `invalid` and nothing is stored. A publish runs under a lock on the domain, in one transaction with its `publish` entry on `game_config` `rules`, which has both snapshots and the new `configVersion`. A rollback is an `S` action: it needs rollback on snapshots, which `live_ops` and `super_admin` have for this domain because they publish it, plus a reason and a fresh sign-in. It writes a `rollback` entry, and an earlier snapshot that is outside today's bounds is refused. Game servers pick up a new version within `HD_CONFIG_POLL`, and it applies to matches that open afterwards.
- **Missions and medals (#141).** Both catalogs are checked by the same `mission.Parse` and `medal.Parse` as the game servers, so an invalid one gets 400 `invalid` with the reason and nothing is stored. The editor never types server fields:
  - the server sets each mission's `nameKey` and each medal's `nameKey`, `descriptionKey` and `staticAddress` from the id;
  - it sets `revision` to 1 for a new entry, keeps it for an unchanged one, and adds 1 to a changed one (5.8, 5.9).

  A publish keeps every published id (ids are never reused): a mission is retired and a medal retired or disabled instead, and one left out gets 400 `invalid`. A rollback restores the earlier content; an entry added since stays in the catalog, retired.

  Every mission bundle's medals and cosmetics must list that mission as a source (`docs/economy.md` section 4). A mission publish checks this against the published medals and cosmetics. A medal publish checks it against the published missions, so a medal cannot lose a source a live mission grants it through. The order for a new medal is therefore: publish the medal listing the mission, then the mission.

  The panel edits missions on `/missions` and medals on `/medals` (`admin/README.md`).

  A publish or rollback writes a `publish` or `rollback` entry on `missions` `missions` or `medals` `medals` with both snapshots, under a lock on the domain. The body may be up to 1 MiB. Game servers pick up the version within `HD_CONFIG_POLL`; rotation stays automatic per player and slot (5.8).
- **Cosmetics manifest (#136).** The panel is `/cosmetics` (A-5, A-6). The client sends the items; the server sets `manifestVersion`, each `revision`, `nameKey` and the addresses that follow from the id, the type and `animated` (`docs/cosmetics.md` 8.1), so a character's `clips` and models and a table's layer addresses are never typed. A revision rises once against the published item, only if its content changed. The first publish follows the built-in manifest. The manifest is checked by `cosmetics.ValidateManifest` (V-1 to V-11, V-14, V-19, strict: an undeclared field fails) and against the previous one (V-12, V-13), and each failure names its rule in `detail` with 400 `invalid`. A publish or rollback that drops a source a published mission, pass level, rank reward or chest pool grants from is refused the same way, so a grant never loses its listing (V-15, the record-to-item direction). A publish or rollback writes a `publish` or `rollback` entry on `cosmetic_items` `cosmetics` with both manifests.
- **Ranking config (#143).** The config is checked by the same `rating.Parse` as the game servers, so an invalid one gets 400 `invalid` with the reason. While a ranked season is open (started, and not yet closed in both modes, so also while its end runs), a change to a structural key of `docs/ranking.md` 7.1 in either mode gets 409 `frozen` with the mode and keys in `detail`: `divisionLp`, `demotionLp`, `apexRecalcHourUtc`, the apex slots and minimum LP, `placementMatches`, and the reset table (`resetTierDrop`, `resetFloor`, `resetCeiling` and the effective `seasonResetTable`). MMR and LP coefficients and the other keys may change at any time. Every medal and cosmetic of a `rankRewards` bundle must list the source `rank_reward` with that bundle's id in the published medals and cosmetics, and a medal publish keeps the sources the published ranking config needs. A publish or rollback writes a `publish` or `rollback` entry on `ranked_seasons` `rating` with both snapshots, under a lock on the domain. Creating and moving seasons is a separate step (#143). The panel is `/ranked` (A-19).
- **Ranked seasons (#143).** Seasons are rows of `ranked_seasons` (`docs/ranking.md` 7.1) and stay contiguous: a new season gets an id `season.<slug>`, the name key `<id>.name`, a start in the future equal to `next_start`, and an end after it; any other start gets 400 `invalid` with the reason. A season that has not started may change its end, its `no_rewards` flag and, if no season ends at its start, its start; once started it gets 409 `season_started`. Only the live season may be extended (409 `season_not_live` otherwise), and only later. Moving an end shifts every following season by the same amount, so the next season always starts at the end of the previous one. Only the last season can be removed, and only before it starts. Each change runs in one transaction under a lock on the season and writes a `create_season`, `reschedule_season`, `extend_season` or `delete_season` entry on `ranked_seasons` with the season id and the season before and after; a reason is required.
- **Pass seasons (#140).** The config is checked by the same `pass.Parse` as the game servers, so an invalid one gets 400 `invalid` with the reason and nothing is stored: windows never overlap, every premium level 1 to `levels` has a bundle, track bundles grant only their track's currency and cosmetics, and each track stays within `freeTrackCoins` or `premiumTrackCrystals`. Every track cosmetic must list the source `pass` with ref `pass.<seasonId>.<track>.level_<n>` in the published cosmetics, and none may be `mythic` (`docs/pass-missions-medals.md` 2.4). A season that has started stays in the config; a planned one may be dropped until its start. While a season runs on server time, a change to its curve or to any of its track bundles gets 409 `frozen`; catalog-wide values such as match XP may change at any time. A rollback restores the earlier content and keeps a started season the target version lacks. A publish or rollback writes a `publish` or `rollback` entry on `pass_seasons` `pass` with both snapshots, under a lock on the domain. Replacing a `disabled` cosmetic in a running track is not offered yet; it is refused as a frozen change. The panel edits the config on `/pass` (`admin/README.md`).
- **Shop catalog (#139).** The catalog is checked by the same `shop.Parse` as the game servers, with the rotation slots and reset hour of `HD_SHOP_DAILY_SLOTS`, `HD_SHOP_WEEKLY_SLOTS` and `HD_SHOP_RESET_HOUR_UTC`, so an invalid catalog gets 400 `invalid` and nothing is stored. A publish keeps every offer of the current catalog (AP-5): an offer is ended by its window, and one left out gets 400 `invalid`. The body may be up to 1 MiB; a larger one gets 413 `too_large`. A publish or rollback writes a `publish` or `rollback` entry on `shop_offers` `shop` with both snapshots and the new `configVersion`, under a lock on the domain. A rollback is an `S` action like the rules rollback, and it restores the earlier catalog exactly. Scheduling needs no extra step: offer windows and rotation starts are part of the catalog, and game servers open and close them on server time once they pick up the version within `HD_CONFIG_POLL`.
- **Player lookup (#144).** Opening a record writes a `view` entry on `players` with the account id in the same transaction as the read (AP-6), so the entry exists exactly when the record was shown. A search writes no entry. An unknown or malformed id gets 404 `not_found`, an empty search 400 `invalid`. Devices, pass and mission progress, rank state, offenses, sanctions, risk signals, reports, social graph and consent records join the record when their server data exists.
- **Coin adjustment (#144).** It is an `S` action: a reason and a sign-in no older than `AdminStepUpMinutes`. The entry goes through the wallet like any other (`docs/economy.md` section 5): source `admin` with the admin's id, and the idempotency key `admin:<admin id>:<request_id>`, so a repeated request answers the first result with `"repeated":true` and writes nothing, and a request id reused for another amount gets 400 `invalid`. A debit that would take the balance below zero gets 409 `insufficient`; a credit may leave a refund debt partly unpaid. An amount whose absolute value is above `AdminGrantApprovalCoins` (`HD_ADMIN_GRANT_APPROVAL_COINS`) is stored as a `player_grant` proposal with a `submit` entry, and only a `support` admin may propose it. Approving it applies the entry in the same transaction and writes an `approve` entry and an `adjust_coins` entry whose `approval` is the `approve` entry; if the wallet refuses it, nothing changes and the proposal stays pending. Each applied adjustment writes an `adjust_coins` entry on `players` with the balance before and after and the ledger entry id.
- **Temporary ban (#144).** Ban and lift are `S` actions: a reason and a fresh sign-in. A ban sets `banned_until` on the account to now plus `TempBanHours`, renewing a running ban from now, and revokes every refresh session (revoke reason `banned`) in the same transaction, with a `temporary_ban` entry holding the earlier and new end and the number of sessions revoked. Sign-in answers 403 `banned` until then. An access token issued before the ban keeps working until it expires (`HD_ACCESS_TTL`) and a match already running is not interrupted. `TempBanHours` has no value yet (`docs/anti-cheat.md` section 7 leaves it to the panel before launch), so while `HD_TEMP_BAN_HOURS` is unset a ban answers 409 `not_configured`. Lifting a ban that is not running answers 409 `not_banned`; a lift writes a `lift_temporary_ban` entry.
- **Mute (#144).** Mute and lift are `S` actions: a reason and a fresh sign-in. A mute sets `muted_until` on the account to now plus `ModerationMuteHours`, renewing a running mute from now, with a `mute` entry holding the earlier and new end. A room drops every emoji from a seat whose player is muted; play, sign-in and sessions are untouched (O-4). The seat roster carries the flag from `muted_until` when a match starts (the roster is filled once matchmaking is wired), so a mute set during a match takes effect from the next one. Quick messages do not exist yet; they follow the same flag when added. `ModerationMuteHours` comes from `HD_MUTE_HOURS` (default 24) until panel-edited game config carries it. Lifting a mute that is not running answers 409 `not_muted`; a lift writes an `unmute` entry.
- **Force rename (#144).** An `S` action: a reason and a fresh sign-in. It sets `rename_required` on the profile and writes a `force_rename` entry with the nickname; an account already flagged answers the same view and writes nothing, and an account without a nickname gets 409 `no_profile`. The player sees `rename_required` in `GET /profile/me`, may rename at once whatever the cooldown, and cannot keep the reported name; that rename clears the flag and does not start the cooldown. Keeping a flagged player out of ranked matches and custom rooms is checked where those entries are served, which the game server does not expose yet.
- **Grant (#144).** An `S` action: a reason and a fresh sign-in. The bundle follows the bundle rules (`docs/economy.md` 4) and `source` is `{"type","ref"}` of a type of `docs/cosmetics.md` 5 other than `default`. Every cosmetic and medal of the bundle must list that source, so a grant never adds a way to obtain an item; otherwise it gets 400 `invalid` and nothing is stored. The engine applies it in one transaction under the key `admin-grant:<admin id>:<request_id>`: coins and crystals as `admin_adjust` ledger entries with source `admin:<admin id>` like a coin adjustment, cosmetics and medals with the chosen source, a duplicate cosmetic as its compensation (`docs/economy.md` 2.7), and pass XP to the active season, or dropped when none runs. A repeated request answers the first grant with `"repeated":true`; a request id reused for another bundle gets 400 `invalid`. Above `AdminGrantApprovalCoins` coins or with a `mythic` item it is a `player_grant` proposal that only `support` may submit, like a coin adjustment; approving it applies the grant and writes `approve` and `grant_reward` entries. An applied grant writes a `grant_reward` entry on `players` with the engine's report (coins, cosmetics, duplicates, medals, pass XP).
- **Replay view (#144).** Opening a replay writes a `view_replay` entry on `players` with the match id in the same transaction as the read, like a player record. The stored record is decoded and run through the rules engine (`server/internal/replay`), so the view shows what the server decided, including the hidden hands and the boneyard draws needed for collusion review (`docs/anti-cheat.md` V-9). A record that does not decode, an intent the engine refuses, or a played-back result that differs from the stored one is shown with `problem` and the hands up to that point, never hidden. An unknown or malformed match id gets 404 `not_found`.
- **Promo codes (#142).** The admin API runs the same `promo.Store` checks as `cmd/promo` (`docs/economy.md` section 8), so a bad id, campaign, window, cap, code or bundle gets 400 `invalid` and an id or code already used 409 `duplicate`. Creating needs a reason; the creator is the admin. A `create_promo` or `create_promo_batch` entry holds the code fields and, for a batch, the count, never a plaintext. A code or batch above `PromoCoinsApprovalThreshold` or with a `mythic` item is stored as `pending_approval` with a `promo_code` proposal (section 8) that only `live_ops` may submit. One proposal covers a whole batch. Approving it activates the code or every pending code of the batch in the same transaction and writes `approve` and `activate_promo` entries. Rejecting it disables them with a `discard_promo` entry, so a rejected code never redeems. Disabling is an `S` action without a second admin: a reason and a fresh sign-in, a `disable_promo` entry with the number of codes, and nothing when they are already disabled. 10,000 codes take about 0.3 s against a local database (`TestTenThousandCodesComeOutInUnderTenSeconds`). Without `HD_PROMO_KEY` every promo route answers 409 `not_configured`.
- **Targeted account list (#142).** Only `support` lists accounts (2.1: `live_ops` holds no personal data), so `POST /admin/promo-codes` refuses `accounts` with 400 `invalid`. A `targeted` code starts with cap 0. Setting the list is an `S` action: a reason and a fresh sign-in, a body of up to 256 KiB, unique existing account ids, and every account that already redeemed stays listed. A disabled code gets 409 `disabled`. It writes a `set_promo_targets` entry on the code with the counts before and after, and one `add_promo_target` or `remove_promo_target` entry per account on `players`, so the ids reach `super_admin` only (section 7). The per-account redemptions are served to `support` only.
- **Promo templates (#142).** A template is kept in `promo_templates`, apart from the codes, so it never redeems and never shows in usage or the code list. Saving one needs a reason and runs the code checks on id, campaign, kind and bundle; the bundle may be left out, as for the compensation template (`docs/launch-content-gate.md` 3.6). A bad field gets 400 `invalid`, an id already used 409 `duplicate`, an unknown id on edit 404 `not_found`. It writes a `create_promo_template` or `edit_promo_template` entry on `promo_codes` with the template before and after. Cloning sends the template's fields with a window, cap or code to `POST /admin/promo-codes`, so the clone follows every code rule.
- **Account guards.** An admin cannot change or disable their own account (`own_account`). A new account never starts as `super_admin` (`grant_by_approval`): it is created with other roles, and the role is granted afterwards.
- **Granting `super_admin`.**
  - **Proposal.** Adding the role is an `S` action. It needs a reason and a sign-in no older than `AdminStepUpMinutes`. The roles do not change yet: the change is stored as a `super_admin_grant` proposal that expires after `AdminApprovalTtlHours`, and a `submit` entry records it.
  - **Approval.** It needs another `super_admin` with a fresh sign-in. In one transaction it sets the proposed roles and writes two entries: an `approve` entry by the approver, and an `edit` entry with the before and after roles, the proposal's reason and `approval` set to the `approve` entry's id.
  - **Reject.** It writes a `reject` entry.
  - **Errors.** Self-approval gets `self_decision`, a decided proposal `decided` and an expired one `expired`.
- **First `super_admin`.** `admin-api -bootstrap-email <company email> -bootstrap-name <name>` creates it and exits, only while `admin_accounts` is empty. It locks the table, so two runs cannot both succeed. Its audit entry has actor and session `bootstrap`.
- **Errors.** 400 `malformed`, `invalid` (with a `detail` text), `reason_required`; 401 `token_rejected`, `token_expired`, `nonce_rejected`, `session_rejected`; 403 `not_company_account`, `account_refused`, `different_account`, `forbidden`, `step_up_required`, `self_decision`; 404 `not_found`; 409 `email_taken`, `grant_by_approval`, `own_account`, `decided`, `expired`, `not_supported`, `insufficient`, `not_configured`, `not_banned`, `no_profile`, `duplicate`, `not_pending`, `disabled`; 413 `too_large`; 422 `blocked`; 429 `rate_limited`; 503 `provider_unavailable`.

## 10. Screens and navigation

The panel UI is localized itself (#213) in `az` and `en`, default `az` (owner decision 2026-10-31). Every text comes from a message key in `admin/src/i18n/messages`; `en` is the reference, and a key missing from or extra in another language fails `pnpm typecheck` in CI. The chosen language is kept on the admin account (`admin_accounts.locale`, set by `PUT /admin/me/locale` `{"locale"}`, 400 `unknown_locale` for another value) and wins over the browser cookie once signed in; an account that never chose one uses the cookie, then `az`. Navigation shows only groups and screens the account's roles allow.

| Group | Screen | Roles | Section |
|---|---|---|---|
| Home | A-1 Dashboard: active `configVersion`, scheduled change sets, pending approvals, unconfirmed break-glass actions, live control states, economy monitors (`docs/economy.md` section 10) | All | 3, 4, 8 |
| Publishing | A-2 Change sets: list, editor, diff, validation report | `super_admin`, `content`, `live_ops` | 3.2 to 3.4 |
| | A-3 Approvals | Accounts with P | 3.5, 8 |
| | A-4 Snapshots and rollback | All but `analyst` | 3.7, 5.17 |
| | A-41 Content gate report: run, history, failures per check, JSON export | All; run: `super_admin`, `content`, `live_ops` | `docs/launch-content-gate.md` section 4 |
| Catalog | A-5 Cosmetics list with filters and bulk edit; A-6 item form per type with preview | `super_admin`, `content`; others view | 5.1 |
| | A-7 Assets and catalog builds | `super_admin`, `content` | 5.2 |
| | A-8 Rarity table | `super_admin`, `content`; others view | 5.3 |
| | A-9 Medals; A-10 Emblems | `super_admin`, `content`; others view | 5.9, 5.10 |
| Economy | A-11 Reward bundles with usage | `super_admin`, `live_ops`; others view | 5.4 |
| | A-12 Shop offers; A-13 Rotations | `super_admin`, `live_ops`; others view | 5.5 |
| | A-14 Store products | `super_admin`, `live_ops`; `support`, `analyst` view | 5.6 |
| | A-15 Promo codes and campaigns | `super_admin`, `live_ops`; `support` targeted lists | 5.11 |
| Progression | A-16 Pass seasons; A-17 Missions; A-18 Daily login | `super_admin`, `live_ops`; others view | 5.7, 5.8 |
| Ranked | A-19 Seasons and reset table; A-20 Leaderboards (public data) | `super_admin`, `live_ops`; others view | 5.12 |
| Live ops | A-21 Calendar | All; promo campaigns only with view on promo codes | 5.13 |
| | A-22 Events; A-23 Banners; A-24 Push campaigns | `super_admin`, `live_ops`; `content` texts | 5.13 |
| | A-25 Live controls | `super_admin`, `live_ops`; others view | 4 |
| | A-26 Config values and flags | Per domain, 2.3 | 5.14 |
| Localization | A-27 Game strings and missing report; A-28 Bundles per language | `super_admin`, `content`; `live_ops` edits drafts | 5.15 |
| Players | A-29 Search; A-30 Player record with tabs per 6.1 | `super_admin`, `support` | 6.1, 6.2 |
| | A-31 Review queue; A-32 Reports; A-33 Appeals | `super_admin`, `support` | 6.4 |
| | A-34 Moderation lists | `super_admin`, `support` | 5.16 |
| | A-35 Rights requests | `super_admin`, `support` | 6.5 |
| Analytics | A-36 KPIs, cohorts, economy sink and source, CSV export of aggregates | All; event-level data `analyst`, `super_admin` | #147 |
| Admin | A-37 Admin accounts and roles; A-38 Audit log; A-39 Retention and privacy config; A-40 Anti-cheat config | `super_admin`; A-38 per section 7 | 5.18, 7, 5.14 |

Every list screen paginates on the server, filters by state and searches by id and name. Every object screen links to its history in the audit log and to the change sets that touched it.

## 11. Config

Deployment config of the admin API (section 9), changed by pull request. Values are launch defaults (O-6).

| Key | Proposed default | Section |
|---|---|---|
| `AdminSessionIdleMinutes` | 30 | 9 |
| `AdminSessionMaxHours` | 12 | 9 |
| `AdminStepUpMinutes` | 10 | 9 |
| `AdminApprovalTtlHours` | 24 | 8 |
| `AdminGrantApprovalCoins` | 2000, the value of `PromoCoinsApprovalThreshold` | 6.2, 8 |
| `AdminInactiveDisableDays` | 30 | 5.18 |
| `AdminRateAccountPerMinute` | 120 | 9.1 |
| `AdminRateAccountBurst` | 40 | 9.1 |
| `AdminRateSignInPerMinute` | 10 | 9.1 |
| `AdminRateSignInBurst` | 10 | 9.1 |

Game config defined here and edited in the panel: `ModerationMuteHours` (default 24, section 6).

Used here and defined elsewhere: `PromoCoinsApprovalThreshold`, `PromoCodeLength`, `PromoMaxValidityDays`, `ShopDailyRotationSlots`, `ShopWeeklyRotationSlots`, `ReferencePriceDays`, `DailyLoginCoins`, `CrystalPackCrystals`, `RankRewardCoins`, `PassFreeTrackCoins`, `PassPremiumTrackCrystals` (`docs/economy.md`), `PassLevels`, `PassXpBase`, `PassXpStep`, `PassXpMax`, `EventMissionXpBudget`, `DailyMissionSlotCoins`, `DailyMissionSlotXp`, `WeeklyMissionSlotCoins`, `WeeklyMissionSlotXp` (`docs/pass-missions-medals.md`), `TableMaxLayers` (`docs/cosmetics.md`), `EmblemBrackets`, `SeasonResetTable`, `LeaderboardRegions`, `LeaderboardTopSize` (`docs/ranking.md`), `ReviewScore`, `TempBanHours` (`docs/anti-cheat.md`), `AdminAuditRetentionDays` (`docs/privacy.md`), `ReservedNames`, `LocaleFallbackChains` (`docs/localization.md`), `ContentGateMin*` (`docs/launch-content-gate.md`).

## 12. Implementation

| Issue | Sections |
|---|---|
| #134 Skeleton, login, RBAC, audit log | 2, 7, 9, 5.18 |
| #135 Asset upload and publish pipeline | 5.2, 3.4 |
| #136 Cosmetics catalog CRUD engine | 5.1, 3.3 |
| #137 Forms per cosmetic type | 5.1 |
| #138 Reward bundles | 5.4 |
| #139 Shop offers and rotations | 5.5, 5.6 |
| #140 Pass seasons | 5.7 |
| #141 Missions and medals | 5.8, 5.9 |
| #142 Promo codes | 5.11 |
| #143 Ranked seasons and tiers | 5.12, 5.10 |
| #144 Player and support tools | 6 |
| #145 Remote config and feature flags | 3.1, 5.14 |
| #146 Live-ops calendar and announcements | 4, 5.13 |
| #147 Analytics dashboard | 10 (A-1, A-36) |
| #148 Performance and security hardening | 9 |
| #213 Admin UI localization | 10 |
| #214 Game string management | 5.15, 3.7 |

The content gate report (A-41) has no sub-issue yet (`docs/launch-content-gate.md` section 6).

## 13. Decisions

Owner decisions of 2026-10-26 (O-1 to O-4); O-5 and O-6 taken as proposed by the lead.

| Id | Question | Decision |
|---|---|---|
| O-1 | Two-person rule scope (section 8) | High-risk actions only; ordinary production publishes need one author. Break-glass for incidents |
| O-2 | Network restriction for the panel | Identity-aware access proxy in front of SSO with 2FA; no public endpoint |
| O-3 | Variant tests in the shop (#139) | Presentation only; price, content, window and eligibility are the same for everyone (`docs/economy.md` EP-8) |
| O-4 | Meaning of "mute" in #144 | Blocks in-match emoji and quick messages for `ModerationMuteHours`; play is unaffected |
| O-5 | Second factor for `super_admin` | Phishing-resistant keys only |
| O-6 | Section 11 defaults | As listed |
| O-7 | Bounds of the `rules` domain keys (5.14, #145) | `docs/rules.md` section 13; set by the lead with the owner's permission on 2026-10-31 |
