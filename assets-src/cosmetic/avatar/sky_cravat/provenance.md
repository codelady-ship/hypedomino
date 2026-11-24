# avatar.sky_cravat

Static avatar of set 1 (#172), batch B, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept generated on 2026-11-15 by an AI agent (Claude Code) running Codex CLI on the project owner's computer. W-3 clean-up and W-5 export by the same agent with `work/sky_cravat_cleanup.py`. No person has done W-3
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model `gpt-image-2` per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifest in the PNG names `ChatGPT` / `gpt-image` (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: none; one prompt, one image, no edits, in a new Codex session (`01a1e005-b9a3-7e70-87c3-7de207809203`).
- Prompt: P-3 with the style key of section 2.2 pasted verbatim, quoted below
- Concept against the prompt: Matches the prompt; the silver pin is a straight bar across the cravat. Not asked for but present: gold scrollwork on the coat lapels, an ivory brocade waistcoat and a sapphire-coloured button.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up and export are scripted image operations run by an AI agent, so RC-5 stays open; the master and the script are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; the expression is friendly; the costume cues are salon dress, not a uniform or a symbol. For the reviewer: the pale sky blue of the cravat sits near the hue of `color.role.teamA`, but it is a small, pale accent on a silver-grey coat and does not dominate the item
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `rare`, source `shop:offer.avatar_sky_cravat`. The offer id is a placeholder until the shop record exists (V-15).

2026-11-16: owner decision — static avatars are common or rare; higher tiers are for animated avatars.

## Embedded C2PA manifest

`concept/20261115-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-15T16:24:32Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 16:23:53 UTC on 2026-11-15 (agent clock), shortly before `c2pa.created`. Codex saved the image under `~/.codex/generated_images/01a1e005-b9a3-7e70-87c3-7de207809203/` and copied it into `concept/`; the two files are byte-identical (checked with `cmp`).

## Clean-up (W-3) and export (W-5)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/sky_cravat/work/sky_cravat_cleanup.py sky_cravat assets-src/cosmetic/avatar/sky_cravat/concept/20261115-01.png assets-src/cosmetic/avatar/sky_cravat/work assets-src/cosmetic/avatar/sky_cravat/export
```

It printed background `[187.0, 187.0, 187.0]`, pockets 1127 px, crop `(124, 0, 1155, 1031)`, canvas 512. Two more runs into two empty folders wrote files byte-identical to each other and to the committed ones.

The script is `amethyst_courtier_cleanup.py` of batch A. That script is set 1 `court_poet_cleanup.py` plus one step: grey background that the 3 px opening cuts off from the outside counts as a pocket, so it is not filled back as subject.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | The flat grey background is the median of a 6 px border strip. Pixels within 30 of it (RGB distance) that connect to the top, left or right edge are background, opened by 3 px; enclosed pockets (within 10 of the background colour, at least 20 px, at most 80 px inside the subject's outline) and background cut off by the opening are background too; the largest subject part is kept, its other holes filled, eroded by 2 px and feathered (Gaussian sigma 1), and edge pixels are unmixed from the grey. Crop: a square of 0.86 of the shorter side, centred on the head (the top 35% of the subject), 7% headroom, clamped to the image. Canvas 512 x 512 with a circular alpha (1.5 px soft edge), because the portrait is shown in a circle (`docs/art-direction.md` 5.1); the class canvas is open (O-6) |
| 2. Paint-over, AI artefacts | No | No paint-over. The white hair at the sides keeps a light rim and some grey in fine wisps; an artist cleans the edge |
| 3. Moving parts on their own layers | Not needed | Static item |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen |
| 5. Silhouette and smallest-size tests | In part | The agent looked at the static on a dark backdrop and at 256 px and 64 px; face and expression read. The smallest display size and frame band of an avatar are open (#168, O-6), so the test is not final |

`npm run asset:check -- assets-src/cosmetic/avatar/sky_cravat`: 1 sprite, ASTC 6x6 estimate 118,336 bytes, no limit set for a static avatar (O-5).

Not done: W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `sky_cravat: background [187.0, 187.0, 187.0], pockets 1127 px, crop (79, -174, 1200, 947), canvas 512`
- Crop `(124, 0, 1155, 1031)` to `(79, -174, 1200, 947)` on the concept
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.016 before, 0.170 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/sky_cravat/work/sky_cravat_cleanup.py sky_cravat assets-src/cosmetic/avatar/sky_cravat/concept/20261115-01.png assets-src/cosmetic/avatar/sky_cravat/work assets-src/cosmetic/avatar/sky_cravat/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 2 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/sky_cravat` after the re-crop: 1 sprites, ASTC 6x6 estimate 118,336 bytes, no size limit set for this type and kind (docs/asset-workflow.md O-5).

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/static.png | `22fb557240946359eb9e0c8745ea7f7ab58716b3a310997f5282d5b3f665161a` |
| work/sky_cravat_cleanup.py | `2203d04a41dd2a85e740e8c685588424e4a4f9a50ad8c48b7f8a7c447371edc0` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `2a56083945d0eaf6b89990ded55e2467a9621d37c90d6b6d193d4ae9e480408f` | 1312 x 1199 RGB, the generator's output, unchanged | Codex CLI `$imagegen`, copied byte for byte by Codex into `concept/` |
| work/sky_cravat_cleanup.py | `3492a2b5d061b1c2729a06914305614c4a252a7c2579696bd081c98668dc9e18` | W-3 and W-5 script | Written by Claude Code, 2026-11-15, from `amethyst_courtier_cleanup.py` of batch A |
| work/sky_cravat_master.png | `e819c23de4272af5bfd6ccb2394ce18828496aa29c1940ff86666a8c501780cc` | 1312 x 1199 RGBA, the cut-out master | `sky_cravat_cleanup.py` |
| export/static.png | `8cbb87340817477d58f8c2c7a949604f1c0f0cf8f8868241ab8dc4e70eab52bb` | 512 x 512 RGBA static, circular alpha | `sky_cravat_cleanup.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted below. Sent 16:23:53 UTC by Claude Code, Codex session `01a1e005-b9a3-7e70-87c3-7de207809203`; `c2pa.created` 2026-11-15T16:24:32Z |
| 2026-11-15 | work/sky_cravat_master.png, export/static.png | `sky_cravat_cleanup.py` on the concept |
| 2026-11-16 | export/static.png, work/sky_cravat_cleanup.py | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(124, 0, 1155, 1031)` to `(79, -174, 1200, 947)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261115-01.png`, verbatim (the Codex call wraps it in `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`):

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his seventies with ruddy light skin, a bald head with white hair at the sides, bushy white eyebrows, a short white beard, a silver-grey wool frock coat and a pale sky blue silk cravat with a small silver pin, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-15.
