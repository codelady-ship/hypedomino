# emoji.cool

Static emoji of set 1 (#175), one cell of a concept sheet of nine emoji made with template P-6 of `docs/asset-workflow.md` section 2. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept sheet generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up and W-5 export by the same agent with the scripts in `work/`. No person has done W-3
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifest in the PNG names ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-03.png, row 3, column 3 of its 3 x 3 sheet. The sheet is shared: every emoji cut from it holds a byte-identical copy under the same name (sha256 below). The sheet was made from the canonical style sheet of the set, `assets-src/cosmetic/emoji/grin/concept/20261115-01.png` (sha256 `d5411efee91748223ffa258facec43952dc69135b3499ddcef5f4ba42b50a410`)
- Inputs uploaded to the generator: `assets-src/cosmetic/emoji/grin/concept/20261115-01.png`, our own canonical style sheet of this set, attached with `codex exec --image` as the style reference. Nothing else was uploaded
- Prompt: the style key of section 2.2 pasted verbatim, then a sheet instruction in which each of the nine cells is one P-6 icon, with `{emotion}` = "cool confidence, dark lacquer sunglasses with thin gold frames, a relaxed smile" for this cell; quoted below. Deviation from section 2.3, decided for #175 to save generations: P-6 asks for a single icon per image, the sheet asks for nine in one image, each cell being one P-6 icon and one item
- Concept against the prompt: the cell matches its `{emotion}`. Not asked for but present in every cell of the set: ruby settings at the top and bottom of the rim, small gold knobs at the sides and a crimson inner band in the rim
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): no person is depicted; the face is a stylised expression painted on an ivory medallion
- Human work: none. Clean-up and export are scripted image operations run by an AI agent, so RC-5 stays open; the master and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no hands or gestures; no glow or drop shadow painted in; the expression is friendly and not taunting
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `rare`, group `greet`, source `shop:offer.emoji_cool`. The offer id is a placeholder until the shop record exists (V-15). The repository names no emoji group yet besides `basic` in the example manifests, so set 1 proposes the groups joy, cheer, love, surprise, think, sorry, greet (`--group` of `npm run asset:add`). Static emoji are proposed at `common` and `rare`, animated ones at `rare` and above.

## Embedded C2PA manifest

`concept/20261115-03.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-15T17:32:26Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 17:31:03 |
| Codex session | `01a1e043-389d-7511-ad4f-9b84d78ff88d` |

Codex saved the image under `~/.codex/generated_images/<session>/` and copied it into the agent's work folder; the two files are byte-identical (checked with `cmp`), and `concept/` holds a byte copy.

The nine cells of `concept/20261115-03.png`, row by row:

- row 1, column 1: `emoji.monocle`
- row 1, column 2: rejected (a bright idea): the lightbulb is painted lit with a soft halo, a baked glow (docs/art-direction.md section 4)
- row 1, column 3: rejected (concentration): the drawn-together brows read as anger, a taunt risk
- row 2, column 1: `emoji.sad`
- row 2, column 2: `emoji.oops`
- row 2, column 3: `emoji.apology`
- row 3, column 1: `emoji.pleading`
- row 3, column 2: `emoji.relieved`
- row 3, column 3: `emoji.cool`

No whole sheet was rejected: six sheets were generated for the set, and the rejected cells above were left out.

## Clean-up (W-3)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root (also run for every emoji by `npm run asset:emoji`):

```bash
python -I assets-src/cosmetic/emoji/cool/work/cool_cleanup.py assets-src/cosmetic/emoji/cool
```

It printed `cool_cleanup: sheet 20261115-03.png cell (2, 2), background [209.0, 208.0, 209.0], pockets 0 px, medallion width 355 px at (1039.0, 1045.5), scale 1.0366, master 480, static written`.

The script is the clean-up of the set 1 avatars (`avatar.jade_tassel`) adapted to a sheet: the flat grey is the median of a 6 px border strip; pixels within 30 of it that connect to any edge are background, opened by 2 px; enclosed grey pockets are background only when flat (spread under 6) so crystal stays; the subject parts whose centre lies in this item's cell are kept; the alpha is eroded by 1 px, feathered (sigma 1) and keyed along a 5 px band against the colour of the nearby subject, and edge pixels are unmixed from the grey. The medallion width is measured on the lower half of the largest part, so a hat, stars or hearts above it do not change it, and the item is scaled so the medallion is 368 px wide on a 480 x 480 master (92 px at 120), centred, and moved or shrunk only as far as needed to keep everything 8 px inside the canvas, so every emoji of the set shows the medallion at the same size.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas set by D-2 | As above; `work/cool_master.png` is 480 x 480; the static is it box-downsampled by 4 at 120 x 120 (D-2 of the Epic 15 plan; the class canvas O-6 is open) |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Not needed | Static item |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen |
| 5. Silhouette and smallest-size tests | In part | The agent looked at the static at 120 px and at 2x on a dark backdrop; the round medallion and the expression read. The smallest display size of an emoji is open (#168, O-6), so the test is not final |

## Export (W-5)

`export/static.png` is written by the clean-up script: 120 x 120 RGBA PNG, 8 bit, straight alpha. `npm run asset:check`: 1 sprite, ASTC 6x6 estimate 6,400 bytes, under the 100,000 bytes of #175.

Not done: W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-03.png | `24aa208a1b82d6d974e407fce41e87cf89e5b26e1ba139bde2a347a30fa98674` | 1254 x 1254 RGB, the concept sheet, unchanged | Codex CLI `$imagegen`, session `01a1e043-389d-7511-ad4f-9b84d78ff88d` |
| work/cool_cleanup.py | `4a850c8e5782f25ac33c0760b90193064314a438c50178dddb946cac3d3a7358` | W-3 script and W-5 export | Written by Claude Code, 2026-11-15; the same script for every emoji of the set with the item's sheet and cell |
| work/cool_master.png | `b4f1626c43b419b533f505aca95619fbba7c97cefeacba0ec3fd79688c300fc4` | 480 x 480 RGBA, the cut-out master on the 4x canvas | `cool_cleanup.py` |
| export/static.png | `afbce46f42bc8054f32f9e862f57e0082bcadc8e72733ee7396c4f10dc2c3ed5` | 120 x 120 RGBA static | `cool_cleanup.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-03.png | Style key and sheet of nine P-6 icons, quoted below, with the canonical style sheet attached. Sent 17:31:03 UTC; `c2pa.created` 2026-11-15T17:32:26Z |
| 2026-11-15 | work/, export/ | The scripts above |

Prompt of `concept/20261115-03.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. A sheet of nine reaction icons of one set, arranged in an even 3 x 3 grid with wide empty gaps of the same plain grey between them; nothing touches or crosses into another cell, every icon is whole with space around it, and there are no lines, numbers or labels between the cells. Each cell is one P-6 icon: Single expressive reaction icon: {emotion}, readable as a silhouette, one subject, no hands making any sign. Every icon is the same kind of object: a round face medallion of polished warm ivory, gently domed, set in a thin polished gold rim, seen from the front. The expression is painted on the ivory with simple bold features in deep brown lacquer: large clear eyes, expressive brows and mouth, soft rose cheeks. Accents are few and small: faceted ruby hearts, clear crystal drops, small polished gold stars. Same size, same light and same rendering in every cell. No hands or arms anywhere. The nine {emotion} values, row by row, left to right: 1, curiosity, inspecting with a round gold monocle over one eye, the other brow raised, a small smile; 2, a bright idea, a delighted open smile, eyes looking up at a small brass and crystal lightbulb floating just above the medallion; 3, concentration, narrowed determined eyes, a firm small mouth, brows drawn together; 4, quiet sadness, a downturned mouth, brows slanted up, glistening eyes; 5, an embarrassed oops, an awkward toothy grin, raised brows, one crystal drop of sweat on the brow; 6, apology, eyes closed, brows raised in the middle, a small wavy frown; 7, pleading, very large shining eyes, a small hopeful frown; 8, relief, eyes closed, a gentle smile, one crystal drop of sweat sliding from the brow; 9, cool confidence, dark lacquer sunglasses with thin gold frames, a relaxed smile. The attached image is our own canonical concept sheet of this set: keep exactly its medallion, its gold rim with the small ruby settings, the ivory, the lacquer features, the cheeks, the icon size, the light, the plain flat light grey background and the 3 x 3 grid with wide gaps; change only the nine expressions to the ones listed.
```

Codex was given it with `--image assets-src/cosmetic/emoji/grin/concept/20261115-01.png` (a byte copy in the agent's work folder) inside this instruction: `$imagegen <prompt> Generate exactly one image, using the attached image as the style reference. Save the final PNG in the current working directory as 20261115-03.png. Do not use any API key.`

RC-2 against this prompt and the Inputs line: it names no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical style sheet of this set. Checked by Claude Code on 2026-11-15.
