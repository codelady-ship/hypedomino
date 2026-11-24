# table.saffron_scrolls

Animated table of set 1 (#174). The carpet cloth is a concept made with template P-5 of `docs/asset-workflow.md` section 2, back layer wording; the gold inlay line and the wood rim are drawn by the script from colour tokens, and the motion layer is drawn by the same script. Motion: dust motes: 110 small warm specks drift in the lamp light over the whole table. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept generated on 2026-11-15 by an AI agent (Claude Code) running Codex CLI on the project owner's computer. W-3 layers, W-4 motion and W-5 export by the same agent with `work/saffron_scrolls_table.py`. No person has done W-3
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model `gpt-image-2` per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifest in the PNG names `ChatGPT` / `gpt-image` (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png (back layer)
- Inputs uploaded to the generator: none; one prompt, one image, no edits, in a new Codex session (`01a1e013-99af-79a2-b3a9-122f7abd6cb6`)
- Prompt: P-5 with the style key of section 2.2 pasted verbatim and the back layer named, quoted below. A P-5 prompt for a middle or front layer alone was not sent: set 1 showed that the generator draws a whole carpet whatever the layer words say (`table.crimson_carpet`), so the other layers are drawn by the script
- Concept against the prompt: Matches the prompt: a plain dark umber field framed by burnt-saffron acanthus scrolls with a palmette at the middle of each side and small madder buds, a madder guard, and a wide ivory border of small madder rosettes and stepped saffron diamonds. The saffron of the scrolls is the most saturated colour of the item (team hue check below). The image is 1672 x 941 with no light margin. Codex ran in `concept/` of a folder first named `saffron_medallion`; the agent renamed the item to `saffron_scrolls` before the first commit, because the art has scrolls and no medallion.
- Similarity check: not done (RC-3 open, O-7)
- Human work: none. The layers, the motion and the export are scripted image operations run by an AI agent, so RC-5 stays open; the script and its settings are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no script, letters, numbers, flags, emblems, religious symbols or casino cues seen in the carpet; the burnt-saffron scrolls are the closest colour of this set to `color.role.teamB` (team hue line below); they frame the field and do not dominate it, which the board review confirms or not; the pattern is geometric and floral (`docs/art-direction.md` 5.4)
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic` (a painted carpet with one motion; #174 asks for richer art at higher rarity), source `shop:offer.table_saffron_scrolls`. The offer id is a placeholder until the shop record exists (V-15). `table.starter` stays the `common` default table.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 1 animated detail (dust motes), so `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifest

`concept/20261115-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-15T16:39:48Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 16:39:03 UTC on 2026-11-15 (agent clock), shortly before `c2pa.created`. Codex saved the image under `~/.codex/generated_images/01a1e013-99af-79a2-b3a9-122f7abd6cb6/` and copied it into `concept/`; the two files are byte-identical (checked with `cmp`).

## Layers (W-3), motion (W-4) and export (W-5)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/table/saffron_scrolls/work/saffron_scrolls_table.py assets-src/cosmetic/table/saffron_scrolls/work/saffron_scrolls_table.json docs/design-tokens.json assets-src/cosmetic/table/saffron_scrolls assets-src/cosmetic/table/saffron_scrolls
```

It printed `saffron_scrolls: trim (0, 0, 1672, 941), stretch 1.001, crop (0, 0.0, 1672, 941.0), edges depth 300 step 420 start 330 score 2.1289, canvas 1280x720, seed 17412, motes 10 frames, key frame 6` (run again on 2026-11-16). Two more runs into two empty folders wrote files byte-identical to each other and to the committed ones.

The script reads its settings from `work/saffron_scrolls_table.json`; every table of this set uses the same script with its own settings file. Wood and gold come only from `docs/design-tokens.json`, so the rim and the inlay line are those of `table.starter` and the set-1 tables at every canvas size (`docs/art-direction.md` 5.4: the same layout on every table).

| Layer | File | How it is made |
|---|---|---|
| back | `work/saffron_scrolls_back.png` | The concept, its light margin trimmed (rows and columns where 98% of pixels are within 30 of the outermost ring's median colour; trim box `(0, 0, 1672, 941)`), fitted to 1920 x 1080 with a horizontal stretch of 1.001 and the crop `(0, 0.0, 1672, 941.0)`, as `table.crimson_carpet` does. Resampled to the canvas with a box filter. |
| middle | `work/saffron_scrolls_middle.png` | The gold inlay line of `table.crimson_carpet`: a 4 px line 54 px inside the edge with a small lozenge at each corner (at the 1920 x 1080 scale), shaded with light from above from `color.gold.lo` through `color.gold.mid` and `color.gold.default` to `color.gold.hi`, drawn at 2x and box-filtered down |
| front | `work/saffron_scrolls_front.png` | The wood rim of `table.crimson_carpet`: a 40 px band with a 28 px corner radius (at the 1920 x 1080 scale), shaded from `color.wood.lo` to `color.wood.hi`, drawn at 2x and box-filtered down |
| motion key | `work/saffron_scrolls_motion_key.png` | Frame 6 of the motion layer, kept as the source of the static |
| export layer `base` | `export/layer/base/0001.png` | back, middle and front composited: a layer of one frame, so a static layer (`docs/asset-workflow.md` 4.2) |
| export layer `motion` | `export/layer/motion/0001.png` to `0010.png` | The animated layer, transparent except the moving parts (below) |
| static | `export/static.png` | `base` with motion frame 6 on top: the flattened composition of both layers at the chosen key frame (`docs/art-direction.md` section 7, not frame 1) |

| Choice | Value | Source |
|---|---|---|
| Canvas | 1280 x 720 | Open (O-6). Loop frames keep the canvas of the static (`docs/art-direction.md` section 7, `tools/asset-export`), so the canvas sets the size of every frame. At the 1920 x 1080 of the static tables, 12 sprites would be 11,059,200 bytes by the ASTC 6x6 rule; at 1600 x 900 7,689,600; at 1280 x 720 4,930,560; at 960 x 540 2,764,800. 1280 x 720 keeps an animated table under 5 MB and is scaled up 1.5 times on a phone with a 1080 px short side, where the wool texture and the 2.7 px inlay line still read; 960 x 540 would double every pixel. The layout is the same as on every table: rim, inlay and lamp are scaled from the 1920 x 1080 values |
| Layers | 2: `base` (static, 1 frame) and `motion` (animated, 10 frames) | `docs/cosmetics.md` section 4: at least one layer animated, at most `TableMaxLayers` (not set yet, #97). The cloth, the inlay and the rim do not move, so they are one static layer; one animated layer takes one slot of the screen's `AnimationBudget` |
| Frames | 10 | Every motion is periodic in the 10 frames. The frame rate is not stored with the loop (`SpriteLoop` holds frames only) and the rate per class is open (O-5, O-6): at 8 fps the loop lasts 1.25 s, at the 12 fps of the ADR 0012 bench 0.83 s. The motion is slow enough for either; 8 fps is proposed for tables |
| Loop content | 110 motes from the seed 17412, each a soft round speck of the warm ivory `(255, 236, 198)` with a radius of 3.0 to 6.0 px (1920 x 1080 scale) and a Gaussian edge, opacity 0.5 to 0.9 scaled by the lamp; each circles its own point on an ellipse of 3 to 9 px once per loop, clockwise or not, and its opacity breathes once per loop with its own phase | Glow and sheen stay runtime (`docs/art-direction.md` section 4); the ambient detail of a table is part of its animation (section 6, Ambient) |
| Seamless loop | Mean channel change from frame 10 to frame 1 is 0.102 of 255, against 0.103 on average and 0.105 at most between the other neighbouring frames | Measured on `base` with each motion frame on top |
| Static | Frame 6 | `export/static.png` is `base` with motion frame 6 on top; it matches frame 6 of the loop (mean difference 0.000 of 255) |

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | Trim and fit as above. Canvas 1280 x 720 (the animated-table canvas of this set, above); the class canvas and how the table covers 4:3 to 21:9 screens are open (O-6) |
| 2. Paint-over, AI artefacts | No | No paint-over; generator artefacts in the fine border motifs are not checked at full size. The rim is flat-shaded wood colour with no grain; an artist paints the rim and the inlay |
| 3. Moving parts on their own layers | Yes | The motion is its own export layer; the cloth, the inlay and the rim stay as separate layers in `work/` |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow or rarity colour seen; team hues in the readability section; the motes are drawn inside their own pixels, with no halo |
| 5. Silhouette and smallest-size tests | Tile readability only | Readability check below |

## Readability check (acceptance of #174)

```bash
python -I assets-src/cosmetic/table/saffron_scrolls/work/saffron_scrolls_check.py assets-src/cosmetic/table/saffron_scrolls assets-src/tile/ivory/export/face assets-src/effect/tile_shadow/export/static.png docs/design-tokens.json <out>
```

The check measures the static and every frame of the loop (`base` with each motion frame on top). Centre: the 2v2 landscape board region of `docs/tiles.md` section 3 (1516 x 521 of the 1080-unit canvas), centred on the table. Ivory: the median relative luminance (WCAG) of the opaque pixels of the 28 `tile.ivory` faces, box-filtered to 32 x 64 px, the smallest display size of `docs/tiles.md` section 3; this gives 0.743, lower and so stricter than the 0.765 of the full-size face. Pass rule of this set: ivory against the median of the centre at least 7:1, and against its 90th percentile (the light motifs) at least 4.5:1. The same measure gives `table.starter` 9.2:1 and 8.3:1.

Output on 2026-11-16: `ivory face 0.743; worst scene frame 0001: centre median 0.024, 90th percentile 0.039; contrast 10.7:1 against the median (needs 7.0:1), 8.9:1 against the 90th percentile (needs 4.5:1): pass`

The script also writes `check.png`: the static scaled to the 828 px short side of the iPhone XR with a chain of nine `tile.ivory` faces at 32 x 64 px and their runtime shadow (`effect.tile_shadow`, `color.shadow` at `elevation.e1`) in the centre, and `check_x3.png`, its centre enlarged three times. The agent looked at both: pips and faces read on the carpet, and no motif in the centre competes with them.

Team hues (`docs/art-direction.md` section 2): share of static pixels with saturation and value of at least 0.5 within 15 degrees of hue of `color.role.teamA` and `color.role.teamB`: teamA 0.00%, teamB 20.83%. The same measure gives `table.crimson_carpet` teamB 17.42%, `table.indigo_carpet` 8.47% and `table.starter` 0.20% (teamA 0.00% on all three); the teamB share counts ochre, saffron and copper, which sit next to the team-B orange on the hue circle. Not a pass rule; the board review of section 9 decides.

`npm run asset:check -- assets-src/cosmetic/table/saffron_scrolls`: 12 sprites, ASTC 6x6 estimate 4,930,560 bytes, no size limit set for this type and kind (docs/asset-workflow.md O-5)

Not done: the reduce-motion and low-level check of 5.2 (Unity), board panel B-8, W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Any screen shape (2026-11-16)

Owner decision of 2026-11-16, after the Art Bench check of #724 on a 20:9 phone: the rim, the inlay and the carpet border must show whole on every screen shape (16:9 to 21:9, 4:3, nearly square). Done on 2026-11-16 by Claude Code in `work/saffron_scrolls_table.py` (`periodic_edges`, `write_slice`), with no new canvas and no new layer; the rule is `docs/cosmetics.md` 8.4 and `docs/art-direction.md` 5.4.

- Ring depth 300 px at the 1920 x 1080 scale (200 and 280 px on the 1280 x 720 canvas): the rim, the inlay, the carpet border and its guards, the same on all four sides. The ring takes in the inner band of scrolls with its palmettes, so it repeats with the outer border and is never cropped.
- The top and bottom borders now repeat. Each band is split into its light (a Gaussian blur of 120 px along the edge, made symmetric left to right) and its detail. The script tries steps of 150 to 480 px in steps of 6 (so a 1280 x 720 canvas keeps whole pixels) and corner pieces of at least the depth plus 30 px and at most 520 px less a quarter step, scores each by how well the art matches at both ends of one step (windows of 12 and 48 px), how mirror-symmetric it is at the start, and how far the repeated band drifts from the art, and keeps the lowest score. Chosen: step 420 px, corner pieces 0 to 330 px and 1590 to 1920 px, score 2.1289. Between the corner pieces both bands are the run from 330 to 750 px repeated 3 times, joined by 32 px cross-fades, and they fade into the untouched field over the 16 px inside the ring. The corner pieces and the side bands keep the art as it was; a motif unique to the middle of the top or bottom edge is replaced by the run.
- `export/slice.json`, `{"depth": 200, "step": 280}`, records the depth and the step in pixels of the static; `CosmeticContent` sets them as the sprite border of the static and of every layer frame, and `TableImage` draws the table at the screen's shape. Drawn by the same rule as `TableCloth` (scratch render, not committed): 16:9 steps +0 x1.000; 20:9 steps +1 x1.026; 21:9 steps +1 x1.077; 4:3 steps -1 x0.960 (steps gained or lost in the middle of the long edges, then the horizontal scale of the whole table).
- Readability check of #708 run again on the new static (section above): pass. `npm run asset:check` gives the same sprite count and estimate as before.
- Not done: the look of the repeated edges is not reviewed by a person; the device check on the Art Bench comes with the bench build of this change.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `dc5535fc63def20d403203c5d06082bf95b48b4280c7f1f9adecc43fadeb7a6c` | 1672 x 941 RGB, the generator's output, unchanged | Codex CLI `$imagegen`, copied byte for byte by Codex into `concept/` |
| work/saffron_scrolls_table.py | `f194b3c59457057f156fd4c8a77a477594a7c9d29783f205825c38e023cf07ba` | W-3, W-4 and W-5 script | Written by Claude Code, 2026-11-15, from `starter_carpet.py` and `crimson_carpet_cleanup.py` of set 1; the same script for every table of this set. `periodic_edges` and `write_slice` added on 2026-11-16 (section Any screen shape) |
| work/saffron_scrolls_table.json | `94bc32ab08e85b177bb37b0b0896581390466730ac1835caca7d50854c634823` | The item's settings: canvas, seed, concept and clean-up, motion | Written by Claude Code, 2026-11-15 |
| work/saffron_scrolls_check.py | `34c3c89c3df7b87a1711fbe2451b7109eb1da623991e79755bfe7953d61b5421` | The readability check | Written by Claude Code, 2026-11-15; the same script for every table of this set |
| work/saffron_scrolls_back.png | `86e9d63332df3fca87ac4ee5c5f3825872b2d5599438447ab3e2c31e8ed9ff40` | 1280 x 720 RGBA, back layer | `saffron_scrolls_table.py` |
| work/saffron_scrolls_middle.png | `f1ce267d022ab91f5ae083b775b9e997fec40232ea90496aa577c26a0af68586` | 1280 x 720 RGBA, middle layer, gold inlay | `saffron_scrolls_table.py` |
| work/saffron_scrolls_front.png | `db81edc5c67289b8fc6cf6a1c19f63bd80e90d8454ae2877dffaf0bd21faf680` | 1280 x 720 RGBA, front layer, wood rim | `saffron_scrolls_table.py` |
| work/saffron_scrolls_motion_key.png | `bbedbf2f171b6efd95c1671b98b91f78ffcfa9f9e9de010de2d612caa5fd3278` | 1280 x 720 RGBA, motion frame 6 | `saffron_scrolls_table.py` |
| export/layer/base/0001.png | `c045b8d9cefa62504938af038db797de9229d57968f46b3b2b8f2ee7c0a4e2c8` | 1280 x 720 RGBA, the static `base` layer | `saffron_scrolls_table.py` |
| export/layer/motion/0001.png to 0010.png | `58d339adb6eaea03ea98dbab38f1f808ffd259655ebce02ea05f73a4ffe314d6` (sha256 of the 10 files concatenated in name order) | 1280 x 720 RGBA, the `motion` loop | `saffron_scrolls_table.py` |
| export/static.png | `a7cac68b1365df8e0b8b0102b477d8119cc151e06a0fa5751334cadad4534e17` | 1280 x 720 RGBA static, base and motion frame 6 | `saffron_scrolls_table.py` |
| export/slice.json | `1539bc3b244cdaa813aafbbe564f67370bfde0dd43cb4eaf37934ea4f1adbb85` | `{"depth": 200, "step": 280}`: the ring depth and the edge step in pixels of the static (`docs/cosmetics.md` 8.4) | `saffron_scrolls_table.py`, 2026-11-16 |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-5 back layer with the style key, quoted below. Sent 16:39:03 UTC by Claude Code, Codex session `01a1e013-99af-79a2-b3a9-122f7abd6cb6`; `c2pa.created` 2026-11-15T16:39:48Z |
| 2026-11-15 | work/, export/ | `saffron_scrolls_table.py` on the concept |
| 2026-11-16 | work/, export/ | `saffron_scrolls_table.py` with the repeated top and bottom borders and `export/slice.json` (section Any screen shape) |

Prompt of `concept/20261115-01.png`, verbatim (the Codex call wraps it in `$imagegen <prompt> Generate exactly one image, in a wide landscape format. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`):

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Top-down view of a woven carpet table cloth, a dark umber salon carpet with a plain centre framed by saffron palmette scrolls, and a wide border of small rosettes and stepped diamonds in saffron and ivory, back (the whole carpet, field and border, filling the image edge to edge) layer only, original geometric and floral pattern, calm low-contrast centre where game pieces will sit. Palette: dark umber brown, burnt saffron, ivory and deep madder red.
```

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-15.
