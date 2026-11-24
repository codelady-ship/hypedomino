# avatar.grand_matriarch

Static avatar of set 1 (#172), a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-14 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept generated in the project owner's ChatGPT account on 2026-11-14 by an AI agent (Claude Code) driving the owner's browser. W-3 clean-up and W-5 export by the same agent with `work/grand_matriarch_cleanup.py`. No person has done W-3
- Generator: ChatGPT in the browser, Create image tool, model `gpt-image` as named by the C2PA manifest embedded in the concept (below). Account holder: the project owner, own ChatGPT account, the account signed in to Chrome on the owner's computer, whose chat history holds the `tile.sema` conversation. Plan: Free (shown in the account menu on 2026-11-14)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-14 for the Free plan and records the date here
- Canonical concept: concept/20261114-01.png
- Inputs uploaded to the generator: none; one prompt, one image, no edits, in a new conversation (`chatgpt.com/c/6af83e9d...`, titled "Create portrait art" by ChatGPT)
- Prompt: P-3 with the style key of section 2.2 pasted verbatim, quoted below
- Concept against the prompt: Drift from the prompt: the necklace reads as gold with red stones rather than silver filigree with garnets, and the earrings carry red drops. Kept, because the drift is in jewellery colour only, not in proportions or outline (`docs/asset-workflow.md` 2.4 step 4).
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up and export are scripted image operations run by an AI agent, so RC-5 stays open; the master and the script are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; the expression is friendly; the costume cues are regional dress, not a uniform or a symbol
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `rare`, source `shop:offer.avatar_grand_matriarch`. The offer id is a placeholder until the shop record exists (V-15).

2026-11-16: owner decision — static avatars are common or rare; higher tiers are for animated avatars.

## Embedded C2PA manifest

`concept/20261114-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-14; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-14T10:58:07Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 10:57:47 UTC on 2026-11-14 (agent clock), shortly before `c2pa.created`.

## Clean-up (W-3) and export (W-5)

Run on 2026-11-14 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/grand_matriarch/work/grand_matriarch_cleanup.py grand_matriarch assets-src/cosmetic/avatar/grand_matriarch/concept/20261114-01.png assets-src/cosmetic/avatar/grand_matriarch/work assets-src/cosmetic/avatar/grand_matriarch/export
```

It printed background `[205.0, 204.0, 204.0]`, crop `(58, 0, 1115, 1057)`, canvas 512. A second run into an empty folder wrote byte-identical files.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | The flat grey background is the median of a 6 px border strip. Pixels within 30 of it (RGB distance) that connect to the top, left or right edge are background, opened by 3 px; the largest subject part is kept, its holes filled, eroded by 2 px and feathered (Gaussian sigma 1), and edge pixels are unmixed from the grey. Crop: a square of 0.86 of the shorter side, centred on the head (the top 35% of the subject), 7% headroom, clamped to the image. Canvas 512 x 512 with a circular alpha (1.5 px soft edge), because the portrait is shown in a circle (`docs/art-direction.md` 5.1); the class canvas is open (O-6) |
| 2. Paint-over, AI artefacts | No | No paint-over. Hair edges keep a thin light rim where the painted rim light and the grey background meet; an artist cleans the edge |
| 3. Moving parts on their own layers | Not needed | Static item |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen |
| 5. Silhouette and smallest-size tests | In part | The agent looked at the static as a filled silhouette and at 256, 120, 96 and 64 px on `color.background.base`, `color.surface.default` and `color.surface.raised`; face and expression read at 64 px. The smallest display size and frame band of an avatar are open (#168, O-6), so the test is not final |

`npm run asset:check -- assets-src/cosmetic/avatar/grand_matriarch`: 1 sprite, ASTC 6x6 estimate 118,336 bytes, no limit set for a static avatar (O-5).

Not done: W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `grand_matriarch: background [205.0, 204.0, 204.0], crop (12, -184, 1162, 966), canvas 512`
- Crop `(58, 0, 1115, 1057)` to `(12, -184, 1162, 966)` on the concept
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.012 before, 0.170 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/grand_matriarch/work/grand_matriarch_cleanup.py grand_matriarch assets-src/cosmetic/avatar/grand_matriarch/concept/20261114-01.png assets-src/cosmetic/avatar/grand_matriarch/work assets-src/cosmetic/avatar/grand_matriarch/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 2 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/grand_matriarch` after the re-crop: 1 sprites, ASTC 6x6 estimate 118,336 bytes, no size limit set for this type and kind (docs/asset-workflow.md O-5).

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/static.png | `24caa195d697717da938879eabaa2b9a82905519089b07dadac8e6877c5ee40d` |
| work/grand_matriarch_cleanup.py | `cda425cb6c727439dbc4471cd4ebe9411d29fbd5b10931d2c819e09a62b311ad` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261114-01.png | `ef1b572298fd9eaa0fd44543b8dea5b4df58cc78fd4efe1ccc68430095ac20d4` | 1230 x 1278 RGB, the generator's download, unchanged | ChatGPT image viewer Download button, copied byte for byte |
| work/grand_matriarch_cleanup.py | `07b450f557229a4e0e29f8e963caa0595b9d4766b456d6ea0845a97c483f5c26` | W-3 and W-5 script | Written by Claude Code, 2026-11-14; the same script for every avatar of this set |
| work/grand_matriarch_master.png | `a676e0df94535b38683eb50bf22ffda591e48584f928683ad64e226a3777397d` | 1230 x 1278 RGBA, the cut-out master | `grand_matriarch_cleanup.py` |
| export/static.png | `1b100af4e03a0b65319ae8fb9ed3a772886e96392fbb25cf74e2da9cdf2f9c31` | 512 x 512 RGBA static, circular alpha | `grand_matriarch_cleanup.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-14 | concept/20261114-01.png | P-3 with the style key, the only prompt of the conversation, quoted below. Sent 10:57:47 UTC by Claude Code; `c2pa.created` 2026-11-14T10:58:07Z |
| 2026-11-14 | work/grand_matriarch_master.png, export/static.png | `grand_matriarch_cleanup.py` on the concept |
| 2026-11-16 | export/static.png, work/grand_matriarch_cleanup.py | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(58, 0, 1115, 1057)` to `(12, -184, 1162, 966)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261114-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her seventies with light skin, silver hair in a low bun held by a small gold comb, warm laugh lines, a deep emerald velvet kaftan with dense gold embroidery and a silver filigree necklace set with garnets, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

RC-2 against this prompt and the Inputs line: it names no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-14.
