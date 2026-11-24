# emoji.dizzy

Animated emoji of set 1 (#175), one cell of a concept sheet of nine emoji made with template P-6 of `docs/asset-workflow.md` section 2, animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept sheet generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifest in the PNG names ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-02.png, row 3, column 3 of its 3 x 3 sheet. The sheet is shared: every emoji cut from it holds a byte-identical copy under the same name (sha256 below). The sheet was made from the canonical style sheet of the set, `assets-src/cosmetic/emoji/grin/concept/20261115-01.png` (sha256 `d5411efee91748223ffa258facec43952dc69135b3499ddcef5f4ba42b50a410`)
- Inputs uploaded to the generator: `assets-src/cosmetic/emoji/grin/concept/20261115-01.png`, our own canonical style sheet of this set, attached with `codex exec --image` as the style reference. Nothing else was uploaded
- Prompt: the style key of section 2.2 pasted verbatim, then a sheet instruction in which each of the nine cells is one P-6 icon, with `{emotion}` = "dizziness, eyes drawn as two gold spirals, a wobbly smile, small gold stars circling above" for this cell; quoted below. Deviation from section 2.3, decided for #175 to save generations: P-6 asks for a single icon per image, the sheet asks for nine in one image, each cell being one P-6 icon and one item
- Concept against the prompt: the cell matches its `{emotion}`. Not asked for but present in every cell of the set: ruby settings at the top and bottom of the rim, small gold knobs at the sides and a crimson inner band in the rim
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): no person is depicted; the face is a stylised expression painted on an ivory medallion
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no hands or gestures; no glow or drop shadow painted in; the expression is friendly and not taunting
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, group `surprise`, source `shop:offer.emoji_dizzy`. The offer id is a placeholder until the shop record exists (V-15). The repository names no emoji group yet besides `basic` in the example manifests, so set 1 proposes the groups joy, cheer, love, surprise, think, sorry, greet (`--group` of `npm run asset:add`). Static emoji are proposed at `common` and `rare`, animated ones at `rare` and above.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 2 animated details (spiral_left, spiral_right), so `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifest

`concept/20261115-02.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-15T17:30:17Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 17:28:53 |
| Codex session | `01a1e041-3e2e-7841-9a29-77c79634b176` |

Codex saved the image under `~/.codex/generated_images/<session>/` and copied it into the agent's work folder; the two files are byte-identical (checked with `cmp`), and `concept/` holds a byte copy.

The nine cells of `concept/20261115-02.png`, row by row:

- row 1, column 1: `emoji.beam`
- row 1, column 2: `emoji.giggle`
- row 1, column 3: `emoji.hooray`
- row 2, column 1: `emoji.party`
- row 2, column 2: rejected (pride): the face reads as smug, a taunt risk (docs/art-direction.md section 8, friendly and all-ages)
- row 2, column 3: rejected (blowing a kiss): the puckered lips are drawn as a shape that reads as the numeral 3 (no numbers in art, docs/art-direction.md section 8)
- row 3, column 1: `emoji.blush`
- row 3, column 2: `emoji.shock`
- row 3, column 3: `emoji.dizzy`

No whole sheet was rejected: six sheets were generated for the set, and the rejected cells above were left out.

## Clean-up (W-3)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root (also run for every emoji by `npm run asset:emoji`):

```bash
python -I assets-src/cosmetic/emoji/dizzy/work/dizzy_cleanup.py assets-src/cosmetic/emoji/dizzy
```

It printed `dizzy_cleanup: sheet 20261115-02.png cell (2, 2), background [207.0, 207.0, 207.0], pockets 0 px, medallion width 357 px at (1040.0, 1045.5), scale 1.0308, master 480, static from the loop`.

The script is the clean-up of the set 1 avatars (`avatar.jade_tassel`) adapted to a sheet: the flat grey is the median of a 6 px border strip; pixels within 30 of it that connect to any edge are background, opened by 2 px; enclosed grey pockets are background only when flat (spread under 6) so crystal stays; the subject parts whose centre lies in this item's cell are kept; the alpha is eroded by 1 px, feathered (sigma 1) and keyed along a 5 px band against the colour of the nearby subject, and edge pixels are unmixed from the grey. The medallion width is measured on the lower half of the largest part, so a hat, stars or hearts above it do not change it, and the item is scaled so the medallion is 368 px wide on a 480 x 480 master (92 px at 120), centred, and moved or shrunk only as far as needed to keep everything 8 px inside the canvas, so every emoji of the set shows the medallion at the same size.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas set by D-2 | As above; `work/dizzy_master.png` is 480 x 480; the loop renders every frame and the static from it at 120 x 120 (D-2 of the Epic 15 plan; the class canvas O-6 is open) |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Yes | Written by the loop script into `work/` (files below) |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen |
| 5. Silhouette and smallest-size tests | In part | The agent looked at the static at 120 px and at 2x on a dark backdrop; the round medallion and the expression read. The smallest display size of an emoji is open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-15 by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/emoji/dizzy/work/dizzy_loop.py assets-src/cosmetic/emoji/dizzy
```

It printed `dizzy_loop: parts 2 (spiral_left, spiral_right), 12 frames at 120x120, static frame 7`. It writes the layers below into `work/`, `export/anim/0001.png` to `0012.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. Every frame is rendered at 480 x 480 in premultiplied alpha and box-downsampled by 4.

| Choice | Value | Source |
|---|---|---|
| Canvas | 120 x 120 | D-2 of the Epic 15 plan |
| Frames | 12 | D-2 |
| Frame rate | 12 fps, a loop of 1 s | D-2. The rate is not stored with the loop (O-5, O-6) |
| Motion | The gold spiral eyes turn once per loop in opposite directions; the medallion wobbles by up to 8 degrees | Affine moves of the whole medallion about its bottom centre and of the parts about their own centre (BODY and PARTS in the script); px at 120 |
| Parts | `spiral_left` (spin), `spiral_right` (spin) | Each part is the master through a feathered mask: its colour (red, crystal, gold or dark) inside an ellipse set in the script. The place a moving part leaves in the base layer is filled from its surroundings (normalised Gaussian fill) for spiral_left, spiral_right. |
| Seamless loop | Mean channel change from frame 12 to frame 1 is 24.51 of 255, against 18.60 on average and 24.58 at most between the other neighbouring frames | Every motion is periodic in the 12 frames |
| Static | Frame 7 | The rest frame: no lift, tilt or squash, every part at its painted place and size. `export/static.png` is byte-identical to `export/anim/0007.png` |
| Size | 13 sprites, 83,200 bytes | `npm run asset:check`, ASTC 6x6 estimate (13 x 6,400), under the 100,000 bytes of #175 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-02.png | `0fbefbef3fba69a825148ac314c28a743719a018fd5a6d2e7641f5c9837d1566` | 1254 x 1254 RGB, the concept sheet, unchanged | Codex CLI `$imagegen`, session `01a1e041-3e2e-7841-9a29-77c79634b176` |
| work/dizzy_base.png | `bd12cbaa076fe5b3293f8f34c46e4433cc970b4b009fb84732223b97c85e87c4` | The master with the moving parts taken out (filled) or kept | `dizzy_loop.py` |
| work/dizzy_cleanup.py | `35ddf04fbeca1e58c1d0b43a6f6b17bd72f3ee20ccb8ccf09993c1cf85ad5b54` | W-3 script | Written by Claude Code, 2026-11-15; the same script for every emoji of the set with the item's sheet and cell |
| work/dizzy_loop.py | `2041186c49f8ac2778ea3b061512ad0222f69d702bdd3f0c63b6fbc73b328568` | W-4 and W-5 script | Written by Claude Code, 2026-11-15; the same script for every animated emoji of the set with the item's motion and parts |
| work/dizzy_master.png | `f03db3d7b55f459c270d8932479d6e2f46449cadd7c746deb2b3fab1f40ce616` | 480 x 480 RGBA, the cut-out master on the 4x canvas | `dizzy_cleanup.py` |
| work/dizzy_spiral_left.png | `337bb8046c674f1d789912918e1487da03085c4106a4ed622876d7ed55cae783` | Moving part `spiral_left`: the master through its feathered mask | `dizzy_loop.py` |
| work/dizzy_spiral_right.png | `8d8232878f56d0ef7bf01118f63b14fc3055e2fe59a096eda0cb88f2086d553b` | Moving part `spiral_right`: the master through its feathered mask | `dizzy_loop.py` |
| export/static.png | `8b7643101c7a84a328251de57110b7deaecca61d9b40558a51f951957db3d5ce` | 120 x 120 RGBA static, loop frame 7 | `dizzy_loop.py` |
| export/anim/0001.png to 0012.png | per file in git | 120 x 120 RGBA loop frames | `dizzy_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-02.png | Style key and sheet of nine P-6 icons, quoted below, with the canonical style sheet attached. Sent 17:28:53 UTC; `c2pa.created` 2026-11-15T17:30:17Z |
| 2026-11-15 | work/, export/ | The scripts above |

Prompt of `concept/20261115-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. A sheet of nine reaction icons of one set, arranged in an even 3 x 3 grid with wide empty gaps of the same plain grey between them; nothing touches or crosses into another cell, every icon is whole with space around it, and there are no lines, numbers or labels between the cells. Each cell is one P-6 icon: Single expressive reaction icon: {emotion}, readable as a silhouette, one subject, no hands making any sign. Every icon is the same kind of object: a round face medallion of polished warm ivory, gently domed, set in a thin polished gold rim, seen from the front. The expression is painted on the ivory with simple bold features in deep brown lacquer: large clear eyes, expressive brows and mouth, soft rose cheeks. Accents are few and small: faceted ruby hearts, clear crystal drops, small polished gold stars. Same size, same light and same rendering in every cell. No hands or arms anywhere. The nine {emotion} values, row by row, left to right: 1, contentment, a beaming closed-mouth smile, eyes closed in happy arcs, deep rose cheeks; 2, a giggle, eyes closed, cheeks puffed, a small open smile; 3, jubilation, mouth wide open in a cheer, brows high, three small gold stars beside the medallion; 4, celebration, a small striped ivory and ruby silk party cone hat on top of the medallion and a few small gold leaf flakes around it; 5, pride, chin raised, eyes closed, a satisfied smile; 6, blowing a kiss, puckered lips, one eye closed, one small ruby heart floating away from the lips; 7, shyness, eyes looking down, a small smile, deep rose blush across both cheeks; 8, shock, very wide eyes with small pupils, mouth open in a tall oval, brows high; 9, dizziness, eyes drawn as two gold spirals, a wobbly smile, small gold stars circling above. The attached image is our own canonical concept sheet of this set: keep exactly its medallion, its gold rim with the small ruby settings, the ivory, the lacquer features, the cheeks, the icon size, the light, the plain flat light grey background and the 3 x 3 grid with wide gaps; change only the nine expressions to the ones listed.
```

Codex was given it with `--image assets-src/cosmetic/emoji/grin/concept/20261115-01.png` (a byte copy in the agent's work folder) inside this instruction: `$imagegen <prompt> Generate exactly one image, using the attached image as the style reference. Save the final PNG in the current working directory as 20261115-02.png. Do not use any API key.`

RC-2 against this prompt and the Inputs line: it names no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical style sheet of this set. Checked by Claude Code on 2026-11-15.
