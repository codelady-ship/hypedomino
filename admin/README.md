# admin

Next.js admin panel (TypeScript, Tailwind only, no component library: `docs/admin.md` O-11) for content, economy, live-ops and player support. Every designer-tunable value is managed here.

The panel holds no permission logic of its own: the admin API decides every permission (`docs/admin.md` AP-2, matrix in `server/internal/rbac`).

## Run

```
cd admin
pnpm install
pnpm dev
pnpm typecheck
pnpm build
```

Node 24 and pnpm 11 (`packageManager` in `package.json`).

## Sign-in and navigation

The home page shows the Google sign-in button (Google Identity Services, popup) when there is no session. The button gets a single-use nonce from `POST /admin/nonce`; the ID token and that nonce go to `POST /admin/session` from a server action, and the returned token is stored in the `hd_admin_session` cookie (`HttpOnly`, `SameSite=Strict`, `Secure` in production, expiring with the session, `docs/admin.md` section 9). The browser never holds the token in script. Sign-out revokes the session with `DELETE /admin/session` and drops the cookie. A refused sign-in shows the reason (`account_refused`, `not_company_account`, an expired token or nonce) and prepares a new nonce.

Every page reads `GET /admin/me` once per request (`currentAdmin` in `src/admin/session.ts`). The header lists only the screens of `src/admin/navigation.ts` whose object the account may view, and each screen checks the same rule before it calls the API, which still refuses on its own (AP-7). A session the API rejects shows the sign-in again. `src/admin/pages.test.ts` reads every `page.tsx` under `src/app`: each page is a navigation screen, a detail page whose object the test lists, or the home page, and each one shows `auth.required` without a session and `auth.forbidden` without view on its object before its first admin API call. A new page fails the test until it is classified and guarded.

An `S` action needs a sign-in no older than `AdminStepUpMinutes` (`docs/admin.md` 9). When the API answers `step_up_required`, the screen shows `StepUpPrompt`: the same Google button, whose ID token and a new nonce go to `POST /admin/session/step-up` from a server action. The session keeps its token, and a sign-in with another account is refused (`different_account`). Once it succeeds, the action is sent again with what the admin entered. `GoogleButton` holds the shared Google Identity Services code for both. Calls to the admin API forward the browser's `X-Forwarded-For` and `User-Agent`, so the audit source is the admin's browser, not the web server.

## Dashboard

Signed in, `/` is the A-1 dashboard (`docs/admin.md` section 10). It shows the account with its roles by their localized names and, in parallel and each only when the roles allow it: unconfirmed break-glass actions and the live control states (V on live controls, `GET /admin/live-controls/break-glass` and `GET /admin/live-controls`), the number of pending approvals with a link to `/approvals` (every admin), the active config version with its domain and publish time (`GET /admin/config/version`) and the economy monitors with their alerts and a link to `/analytics` (V on dashboards). A section whose call fails says so without hiding the others. Scheduled change sets are not shown; they come with A-2.

## Audit log

`/audit` (A-38) searches `GET /admin/audit` by actor, object type and id, action, reason text and a time range, newest first, 50 per page; "older entries" continues below the last id shown (`before_id`). The form takes and shows times in UTC (AP-8): a `datetime-local` value is sent as that minute in UTC. Each entry shows the actor with the roles it used, environment, reason, reference, config version, source (address, user agent, session) and its before and after values. Which entries a role sees is decided by the API (`docs/admin.md` section 7); a malformed filter shows a notice instead of results.

## Admin accounts

`/accounts` (A-37, `docs/admin.md` 5.18) is in the nav for roles that may view admin accounts, which is `super_admin` only. It lists `GET /admin/accounts` with display name, email, roles by their localized names, state and last sign-in. With C on admin accounts a form creates an account from a company email, a display name and roles; `super_admin` is not offered there, because the API grants it only to an existing account. On every other active account, E edits the roles and R disables it; both take a reason and an optional ticket or incident id, and disabling needs the reason. Adding `super_admin` needs S: grant `super_admin`, a reason and a fresh sign-in, and becomes a proposal another `super_admin` decides in `/approvals`, which links it back here. An admin's own row shows no actions; the API refuses changes to one's own account as well.

## Cosmetics

`/cosmetics` (A-5 and A-6, #136, `docs/admin.md` 5.1) edits the manifest of `GET /admin/config/cosmetics` without a code change. With no published manifest it starts from the built-in one.
- **Catalog:** every item with its rarity, availability, animation, sources and revision. It can be searched by id or source record and filtered by type, rarity, availability and source type.
- **Bulk edit:** rarity and availability over the checked items. The header box checks every listed item.
- **New item:** a type and a slug become `<type>.<slug>`; a taken id is refused. The id and the type never change, and a published item cannot be removed: retire or disable it.
- **Item form:** rarity, availability, art source, animation, the cultural review and the sources. Type fields follow the type and the animation flag: an emoji group, a table's ordered layers. The emoji group field suggests the groups the other emoji already use and notes when a typed group is used by no other emoji, so a misspelled group is seen before it becomes a new one. Groups stay free text; there is no fixed list (`docs/admin.md` O-12). Turning animation off drops the fields a static item must not have.
- **Preview:** for roles that can view assets, the item form shows the newest `static` upload of that item that passed its checks (`GET /admin/assets?item=<id>&exact=true&slot=static&status=passed&limit=1`), which is what players see with reduce motion on and at the low effect level, with its size and upload time and a link to its uploads on `/assets`. With no passed static it says so, because a publish that adds the item is refused until one exists. Without asset storage the preview is hidden. The animated render, the effect levels and the rarity diamond and halo of A-6 are not shown: the `anim` and `layer/<name>` uploads and a client render are not built (#135).
- **Server fields:** `revision`, `nameKey` and every address are set by the API on publish, so they are shown and never typed.
- **Cultural review:** only roles with the review action can set it (`cultural_review`). An item without it blocks the publish.
- **Checks:** a missing source, a malformed record id, a repeated source, a default item that is not live and static, a default action, a missing emoji group, bad table layers, or a type without a live default item marks the item and blocks the publish. Everything else, such as dropping a source another domain grants from, is refused by the API with its reason.
- **Publish** (`PUT /admin/config/cosmetics`) takes a reason and a confirmation step that lists added, changed and newly disabled items. It is shown only to roles with publish on cosmetic items.
- **History and rollback:** on `/admin/config/cosmetics/history` and `/admin/config/cosmetics/rollback`. A rollback brings content back under new revisions. Each version lists what it changed against the version before, like the missions history: a schema version change, and each item it added or each field of an item it changed by path, as `rarityId common → rare`. The `revision` the API bumps is left out; the oldest listed version is compared with the built-in manifest.

## Reward bundles

`/rewards` (#138, `docs/admin.md` 5.4) reads `GET /admin/reward-bundles`: every bundle the published config grants, with its contents, guide value and uses. The API searches ids, items, medals and objects, filters by the domain using a bundle or by ids holding differing content, and returns 50 bundles a page with a "Next page" link (#148). Bundles live inside the objects that grant them, so an edit (`PUT /admin/reward-bundles/{id}`) writes the new content to every holder and publishes each affected domain in one transaction. The editor lists those objects and needs a confirmation and a reason. A domain's own rules still apply, so one refusal, such as a running pass season, stops the whole edit. A bundle held by rank rewards, daily login or promo codes, or an id holding differing content, is not editable here. **New bundle** (C on reward bundles) creates a bundle with new content (`POST /admin/reward-bundles`): an id `bundle.<slug>` that is not used yet, coins, crystals, pass XP, published item and medal ids and a reason. It is listed as "Created here" and a shop offer takes it by id through **New offer** on `/shop`.

## Shop offers

`/shop` (A-17, `docs/admin.md` 5.5) edits the published catalog of `GET /admin/config/shop`. With no published catalog the editor starts with no offers and no rotations; offers are added with **New offer** and the first publish creates the catalog.
- **Offers:** for the selected offer, its price in the currencies its kind and section allow, and its window. The window opens and optionally closes, shown and entered in UTC (AP-8). A window that closes before it opens blocks the publish, and so does a `mythic` item in an offer that is not time-limited: only a rotation offer, or a featured or event offer whose window closes, may sell one (`docs/economy.md` 3.1). A struck-through price that is not the lowest price of its offer in the published versions of the last `reference_price_days` (from `GET /admin/config/shop`) is flagged under the price and blocks the publish too, unless the price and reference are unchanged from the current version (`docs/economy.md` 6.3); the panel reads the versions of `GET /admin/config/shop/history` and the API checks every version. A store product price also takes an optional note about the offer; switching to coins or crystals drops it. Country prices are not noted here: they live on the Store prices page. A coin or crystal price also takes the struck-through price in the same currency; it must be above the price, and an empty field drops it.
- **Purchase terms:** for the selected offer, the purchases per account and the eligibility: only accounts younger than a number of hours, only an account that has bought nothing yet, or both (`docs/economy.md` 6.2). An empty field means no limit or no condition. The values are admin config; the panel flags what the API refuses and blocks the publish: a cosmetic offer not limited to one purchase per account, a chest with a limit or eligibility, and a struck-through price not above the price.
- **A/B variants:** for the selected offer, variants with an id (`[a-z0-9_]+`), a name key and/or a position in its section (`docs/admin.md` 5.5, O-14). A variant changes nothing else; players are split evenly between the offer as written and its variants by a stable hash of their account id. A variant id that is invalid or listed twice, a position that is not a whole number from 1 and a variant that changes neither field block the publish; variant name keys are listed with the other unnamed keys above the publish button.
- **New offer:** adds an offer to the draft from its id, kind, section, reward bundle and name key. The id is `offer.<slug>` and not yet in the catalog; the section list holds only the sections that take the kind (`docs/admin.md` 5.5). The bundle is copied from an existing reward bundle (`GET /admin/reward-bundles`) whose content is the same wherever it is used; only a `pass` or `pass_levels` offer may have none. The name key needs an approved `en` value (`POST /admin/strings/unapproved`, an exact match on the key), as the API refuses the publish otherwise. The added offer is selected with no price and no window: the panel flags both, a bundle content its kind does not allow, and a cosmetic whose sources do not list the offer as a `shop` source, and blocks the publish until they are fixed. An offer that is not in the published catalog can be removed from the draft, which also takes it out of the draft rotations. A published offer stays in the catalog (the API refuses its removal); its window ends it.
- **Catalog-wide:** crystal pack amounts, and chest pools and odds.
- **Rotations:** plan the offers of a daily or weekly period ahead. The slot counts and reset hour come from the server (`rotation` in `GET /admin/config/shop`). A daily period starts at the reset hour, a weekly one on a Monday at the reset hour. Candidates are the offers of the period's rotation section; checked offers take the slots in the order they were checked. A rotation with no offers or more offers than slots blocks the publish.
- **Preview:** shows the offer ids each section of the draft would show at a chosen UTC moment, or now (`POST /admin/config/shop/preview`). For an offer with A/B variants a variant can be chosen; the preview then shows the offer at the variant's position with the variant id and its name key. Nothing is published.
- **Publish** (`PUT /admin/config/shop`) takes a reason. The API checks the catalog like the game servers, and they open and close offers on server time once they load the version. A publish or rollback is refused while any offer `nameKey` has no approved `en` value on `/strings`.
- **History:** the last 50 versions, each with its offer count and the offers it added or changed against the version before.
- **Rollback:** roles with publish on shop offers and rollback on snapshots can roll back to an earlier version. It needs a reason and a fresh sign-in (`POST /admin/config/shop/rollback`), using the same `Rollback` component as match rules.

## Store prices

`/store-prices` (A-14, `docs/admin.md` 5.6, `docs/economy.md` 7.4) edits the `store_prices` table of `GET /admin/config/store-prices`: the USD reference price of each real-money product, tier percents, the tier of each store country (searchable by code or by its name in the panel language), and exact overrides per product and country. Before the first publish the page starts from the server's default.
- **Preview:** the target of every product in every priced country, override in bold, computed in the browser like `storeprice.Target` (`admin/src/storeprices/prices.ts`). Countries without a tier or override keep the store's own price and are not shown.
- **Publish** (`PUT /admin/config/store-prices`) takes a reason; the API refuses `RU`, an unknown tier, product or country code, a percent outside 1 to 100 and an override above its reference.
- **CSV for the consoles:** after a publish, one file per store (Google Play Console, App Store Connect) with `product_id,country_code,target_usd` rows of the published version's server-computed targets.
- **History and rollback** like the other config pages, under publish on store products and rollback on snapshots.

## Pass seasons

`/pass` (#140, `docs/admin.md` 5.7) edits the pass config of `GET /admin/config/pass` without a code change.
- **Settings:** track budgets, premium plus levels, match XP, the daily XP cap, the bot seat share and level prices apply to every season and may change at any time. An empty field keeps the server default.
- **New season:** clone a season or start an empty one, with a start and end in UTC. The start and end default to the first ranked season (`GET /admin/ranked/seasons`) that has not ended and that no pass season overlaps (`docs/pass-missions-medals.md` 2.1); the form starts blank for a role without view rights on ranked seasons, or when no such ranked season exists. The new season takes the next `season_<n>`. A clone copies the curve and every coin and crystal reward under new bundle ids `bundle.pass_<season>_<track>_<level>`; cosmetics are left out, because each item must first list the new level as its `pass` source. An empty season gets an empty premium bundle per level.
- **Curve:** levels, XP base, step and max, with the XP to the next level and the running total per level as the server computes them. Changing the levels adds or trims premium levels and trims free levels above the last.
- **Level rewards:** per level, free coins and cosmetic ids, premium crystals and cosmetic ids, with the track totals against the budgets. An emptied free level is removed; a premium level must grant something. Editing a published bundle raises its revision once.
- **Running seasons:** the state (planned, running, ended) comes from server time (`rules.server_time`). Once a season has started its start, curve, coins and crystals are read-only; only the end of a running season and its cosmetic ids may change, and the API accepts only the replacement of a `disabled` item with a `live` one of the same type and rarity (`docs/admin.md` 5.7). An ended season is read-only. A planned season may be removed until it starts.
- **Checks:** a window that ends before it starts, an overlap, curve values below 1, XP max below XP base, levels not above premium plus levels, an empty premium level, a track over its budget or an invalid setting marks the field and blocks the publish. Item sources and mythic items are checked by the API.
- **Publish** takes a reason and a confirmation step that lists the added or changed seasons with their totals, dropped seasons and changed settings (`PUT /admin/config/pass`). It is shown only to roles with publish on pass seasons.
- **History and rollback:** as for shop offers, on `/admin/config/pass/history` and `/admin/config/pass/rollback`. Each version lists what it changed against the version before, like the missions history: each setting as `before → after`, each season it added or removed, and each field of a season it changed by path, such as `curve.levels` or `premium.4.crystals`.

## Ranked seasons

`/ranked` (A-19, #143, `docs/admin.md` 5.12) schedules seasons and edits the ranking config without a code change.
- **Seasons:** every season with its state (scheduled, live, ending, closed), window, reward flag and the progress of its season end per mode, all from `GET /admin/ranked/seasons`. A new season takes an id `season.<slug>` and starts at the end of the last one; it is refused until its name key `season.<slug>.name` has an approved `en` value on `/strings`, and the form shows that key with a link to its search there before the season is created; its end defaults to `seasonLengthMonths` of the 1v1 config. A scheduled season may change its end, its reward flag and, if no season ends at its start, its start. The live season may only end later. Moving an end names the seasons that move with it. Only the last season can be removed, before it starts. Each change needs a reason and is audited by the API.
- **Leaderboards and coins:** the board size, the regions with their countries and `rankRewardCoins` apply to both modes.
- **Per mode:** structural values and the reset table are read-only while a season is open (`rules.open_season`). MMR, LP and placement values may change at any time.
- **Reset table:** generated from the tier drop, floor and ceiling as the server computes it. A row can be edited on its own and is marked as edited; a row set back to the generated rank stops being an override.
- **Rank rewards:** per final tier, a bundle of cosmetics and medals, `bundle.rank_<tier>` by default. Editing a published bundle raises its revision once and an emptied row removes the bundle. Item sources are checked by the API.
- **Publish** takes a reason and a confirmation step that lists each changed value as `field before → after`, per mode by path (`1v1.lpBase`), and warns about frozen keys (`PUT /admin/config/rating`). The first publish starts from the server defaults (`rules.defaults`). The abuse values and emblem brackets are kept as published; this screen does not edit them.
- **History and rollback:** on `/admin/config/rating/history` and `/admin/config/rating/rollback`. Each version lists what it changed against the version before, like the missions history: each setting, region, rank coin amount and per-mode value as `field before → after` (`rankRewardCoins.Gold`, `leaderboardRegions.<id>`, `2v2.seasonResetTable.Gold II`, `1v1.rankRewards.Gold.cosmetics`). A key left out counts as its server default, and the oldest listed version is compared with the defaults.

## Missions

`/missions` (#141, `docs/admin.md` 5.8) edits the mission catalog of `GET /admin/config/missions` without a code change. With no published catalog the editor starts with no missions. The API fills slot rewards, budgets and rerolls from its defaults, so the first publish needs a live daily and weekly mission for every default slot.
- **New mission:** a slug (lowercase letters, digits, underscores) becomes `mission.<slug>`; a taken id is refused. The type sets the fields: daily and weekly take a slot and a weight, achievement and event take a reward bundle `bundle.mission_<slug>`, an achievement may chain to another achievement, and an event takes an event id.
- **Condition:** the counted progress event, the target, and optional queue and mode filters. Events, types, queues and modes come from the server (`rules` in the response), so the editor offers only what the game servers count.
- **Reward:** periodic missions pay their slot's reward from the catalog. An achievement or event bundle grants coins, cosmetic ids and medal ids, and an event bundle also pass XP. Editing a published bundle raises its revision once; restoring the published content restores its revision.
- **Server fields:** `nameKey` and the mission revision are shown, never typed; the API sets them on publish.
- **Checks:** a target, slot or weight below 1, a slot outside the type's slots, a bundle granting nothing or a malformed event id marks the mission and blocks the publish. Everything else, such as a granted medal or cosmetic that does not list the mission as a source, is refused by the API with its reason.
- **Publish** (`PUT /admin/config/missions`) takes a reason and is shown only to roles with publish on missions. A published mission is never removed; set it to retired. It is refused while any mission name key (`mission.<slug>.name`) has no approved `en` value on `/strings`.
- **History and rollback:** as for shop offers, on `/admin/config/missions/history` and `/admin/config/missions/rollback`. Each version lists what it changed against the version before: catalog settings such as the slot rewards, and each mission it added or each field of a mission it changed, as `field before → after` (`bundle.coins 0 → 50`). The oldest listed version shows its missions as added.

## Assets

`/assets` (#135, `docs/admin.md` 5.2) uploads the static PNG of a cosmetic item or a medal (`medal.<slug>`, stored at `medal/<slug>/static`; a medal publish, and a cosmetics publish that adds the item, are refused until its static has passed). The panel asks `POST /admin/assets/uploads` for a signed URL, the browser sends the file straight to the object store, and `POST /admin/assets/{id}/check` makes the server read it back and record format, alpha, width, height, bytes and SHA-256. A file that fails stays listed as rejected and is never shown; a new upload makes a new row. The list pages on the server, 50 rows at a time, and filters by status and item id prefix; the API also takes `exact=true` to match the item id exactly and `slot` to keep one slot. The file does not reach players yet.

When a cosmetics publish is refused with `cosmetic_art_missing`, or a medal publish or rollback with `medal_art_missing`, the refusal lists the items the API named, each linking to `/assets?item=<id>`. That link filters the list to the item and fills the upload form with its id. The panel does not check this before the publish; the API decides.

The browser needs the store in two places: `HD_ADMIN_ASSET_ORIGIN` adds its origin to `img-src` and `connect-src` of the Content Security Policy, and the bucket's CORS rules must allow `PUT` from the panel's origin. Without `HD_ASSET_STORE_URL` on the admin API, the page lists rows and closes uploads.

## Daily login

`/daily-login` (#141, `docs/admin.md` 5.8, `docs/economy.md` 2.3) edits the coins of each day of the calendar and publishes through `/admin/config/daily-login`, with a reason, history and rollback. Until a calendar is published the server has none and claims are refused, which the page says. The number of days is not edited here: a 30-day calendar is not proposed for launch.

## Medals

`/medals` (#141, `docs/admin.md` 5.9) edits the medal catalog of `GET /admin/config/medals`. With no published catalog the editor starts empty and the first publish creates it.
- **Catalog-wide:** the number of profile showcase slots, a whole number of 0 or more.
- **New medal:** a slug (lowercase letters, digits, underscores) becomes `medal.<slug>`; a taken id is refused. It starts with the server's first rarity and availability and one source to fill.
- **Fields:** rarity and availability (live, retired, disabled) come from the server (`rules` in the response). Sources are a list of a type (mission or rank reward) and an id; a medal needs at least one, every source needs an id, and a source listed twice blocks the publish.
- **Server fields:** `nameKey`, `descriptionKey`, `staticAddress` and the revision are shown, never typed; the API sets them on publish.
- **Order with missions:** a medal must list every published mission that grants it, and the API refuses a medal publish that drops one. Publish a new medal listing its mission first, then the mission that grants it.
- **Publish** (`PUT /admin/config/medals`) takes a reason and is shown only to roles with publish on medals. A published medal is never removed; retire or disable it. It is refused while any medal name or description key (`medal.<slug>.name`, `medal.<slug>.description`) has no approved `en` value on `/strings`; both are listed above the publish button.
- **History and rollback:** as for shop offers, on `/admin/config/medals/history` and `/admin/config/medals/rollback`. Each version lists what it changed against the version before, like the missions history: the showcase slots, and each medal it added or each field of a medal it changed.

## Game strings

`/strings` (#214, `docs/admin.md` 5.15, A-27 and A-28) edits game strings over `/admin/strings`.
- **Missing translations:** per language, approved values over live keys, the percentage and the number of missing keys, linked to the key list filtered to them.
- **Keys:** the API searches keys, notes and texts and filters to the keys a language is missing, 50 keys a page with a "Next page" link (#148); the coverage table counts missing keys through `GET /admin/strings/coverage` and links to that filter. Languages sit side by side. A new key, a note, the max length and the placeholders (`name:type`, comma separated) are saved with `PUT /admin/strings/{key}`; ticking deprecated retires the key.
- **Values:** each cell saves its text with status `draft`, or `approved` for roles with edit and publish on game strings.
- **Import and export:** a language as `csv` or `json` for an external translator. The panel sends a file through a server action, so it takes files up to 1 MB; larger ones go straight to `POST /admin/strings/import`. A blocked import lists each failing key.
- **Bundles:** one language at a time, its versions newest first, 50 a page (#148). Publish builds the next bundle of a language and uploads it to the CDN; a blocked publish lists each failing key, and `content_store_off` or `content_store_unavailable` means the admin API has no content store or the store refused the file, so nothing was published. Activating a newer bundle publishes the next `localization` snapshot with a reason. Activating an older one is a rollback and needs rollback on snapshots and a fresh sign-in.
- **Fallback chains:** each language's chain (`docs/localization.md` 1.2) as comma separated codes. Roles with publish on game strings save a chain with a reason; the panel refuses one that does not start with its language, end with `en` or name each game language at most once. Saving publishes the next `localization` snapshot through `PUT /admin/strings/active` with the active bundles unchanged, so it is refused until an `en` bundle is active.
- **Unnamed keys on other screens:** when a cosmetics, mission, medal, reward bundle, shop offer or season publish or rollback is refused with `invalid` because a name key, or a medal description key, has no approved `en` value, the refusal lists those keys from the API's `keys` field (the `detail` text is shown as is, never parsed), each linking to `/strings?q=<key>`. Above the publish button, `/missions`, `/medals` and `/shop` list the name keys (on `/medals` also the description keys) of the edited catalog that have no approved `en` value yet, with the same links, through `POST /admin/strings/unapproved` (needs view on game strings). A reward bundle edit lists the same for the name keys of the published missions and shop catalogs it republishes (read only when the editor can view them), and a new season lists its `<id>.name` above the create button. The list only warns; the API still decides on publish. Season reschedule, extend and removal take no name check.

## Players

`/players` (A-30 lookup, `docs/admin.md` 6.1) searches `GET /admin/players?q=` by account id, exact email or nickname prefix, optionally in one account state; a state alone lists those accounts, newest first. Results come 50 a page with a "Next page" link (#148). A `super_admin` also gets a second form that finds the account of one sign-in identity, a provider (`google` or `apple`) and its subject, through `GET /admin/players?provider=&subject=` (D-2). A search writes no audit entry. `/players/{id}` shows the record from `GET /admin/players/{id}`: account state, country, emails (sign-in identities only for `super_admin`), the temporary ban while it runs, the ranked ban while it runs with its reason category, balances, level, the forced-rename flag, nickname history, and the newest 50 ledger entries, inventory items, reward grants, purchases and matches. Its bans section lists the newest 50 sanctions of `account_sanctions`: start, kind, the reason category shown to the player, end and lift time. Its reports section lists the newest 50 player reports about or by the player, each with the side (about or by), the other account linked to its record, reason, reported name, match, state and decision time. The API writes a `view` entry for every opening (AP-6), so the links to a record do not prefetch it: a prefetch would log a view nobody made.

Roles with `act_on_player` (support and `super_admin`) get the support actions of 6.2 on a record that is not deleted. Each one needs a reason and a fresh sign-in, and `StepUpPrompt` covers the second.
- **Coins:** a whole, non-zero amount; a minus sign debits. The form keeps one `request_id` until the adjustment is applied, so a resend after step-up cannot apply it twice (`repeated`). An amount above `AdminGrantApprovalCoins` becomes a proposal for a second admin (202).
- **Ban reason:** a select of the six categories of 6.2 (`cheating`, `collusion`, `afk_leaving`, `offensive_name`, `payment_fraud`, `other`, labelled in `en` and `az`). The temporary and ranked ban buttons stay disabled until one is picked; the API refuses a ban without one with `invalid_category`.
- **Temporary ban:** ban or lift, whichever applies now. The ban is refused with `not_configured` while `HD_TEMP_BAN_HOURS` is unset.
- **Ranked ban:** ban or lift, whichever applies now, through `POST` and `DELETE /admin/players/{id}/ranked-ban`. It closes the ranked queue and custom rooms for `RankedBanHours`; the player keeps signing in and playing casual.
- **Grant:** a reward bundle (bundle id, coins, pass XP, cosmetic and medal ids) under a source picked by type and id, such as the event or campaign whose reward the player missed. Every cosmetic and medal must list that source, or the API refuses the grant and says which one. Like coins, the form keeps one `request_id` until the grant is applied. Above `AdminGrantApprovalCoins` coins or with a `mythic` item it becomes a proposal, which `/approvals` shows as bundle, grants and source.
- **Force rename:** offered while the player is not already flagged.
- **Remove from a leaderboard:** a started season (newest first, from `GET /admin/ranked/seasons`) and a mode, sent to `POST /admin/players/{id}/leaderboard-removal`. The player leaves every board of that season and mode for good (owner decision 2026-11-20); `not_on_board` means the player has no placed rating or result there and `already_removed` that it was done before. There is no whole-board reset.

**Replay `/matches/{id}`** (D-13, for support and `super_admin`) opens from the Replay link in a player's matches. It shows:
- the match and its seats, with links to the players;
- every hand: the tiles dealt to each seat in seat order, each move with tile, end and points, and the hand's outcome.

A record that does not decode, is refused by the engine or ends on another result shows the problem above the hands. Each opening is a `view_replay` audit entry.

The page reloads the record after each change.

## Approvals

`/live-controls` (A-25, `docs/admin.md` section 4, #210) is in the nav for roles that may view live controls. It edits the purchase and ranked kill switches, the minimum app version and the build allowlist of `docs/anti-cheat.md` 5.2. A change needs a reason and a fresh sign-in. Staging applies it at once; production waits in `/approvals` for a second admin. During a declared SEV1 or SEV2 incident, break-glass (`docs/admin.md` section 8) takes the severity and the incident id and switches production at once; the screen lists every unconfirmed break-glass action from `GET /admin/live-controls/break-glass` and links to `/approvals`, where another `live_ops` or `super_admin` confirms it. There the row shows the incident and the published version and never expires.

`/reviews` (A-31, `docs/admin.md` 6.4, #209) is in the nav for roles that may view queues. It lists the open reviews of `GET /admin/reviews` with their signals and evidence and the earlier decisions on the account, as on `/reports`, and links each account to its player record. The list pages on the server, 50 at a time, filters by cause and searches by account id prefix; `/reports` pages, filters by reason and searches the same way, and under each reported player shows the earlier decisions on the account from `GET /admin/reports` (queue, outcome, reasons or cause, reports closed, decision time) with the upheld and rejected report totals. Each review and each reported player shows its due time, the `due_at` the server computes from `ReviewSlaHours` or `ReportSlaHours` (`docs/admin.md` 6.4), and an entry the server marks `overdue` is shown in the danger color with an overdue label. Clear lifts a shadowban and confirm keeps it (`docs/anti-cheat.md` 5.3). Both need a reason and a fresh sign-in. `/appeals` (A-33) lists the open appeals of `GET /admin/appeals` the same way: 50 a page, filtered by sanction kind and searched by account id prefix, each with the player, the appealed sanction (kind, ban reason, start, end), the player's text, its due time from `AppealSlaHours` and the earlier decisions on the account (the shared `EarlierDecisions`, which now also lists decided appeals). Uphold keeps the sanction; lift ends it (`docs/admin.md` 6.4, Appeals); both need a reason and a fresh sign-in. Proposing a permanent ban also needs a ban reason category from the select above the buttons. An admin who wants a temporary ban uses the player record first, then confirms.

`/approvals` (two-person rule, `docs/admin.md` section 8) is in the nav for every signed-in admin. It lists what `GET /admin/approvals` returns: pending proposals the admin's roles may decide, and their own. A row shows the kind, the object (a player links to the record), the change without its bookkeeping ids, the reason, the proposer with their roles, and the expiry. An admin's own proposal says another admin decides. On the others, Approve and Reject take an optional note. Approve needs a fresh sign-in and uses `StepUpPrompt`, and the list reloads after a decision.
A `promo_code` proposal reads as its code count, campaign, bundle and window, and links to `/promo`.

## Promo codes

`/promo` (A-15, `docs/admin.md` 5.11, #142) is in the nav for roles that may view promo codes. It needs `HD_PROMO_KEY` on the admin API; without it the screen says promo codes are off.

- **Usage:** per campaign, the codes, how many are active, pending approval or disabled, redemptions against the summed cap, and the window. A campaign links to its codes.
- **Codes:** newest first, 50 a page with an "Older codes" link, of all campaigns or one, filtered by state and searched by code id prefix: type, state (an active code past its end shows `expired`), what it grants, window, redemptions and cap, creator and approver.
- **Create (`live_ops`, `super_admin`):** one code of any type, or a batch of up to 10,000 `single_use` codes. The bundle is a bundle id with coins, pass XP and cosmetic item ids. The window is entered in UTC. A `multi_use` code takes a cap and an optional chosen code. A `targeted` code starts without accounts. A reason is required.
  - **One code:** its plaintext is shown once.
  - **Batch:** downloads as `<id>.csv` from the browser. Nothing keeps the plaintext afterwards, beyond a "download again" button while the page stays open.
  - **Second admin:** a code above the coin threshold or with a `mythic` item says so and waits in `/approvals`.
- **Templates (`docs/launch-content-gate.md` 3.6):** every role that may view promo codes sees them: id, campaign, type, what they grant (or "set when cloned" without a bundle), creator and last change. In the create form, "Save as template" keeps the id, campaign, type and bundle with a reason; on an existing id it updates that template. "Start from a template" fills the form, and the code still needs its own window, cap or code. A template never redeems and is not listed with the codes.
- **Code page `/promo/{id}`:** the code's fields for every role that may view it. `support` also sees who redeemed and when, linked to the player. On a targeted code that is not disabled, `support` edits the account list: ids by line or comma, a reason and a fresh sign-in. The screen keeps the save off while an account that redeemed is missing (`docs/admin.md` 9.1).
- **Disable (`live_ops`):** a code, or the whole batch from its first listed code. It takes a reason and a fresh sign-in (`StepUpPrompt`) and is immediate.

## Match rules

`/config/rules` is the `rules` part of A-26 (`docs/admin.md` 5.14). It lists the keys of `GET /admin/config/rules` with their unit, allowed range, default and current value. Roles with publish on config values may edit them; a value outside the range or not a whole number blocks the publish before the API sees it, and the API checks the bounds again. A publish (`PUT /admin/config/rules`) sends only the values that differ from their defaults, with the reason, and the page then shows the new `config_version`. Below, the last 50 published versions show their non-default values and each key they changed against the version before, as `key before → after`; a key left out counts as its default, and the oldest listed version is compared with the defaults. A value applies to matches that open after the game servers pick it up (`HD_CONFIG_POLL`). Each earlier version has a rollback for roles with publish on config values and rollback on snapshots. It needs a reason and a fresh sign-in, and it publishes that version's values as the next version (`POST /admin/config/rules/rollback`).

## Live-ops config

`/config/liveops` is the `liveops` part of A-26 (`docs/admin.md` 5.14) and works exactly like `/config/rules`, on `/admin/config/liveops`. So far it holds the content gate minimums (`docs/launch-content-gate.md` section 5): whole numbers from 0 with no upper bound, shown as `≥ 0`; the analytics queue keys (`docs/analytics.md` section 6); the moderation queue SLA keys `ReviewSlaHours`, `ReportSlaHours` and `AppealSlaHours` (`docs/admin.md` 6.4), 1 to 720 hours; and the appeal keys `AppealTextMaxLength` (50 to 2000 characters) and `AppealTokenMinutes` (5 to 1440 minutes). The content gate minimums and the SLA keys stay in the admin API and are never sent to the client.

## Economy monitors

`/analytics` is the economy sink and source part of A-36 (#147, `docs/economy.md` section 10). Every role with view on dashboards sees it. It shows `GET /admin/economy/monitors` with the alerts raised on top: the 7-day window, coins created and spent, the sink ratio and the median balance, then the economy day with the accounts above the daily earn alert and the coins created per reason. A config version in the form adds the same monitors for the 7 days before and after its publish (`GET /admin/economy/monitors/compare`) as two more columns. Only aggregates are shown, never an account id. "Export CSV" downloads what the page shows as `economy-monitors-<day>.csv` (`-v<version>` added when a version is compared), one row per window and monitor with the columns `window,from,to,day,metric,reason,value`: `window` is `now`, `before` or `after`, `metric` is `coins_created`, `coins_spent`, `sink_ratio_percent` (empty when there is no ratio), `median_balance`, `accounts_over_daily_earn` or `source_mix` with the ledger reason in `reason`. The file is built from the same responses, with no extra request; alerts are not in it.

## KPIs and cohorts

`/analytics/kpis` is the KPI and cohort part of A-36 (#147, `docs/closed-beta.md` 4.1 BS-5 to BS-7). Every role with view on dashboards sees it. It shows `GET /admin/analytics/kpis` for the last 1 to 13 economy weeks (form field, default 4), up to yesterday: cards for the completed matches and the new analytics ids of the last economy day and the D1 and D7 retention of the newest cohort whose return day has ended, then the weekly cohorts, the completed matches per day and mode, and the daily cohorts. A share is the retained ids divided by the cohort size with one decimal, shown with the count; a return day that has not ended shows `pending`. Matches count all players; cohorts count analytics ids, so only players with `analytics` consent. Only aggregates are shown, never an id. "Export CSV" downloads the same report as `kpis-<from>-<to>.csv` (`to` is today, left out), with the columns `table,start,mode,metric,value`: `table` is `matches` (metric `completed` per day and mode), `daily_cohort` or `weekly_cohort` (metrics `size`, `d1_retained`, `d1_percent`, `d7_retained`, `d7_percent`, empty while pending). The file is built from the same response, with no extra request.

## Security headers

`src/proxy.ts` makes a fresh 128-bit nonce for every page request and sends a Content Security Policy (`src/security/headers.ts`): scripts only with that nonce and `'strict-dynamic'`, no `'unsafe-inline'`, `'unsafe-eval'` only under `next dev`; no framing (`frame-ancestors 'none'`), forms only to the panel, no `<base>`, no plugins. Google sign-in may load its button frame, its stylesheet and its calls under `https://accounts.google.com/gsi/` and nothing wider. Next.js puts the nonce on its own scripts; the sign-in button adds the Google script with the nonce of the loaded document, so it keeps working after a client-side navigation, when the page's nonce no longer matches the document's policy. `next.config.ts` adds `X-Content-Type-Options: nosniff`, `Referrer-Policy: no-referrer`, `X-Frame-Options: DENY`, `Cross-Origin-Opener-Policy: same-origin-allow-popups` (the sign-in popup) and a `Permissions-Policy` to every response, and HSTS in production.

Cross-site requests: the session cookie is `SameSite=Strict`, every change goes through a server action, which Next.js accepts only when the `Origin` header matches the host, and no `GET` changes anything. These two are the whole protection; the panel uses no CSRF tokens (`docs/admin.md` O-8). If the access proxy in front of the panel rewrites the host, its public host goes into `serverActions.allowedOrigins`.

The admin API limits sign-in per address and requests per account (`docs/admin.md` 9.1); a sign-in refused with `rate_limited` shows `signin.rate_limited`.

## Hardening check

The #148 acceptance is Lighthouse above 90 in performance, accessibility and best practices, and no critical vulnerability (`docs/test-strategy.md` 2.6). SEO is not part of the target, because every page is `noindex` on purpose (`docs/admin.md` O-9). To repeat it locally:

1. Start PostgreSQL. In `server/`, run `go run ./cmd/migrate up`, then `go run ./cmd/admin-api -bootstrap-email root@example.com -bootstrap-name Root`, then run the admin API.
2. Google sign-in needs a real client, so a local-only session stands in for it: insert an `admin_sessions` row for that account with `token_hash = sha256('<local token>')`, and send `Cookie: hd_admin_session=<local token>`. Never do this outside a throwaway database.
3. `pnpm build`, then `pnpm start` with `HD_ADMIN_API_URL` pointing at the local API.
4. Run `lighthouse <url> --preset=desktop` and the default mobile run on `/` signed out, and on `/`, `/audit` and `/shop` with `--extra-headers '{"Cookie":"hd_admin_session=<local token>"}'`.
5. Run `pnpm audit` here and `govulncheck ./cmd/admin-api/...` in `server/`.

CI repeats step 5 on every change to its area: the `admin` job fails on a critical advisory (`pnpm audit --audit-level=critical`) and the `server` job on any vulnerability `govulncheck` finds reachable from the admin API, which has no severity levels, so every reachable finding blocks the PR (`docs/test-strategy.md` 10.1, `docs/admin.md` O-10). Lighthouse needs a running panel and API, so it stays a manual run.

Last result: 2026-11-03, Lighthouse 13.6.0, headless Chrome on Windows, empty catalog and one audit entry. The scores are performance / accessibility / best practices.

| Page | Desktop | Mobile |
|---|---|---|
| `/` signed out (sign-in) | 100 / 100 / 100 | 93 / 100 / 100 |
| `/` signed in | 99 / 100 / 100 | 97 / 100 / 100 |
| `/audit` | 100 / 100 / 100 | 97 / 100 / 100 |
| `/shop` | 100 / 100 / 100 | 98 / 100 / 100 |

Mobile LCP is 0.7 to 0.8 s and CLS is 0. SEO scores 50 by design and is outside the target: every page is `noindex`, which is the failing audit. `pnpm audit` and `govulncheck` found no known vulnerabilities.

The same run probed the panel and the API:
- Every response carried the headers above.
- `/audit` without a session, or with a forged cookie, showed only the sign-in prompt.
- A server action posted with a foreign `Origin` was refused, and one with the panel's origin passed.
- The API answered 401 `session_rejected` without a bearer token, and the eleventh nonce from one address within a minute got 429 with `Retry-After`.

## Configuration

`HD_ADMIN_API_URL` and `HD_ADMIN_GOOGLE_CLIENT_ID` (`docs/environments.md` 2.2), read on the server only.

## Colours

`src/app/globals.css` maps the Tailwind theme to `docs/design-tokens.json` (`color.background`, `color.surface`, `color.text`, `color.role.primary`). Change the token file first, then the theme.

## Languages

The panel UI is in `az` (default) and `en` (`docs/admin.md` section 10). Texts live in `src/i18n/messages`, one file per language, with `en.ts` as the reference. Server components use `await serverTranslator()`, client components `useTranslator()`; both return the same `t(key)`. A missing or extra key in a language is a type error, so `pnpm typecheck` fails. The choice is kept in the `hd_admin_locale` cookie, and switching re-renders the page without a reload.
