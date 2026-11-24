# medal.domino_5000

Medal art of set 1 (#178). Art slot for a proposed achievement medal for dominoes (`domino`), the Dominoes chain of 8.3. No catalog entry, mission, target or reward exists for it: the slug follows the `<chain>_<target>` pattern of 8.3 so the step it was drawn for is clear, and the admin who creates the mission sets the real target. If the target differs, the folder is renamed before the medal is first published, as no id is published yet. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept generated on 2026-11-15 by an AI agent (Claude Code) running Codex CLI on the project owner's computer, as one sheet of nine medals. W-3 clean-up and W-5 export by the same agent with `work/domino_5000_cleanup.py`. No person has done W-3
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model `gpt-image-2` per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifest in the PNG names `ChatGPT` / `gpt-image` (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png, a shared sheet (sha256 `b5d5f71e92a3af146c3442976bcec577b55b773c6a7e8b87c609241a92c46dc6`), cell 4 of 3 x 3
- Inputs uploaded to the generator: none; one prompt, one image, no edits, in a new Codex session (`01a1e06f-7455-75a0-858a-5fc9ab7d0037`)
- Prompt: the style key of `docs/asset-workflow.md` 2.2 pasted verbatim, then a sheet prompt, quoted below. Section 2.3 has no medal template; this prompt follows the rules of 2.1
- Concept against the prompt: the cell shows one upright ivory domino tile on a gold hexagonal plaque with spread wings and emeralds at its corners.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not applicable, no person is depicted
- Human work: none. Clean-up and export are scripted image operations run by an AI agent, so RC-5 stays open; the master and the script are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, religious symbols, weapons, casino cues or team hues seen in the art
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `legendary` (expected earn share, 5.3). No source yet; the mission that grants it is published after the medal lists it (`docs/admin.md`, Missions and medals note).

## Embedded C2PA manifest

`concept/20261115-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-15T18:20:46Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 18:19:22 UTC on 2026-11-15 (agent clock), shortly before `c2pa.created`.

Cells 5 (two interlocked gold rings) and 9 (an amber crystal whose orange sits next to `color.role.teamB`) of this sheet are not used; `medal.chain_first` and `medal.points_1k` were drawn again in sheet E.

## Clean-up (W-3) and export (W-5)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/medal/domino_5000/work/domino_5000_cleanup.py domino_5000 assets-src/medal/domino_5000/concept/20261115-01.png 3 3 4 assets-src/medal/domino_5000/work assets-src/medal/domino_5000/export
```

It printed `domino_5000: background [211.0, 211.0, 213.0], pockets 31056 px of at least 20, cell 4 of 3x3, box (41, 436, 382, 815), canvas 256`. A second run into an empty folder wrote byte-identical files and the same line.

The script is `court_poet_cleanup.py` of set 1 changed for a sheet: the background is the median of a 6 px strip on all four edges; pixels within 30 of it (RGB distance) that connect to an edge are background, opened by 2 px; enclosed regions within 10 of the background colour of at least `--pocket-min` pixels (default 20) are holes, such as the inside of a ring; the rest is subject. The parts of the subject whose centre lies in this medal's cell of the grid, each of at least 400 px, are the medal. Its mask is eroded by 1 px and feathered (Gaussian sigma 1), edge pixels are unmixed from the grey, and the medal is cropped with 4 px around it as the master. The static is the master trimmed to its pixels, scaled with Lanczos on premultiplied alpha so its longer side is 240, and centred on a transparent 256 x 256 canvas (`docs/pass-missions-medals.md` 5.6).

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes | As above; canvas and pivot of `docs/pass-missions-medals.md` 5.6 |
| 2. Paint-over, AI artefacts | No | No paint-over. An artist checks the edge and the ornament |
| 3. Moving parts on their own layers | Not needed | Static item; medals are static (5.1) |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen |
| 5. Silhouette and smallest-size tests | Yes, by the agent | Looked at on `color.background.base` and `color.surface.default` at 256, 120 (`size.touchTargetMin`, the smallest display size of 5.6) and 64 px, and with the locked tint of 5.6 (the lock icon of #168 does not exist yet); the outline reads in each |

`npm run asset:check` takes only cosmetic item folders (`tools/asset-export`), so it was not run on this medal. By its size rule (ASTC 6x6, 16 bytes per block) the one 256 x 256 sprite is 43 x 43 blocks, 29,584 bytes; no medal limit is set (O-5).

Not done: upload of `export/static.png` for `medal/domino_5000/static`. The admin asset upload takes only cosmetic item ids so far (`docs/admin.md` 5.2), and the cultural review above is open.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `b5d5f71e92a3af146c3442976bcec577b55b773c6a7e8b87c609241a92c46dc6` | 1254 x 1254 RGB, the generated sheet of nine medals, unchanged; this medal is row 2, column 1 (cell 4) | Codex CLI `$imagegen`, copied byte for byte by Codex; byte-identical to the file in `~/.codex/generated_images/01a1e06f-7455-75a0-858a-5fc9ab7d0037/` (checked with `cmp`) |
| work/domino_5000_cleanup.py | `a73b621f6087178a71d2494916865384d0f93516d0fa7b3cf5fed9c700c97a10` | W-3 and W-5 script | Written by Claude Code, 2026-11-15, from `court_poet_cleanup.py` of set 1; the same script for every medal of this PR |
| work/domino_5000_master.png | `602864b8324c596566c85a850467104b0cc9c153b12dda058f1bdc23b4ffe2e2` | 341 x 379 RGBA, the cut-out medal at the resolution of the sheet | `domino_5000_cleanup.py` |
| export/static.png | `40d0aafe2644293c8ffc6ff6fdc4bd84a448c73254a0218863115b19e16601d9` | 256 x 256 RGBA static, medal centred, longer side 240 | `domino_5000_cleanup.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | Style key and sheet prompt, quoted below. Codex session `01a1e06f-7455-75a0-858a-5fc9ab7d0037`; `c2pa.created` 2026-11-15T18:20:46Z |
| 2026-11-15 | work/domino_5000_master.png, export/static.png | `domino_5000_cleanup.py` on cell 4 of the concept |

Prompt of `concept/20261115-01.png`, verbatim (the Codex call wraps it in `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as sheet_c.png. Do not use any API key and do not edit the image.`):

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Sheet of nine separate achievement medal icons for a collection screen, arranged in an even 3 by 3 grid, front view, each medal symmetric left to right, fully inside its own cell with wide empty background between neighbours, nothing touching or overlapping, no ribbons, no stars, no crosses, no crescents. All medals share one family look: the same warm light, the same painted finish, chunky readable outer shapes that still read at thumbnail size. Each medal has its own outer silhouette. Row 1, left to right: one upright ivory domino tile with round black pips on a bronze hexagonal plaque; one upright ivory domino tile with round black pips on a silver hexagonal plaque with a laurel border; one upright ivory domino tile with round black pips on a gold hexagonal plaque with filigree and four small garnets. Row 2, left to right: one upright ivory domino tile with round black pips inside a gold hexagonal plaque with spread gold wings and emeralds at its corners; two interlocked polished brass rings; three interlocked bronze rings in a horizontal row. Row 3, left to right: a closed round loop of interlocked silver chain links around an ivory centre disc; a closed round loop of interlocked gold chain links set with small rubies around a gold centre holding a faceted emerald; a single faceted amber crystal in a small brass claw setting.
```

RC-2 against this prompt and the Inputs line: it names no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-15.
