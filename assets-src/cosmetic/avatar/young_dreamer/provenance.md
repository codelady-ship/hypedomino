# avatar.young_dreamer

Static avatar of set 1 (#172), a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-14 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept generated in the project owner's second ChatGPT account on 2026-11-14 by an AI agent (Claude Code) driving the owner's browser. W-3 clean-up and W-5 export by the same agent with `work/young_dreamer_cleanup.py`. No person has done W-3
- Generator: ChatGPT in the browser, Create image tool, model `gpt-image` as named by the C2PA manifest embedded in the concept (below). Account holder: the project owner, second ChatGPT account (the first one reached its image limit after `avatar.grand_matriarch`), signed in to the second browser on the owner's computer. Plan: Free (shown under the account name in the ChatGPT sidebar on 2026-11-14)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-14 for the Free plan and records the date here
- Canonical concept: concept/20261114-01.png
- Inputs uploaded to the generator: none; one prompt, one image, no edits, in a new conversation (`chatgpt.com/c/6af85160...`, titled "Create portrait art" by ChatGPT)
- Prompt: P-3 with the style key of section 2.2 pasted verbatim, quoted below
- Concept against the prompt: Matches the prompt. Not asked for but present: gold floral embroidery covering the waistcoat, a standing collar with gold trim, and wide white sleeves.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up and export are scripted image operations run by an AI agent, so RC-5 stays open; the master and the script are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; the expression is friendly; the costume cues are regional dress, not a uniform or a symbol
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `rare`, source `shop:offer.avatar_young_dreamer`. The offer id is a placeholder until the shop record exists (V-15).

## Embedded C2PA manifest

`concept/20261114-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-14; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-14T12:18:11Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 12:18:02 UTC on 2026-11-14 (agent clock), shortly before `c2pa.created`.

## Clean-up (W-3) and export (W-5)

Run on 2026-11-14 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/young_dreamer/work/young_dreamer_cleanup.py young_dreamer assets-src/cosmetic/avatar/young_dreamer/concept/20261114-01.png assets-src/cosmetic/avatar/young_dreamer/work assets-src/cosmetic/avatar/young_dreamer/export
```

It printed background `[202.0, 201.0, 201.0]`, pockets 1132 px, crop `(127, 0, 1191, 1064)`, canvas 512. A second run into an empty folder wrote byte-identical files.

The script is the set 1 script of `avatar.silk_weaver` with one added step: grey pockets enclosed by curls or loose strands (within 10 of the background colour, at least 20 px, at most 80 px inside the subject's outline) are background too. Without it, grey between curls stayed opaque, because the first script fills every hole of the subject.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | The flat grey background is the median of a 6 px border strip. Pixels within 30 of it (RGB distance) that connect to the top, left or right edge are background, opened by 3 px; enclosed pockets as above are background too; the largest subject part is kept, its other holes filled, eroded by 2 px and feathered (Gaussian sigma 1), and edge pixels are unmixed from the grey. Crop: a square of 0.86 of the shorter side, centred on the head (the top 35% of the subject), 7% headroom, clamped to the image. Canvas 512 x 512 with a circular alpha (1.5 px soft edge), because the portrait is shown in a circle (`docs/art-direction.md` 5.1); the class canvas is open (O-6) |
| 2. Paint-over, AI artefacts | No | No paint-over. Hair edges keep a thin light rim where the painted rim light and the grey background meet, and fine loose strands keep some grey; an artist cleans the edge |
| 3. Moving parts on their own layers | Not needed | Static item |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen |
| 5. Silhouette and smallest-size tests | In part | The agent looked at the static on a dark backdrop and at 64 px; face and expression read. The smallest display size and frame band of an avatar are open (#168, O-6), so the test is not final |

`npm run asset:check -- assets-src/cosmetic/avatar/young_dreamer`: 1 sprite, ASTC 6x6 estimate 118,336 bytes, no limit set for a static avatar (O-5).

Not done: W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `young_dreamer: background [202.0, 201.0, 201.0], pockets 1132 px, crop (81, -180, 1238, 977), canvas 512`
- Crop `(127, 0, 1191, 1064)` to `(81, -180, 1238, 977)` on the concept
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.016 before, 0.170 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/young_dreamer/work/young_dreamer_cleanup.py young_dreamer assets-src/cosmetic/avatar/young_dreamer/concept/20261114-01.png assets-src/cosmetic/avatar/young_dreamer/work assets-src/cosmetic/avatar/young_dreamer/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 2 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/young_dreamer` after the re-crop: 1 sprites, ASTC 6x6 estimate 118,336 bytes, no size limit set for this type and kind (docs/asset-workflow.md O-5).

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/static.png | `8122cd29d36a53958ef0aaa67e388ea742e8c244d7cbf456841fcfd108dfc35c` |
| work/young_dreamer_cleanup.py | `242a131f47bcbee479780b345db58ff23706fd0567bf547fd4df51ca0c99f999` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261114-01.png | `0b9f050a347a623a12f579d0aa0568579795264c042075fe0c92edf5be415360` | 1271 x 1238 RGB, the generator's download, unchanged | ChatGPT image viewer Download button, copied byte for byte |
| work/young_dreamer_cleanup.py | `e46ea3dfeef9cc0b5a8dba2dba88ea1da85cfa3513a81ea7f9258d66b1412283` | W-3 and W-5 script | Written by Claude Code, 2026-11-14; the same script for every avatar of this PR |
| work/young_dreamer_master.png | `e8cdf79b59722a18733a08e29dfe1665e1e84ea1a83fd5bf4ee740697979da77` | 1271 x 1238 RGBA, the cut-out master | `young_dreamer_cleanup.py` |
| export/static.png | `0abb253fd9d044c76dc82190cba45dbf16bbbb48a92d89cfc0233cecc0f2b625` | 512 x 512 RGBA static, circular alpha | `young_dreamer_cleanup.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-14 | concept/20261114-01.png | P-3 with the style key, the only prompt of the conversation, quoted below. Sent 12:18:02 UTC by Claude Code; `c2pa.created` 2026-11-14T12:18:11Z |
| 2026-11-14 | work/young_dreamer_master.png, export/static.png | `young_dreamer_cleanup.py` on the concept |
| 2026-11-16 | export/static.png, work/young_dreamer_cleanup.py | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(127, 0, 1191, 1064)` to `(81, -180, 1238, 977)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261114-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a boy of about ten with medium brown skin, curly black hair and a wide smile, a bright blue embroidered waistcoat with brass buttons over a white linen shirt, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

RC-2 against this prompt and the Inputs line: it names no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-14.
