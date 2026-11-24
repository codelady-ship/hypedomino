# Localization and content policy

Languages, text delivery, text rules, nickname policy and the cultural content checklist. Rule and penalty meaning stays in `docs/rules.md` and `docs/penalties.md`; this document only fixes how their text is shown. Every name in `CamelCase` code font below is admin config (ADR 0006); any value shown is a proposed default, not a constant.

## 1. Launch languages

| Code | Language | Script | Notes |
|---|---|---|---|
| `az` | Azerbaijani | Latin | Primary market. Source language for new keys. |
| `en` | English | Latin | Final fallback. Always complete. |
| `tr` | Turkish | Latin | Dotted/dotless i casing (4.6). |
| `ru` | Russian | Cyrillic | Three plural forms (4.2). |

Adding a language is an admin and bundle change plus a glyph check (4.5); it needs no client release unless the font set lacks the glyphs.

### 1.1 Selection

| Step | Rule |
|---|---|
| First launch | Walk the device locale list in order and take the first entry whose language subtag maps to a launch language. |
| Mapping | `az*` (any script or region) to `az`; `tr*` to `tr`; `ru*` to `ru`; `en*` to `en`. Other languages are skipped. If no entry maps, use `en`. |
| Persistence | The choice is stored as a player preference and synced with the account. It is a display setting only; the server never trusts it for game logic. |
| Change | Settings lists the four languages by their own name (Azərbaycanca, English, Türkçe, Русский). It applies immediately, without restart. |
| Later launches | The device locale is no longer consulted. |

Account sync (D-11 in `docs/privacy.md`): the account language is `accounts.language` (migration `00058_account_language`), empty until first set. `GET /profile/language` answers `{"language"}`; `PUT /profile/language` `{"language"}` stores one of the four launch codes (400 `language.invalid`, 404 `not_found` for an unknown account). It appears in the data export and is erased with the account. On the client, `Game/Profile/LanguageSync` reads it after every accepted hello for an account it has not read yet. An account without a language gets the device choice; after that, every change made in Settings while signed in is sent, one request at a time, with the latest choice. A failed read or save is retried at the next accepted hello. When the account holds a language that differs from the device choice, both are left as they are: which one wins on sign-in is an open owner decision.

### 1.2 Fallback order

A key is looked up in the selected language, then along its chain, ending in `en`.

| Selected | Chain | Why |
|---|---|---|
| `az` | `az`, `tr`, `en` | Azerbaijani and Turkish share Latin script and are largely mutually understood, so a Turkish string is a better stand-in than English for most `az` players. |
| `tr` | `tr`, `en` | The relation is not symmetric in practice: a Turkish player gets the neutral language rather than a text they did not choose. |
| `ru` | `ru`, `en` | No related launch language. |
| `en` | `en` | Final fallback. |

The chains are admin config (`LocaleFallbackChains`) and `en` is always last. The owner may change them without a release.

## 2. Text system

### 2.1 Keys and values

| Item | Rule |
|---|---|
| Key format | Stable dot-separated lowercase ids, `area.screen.item`, for example `match.result.win`. Characters: `a-z`, `0-9`, `.`, `_`. |
| Stability | A key is never renamed or reused for a different meaning. A retired key is marked deprecated in admin and removed once no supported client references it. |
| Value | A UTF-8 string in the ICU MessageFormat subset (4.3). |
| Ownership | Values are managed in admin as key-value per language, with the `en` reference and a per-language status (missing, draft, approved). Only approved values enter a published bundle. |
| Context | Each key has a translator note: where it appears, max length, what each placeholder is. |

Max length counts the visible characters of the longest text the value can produce: literal characters after NFC, combining marks not counted, placeholders and `#` not counted, and for a plural or select the longest branch.

Implemented in `server/internal/gamestrings` (#214): the key format, entries with note, max length and typed placeholders (the param types of section 3), value checks for the 4.3 subset with exactly the declared placeholders, plural only on `integer` or `decimal`, select only on `string`, every 4.2 category plus `other`, `=N` branches allowed, and the max length above; a bundle per language built from approved, non-deprecated values that is blocked as a whole when any of them fails a check, encoded as JSON with a SHA-256 content hash; and the missing-value report per language with percentage and key list.

Storage (migration `00011_game_strings`): entries and their values per language are stored with who changed them and when. A value may be saved as a draft that still fails its checks, so it can be fixed in the editor; it is approved only if it passes them. Publishing a language builds its bundle from the stored entries, refuses it with the list of failing keys, and otherwise stores it as the next version of that language with its content and hash. Publishing unchanged content returns the existing version instead of a new one. Stored bundles are append-only, so every earlier version stays available for rollback (2.2). The admin API routes for the editor, import, export and publishing are `docs/admin.md` 9.1, and every change made through them has an audit entry. The editor screens are the panel's `/strings` (`admin/README.md`). A catalog that names its records by game strings (cosmetics, missions, medals, shop offers, ranked seasons) is refused at publish while one of those name keys, or a medal's description key, has no approved `en` value (`docs/admin.md` 3.3); the refusal lists the keys in a structured `keys` field that the panel reads (owner decision 2026-11-20). Publishing a language also uploads the bundle to the CDN (2.2) in the same step: a bundle the content store does not take is not stored as a version.

Import and export for external translators: a file covers one language and every non-deprecated key in key order, with the note, max length, placeholders (`name:type`), the `en` reference (left out of an `en` file), the current text and its status. CSV has a UTF-8 byte order mark so spreadsheets read it correctly. Columns are `key`, `note`, `max_length`, `placeholders`, `en`, the language code and `status`. JSON is `{"language", "rows"}`. On import only the key and the text of the chosen language are read, so CSV columns may be reordered and the other columns edited freely, and a JSON file must name the chosen language. An empty text or one equal to the stored text changes nothing. Every other text is checked as for approval. A repeated, unknown or deprecated key, or a text that fails its checks, blocks the whole file with the list of failing rows, and nothing is stored. Otherwise the changed texts are stored in one transaction as drafts, so an imported text is approved only after the native-speaker review (6.1 step 2).

### 2.2 Delivery

| Item | Rule |
|---|---|
| Bundle | One versioned bundle per language, published from admin and served from the CDN as an immutable content-hashed file (ADR 0002). The active version per language comes from the server after the handshake (ADR 0004); URLs come from server config. |
| File | `<content_base_url>/strings/<language>/<hash>.json`, where `content_base_url` is the CDN root of `docs/content-delivery.md` section 2 and `<hash>` the lowercase SHA-256 hex of the file, the `hash` of its `string_bundles` entry. |
| Built-in bundle | The install package ships a fallback bundle for all four languages, built from the approved set at build time. The client works offline and on first start with it alone. |
| Update | The client downloads the active bundle for the selected language and `en` in the background and swaps it in at the next screen load, never mid-match. A failed download keeps the previous bundle. |
| Rollback | Switching the active bundle version in admin. |
| Hot path | Bundles load as one table per language. Strings for a live screen are formatted when the screen opens or a value changes, never per frame (zero GC alloc budget in `CLAUDE.md`). |

The active versions are the `localization` config domain (`docs/admin.md` 3.1), parsed by `gamestrings.ParseLocalization`: `{"bundles": {"en": 4, "ru": 2}, "fallbackChains": {"az": ["az", "tr", "en"]}}`. `bundles` must name `en`, and other languages may be left out, in which case they use the built-in bundle. A language without an entry in `fallbackChains` keeps its chain from 1.2. Every chain starts with its own language, ends with `en` and repeats no language. Before publishing, every named version must be a published bundle, and the `en` bundle must have every non-deprecated key. A rollback publishes the earlier version numbers as a new snapshot. The admin API publishes it (`PUT /admin/strings/active`, `docs/admin.md` 9.1), where lowering a language's version or removing its bundle counts as a rollback; `cmd/config-publish -domain localization` remains for operators. Every game server node watches the domain and resolves each named version to its content hash. Its accepted `ServerHello` carries `string_bundles` (language, version, hash) and `fallback_chains`. A snapshot that no longer resolves is logged, and the node keeps the previous one. The client therefore learns a new active version at its next connection.

### 2.3 Missing key behaviour

| Step | Condition | Behaviour |
|---|---|---|
| 1 | Key present in the selected language (downloaded bundle, else built-in) | Use it. |
| 2 | Missing | Try the next language in the chain (1.2). |
| 3 | Missing in the whole chain including `en` | Use `common.text.unavailable` from the built-in `en` bundle. If that is missing too, show an empty-state glyph, not words. |
| Logging | Any lookup that reaches step 2 or later | Log once per key per session (key, language, bundle version) and send it to telemetry so admin can list untranslated keys. |
| Release builds | | A raw key is never shown to a player. |
| Debug builds | | Show `[!key]` so testers see the gap. |

A value that fails to parse (bad placeholder syntax) is treated as missing for that lookup only.

### 2.4 Client implementation

Implemented in `client/Assets/HypeDomino/UI/Localization` (#70) as plain C#, covered by the `LocalizationTests` EditMode tests:

- **Selection:** `GameLanguage.Pick` applies 1.1: a saved launch language wins, else the first device locale whose language subtag maps, else `en`.
- **Saved choice:** `LanguageChoice` holds the language in use. `Game/LanguageBoot` creates it before the first scene from the saved `language.choice`. Only when no launch language is saved does it read the device locale list: Android `LocaleList.getDefault()`, iOS `NSLocale preferredLanguages` through `Plugins/iOS/HypeDominoLocale.mm`, none in the editor. The first pick is saved at once, so later launches no longer consult the device. `Choose` takes only a launch language, raises `Changed` when the language really changes, and the boot saves it.
- **Built-in bundle:** `UI/Localization/Resources/builtin-strings.json` holds the built-in value of each key per language. It sits in a `Resources` folder, like `Resources/TypeSet` (`docs/ui-motion.md` section 3), so it ships in the install package and loads without a scene or an Addressables download. `BuiltinStrings.Load` reads it once and unloads the asset; a missing file is logged as `Strings builtin missing resource=builtin-strings` and leaves the bundle empty. `LanguageBoot` then builds `LanguageBoot.Strings`, the app's one `GameStrings`, over it and the saved choice, with `[!key]` on in a debug build. The font atlases and `node tools/fonts/fonts.mjs` read the same file.
- **Lookup:** `GameStrings` runs 1.2 and 2.3 in the language of the `LanguageChoice` it is built with, so a chosen language applies to the next lookup at once. A language change and an applied bundle each raise `GameStrings.Changed`, the signal for live screens to format their strings again. The chain comes from `fallback_chains` through `SetChain`; a chain that does not start with its language or end with `en` is refused. Without a server chain, the 1.2 default applies. Each step reads the downloaded bundle, else the built-in one. Step 3 returns `common.text.unavailable` from the built-in `en` bundle, else an empty string for the view to show its empty-state glyph. A debug build shows `[!key]`. Each key reaching step 2 or later is logged once per session as `Strings missing key=... language=... version=...`.
- **Bundles:** `StringBundle` reads the `{"language","strings"}` JSON that the server publishes. It accepts the bundle only when its SHA-256 hex matches the `hash` of `string_bundles`. `GameStrings.Holds` tells whether a language already holds that hash. `Stage` keeps a downloaded bundle. `ApplyStaged` swaps it in at the next screen load, never in a match; a language change alone does not apply a staged bundle. `AppBoot` calls it through `AppBoot.ApplyStagedStrings` at every app state change, like the cosmetics manifest (`docs/cosmetics.md` 7.3), so a bundle downloaded after start is applied at the next sign-in, lobby, room, update or maintenance screen and raises `Changed` once; entering a match keeps it staged until the match is left. The `AppBootTests` EditMode tests cover it. A failed or mismatched download keeps the previous bundle.
- **Announcement:** `Game/AppBoot.Listen` takes the app's `GameStrings` and its `StringBundleUpdater` with the session, and an accepted `ServerHello` goes through `AppBoot.FollowStrings`. It passes each of `fallback_chains` to `SetChain`, logging a refused one as `Strings chain refused language=...`. It then returns the `string_bundles` entries to fetch: those for the selected language and `en` (2.2) whose hash `Holds` does not report, in the order the hello lists them. A bundle already held or staged with that hash is not fetched again, so a rollback to an earlier hash is fetched like any change. The hello handler logs each as `Strings fetch language=... version=...` and passes it to the updater (Download and storage). A hello that blocks play leaves the chains and bundles alone. `Listen` keeps the last accepted hello. When the language changes, through `GameStrings.Changed`, it first stages the kept bundle of the new language (Download and storage) unless that hash is already held, then fetches the bundles of the last accepted hello that the new language now wants and does not hold, the same way. Before any accepted hello only the kept bundle is staged. An applied bundle in the same language reads and fetches nothing. The `AppBootTests` EditMode tests cover it.
- **Download and storage:** `Content/StringBundleUpdater` takes each bundle `FollowStrings` returns from an accepted hello and downloads `ContentUrls.Strings`, the file of 2.2 under the hello's `content_base_url`, through `Content/UnityStringBundleSource`. A language outside the four launch languages is never requested, and a hash already held or being downloaded is not requested again. The bytes go to `Stage`, so a file whose SHA-256 hex differs from the announced hash, or that names another language, is refused and logged as `Strings fetch refused language=... version=...`; a failed request is logged as `Strings fetch failed language=... version=...`. Either way the previous bundle stays, and the next accepted hello tries again. A staged bundle is kept in the content folder of `docs/content-delivery.md` 7 as `content/strings/<language>/<version>_<hash>.json`, one file per language that replaces the previous one; it counts in the cache size shown to the player and a cache clear keeps it (CD-25, CD-26). Before the first scene `LanguageBoot` creates `LanguageBoot.Bundles` over the content folder, stages the kept bundles of the selected language and `en`, and applies them at once, since no screen is open yet. A kept file is read through `Stage` too, so a damaged or edited one is refused, logged as `Strings stored refused language=... version=...`, and the built-in bundle stays. The next hello then finds the kept hash held and fetches nothing. The editor neither keeps nor downloads bundles, like the content catalog. The request uses no timeout or retry of its own, because none is set for language bundles; a request that never ends keeps the previous bundle until the next start. `StringBundleUpdaterTests` cover it with a fake download over real files in a temporary folder.
- **Built-in keys:** besides the currency names and the seat mute keys (5.8), the built-in bundle holds the connection notice keys of `docs/ui-motion.md` (Connection notices), with `az` and `en` texts (owner decision of 2026-11-14), and the results screen keys of `docs/ui-motion.md` (Match result), whose `az` win, loss and button texts are the wording of #86. It also holds the mission load and reroll keys (`mission.load.*`, `mission.reroll.*`, `client/README.md` Missions, #128) with `en` and `az` texts, and the promo code reply keys of `docs/economy.md` 8 with `notice.generic`, whose `az` texts for an invalid, expired and already used code follow the wording of #109. Their `tr` and `ru` values are left to the translator; until then those languages fall back along 1.2 to `en`.
- **Ban keys:** the built-in bundle holds `ban.reason.<category>` for the six ban reason categories of `docs/admin.md` 6.2 and the ranked ban refusals `lobby.queue.refused.ranked_ban` and `lobby.room.refused.ranked_ban` (#186), with draft `en` and `az` texts awaiting the owner. `Game/Lobby/BanReasonKeys` maps the `BanReason` of a refusal and the sign-in `category` to the same key, `ban.reason.other` for an unknown or missing category. `LoginScreen.BanReasonKey`, `MatchQueue.BanReasonKey` and `FriendRoom.BanReasonKey` name it for the views (`client/README.md`); the views are not built yet.
- **Appeal keys:** the built-in bundle holds `appeal.sent`, `appeal.refused.<refusal>` (`expired`, `text_required`, `text_too_long`, `open`, `rejected`, `ended`, `not_appealable`), `appeal.offline` and `appeal.unavailable` for the answers of `POST /appeals` (`docs/admin.md` 6.4, Appeals, #186), with draft `en` and `az` texts awaiting the owner. `Game/Lobby/AppealKeys` maps each refusal code to its key and any other to `notice.generic` (`client/README.md`, Appeals).
- **Formatting:** `MessageFormat` renders the 4.3 subset with typed `string` and integer arguments. A plural picks an `=N` branch first, then the CLDR category of 4.2 for the value's language, then `other`. Numbers use the grouping of the selected language (4.4). A `decimal` argument keeps the digits it was given and uses the decimal separator of the selected language (`MessageFormat.Decimal`: `,` in `az`, `tr` and `ru`, `.` in `en`); rounding is the caller's. A `DateTimeOffset` argument is a timestamp shown in the device time zone (`MessageFormat.Timestamp`), in the CLDR numeric date order with a 24-hour clock: `M/d/yyyy, HH:mm` in `en`, `dd.MM.yyyy, HH:mm` in `ru`, `dd.MM.yyyy HH:mm` in `az` and `tr`; the instant comes from the server (`ServerClock`), never the device clock. A `TimeSpan` argument is a duration: whole seconds, never negative. It is written by the unit key of its largest unit (`MessageFormat.DurationKey`: `duration.days`, `duration.hours`, `duration.minutes` or `duration.seconds`), looked up in the language of the message and formatted as a message with the integer arguments `days`, `hours`, `minutes` and `seconds`, so the text decides which units it shows and their plurals (4.2). Until a unit key has a text in that language, the duration falls back to a clock of total hours, minutes and seconds (`MessageFormat.Clock`, `49:05:09`). A decimal is not a plural count. Formatting allocates, so it is for text that changes, never a frame.
- **Medal keys:** the built-in bundle also holds the medal load and showcase keys (`medal.load.*`, `medal.showcase.*`, `client/README.md` Medals, #130) with `en` and `az` texts; their `tr` and `ru` values fall back along 1.2 to `en` until translated.
- **Hype Pass keys:** the built-in bundle also holds the pass load and level purchase keys (`pass.no_season`, `pass.load.*`, `pass.levels.*`, `client/README.md` Hype Pass, #126) with `en` and `az` texts; their `tr` and `ru` values fall back along 1.2 to `en` until translated.
- **Shop keys:** the built-in bundle also holds the shop purchase, load and section title keys (`shop.buy.*`, `shop.load.*`, `shop.section.<section>.title`, `client/README.md` Shop and wallet, #105) with `en` and `az` texts, among them the four refusals `shop.buy.store_only`, `shop.buy.in_game_only`, `shop.buy.purchase_blocked` and `shop.buy.purchases_disabled` for the server codes `offer.store_only`, `offer.in_game_only`, `purchase.blocked` and `purchases.disabled`, and the store purchase keys `shop.purchase.pending`, `shop.purchase.store_unavailable`, `shop.purchase.not_granted` and `shop.purchase.unfinished` (`Game/Shop/ShopPurchase`); their `tr` and `ru` values fall back along 1.2 to `en` until translated. No text names a price, limit or time, since those come from the server and admin config.
- **Nickname keys:** the built-in bundle also holds the nickname rename reply keys (`nickname.rename.*`, one per reason code of 5.5 plus `cooldown`, `offline` and `rate_limited`, `client/README.md` Profile, #75) with `en` and `az` texts; their `tr` and `ru` values fall back along 1.2 to `en` until translated. The texts name no length or other admin-tuned value.
- **Privacy settings keys:** the built-in bundle also holds the Settings, Privacy reply keys (`privacy.settings.offline`, `privacy.settings.rate_limited`, `client/README.md` Settings screen, #157) with `en` and `az` texts; their `tr` and `ru` values fall back along 1.2 to `en` until translated.
- **Push settings keys:** the built-in bundle also holds the Settings, Notifications reply keys (`push.settings.offline`, `push.settings.rate_limited`, `client/README.md` Settings screen, `docs/push.md` 3, #157) with `en` and `az` texts, the same wording as the privacy settings keys; their `tr` and `ru` values fall back along 1.2 to `en` until translated.
- **Data export keys:** the built-in bundle also holds the Settings, Account data export reply keys (`account.export.cooldown`, `account.export.unavailable`, `account.export.offline`, `account.export.rate_limited`, `client/README.md` Settings screen, #157) with `en` and `az` texts; their `tr` and `ru` values fall back along 1.2 to `en` until translated. The cooldown text names no date or day count, since both come from the server and admin config.
- **Account deletion keys (draft):** the built-in bundle also holds the Settings, Account deletion reply keys (`account.deletion.*`, one per `AccountDeletionOutcome` with a text plus `account.deletion.pending_undated`, `client/README.md` Settings screen, `docs/privacy.md` 8.2, #157) with `en` and `az` texts. They are drafts for review like the duration texts: `docs/privacy.md` gives no wording, and this note is their draft mark. `account.deletion.pending` takes the `deleteAt` timestamp from the server's `delete_at`; no text names the grace length, since it is `AccountDeletionGraceDays`. Their `tr` and `ru` values fall back along 1.2 to `en` until translated.
- **Profile and report keys:** the built-in bundle also holds the profile load keys (`profile.load.*`) and the report reply keys (`report.send.*`, 5.8, `client/README.md` Profile, #75) with `en` and `az` texts; their `tr` and `ru` values fall back along 1.2 to `en` until translated. `report.send.received` is the generic acknowledgement of 5.8 and says nothing about the outcome.
- **Friend room keys:** the built-in bundle also holds the friend room refusal keys (`lobby.room.refused.*`, `docs/friend-lobby.md` 4, Refusal texts, #154) with `en` and `az` texts; their `tr` and `ru` values fall back along 1.2 to `en` until translated.
- **Duration texts (draft):** the built-in bundle holds `en` and `az` texts for the `duration.*` unit keys (owner decision 2026-11-20, #214): `duration.days` shows days and hours, `duration.hours` hours and minutes, `duration.minutes` minutes and seconds, and `duration.seconds` seconds, each unit with its plural (4.2; `az` keeps the singular noun). They are drafts for review: the native-speaker review of 6.1 step 2 has not passed them, and the built-in bundle has no status field, so this note is their draft mark. The reviewed texts are entered on `/strings` and approved there; once published they replace the built-in ones, and the built-in bundle takes the reviewed wording in the same change that removes this mark. `tr` and `ru` have no unit text, so a duration in those languages still shows the clock fallback. The pass and mission countdowns (#126, #128) are their first users.

- **Tutorial keys:** the built-in bundle also holds the tutorial step, control and first offer prompt keys (`tutorial.*`, `tutorial.offer.*`, `docs/tutorial.md`, `client/README.md` Tutorial, #88) with `en` and `az` texts; their `tr` and `ru` values fall back along 1.2 to `en` until translated. The step texts name the scripted tiles and points of TU-2 and the target score of R-43.
- **Cache keys:** the built-in bundle also holds the cache clear outcome keys (`settings.cache.*`, `docs/content-delivery.md` 7, #160) with `en` and `az` texts; their `tr` and `ru` values fall back along 1.2 to `en` until translated. The texts name no size or limit.
- **Unavailable text:** the built-in bundle holds `common.text.unavailable`, the text of 2.3 step 3, with `en` and `az` texts. Step 3 reads only the `en` value, so every player sees `Text unavailable` for a key missing in the whole chain; the `az` text is a draft for review like the duration texts and is not shown until step 3 reads the selected language. The `LocalizationTests` EditMode tests cover it in all four languages, and the duration clock fallback of `tr` and `ru` over the shipped bundle.

Not built yet:
- **Account sync:** syncing the choice with the account (`docs/privacy.md` D-11) needs a server endpoint that does not exist yet.
- **Wiring:** Settings has no screen yet; `GameLanguage.LaunchCount`, `LaunchAt` and `OwnName` give its list of the four languages by their own name, and `Game/SettingsModel` holds the choice for it.

## 3. Server-originated text

The server never sends rendered text for display. It sends `key` plus typed parameters and the client renders in the player's current language, so a language change re-renders stored notices.

| Rule | Detail |
|---|---|
| Shape | `{ key, params: { name: typed value } }`. Param types: integer, decimal, string, timestamp (UTC instant), duration (seconds), item id. |
| Items and currencies | Sent as ids. The client resolves the display name from the localized catalog, not from the notice. |
| Numbers | Sent raw. Formatting per locale is the client's job (4.4). |
| Applies to | Rewards, system notices, penalty notices and appeal outcomes (`docs/penalties.md`), error and rejection reasons, rank changes. |
| Errors | Responses carry a stable reason code that maps to a key, never prose. |
| Player-supplied strings | Nicknames and custom room names pass through as parameters, are not translated, and are inserted by placeholder (4.7). |
| Unknown key (newer server, older client) | Handled by 2.3 with the generic `notice.generic` key, never the raw key. |
| Server logs | English; this section covers player-facing text only. The admin panel UI has its own languages (`docs/admin.md` section 10). |

## 4. Text rules

### 4.1 Text expansion budget

Layouts are designed against the longest language. Admin enforces a max length per key.

| Language | Typical length vs `en` | UI design budget |
|---|---|---|
| `en` | 100% | Reference |
| `az` | about 110% | 130% |
| `tr` | about 115% | 130% |
| `ru` | about 120% to 130% | 150% |

Short labels (buttons, tabs, badges) expand worst: design them to 200% or allow wrap or shrink within admin-set limits. Every screen is reviewed in `ru` and `tr` before it ships.

### 4.2 Plurals

Plural form is chosen by CLDR category in the message, never by code testing for `1`.

| Language | Categories | Note |
|---|---|---|
| `en` | `one`, `other` | |
| `ru` | `one`, `few`, `many`, `other` | `one`: 1, 21, 31 (not 11). `few`: 2-4, 22-24 (not 12-14). `many`: 0, 5-20, 25-30. `other`: fractions. |
| `az` | `one`, `other` | After a number the noun stays singular (`5 oyun`); write count messages with the singular noun in `other`. |
| `tr` | `one`, `other` | Same: no plural agreement after a number. |

Admin rejects a plural message that lacks a category its language needs plus `other`.

### 4.3 Placeholders

ICU MessageFormat subset: `{name}`, `{count, plural, one {...} few {...} many {...} other {...}}`, `{kind, select, ...}`.

- Names are descriptive (`{opponentName}`, `{coins}`) and identical across languages for one key.
- Translators may reorder placeholders; code never assumes order.
- A value uses exactly the placeholders declared for its key; admin rejects missing or extra ones.
- Placeholders carry typed values; the formatter, not the translator, formats numbers, dates and durations.

### 4.4 Number, date and duration formatting

| Item | Rule |
|---|---|
| Locale source | The selected game language, not the device locale, so one screen is internally consistent. |
| Numbers | Locale grouping and decimal separators (for example `1 234,5` in `ru`, `1.234,5` in `az` and `tr`, `1,234.5` in `en`). Digits are always Latin. |
| Dates and times | Locale order, 24-hour clock. Timestamps arrive in UTC and display in the device time zone. Time logic never uses the device clock. |
| Durations | Built from localized unit keys with plural rules (4.2), not concatenated fragments. |
| Compact numbers | Localized compact units only where a layout key allows them. |

### 4.5 Font glyph coverage

| Requirement | Detail |
|---|---|
| Latin | `a-z` plus `ə Ə ğ Ğ ı İ ş Ş ç Ç ö Ö ü Ü` (az, tr). |
| Cyrillic | The Russian alphabet including `ё Ё`. |
| Shared | Digits, punctuation, currency signs, UI symbols. |
| Display fonts | Every stylised or effect font contains all of the above, or has a per-script fallback font. A missing glyph (tofu box) blocks release. |
| Atlas | Static SDF atlases holding the sets above and every character of the built-in bundle (`docs/ui-motion.md` section 3), so a match never adds a glyph at runtime. They ship in the install package. |
| Check | `node tools/fonts/fonts.mjs` (in CI through `npm run test:tools`) checks every role's fallback chain, in the font files and in the generated font assets, against these sets and every character of the built-in bundle; `TypeSetTests` checks the same in the editor. Approved bundle values published from admin are not checked yet. |

### 4.6 Casing: the Turkish i trap

| Language | Pairs |
|---|---|
| `tr`, `az` | `i`/`İ` and `ı`/`I` |
| `en`, `ru` | `i`/`I` |

- Every case change of display text uses the selected language's culture (`ToUpper(culture)`, `ToLower(culture)`). Invariant casing is only for identifiers and keys.
- A bare `ToUpper()` or `ToLower()` on display text is a review and lint item: in `tr` and `az` it turns `i` into `I`.
- Headlines that must be uppercase are stored as their own approved value rather than uppercased at runtime.
- Case-insensitive comparison of player text uses the normalisation in 5.2, not display casing.

In the client the lint is `RuntimeCodeRulesTests` (EditMode, #70): runtime code that calls `ToUpper()` or `ToLower()` without a culture, `char.ToUpper` or `char.ToLower` without a culture, or sets the TextMeshPro `FontStyles.UpperCase`, `LowerCase` or `SmallCaps` style fails the test. `ToUpperInvariant` and `ToLowerInvariant` stay allowed for identifiers and codes (room codes, promo codes, locale tags). No display text is case-changed at runtime today (`docs/ui-motion.md` section 3).

### 4.7 No sentence concatenation

A sentence is one key with placeholders. Code never joins fragments (`name + " " + verb + count`): word order, case endings and plurals differ per language. Lists use a localized list-pattern key. `az`, `tr` and `ru` inflect nouns and names, so keys that insert a player name are phrased without inflecting it (a label followed by the name).

## 5. Nickname policy

The server alone validates, normalises and stores nicknames. The client may pre-check for responsiveness but never decides.

Implemented in `server/internal/profile` (#59): 5.1 (`NicknameAllowedSet` is fixed to the sets below and not configurable), 5.2 steps 1 to 4 and uniqueness on the step 4 form, 5.4, the format and taken codes of 5.5 with `nickname.invalid_separators`, `nickname.mixed_scripts`, `nickname.digits_only` and `nickname.cooldown` added, the registration default, and the frequency and history rows of 5.6. The word filter and reserved names of 5.3 with the step 5 and 6 forms of 5.2 and `nickname.not_allowed` are #60: the `moderation` config snapshot `{"words": {"any"|"az"|"en"|"tr"|"ru": [{"entry", "match": "substring"|"word"}]}, "reserved": [...]}`, published from the admin panel, read by every node every `HD_CONFIG_POLL`; entries are folded like the nickname, a substring entry matches the step 6 form, a word entry matches a whole word of the step 5 form or the whole step 6 form, and a reserved name matches as a substring. Before the first publish nothing is filtered. Staff accounts are admin accounts and hold no nickname, so no player may use a reserved name. Lazy revalidation at login (5.6 rule changes) is not built yet; a moderator rename covers names that slip through. The rate limit is #184, the rename price is #101. The admin names map to the server variables `HD_NICKNAME_MIN_LENGTH`, `HD_NICKNAME_MAX_LENGTH`, `HD_NICKNAME_ALLOW_MIXED_SCRIPT`, `HD_NICKNAME_RENAME_COOLDOWN` and `HD_NICKNAME_RELEASE` until admin config serves them.

### 5.1 Format

| Item | Rule | Proposed default (admin config) |
|---|---|---|
| Length | Grapheme clusters, after normalisation | min 3, max 16 (`NicknameMinLength`, `NicknameMaxLength`) |
| Allowed characters | Only letters the `name` style draws (`docs/ui-motion.md` section 3): Basic Latin `A-Z a-z`, the Azerbaijani and Turkish letters `Ə ə Ğ ğ İ ı Ş ş Ç ç Ö ö Ü ü`, the Turkish circumflex letters `Â â Î î Û û`, and Russian Cyrillic `А-Я а-я Ё ё`; digits `0-9`; and `_`, `.`, `-`, single space | `NicknameAllowedSet`, fixed to the font atlases |
| Not allowed | Every other letter, also Latin and Cyrillic ones outside the sets above (for example `é`, `ß`, `ä`, `ñ`, `ł`, `і`, `ї`, `ў`, `ә`), combining marks, emoji, symbols, control and format characters (zero-width, bidi overrides), leading or trailing separators, more than two identical separators in a row | fixed |
| Mixed scripts | Latin and Cyrillic letters may not be mixed within one word | `NicknameAllowMixedScript` = off |
| Digits only | Rejected | fixed |

The allowed set is narrowed to the static font atlases instead of growing the atlases (owner decision 2026-11-12, #597). It lives in one file, `server/internal/profile/nickname-characters.json` (letters per set, digits, separators), which the server embeds and validates against and which `node tools/fonts/fonts.mjs` (CI, `npm run test:tools`) checks against the `name` style chain (`display-900 > ui-400 > uiCyrillic-400`) in both the font files and the generated font assets, so a nickname character the atlases cannot draw fails CI. Adding a letter means adding it to the atlases (`fonts.json`) and this file in one change. Any letter outside the set, such as `é`, `ß` or `і`, is refused with `nickname.invalid_characters`. After NFKC every allowed letter is one code point, so a combining mark is always refused, and the length counts code points. The client sends the typed text without a pre-check (`client/Assets/HypeDomino/UI/Profile/NicknameRename`, #75) and shows the server's reason code; if it ever pre-checks, it uses the same file for UX only, the server decides.

Names stored before #597 that hold a letter outside the set are not migrated: the game has no production players before launch, so only development and test data can hold one. Such a name stays readable as stored (it may draw a box in the `name` style) until the player renames; a rename to the same text is refused like any other invalid name, and a moderator can force a rename (5.6). Lazy revalidation at login, when it is built, covers these names with the rest of 5.6.

### 5.2 Normalisation before filtering and uniqueness

Applied in order. The stored display form is the player's text after step 1; the later steps produce comparison forms only.

| Step | Operation |
|---|---|
| 1 | Unicode NFKC. |
| 2 | Strip zero-width and format characters, collapse whitespace, trim. |
| 3 | Full Unicode case folding, locale independent, then `İ`, `I`, `ı` and `i̇` (i with combining dot) all map to `i`. Standard folding alone keeps `İ` apart from `I`, so this explicit step stops Turkish casing from dodging uniqueness. |
| 4 | Confusable folding to a canonical letter from a maintained table. Includes Latin/Cyrillic look-alikes (`а`/`a`, `е`/`e`, `о`/`o`, `р`/`p`, `с`/`c`, `у`/`y`, `х`/`x`, `і`/`i`) and the az/tr set (`ə`/`a`, `ı`/`i`, `ö`/`o`, `ü`/`u`, `ş`/`s`, `ç`/`c`, `ğ`/`g`). |
| 5 | Leetspeak folding (`0`->`o`, `1`->`i`, `3`->`e`, `4`->`a`, `5`->`s`, `7`->`t`, `@`->`a`, `$`->`s`). |
| 6 | A second form with separators (`_ . - space`) removed, so split spellings still match the word filter. |

The word filter and reserved names match on the step 5 and 6 forms. Uniqueness uses the step 4 form (leetspeak digits are legitimate in names, so they are not folded for uniqueness).

### 5.3 Filtering and reserved names

| Item | Rule |
|---|---|
| Word filter | One list per language (`az`, `en`, `tr`, `ru`) plus a language-independent list, maintained in admin with entry, match type (substring or whole word) and a list version. All lists apply to every nickname regardless of the player's language. |
| Categories | Profanity, sexual content, hate and slurs, violence and self-harm, drugs, personal data patterns (phone, email, URL), impersonation. The lists are owned by the owner or moderators; this document does not contain the words. |
| Reserved names | Staff and system identities (for example `admin`, `moderator`, `support`, `system`, the game name, and localized equivalents) matched after folding, including as a substring. Only staff accounts may hold them. Admin config `ReservedNames`. |

### 5.4 Uniqueness

Unique across the game on the step 4 comparison form. Check and write are one atomic server operation so concurrent requests cannot both win. `Ayşe` and `Ayse` are both stored as typed but cannot coexist. A released nickname (rename or deleted account) becomes available after `NicknameReleaseDays`.

### 5.5 Rejection

| Case | Result |
|---|---|
| Format violation | Reason code (`nickname.invalid_length`, `nickname.invalid_characters`, ...), shown as a localized message. |
| Taken | `nickname.taken`; the server may offer suggestions. |
| Filter or reserved hit | One generic `nickname.not_allowed`. The matched word or list is never revealed. |
| Repeated attempts | Rate-limited per account and device (`NicknameAttemptsPerHour`). |
| Registration | Cannot continue without an accepted nickname. A server-generated default is offered as a one-tap option. |

### 5.6 Renaming

| Rule | Detail |
|---|---|
| Frequency | One free rename at setup. Later renames are limited by `NicknameRenameCooldownDays` and may cost currency (price is economy config, not set here). |
| Moderator rename | After a confirmed report a moderator forces a rename. The player cannot enter ranked matches or custom rooms until they pick a valid name. A forced rename does not start the cooldown. |
| Rule changes | When the filter or confusables table changes, stored names are revalidated lazily at login; new violations force a rename. |
| History | Previous nicknames are kept per account for moderation and never shown to players. |

### 5.7 Scope: no free-text chat

In-game communication is emoji and quick messages only (Epic 13, #149). There is no free-text chat, so the filter applies to exactly two player-authored strings.

| String | Rules |
|---|---|
| Nickname | 5.1 to 5.6. |
| Custom room name | Same normalisation (5.2) and word filter (5.3). Own limit `RoomNameMaxLength` (proposed default 24). No uniqueness. A rejected name is replaced by a generated default. |
| Emoji and quick messages | A fixed catalog of keys (section 2). Only ids travel over the wire, never player text. |

Quick messages (#152, owner decisions 2026-11-08):

- The catalog is the `quick_messages` config snapshot `{"keys"}`: game string keys `quick.<slug>`, each listed once. Every player may send every listed key; there is no ownership. A key not in the published catalog is dropped, and before the first publish every key is.
- A `QuickMessageIntent` carries `key` and `team_only`. Without `team_only` every seat sees it; with it only the sender's team does, so in 1v1 only the sender. Every seat that sees it gets `QuickMessageShown` with the sender's seat, the key and `team_only`. The client shows the key's text in the viewer's language. Owner decision 2026-11-14: the text stays on screen for the emoji duration (`TokenEffect.EmojiDurationMs`) at the seat anchor where that seat's emoji appears (`docs/ui-motion.md` 7, Quick message).
- It spends from the same per-seat bucket as emoji (`HD_EMOJI_BURST`, default 3, refilled one every `HD_EMOJI_REFILL`, default 2 s), so emoji and quick messages share one limit. A bot seat, a muted player (`docs/admin.md` 6.2) and a seat with no connection send nothing.
- Nodes read the snapshot at start and refresh it every `HD_CONFIG_POLL`. The panel edits it at `/quick-messages` (A-27 rights: view with view on game strings, publish and rollback with publish on game strings). A publish is refused with `invalid` and the failing keys in `detail` unless every key has an approved, non-deprecated value in every game language (section 2.1).

Any future feature that accepts free text must update this section and add moderation before it ships.

### 5.8 Reports

| Step | Rule |
|---|---|
| Entry | A report action on a player profile and on the match end screen with fixed reasons: offensive nickname, offensive room name, impersonation, other. No free-text comment. |
| Record | Reporter, target, reason, the name text at report time, match id. Repeat reports by one reporter on one target merge. |
| Review | Reports go to a queue in the admin panel. A moderator approves the name, forces a rename, or adds a pattern to the word filter. Automatic action thresholds are admin config (`ReportAutoRenameThreshold`). |
| Abuse | Reporters with many rejected reports are rate-limited. |
| Feedback | The reporter gets a generic localized acknowledgement, not the outcome. The reported player gets a localized notice (reason key plus the rename requirement). |

Implemented in `server/internal/report` (#155, owner decisions 2026-11-08):

- `POST /reports` `{"target","reason","match_id"}` with the player's access token. `target` is an account id; `reason` is `offensive_nickname`, `offensive_room_name`, `impersonation` or `other`; `match_id` is optional. Errors: 400 `report.reason`, `report.self`, `report.match`; 404 `report.unknown_player`. Every other outcome, also a dropped report, answers 202 `{"status":"received"}`, the generic acknowledgement.
- A report stores the target's nickname at report time (`player_reports.name_text`). Custom room names are not stored yet, so a room name report keeps the nickname until they are. While a report of one reporter on one target is open, a new one replaces its reason, name and match.
- `ReportAutoRenameThreshold` (`HD_REPORT_AUTO_RENAME_THRESHOLD`, default 5, 0 turns it off): when that many different reporters have an open `offensive_nickname` report on one target updated within `HD_REPORT_AUTO_RENAME_WINDOW` (default 7 days), the target must pick a new nickname (`rename_required`, as a forced rename). The reports stay open for a moderator.
- Abuse: a reporter whose reports were touched `HD_REPORT_DAILY_LIMIT` times (default 10) in the last 24 hours, or who has `HD_REPORT_REJECTED_LIMIT` (default 5) reports rejected within `HD_REPORT_REJECTED_WINDOW` (default 30 days) with the latest rejection less than `HD_REPORT_REJECTED_SILENCE` (default 7 days) ago, is answered as usual but nothing is stored.
- Muting another player's emoji and quick messages during a match is a choice on the device only; the server is not involved. Owner decision 2026-11-14: the mute lasts for the current match only and is never stored on the device; it is chosen in a menu opened by a long press on that player's seat avatar (`docs/ui-motion.md` 7, Seat mute).
- Admin queue (A-32, `queues` object): `GET /admin/reports` lists open reports grouped by target, oldest target first, 50 targets a page (at most 200 through `limit`): `reason` keeps targets with an open report of that reason, `account` is an account id prefix search and `after=<account id>` asks for the next page (#148). Each target also carries its earlier decisions, and its `due_at` (the oldest open report's time plus `ReportSlaHours`, default 48) and `overdue` flag from the server clock (`docs/admin.md` 6.4). `POST /admin/players/{id}/reports/uphold` and `/reject` take a reason and a recent sign-in, close every open report on the player and are audited on the player as `uphold_reports` or `reject_reports`; 409 `no_open_reports` when none is open. Upholding changes nothing on the account; the moderator forces a rename or mutes from the player page. Rejected reports feed the reporter silence above.

## 6. Cultural content checklist

Applies to every art asset, animation, sound, store screenshot and text before it ships. Launch markets are Azerbaijan, Turkey and Russian-speaking regions other than Russia, which is not a store country at launch (`docs/privacy.md` 9.3); content must also pass the global store review.

| Area | Rule |
|---|---|
| Colours | UI, effect, team, score and role colours come only from colour tokens. Cosmetic illustration art (avatars, tables, frames, tile skins, emoji) may use its own palette, but is never dominated by a team or role hue. No colour used to evoke a national flag, party or political side. A status colour is never the only signal; add shape or label. |
| Symbols | No national or party emblems, no military insignia, no symbols tied to banned organisations. Carpet and filigree ornament is an original geometric or floral pattern with no script, emblem or religious symbol worked into it. |
| Flags and maps | No flags as decoration and no flag standing in for a language (the language list uses names). No maps in launch content. Any later use needs owner review for disputed territory. |
| Religion | No religious symbols, scriptures, places of worship or clergy in assets or cosmetic themes. Holiday events are reviewed per region by the owner. |
| Gestures | No avatar or emote with a hand or body sign that is offensive in any launch market. The emote list is reviewed for this. |
| Gambling imagery | Dominoes with coins must not read as a casino. Avoid slot and roulette cues, card suits, chips, cash piles, spinning-reel reward presentation and the words jackpot, bet, wager. Currency is abstract game tokens. Random rewards follow store disclosure rules where they apply. |
| Alcohol, tobacco, drugs | None in assets, avatars, items or text. |
| Violence and fear | Cartoon impact effects only. No weapons, blood or real tragedies. |
| Sexual content | No sexualised figures, poses or outfits. Avatars are all-ages. |
| Representation | No ethnic, regional, gender or disability stereotypes in avatars or names. AI-generated people are checked for artefacts and for resemblance to real people. |
| Likeness and brands | No real persons, brands, third-party game names or look-alike styles. The art source (including AI-generated) is recorded in the manifest. |
| Text in art | No text baked into images; every word is a key (section 2). Numerals and icons read the same in all four languages. |
| Motion and sound | Flashing stays within the photosensitivity limits (admin config) and a reduce-effects setting exists. No sudden loud sounds. |
| Wording | A word acceptable in one launch language but offensive in another is not used. A native speaker reviews each language's text. |

### 6.1 Review step

| Step | Who | When |
|---|---|---|
| 1. Self-check against section 6 | Asset author | Before upload to admin |
| 2. Native-speaker review of text values (`az`, `tr`, `ru`, `en`) | Reviewer per language | Before a value is set to approved |
| 3. Cultural review of art and sound | Owner or delegate | Before the manifest entry can go live; recorded as approved in the manifest |
| 4. Store-facing set (icon, screenshots, descriptions, trailer) | Owner | Before each store submission |

The pipeline does not publish an asset without an approved cultural review. A shipped asset found in violation is switched off from admin, its static fallback included, and fixed before it returns.

## 7. Age rating targets

A proposal for the owner to confirm. The stores' questionnaires decide the final rating from the real content at submission.

| System | Proposed target | Condition |
|---|---|---|
| IARC (Google Play and others) | Lowest tier (all ages) | No simulated gambling; no free-text chat; purchases and random items disclosed |
| PEGI | 3 | Same; in-app purchase and random-item notices may apply |
| App Store | 4+ | No gambling imagery; no unrestricted user text; purchase and random-reward answers given honestly |
| Turkey | Follow the local classification shown on the store listing | Checked at submission |

Features that would raise the target and need an owner decision before they are built: free-text chat, voice chat, real-money prizes, casino-style reward presentation. Nicknames and room names are user-generated text, so the store answer is "moderated user content" and the report flow (5.8) must exist before submission.
