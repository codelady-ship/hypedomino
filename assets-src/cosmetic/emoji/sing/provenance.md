# emoji.sing

Animated emoji of set 1 (#175), one cell of a concept sheet of nine emoji made with template P-6 of `docs/asset-workflow.md` section 2, animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept sheet generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifest in the PNG names ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-05.png, row 1, column 1 of its 3 x 3 sheet. The sheet is shared: every emoji cut from it holds a byte-identical copy under the same name (sha256 below). The sheet was made from the canonical style sheet of the set, `assets-src/cosmetic/emoji/grin/concept/20261115-01.png` (sha256 `d5411efee91748223ffa258facec43952dc69135b3499ddcef5f4ba42b50a410`)
- Inputs uploaded to the generator: `assets-src/cosmetic/emoji/grin/concept/20261115-01.png`, our own canonical style sheet of this set, attached with `codex exec --image` as the style reference. Nothing else was uploaded
- Prompt: the style key of section 2.2 pasted verbatim, then a sheet instruction in which each of the nine cells is one P-6 icon, with `{emotion}` = "singing happily, eyes closed, a round open mouth, two small gold musical notes floating beside the medallion" for this cell; quoted below. Deviation from section 2.3, decided for #175 to save generations: P-6 asks for a single icon per image, the sheet asks for nine in one image, each cell being one P-6 icon and one item
- Concept against the prompt: the cell matches its `{emotion}`. Not asked for but present in every cell of the set: ruby settings at the top and bottom of the rim, small gold knobs at the sides and a crimson inner band in the rim
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): no person is depicted; the face is a stylised expression painted on an ivory medallion
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no hands or gestures; no glow or drop shadow painted in; the expression is friendly and not taunting
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, group `cheer`, source `shop:offer.emoji_sing`. The offer id is a placeholder until the shop record exists (V-15). The repository names no emoji group yet besides `basic` in the example manifests, so set 1 proposes the groups joy, cheer, love, surprise, think, sorry, greet (`--group` of `npm run asset:add`). Static emoji are proposed at `common` and `rare`, animated ones at `rare` and above.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 2 animated details (note_big, note_small), so `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifest

`concept/20261115-05.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-15T17:36:14Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 17:34:48 |
| Codex session | `01a1e046-a56b-7b23-86aa-1b458eff65dc` |

Codex saved the image under `~/.codex/generated_images/<session>/` and copied it into the agent's work folder; the two files are byte-identical (checked with `cmp`), and `concept/` holds a byte copy.

The nine cells of `concept/20261115-05.png`, row by row:

- row 1, column 1: `emoji.sing`
- row 1, column 2: `emoji.touched`
- row 1, column 3: `emoji.smitten`
- row 2, column 1: rejected (silliness, the face upside down): the upside-down face does not read as a face at small size
- row 2, column 2: `emoji.grimace`
- row 2, column 3: `emoji.dreamy`
- row 3, column 1: `emoji.excited`
- row 3, column 2: rejected (laughing with the face turned sideways): the sideways face does not read at small size
- row 3, column 3: `emoji.grateful`

No whole sheet was rejected: six sheets were generated for the set, and the rejected cells above were left out.

## Clean-up (W-3)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root (also run for every emoji by `npm run asset:emoji`):

```bash
python -I assets-src/cosmetic/emoji/sing/work/sing_cleanup.py assets-src/cosmetic/emoji/sing
```

It printed `sing_cleanup: sheet 20261115-05.png cell (0, 0), background [210.0, 210.0, 210.0], pockets 0 px, medallion width 354 px at (208.5, 212.5), scale 1.0395, master 480, static from the loop`.

The script is the clean-up of the set 1 avatars (`avatar.jade_tassel`) adapted to a sheet: the flat grey is the median of a 6 px border strip; pixels within 30 of it that connect to any edge are background, opened by 2 px; enclosed grey pockets are background only when flat (spread under 6) so crystal stays; the subject parts whose centre lies in this item's cell are kept; the alpha is eroded by 1 px, feathered (sigma 1) and keyed along a 5 px band against the colour of the nearby subject, and edge pixels are unmixed from the grey. The medallion width is measured on the lower half of the largest part, so a hat, stars or hearts above it do not change it, and the item is scaled so the medallion is 368 px wide on a 480 x 480 master (92 px at 120), centred, and moved or shrunk only as far as needed to keep everything 8 px inside the canvas, so every emoji of the set shows the medallion at the same size.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas set by D-2 | As above; `work/sing_master.png` is 480 x 480; the loop renders every frame and the static from it at 120 x 120 (D-2 of the Epic 15 plan; the class canvas O-6 is open) |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Yes | Written by the loop script into `work/` (files below) |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen |
| 5. Silhouette and smallest-size tests | In part | The agent looked at the static at 120 px and at 2x on a dark backdrop; the round medallion and the expression read. The smallest display size of an emoji is open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-15 by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/emoji/sing/work/sing_loop.py assets-src/cosmetic/emoji/sing
```

It printed `sing_loop: parts 2 (note_big, note_small), 12 frames at 120x120, static frame 7`. It writes the layers below into `work/`, `export/anim/0001.png` to `0012.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. Every frame is rendered at 480 x 480 in premultiplied alpha and box-downsampled by 4.

| Choice | Value | Source |
|---|---|---|
| Canvas | 120 x 120 | D-2 of the Epic 15 plan |
| Frames | 12 | D-2 |
| Frame rate | 12 fps, a loop of 1 s | D-2. The rate is not stored with the loop (O-5, O-6) |
| Motion | The two gold notes bob by up to 2.5 px in turn; the medallion sways by up to 5 degrees | Affine moves of the whole medallion about its bottom centre and of the parts about their own centre (BODY and PARTS in the script); px at 120 |
| Parts | `note_big` (bob), `note_small` (bob) | Each part is the master through a feathered mask: its colour (red, crystal, gold or dark) inside an ellipse set in the script. The place a moving part leaves in the base layer is filled from its surroundings (normalised Gaussian fill) for note_big, note_small. |
| Seamless loop | Mean channel change from frame 12 to frame 1 is 15.81 of 255, against 11.56 on average and 16.10 at most between the other neighbouring frames | Every motion is periodic in the 12 frames |
| Static | Frame 7 | The rest frame: no lift, tilt or squash, every part at its painted place and size. `export/static.png` is byte-identical to `export/anim/0007.png` |
| Size | 13 sprites, 83,200 bytes | `npm run asset:check`, ASTC 6x6 estimate (13 x 6,400), under the 100,000 bytes of #175 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-05.png | `cb5eb2c482de52f8eb44924adae50fd6bd59636742c9742edb0ab81bf8ba7cc7` | 1254 x 1254 RGB, the concept sheet, unchanged | Codex CLI `$imagegen`, session `01a1e046-a56b-7b23-86aa-1b458eff65dc` |
| work/sing_base.png | `b2acd14ec9947e3f04b6f3c942e15029835041ac2d08cc4aac0002d36985a81d` | The master with the moving parts taken out (filled) or kept | `sing_loop.py` |
| work/sing_cleanup.py | `2c9e2c98c28c0b16f9a74e6979ea3cf6ed57d4599a64b30e9b772586a7780025` | W-3 script | Written by Claude Code, 2026-11-15; the same script for every emoji of the set with the item's sheet and cell |
| work/sing_loop.py | `5c31b87cb5b0bdc20776800562a479f086af2f4536efffce416865995f192874` | W-4 and W-5 script | Written by Claude Code, 2026-11-15; the same script for every animated emoji of the set with the item's motion and parts |
| work/sing_master.png | `f4f3aa2dfe37787d98aa4a5f0a60d3050a7dfd49cd3bdd98927e705c9fa559ac` | 480 x 480 RGBA, the cut-out master on the 4x canvas | `sing_cleanup.py` |
| work/sing_note_big.png | `975d1d70827bc2bf753b98e32ebd3e622213c2d5716cef9f1c20770b8e6d8f8d` | Moving part `note_big`: the master through its feathered mask | `sing_loop.py` |
| work/sing_note_small.png | `0ec88e83a75faf5a6a57c259d994f37e9d406e729374a21528a9e93f0e9ab267` | Moving part `note_small`: the master through its feathered mask | `sing_loop.py` |
| export/static.png | `3915a732e3c947afabc70e0826a556a90f75f209cd3d72491e124ae8a9845bc0` | 120 x 120 RGBA static, loop frame 7 | `sing_loop.py` |
| export/anim/0001.png to 0012.png | per file in git | 120 x 120 RGBA loop frames | `sing_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-05.png | Style key and sheet of nine P-6 icons, quoted below, with the canonical style sheet attached. Sent 17:34:48 UTC; `c2pa.created` 2026-11-15T17:36:14Z |
| 2026-11-15 | work/, export/ | The scripts above |

Prompt of `concept/20261115-05.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. A sheet of nine reaction icons of one set, arranged in an even 3 x 3 grid with wide empty gaps of the same plain grey between them; nothing touches or crosses into another cell, every icon is whole with space around it, and there are no lines, numbers or labels between the cells. Each cell is one P-6 icon: Single expressive reaction icon: {emotion}, readable as a silhouette, one subject, no hands making any sign. Every icon is the same kind of object: a round face medallion of polished warm ivory, gently domed, set in a thin polished gold rim, seen from the front. The expression is painted on the ivory with simple bold features in deep brown lacquer: large clear eyes, expressive brows and mouth, soft rose cheeks. Accents are few and small: faceted ruby hearts, clear crystal drops, small polished gold stars. Same size, same light and same rendering in every cell. No hands or arms anywhere. The nine {emotion} values, row by row, left to right: 1, singing happily, eyes closed, a round open mouth, two small gold musical notes floating beside the medallion; 2, being touched, a trembling smile with one crystal tear, shining eyes; 3, smitten, dreamy half-closed eyes, a soft smile, one small ruby heart above the medallion; 4, silliness, the whole face painted upside down on the medallion, a happy smile; 5, an awkward grimace, all teeth showing in a wide flat line, wide eyes; 6, daydreaming, eyes looking up, a soft smile, two small gold stars drifting above the medallion; 7, excitement, wide sparkling eyes, a huge open smile, brows high; 8, laughing so hard that the face is turned sideways on the medallion, eyes shut, small crystal tears of joy; 9, gratitude, eyes closed, a warm thankful smile, the face bowed slightly forward. The attached image is our own canonical concept sheet of this set: keep exactly its medallion, its gold rim with the small ruby settings, the ivory, the lacquer features, the cheeks, the icon size, the light, the plain flat light grey background and the 3 x 3 grid with wide gaps; change only the nine expressions to the ones listed.
```

Codex was given it with `--image assets-src/cosmetic/emoji/grin/concept/20261115-01.png` (a byte copy in the agent's work folder) inside this instruction: `$imagegen <prompt> Generate exactly one image, using the attached image as the style reference. Save the final PNG in the current working directory as 20261115-05.png. Do not use any API key.`

RC-2 against this prompt and the Inputs line: it names no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical style sheet of this set. Checked by Claude Code on 2026-11-15.
