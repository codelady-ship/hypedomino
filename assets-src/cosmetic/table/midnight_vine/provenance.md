# table.midnight_vine

Animated table of set 1 (#174). The carpet cloth is a concept made with template P-5 of `docs/asset-workflow.md` section 2, back layer wording; the gold inlay line and the wood rim are drawn by the script from colour tokens, and the motion layer is drawn by the same script. Motion: six silver silk tassels on gold caps at the inner corners and the middle of the left and right sides of the border sway, and the gold threads shimmer in two slow sweeping waves with twinkling knots. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept generated on 2026-11-15 by an AI agent (Claude Code) running Codex CLI on the project owner's computer. W-3 layers, W-4 motion and W-5 export by the same agent with `work/midnight_vine_table.py`. No person has done W-3
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model `gpt-image-2` per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifest in the PNG names `ChatGPT` / `gpt-image` (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png (back layer)
- Inputs uploaded to the generator: none; one prompt, one image, no edits, in a new Codex session (`01a1e014-9b61-7de3-b7ed-97f58aaae67c`)
- Prompt: P-5 with the style key of section 2.2 pasted verbatim and the back layer named, quoted below. A P-5 prompt for a middle or front layer alone was not sent: set 1 showed that the generator draws a whole carpet whatever the layer words say (`table.crimson_carpet`), so the other layers are drawn by the script
- Concept against the prompt: Matches the prompt: a blue-black field with a plain centre framed by silver-ivory vines with ivory six-petal flowers, grey leaves and small gold buds, a gold-thread guard band, and a wide border of ivory and silver acanthus scrolls with stepped ivory and grey diamonds edged in gold. The image is 1672 x 941 with no light margin.
- Similarity check: not done (RC-3 open, O-7)
- Human work: none. The layers, the motion and the export are scripted image operations run by an AI agent, so RC-5 stays open; the script and its settings are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no script, letters, numbers, flags, emblems, religious symbols or casino cues seen in the carpet; no team hue dominates; the pattern is geometric and floral (`docs/art-direction.md` 5.4)
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `mythic` (the richest of the set, a painted carpet with two motions, tassels and shimmer; #174 asks for richer art at higher rarity), source `shop:offer.table_midnight_vine`. The offer id is a placeholder until the shop record exists (V-15). `table.starter` stays the `common` default table.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 3 animated details (tassels, gold-thread light bands, knot twinkle), so `mythic`. Any tier split stated above is superseded.

## Embedded C2PA manifest

`concept/20261115-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-15T16:41:07Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 16:40:09 UTC on 2026-11-15 (agent clock), shortly before `c2pa.created`. Codex saved the image under `~/.codex/generated_images/01a1e014-9b61-7de3-b7ed-97f58aaae67c/` and copied it into `concept/`; the two files are byte-identical (checked with `cmp`).

## Layers (W-3), motion (W-4) and export (W-5)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/table/midnight_vine/work/midnight_vine_table.py assets-src/cosmetic/table/midnight_vine/work/midnight_vine_table.json docs/design-tokens.json assets-src/cosmetic/table/midnight_vine assets-src/cosmetic/table/midnight_vine
```

It printed `midnight_vine: trim (0, 0, 1672, 941), stretch 1.001, crop (0, 0.0, 1672, 941.0), edges depth 216 step 414 start 339 score 2.4578, canvas 1280x720, seed 17413, shimmer+tassels 10 frames, key frame 3` (run again on 2026-11-16). Two more runs into two empty folders wrote files byte-identical to each other and to the committed ones.

The script reads its settings from `work/midnight_vine_table.json`; every table of this set uses the same script with its own settings file. Wood and gold come only from `docs/design-tokens.json`, so the rim and the inlay line are those of `table.starter` and the set-1 tables at every canvas size (`docs/art-direction.md` 5.4: the same layout on every table).

| Layer | File | How it is made |
|---|---|---|
| back | `work/midnight_vine_back.png` | The concept, its light margin trimmed (rows and columns where 98% of pixels are within 30 of the outermost ring's median colour; trim box `(0, 0, 1672, 941)`), fitted to 1920 x 1080 with a horizontal stretch of 1.001 and the crop `(0, 0.0, 1672, 941.0)`, as `table.crimson_carpet` does. Resampled to the canvas with a box filter. |
| middle | `work/midnight_vine_middle.png` | The gold inlay line of `table.crimson_carpet`: a 4 px line 54 px inside the edge with a small lozenge at each corner (at the 1920 x 1080 scale), shaded with light from above from `color.gold.lo` through `color.gold.mid` and `color.gold.default` to `color.gold.hi`, drawn at 2x and box-filtered down |
| front | `work/midnight_vine_front.png` | The wood rim of `table.crimson_carpet`: a 40 px band with a 28 px corner radius (at the 1920 x 1080 scale), shaded from `color.wood.lo` to `color.wood.hi`, drawn at 2x and box-filtered down |
| motion key | `work/midnight_vine_motion_key.png` | Frame 3 of the motion layer, kept as the source of the static |
| export layer `base` | `export/layer/base/0001.png` | back, middle and front composited: a layer of one frame, so a static layer (`docs/asset-workflow.md` 4.2) |
| export layer `motion` | `export/layer/motion/0001.png` to `0010.png` | The animated layer, transparent except the moving parts (below) |
| static | `export/static.png` | `base` with motion frame 3 on top: the flattened composition of both layers at the chosen key frame (`docs/art-direction.md` section 7, not frame 1) |

| Choice | Value | Source |
|---|---|---|
| Canvas | 1280 x 720 | Open (O-6). Loop frames keep the canvas of the static (`docs/art-direction.md` section 7, `tools/asset-export`), so the canvas sets the size of every frame. At the 1920 x 1080 of the static tables, 12 sprites would be 11,059,200 bytes by the ASTC 6x6 rule; at 1600 x 900 7,689,600; at 1280 x 720 4,930,560; at 960 x 540 2,764,800. 1280 x 720 keeps an animated table under 5 MB and is scaled up 1.5 times on a phone with a 1080 px short side, where the wool texture and the 2.7 px inlay line still read; 960 x 540 would double every pixel. The layout is the same as on every table: rim, inlay and lamp are scaled from the 1920 x 1080 values |
| Layers | 2: `base` (static, 1 frame) and `motion` (animated, 10 frames) | `docs/cosmetics.md` section 4: at least one layer animated, at most `TableMaxLayers` (not set yet, #97). The cloth, the inlay and the rim do not move, so they are one static layer; one animated layer takes one slot of the screen's `AnimationBudget` |
| Frames | 10 | Every motion is periodic in the 10 frames. The frame rate is not stored with the loop (`SpriteLoop` holds frames only) and the rate per class is open (O-5, O-6): at 8 fps the loop lasts 1.25 s, at the 12 fps of the ADR 0012 bench 0.83 s. The motion is slow enough for either; 8 fps is proposed for tables |
| Loop content | Gold-thread shimmer: the gold threads of the carpet are picked from the cloth by colour (hue 0.08 to 0.17 of the hue circle, saturation at least 0.5, value at least 0.45), and only those pixels are tinted toward `color.gold.hi`, at most 80% opacity: 2 light bands sweep once per loop, mirror-symmetric about the middle of the table (changed 2026-11-16, section Any screen shape) (cos to the power 3, weight 0.8), and each block of 2 x 2 knots twinkles once per loop with its own phase (weight 0.5); scaled by the lamp and nothing under the rim; Tassels at the four inner corners of the border and the middle of the left and right sides (136 and 126 px in from the edge at the 1920 x 1080 scale), each a gold cap of `color.gold.default` with a `color.gold.hi` highlight and a `color.gold.mid` band, and 13 silk strands 66 px long fanning out to 20 px, in the item's own silk colours, darker toward the tips; each tassel turns by up to 11 degrees about its cap and its strands bend by up to 6 px, once per loop with its own phase; drawn at 4x and box-filtered down, then shaded by the lamp | Glow and sheen stay runtime (`docs/art-direction.md` section 4); the ambient detail of a table is part of its animation (section 6, Ambient) |
| Seamless loop | Mean channel change from frame 10 to frame 1 is 0.651 of 255, against 0.690 on average and 0.780 at most between the other neighbouring frames | Measured on `base` with each motion frame on top |
| Static | Frame 3 | `export/static.png` is `base` with motion frame 3 on top; it matches frame 3 of the loop (mean difference 0.000 of 255) |

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | Trim and fit as above. Canvas 1280 x 720 (the animated-table canvas of this set, above); the class canvas and how the table covers 4:3 to 21:9 screens are open (O-6) |
| 2. Paint-over, AI artefacts | No | No paint-over; generator artefacts in the fine border motifs are not checked at full size. The rim is flat-shaded wood colour with no grain; an artist paints the rim and the inlay |
| 3. Moving parts on their own layers | Yes | The motion is its own export layer; the cloth, the inlay and the rim stay as separate layers in `work/` |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow or rarity colour seen; team hues in the readability section; the shimmer is drawn inside its own pixels, with no halo |
| 5. Silhouette and smallest-size tests | Tile readability only | Readability check below |

## Readability check (acceptance of #174)

```bash
python -I assets-src/cosmetic/table/midnight_vine/work/midnight_vine_check.py assets-src/cosmetic/table/midnight_vine assets-src/tile/ivory/export/face assets-src/effect/tile_shadow/export/static.png docs/design-tokens.json <out>
```

The check measures the static and every frame of the loop (`base` with each motion frame on top). Centre: the 2v2 landscape board region of `docs/tiles.md` section 3 (1516 x 521 of the 1080-unit canvas), centred on the table. Ivory: the median relative luminance (WCAG) of the opaque pixels of the 28 `tile.ivory` faces, box-filtered to 32 x 64 px, the smallest display size of `docs/tiles.md` section 3; this gives 0.743, lower and so stricter than the 0.765 of the full-size face. Pass rule of this set: ivory against the median of the centre at least 7:1, and against its 90th percentile (the light motifs) at least 4.5:1. The same measure gives `table.starter` 9.2:1 and 8.3:1.

Output on 2026-11-16: `ivory face 0.743; worst scene static: centre median 0.008, 90th percentile 0.015; contrast 13.8:1 against the median (needs 7.0:1), 12.2:1 against the 90th percentile (needs 4.5:1): pass`

The script also writes `check.png`: the static scaled to the 828 px short side of the iPhone XR with a chain of nine `tile.ivory` faces at 32 x 64 px and their runtime shadow (`effect.tile_shadow`, `color.shadow` at `elevation.e1`) in the centre, and `check_x3.png`, its centre enlarged three times. The agent looked at both: pips and faces read on the carpet, and no motif in the centre competes with them.

Team hues (`docs/art-direction.md` section 2): share of static pixels with saturation and value of at least 0.5 within 15 degrees of hue of `color.role.teamA` and `color.role.teamB`: teamA 0.00%, teamB 1.59%. The same measure gives `table.crimson_carpet` teamB 17.42%, `table.indigo_carpet` 8.47% and `table.starter` 0.20% (teamA 0.00% on all three); the teamB share counts ochre, saffron and copper, which sit next to the team-B orange on the hue circle. Not a pass rule; the board review of section 9 decides.

`npm run asset:check -- assets-src/cosmetic/table/midnight_vine`: 12 sprites, ASTC 6x6 estimate 4,930,560 bytes, no size limit set for this type and kind (docs/asset-workflow.md O-5)

Not done: the reduce-motion and low-level check of 5.2 (Unity), board panel B-8, W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Any screen shape (2026-11-16)

Owner decision of 2026-11-16, after the Art Bench check of #724 on a 20:9 phone: the rim, the inlay and the carpet border must show whole on every screen shape (16:9 to 21:9, 4:3, nearly square). Done on 2026-11-16 by Claude Code in `work/midnight_vine_table.py` (`periodic_edges`, `write_slice`), with no new canvas and no new layer; the rule is `docs/cosmetics.md` 8.4 and `docs/art-direction.md` 5.4.

- Ring depth 216 px at the 1920 x 1080 scale (144 and 276 px on the 1280 x 720 canvas): the rim, the inlay, the carpet border and its guards, the same on all four sides. The ring takes in the vine border inside the guard and the tassels, so both stay whole.
- The top and bottom borders now repeat. Each band is split into its light (a Gaussian blur of 120 px along the edge, made symmetric left to right) and its detail. The script tries steps of 150 to 480 px in steps of 6 (so a 1280 x 720 canvas keeps whole pixels) and corner pieces of at least the depth plus 30 px and at most 520 px less a quarter step, scores each by how well the art matches at both ends of one step (windows of 12 and 48 px), how mirror-symmetric it is at the start, and how far the repeated band drifts from the art, and keeps the lowest score. Chosen: step 414 px, corner pieces 0 to 339 px and 1581 to 1920 px, score 2.4578. Between the corner pieces both bands are the run from 339 to 753 px repeated 3 times, joined by 32 px cross-fades, and they fade into the untouched field over the 16 px inside the ring. The corner pieces and the side bands keep the art as it was; a motif unique to the middle of the top or bottom edge is replaced by the run. The shimmer's light bands now sweep mirror-symmetrically from the middle of the table toward both ends (the angle is measured from the vertical axis on both sides) instead of turning around it, so the art on both sides of the middle of a long edge matches where the client joins two runs; the loop numbers above are measured again. The two tassels in the middle of the top and bottom edges are dropped (`places: ends`), since the middle of those edges now repeats; the corner tassels and the tassels in the middle of the left and right sides stay. The counted details of the rarity line are unchanged (tassels, light bands, knot twinkle).
- `export/slice.json`, `{"depth": 144, "step": 276}`, records the depth and the step in pixels of the static; `CosmeticContent` sets them as the sprite border of the static and of every layer frame, and `TableImage` draws the table at the screen's shape. Drawn by the same rule as `TableCloth` (scratch render, not committed): 16:9 steps +0 x1.000; 20:9 steps +1 x1.028; 21:9 steps +1 x1.080; 4:3 steps -1 x0.956 (steps gained or lost in the middle of the long edges, then the horizontal scale of the whole table).
- Readability check of #708 run again on the new static (section above): pass. `npm run asset:check` gives the same sprite count and estimate as before.
- Not done: the look of the repeated edges is not reviewed by a person; the device check on the Art Bench comes with the bench build of this change.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `66abb1beb8cf8b122e64d2e4dabfee6284b32a973569efa79e123d5e13060af4` | 1672 x 941 RGB, the generator's output, unchanged | Codex CLI `$imagegen`, copied byte for byte by Codex into `concept/` |
| work/midnight_vine_table.py | `f194b3c59457057f156fd4c8a77a477594a7c9d29783f205825c38e023cf07ba` | W-3, W-4 and W-5 script | Written by Claude Code, 2026-11-15, from `starter_carpet.py` and `crimson_carpet_cleanup.py` of set 1; the same script for every table of this set. `periodic_edges` and `write_slice` added on 2026-11-16 (section Any screen shape) |
| work/midnight_vine_table.json | `99919ddeba8a7b7fda2f8eb7f04e908774833b595c0460f7a1af49872d4ccaf6` | The item's settings: canvas, seed, concept and clean-up, motion | Written by Claude Code, 2026-11-15 |
| work/midnight_vine_check.py | `34c3c89c3df7b87a1711fbe2451b7109eb1da623991e79755bfe7953d61b5421` | The readability check | Written by Claude Code, 2026-11-15; the same script for every table of this set |
| work/midnight_vine_back.png | `092e45216001451889f7e14aef2b1369062d0faa4a0b4555f992ff9e8af7c906` | 1280 x 720 RGBA, back layer | `midnight_vine_table.py` |
| work/midnight_vine_middle.png | `f1ce267d022ab91f5ae083b775b9e997fec40232ea90496aa577c26a0af68586` | 1280 x 720 RGBA, middle layer, gold inlay | `midnight_vine_table.py` |
| work/midnight_vine_front.png | `db81edc5c67289b8fc6cf6a1c19f63bd80e90d8454ae2877dffaf0bd21faf680` | 1280 x 720 RGBA, front layer, wood rim | `midnight_vine_table.py` |
| work/midnight_vine_motion_key.png | `1783b04c94801920c8a2782e949b8dfda463bd9c3e5e432053e6885f6526d5a1` | 1280 x 720 RGBA, motion frame 3 | `midnight_vine_table.py` |
| export/layer/base/0001.png | `39ecdc2dbfb936cac536925ad16723f2513f8b939b7c14ceab6d28012eb5aad1` | 1280 x 720 RGBA, the static `base` layer | `midnight_vine_table.py` |
| export/layer/motion/0001.png to 0010.png | `a8e81471c5c4c9944541108d2cf15c003e7f1c7b34b4e0f5f5de9dc43b981352` (sha256 of the 10 files concatenated in name order) | 1280 x 720 RGBA, the `motion` loop | `midnight_vine_table.py` |
| export/static.png | `ab7b1db9b7129cf38086aa62d1a5d3a0d4c6200bb0d7320b03527e3ab48362b9` | 1280 x 720 RGBA static, base and motion frame 3 | `midnight_vine_table.py` |
| export/slice.json | `9c9efc6d694cd1798f7111a07f0f46940f6cf877507cd8e6ba7749f2bf94fe2c` | `{"depth": 144, "step": 276}`: the ring depth and the edge step in pixels of the static (`docs/cosmetics.md` 8.4) | `midnight_vine_table.py`, 2026-11-16 |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-5 back layer with the style key, quoted below. Sent 16:40:09 UTC by Claude Code, Codex session `01a1e014-9b61-7de3-b7ed-97f58aaae67c`; `c2pa.created` 2026-11-15T16:41:07Z |
| 2026-11-15 | work/, export/ | `midnight_vine_table.py` on the concept |
| 2026-11-16 | work/, export/ | `midnight_vine_table.py` with the repeated top and bottom borders and `export/slice.json` (section Any screen shape) |

Prompt of `concept/20261115-01.png`, verbatim (the Codex call wraps it in `$imagegen <prompt> Generate exactly one image, in a wide landscape format. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`):

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Top-down view of a woven carpet table cloth, a midnight salon carpet with fine silver-ivory vines and small gold buds on a deep field, and a wide border of scrolling vines and stepped diamonds edged with gold thread, back (the whole carpet, field and border, filling the image edge to edge) layer only, original geometric and floral pattern, calm low-contrast centre where game pieces will sit. Palette: midnight blue-black, silver grey, ivory and antique gold.
```

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-15.
