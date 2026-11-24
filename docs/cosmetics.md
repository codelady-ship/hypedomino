# Cosmetics catalog

Data model for the five cosmetic types: shared fields, type fields, acquisition sources, the published manifest, Addressables addressing and validation. The contract is [`docs/cosmetics/schema.json`](cosmetics/schema.json) (JSON Schema draft 2020-12); this document gives its meaning and the rules a schema cannot express. Valid examples: [`docs/cosmetics/examples/`](cosmetics/examples/). `node tools/cosmetics-schema/cosmetics-schema.mjs [manifest.json ...]` checks manifests against the schema and the single-manifest rules in section 9 (without arguments it checks the examples); CI runs its tests through `npm run test:tools`.

Delivery follows `docs/adr/0002-asset-delivery.md`, publishing and versioning follow `docs/adr/0006-admin-panel.md`, name keys follow `docs/localization.md` section 2, and the static or animated choice follows the effect levels in `docs/ui-motion.md` section 7.

## 1. Owned elsewhere

This document stores references to these decisions and never their values.

| Decision | Owner | How an item refers to it |
|---|---|---|
| Rarity tiers, their order, names, colours, the status diamond (shape, colour, glow) and the rarity ordinal used for table background selection | `docs/rarity.md` | `rarityId`, an id in the rarity table. No tier, name, ordinal or colour is defined here |
| Status colour | `docs/rarity.md` | Derived from `rarityId`; the manifest has no colour field |
| Table background selection at match start | `docs/rarity.md` section 5, #98 | Uses the rarity ordinal of each seat's equipped table |
| Prices, currencies, discounts, offers | `docs/economy.md` (#7) | A `shop` or `promo` source holds an offer or campaign id, never a price |
| Pass, mission and medal rewards | `docs/pass-missions-medals.md` (#9) | A `pass` or `mission` source holds a reward or mission id |
| Ranked season rewards | #8 | A `rank_reward` source holds a season reward id |
| Animated asset format and player, active animation budget per screen | #93, ADR 0012 | An address; the asset behind it is a sprite sheet loop |
| Emoji groups and equipped emoji slot count | #150 | `group` id, free text with no fixed list (`docs/admin.md` O-12); slot count is admin config there |
| Download and prefetch policy | #68 | Addresses and labels in section 8 |
| Grant engine and duplicate grants | #102 | Every grant goes through the reward bundle engine |

## 2. Types

| `type` | What it is | Where it shows |
|---|---|---|
| `avatar` | Player portrait | Profile, seat, lists, shop, collection |
| `frame` | Border around the avatar | Everywhere the avatar shows |
| `table` | Match table background | Match scene, collection, shop |
| `tile` | Back face of the player's domino tiles | Match scene: the player's face-down tiles as the other seats see them, including the deal (`docs/ui-motion.md` Hand motion); collection, shop |
| `emoji` | Reaction sent by id in a match or a friend lobby (`docs/localization.md` 5.7) | Match seat, friend lobby, emoji wheel, collection |

Adding a type is a schema change (section 7.2) and a change to this table.

In the server's catalog engine (`server/internal/cosmetics`) a new type is one entry in `Types` with its constant: the id pattern is built from that list, and every type except `table` gets the one-animation fields of section 4 (`animationAddress`), so validation and the stamped addresses need no new code. The admin panel reads its type list from the server. The type also needs a `live` `default` item in the built-in manifest and the examples (section 7.3), an equip slot in `server/internal/inventory` and a `GuidePriceTypePercent` value (`docs/economy.md` 3.1). The tests in `server/internal/cosmetics/schema_test.go` fail until the schema's type, source type, availability and `artSource` lists, its id, `nameKey` and address patterns and its one-animation branch match the server.

- A `tile` skin changes only the back face. The face side, pips and the divider are the same for every player, so a skin never affects how a tile's value reads (`docs/art-direction.md` section 3). Boneyard tiles (R-3) belong to no seat and use the first `live` `default` tile item.
- An animated `tile` plays one animation per seat: every face-down tile of that seat draws from the same atlas entry on the same animation clock, so a seat costs the same whether it holds 1 or 14 tiles.
- The match view sends each seat's equipped skin as `RosterEntry.tile`, taken from the loadout frozen at match start and stored with the match. A bot seat sends none and shows the default. The client draws a seat's face-down tiles with `UI/Cosmetics/TileBackView`: the first tile loads the skin and holds its one animation slot, and the others show the same frame. All of them use one texture, so they batch into one draw call.
- Owner decision 2026-11-12: the game has no 3D characters anywhere, so the `character` type (a full-body figure at the seat and in the friend lobby) and the `action` type (lobby motions for that figure) are removed. A seat shows the avatar, its animated frame and the nickname; the friend lobby shows avatars and frames (`docs/friend-lobby.md`).
- Owner decision 2026-11-12: these proposed types are not added before launch: player title, profile banner, chat bubble, tile-place effect, win effect and sound pack. Each is a schema change with its own art and performance work. The tile-place and win effects come first after launch, because every player sees them in every match.

## 3. Shared fields

All required on every item.

| Field | Type | Rule |
|---|---|---|
| `id` | string | `<type>.<slug>`, slug `[a-z0-9_]+`. Stable: never renamed, never reused for another item, its type never changes. Ownership, equips, rewards and wire messages use it |
| `type` | enum | One of section 2. Equals the `id` prefix |
| `revision` | integer >= 1 | Item version. Increases by 1 on every publish that changes any field or asset of the item. Never decreases; a rollback republishes the earlier content under a new revision |
| `nameKey` | string | Always `cosmetic.<id>.name` (for example `cosmetic.avatar.comet_fox.name`). The localized value lives in the language bundles, not in the manifest |
| `rarityId` | string | Reference into the rarity table of `docs/rarity.md` |
| `animated` | boolean | True if the item has animation assets. Decides which type fields are required (section 4) |
| `staticAddress` | address | The static sprite. Mandatory for every item, animated or not: it is the asset of a static item and the fallback of an animated one (section 8.3) |
| `availability` | enum | `live`, `retired` or `disabled` (section 6) |
| `sources` | array, at least 1 | How the item is obtained (section 5) |
| `artSource` | enum | `handmade` or `ai_assisted`. Records AI involvement as `docs/localization.md` section 6 requires |
| `culturalReview` | const `approved` | The review in `docs/localization.md` 6.1 passed. An item without it cannot be published |

## 4. Type fields

| Type | Field | When | Rule |
|---|---|---|---|
| `avatar`, `frame`, `tile`, `emoji` | `animationAddress` | Required if `animated`, forbidden otherwise | One looping animation |
| `table` | `layers` | Required if `animated`, forbidden otherwise | Back-to-front list of `{ name, address, animated }`, at least 1, names unique, at least one layer animated. Count at most `TableMaxLayers` (admin config; its value comes from the measured base-scene cost in #97, ADR 0008) |
| `emoji` | `group` | Required on emoji, forbidden on other types | Emoji group id (#150) |

A field of another type is invalid. Frames carry no effect field: the rarity glow around a frame is drawn from `rarityId` as `docs/rarity.md` section 2 defines.

A static table is one flattened sprite (`staticAddress`), one draw call. An animated table composes its layers; its `staticAddress` is the same scene flattened, used wherever the animated version cannot play.

## 5. Acquisition sources

A source is `{ "type": ..., "ref": ... }`. `ref` is the id of a record owned by another system; the record, not the cosmetic, holds price, amount, chance, level and time window. Duplicate sources on one item are invalid.

| `type` | `ref` points to | Owner |
|---|---|---|
| `default` | No `ref`. Granted to every account at creation; cannot be removed. The item is static (V-19) | This document |
| `shop` | Shop offer id | #7, #104 |
| `pass` | Pass reward id (season, track, level) | #9, #123 |
| `mission` | Mission id | #9, #127 |
| `promo` | Promo campaign id | #7, #108 |
| `first_purchase` | First-purchase bonus bundle id, granted once with the first validated real-money purchase (`docs/economy.md` 7) | #106 |
| `rank_reward` | Ranked season reward bundle id (`RankRewards` in the `rating` config) | #8, #120 |
| `event` | Live event id | Admin live-ops config |
| `chest` | Chest offer id. Only `common` and `rare` items (`docs/economy.md` 3.4) | #351 |

Rules:
- `sources` is the complete list of ways an item can be granted. The admin rejects publishing an offer, reward, mission, campaign or event that grants an item whose `sources` does not list it, and the server's grant path (#102) refuses such a grant. This is what the collection screen shows as the way to earn an item (#99) and what abuse checks (#211) compare against.
- A source listed on an item does not mean it is active now; whether a shop offer is on sale or an event is running is decided by the referenced record and server time.
- Every type has at least one `live` item with a `default` source, so a new account has something equipped in every slot and a disabled item has a replacement (section 6).
- The `default` source and the built-in install of default items (8.2) are owner-confirmed (#6, 2026-10-26). Default items are static (`animated: false`) to protect the install size budget (V-19).

## 6. Availability

| Value | Shown in shop and earn paths | Owners keep it | Equippable | Rendered |
|---|---|---|---|---|
| `live` | Through its active sources | Yes | Yes | Yes |
| `retired` | No; the collection shows it as no longer obtainable | Yes | Yes | Yes |
| `disabled` | No | Yes, in inventory | No | No |

- `disabled` is the switch-off in `docs/localization.md` 6.1: it removes the item and its static fallback from display. The server unequips it and equips the first `live` `default` item of the same type in manifest order; other players see that item too.
- An item with a `default` source is always `live` and static.
- An item that was ever published is never removed from the manifest; it moves to `retired` or `disabled`. Its id stays reserved.

## 7. Manifest

### 7.1 Format

```json
{
  "$schema": "../schema.json",
  "schemaVersion": 1,
  "manifestVersion": 7,
  "items": [ { "id": "avatar.starter", "...": "..." } ]
}
```

| Field | Rule |
|---|---|
| `$schema` | Optional, editor hint only |
| `schemaVersion` | Version of the contract in `docs/cosmetics/schema.json`. Currently `1` |
| `manifestVersion` | Increases by 1 on every admin publish (ADR 0006). A rollback is a new publish with earlier content |
| `items` | Every item ever published, ids unique |

### 7.2 Schema changes

- Admin and server validate strictly: an unknown field, type or enum value fails the publish.
- The client is lenient so older builds survive additive changes: it ignores unknown fields and skips items whose `type`, or sources whose `type`, it does not know.
- An additive change (new optional field, new type, new source type) edits the schema and this document without changing `schemaVersion`.
- A breaking change (removing or renaming a field, changing a field's meaning, making a field required) increments `schemaVersion` and needs a minimum client version that understands it. The client keeps its current manifest when it sees a `schemaVersion` it does not know.
- Exception, 2026-11-12: removing the `character` and `action` types with the `clips`, `modelAddress` and `modelHqAddress` fields keeps `schemaVersion` 1. No item of those types was published outside development and no released client reads them. A development database whose cosmetics snapshot still lists such items fails the new checks (V-1, V-8); reset its `cosmetics` config domain and publish the examples again (`tools/dev/publish-configs.sh`). A stored loadout that still names a `character` or `actions` is read without them.

### 7.3 Delivery

| Item | Rule |
|---|---|
| Published file | One JSON file per `manifestVersion`, immutable and content-hashed on the CDN (ADR 0002). The admin publishes it after validation; the server reads the same version from its config snapshot (ADR 0006) |
| File | `<content_base_url>/cosmetics/<hash>.json`, where `content_base_url` is the CDN root of `docs/content-delivery.md` section 2 and `<hash>` the lowercase SHA-256 hex of the file. The admin API uploads it in the same step as every publish, rollback and switch-off; a manifest the content store does not take is not published |
| Active version | Sent by the server after the handshake (ADR 0004) as `ServerHello.cosmetics_manifest`: `manifestVersion` and `hash` of the newest `cosmetics` snapshot, absent before the first publish. URLs come from server config |
| Built-in manifest | The install package ships a manifest with at least every `default` item, so a first or offline start works without a download |
| Update | Downloaded in the background and swapped in at the next screen load, never mid-match. A download, parse or validation failure keeps the previous manifest and is reported to telemetry |
| Match record | Each match records the manifest version in use, with the config version (ADR 0006) |

## 8. Addressables

### 8.1 Addresses

Every asset address is derived from the item id; the manifest stores the full string so the client loads it without building strings at runtime.

| Asset | Address |
|---|---|
| Static sprite | `cosmetic/<type>/<slug>/static` |
| Animation (avatar, frame, tile, emoji) | `cosmetic/<type>/<slug>/anim` |
| Table layer | `cosmetic/table/<slug>/layer/<layer name>` |

`<type>/<slug>` is the `id` with the dot replaced by `/`. Any other address is invalid.

### 8.2 Groups and labels

| Item | Rule |
|---|---|
| Group | One Addressables group per item, `cosmetic-<type>-<slug>`, holding the static sprite and every animation asset of the item (ADR 0002). Bundle mode packs each entry separately, so the static sprite downloads on its own and before the animation |
| Atlas | The frames of one loop (the animation, or one table layer) share one atlas inside the item's group. The static sprite is not packed, so it downloads on its own (`docs/asset-workflow.md` 5.4) |
| Built-in group | Every asset of a `default` item is in the install-time group `cosmetic-builtin` instead of an item group, so offline start and fallbacks render without a download. Default items are static (V-19), so the group holds one static sprite per default item and stays within the install size budget (`docs/performance.md`) |
| Labels | `cosmetic`, `cosmetic-<type>`, and `cosmetic-static` on static sprites. Which labels are prefetched and when is #68's decision |
| Source art | Lives in `assets-src/` and never ships directly |

### 8.3 Static or animated

The client plays an item's animation only when all of these hold; otherwise it shows `staticAddress`:
1. `animated` is true.
2. The current effect level allows animated cosmetics (`animatedCosmetics` in `docs/design-tokens.json`; false on the low level).
3. The animation assets are loaded. A missing or failed download is a static, not a wait.
4. The screen's active animation budget is not exhausted (`maxAnimatedCosmetics` in `docs/design-tokens.json`, ADR 0012).
5. Reduce motion is off.

Until `staticAddress` itself is loaded, the client shows the built-in static of the first `live` `default` item of the same type. Input never waits for a cosmetic to load.

Client (`UI/Cosmetics`, #94):

- `AvatarView` shows an avatar, its frame and the rarity marks a prefab gives it (`docs/rarity.md` 2.3). A prefab for a seat or profile has a halo and no diamond; the caller passes the frame's tier. An item card has both and passes the item's tier. The avatar, the frame and the halo are children of the view, and the view fits all three into one square centred in its box, with the side of the box's shorter edge, again whenever the box is resized. The side is capped so the frame never covers more screen pixels than its sprite has (`docs/art-direction.md` 5.2): the larger side of the frame sprite shown, divided by the root canvas's scale factor, again whenever the frame shows a new sprite. The avatar and frame images keep their sprite's aspect, so a frame is never stretched or cut at any avatar size or box shape (#96), and the halo's glow scale counts from the frame's square. The size of the box comes with the screen's layout.
- `AvatarLook.Of(manifest, avatarId, frameId)` builds what a seat or profile shows from the equipped ids, such as a roster entry's `avatar` and `frame`. Each id gives its item's art when the manifest has it with that type and it is rendered (`live` or `retired`, section 6). Otherwise, also for an empty or unknown id, the first `live` `default` item of the type takes its place, as the server does for a `disabled` item. The halo's tier is the `rarityId` of the frame shown. Without such a default the view shows its built-in art and no mark. `CosmeticManifest.ShownOf(type, id)` is that choice for one item.
- In the match each seat's `AvatarView` gets its look from `SeatAvatars` (`docs/ui-motion.md`, Match layout). After an effect change the screen calls its `RefreshEffects`, which refreshes every seat.
- `CosmeticImage` runs the rules above for one layer. It loads the static when shown, and the loop only after the static, only if the effect level, reduce motion and the budget allow it. A row reused while a load is pending ignores that art and frees it.
- A playing avatar and a playing frame each take one slot of the screen's `AnimationBudget`.
- `CosmeticManifest.TryParse` reads a manifest by the client rules of 7.2. It refuses one with an unknown `schemaVersion`, unreadable JSON or no items, so the caller keeps its current manifest. It skips an item of an unknown type, without an `id` or `staticAddress`, or with an `id` already read, and ignores unknown fields. Each item gives its `CosmeticArt` (no loop unless `animated` with an `animationAddress`), and a table its `TableLook` with `layers` back to front. `DefaultOf(type)` is the first `live` `default` item of a type, the built-in fallback above. The built-in manifest of 7.3 is `UI/Cosmetics/Resources/cosmetics-builtin.json`, loaded by `CosmeticManifest.LoadBuiltin`, a copy of the server's `server/internal/cosmetics/builtin.json`; `CosmeticManifestTests` fails when they differ, and the CI `client` job runs when either changes.
- `TableView` shows a table (#97). Its images are `TableImage`s, so every layer is drawn at the screen's shape by the rule of 8.4. It shows the built-in table until `staticAddress` is loaded, then the flattened static. An animated table loads its layers only after the static, and only when the rules above allow animation, the view has a layer image for every layer and the budget has a slot for every animated layer. It switches to the composed layers, back to front, only once every layer has loaded, so a half-built table is never shown. A failed layer, a spent budget or a drop to the low level or reduce motion returns it to the static and frees the layers. Each playing layer takes one slot. In the match the `TableLook` comes from `MatchView.table` through `Game/Match/TableBackground` (`docs/ui-motion.md`, Match layout).
- `CosmeticCollection` is the model of the collection screen (#99). A tab is one type; it lists that type's `live` and `retired` items in manifest order and never a `disabled` one, even when held (section 6). An optional rarity filter keeps one tier. Held and equipped come only from the server: `Show` takes the `GET /inventory` reply (every held item, the `default` items included, the resolved loadout and `emoji_slots`, the `EmojiSlots` count of #150) and `Apply` the loadout of a `PUT /inventory/equip` reply. Until the first reply nothing is held. A `retired` item is marked as no longer obtainable. `TryEquip` gives the equip intent `{ "slot": <type>, "items": [<id>] }` only for a held item that is not worn; the entry stays as it is until the server's loadout arrives (section 10). Emoji are equipped as one list of slots: for a held emoji `TryEquip` gives `{ "slot": "emoji", "items": <the worn list> }` with the emoji appended when it is not worn and fewer than `emoji_slots` are worn, or removed when it is worn; with every slot taken an emoji that is not worn gives no intent. The server still enforces the slot count. The earn path of an entry (`EarnPaths`) is the source types of its item (section 5) in manifest order, or none when the item is held or `retired`. `CosmeticManifest` keeps those types on each item as `Sources`, skipping a source type the client does not know (7.2); the refs stay out, since the referenced record decides whether a source is active now. The text shown per source type and the sort order inside a tab, beyond manifest order, are not decided yet.
- `Game/Inventory/LoadoutEquipper` sends the intent `TryEquip` built as `PUT /inventory/equip`, one at a time, and hands the server's `{"loadout"}` reply to `CosmeticCollection.Apply`. A refusal leaves the shown loadout unchanged and names its text: `403 item.not_owned` `inventory.equip.not_owned`, `409 item.disabled` and `404 item.unknown` `inventory.equip.unavailable`, `400 slot.too_many` `inventory.equip.slots_full`, no connection and rate limit their own keys; `409 profile.setup_required` and an expired session show no text, since the screen sends the player to profile setup or sign-in. Any other refusal, such as `slot.unknown` or `item.wrong_type`, which a correct client never causes, shows `notice.generic`. After `item.not_owned`, `item.disabled` or `item.unknown` (`InventoryStale`) the screen reads `GET /inventory` again, since the server's inventory is the only one that counts. `InventoryLoader` reads `GET /inventory`; its offline and rate limit texts are `inventory.load.offline` and `inventory.load.rate_limited`.
- **Delivery (7.3).** `ContentBoot.Cosmetics` (`ActiveCosmeticManifest`) starts from the built-in manifest. An accepted hello with `cosmetics_manifest` goes from `AppBoot` to `Content/CosmeticManifestUpdater`, which downloads `ContentUrls.Cosmetics` under the hello's `content_base_url` unless that hash is already held, staged or in flight. `Stage` accepts the file only when its SHA-256 hex is the announced hash, `TryParse` reads it and its `manifestVersion` is the announced one; a failed request is logged as `Cosmetics fetch failed version=...`, a refused file as `Cosmetics fetch refused version=...`, and the previous manifest stays until the next hello. An accepted file is kept as `content/cosmetics/<version>_<hash>.json`, one file that replaces the previous one. `ApplyStaged` swaps it in at the next app state change outside the match (`AppBoot`), never mid-match, and raises `Changed`; a screen reads `Current` when it opens. Before the first scene the kept file is staged and applied at once; a damaged one is logged as `Cosmetics stored refused version=...` and the built-in stays. The editor neither keeps nor downloads it. `CosmeticManifestUpdaterTests` cover it with a fake download over real files in a temporary folder.
- `CosmeticArtCache` shares one load between all views of an item on a screen and frees the art on the last release. It is memory only; the disk cache belongs to Addressables and #160.
- `ICosmeticArtSource` is where art comes from. The Addressables source is #68.
- `CosmeticStage` holds one screen's sources, budget and effect level. A screen keeps it in step with the settings through a `CosmeticEffectsLink`, given the stage, `GraphicsQuality`, `MotionPreference` and the screen's refresh (such as `MatchTable.RefreshEffects`), and disposed with the screen. The link reads `GraphicsQuality.Effects` (`docs/performance.md` 1.1) and `MotionPreference.ReduceMotion` when it is made and on each of their `Changed` events, a heat step included. When either differs from the stage, it calls `SetEffects` and then the refresh, so a drop to the low level or reduce motion turns every avatar, frame and table on the screen static at once. A change that leaves both the same, such as high to ultra, leaves the stage and its budget alone and refreshes nothing.

### 8.4 Table at any aspect

Owner decision 2026-11-16: a table's rim, gold inlay and carpet border show whole on every screen shape of `docs/ui-motion.md` (16:9 to 21:9 phones in either orientation, 4:3 tablets, nearly square displays). One set of sprites serves every shape; no table has art per aspect.

- Each table item exports `export/slice.json`, `{ "depth": <px>, "step": <px> }` in pixels of its static (`docs/asset-workflow.md` 4.2). `depth` is the ring: the rim, the inlay, the carpet border and its guards, the same on all four sides. `step` is one repeat of the top and bottom border; the art makes those two edges repeat with that step between their corner pieces (`docs/art-direction.md` 5.4).
- `CosmeticContent` reads it on import and sets the border of the static and of every layer frame to (left, bottom, right, top) = ((width - step) / 2, depth, (width - step) / 2, depth): the fixed parts, and the centred run of one step that the long edges repeat. A table without the file is imported without a border.
- `TableImage` (`UI/Cosmetics`, an `Image`) draws such a sprite into its rect. The sprite's height fills the rect's short side; a rect taller than wide shows the table turned a quarter turn, so the long edges stay long. The top and bottom bands are drawn as two runs from the corners that meet in the middle, a whole number of steps more or fewer than the art holds, the one closest to the rect's width; what is left is a uniform horizontal scale of the whole table, within half a step. The side bands keep their art. The field inside the ring is drawn whole-scaled to cover its place, centred, cropped where it overhangs, and fades in over the inner 16/1080 of the ring height, so its edge meets the ring without a seam. A sprite without a border is drawn covering the rect, centred.
- `TableCloth` is that layout as plain numbers (15 quads at most), so it is tested without a scene. Building the mesh allocates nothing, so a layer that changes frame allocates nothing per frame.
- A layer frame shares the static's canvas and slice, so the layers of an animated table line up at every shape. The middle of each long edge is the only place where art from two runs meets; the art keeps it a repeat of the edge, and motion near it repeats or meets in the same way (`docs/art-direction.md` 5.4).

## 9. Validation

Single-manifest rules, checked by the admin at publish, by the server when it loads a snapshot, and by `tools/cosmetics-schema`:

| Id | Rule |
|---|---|
| V-1 | The manifest conforms to `docs/cosmetics/schema.json`, including: required shared fields, no undeclared field (a price on an item or source is invalid), known types and source types, `culturalReview` is `approved`, `schemaVersion` is known |
| V-2 | Every item has `staticAddress`. An animated item without a static fallback is invalid |
| V-3 | Animation fields match `animated` and `type` (section 4): required when animated, forbidden when static, forbidden on other types |
| V-4 | Removed 2026-11-12 with the `character` type (it required the clips and model of an animated character) |
| V-5 | An animated table has unique layer names and at least one animated layer |
| V-6 | An emoji has a `group`; other types do not |
| V-7 | Sources are non-empty and unique; `default` has no `ref`, every other source has one |
| V-8 | Ids are unique and their prefix equals `type` |
| V-9 | `nameKey` is `cosmetic.<id>.name` |
| V-10 | Every address follows section 8.1 for its item and slot |
| V-11 | An item with a `default` source is `live`, and every type has at least one such item |
| V-19 | An item with a `default` source is static: `animated` is false |

Rules that need more than one manifest or other admin data, checked by the admin at publish only:

| Id | Rule |
|---|---|
| V-12 | No item of the previous manifest is missing; an id keeps its `type` |
| V-13 | `revision` never decreases and increases on every change to the item |
| V-14 | `rarityId` exists in the rarity table (`docs/rarity.md` section 7) |
| V-15 | Every `ref` resolves to an existing record of its source type, and every record that grants an item is listed in that item's `sources` (section 5) |
| V-16 | `nameKey` has an approved `en` value (`docs/localization.md` 2.1) |
| V-17 | Every address resolves to an asset in the Addressables build of the target catalog version |
| V-18 | A table's layer count is at most `TableMaxLayers` |

An invalid manifest is never published. A client that receives one keeps its previous manifest (7.3).

Implemented in `server/internal/cosmetics` (#136):
- **Publish check.** A publish must follow the previous manifest. Its `manifestVersion` is exactly one higher (7.1). V-12 holds: every earlier item is still present and keeps its type. V-13 holds: no revision goes down, and an item whose content changed has a higher revision. Content means every field except `revision`.
- **Rollback** (`docs/admin.md` 3.7). It builds the next manifest from the current one. An item whose content differs from the target version gets the target's content under the next revision. Unchanged items, and items added after the target, stay as they are. The result passes the publish check.
- **Delivery to the game server.** The manifest is the `cosmetics` config domain. `config-publish -domain cosmetics` runs the single-manifest checks and then the publish check against the newest published manifest. Every node polls the domain (`HD_CONFIG_POLL`) and switches its inventory to the new catalog without a restart. A snapshot that fails the checks is logged, and the previous catalog stays.
- **Admin publish** (`docs/admin.md`, Cosmetics manifest note). `cosmetics.ValidateManifest` adds the strict schema, V-2, V-3, V-5, V-6, V-9, V-10 and V-14 at the admin API; a game server still loads with the lenient `Parse` (7.2). V-15 holds from record to item: a publish that drops a source a published record grants from is refused. V-16 holds: a publish or rollback is refused while any item lacks an `approved` `en` value for its `nameKey` in the game strings, and `detail` lists those items; an incident switch-off (`docs/admin.md` 5.1) skips V-16. The static fallback of V-2 must also exist as art: a publish that adds an item is refused (`cosmetic_art_missing`) until the item's `static` upload passed its checks (`docs/admin.md` 5.2).
- **Not yet.** V-15 from `ref` to record: an item lists its source before the record exists, so this needs one change set. V-17 waits for the asset pipeline (#135), and V-18 for the `TableMaxLayers` value (#97).

## 10. Server authority

- The manifest describes what exists. It holds no ownership, price, amount, chance or time window.
- Ownership is the server inventory. Items are granted only through the reward bundle engine (#102) from a listed source (section 5).
- Equip is an intent: the server accepts it only for an owned item of that slot's type whose availability is `live` or `retired`. Sending an emoji requires owning it (#150).
- The client never unlocks, previews as owned or equips locally ahead of the server. A modified client that edits its manifest only changes what it draws for itself.
- Implemented in `server/internal/cosmetics` and `server/internal/inventory` (#61): a `default` item is owned by every account implicitly, with no inventory row, so it can never be revoked; every other item is a row with its granting source, accepted only when listed in `sources`. Equip checks type, availability and ownership; the loadout is resolved again at match start and recorded with the match (`match_participants.loadout`). The manifest is the newest `cosmetics` config snapshot (section 9, ADR 0006). Until one is published, it is `HD_COSMETICS_MANIFEST` or the built-in copy of `examples/minimal.json`.

## 11. One schema for admin, server and client

`docs/cosmetics/schema.json` is the only definition of the item shape. The admin catalog forms (#136, #137) validate against it, the admin API and the game server validate a publish against it plus section 9, and the client manifest types (#68) are written or generated from it. A field added in one place and not in the schema is a defect.
