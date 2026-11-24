# medal.hands_1000

Medal art of set 1 (#178). Art slot for a proposed achievement medal for hands won (`hand_won`). No catalog entry, mission, target or reward exists for it: the slug follows the `<chain>_<target>` pattern of 8.3 so the step it was drawn for is clear, and the admin who creates the mission sets the real target. If the target differs, the folder is renamed before the medal is first published, as no id is published yet. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done; the pip repaint (#178) recorded on 2026-11-16 the same way.

- artSource: ai_assisted
- Authors: W-1 concept generated on 2026-11-15 by an AI agent (Claude Code) running Codex CLI on the project owner's computer, as one sheet of nine medals. W-3 clean-up and W-5 export by the same agent with `work/hands_1000_cleanup.py`; W-3 pip repaint on 2026-11-16 by Claude Code with the same script and `work/hands_1000_pips.json`. No person has done W-3
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model `gpt-image-2` per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifest in the PNG names `ChatGPT` / `gpt-image` (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png, a shared sheet (sha256 `5637966b48b8f1c57a0965226320795572c27c26a154a5147cba1896623134d8`), cell 7 of 3 x 3
- Inputs uploaded to the generator: none; one prompt, one image, no edits, in a new Codex session (`01a1e06c-bbd6-7a83-b248-a154e700f93e`)
- Prompt: the style key of `docs/asset-workflow.md` 2.2 pasted verbatim, then a sheet prompt, quoted below. Section 2.3 has no medal template; this prompt follows the rules of 2.1
- Concept against the prompt: the cell shows a fan of five ivory domino tiles on a silver fan-shaped mount. The pips of the five tiles do not all follow the fixed layouts of `docs/tiles.md`. Counted on the master on 2026-11-16 (dark blobs per upper half, left to right): 1, 2, 3, 4, 5, so no half carries more than six, but the 2 and the 3 lie on the rising diagonal (the mirror of the layout) and the five is not the corners and the centre. The lower halves are blank where the mount leaves them visible. The concept stays as generated; the pips are repainted by script in W-3 (below), not by an artist
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not applicable, no person is depicted
- Human work: none. Clean-up and export are scripted image operations run by an AI agent, so RC-5 stays open; the master and the script are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, religious symbols, weapons, casino cues or team hues seen in the art
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `rare` (expected earn share, 5.3). No source yet; the mission that grants it is published after the medal lists it (`docs/admin.md`, Missions and medals note).

## Embedded C2PA manifest

`concept/20261115-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-15T18:17:31Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 18:16:23 UTC on 2026-11-15 (agent clock), shortly before `c2pa.created`.

## Clean-up (W-3) and export (W-5)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/medal/hands_1000/work/hands_1000_cleanup.py hands_1000 assets-src/medal/hands_1000/concept/20261115-01.png 3 3 7 assets-src/medal/hands_1000/work assets-src/medal/hands_1000/export --pips assets-src/medal/hands_1000/work/hands_1000_pips.json
```

Run again on 2026-11-16 with the same versions and `--pips`. It printed `hands_1000: background [224.0, 224.0, 224.0], pockets 45948 px of at least 20, cell 7 of 3x3, box (25, 920, 393, 1193), canvas 256, pips 15 erased, 15 painted, values 1-2-3-4-5`. A second run into an empty folder wrote byte-identical files and the same line. Without `--pips` the script still writes the master and static of 2026-11-15 byte for byte (checked on 2026-11-16).

The script is `court_poet_cleanup.py` of set 1 changed for a sheet: the background is the median of a 6 px strip on all four edges; pixels within 30 of it (RGB distance) that connect to an edge are background, opened by 2 px; enclosed regions within 10 of the background colour of at least `--pocket-min` pixels (default 20) are holes, such as the inside of a ring; the rest is subject. The parts of the subject whose centre lies in this medal's cell of the grid, each of at least 400 px, are the medal. Its mask is eroded by 1 px and feathered (Gaussian sigma 1), edge pixels are unmixed from the grey, and the medal is cropped with 4 px around it as the master. The static is the master trimmed to its pixels, scaled with Lanczos on premultiplied alpha so its longer side is 240, and centred on a transparent 256 x 256 canvas (`docs/pass-missions-medals.md` 5.6).

### Pip repaint (2026-11-16)

`--pips` repaints the pips of the master after the cut-out, so every visible half shows the layout of `docs/tiles.md` section 1 (Pip layout) in the frame of its tile, far end up: 1 the centre; 2 and 3 on the falling diagonal from the upper left; 4 the corners; 5 the corners and the centre; 6 two columns of three. `work/hands_1000_pips.json` holds, for each tile, the centre of its upper half, its tilt and its value, and for the medal the grid step across and along the tile, the pip used as the stamp and the disc radius. The agent measured the centres, tilts and steps on the old pips of the concept; the step keeps the painted proportions of these tiles (taller halves than the game tile), so it is not the 0.27 W of the game face.

The script finds the old pips (dark blobs, mean RGB below 70, at least `pipMinPixels` px, within 1.5 grid steps of a half centre), fills a disc of `pipRadius` px around each with a harmonic fill (800 Jacobi steps from the 2 px ring around the discs), then paints each new pip by adding the difference between one old painted pip (the centre pip of the middle tile) and its fill, shifted bilinearly to the sub-pixel grid position. Every pip is the generator's own painted pip, moved; none is drawn from scratch. Left to right the upper halves now show 1, 2, 3, 4 and 5, the values the concept meant; the lower halves stay blank where visible, so the fan reads as 1-0 to 5-0, five different tiles of the set.

The agent looked at the master at 4x and the old one at 3x (no trace of the old pips, no seam in the ivory) and at the static at 256, 120 and 64 px on `color.background.base` and at 256 and 64 px on `color.surface.default`: the pips read as separate dots at 256 and 120 px; at 64 px the 4 and the 5 read as small clusters of dots.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes | As above; canvas and pivot of `docs/pass-missions-medals.md` 5.6 |
| 2. Paint-over, AI artefacts | Pips only, by script | Pips repainted to the layouts by the agent's script (above); no other paint-over. An artist checks the edge, the ornament and the pips |
| 3. Moving parts on their own layers | Not needed | Static item; medals are static (5.1) |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen |
| 5. Silhouette and smallest-size tests | Yes, by the agent | Looked at on `color.background.base` and `color.surface.default` at 256, 120 (`size.touchTargetMin`, the smallest display size of 5.6) and 64 px, and with the locked tint of 5.6 (the lock icon of #168 does not exist yet); the outline reads in each |

`npm run asset:check` takes only cosmetic item folders (`tools/asset-export`), so it was not run on this medal. By its size rule (ASTC 6x6, 16 bytes per block) the one 256 x 256 sprite is 43 x 43 blocks, 29,584 bytes; no medal limit is set (O-5).

Not done: upload of `export/static.png` for `medal/hands_1000/static`. The admin asset upload takes only cosmetic item ids so far (`docs/admin.md` 5.2), and the cultural review above is open.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `5637966b48b8f1c57a0965226320795572c27c26a154a5147cba1896623134d8` | 1254 x 1254 RGB, the generated sheet of nine medals, unchanged; this medal is row 3, column 1 (cell 7) | Codex CLI `$imagegen`, copied byte for byte by Codex; byte-identical to the file in `~/.codex/generated_images/01a1e06c-bbd6-7a83-b248-a154e700f93e/` (checked with `cmp`) |
| work/hands_1000_cleanup.py | `6ecf3ae437aae1d01effea8b0e3d0266c260026c8d6612a09c3e66d5ef2dd66c` | W-3 and W-5 script | Written by Claude Code, 2026-11-15, from `court_poet_cleanup.py` of set 1, the same script for every medal of set 1 (`a73b621f...`); the `--pips` step added on 2026-11-16, the same script for `hands_1000` and `hands_10000` |
| work/hands_1000_pips.json | `862668f3d2a5f5a03ff2057ac577d537e5537f66a94c1394f97612fb63d1dca8` | Tile centres, tilts, values, grid step and stamp of the pip repaint | Written by Claude Code, 2026-11-16, from measurements on the master |
| work/hands_1000_master.png | `500cbb44378baff92ad6fa4b74f7669b9c593ebc81782374f50e144230d7b7e9` | 368 x 273 RGBA, the cut-out medal at the resolution of the sheet, pips repainted | `hands_1000_cleanup.py` |
| export/static.png | `3bd27e31a211763cf44c9cabde535dfc764e6be9cf0d54dbf396fd06ccf2c891` | 256 x 256 RGBA static, medal centred, longer side 240 | `hands_1000_cleanup.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | Style key and sheet prompt, quoted below. Codex session `01a1e06c-bbd6-7a83-b248-a154e700f93e`; `c2pa.created` 2026-11-15T18:17:31Z |
| 2026-11-15 | work/hands_1000_master.png, export/static.png | `hands_1000_cleanup.py` on cell 7 of the concept |
| 2026-11-16 | work/hands_1000_master.png, export/static.png | `hands_1000_cleanup.py --pips work/hands_1000_pips.json` on cell 7 of the concept: pips repainted to the layouts of `docs/tiles.md` (#178) |

Prompt of `concept/20261115-01.png`, verbatim (the Codex call wraps it in `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as sheet_b.png. Do not use any API key and do not edit the image.`):

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Sheet of nine separate achievement medal icons for a collection screen, arranged in an even 3 by 3 grid, front view, each medal symmetric left to right, fully inside its own cell with wide empty background between neighbours, nothing touching or overlapping, no ribbons, no stars, no crosses, no crescents. All medals share one family look: the same warm light, the same painted finish, chunky readable outer shapes that still read at thumbnail size. Each medal has its own outer silhouette. Row 1, left to right: a tall arch-shaped silver plaque with a relief of stair steps rising to a gold laurel sprig and an amber gem; a tall arch-shaped gold plaque with stair step relief, scroll ornaments on both sides and a large faceted emerald at the apex; a brass hourglass with ivory sand standing inside a round bronze ring. Row 2, left to right: a silver hourglass inside an octagonal gold frame; a gold hourglass with gem-capped ends inside a scalloped ivory and gold rosette; a fan of three ivory domino tiles with round black pips on a small bronze half-round mount. Row 3, left to right: a fan of five ivory domino tiles with round black pips on a silver fan-shaped mount; a fan of seven ivory domino tiles with round black pips on a gold fan-shaped mount edged with small emeralds; one upright ivory domino tile with round black pips on a small round brass disc.
```

RC-2 against this prompt and the Inputs line: it names no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-15.
