# frame.brass_gears

Animated frame of #173 (batch A of the 8 animated frames), an ornamental ring made with template P-4 of `docs/asset-workflow.md` section 2 and animated offline: parts of the cleaned master and drawn ambient detail moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifest in the PNG names ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: `concept/20261115-01.png`, our own canonical concept, attached to one edit (`codex exec -i`) that removed every gear to give the plate under the moving parts (`concept/20261115-02.png`). No other input
- Prompt: P-4 with the style key of section 2.2 pasted verbatim; `{theme in plain words}` = "a ring of warm brass clockwork with several separate round toothed gears of different sizes sitting on the outer edge of the ring, the largest gear at the top, small rubies set in the ring", `{colour words}` = "warm brass, copper and dark jade enamel"; quoted below
- Concept against the prompt: Matches the prompt and adds more: a brass ring with a dark jade enamel band and gold scrollwork, seven separate toothed gears on its outer edge (a large five-spoke gear at the top, copper-faced gears with round holes on the upper and lower sides, two small eight-spoke gears), rubies at the sides and bottom in crystal-petal mounts, and crimson drapery on the lower band that the prompt did not ask for. The small eight-spoke gears have teeth; whether a spoked wheel reads as a symbol is left to the cultural review. The rendering is flatter than the other items of this batch.
- Similarity check: not done (RC-3 open, O-7)
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the masters, layers and scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; points for the cultural review are named under Concept against the prompt
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `legendary`, source `shop:offer.frame_brass_gears`. The offer id is a placeholder until the shop record exists (V-15). Animated frames are proposed at `epic` and above, as the batch leans to the higher tiers of `docs/rarity.md`.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 2 animated details (turning gear rims, ruby glints), so `legendary`. Any tier split stated above is superseded.

## Embedded C2PA manifest

`concept/20261115-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-15T15:44:41Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 15:43:54 UTC on 2026-11-15 (agent clock), Codex session `01a1dfe1-1e8d-7060-a36c-16b55fca1645`. Codex saved its image from `~/.codex/generated_images/01a1dfe1-1e8d-7060-a36c-16b55fca1645/` into `concept/`; the two files have the same sha256. No generation for this item was rejected: this image and the edit below are the only ones generated for it.

`concept/20261115-02.png` (the edit) carries its own manifest, read the same way: `c2pa.created` 2026-11-15T15:49:31Z, software agent `ChatGPT` version `gpt-image`, `trainedAlgorithmicMedia`, claim generator `OpenAI Media Service API`, signer OpenAI OpCo, LLC; it names no ingredient. Sent at 15:48:31 UTC, Codex session `01a1dfe5-5968-7850-9808-e6a04d0eb6a0`.

## Clean-up (W-3)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/frame/brass_gears/work/brass_gears_cleanup.py brass_gears assets-src/cosmetic/frame/brass_gears/concept/20261115-01.png assets-src/cosmetic/frame/brass_gears/work
python -I assets-src/cosmetic/frame/brass_gears/work/brass_gears_cleanup.py brass_gears_plate assets-src/cosmetic/frame/brass_gears/concept/20261115-02.png assets-src/cosmetic/frame/brass_gears/work
python -I assets-src/cosmetic/frame/brass_gears/work/brass_gears_parts.py assets-src/cosmetic/frame/brass_gears/work
```

The clean-up printed background `[199.0, 198.0, 198.0]`, box `(-7, -11, 1260, 1255)`, opening radius 0.619 of the half box. A second run of every script into an empty folder wrote byte-identical files.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | The set 1 frame clean-up (`frame/ember_crown/work/ember_crown_cleanup.py`) with two changes: the script writes only the master (the static and the loop come from the loop script), and background pockets inside the ornament are found by flatness, so light metal close to the grey is kept. A pocket is a region within 14 of the grey whose local spread (standard deviation of luminance over 5 x 5 px) has a median under 1.2, or pixels within 6 of the grey with a spread under 1.2, in parts of at least 20 px; it grows over neighbouring pixels within 40 of the grey with a saturation under 12 and a spread under 4. Everything else is as in set 1: border median colour, edge-connected background within 30, the open centre, specks under 200 px dropped, 1 px erosion, feather sigma 0.8, fringe unmixed. The loop script places the master on a square centred on its bounding box with 2% margin on each side, as set 1 did, so the opening keeps the share printed above |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Yes | `work/brass_gears_parts.py` splits the master into `work/brass_gears_teeth.png` (the tooth rims) and `work/brass_gears_under.png` (the master with the tooth rims replaced by the cleaned edit `work/brass_gears_plate_master.png`, so the band shows between the turning teeth). A tooth pixel lies between the root and tip radius of a gear (centres and radii measured on the master and listed in `GEARS`), differs from the plate by more than 0.12, is gold (red over green by 0.04) and opaque; the mask is opened by 1 px, grown by 2 px and feathered (Gaussian sigma 0.8) |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen in the concept, the master or the frames; the ambient detail is drawn without glow. The static shows the metal and gems at rest with no aura, sheen or ambient detail (`docs/art-direction.md` section 7) |
| 5. Silhouette and smallest-size tests | In part | The agent drew the static and loop frame 5 around `avatar.court_poet` set inside the opening at 256, 128 and 64 px on `color.background.base`. No avatar size is set in `docs/ui-motion.md` or the client (`AvatarView` fits the avatar and frame to its box), so these three sizes are the task's default, not a spec. At 64 px a large gear on top and six smaller gears around the ring still read. The smallest display size is open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-15 by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/frame/brass_gears/work/brass_gears_loop.py assets-src/cosmetic/frame/brass_gears/work assets-src/cosmetic/frame/brass_gears/export
```

It writes `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. Each frame is composed at 1024 x 1024 (the work canvas) in premultiplied alpha and Lanczos-resized to the canvas. A second run into an empty folder wrote byte-identical files.

| Choice | Value | Source |
|---|---|---|
| Canvas | 256 x 256 | Open (O-6). The set 1 frames are 512 x 512; 17 sprites at 512 would be 2,011,712 bytes by the ASTC 6x6 rule, about seven times the 300,000-byte limit of an animated avatar. 256 is the largest size of the three-size test, so the frame is never scaled up there and stays sharp at 128 and 64 (checked on the test sheet). The framing (2% margin around the bounding box, pivot at the centre) is the set 1 framing |
| Frames | 16 | ADR 0012 first measurement (16 frames per loop) |
| Frame rate | 12 fps, a loop of 1.33 s | ADR 0012 first measurement. The rate is not stored with the loop; the frame rate per class is open (O-5, O-6) |
| Loop content | Clockwork: the gear rims turn. Moving parts: the tooth rim of each of the 7 gears (`work/brass_gears_teeth.png`) turns about its centre by one tooth pitch per loop, 360 degrees divided by the tooth count estimated from the master (24, 20, 20, 16, 16, 19 and 19, the strongest angular repeat of the tooth ring; the painted teeth are not perfectly even), neighbouring gears in opposite directions. Each frame blends the rim turned forward by the share t of the loop (weight 1 - t) with the rim turned one pitch back (weight t), so the last frame flows into the first even where the painted teeth differ; t is 0 at frame 9. Hubs, spokes and faces stay still and only the teeth move: a gear face that turns with its painted light would need a paint-over. Ambient detail: a four-point sparkle (warm white, astroid shape, 16 to 26 work px) on each of the 3 rubies the script finds (red over green by 0.35, at least 50 px apart), lit for 5 frames once per loop at a start frame from seed 20261115 | Sheen and aura stay runtime (`docs/art-direction.md` sections 4 and 5.2); the moving parts and the ambient detail are drawn into the frames as 5.2 allows for frames. f counts frames from 0 |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 1.11 of 255, against 1.11 on average and 1.18 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 without the ambient detail | The rest frame: every moving part is at its rest position at frame 9. `export/static.png` is frame 9 drawn without the ambient detail; it differs from `export/anim/0009.png` by 0.000 of 255 on average (no ambient detail is lit at frame 9, so the two files hold the same pixels) |
| Size | 17 sprites, 502,928 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 29,584); frames have no limit yet (O-5) |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `2b73a51aa510fe633d6c58fef8fb42399afc792b68c22347556e66455ea0a86c` | 1254 x 1254 RGB, the generator's output, unchanged | Saved by Codex from `~/.codex/generated_images/01a1dfe1-1e8d-7060-a36c-16b55fca1645/` |
| concept/20261115-02.png | `144890085124e85eb584953b383f53f04a0f8fd966b4db3aa94dc4b600f251d5` | 1254 x 1254 RGB, the edit of the canonical concept, unchanged | Saved by Codex from `~/.codex/generated_images/01a1dfe5-5968-7850-9808-e6a04d0eb6a0/` |
| work/brass_gears_cleanup.py | `5e34e5d2919f7f5755d87a536bbbcd8801444203795050beb35f7d390fcd208a` | W-3 script | Written by Claude Code, 2026-11-15; the same script for every frame of this PR |
| work/brass_gears_master.png | `3a0f37a7b322ee05a770472a76ad0528b0321f2d90d69eb8516c4933e1a7a585` | 1254 x 1254 RGBA, the cut-out master | `brass_gears_cleanup.py` |
| work/brass_gears_plate_master.png | `e990d278155cdc27c556e96d836e840f05b8d2a01d884a55610aa9cb7e6d9224` | 1254 x 1254 RGBA, the cut-out edit | `brass_gears_cleanup.py` on the edit |
| work/brass_gears_parts.py | `8cb75948dfb3fa9da4ce2956d80472417d71e190af2d51d43ede146ca1fc00cf` | Script that splits the moving parts | Written by Claude Code, 2026-11-15 |
| work/brass_gears_teeth.png | `c347e81b26c06a906337de53f864545b72f45a7c1cc7c1cc6c01e4f5dcc2e624` | 1254 x 1254 RGBA, the moving layer | `brass_gears_parts.py` |
| work/brass_gears_under.png | `bb63eecb802c63a3520968eb0b9e17e5a5950ea5739287347d14f608d37dc714` | 1254 x 1254 RGBA, the still layer | `brass_gears_parts.py` |
| work/brass_gears_loop.py | `75059ea48c8026e998820deb66acc2c3d83160252022e1a6dd235c8efe409e91` | W-4 and W-5 script | Written by Claude Code, 2026-11-15 |
| export/static.png | `07fcfbb2df0f4f30e2c6ae666c596a40a0f68617cd4465e521bc13f9e230aa66` | 256 x 256 RGBA static | `brass_gears_loop.py`, frame 9 without ambient detail |
| export/anim/0001.png to 0016.png | per file in git | 256 x 256 RGBA loop frames | `brass_gears_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-4 with the style key, quoted below. Sent 15:43:54 UTC by Claude Code; `c2pa.created` 2026-11-15T15:44:41Z |
| 2026-11-15 | concept/20261115-02.png | Edit of the canonical concept: the same P-4 prompt plus one change, quoted below. Sent 15:48:31 UTC |
| 2026-11-15 | work/, export/ | The scripts above |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Ornamental round frame for a circular portrait, a ring of warm brass clockwork with several separate round toothed gears of different sizes sitting on the outer edge of the ring, the largest gear at the top, small rubies set in the ring, symmetric, open centre left empty, thin enough not to cover a face. Palette: warm brass, copper and dark jade enamel.
```

Codex was given the prompt inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261115-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Ornamental round frame for a circular portrait, a ring of warm brass clockwork with several separate round toothed gears of different sizes sitting on the outer edge of the ring, the largest gear at the top, small rubies set in the ring, symmetric, open centre left empty, thin enough not to cover a face. Palette: warm brass, copper and dark jade enamel. One change only: remove every gear completely and show the ring band, its enamel and its ornament continuing where the gears were. Keep the size, position, framing, light, colours and every other part of the image exactly the same.
```

Codex was given the canonical concept with `-i concept/20261115-01.png` and this instruction: `$imagegen Edit the attached image, our own canonical concept. <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept. Checked by Claude Code on 2026-11-15.
