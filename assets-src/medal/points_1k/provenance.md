# medal.points_1k

Medal art of set 1 (#178). Art slot for a proposed achievement medal for board points (`board_points`), the Board points chain of 8.3. No catalog entry, mission, target or reward exists for it: the slug follows the `<chain>_<target>` pattern of 8.3 so the step it was drawn for is clear, and the admin who creates the mission sets the real target. If the target differs, the folder is renamed before the medal is first published, as no id is published yet. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept generated on 2026-11-15 by an AI agent (Claude Code) running Codex CLI on the project owner's computer, as one sheet of six medals. W-3 clean-up and W-5 export by the same agent with `work/points_1k_cleanup.py`. No person has done W-3
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model `gpt-image-2` per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifest in the PNG names `ChatGPT` / `gpt-image` (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png, a shared sheet (sha256 `d7b87e33afca4b45282a953fbc2e84d33a7b37fb235bd36ab3085a2a25e6b541`), cell 5 of 2 rows x 3 columns
- Inputs uploaded to the generator: none; one prompt, one image, no edits, in a new Codex session (`01a1e077-9a04-7820-ba7b-f004aeff44d2`)
- Prompt: the style key of `docs/asset-workflow.md` 2.2 pasted verbatim, then a sheet prompt, quoted below. Section 2.3 has no medal template; this prompt follows the rules of 2.1
- Concept against the prompt: the cell shows a single faceted clear rock crystal with a pale champagne tint in a small brass claw setting. Run with `--pocket-min 100`: grey facets of the crystal are close to the background colour, and the default (20 px) cut them out as holes.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not applicable, no person is depicted
- Human work: none. Clean-up and export are scripted image operations run by an AI agent, so RC-5 stays open; the master and the script are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, religious symbols, weapons, casino cues or team hues seen in the art
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `common` (expected earn share, 5.3). No source yet; the mission that grants it is published after the medal lists it (`docs/admin.md`, Missions and medals note).

## Embedded C2PA manifest

`concept/20261115-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-15T18:29:21Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 18:28:16 UTC on 2026-11-15 (agent clock), shortly before `c2pa.created`.

This sheet also redraws `medal.points_1k` (the amber crystal of sheet C, cell 9, was too close to `color.role.teamB`) and `medal.chain_first` (sheet C, cell 5).

## Clean-up (W-3) and export (W-5)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/medal/points_1k/work/points_1k_cleanup.py points_1k assets-src/medal/points_1k/concept/20261115-01.png 2 3 5 assets-src/medal/points_1k/work assets-src/medal/points_1k/export --pocket-min 100
```

It printed `points_1k: background [187.0, 187.0, 188.0], pockets 0 px of at least 100, cell 5 of 2x3, box (488, 662, 768, 1077), canvas 256`. A second run into an empty folder wrote byte-identical files and the same line.

The script is `court_poet_cleanup.py` of set 1 changed for a sheet: the background is the median of a 6 px strip on all four edges; pixels within 30 of it (RGB distance) that connect to an edge are background, opened by 2 px; enclosed regions within 10 of the background colour of at least `--pocket-min` pixels (default 20) are holes, such as the inside of a ring; the rest is subject. The parts of the subject whose centre lies in this medal's cell of the grid, each of at least 400 px, are the medal. Its mask is eroded by 1 px and feathered (Gaussian sigma 1), edge pixels are unmixed from the grey, and the medal is cropped with 4 px around it as the master. The static is the master trimmed to its pixels, scaled with Lanczos on premultiplied alpha so its longer side is 240, and centred on a transparent 256 x 256 canvas (`docs/pass-missions-medals.md` 5.6).

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes | As above; canvas and pivot of `docs/pass-missions-medals.md` 5.6 |
| 2. Paint-over, AI artefacts | No | No paint-over. An artist checks the edge and the ornament |
| 3. Moving parts on their own layers | Not needed | Static item; medals are static (5.1) |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen |
| 5. Silhouette and smallest-size tests | Yes, by the agent | Looked at on `color.background.base` and `color.surface.default` at 256, 120 (`size.touchTargetMin`, the smallest display size of 5.6) and 64 px, and with the locked tint of 5.6 (the lock icon of #168 does not exist yet); the outline reads in each |

`npm run asset:check` takes only cosmetic item folders (`tools/asset-export`), so it was not run on this medal. By its size rule (ASTC 6x6, 16 bytes per block) the one 256 x 256 sprite is 43 x 43 blocks, 29,584 bytes; no medal limit is set (O-5).

Not done: upload of `export/static.png` for `medal/points_1k/static`. The admin asset upload takes only cosmetic item ids so far (`docs/admin.md` 5.2), and the cultural review above is open.

## Value and contrast (2026-11-16)

On the phone in the Art Bench (2026-11-16) this medal read pale and flat on `color.background.base` at 120 px next to the other medals: a milky, low-chroma crystal and setting with little difference between its darks and its lights. On 2026-11-16 Claude Code added one step to `work/points_1k_cleanup.py`, after the fringe removal and before the crop: each pixel's colour is scaled so its Rec. 709 luma `L` (0 to 1) becomes `L` to the power 1.7 (`VALUE_GAMMA`), which keeps the highlights and deepens the mid and dark tones without a hue shift, and then its distance from that luma is multiplied by 1.15 (`CHROMA_GAIN`) for a richer champagne and metal. No outline, glow or shadow was added; the outer shape and the alpha are unchanged. Run with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1:

```bash
python -I assets-src/medal/points_1k/work/points_1k_cleanup.py points_1k assets-src/medal/points_1k/concept/20261115-01.png 2 3 5 assets-src/medal/points_1k/work assets-src/medal/points_1k/export --pocket-min 100
```

It printed `points_1k: background [187.0, 187.0, 188.0], pockets 0 px of at least 100, cell 5 of 2x3, box (488, 662, 768, 1077), canvas 256, value gamma 1.7, chroma gain 1.15`. A second and a third run into empty folders wrote byte-identical files and the same line.

Measured by Claude Code on `export/static.png` resized to 120 px with Lanczos, WCAG relative luminance, alpha-weighted over the medal; "rim" is the outer 3 px of the medal's opaque area. The family median is over the 38 other medals in `assets-src/medal/`.

| Measure | Before | After | Family median |
|---|---|---|---|
| Mean luminance | 0.471 | 0.331 | 0.31 |
| Internal contrast, 90th over 10th percentile luminance | 4.96:1 | 9.43:1 | 6.14:1 |
| Mean against `color.background.base` | 9.99:1 | 7.33:1 | 6.95:1 |
| Mean against `color.surface.default` | 8.52:1 | 6.25:1 | 5.93:1 |
| Rim against `color.background.base` | 7.93:1 | 5.68:1 | 6.31:1 |
| Rim against `color.surface.default` | 6.77:1 | 4.84:1 | 5.39:1 |

The luminance contrast against both backgrounds falls, as the medal was among the brightest of the set and its paleness was a lack of dark tones and chroma, not a lack of light; every value stays above the 3:1 non-text minimum (WCAG 2.2 SC 1.4.11) that `docs/ui-motion.md` uses for fills and inside the range of the other medals. Not done: a look on the phone after the change and a human paint-over (RC-5).

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `d7b87e33afca4b45282a953fbc2e84d33a7b37fb235bd36ab3085a2a25e6b541` | 1254 x 1254 RGB, the generated sheet of six medals, unchanged; this medal is row 2, column 2 (cell 5) | Codex CLI `$imagegen`, copied byte for byte by Codex; byte-identical to the file in `~/.codex/generated_images/01a1e077-9a04-7820-ba7b-f004aeff44d2/` (checked with `cmp`) |
| work/points_1k_cleanup.py | `c82a357b0f334a9b104831e9729f42c1f0a296a74a901f87fad87bbc0647df0f` | W-3 and W-5 script | Written by Claude Code, 2026-11-15, from `court_poet_cleanup.py` of set 1; the value and chroma step added 2026-11-16 (Value and contrast below), so it now differs from the script of the other medals |
| work/points_1k_master.png | `6a503eff9c06a3b8b3e5f294d7006d05c8f3d3c64a443ee645e3f0f53a54c634` | 280 x 415 RGBA, the cut-out medal at the resolution of the sheet | `points_1k_cleanup.py` |
| export/static.png | `28470e049fa65b047b6de97353c45f65da044a4453c2ff617d5d1bf617466195` | 256 x 256 RGBA static, medal centred, longer side 240 | `points_1k_cleanup.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | Style key and sheet prompt, quoted below. Codex session `01a1e077-9a04-7820-ba7b-f004aeff44d2`; `c2pa.created` 2026-11-15T18:29:21Z |
| 2026-11-15 | work/points_1k_master.png, export/static.png | `points_1k_cleanup.py` on cell 5 of the concept |
| 2026-11-16 | work/points_1k_master.png, export/static.png | `points_1k_cleanup.py` again with the value and chroma step (value gamma 1.7, chroma gain 1.15) |

Prompt of `concept/20261115-01.png`, verbatim (the Codex call wraps it in `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as sheet_e.png. Do not use any API key and do not edit the image.`):

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Sheet of six separate achievement medal icons for a collection screen, arranged in an even 2 rows by 3 columns grid, front view, each medal symmetric left to right, fully inside its own cell with wide empty background between neighbours, nothing touching or overlapping, no ribbons, no stars, no crosses, no crescents. All medals share one family look: the same warm light, the same painted finish, chunky readable outer shapes that still read at thumbnail size. Each medal has its own outer silhouette. No fleur-de-lis, no heraldic lilies, no mountains, no landscapes. Top row, left to right: a clear faceted crystal medallion shaped like a cut gemstone, set in a thin platinum rim, with three ivory domino tiles standing side by side rising from short to tall in its centre; a round gold medallion with spread feathered gold wings, a deep garnet at the top and the three rising ivory domino tiles in relief; a round gold medallion with larger spread gold wings, a crest of three faceted gems (garnet, emerald, garnet) on top and the three rising domino tiles in relief on an ivory centre. Bottom row, left to right: a grand gold rosette medallion with spread gold wings, a rim set with emeralds, garnets and pearls, and the three rising domino tiles in relief on an ivory centre beneath a large faceted clear crystal; a single faceted clear rock crystal with a pale champagne tint standing in a small brass claw setting; two interlocked oval brass chain links lying on a small round polished brass disc.
```

RC-2 against this prompt and the Inputs line: it names no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-15.
