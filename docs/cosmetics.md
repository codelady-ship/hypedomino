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
| Animated asset format and player, active animation budget per screen | #93 | An address; the asset behind it is in the format #93 picks |
| Emoji groups and equipped emoji slot count | #150 | `group` id; slot count is admin config there |
| Download and prefetch policy | #68 | Addresses and labels in section 8 |
| Grant engine and duplicate grants | #102 | Every grant goes through the reward bundle engine |

## 2. Types

| `type` | What it is | Where it shows |
|---|---|---|
| `avatar` | Player portrait | Profile, seat, lists, shop, collection |
| `character` | Full-body figure that reacts to match events | Match seat, loading screen, friend lobby (its clips are the lobby actions, `docs/friend-lobby.md`), profile, shop, collection |
| `frame` | Border around the avatar | Everywhere the avatar shows |
| `table` | Match table background | Match scene, collection, shop |
| `emoji` | Reaction sent by id in a match or a friend lobby (`docs/localization.md` 5.7) | Match seat, friend lobby, emoji wheel, collection |
| `action` | Full-body motion the equipped character plays in the friend lobby, on demand (`docs/friend-lobby.md`) | Friend lobby action wheel, collection, shop |

Adding a type is a schema change (section 7.2) and a change to this table.

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
| `avatar`, `frame`, `emoji` | `animationAddress` | Required if `animated`, forbidden otherwise | One looping animation |
| `character` | `clips` | Required if `animated`, forbidden otherwise | Object with exactly `idle`, `win`, `lose`, `reaction`, each an address of an animation clip on the shared humanoid skeleton (ADR 0009). A static character shows `staticAddress` in every state |
| `character` | `modelAddress` | Required if `animated`, forbidden otherwise | The real-time 3D model at medium detail: seat level of detail and base colour texture (ADR 0009) |
| `character` | `modelHqAddress` | Optional if `animated`, forbidden otherwise | The high-detail model: hero level of detail, larger textures and normal map. Without it the high level shows `modelAddress` |
| `table` | `layers` | Required if `animated`, forbidden otherwise | Back-to-front list of `{ name, address, animated }`, at least 1, names unique, at least one layer animated. Count at most `TableMaxLayers` (admin config; its value comes from the measured base-scene cost in #97, ADR 0008) |
| `emoji` | `group` | Required on emoji, forbidden on other types | Emoji group id (#150) |
| `action` | `animated` | Always `true` | An action is motion by definition; its `staticAddress` is its icon, shown in the action wheel and played as a pop-up over a static character |
| `action` | `animationAddress` | Required | One clip on the shared humanoid skeleton (ADR 0009), so every character can play every action |

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
| `rank_reward` | Ranked season reward bundle id (`RankRewards` in the `rating` config) | #8, #120 |
| `event` | Live event id | Admin live-ops config |

Rules:
- `sources` is the complete list of ways an item can be granted. The admin rejects publishing an offer, reward, mission, campaign or event that grants an item whose `sources` does not list it, and the server's grant path (#102) refuses such a grant. This is what the collection screen shows as the way to earn an item (#99) and what abuse checks (#211) compare against.
- A source listed on an item does not mean it is active now; whether a shop offer is on sale or an event is running is decided by the referenced record and server time.
- Every type except `action` has at least one `live` item with a `default` source, so a new account has something equipped in every slot and a disabled item has a replacement (section 6). An `action` never has a `default` source: it is always animated, and the friend lobby already offers the equipped character's own clips without any owned action.
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

### 7.3 Delivery

| Item | Rule |
|---|---|
| Published file | One JSON file per `manifestVersion`, immutable and content-hashed on the CDN (ADR 0002). The admin publishes it after validation; the server reads the same version from its config snapshot (ADR 0006) |
| Active version | Sent by the server after the handshake (ADR 0004). URLs come from server config |
| Built-in manifest | The install package ships a manifest with at least every `default` item, so a first or offline start works without a download |
| Update | Downloaded in the background and swapped in at the next screen load, never mid-match. A download, parse or validation failure keeps the previous manifest and is reported to telemetry |
| Match record | Each match records the manifest version in use, with the config version (ADR 0006) |

## 8. Addressables

### 8.1 Addresses

Every asset address is derived from the item id; the manifest stores the full string so the client loads it without building strings at runtime.

| Asset | Address |
|---|---|
| Static sprite | `cosmetic/<type>/<slug>/static` |
| Animation (avatar, frame, emoji), action clip | `cosmetic/<type>/<slug>/anim` |
| Character clip | `cosmetic/character/<slug>/clip/<idle\|win\|lose\|reaction>` |
| Character model | `cosmetic/character/<slug>/model` |
| Character high-detail model | `cosmetic/character/<slug>/model_hq` |
| Table layer | `cosmetic/table/<slug>/layer/<layer name>` |

`<type>/<slug>` is the `id` with the dot replaced by `/`. Any other address is invalid.

### 8.2 Groups and labels

| Item | Rule |
|---|---|
| Group | One Addressables group per item, `cosmetic-<type>-<slug>`, holding the static sprite and every animation asset of the item (ADR 0002). Bundle mode packs each entry separately, so the static sprite downloads on its own and before the animation |
| Atlas | Sprites of one item share one atlas inside its group |
| Built-in group | Every asset of a `default` item is in the install-time group `cosmetic-builtin` instead of an item group, so offline start and fallbacks render without a download. Default items are static (V-19), so the group holds one static sprite per default item and stays within the install size budget (`docs/performance.md`) |
| Labels | `cosmetic`, `cosmetic-<type>`, and `cosmetic-static` on static sprites. Which labels are prefetched and when is #68's decision |
| Source art | Lives in `assets-src/` and never ships directly |

### 8.3 Static or animated

The client plays an item's animation only when all of these hold; otherwise it shows `staticAddress`:
1. `animated` is true.
2. The current effect level allows animated cosmetics (`animatedCosmetics` in `docs/design-tokens.json`; false on the low level).
3. The animation assets are loaded. A missing or failed download is a static, not a wait.
4. The screen's active animation budget from #93 is not exhausted.

When a character plays animated, the high effect level loads `modelHqAddress` if the item has one and `modelAddress` otherwise; the medium level loads only `modelAddress`, so a medium device never downloads the high-detail bundle (ADR 0009). The effect level follows the graphics setting (`docs/ui-motion.md` section 7).

Until `staticAddress` itself is loaded, the client shows the built-in static of the first `live` `default` item of the same type. Input never waits for a cosmetic to load.

## 9. Validation

Single-manifest rules, checked by the admin at publish, by the server when it loads a snapshot, and by `tools/cosmetics-schema`:

| Id | Rule |
|---|---|
| V-1 | The manifest conforms to `docs/cosmetics/schema.json`, including: required shared fields, no undeclared field (a price on an item or source is invalid), known types and source types, `culturalReview` is `approved`, `schemaVersion` is known |
| V-2 | Every item has `staticAddress`. An animated item without a static fallback is invalid |
| V-3 | Animation fields match `animated` and `type` (section 4): required when animated, forbidden when static, forbidden on other types |
| V-4 | An animated character has all four clips and `modelAddress` |
| V-5 | An animated table has unique layer names and at least one animated layer |
| V-6 | An emoji has a `group`; other types do not |
| V-7 | Sources are non-empty and unique; `default` has no `ref`, every other source has one |
| V-8 | Ids are unique and their prefix equals `type` |
| V-9 | `nameKey` is `cosmetic.<id>.name` |
| V-10 | Every address follows section 8.1 for its item and slot |
| V-11 | An item with a `default` source is `live`, and every type except `action` has at least one such item; an `action` has no `default` source |
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

## 10. Server authority

- The manifest describes what exists. It holds no ownership, price, amount, chance or time window.
- Ownership is the server inventory. Items are granted only through the reward bundle engine (#102) from a listed source (section 5).
- Equip is an intent: the server accepts it only for an owned item of that slot's type whose availability is `live` or `retired`. Sending an emoji requires owning it (#150).
- The client never unlocks, previews as owned or equips locally ahead of the server. A modified client that edits its manifest only changes what it draws for itself.
- Implemented in `server/internal/cosmetics` and `server/internal/inventory` (#61): a `default` item is owned by every account implicitly, with no inventory row, so it can never be revoked; every other item is a row with its granting source, accepted only when listed in `sources`. Equip checks type, availability and ownership; the loadout is resolved again at match start and recorded with the match (`match_participants.loadout`). The manifest comes from `HD_COSMETICS_MANIFEST` or the built-in copy of `examples/minimal.json` until admin config serves it (ADR 0006).

## 11. One schema for admin, server and client

`docs/cosmetics/schema.json` is the only definition of the item shape. The admin catalog forms (#136, #137) validate against it, the admin API and the game server validate a publish against it plus section 9, and the client manifest types (#68) are written or generated from it. A field added in one place and not in the schema is a defect.
