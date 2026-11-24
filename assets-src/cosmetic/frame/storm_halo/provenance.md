# frame.storm_halo

Animated frame of #173 (batch A of the 8 animated frames), an ornamental ring made with template P-4 of `docs/asset-workflow.md` section 2 and animated offline: parts of the cleaned master and drawn ambient detail moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifest in the PNG names ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: none; one prompt, one image, no edits
- Prompt: P-4 with the style key of section 2.2 pasted verbatim; `{theme in plain words}` = "a ring of brushed silver shaped as rolling storm-cloud scrolls, the clouds swelling into a larger billow at the top and a smaller one at the bottom, set with violet amethysts, the band itself bare of any lightning", `{colour words}` = "brushed silver, storm violet and pale gold"; quoted below
- Concept against the prompt: Matches the prompt: a brushed silver ring of cloud scrolls with a large billow at the top and a smaller one at the bottom, gold mounts with violet amethysts at the top, bottom and both sides and small round amethysts between them. No lightning is painted in, as asked, so every bolt of the loop is drawn by the script.
- Similarity check: not done (RC-3 open, O-7)
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the masters, layers and scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; points for the cultural review are named under Concept against the prompt
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `legendary`, source `shop:offer.frame_storm_halo`. The offer id is a placeholder until the shop record exists (V-15). Animated frames are proposed at `epic` and above, as the batch leans to the higher tiers of `docs/rarity.md`.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 2 animated details (breathing billows, lightning bolts), so `legendary`, was `mythic`. Any tier split stated above is superseded.

## Embedded C2PA manifest

`concept/20261115-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-15T15:38:43Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 15:38:01 UTC on 2026-11-15 (agent clock), Codex session `01a1dfdb-ba58-7943-87cb-24e772e12382`. Codex saved its image from `~/.codex/generated_images/01a1dfdb-ba58-7943-87cb-24e772e12382/` into `concept/`; the two files have the same sha256. No generation for this item was rejected: this is the only image generated for it.

## Clean-up (W-3)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/frame/storm_halo/work/storm_halo_cleanup.py storm_halo assets-src/cosmetic/frame/storm_halo/concept/20261115-01.png assets-src/cosmetic/frame/storm_halo/work
```

The clean-up printed background `[189.0, 189.0, 190.0]`, box `(17, -13, 1236, 1205)`, opening radius 0.727 of the half box. A second run of every script into an empty folder wrote byte-identical files.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | The set 1 frame clean-up (`frame/ember_crown/work/ember_crown_cleanup.py`) with two changes: the script writes only the master (the static and the loop come from the loop script), and background pockets inside the ornament are found by flatness, so light metal close to the grey is kept. A pocket is a region within 14 of the grey whose local spread (standard deviation of luminance over 5 x 5 px) has a median under 1.2, or pixels within 6 of the grey with a spread under 1.2, in parts of at least 20 px; it grows over neighbouring pixels within 40 of the grey with a saturation under 12 and a spread under 4. Everything else is as in set 1: border median colour, edge-connected background within 30, the open centre, specks under 200 px dropped, 1 px erosion, feather sigma 0.8, fringe unmixed. The loop script places the master on a square centred on its bounding box with 2% margin on each side, as set 1 did, so the opening keeps the share printed above |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | In part | No separate layer: the billow region is a weight field in `work/storm_halo_loop.py`, and the bolts are drawn by it |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen in the concept, the master or the frames; the ambient detail is drawn without glow. The static shows the metal and gems at rest with no aura, sheen or ambient detail (`docs/art-direction.md` section 7) |
| 5. Silhouette and smallest-size tests | In part | The agent drew the static and loop frame 5 around `avatar.court_poet` set inside the opening at 256, 128 and 64 px on `color.background.base`. No avatar size is set in `docs/ui-motion.md` or the client (`AvatarView` fits the avatar and frame to its box), so these three sizes are the task's default, not a spec. At 64 px a cloud crest on top, a smaller billow at the bottom and four diamond mounts still read. The smallest display size is open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-15 by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/frame/storm_halo/work/storm_halo_loop.py assets-src/cosmetic/frame/storm_halo/work assets-src/cosmetic/frame/storm_halo/export
```

It writes `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. Each frame is composed at 1024 x 1024 (the work canvas) in premultiplied alpha and Lanczos-resized to the canvas. A second run into an empty folder wrote byte-identical files.

| Choice | Value | Source |
|---|---|---|
| Canvas | 256 x 256 | Open (O-6). The set 1 frames are 512 x 512; 17 sprites at 512 would be 2,011,712 bytes by the ASTC 6x6 rule, about seven times the 300,000-byte limit of an animated avatar. 256 is the largest size of the three-size test, so the frame is never scaled up there and stays sharp at 128 and 64 (checked on the test sheet). The framing (2% margin around the bounding box, pivot at the centre) is the set 1 framing |
| Frames | 16 | ADR 0012 first measurement (16 frames per loop) |
| Frame rate | 12 fps, a loop of 1.33 s | ADR 0012 first measurement. The rate is not stored with the loop; the frame rate per class is open (O-5, O-6) |
| Loop content | Storm: the cloud billows breathe and lightning flicks off the ring. Moving part: the top billow (work rows above 330 of 1024) lifts by up to 9 work px with sin(2 pi f / 16), fading to none at row 330; the bottom billow (rows below 880) moves the opposite way at 60%. Ambient detail: 8 bolts drawn by the script (seed 20261115), each a jagged 6-segment line with one fork, from the band at radius 418 out to 585 work px, at 36, 58, 122, 144, 216, 238, 302 and 324 degrees from the top; a white-lilac core and a violet rim, tapered, no glow. Each bolt shows once per loop for two frames (full, then 45%), starting at frames 1, 3, 6, 8, 9, 11, 14 and 16, so at most two bolts show at once | Sheen and aura stay runtime (`docs/art-direction.md` sections 4 and 5.2); the moving parts and the ambient detail are drawn into the frames as 5.2 allows for frames. f counts frames from 0 |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 2.88 of 255, against 2.13 on average and 3.24 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 without the ambient detail | The rest frame: every moving part is at its rest position at frame 9. `export/static.png` is frame 9 drawn without the ambient detail; it differs from `export/anim/0009.png` by 1.430 of 255 on average |
| Size | 17 sprites, 502,928 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 29,584); frames have no limit yet (O-5) |

Flashing: a bolt covers well under 1% of the canvas and at most two show at once; nothing flashes the whole frame. The photosensitivity limits are admin config (`docs/localization.md` section 6) and this loop was not measured against them.

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `61380aa0f67ac294c12edc5a9bef7dd605f70f1f492e28923f613be3216cf7da` | 1254 x 1254 RGB, the generator's output, unchanged | Saved by Codex from `~/.codex/generated_images/01a1dfdb-ba58-7943-87cb-24e772e12382/` |
| work/storm_halo_cleanup.py | `5e34e5d2919f7f5755d87a536bbbcd8801444203795050beb35f7d390fcd208a` | W-3 script | Written by Claude Code, 2026-11-15; the same script for every frame of this PR |
| work/storm_halo_master.png | `d3042f1cb4aac4baf0f3effe3908b1d93cfe86221ffa353324fe72c23b25c522` | 1254 x 1254 RGBA, the cut-out master | `storm_halo_cleanup.py` |
| work/storm_halo_loop.py | `e7c5dc722dfc556c0c300930835eb160dadeb5d60c7901f360b7712e25910919` | W-4 and W-5 script | Written by Claude Code, 2026-11-15 |
| export/static.png | `7245b6464191de321f5752a52cb7204bbac94f4b15d38e764169528df921c217` | 256 x 256 RGBA static | `storm_halo_loop.py`, frame 9 without ambient detail |
| export/anim/0001.png to 0016.png | per file in git | 256 x 256 RGBA loop frames | `storm_halo_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-4 with the style key, quoted below. Sent 15:38:01 UTC by Claude Code; `c2pa.created` 2026-11-15T15:38:43Z |
| 2026-11-15 | work/, export/ | The scripts above |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Ornamental round frame for a circular portrait, a ring of brushed silver shaped as rolling storm-cloud scrolls, the clouds swelling into a larger billow at the top and a smaller one at the bottom, set with violet amethysts, the band itself bare of any lightning, symmetric, open centre left empty, thin enough not to cover a face. Palette: brushed silver, storm violet and pale gold.
```

Codex was given the prompt inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

RC-2 against this prompt and the Inputs line: it names no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-15.
