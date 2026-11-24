# table.crimson_carpet

Static table of set 1 (#174), the crimson carpet. The back layer is a concept made with template P-5 of `docs/asset-workflow.md` section 2; the middle (gold inlay) and front (wood rim) layers are drawn by the clean-up script from colour tokens. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-14 while the work was done; the calmed centre (#174, readability rule of #708) recorded on 2026-11-16 the same way.

- artSource: ai_assisted
- Authors: W-1 back-layer concept generated in the project owner's second ChatGPT account on 2026-11-14 by an AI agent (Claude Code) driving the owner's browser. W-3 layers and W-5 export by the same agent with `work/crimson_carpet_cleanup.py`; the centre calmed on 2026-11-16 by Claude Code in the same script. No person has done W-3
- Generator: ChatGPT in the browser, Create image tool, model `gpt-image` as named by the C2PA manifest embedded in the concept (below). Account holder: the project owner, second ChatGPT account (the first one reached its image limit during #172), signed in to the second browser on the owner's computer. Plan: Free (shown under the account name in the ChatGPT sidebar on 2026-11-14)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-14 for the Free plan and records the date here
- Canonical concept: concept/20261114-01.png (back layer)
- Inputs uploaded to the generator: none; one prompt, one image, no edits, in a new conversation (`chatgpt.com/c/6af85356...`, titled "Create Carpet Game Art" by ChatGPT)
- Prompt: P-5 with the style key of section 2.2 pasted verbatim and the back layer named, quoted below
- Concept against the prompt: Matches the prompt: a deep crimson field with small ochre and ivory floral motifs and stepped medallions, an ivory inner guard band, and a dark indigo outer border of eight-point rosettes and stepped diamonds. The carpet fills the image except a light margin of about 10 px, which the script trims.
- Similarity check: not done (RC-3 open, O-7)
- Human work: none. The layers and the export are scripted image operations run by an AI agent, so RC-5 stays open; the layers and the script are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no script, letters, numbers, flags, emblems, religious symbols, casino cues or team hues seen in the carpet; the pattern is geometric and floral (`docs/art-direction.md` 5.4)
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `rare`, source `shop:offer.table_crimson_carpet`. The offer id is a placeholder until the shop record exists (V-15). `table.starter` stays the `common` default table.

## Embedded C2PA manifest

`concept/20261114-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-14; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-14T12:26:29Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 12:26:17 UTC on 2026-11-14 (agent clock), shortly before `c2pa.created`.

## Middle and front layers: not generated

P-5 with the middle layer filled in was sent twice for `table.crimson_carpet`. Both images were rejected and not downloaded (`docs/asset-workflow.md` 2.4 step 4):

| Time (UTC, agent clock) | Conversation | Result |
|---|---|---|
| 2026-11-14 12:27:51 | the conversation of the back layer, after it | The whole rosette border drawn again around a thin gold line, plain grey only inside the line |
| 2026-11-14 12:29:05 | a new conversation (`chatgpt.com/c/6af85400...`, titled "Create Game Carpet Art") | A whole carpet, red field and rosette border, with a gold line and gold corner ornaments inside the border, grey only around the carpet |

The template starts with "Top-down view of a woven carpet table cloth, {theme}", so the generator draws a carpet whatever the layer words say. The front layer prompt was not sent, as it would have hit the same problem and used the account's image budget. Both were the same middle-layer prompt, quoted at the end of this file.

## Layers (W-3) and export (W-5)

Run on 2026-11-14 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/table/crimson_carpet/work/crimson_carpet_cleanup.py crimson_carpet 174 assets-src/cosmetic/table/crimson_carpet/concept/20261114-01.png assets-src/cosmetic/table/crimson_carpet/work assets-src/cosmetic/table/crimson_carpet/export
```

It printed `crimson_carpet: trim (11, 10, 1661, 929), stretch 0.990, crop (0, 0.0, 1650, 919.0), edges depth 174 step 438 start 303 score 2.2542, canvas 1920x1080` (run again on 2026-11-16 with the ring depth `174`). A second run into an empty folder wrote byte-identical files.

Run again on 2026-11-16 with the same command and versions after the centre step was added; it printed the same line, and a second run into an empty folder wrote byte-identical files. The middle and front layers are byte-identical to those of 2026-11-14; only the back layer and the static changed.

| Layer | File | How it is made |
|---|---|---|
| back | `work/crimson_carpet_back.png` | The concept, its light margin trimmed (rows and columns where 98% of pixels are within 30 of the outermost ring's median colour), fitted to the canvas. The trimmed carpet is 1650 x 919, already 16:9 within 1%, so it is scaled with a horizontal factor of 0.990 and no crop. Then the centre is calmed as `table.plum_trellis` does (#708), with milder settings: inside an ellipse of 0.44 x 0.40 of the canvas (half axes), fading in over the outer 25%, the detail above a Gaussian base of sigma 12 px is reduced by 30% and the value by 5% (`CALM_*` in the script). The field motifs stay; they are a little softer and darker in the middle of the field |
| middle | `work/crimson_carpet_middle.png` | A 4 px gold inlay line, a rectangle 54 px inside the canvas edge, with a small diamond (11 px half-diagonal) at each corner. Shaded with light from above across the line from `color.gold.lo` through `color.gold.mid` and `color.gold.default` to `color.gold.hi` (`docs/ui-motion.md`), drawn at 2x and box-filtered down. Everything else transparent |
| front | `work/crimson_carpet_front.png` | The wood rim: a 40 px band at the canvas edge with an opening of 28 px corner radius, shaded with light from above across its inner bevel from `color.wood.lo` through `color.wood.default` to `color.wood.hi`, drawn at 2x and box-filtered down. Everything inside transparent |
| static | `export/static.png` | back, middle and front composited in that order; 1920 x 1080 RGBA, fully opaque |

The middle and front layers are the same for every table of this set, so the layout of `docs/art-direction.md` 5.4 is the same on all of them. The table is static: `docs/cosmetics.md` section 4 forbids `layers` on a static table, so the layers stay in `work/` and only `export/static.png` is exported.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | Trim and fit as above. Canvas 1920 x 1080 (16:9), the frame of the reference prototype table; the class canvas and how the table covers 4:3 to 21:9 screens are open (O-6) |
| 2. Paint-over, AI artefacts | No | No paint-over; generator artefacts in the fine border motifs are not checked at full size. The rim is flat-shaded wood colour with no grain; an artist paints the rim and the inlay |
| 3. Moving parts on their own layers | Yes | back, middle and front in `work/`; no layer animates in this static item |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen |
| 5. Silhouette and smallest-size tests | Tile readability only | Tile faces and backs from the reference prototype renders (`table-v2/out/face_*.png`, `back_sema.png`, outside the repository) laid on the static at about the match size: centre field relative luminance median 0.064 (90th percentile 0.120); tile ivory 0.762; contrast of the ivory face against the median field 7.1:1. The centre field is calm: small motifs in ochre on red at low contrast. The tile backs of `tile.sema` sit on the indigo border in the check and stay readable by their gold rim. On 2026-11-16, by the readability rule of #708 (below): pass |

## Tile readability (#174, rule of #708)

The rule set by #708 for every table of set 1: the ivory face (median relative luminance of the 28 `tile.ivory` faces box-filtered to 32 x 64 px) against the 2v2 landscape board region (1516 x 521 of 1080 units, centred) has a contrast of at least 7:1 against the median and 4.5:1 against the 90th percentile. `work/crimson_carpet_check.py` is `teal_lattice_check.py` of #708, copied unchanged:

```bash
python -I assets-src/cosmetic/table/crimson_carpet/work/crimson_carpet_check.py assets-src/cosmetic/table/crimson_carpet assets-src/tile/ivory/export/face assets-src/effect/tile_shadow/export/static.png docs/design-tokens.json <out>
```

| Static | Ivory face | Centre median | 90th percentile | Against the median | Against the 90th percentile | Result |
|---|---|---|---|---|---|---|
| 2026-11-14 (before) | 0.7425 | 0.0635 | 0.1276 | 6.98:1 | 4.46:1 | fail |
| 2026-11-16 (calmed centre) | 0.7425 | 0.0588 | 0.1025 | 7.28:1 | 5.20:1 | pass |

Run on 2026-11-16; the script prints one decimal (`7.3:1`, `5.2:1`, `pass`), the two-decimal values are from the same computation with more digits. The check also renders nine ivory faces with their runtime shadow at 32 x 64 on the table at 828 px height; the agent looked at it.

`npm run asset:check -- assets-src/cosmetic/table/crimson_carpet`: 1 sprite, ASTC 6x6 estimate 921,600 bytes, no limit set for a static table (O-5); run again on 2026-11-16 with the same result.

Not done: W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Any screen shape (2026-11-16)

Owner decision of 2026-11-16, after the Art Bench check of #724 on a 20:9 phone: the rim, the inlay and the carpet border must show whole on every screen shape (16:9 to 21:9, 4:3, nearly square). Done on 2026-11-16 by Claude Code in `work/crimson_carpet_cleanup.py` (`periodic_edges`, `write_slice`), with no new canvas and no new layer; the rule is `docs/cosmetics.md` 8.4 and `docs/art-direction.md` 5.4.

- Ring depth 174 px at the 1920 x 1080 scale: the rim, the inlay, the carpet border and its guards, the same on all four sides.
- The top and bottom borders now repeat. Each band is split into its light (a Gaussian blur of 120 px along the edge, made symmetric left to right) and its detail. The script tries steps of 150 to 480 px in steps of 6 (so a 1280 x 720 canvas keeps whole pixels) and corner pieces of at least the depth plus 30 px and at most 520 px less a quarter step, scores each by how well the art matches at both ends of one step (windows of 12 and 48 px), how mirror-symmetric it is at the start, and how far the repeated band drifts from the art, and keeps the lowest score. Chosen: step 438 px, corner pieces 0 to 303 px and 1617 to 1920 px, score 2.2542. Between the corner pieces both bands are the run from 303 to 741 px repeated 3 times, joined by 32 px cross-fades, and they fade into the untouched field over the 16 px inside the ring. The corner pieces and the side bands keep the art as it was; a motif unique to the middle of the top or bottom edge is replaced by the run.
- `export/slice.json`, `{"depth": 174, "step": 438}`, records the depth and the step in pixels of the static; `CosmeticContent` sets them as the sprite border of the static and of every layer frame, and `TableImage` draws the table at the screen's shape. Drawn by the same rule as `TableCloth` (scratch render, not committed): 16:9 steps +0 x1.000; 20:9 steps +1 x1.018; 21:9 steps +1 x1.069; 4:3 steps -1 x0.972 (steps gained or lost in the middle of the long edges, then the horizontal scale of the whole table).
- Readability check of #708 run again on the new static (section above): pass. `npm run asset:check` gives the same sprite count and estimate as before.
- Not done: the look of the repeated edges is not reviewed by a person; the device check on the Art Bench comes with the bench build of this change.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261114-01.png | `693eb2abf4c094e66d203ef4db4474544b849aa655c45549c9459726bfc36b15` | 1672 x 941 RGB, the generator's download, unchanged | ChatGPT image viewer Download button, copied byte for byte |
| work/crimson_carpet_cleanup.py | `ee06acf9a0296d496da2460f506128fad78c5a1a00e6c4322d83afdf4caf3b0d` | W-3 and W-5 script | Written by Claude Code, 2026-11-14; the same script for both tables of that PR. The centre step (`calm_centre`) added on 2026-11-16 for this table only. `periodic_edges` and `write_slice` added on 2026-11-16 (section Any screen shape) |
| work/crimson_carpet_back.png | `a0df41ef9dd7e27661ce07978ff5ea295fa97f8da833085d8995c2a4ecb71951` | 1920 x 1080 RGBA, back layer | `crimson_carpet_cleanup.py` |
| work/crimson_carpet_middle.png | `bf002a633c302a3d7b799119253bb820e9e5220232919a6c245c23d1d319cf1c` | 1920 x 1080 RGBA, middle layer | `crimson_carpet_cleanup.py` |
| work/crimson_carpet_check.py | `34c3c89c3df7b87a1711fbe2451b7109eb1da623991e79755bfe7953d61b5421` | Readability check of #708 | `teal_lattice_check.py`, copied unchanged on 2026-11-16 |
| work/crimson_carpet_front.png | `7c5379bb0f1fb5766c32a4b9d171e1df2ddb7a3a5c6f3b8296a42fd1c928192d` | 1920 x 1080 RGBA, front layer | `crimson_carpet_cleanup.py` |
| export/static.png | `a978e4c9c858485f28b9374792f0c4351289b2afabbd56a8f65108b455389534` | 1920 x 1080 RGBA static, flattened layers | `crimson_carpet_cleanup.py` |
| export/slice.json | `520935835d6857efd41ec6a1def6ae2ab630d793b0b50d39c7d2d1b4b91a03a0` | `{"depth": 174, "step": 438}`: the ring depth and the edge step in pixels of the static (`docs/cosmetics.md` 8.4) | `crimson_carpet_cleanup.py`, 2026-11-16 |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-14 | concept/20261114-01.png | P-5 back layer with the style key, the first prompt of its conversation, quoted below. Sent 12:26:17 UTC by Claude Code; `c2pa.created` 2026-11-14T12:26:29Z |
| 2026-11-14 | work/crimson_carpet_back.png, work/crimson_carpet_middle.png, work/crimson_carpet_front.png, export/static.png | `crimson_carpet_cleanup.py` on the concept |
| 2026-11-16 | work/crimson_carpet_back.png, export/static.png | `crimson_carpet_cleanup.py` with the calmed centre, on the concept (#174, rule of #708) |
| 2026-11-16 | work/, export/ | `crimson_carpet_cleanup.py` with the repeated top and bottom borders and `export/slice.json` (section Any screen shape) |

Prompt of `concept/20261114-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Top-down view of a woven carpet table cloth, a crimson salon carpet with a wide border of small eight-point rosettes and stepped diamonds, back (the whole carpet, field and border, filling the image edge to edge) layer only, original geometric and floral pattern, calm low-contrast centre where game pieces will sit. Palette: deep crimson red, dark indigo, ivory and muted ochre.
```

Prompt of the two rejected middle-layer images, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Top-down view of a woven carpet table cloth, a crimson salon carpet with a wide border of small eight-point rosettes and stepped diamonds, middle (only a thin polished gold inlay line forming a rectangle with small gold corner ornaments, everything inside and outside the line left as the plain background) layer only, original geometric and floral pattern, calm low-contrast centre where game pieces will sit. Palette: deep crimson red, dark indigo, ivory and muted ochre.
```

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-14.
