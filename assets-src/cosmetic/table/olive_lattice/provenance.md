# table.olive_lattice

Static table of set 1 (#174): an olive carpet: an olive field with a lozenge lattice of small cream four-petal flowers with madder centres, a central lozenge medallion in ochre and cream lines with acanthus finials at its four points, cream and ochre guard lines, and an umber border of ochre-and-madder stepped diamond rosettes alternating with cream leaf sprays, inside the wood rim and gold inlay line of every table (`docs/art-direction.md` 5.4). The back layer is the painted concept `concept/20261122-01.png`, made with template P-5 of `docs/asset-workflow.md` section 2; the middle (gold inlay) and front (wood rim) layers are drawn by the clean-up script from colour tokens, as for `table.crimson_carpet`. Every line below is read from a file or a record, or marked as not done. The scripted carpet of 2026-11-15 was replaced on 2026-11-22; its script and settings stay in the repository history.

- artSource: ai_assisted
- Authors: W-1 concept generated on 2026-11-22 by an AI agent (Claude Code) through Codex CLI, in the project owner's ChatGPT account. W-3 layers and W-5 export on 2026-11-22 by Claude Code with `work/olive_lattice_cleanup.py`. No person has done W-3
- Generator: Codex CLI 0.171.0 built-in image generation (`$imagegen`, `docs/asset-workflow.md` 2.5), signed in with the project owner's ChatGPT account (`codex login status`: Logged in using ChatGPT); the plan was not read on 2026-11-22
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open, O-2). The account holder reads the terms in force on 2026-11-22 for the plan and records the date here
- Canonical concept: concept/20261122-01.png (back layer), used for W-3 since 2026-11-22
- Inputs uploaded to the generator: none; one prompt, one image, no edits
- Similarity check: not done (RC-3 open, O-7)
- Human work: none. The layers and the export are scripted image operations run by an AI agent, so RC-5 stays open; the layers and the script are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no script, letters, numbers, flags, emblems, religious symbols, casino cues or team hues seen in the static; the pattern is lozenges, four-petal flowers, acanthus finials, stepped diamonds and leaves in olive, umber, madder, cream and ochre (`docs/art-direction.md` 5.4)
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `common` (proposed on 2026-11-15 for the scripted pattern; not revisited for the painted art), source `shop:offer.table_olive_lattice`. The offer id is a placeholder until the shop record exists (V-15). `table.starter` stays the `common` default table.

## Layers (W-3) and export (W-5)

Run on 2026-11-22 by Claude Code with Python 3.12.7, NumPy 2.5.5, SciPy 1.18.1 and Pillow 13.0.0, from the repository root:

```bash
python -I assets-src/cosmetic/table/olive_lattice/work/olive_lattice_cleanup.py olive_lattice 132 assets-src/cosmetic/table/olive_lattice/concept/20261122-01.png assets-src/cosmetic/table/olive_lattice/work assets-src/cosmetic/table/olive_lattice/export
```

It printed `olive_lattice: trim (0, 0, 1535, 1024), widen strip 140 seam 402, stretch 1.003, crop (0.0, 0, 1815.0, 1024), edges depth 132 step 204 start 348 score 1.0333, canvas 1920x1080`. A second run into an empty folder wrote byte-identical files.

The concept is 1536 x 1024 (3:2), not 16:9 like the concept of `table.crimson_carpet`. Fitted as that script does (stretch at most 10%, crop the rest from top and bottom), it lost 37 to 41 px of the top and bottom border, so the rim covered half of those motifs and the ring was up to 60 px thinner at the top than at the sides (`docs/art-direction.md` 5.4: the ring has the same depth on all four sides). The script therefore widens the carpet before the fit (`widen_field`), so the border keeps its drawn thickness on all four sides:

- The field rows (inside the ring depth, at the concept's scale) get two strips of the field repeated, one on each side of the centre, mirror placed. The script tries strip widths of 30% to 70% of the missing width and seams between 15% and 45% of the width, scores each by how much the strip differs from the strip next to it (the field must repeat with that width there), and keeps the lowest. Chosen: strip 140 px of the concept, seams 402 px from each side, so the strip from 262 to 402 px and its mirror on the right are repeated.
- The top and bottom band rows get a strip of twice that width repeated at the centre; `periodic_edges` then rebuilds the middle of both bands from a run between the corner pieces, so that seam does not show.
- `fit_to_canvas` then fits the result with a horizontal factor of 1.003 and the box `(0.0, 0, 1815.0, 1024)`.

| Layer | File | How it is made |
|---|---|---|
| back | `work/olive_lattice_back.png` | The concept, its light margin trimmed (rows and columns where 98% of pixels are within 30 of the outermost ring's median colour; trim box `(0, 0, 1535, 1024)`), widened as above, fitted to 1920 x 1080. Then the centre is calmed as for `table.crimson_carpet`: inside an ellipse of 0.44 x 0.40 of the canvas (half axes), fading in over the outer 25%, the detail above a Gaussian base of sigma 12 px is reduced by 30% and the value by 25% (`CALM_GAIN = 0.75`; the 5% of `table.crimson_carpet` failed the readability check, below). Then the top and bottom borders are made to repeat (section Any screen shape) |
| middle | `work/olive_lattice_middle.png` | The gold inlay line of `table.crimson_carpet`, byte-identical: a 4 px line 54 px inside the edge with a small diamond at each corner, shaded from `color.gold.lo` through `color.gold.mid` and `color.gold.default` to `color.gold.hi`, drawn at 2x and box-filtered down |
| front | `work/olive_lattice_front.png` | The wood rim of `table.crimson_carpet`, byte-identical: a 40 px band with a 28 px corner radius, shaded from `color.wood.lo` through `color.wood.default` to `color.wood.hi`, drawn at 2x and box-filtered down |
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

The rule set by #708 for every table of set 1: the ivory face (median relative luminance of the 28 `tile.ivory` faces box-filtered to 32 x 64 px) against the 2v2 landscape board region (1516 x 521 of 1080 units, centred) has a contrast of at least 7:1 against the median and 4.5:1 against the 90th percentile. `work/olive_lattice_check.py` is `teal_lattice_check.py` of #708, copied unchanged:

```bash
python -I assets-src/cosmetic/table/olive_lattice/work/olive_lattice_check.py assets-src/cosmetic/table/olive_lattice assets-src/tile/ivory/export/face assets-src/effect/tile_shadow/export/static.png docs/design-tokens.json <out>
```

Output on 2026-11-22: `ivory face 0.743; worst scene static: centre median 0.059, 90th percentile 0.095; contrast 7.3:1 against the median (needs 7.0:1), 5.5:1 against the 90th percentile (needs 4.5:1): pass`

| Static | Ivory face | Centre median | 90th percentile | Against the median | Against the 90th percentile | Result |
|---|---|---|---|---|---|---|
| 2026-11-22, value -5% (crimson strength) | 0.743 | 0.090 | 0.135 | 5.7:1 | 4.3:1 | fail |
| 2026-11-22, value -25% (shipped) | 0.743 | 0.059 | 0.095 | 7.3:1 | 5.5:1 | pass |

The script also writes `check.png` and `check_x3.png` (nine ivory faces with their runtime shadow at 32 x 64 on the table at 828 px height, and its centre enlarged three times); the agent looked at both.

Team hues (`docs/art-direction.md` section 2): share of static pixels with saturation and value of at least 0.5 within 15 degrees of hue of `color.role.teamA` and `color.role.teamB`: teamA 0.00%, teamB 7.31%. Measured on 2026-11-22 with Pillow's HSV conversion, which gives `table.crimson_carpet` teamB 16.72% (17.42% recorded with the earlier measure); the teamB share counts ochre, rust and copper, which sit next to the team-B orange on the hue circle. Not a pass rule; the board review of section 9 decides.

`npm run asset:check -- assets-src/cosmetic/table/olive_lattice`: 1 sprites, ASTC 6x6 estimate 921,600 bytes, limit under 1,000,000 (owner decision 2026-11-16, static tables under 1 MB each): pass

Not done: W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Any screen shape

The rule is `docs/cosmetics.md` 8.4 and `docs/art-direction.md` 5.4 (owner decision of 2026-11-16), done in `work/olive_lattice_cleanup.py` (`periodic_edges`, `write_slice`) as for `table.crimson_carpet`.

- Ring depth 132 px at the 1920 x 1080 scale: the rim, the inlay, the carpet border and its guards to the inner edge of the innermost guard, the same on all four sides; read by the agent from the static. The lozenge medallion and its finials stay in the field; on 4:3 and narrower the points of the left and right finials come close to the field's ends.
- The top and bottom borders repeat. Each band is split into its light (a Gaussian blur of 120 px along the edge, made symmetric left to right) and its detail. The script tries steps of 150 to 480 px in steps of 6 and corner pieces of at least the depth plus 30 px and at most 520 px less a quarter step, scores each by how well the art matches at both ends of one step (windows of 12 and 48 px), how mirror-symmetric it is at the start, and how far the repeated band drifts from the art, and keeps the lowest score. Chosen: step 204 px, corner pieces 0 to 348 px and 1572 to 1920 px, score 1.0333. Between the corner pieces both bands are the run from 348 to 552 px repeated 6 times, joined by 32 px cross-fades, and they fade into the untouched field over the 16 px inside the ring.
- `export/slice.json`, `{"depth": 132, "step": 204}`, records the depth and the step in pixels of the static; `CosmeticContent` sets them as the sprite border of the static and of every layer frame, and `TableImage` draws the table at the screen's shape.
- Not done: the look of the repeated edges and of the widened field is not reviewed by a person, and no device check on the Art Bench was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261122-01.png | `e99491841dbe998a13543072485456047f0f2444e5ccfdbcb1f76099b7bd0b36` | 1536 x 1024, the generator's output, unchanged | Codex CLI, section Concept |
| work/olive_lattice_cleanup.py | `2548098d16feeb5f8b5c900aba0044894844fdf6b2137316f4749590cb382edc` | W-3 and W-5 script | `crimson_carpet_cleanup.py` copied on 2026-11-22 by Claude Code; `widen_field` and `strip_cost` added for the 3:2 concepts of this set (section Layers); `CALM_GAIN` set to 0.75 |
| work/olive_lattice_check.py | `34c3c89c3df7b87a1711fbe2451b7109eb1da623991e79755bfe7953d61b5421` | Readability check of #708 | `teal_lattice_check.py`, copied unchanged |
| work/olive_lattice_back.png | `6fad4c09fc056fb1e76dc16eb548291f01fee633492fe7032b953e8a3a1358c9` | 1920 x 1080 RGBA, back layer | `olive_lattice_cleanup.py` |
| work/olive_lattice_middle.png | `bf002a633c302a3d7b799119253bb820e9e5220232919a6c245c23d1d319cf1c` | 1920 x 1080 RGBA, middle layer, gold inlay | `olive_lattice_cleanup.py` |
| work/olive_lattice_front.png | `7c5379bb0f1fb5766c32a4b9d171e1df2ddb7a3a5c6f3b8296a42fd1c928192d` | 1920 x 1080 RGBA, front layer, wood rim | `olive_lattice_cleanup.py` |
| export/static.png | `136770e165da4ea780ef7e0f37ad65e3be9e7f0e7abdf46b1e35225db27b22ff` | 1920 x 1080 RGBA static, flattened layers | `olive_lattice_cleanup.py` |
| export/slice.json | `82c665abc6e7b52e89e7d715bd86019169d3a8f0596a61b1b9e4b75ed7cdd281` | `{"depth": 132, "step": 204}`: the ring depth and the edge step in pixels of the static (`docs/cosmetics.md` 8.4) | `olive_lattice_cleanup.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-22 | work/olive_lattice_back.png, work/olive_lattice_middle.png, work/olive_lattice_front.png, export/static.png, export/slice.json | `olive_lattice_cleanup.py` on the concept, replacing the scripted carpet of 2026-11-15 |

## Concept (2026-11-22)

Generated by Claude Code with `env -u OPENAI_API_KEY codex exec --skip-git-repo-check --sandbox workspace-write` in `concept/` (`docs/asset-workflow.md` 2.5). Codex session `01a20398-3a03-7b53-8772-b103bca7cdec`, prompt sent 2026-11-22T14:10:36Z. `concept/20261122-01.png` is a byte copy of the file Codex keeps under `~/.codex/generated_images/01a20398-3a03-7b53-8772-b103bca7cdec/`. The C2PA manifest was not read.

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-22 | concept/20261122-01.png | P-5 back layer with the style key, quoted below. SHA-256 `e99491841dbe998a13543072485456047f0f2444e5ccfdbcb1f76099b7bd0b36` |

Prompt of `concept/20261122-01.png`, verbatim (the instruction to save the file follows it in the Codex call):

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Top-down view of a woven carpet table cloth, an olive salon carpet with a low-contrast lozenge lattice of small four-petal flowers, a plain central lozenge medallion, and an umber border of stepped diamonds and small leaves between thin cream lines, back (the whole carpet, field and border, filling the image edge to edge) layer only, original geometric and floral pattern, calm low-contrast centre where game pieces will sit. Palette: olive green, umber brown, ochre, madder red and cream.
```

Concept against the prompt, looked at by the agent: an edge-to-edge carpet, field and border, with a calm centre; no text, letters, numbers, glow, drop shadow, casino cue or team hue seen. Not checked: generator artefacts at full size. The readability rule of #708 is checked on the static (section Readability check).
