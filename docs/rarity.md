# Rarity

Status: proposal. The `on` shade follows the ink of the 2026-11-12 art direction (`docs/art-direction.md`). Every hex value, glow level, size and period here is an initial proposal; the owner approves them by merging the pull request that adds or changes them. Tiers, order and the colour convention are owner decisions of 2026-10-26.

Rarity tiers, their order, colours, the status diamond, the rarity glow, the comparison rule and the table background selection at match start. `docs/design-tokens.json` group `rarity` holds the values; this file explains them. `node tools/design-tokens/design-tokens.mjs` checks the group (ids, ordinals, pips, glow order, contrast) and that the contrast table and colours below match it; CI runs it through `npm run test:tools`.

Items refer to a tier by `rarityId` (`docs/cosmetics.md` section 3). Glow levels, motion rules and effect levels come from `docs/ui-motion.md`; frame budgets from `docs/performance.md`; admin config from ADR 0006.

## 1. Tiers

| `rarityId` | Ordinal | Name key (`en` reference) | `fill` | `on` | `text` | Pips | Glow | Flow |
|---|---|---|---|---|---|---|---|---|
| `common` | 1 | `rarity.common.name` (Common) | `#A3A6BF` | `#0B0605` | `#C9CBDB` | 1 | `glow.none` | No |
| `rare` | 2 | `rarity.rare.name` (Rare) | `#5C78FF` | `#0B0605` | `#A3B3FF` | 2 | `glow.soft` | No |
| `epic` | 3 | `rarity.epic.name` (Epic) | `#B45CFF` | `#0B0605` | `#D3A3FF` | 3 | `glow.strong` | No |
| `legendary` | 4 | `rarity.legendary.name` (Legendary) | `#FFC23D` | `#0B0605` | `#FFD98A` | 4 | `glow.intense` | No |
| `mythic` | 5 | `rarity.mythic.name` (Mythic) | `#FF4530` | `#0B0605` | `#FF9C8F` | 5 | `glow.intense` | Yes |

- Hues follow the genre convention: grey, blue, purple, gold, red. `fill` is the diamond and the glow tint, `on` the pips on the diamond, `text` the tier label on a dark layer, as with the roles in `docs/ui-motion.md` section 2.
- The status colour of an item is its tier's `fill`. The manifest has no colour field.
- Name values in other languages are managed in admin (`docs/localization.md` section 2.1).
- `rare` (hue 230) sits next to `role.teamA` (216), `legendary` (41) next to `role.warning` (39) and `role.score` (43), `mythic` (6) next to `role.danger` (354). Colour is never the only cue: tiers differ by pip pattern, and in a match rarity shows only as a halo behind a seat frame, while team identity comes from seat position and badge, outcomes carry an icon and score gold appears only in score numerals and score effects. `legendary` gold also sits next to the gold ornament of the aristocratic look (`docs/art-direction.md` section 2); a tier still reads by its pip pattern, and the small diamond gems of the score panel carry no pips and show no rarity.

### 1.1 Tier of an item

Owner decisions of 2026-11-16 for the tier an item's admin draft proposes. The admin still sets the tier of each item; these rules say which tier a draft proposes.

- A static avatar is `common` or `rare`. `epic`, `legendary` and `mythic` avatars are animated.
- An animated item's tier rises with its animated detail: the number of independent moving elements its loop script authors, read from `work/*_loop.py` (or the table script) and the item's `provenance.md`. `tile.sema` keeps its tier and is not counted.

| Class | One animated detail | Not counted |
|---|---|---|
| Avatar | Each extra moving part: one part (fan, feather, tassel, single earring, page curl), one matching pair (earrings, hoops, tassels, ribbon ends, a boa on both shoulders), one cloth or chain ripple, one group of stone glints, a full spin. A creature with several moving parts (bird, kitten, puppy, butterfly) counts 2 | The base motion every avatar has: breathing, head tilt, blink |
| Frame | Each moving ornament group (gear rims, plumes, tassels, scrolls, wings, wave band, billows, orbiting stones) and each ambient type (stone glints, rising sparks, orbiting sparkles, petals, drops, bubbles, bolts) | Runtime sheen and aura |
| Table | Each moving element type of the motion layer: tassels, fringe, gold-thread light bands, knot twinkle, dust motes | The still back, middle and front layers |
| Emoji | Each moving part of the loop script's `PARTS` (a heart, a tear, a star, a note, a hat, a flake) | The whole-medallion bounce, sway, tilt or tremble |

| Class | `rare` | `epic` | `legendary` | `mythic` |
|---|---|---|---|---|
| Avatar | — | 1 | 2 | 3 or more |
| Frame | — | 1 | 2 | 3 or more |
| Table | — | 1 | 2 | 3 or more |
| Emoji | 0 | 1 or 2 | 3 | 4 or more |

- An emoji is the cheapest class (`GuidePriceTypePercent` 50, `docs/economy.md` 3.1) and three animated emoji move only as a whole medallion, so animated emoji start at `rare`.
- Set 1 under these thresholds: avatars 21 `epic`, 6 `legendary`, 3 `mythic`; the eight animated shop frames 7 `legendary`, 1 `mythic`; tables 3 `epic`, 1 `legendary`, 2 `mythic`; emoji 3 `rare`, 10 `epic`, 2 `legendary`, 1 `mythic`. Each item's count and tier is in its `provenance.md`.
- Exception, owner decision of 2026-11-16: the ten season rank frames (`frame.season1_*`, `docs/ranking.md` section 3) keep the tier ladder, so their tier rises with the rank and not with their animated detail: `season1_iron` and `season1_bronze` `common`; `season1_silver`, `season1_gold` and `season1_platinum` `rare`; `season1_emerald` and `season1_diamond` `epic`; `season1_master` and `season1_grandmaster` `legendary`; `season1_challenger` `mythic`.

## 2. Marks

### 2.1 Status diamond

A square standing on one corner. Its pips use the layout of a domino half (`docs/art-direction.md` section 3: a value reads by pattern), rotated with the square, so tiers never rely on colour alone and read in greyscale:

| Pips | Layout in the diamond |
|---|---|
| 1 | Centre |
| 2 | Top and bottom corner |
| 3 | Vertical line: top, centre, bottom |
| 4 | All four corners |
| 5 | Four corners and centre |
| 6 | Left and right column of three (reserved for a sixth tier) |

- Two white sprites from one atlas: the diamond tinted with `fill`, the pip overlay tinted with `on`, both through vertex colour. One material, so the two batch together.
- Bounding box at least `rarity.diamondMinSize` (72 units, about 4.5 mm on a 6.5-inch phone). Where it does not fit, no diamond is drawn; there is no smaller variant with fewer pips.
- The tier label is the localized name in `caption` style and the tier's `text` shade, next to the diamond, wherever the layout has room (shop and collection detail, reward reveal).
- Nothing about rarity is painted into an asset (`docs/art-direction.md` section 8).

### 2.2 Glow and flow

- The rarity glow is the runtime halo of `docs/ui-motion.md` section 4: one additive sprite from the shared effects atlas, tinted with `fill`, alpha and scale from the tier's `glow` level, one overdraw layer. `glow.none` draws no sprite.
- A tier with flow uses the flow halo instead: the same halo with one bright arc, rotated 360 degrees every `rarity.flowPeriodMs` (2400 ms, `linear`, looping) by a transform tween. Transform only, no material property, no particles.
- Flow stops at the rest angle (arc at the top) when the effect level has `animatedCosmetics` false (low) or reduce motion is on. The halo, its tint and level stay; only the rotation is dropped.

### 2.2.1 Client

- `RarityTable` holds the tiers by id. Ids are data, not an enum, because the table is admin config and may grow to a sixth tier. An invalid tier is left out, so it draws no mark. A tier is invalid when it breaks a per-tier rule of section 7: the id pattern, an `ordinal` of at least 1, `nameKey` `rarity.<id>.name`, 1 to 6 `pips`, and `fill`, `on` and `text` as `#RRGGBB`. Tiers that share an id, an `ordinal` or a pip count are all left out, whatever their order, so no two drawn tiers compare equal or show the same pip pattern. The client does not check the table-wide rules (ordinals without gaps, glow order, contrast); admin rejects such a publish.
- `UI/Rarity/rarity-defaults.json` is the built-in table. `npm run tokens:gen` generates it from `docs/design-tokens.json`, and a test fails if it is stale. The active table from config (section 7) has the same shape; delivery waits for the client config (#145).
- `RarityDiamond` draws the diamond and picks the pip overlay by tier, and hides itself when its box is under `diamondMinSize`, also after a resize.
- `RarityLabel` draws the tier label: the tier's `nameKey` through `GameStrings` in the tier's `text` shade, on a `TMP_Text` that the screen's prefab gives the `caption` style with `TokenText`. It formats the name again on `GameStrings.Changed`, so a language change or an applied bundle reaches an open screen, and draws nothing for a tier missing from the table.
- `RarityHalo` draws the halo or flow halo with the tier's tint, alpha and scale. Around an avatar, `AvatarView` gives it the frame's square and centre, so the scale counts from the frame (`docs/cosmetics.md` 8.3). It turns the flow halo in `Update` by transform only, with no allocation. The tween pool of `docs/ui-motion.md` section 6 does not exist in the client yet, so this turn is not counted against `maxTweens`.
- A seat or profile avatar takes its tier from the frame it shows: `AvatarLook.Of` resolves it from the manifest (`docs/cosmetics.md` 8.3), so a frame that cannot be shown and gives way to the default frame shows the default frame's tier.
- The halo and flow halo sprites (`halo`, `flow_halo`) and the additive material are in the shared effects atlas (`docs/vfx.md`, #176); the diamond and pip overlays are the UI kit of #168. A screen sets them on its prefab.

### 2.3 Where marks show

| Context | Diamond | Label | Glow |
|---|---|---|---|
| Shop, collection, inventory item card | Yes | Detail view only | Behind the item preview |
| Reward reveal | Yes | Yes | Behind the item |
| Profile and lobby avatar | No | No | Around the frame, from the frame's tier |
| Match seat | No | No | Around the frame, from the frame's tier |
| Match table background | No | No | No; the table art is the whole presentation |

### 2.4 Visual samples

| Tier | Item card on `color.surface.default` |
|---|---|
| `common` | Cool grey diamond with one dark pip in the centre; no halo |
| `rare` | Blue diamond with two pips, top and bottom; faint blue halo (`glow.soft`) |
| `epic` | Purple diamond with a vertical line of three pips; clear purple halo (`glow.strong`) |
| `legendary` | Gold diamond with four corner pips; bright gold halo (`glow.intense`) |
| `mythic` | Red diamond with four corner pips and a centre pip; bright red halo whose bright arc circles the item every 2.4 s; at the low level or with reduce motion the arc rests at the top |

The reference board panel that shows these on the real ground is listed in section 8.

## 3. Contrast

Same thresholds as `docs/ui-motion.md` section 2: `fill` is non-text UI and needs 3:1 on every ground layer (SC 1.4.11), `text` needs 4.5:1 on every ground layer and `on` 4.5:1 on `fill` (SC 1.4.3). Ratios are truncated to two decimals. Generated by `node tools/design-tokens/design-tokens.mjs --rarity-table`; the test fails if this table is stale.

| Foreground | `background.base` | `background.raised` | `surface.default` | `surface.raised` | `on` on `fill` |
|---|---|---|---|---|---|
| `rarity.tiers.common.fill` | 8.40 | 7.96 | 7.17 | 5.91 | 8.40 |
| `rarity.tiers.common.text` | 12.51 | 11.85 | 10.67 | 8.80 | - |
| `rarity.tiers.rare.fill` | 5.35 | 5.07 | 4.57 | 3.76 | 5.35 |
| `rarity.tiers.rare.text` | 10.02 | 9.49 | 8.55 | 7.05 | - |
| `rarity.tiers.epic.fill` | 5.67 | 5.37 | 4.84 | 3.99 | 5.67 |
| `rarity.tiers.epic.text` | 10.02 | 9.49 | 8.55 | 7.05 | - |
| `rarity.tiers.legendary.fill` | 12.50 | 11.84 | 10.66 | 8.79 | 12.50 |
| `rarity.tiers.legendary.text` | 14.91 | 14.11 | 12.72 | 10.48 | - |
| `rarity.tiers.mythic.fill` | 5.89 | 5.58 | 5.03 | 4.14 | 5.89 |
| `rarity.tiers.mythic.text` | 9.98 | 9.45 | 8.51 | 7.02 | - |

## 4. Comparison

- Tiers are ordered by ordinal only: tier A is rarer than tier B if and only if `ordinal(A) > ordinal(B)`. Equal ordinals mean the same tier.
- Ids, names, colours and pip counts carry no order. Code compares ordinals, never ids or strings.
- Ordinals are the integers 1 to N without gaps or repeats, N being the number of tiers.

## 5. Table background selection

At match start the server picks the one table background every client of the match shows.

Input:
- Seats `s_0 .. s_(n-1)` in the server's fixed seat index order, `n` 2 (1v1) or 4 (2v2; North, East, South, West as in `docs/rules.md`).
- `t(s)`: the table item equipped by the player in seat `s`, from the server inventory. Every account always has one: a `default` table exists and replaces a `disabled` one (`docs/cosmetics.md` sections 5 and 6).
- `ord(t)`: the ordinal of `t.rarityId` in the rarity table of the config snapshot the match records (ADR 0006).
- `b`: 32 bytes from the server's cryptographically secure random source (as for shuffling, R-4), drawn once per match for this choice only.

Algorithm:
1. `m = max over s of ord(t(s))`.
2. `C` = the seats with `ord(t(s)) = m`, in seat index order; `k = |C|`.
3. If `k = 1`, `i = 0`. Otherwise `U` = the first 8 bytes of `b` read as an unsigned 64-bit big-endian integer, and `i = U mod k`.
4. The match table is `t(C[i])`.

Rules:
- Highest rarity wins. Ties are uniform over seats: each tied seat has probability `1/k` (modulo bias under 2^-62). Two tied seats with the same item give that item probability `2/k`; the result is the same table either way.
- The server computes the choice and sends only the table item id in the match start state; clients render it and never compute it. `b` stays on the server (`docs/anti-cheat.md` NT-4) and is stored in the replay record with the moves (`docs/privacy.md` D-13), so a replay reproduces the choice.
- The table is fixed for the whole match: hands, reconnects and equip changes during the match do not change it.
- Rendering follows `docs/cosmetics.md` section 8.3: animated if allowed and loaded, otherwise its static, otherwise the built-in default table static.
- Implementation and the uniformity test: `server/internal/cosmetics` (`PickTable`, `Catalog.Table`, `NewDraw`) with the worked examples below and a uniformity test over 60 000 draws per tie size (#98). The draw is `replay.Record.TableDraw`, the chosen id is `room.Options.Table` and `MatchView.table`; ordinals come from `HD_RARITY_ORDINALS` until admin config serves the rarity table.

Worked examples (`U` in hex):

| Seats and tiers | `m` | `C` | `U` | Result |
|---|---|---|---|---|
| N `legendary`, E `common`, S `rare`, W `epic` | 4 | N | unused | N's table |
| N `epic`, E `mythic`, S `common`, W `mythic` | 5 | E, W | `0000000000000003` | 3 mod 2 = 1, W's table |
| N, E, S, W all `rare` | 2 | N, E, S, W | `FFFFFFFFFFFFFFFF` | (2^64 - 1) mod 4 = 3, W's table |
| 1v1: both `common` (`table.starter`) | 1 | seat 0, seat 1 | `000000000000000A` | 10 mod 2 = 0, seat 0's table, `table.starter` |

## 6. Cost and effect levels

Rarity adds no effect to `effects.budgets`; seat halos are part of the base scene and are measured with it.

| Item | Budget |
|---|---|
| Halo, any tier other than `glow.none` | 1 sprite, 1 overdraw layer, 0 particles |
| Flow | 1 transform tween from the pool of `maxTweens` (`docs/ui-motion.md` section 6, rule 3) |
| Diamond | 2 sprites in the item card's batch |
| Match scene, all seats | At most one halo per seat (2 or 4), all from the shared effects atlas and material on the seats' animated sub-canvas: 1 draw call, 1 overdraw layer, at most 4 tweens (0 on the low level). Counted in `baseSceneBatches` and base scene CPU and GPU time |

| Effect level / setting | Glow | Flow |
|---|---|---|
| low (`animatedCosmetics` false) | Static halo at the tier's level | Rest angle |
| medium, high | Static halo at the tier's level | Rotates |
| Reduce motion on, any level | Static halo at the tier's level | Rest angle |

The worst case for the replay harness (`docs/performance.md` section 2) is four seats with `mythic` frames. A PR that changes a halo, the flow or the diamond reports the measurements of `docs/ui-motion.md` section 7.

## 7. Admin config

The rarity table is admin config (ADR 0006): published, versioned and rolled back like any config, with `docs/design-tokens.json` holding the defaults. Each match records the config version it used. The client gets the active table with its config and ships the defaults built in, as for the cosmetics manifest (`docs/cosmetics.md` section 7.3).

| Field | Rule |
|---|---|
| Tier id | Key of `rarity.tiers`, `^[a-z][a-z0-9_]*$` (the `rarityId` pattern of `docs/cosmetics/schema.json`). Stable: never renamed, never reused, never removed while an item of any published manifest references it (`docs/cosmetics.md` V-14) |
| `ordinal` | Integers 1 to N without gaps or repeats. Changing them changes table selection for matches started after the publish |
| `nameKey` | `rarity.<id>.name` |
| `fill`, `on`, `text` | `#RRGGBB`, with the section 3 contrast on all four ground layers |
| `pips` | 1 to 6, unique across tiers. A seventh tier needs a new mark design and a spec change |
| `glow` | A level of the `glow` tokens, never weaker than the glow of a lower tier |
| `flow` | Boolean |
| `diamondMinSize` | Positive integer, UI units |
| `flowPeriodMs` | Positive integer, ms |

A publish that breaks a rule is rejected (ADR 0006). The checker in `tools/design-tokens/` enforces the same rules on the defaults. A client that meets a `rarityId` missing from its table draws no rarity mark for that item.

## 8. Server authority and open items

- The client never sends a rarity, ordinal or table choice. Rarity is display data on the client; the server uses it only for table selection.
- Prices, drop chances and rewards are not derived from rarity here; they belong to #7 and #9.
- Open: the reference board panel with the five item cards and the four-mythic-seat mock (`docs/art-direction.md` section 9, #167).
- Reward reveal for pass, mission and medal grants: `docs/pass-missions-medals.md` 6.1. Open: the shop purchase reveal (#7).
