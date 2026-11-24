# emoji.heart_eyes

Animated emoji of set 1 (#175), one cell of a concept sheet of nine emoji made with template P-6 of `docs/asset-workflow.md` section 2, animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept sheet generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifest in the PNG names ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png, row 2, column 1 of its 3 x 3 sheet. The sheet is shared: every emoji cut from it holds a byte-identical copy under the same name (sha256 below). This sheet is also the canonical style sheet of the whole set
- Inputs uploaded to the generator: none; one prompt, one image, no edits
- Prompt: the style key of section 2.2 pasted verbatim, then a sheet instruction in which each of the nine cells is one P-6 icon, with `{emotion}` = "delight, with eyes shaped as two small ruby hearts and an open smile" for this cell; quoted below. Deviation from section 2.3, decided for #175 to save generations: P-6 asks for a single icon per image, the sheet asks for nine in one image, each cell being one P-6 icon and one item
- Concept against the prompt: the cell matches its `{emotion}`. Not asked for but present in every cell of the set: ruby settings at the top and bottom of the rim, small gold knobs at the sides and a crimson inner band in the rim
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): no person is depicted; the face is a stylised expression painted on an ivory medallion
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no hands or gestures; no glow or drop shadow painted in; the expression is friendly and not taunting
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, group `love`, source `shop:offer.emoji_heart_eyes`. The offer id is a placeholder until the shop record exists (V-15). The repository names no emoji group yet besides `basic` in the example manifests, so set 1 proposes the groups joy, cheer, love, surprise, think, sorry, greet (`--group` of `npm run asset:add`). Static emoji are proposed at `common` and `rare`, animated ones at `rare` and above.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 2 animated details (heart_left, heart_right), so `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifest

`concept/20261115-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-15T17:27:34Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 17:26:36 |
| Codex session | `01a1e03f-2456-7a73-b68d-53629dd529e8` |

Codex saved the image under `~/.codex/generated_images/<session>/` and copied it into the agent's work folder; the two files are byte-identical (checked with `cmp`), and `concept/` holds a byte copy.

The nine cells of `concept/20261115-01.png`, row by row:

- row 1, column 1: `emoji.grin`
- row 1, column 2: `emoji.laugh`
- row 1, column 3: `emoji.joy_tears`
- row 2, column 1: `emoji.heart_eyes`
- row 2, column 2: `emoji.wow`
- row 2, column 3: `emoji.thinking`
- row 3, column 1: `emoji.crying`
- row 3, column 2: `emoji.wink`
- row 3, column 3: `emoji.star_eyes`

No whole sheet was rejected: six sheets were generated for the set, and the rejected cells above were left out.

## Clean-up (W-3)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root (also run for every emoji by `npm run asset:emoji`):

```bash
python -I assets-src/cosmetic/emoji/heart_eyes/work/heart_eyes_cleanup.py assets-src/cosmetic/emoji/heart_eyes
```

It printed `heart_eyes_cleanup: sheet 20261115-01.png cell (1, 0), background [206.0, 206.0, 206.0], pockets 0 px, medallion width 355 px at (210.0, 628.5), scale 1.0366, master 480, static from the loop`.

The script is the clean-up of the set 1 avatars (`avatar.jade_tassel`) adapted to a sheet: the flat grey is the median of a 6 px border strip; pixels within 30 of it that connect to any edge are background, opened by 2 px; enclosed grey pockets are background only when flat (spread under 6) so crystal stays; the subject parts whose centre lies in this item's cell are kept; the alpha is eroded by 1 px, feathered (sigma 1) and keyed along a 5 px band against the colour of the nearby subject, and edge pixels are unmixed from the grey. The medallion width is measured on the lower half of the largest part, so a hat, stars or hearts above it do not change it, and the item is scaled so the medallion is 368 px wide on a 480 x 480 master (92 px at 120), centred, and moved or shrunk only as far as needed to keep everything 8 px inside the canvas, so every emoji of the set shows the medallion at the same size.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas set by D-2 | As above; `work/heart_eyes_master.png` is 480 x 480; the loop renders every frame and the static from it at 120 x 120 (D-2 of the Epic 15 plan; the class canvas O-6 is open) |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Yes | Written by the loop script into `work/` (files below) |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen |
| 5. Silhouette and smallest-size tests | In part | The agent looked at the static at 120 px and at 2x on a dark backdrop; the round medallion and the expression read. The smallest display size of an emoji is open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-15 by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/emoji/heart_eyes/work/heart_eyes_loop.py assets-src/cosmetic/emoji/heart_eyes
```

It printed `heart_eyes_loop: parts 2 (heart_left, heart_right), 12 frames at 120x120, static frame 7`. It writes the layers below into `work/`, `export/anim/0001.png` to `0012.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. Every frame is rendered at 480 x 480 in premultiplied alpha and box-downsampled by 4.

| Choice | Value | Source |
|---|---|---|
| Canvas | 120 x 120 | D-2 of the Epic 15 plan |
| Frames | 12 | D-2 |
| Frame rate | 12 fps, a loop of 1 s | D-2. The rate is not stored with the loop (O-5, O-6) |
| Motion | The ruby heart eyes beat twice per loop, growing by up to 24%; the medallion hops by up to 1.5 px on each beat | Affine moves of the whole medallion about its bottom centre and of the parts about their own centre (BODY and PARTS in the script); px at 120 |
| Parts | `heart_left` (pulse), `heart_right` (pulse) | Each part is the master through a feathered mask: its colour (red, crystal, gold or dark) inside an ellipse set in the script. The parts only grow from their painted place, so the base layer keeps them and no hole opens. |
| Seamless loop | Mean channel change from frame 12 to frame 1 is 3.92 of 255, against 5.38 on average and 7.63 at most between the other neighbouring frames | Every motion is periodic in the 12 frames |
| Static | Frame 7 | The rest frame: no lift, tilt or squash, every part at its painted place and size. `export/static.png` is byte-identical to `export/anim/0007.png` |
| Size | 13 sprites, 83,200 bytes | `npm run asset:check`, ASTC 6x6 estimate (13 x 6,400), under the 100,000 bytes of #175 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `d5411efee91748223ffa258facec43952dc69135b3499ddcef5f4ba42b50a410` | 1254 x 1254 RGB, the concept sheet, unchanged | Codex CLI `$imagegen`, session `01a1e03f-2456-7a73-b68d-53629dd529e8` |
| work/heart_eyes_base.png | `f7d29591eced4e1dfdf7721efa38a4581c3653d7386bf5249d4173415cca2e96` | The master with the moving parts taken out (filled) or kept | `heart_eyes_loop.py` |
| work/heart_eyes_cleanup.py | `ce7d31485258af6b7e961d5dd54422e9524d4bafe8e8ee25f974d3d304c9e761` | W-3 script | Written by Claude Code, 2026-11-15; the same script for every emoji of the set with the item's sheet and cell |
| work/heart_eyes_heart_left.png | `48c45a966a9494885c86a67a0fb784e59acbdb42b8b199e83b6e9236475068e2` | Moving part `heart_left`: the master through its feathered mask | `heart_eyes_loop.py` |
| work/heart_eyes_heart_right.png | `53e621c2ce49539d67a85420c3756b3aa8b5980ea4ca4dff6ab19ea6fdd2f9b7` | Moving part `heart_right`: the master through its feathered mask | `heart_eyes_loop.py` |
| work/heart_eyes_loop.py | `ab4ddf3c4a09060a4184c6af3e98c64e7a42d75003ee2c059b315d46fe289e70` | W-4 and W-5 script | Written by Claude Code, 2026-11-15; the same script for every animated emoji of the set with the item's motion and parts |
| work/heart_eyes_master.png | `54105aa6f826e7fe7cb833c6f597a2a0e1908458e04764fb0d6151341330bd00` | 480 x 480 RGBA, the cut-out master on the 4x canvas | `heart_eyes_cleanup.py` |
| export/static.png | `41aa6eff794a36a256171921d49db4886c5a0d3dad0fbef156fb33c9c92b2774` | 120 x 120 RGBA static, loop frame 7 | `heart_eyes_loop.py` |
| export/anim/0001.png to 0012.png | per file in git | 120 x 120 RGBA loop frames | `heart_eyes_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | Style key and sheet of nine P-6 icons, quoted below. Sent 17:26:36 UTC; `c2pa.created` 2026-11-15T17:27:34Z |
| 2026-11-15 | work/, export/ | The scripts above |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. A sheet of nine reaction icons of one set, arranged in an even 3 x 3 grid with wide empty gaps of the same plain grey between them; nothing touches or crosses into another cell, every icon is whole with space around it, and there are no lines, numbers or labels between the cells. Each cell is one P-6 icon: Single expressive reaction icon: {emotion}, readable as a silhouette, one subject, no hands making any sign. Every icon is the same kind of object: a round face medallion of polished warm ivory, gently domed, set in a thin polished gold rim, seen from the front. The expression is painted on the ivory with simple bold features in deep brown lacquer: large clear eyes, expressive brows and mouth, soft rose cheeks. Accents are few and small: faceted ruby hearts, clear crystal drops, small polished gold stars. Same size, same light and same rendering in every cell. No hands or arms anywhere. The nine {emotion} values, row by row, left to right: 1, a big happy grin with eyes curved into happy arcs; 2, hearty laughter, mouth wide open, eyes shut tight; 3, laughing so hard that two crystal tears of joy spring from the eyes; 4, delight, with eyes shaped as two small ruby hearts and an open smile; 5, amazement, wide round eyes, raised brows and a small round open mouth; 6, thoughtful, eyes looking up to one side, one brow raised, a small pursed mouth; 7, sadness, crying, crystal tears running down both cheeks; 8, a friendly wink with a warm smile; 9, thrilled, eyes shaped as two small gold stars and a wide open smile.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

RC-2 against this prompt and the Inputs line: it names no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-15.
