# table.indigo_carpet

Static table of set 1 (#174), the indigo carpet. The back layer is a concept made with template P-5 of `docs/asset-workflow.md` section 2; the middle (gold inlay) and front (wood rim) layers are drawn by the clean-up script from colour tokens. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-14 while the work was done.

- artSource: ai_assisted
- Authors: W-1 back-layer concept generated in the project owner's second ChatGPT account on 2026-11-14 by an AI agent (Claude Code) driving the owner's browser. W-3 layers and W-5 export by the same agent with `work/indigo_carpet_cleanup.py`. No person has done W-3
- Generator: ChatGPT in the browser, Create image tool, model `gpt-image` as named by the C2PA manifest embedded in the concept (below). Account holder: the project owner, second ChatGPT account (the first one reached its image limit during #172), signed in to the second browser on the owner's computer. Plan: Free (shown under the account name in the ChatGPT sidebar on 2026-11-14)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-14 for the Free plan and records the date here
- Canonical concept: concept/20261114-01.png (back layer)
- Inputs uploaded to the generator: none; one prompt, one image, no edits, in a new conversation (`chatgpt.com/c/6af8544a...`, titled "Create Carpet Game Art" by ChatGPT)
- Prompt: P-5 with the style key of section 2.2 pasted verbatim and the back layer named, quoted below
- Concept against the prompt: Matches the prompt: a deep indigo field with faint vine scrolls, crimson and ochre floral rosettes and a central medallion, red corner spandrels, an ivory inner guard band and a crimson outer border of stepped medallions and vine scrolls. The image is 3:2, not 16:9, with a light margin of about 10 px, which the script trims.
- Similarity check: not done (RC-3 open, O-7)
- Human work: none. The layers and the export are scripted image operations run by an AI agent, so RC-5 stays open; the layers and the script are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no script, letters, numbers, flags, emblems, religious symbols, casino cues or team hues seen in the carpet; the pattern is geometric and floral (`docs/art-direction.md` 5.4)
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `rare`, source `shop:offer.table_indigo_carpet`. The offer id is a placeholder until the shop record exists (V-15). `table.starter` stays the `common` default table.

## Embedded C2PA manifest

`concept/20261114-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-14; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-14T12:30:31Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 12:30:15 UTC on 2026-11-14 (agent clock), shortly before `c2pa.created`.

## Middle and front layers: not generated

Not generated for this table. Two P-5 middle-layer prompts for `table.crimson_carpet` on the same day both returned a whole carpet instead of the line alone (its `provenance.md`, Middle and front layers), so the middle and front layers here are drawn by the clean-up script from colour tokens, the same as for that table.

## Layers (W-3) and export (W-5)

Run on 2026-11-14 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/table/indigo_carpet/work/indigo_carpet_cleanup.py indigo_carpet 138 assets-src/cosmetic/table/indigo_carpet/concept/20261114-01.png assets-src/cosmetic/table/indigo_carpet/work assets-src/cosmetic/table/indigo_carpet/export
```

It printed `indigo_carpet: trim (5, 10, 1528, 1014), stretch 1.100, crop (0, 30.8, 1523, 973.2), edges depth 138 step 396 start 366 score 1.9292, canvas 1920x1080` (run again on 2026-11-16 with the ring depth `138`). A second run into an empty folder wrote byte-identical files.

| Layer | File | How it is made |
|---|---|---|
| back | `work/indigo_carpet_back.png` | The concept, its light margin trimmed (rows and columns where 98% of pixels are within 30 of the outermost ring's median colour), fitted to the canvas. The trimmed carpet is 1523 x 1004 (about 3:2). The script stretches it horizontally by at most 10% (here exactly 1.100) and crops the rest from top and bottom (31 px of the concept on each side), so the medallions are 10% wider than drawn and the top and bottom border are slightly thinner than the side border. |
| middle | `work/indigo_carpet_middle.png` | A 4 px gold inlay line, a rectangle 54 px inside the canvas edge, with a small diamond (11 px half-diagonal) at each corner. Shaded with light from above across the line from `color.gold.lo` through `color.gold.mid` and `color.gold.default` to `color.gold.hi` (`docs/ui-motion.md`), drawn at 2x and box-filtered down. Everything else transparent |
| front | `work/indigo_carpet_front.png` | The wood rim: a 40 px band at the canvas edge with an opening of 28 px corner radius, shaded with light from above across its inner bevel from `color.wood.lo` through `color.wood.default` to `color.wood.hi`, drawn at 2x and box-filtered down. Everything inside transparent |
| static | `export/static.png` | back, middle and front composited in that order; 1920 x 1080 RGBA, fully opaque |

The middle and front layers are the same for every table of this set, so the layout of `docs/art-direction.md` 5.4 is the same on all of them. The table is static: `docs/cosmetics.md` section 4 forbids `layers` on a static table, so the layers stay in `work/` and only `export/static.png` is exported.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | Trim and fit as above. Canvas 1920 x 1080 (16:9), the frame of the reference prototype table; the class canvas and how the table covers 4:3 to 21:9 screens are open (O-6) |
| 2. Paint-over, AI artefacts | No | No paint-over; generator artefacts in the fine border motifs are not checked at full size. The rim is flat-shaded wood colour with no grain; an artist paints the rim and the inlay |
| 3. Moving parts on their own layers | Yes | back, middle and front in `work/`; no layer animates in this static item |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen |
| 5. Silhouette and smallest-size tests | Tile readability only | Tile faces and backs from the reference prototype renders (`table-v2/out/face_*.png`, `back_sema.png`, outside the repository) laid on the static at about the match size: centre field relative luminance median 0.022 (90th percentile 0.090); tile ivory 0.762; contrast of the ivory face against the median field 11.3:1. The field is calm except the central medallion, which is brighter than the field and sits under the first tiles played; the pips stay readable because they are on the ivory face, but an artist may want to tone the medallion down. |

`npm run asset:check -- assets-src/cosmetic/table/indigo_carpet`: 1 sprite, ASTC 6x6 estimate 921,600 bytes, no limit set for a static table (O-5).

Not done: W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Any screen shape (2026-11-16)

Owner decision of 2026-11-16, after the Art Bench check of #724 on a 20:9 phone: the rim, the inlay and the carpet border must show whole on every screen shape (16:9 to 21:9, 4:3, nearly square). Done on 2026-11-16 by Claude Code in `work/indigo_carpet_cleanup.py` (`periodic_edges`, `write_slice`), with no new canvas and no new layer; the rule is `docs/cosmetics.md` 8.4 and `docs/art-direction.md` 5.4.

- Ring depth 138 px at the 1920 x 1080 scale: the rim, the inlay, the carpet border and its guards, the same on all four sides. The corner spandrels stay in the field and may be cropped on a shape other than 16:9.
- The top and bottom borders now repeat. Each band is split into its light (a Gaussian blur of 120 px along the edge, made symmetric left to right) and its detail. The script tries steps of 150 to 480 px in steps of 6 (so a 1280 x 720 canvas keeps whole pixels) and corner pieces of at least the depth plus 30 px and at most 520 px less a quarter step, scores each by how well the art matches at both ends of one step (windows of 12 and 48 px), how mirror-symmetric it is at the start, and how far the repeated band drifts from the art, and keeps the lowest score. Chosen: step 396 px, corner pieces 0 to 366 px and 1554 to 1920 px, score 1.9292. Between the corner pieces both bands are the run from 366 to 762 px repeated 3 times, joined by 32 px cross-fades, and they fade into the untouched field over the 16 px inside the ring. The corner pieces and the side bands keep the art as it was; a motif unique to the middle of the top or bottom edge is replaced by the run.
- `export/slice.json`, `{"depth": 138, "step": 396}`, records the depth and the step in pixels of the static; `CosmeticContent` sets them as the sprite border of the static and of every layer frame, and `TableImage` draws the table at the screen's shape. Drawn by the same rule as `TableCloth` (scratch render, not committed): 16:9 steps +0 x1.000; 20:9 steps +1 x1.036; 21:9 steps +2 x0.929; 4:3 steps -1 x0.945 (steps gained or lost in the middle of the long edges, then the horizontal scale of the whole table).
- Readability check of #708 run again on the new static (section above): pass. `npm run asset:check` gives the same sprite count and estimate as before.
- Not done: the look of the repeated edges is not reviewed by a person; the device check on the Art Bench comes with the bench build of this change.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261114-01.png | `bbc74993fa5e7b9023521298c30bd72c2d4689a02937e957ac947ce0fd999736` | 1536 x 1024 RGB, the generator's download, unchanged | ChatGPT image viewer Download button, copied byte for byte |
| work/indigo_carpet_cleanup.py | `177ba86969d96ce4d799c74638b6a721a9ca7790484c6a2fb578f76b07e01d40` | W-3 and W-5 script | Written by Claude Code, 2026-11-14; the same script for both tables of this PR. `periodic_edges` and `write_slice` added on 2026-11-16 (section Any screen shape) |
| work/indigo_carpet_back.png | `e3b62881fd27513005dd0c55ecef4199bdd0bddfb084df136b107f2203fa9427` | 1920 x 1080 RGBA, back layer | `indigo_carpet_cleanup.py` |
| work/indigo_carpet_middle.png | `bf002a633c302a3d7b799119253bb820e9e5220232919a6c245c23d1d319cf1c` | 1920 x 1080 RGBA, middle layer | `indigo_carpet_cleanup.py` |
| work/indigo_carpet_front.png | `7c5379bb0f1fb5766c32a4b9d171e1df2ddb7a3a5c6f3b8296a42fd1c928192d` | 1920 x 1080 RGBA, front layer | `indigo_carpet_cleanup.py` |
| export/static.png | `7838a27d6306dd0befaa0f110017b4f0a264017f813df428c70b5cff54b063d7` | 1920 x 1080 RGBA static, flattened layers | `indigo_carpet_cleanup.py` |
| export/slice.json | `a44f9031214d585e8577ea7871e23caeabb347e591b6e2cd4bbb14de0651f4fe` | `{"depth": 138, "step": 396}`: the ring depth and the edge step in pixels of the static (`docs/cosmetics.md` 8.4) | `indigo_carpet_cleanup.py`, 2026-11-16 |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-14 | concept/20261114-01.png | P-5 back layer with the style key, the first prompt of its conversation, quoted below. Sent 12:30:15 UTC by Claude Code; `c2pa.created` 2026-11-14T12:30:31Z |
| 2026-11-14 | work/indigo_carpet_back.png, work/indigo_carpet_middle.png, work/indigo_carpet_front.png, export/static.png | `indigo_carpet_cleanup.py` on the concept |
| 2026-11-16 | work/, export/ | `indigo_carpet_cleanup.py` with the repeated top and bottom borders and `export/slice.json` (section Any screen shape) |

Prompt of `concept/20261114-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Top-down view of a woven carpet table cloth, an indigo salon carpet with a wide border of small stepped medallions and vine scrolls, back (the whole carpet, field and border, filling the image edge to edge) layer only, original geometric and floral pattern, calm low-contrast centre where game pieces will sit. Palette: deep indigo blue, dark crimson, ivory and muted ochre.
```

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-14.
