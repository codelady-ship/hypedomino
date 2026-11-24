# table.rosewood_medallion

Static table of set 1 (#174): a rosewood carpet: a rosewood-red field with a fine lozenge lattice of small ochre flowers, a central octagonal medallion with a sage vine band, dark corner spandrels with ivory peonies and sage leaves, a braided ivory guard, and a dark border of ivory eight-petal flowers alternating with rosewood octagonal rosettes and sage sprays, inside the wood rim and gold inlay line of every table (`docs/art-direction.md` 5.4). The back layer is the painted concept `concept/20261122-01.png`, made with template P-5 of `docs/asset-workflow.md` section 2; the middle (gold inlay) and front (wood rim) layers are drawn by the clean-up script from colour tokens, as for `table.crimson_carpet`. Every line below is read from a file or a record, or marked as not done. The scripted carpet of 2026-11-15 was replaced on 2026-11-22; its script and settings stay in the repository history.

- artSource: ai_assisted
- Authors: W-1 concept generated on 2026-11-22 by an AI agent (Claude Code) through Codex CLI, in the project owner's ChatGPT account. W-3 layers and W-5 export on 2026-11-22 by Claude Code with `work/rosewood_medallion_cleanup.py`. No person has done W-3
- Generator: Codex CLI 0.171.0 built-in image generation (`$imagegen`, `docs/asset-workflow.md` 2.5), signed in with the project owner's ChatGPT account (`codex login status`: Logged in using ChatGPT); the plan was not read on 2026-11-22
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open, O-2). The account holder reads the terms in force on 2026-11-22 for the plan and records the date here
- Canonical concept: concept/20261122-01.png (back layer), used for W-3 since 2026-11-22
- Inputs uploaded to the generator: none; one prompt, one image, no edits
- Similarity check: not done (RC-3 open, O-7)
- Human work: none. The layers and the export are scripted image operations run by an AI agent, so RC-5 stays open; the layers and the script are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no script, letters, numbers, flags, emblems, religious symbols, casino cues or team hues seen in the static; the pattern is lozenges, octagons, peonies, eight-petal flowers, octagonal rosettes and leaves in rosewood, dark brown, ivory, sage and ochre (`docs/art-direction.md` 5.4)
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `common` (proposed on 2026-11-15 for the scripted pattern; not revisited for the painted art), source `shop:offer.table_rosewood_medallion`. The offer id is a placeholder until the shop record exists (V-15). `table.starter` stays the `common` default table.

## Layers (W-3) and export (W-5)

Run on 2026-11-22 by Claude Code with Python 3.12.7, NumPy 2.5.5, SciPy 1.18.1 and Pillow 13.0.0, from the repository root:

```bash
python -I assets-src/cosmetic/table/rosewood_medallion/work/rosewood_medallion_cleanup.py rosewood_medallion 150 assets-src/cosmetic/table/rosewood_medallion/concept/20261122-01.png assets-src/cosmetic/table/rosewood_medallion/work assets-src/cosmetic/table/rosewood_medallion/export
```

It printed `rosewood_medallion: trim (3, 0, 1527, 1024), widen strip 92 seam 474, stretch 1.066, crop (0, 0.0, 1708, 1024.0), edges depth 150 step 312 start 336 score 1.2799, canvas 1920x1080`. A second run into an empty folder wrote byte-identical files.

The concept is 1536 x 1024 (3:2), not 16:9 like the concept of `table.crimson_carpet`. Fitted as that script does (stretch at most 10%, crop the rest from top and bottom), it lost 37 to 41 px of the top and bottom border, so the rim covered half of those motifs and the ring was up to 60 px thinner at the top than at the sides (`docs/art-direction.md` 5.4: the ring has the same depth on all four sides). The script therefore widens the carpet before the fit (`widen_field`), so the border keeps its drawn thickness on all four sides:

- The field rows (inside the ring depth, at the concept's scale) get two strips of the field repeated, one on each side of the centre, mirror placed. The script tries strip widths of 30% to 70% of the missing width and seams between 15% and 45% of the width, scores each by how much the strip differs from the strip next to it (the field must repeat with that width there), and keeps the lowest. Chosen: strip 92 px of the concept, seams 474 px from each side, so the strip from 382 to 474 px and its mirror on the right are repeated.
- The top and bottom band rows get a strip of twice that width repeated at the centre; `periodic_edges` then rebuilds the middle of both bands from a run between the corner pieces, so that seam does not show.
- `fit_to_canvas` then fits the result with a horizontal factor of 1.066 and the box `(0, 0.0, 1708, 1024.0)`.

| Layer | File | How it is made |
|---|---|---|
| back | `work/rosewood_medallion_back.png` | The concept, its light margin trimmed (rows and columns where 98% of pixels are within 30 of the outermost ring's median colour; trim box `(3, 0, 1527, 1024)`), widened as above, fitted to 1920 x 1080. Then the centre is calmed as for `table.crimson_carpet`: inside an ellipse of 0.44 x 0.40 of the canvas (half axes), fading in over the outer 25%, the detail above a Gaussian base of sigma 12 px is reduced by 30% and the value by 5%, the strength of `table.crimson_carpet`. Then the top and bottom borders are made to repeat (section Any screen shape) |
| middle | `work/rosewood_medallion_middle.png` | The gold inlay line of `table.crimson_carpet`, byte-identical: a 4 px line 54 px inside the edge with a small diamond at each corner, shaded from `color.gold.lo` through `color.gold.mid` and `color.gold.default` to `color.gold.hi`, drawn at 2x and box-filtered down |
| front | `work/rosewood_medallion_front.png` | The wood rim of `table.crimson_carpet`, byte-identical: a 40 px band with a 28 px corner radius, shaded from `color.wood.lo` through `color.wood.default` to `color.wood.hi`, drawn at 2x and box-filtered down |
| static | `export/static.png` | back, middle and front composited in that order; 1920 x 1080 RGBA, fully opaque |

The table is static: `docs/cosmetics.md` section 4 forbids `layers` on a static table, so the layers stay in `work/` and only `export/static.png` is exported.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | Trim, widen and fit as above. Canvas 1920 x 1080 (16:9), the canvas of the set-1 tables |
| 2. Paint-over, AI artefacts | No | No paint-over; generator artefacts in the fine border motifs are not checked at full size. The rim is flat-shaded wood colour with no grain; an artist paints the rim and the inlay |
| 3. Moving parts on their own layers | Yes | back, middle and front in `work/`; no layer animates in this static item |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | The agent looked at the static: no text, glow, drop shadow, rarity colour or team-hue decoration seen; the centre is calm (readability below) |
| 5. Silhouette and smallest-size tests | Tile readability only | Readability check below |

## Readability check (acceptance of #174)

The rule set by #708 for every table of set 1: the ivory face (median relative luminance of the 28 `tile.ivory` faces box-filtered to 32 x 64 px) against the 2v2 landscape board region (1516 x 521 of 1080 units, centred) has a contrast of at least 7:1 against the median and 4.5:1 against the 90th percentile. `work/rosewood_medallion_check.py` is `teal_lattice_check.py` of #708, copied unchanged:

```bash
python -I assets-src/cosmetic/table/rosewood_medallion/work/rosewood_medallion_check.py assets-src/cosmetic/table/rosewood_medallion assets-src/tile/ivory/export/face assets-src/effect/tile_shadow/export/static.png docs/design-tokens.json <out>
```

Output on 2026-11-22: `ivory face 0.743; worst scene static: centre median 0.045, 90th percentile 0.064; contrast 8.3:1 against the median (needs 7.0:1), 7.0:1 against the 90th percentile (needs 4.5:1): pass`

| Static | Ivory face | Centre median | 90th percentile | Against the median | Against the 90th percentile | Result |
|---|---|---|---|---|---|---|
| 2026-11-22 (shipped) | 0.743 | 0.045 | 0.064 | 8.3:1 | 7.0:1 | pass |

The script also writes `check.png` and `check_x3.png` (nine ivory faces with their runtime shadow at 32 x 64 on the table at 828 px height, and its centre enlarged three times); the agent looked at both.

Team hues (`docs/art-direction.md` section 2): share of static pixels with saturation and value of at least 0.5 within 15 degrees of hue of `color.role.teamA` and `color.role.teamB`: teamA 0.00%, teamB 14.31%. Measured on 2026-11-22 with Pillow's HSV conversion, which gives `table.crimson_carpet` teamB 16.72% (17.42% recorded with the earlier measure); the teamB share counts ochre, rust and copper, which sit next to the team-B orange on the hue circle. Not a pass rule; the board review of section 9 decides.

`npm run asset:check -- assets-src/cosmetic/table/rosewood_medallion`: 1 sprites, ASTC 6x6 estimate 921,600 bytes, limit under 1,000,000 (owner decision 2026-11-16, static tables under 1 MB each): pass

Not done: W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Any screen shape

The rule is `docs/cosmetics.md` 8.4 and `docs/art-direction.md` 5.4 (owner decision of 2026-11-16), done in `work/rosewood_medallion_cleanup.py` (`periodic_edges`, `write_slice`) as for `table.crimson_carpet`.

- Ring depth 150 px at the 1920 x 1080 scale: the rim, the inlay, the carpet border and its guards to the inner edge of the innermost guard, the same on all four sides; read by the agent from the static. The corner spandrels stay in the field and may be cropped on a shape other than 16:9, as on `table.indigo_carpet`. The tips of the top and bottom spandrel arcs touch the inner guard inside the 16 px fade of the ring, so they show there as short ochre marks, as drawn in the concept.
- The top and bottom borders repeat. Each band is split into its light (a Gaussian blur of 120 px along the edge, made symmetric left to right) and its detail. The script tries steps of 150 to 480 px in steps of 6 and corner pieces of at least the depth plus 30 px and at most 520 px less a quarter step, scores each by how well the art matches at both ends of one step (windows of 12 and 48 px), how mirror-symmetric it is at the start, and how far the repeated band drifts from the art, and keeps the lowest score. Chosen: step 312 px, corner pieces 0 to 336 px and 1584 to 1920 px, score 1.2799. Between the corner pieces both bands are the run from 336 to 648 px repeated 4 times, joined by 32 px cross-fades, and they fade into the untouched field over the 16 px inside the ring.
- `export/slice.json`, `{"depth": 150, "step": 312}`, records the depth and the step in pixels of the static; `CosmeticContent` sets them as the sprite border of the static and of every layer frame, and `TableImage` draws the table at the screen's shape.
- Not done: the look of the repeated edges and of the widened field is not reviewed by a person, and no device check on the Art Bench was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261122-01.png | `e189795c4333fd37ed51e2d60f4f69108a972dff8a7db6c0f21e27d698c063d7` | 1536 x 1024, the generator's output, unchanged | Codex CLI, section Concept |
| work/rosewood_medallion_cleanup.py | `99705a30a4747f141ff0aa6f7ef6d2a4081c4a6d8b07babba8790916a3d0fb98` | W-3 and W-5 script | `crimson_carpet_cleanup.py` copied on 2026-11-22 by Claude Code; `widen_field` and `strip_cost` added for the 3:2 concepts of this set (section Layers) |
| work/rosewood_medallion_check.py | `34c3c89c3df7b87a1711fbe2451b7109eb1da623991e79755bfe7953d61b5421` | Readability check of #708 | `teal_lattice_check.py`, copied unchanged |
| work/rosewood_medallion_back.png | `91a710517d8f0057134e66b3cf939915469b8591e080576dc8b3d25cae7a5ef5` | 1920 x 1080 RGBA, back layer | `rosewood_medallion_cleanup.py` |
| work/rosewood_medallion_middle.png | `bf002a633c302a3d7b799119253bb820e9e5220232919a6c245c23d1d319cf1c` | 1920 x 1080 RGBA, middle layer, gold inlay | `rosewood_medallion_cleanup.py` |
| work/rosewood_medallion_front.png | `7c5379bb0f1fb5766c32a4b9d171e1df2ddb7a3a5c6f3b8296a42fd1c928192d` | 1920 x 1080 RGBA, front layer, wood rim | `rosewood_medallion_cleanup.py` |
| export/static.png | `dbcaa0002ba07405470b9828d8fbefa93591b6380f76f335858d269a7a08a097` | 1920 x 1080 RGBA static, flattened layers | `rosewood_medallion_cleanup.py` |
| export/slice.json | `2417bb4bfce2095974b29c877a4d139ba91880975beb37a44117fda66ec7a254` | `{"depth": 150, "step": 312}`: the ring depth and the edge step in pixels of the static (`docs/cosmetics.md` 8.4) | `rosewood_medallion_cleanup.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-22 | work/rosewood_medallion_back.png, work/rosewood_medallion_middle.png, work/rosewood_medallion_front.png, export/static.png, export/slice.json | `rosewood_medallion_cleanup.py` on the concept, replacing the scripted carpet of 2026-11-15 |

## Concept (2026-11-22)

Generated by Claude Code with `env -u OPENAI_API_KEY codex exec --skip-git-repo-check --sandbox workspace-write` in `concept/` (`docs/asset-workflow.md` 2.5). Codex session `01a20399-c83b-74e2-98a1-855c043f360e`, prompt sent 2026-11-22T14:12:17Z. `concept/20261122-01.png` is a byte copy of the file Codex keeps under `~/.codex/generated_images/01a20399-c83b-74e2-98a1-855c043f360e/`. The C2PA manifest was not read.

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-22 | concept/20261122-01.png | P-5 back layer with the style key, quoted below. SHA-256 `e189795c4333fd37ed51e2d60f4f69108a972dff8a7db6c0f21e27d698c063d7` |

Prompt of `concept/20261122-01.png`, verbatim (the instruction to save the file follows it in the Codex call):

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Top-down view of a woven carpet table cloth, a rosewood salon carpet with a fine lozenge lattice, a plain octagonal central medallion, dark corner spandrels, and a dark border of eight-petal flowers alternating with octagonal rosettes, back (the whole carpet, field and border, filling the image edge to edge) layer only, original geometric and floral pattern, calm low-contrast centre where game pieces will sit. Palette: rosewood red-brown, ink black, ochre, ivory and muted sage.
```

Concept against the prompt, looked at by the agent: an edge-to-edge carpet, field and border, with a calm centre; no text, letters, numbers, glow, drop shadow, casino cue or team hue seen. Not checked: generator artefacts at full size. The readability rule of #708 is checked on the static (section Readability check).
