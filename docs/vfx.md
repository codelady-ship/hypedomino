# VFX library

Status: proposal; the owner approves it by merging. The effect sprites, the shared effects atlas and materials, and how every effect of `docs/ui-motion.md` section 7 is built from them at each effect level (#176). Look comes from `docs/art-direction.md` sections 4 and 6, budgets from `effects.budgets`, `effects.levels`, `scoreCascade.steps` and `glow` in `docs/design-tokens.json`, rarity halos from `docs/rarity.md` 2.2. This file raises no budget and defines no colour, duration or price.

## 1. Rules

- Every effect sprite is white or greyscale and tinted at runtime with a token through vertex colour (art direction section 2). Tints below are token paths; `team` is the winning team's `color.role.teamA.fill` or `color.role.teamB.fill`, `rarity` is the tier's `fill`.
- An effect is designed at the low level first and tells its event there. Medium and high only add particles and sprites: a higher level never has fewer of a sprite than the level below (checked, section 5). Timing and meaning never depend on the level.
- Reduce motion: particles use the low level's counts (`ParticleBudget`), shake and scale pops drop (`docs/ui-motion.md` section 6); sprites that carry information (glow behind a popup, stamp, rays) stay.
- Additive or alpha blending from the two shared materials only (section 3); no per-effect material and no material property animation.
- A particle is any quad an effect spawns more than once, trails included (art direction 6: a trail is part of its effect's particle budget). The glow, ring, rays and stamp are one sprite each and are listed apart from particles.

## 2. Sprites

Made by `assets-src/effect/vfx/work/vfx.py` (provenance, shapes and sizes in `assets-src/effect/vfx/provenance.md`); `sheen` and `tile_shadow` come from #169 (`assets-src/effect/sheen`, `assets-src/effect/tile_shadow`).

| Sprite | Pixels | Material | Used by |
|---|---|---|---|
| `glow` | 128 x 128 | additive | `scoreCascade`, `domino`, `matchWin`, `rankUp`; the primary button's `glow.soft` |
| `halo` | 256 x 256 | additive | Rarity halo |
| `flow_halo` | 256 x 256 | additive | Rarity flow halo |
| `star` | 64 x 64 | additive | Bursts of score, `domino`, `spinnerFormation`, `matchWin`, `rankUp`, `emoji` |
| `spark` | 32 x 32 | additive | Bursts |
| `dust` | 32 x 32 | additive | `tilePlace`, `domino`, `block` |
| `streak` | 128 x 32 | additive | Trails: points to the meter, a travelling tile; `spinnerFormation` arm markers |
| `ring` | 128 x 128 | additive | `spinnerFormation` ring pulse, `block` impact |
| `rays` | 256 x 256 | additive | `rankUp` |
| `stamp` | 256 x 256 | alpha | `block` lock stamp |
| `flake_paper` | 24 x 32 | alpha | `matchWin` confetti |
| `flake_leaf` | 32 x 32 | alpha | `matchWin` gold-leaf confetti |
| `sheen` | 64 x 256 | additive | Sheen sweep across gold rims, frames and skins (art direction 4) |
| `tile_shadow` | 176 x 304 | alpha | Runtime tile shadow, tint `color.shadow` at the `elevation` alpha |

## 3. Atlas, materials and import

- Client folder `client/Assets/HypeDomino/UI/Effects/`: the sprites in `Sprites/` (copies of the `export/` PNGs, same names), `effects.json` (section 4), one atlas `Effects.spriteatlasv2` over `Sprites/`, and the two shared materials: `EffectAdditive.mat` (shader `HypeDomino/UI/Effect Additive`, `EffectAdditive.shader`: `Blend SrcAlpha One`, vertex colour times sprite, with the UI stencil and clip rect so masks and `RectMask2D` work) and `EffectAlpha.mat` (Unity's `UI/Default`). Unity has no additive UI shader, hence the one small shader.
- `HypeDomino/Import Effects` (`HypeDomino.Editor.EffectsContent`; batch `-executeMethod HypeDomino.Editor.EffectsContent.ImportAllAndExit`, also through the `unity-editor.yml` workflow) imports every PNG in `Sprites/` as a single sprite with a tight mesh (less overdraw on round and thin shapes), no mipmaps, clamp, sRGB, alpha as transparency and ASTC with the block of `ProjectSetup.SpriteFormat` on Android and iOS; builds the atlas without rotation or tight packing (a UI `Image` needs both off), padding 4, largest page 1024, ASTC; and creates or updates the two materials. It can be run again after any change. `EffectsContentTests` (EditMode) check those settings on the committed assets.
- After `vfx.py` changes an export, copy it to `Sprites/` under the same name and run the importer.
- Size by the ADR 0012 rule (16 bytes per ASTC 6 x 6 block, summed over the sprites): 172,032 B. With 4 px padding the sprites cover about 409,000 pixels, so the packer is expected to need one 1024 x 512 or 1024 x 1024 page (at most 467,856 B in ASTC 6 x 6). The page is measured in a build with the measurements of `docs/performance.md` section 8.
- The atlas is not an Addressables entry: like the UI kit, it ships in the app through the prefabs that reference its sprites.

## 4. Effects

`effects.json` defines each effect per level: its single `sprites` (with a `glow` level where it is a glow; `step` means the level of the score step), its `particles` (sprite, count, tint), the draw calls of interface parts the effect owns (`interfaceDrawCalls`: popup text, banner panel and text, rank badge, emoji art) and the overdraw layers it is designed to. It is the one source of the table in section 5.

| Effect | Low (designed first) | Medium and high add |
|---|---|---|
| `tilePlace` | Four warm dust motes (`color.gold.hi`) puff from under the tile as it settles | Sparks (`color.gold.default`) and one trail streak behind the travelling tile, ending when the travel ends (`motion.duration.base`) |
| `scorePopup` | The "+N" text only | High: eight sparks around the numerals (`color.role.score.fill`) |
| `scoreCascade` | Glow at the popup in `color.role.score.fill` at the step's `glow` level, a burst of stars and sparks, and streaks flying to the team meter. Counts per step in section 5 | More of the same three, by `scoreCascade.steps[].particleShare` of the level |
| `spinnerFormation` | Ring pulse on the double (`color.gold.hi`) and two arm markers (`streak`, `color.role.secondary.fill`) sliding out on the new ends | Sparks; high adds stars |
| `domino` | Glow behind the last tile in the team fill at `glow.intense`, gold stars and sparks in the team fill; the localised banner (panel and text) | Warm dust, more stars and sparks |
| `block` | The lock stamp (`color.gold.default`, alpha) across the board. The board dims by tinting the board's vertex colour toward `color.scrim`, so the dim adds no layer and no draw call | An impact ring (`color.gold.hi`) and a dust puff around the stamp |
| `matchWin` | The banner, paper confetti in the team fill, gold-leaf confetti in `color.gold.default`, a few stars | A glow behind the banner (`glow.strong`), sparks, more confetti |
| `rankUp` | Rays behind the badge (`color.gold.default`), stars and sparks | A glow behind the badge (`glow.strong`), more stars and sparks |
| `emoji` | The emoji art of up to four seats; no particles | A small sparkle of stars and sparks at each pop |

Owner decisions of 2026-11-16 (proposed here first):
- `block` dim amount: the board's vertex colour moves toward `color.scrim` by `opacity.disabled` (0.4); board panel B-8 shows it. The dim belongs to the board view, which does not exist yet (section 8).
- Glow levels where the motion spec names none: `domino` `glow.intense`, `matchWin` and `rankUp` `glow.strong`. `effects.json` uses them.
- Stamp tint `color.gold.default`. `effects.json` uses it.

## 5. Budget table

`node tools/vfx/vfx.mjs` checks `effects.json` against `docs/design-tokens.json` and fails when an effect has more particles, draw calls or overdraw layers than its budget at a level, when overdraw passes the level's `maxOverdrawLayers`, when a budgeted effect has no definition, when a higher level drops a sprite of a lower one, when the score cascade does not follow the step shares, when a tint or glow is not a token, or when this table is stale. `node --test tools/vfx/vfx.test.mjs` runs in CI through `npm run test:tools`. `node tools/vfx/vfx.mjs --print` prints both tables.

How the estimate counts:
- Draw calls: one per material the effect uses (all its sprites share the one atlas, so one material batches in one call when its quads are drawn in sequence on one canvas) plus `interfaceDrawCalls`. Base scene batches, the seats' rarity halos included, are not counted here (`docs/rarity.md` section 6).
- Overdraw layers: the most effect layers designed to stack on one pixel at the effect's peak: the glow behind a burst is one, a sparse burst over it another, a dense burst at the high level one more. Particles of one burst spawn spread over its radius; where they stack more, the device measurement decides.
- Pooled: from `effects.budgets`; this library adds no pool.

| Effect | Level | Particles / budget | Sprites and particles | Overdraw layers / budget | Draw calls / budget | Pooled |
|---|---|---|---|---|---|---|
| `tilePlace` | low | 4 / 4 | dust 4 | 1 / 1 | 1 / 1 | 4 |
| `tilePlace` | medium | 12 / 12 | dust 8, spark 3, streak 1 | 2 / 2 | 1 / 2 | 4 |
| `tilePlace` | high | 24 / 24 | dust 15, spark 8, streak 1 | 2 / 2 | 1 / 2 | 4 |
| `scorePopup` | low | 0 / 0 | none | 1 / 1 | 1 / 1 | 6 |
| `scorePopup` | medium | 0 / 0 | none | 1 / 2 | 1 / 2 | 6 |
| `scorePopup` | high | 8 / 8 | spark 8 | 2 / 2 | 2 / 2 | 6 |
| `scoreCascade` | low | 30 / 30 | glow (step), star 6, streak 6, spark 18 | 2 / 2 | 1 / 3 | 2 |
| `scoreCascade` | medium | 90 / 90 | glow (step), star 18, streak 18, spark 54 | 3 / 3 | 1 / 5 | 2 |
| `scoreCascade` | high | 180 / 180 | glow (step), star 36, streak 36, spark 108 | 4 / 4 | 1 / 6 | 2 |
| `spinnerFormation` | low | 2 / 10 | ring, streak 2 | 1 / 1 | 1 / 2 | 1 |
| `spinnerFormation` | medium | 30 / 30 | ring, streak 2, spark 28 | 2 / 2 | 1 / 3 | 1 |
| `spinnerFormation` | high | 60 / 60 | ring, streak 2, spark 46, star 12 | 3 / 3 | 1 / 4 | 1 |
| `domino` | low | 40 / 40 | glow (intense), star 8, spark 32 | 2 / 2 | 3 / 3 | 1 |
| `domino` | medium | 120 / 120 | glow (intense), star 20, spark 80, dust 20 | 3 / 3 | 3 / 6 | 1 |
| `domino` | high | 240 / 240 | glow (intense), star 40, spark 160, dust 40 | 4 / 4 | 3 / 7 | 1 |
| `block` | low | 0 / 12 | stamp | 1 / 1 | 1 / 2 | 1 |
| `block` | medium | 40 / 40 | stamp, ring, dust 40 | 2 / 2 | 2 / 4 | 1 |
| `block` | high | 80 / 80 | stamp, ring, dust 80 | 3 / 3 | 2 / 5 | 1 |
| `matchWin` | low | 60 / 60 | flake_paper 30, flake_leaf 20, star 10 | 2 / 2 | 4 / 5 | 1 |
| `matchWin` | medium | 200 / 200 | glow (strong), flake_paper 100, flake_leaf 60, star 20, spark 20 | 3 / 3 | 4 / 10 | 1 |
| `matchWin` | high | 400 / 400 | glow (strong), flake_paper 200, flake_leaf 120, star 40, spark 40 | 4 / 4 | 4 / 12 | 1 |
| `rankUp` | low | 50 / 50 | rays, star 10, spark 40 | 2 / 2 | 2 / 5 | 1 |
| `rankUp` | medium | 160 / 160 | rays, glow (strong), star 30, spark 130 | 3 / 3 | 2 / 10 | 1 |
| `rankUp` | high | 320 / 320 | rays, glow (strong), star 60, spark 260 | 4 / 4 | 2 / 12 | 1 |
| `emoji` | low | 0 / 0 | none | 1 / 1 | 4 / 4 | 4 |
| `emoji` | medium | 32 / 32 | star 8, spark 24 | 2 / 2 | 5 / 8 | 4 |
| `emoji` | high | 64 / 64 | star 16, spark 48 | 2 / 2 | 5 / 8 | 4 |

Score combo steps (5, 10, 15, 20, 25 and 30 or more points, `docs/ui-motion.md` section 8). Particles per level are `effects.budgets.scoreCascade` times the step's `particleShare`, rounded half up as `ScoreCascade.Burst` does, then split by the `stepMix` of `effects.json` (stars 20%, streaks 20%, sparks the rest). The glow is the step's `glow` level, tinted `color.role.score.fill`; shake comes from the step as before.

| Step | Points | Glow | low: star / streak / spark | medium: star / streak / spark | high: star / streak / spark |
|---|---|---|---|---|---|
| 1 | 5 | `glow.soft` | 5: 1 / 1 / 3 | 14: 3 / 3 / 8 | 27: 5 / 5 / 17 |
| 2 | 10 | `glow.soft` | 9: 2 / 2 / 5 | 27: 5 / 5 / 17 | 54: 11 / 11 / 32 |
| 3 | 15 | `glow.strong` | 14: 3 / 3 / 8 | 41: 8 / 8 / 25 | 81: 16 / 16 / 49 |
| 4 | 20 | `glow.strong` | 18: 4 / 4 / 10 | 54: 11 / 11 / 32 | 108: 22 / 22 / 64 |
| 5 | 25 | `glow.intense` | 24: 5 / 5 / 14 | 72: 14 / 14 / 44 | 144: 29 / 29 / 86 |
| 6 | 30 | `glow.intense` | 30: 6 / 6 / 18 | 90: 18 / 18 / 54 | 180: 36 / 36 / 108 |

## 6. Marks outside the effect budgets

`effects.json` lists these under `marks`; each is one sprite and one overdraw layer with no particles, and the checker holds them to that.

| Mark | Sprite | Tint | Budget |
|---|---|---|---|
| Rarity halo | `halo` | rarity `fill`, alpha and scale of the tier's `glow` | `docs/rarity.md` section 6: base scene, one draw call for all seats |
| Rarity flow halo | `flow_halo` | as the halo; turned by one transform tween, resting with the arc at the top on low and with reduce motion | `docs/rarity.md` section 6 |
| Primary glow | `glow` | `color.role.primary.fill` at `glow.soft` | Part of its screen |
| Sheen | `sheen` | `color.gold.hi` | Part of the frame or skin it crosses |
| Tile shadow | `tile_shadow` | `color.shadow` at the `elevation` alpha | Part of the board |

Reward reveal and chest open: the reveal of `docs/pass-missions-medals.md` 6.1 adds no particles and runs outside the match scene; it is the item's pop-in with the rarity halo (or flow halo) behind it. The shop's chest reveal shows only the granted item (`docs/economy.md` 3.4) and uses the same halo. Neither has an effect budget, so this library adds no burst, confetti or other particle to them; one would need a `docs/ui-motion.md` budget first. No chest, reel or wheel imagery is drawn (`docs/localization.md` section 6).

## 7. In the client today

- `UI/Motion/EffectPool` and `UI/Motion/ParticleBudget` hold each effect's instances, lifetimes and particle caps per level and with reduce motion; `UI/Motion/ScoreCascade.Burst` gives a step's particle count; `UI/Board/TilePlaceLayer` runs the tile travel. `UI/Rarity/RarityHalo` draws the halo and flow halo and takes its two sprites through `Configure`.
- `UI/Effects/EffectPlayer` is the effect player. It parses `effects.json` once (`EffectLibrary`, which refuses a sprite, tint or glow that is not in the file or the tokens, a count over the effect's budget and overdraw over the level's `maxOverdrawLayers`), prewarms everything when it is built and draws with the two shared materials from the shared atlas: one `EffectQuads` graphic per material on the player's own sub-canvas, so an effect costs at most one draw call per material it uses. Each `Play` takes its counts from `EffectPool.Begin` (never more than the table, and never more than the level's live particles), tints by vertex colour from the token or the team, and splits a cut count over the particle entries in their proportions, which for `scoreCascade` gives the step table of section 5. The single sprites come from the effect level; particles come from the budget's particle level, which is low with reduce motion. Single sprites pop in and out on the score popup curve, so with reduce motion they only fade; nothing shakes. Timing never depends on the level: an instance ends at the effect's max ms.
- Motion uses only tokens: bursts spread over the area the caller names (a tile, the board, a seat) with `enter` and fade with `exit`; a trail travels with `standard` in the travel time it is given (`motion.duration.base` for a tile, `fast` with reduce motion), and when the trail arrives (`tilePlace`) the bursts start where it ends.
- Settings: `UI/Effects/EffectSettingsLink` applies the effect level of the graphics level and reduce motion to the player, `TilePlaceLayer` and `SeatEmojiLayer` when it is made and whenever either setting changes (`docs/ui-motion.md` section 7), so a drop to the low level or reduce motion cuts the counts at once.
- Hooked today: `tilePlace` from `TilePlaceLayer.Place`, `emoji` from `SeatEmojiLayer.Show`, and the score cascade per team, `domino`, `block` and `matchWin` from `Game/Match/MatchEffects.Read`, which reads each match view (`EffectCues`). `domino` and `block` start when the last tile settles, hand points `motion.duration.emphasis` later (after the banner enters), and `matchWin` after the hand-end effect ends.
- Not hooked, because nothing in the client raises the event yet: `rankUp` (no results screen shows a rank change; `docs/ui-motion.md` 8, Match result), `spinnerFormation` (no hook on the spinner tile), `scorePopup` sparks (no popup view) and the turn-total and carry-over popups of a chain. The match screen that creates the player, attaches it and feeds `MatchEffects` does not exist yet, so no prefab sets the sprites and materials. The `tile_shadow` and `sheen` marks are not drawn by the player: they belong to the tile view and the frames, which take them when those views are built.
- No prefab or screen sets the halo sprites and the additive material on `RarityHalo` yet; screens do that when they are built (`docs/rarity.md` 2.2.1).

## 8. Not done

- Device measurement: CPU and GPU ms, live particles, draw calls and overdraw per effect in its worst concurrency set on the reference devices (`docs/ui-motion.md` section 7, Measurement; `docs/performance.md` section 8). Every number in section 5 is a design estimate until the replay harness (#192) measures it on a device. CI checks 0 B GC per frame and the live particle cap (`EffectPlayerFrameTests`).
- Banners and the rank badge art, the block board dim (a board view tint) and the hooks listed in section 7.
- The reference board review of B-8 and the cultural review of the sprites.
