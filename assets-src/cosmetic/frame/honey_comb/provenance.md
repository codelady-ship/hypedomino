# frame.honey_comb

Animated frame of #173 (batch A of the 8 animated frames), an ornamental ring made with template P-4 of `docs/asset-workflow.md` section 2 and animated offline: parts of the cleaned master and drawn ambient detail moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifest in the PNG names ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: none; one prompt, one image, no edits
- Prompt: P-4 with the style key of section 2.2 pasted verbatim; `{theme in plain words}` = "a ring built from golden honeycomb cells, round amber honey drops hanging from the lower cells, a larger cluster of cells at the top", `{colour words}` = "honey amber, beeswax gold and dark walnut brown"; quoted below
- Concept against the prompt: Matches the prompt and adds more: a ring of gold-rimmed hexagonal cells filled with amber honey, a cluster of cells at the top, honey drops hanging from the cells, and gold and walnut acanthus leaves between the cells, which the prompt did not ask for. Drops also hang from the top cluster into the opening, which the prompt did not ask for; at 256 px the longest reaches the top of the avatar's hair in the size test below.
- Similarity check: not done (RC-3 open, O-7)
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the masters, layers and scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; points for the cultural review are named under Concept against the prompt
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `legendary`, source `shop:offer.frame_honey_comb`. The offer id is a placeholder until the shop record exists (V-15). Animated frames are proposed at `epic` and above, as the batch leans to the higher tiers of `docs/rarity.md`.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 2 animated details (stretching drips, falling drops), so `legendary`, was `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifest

`concept/20261115-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-15T15:47:18Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 15:46:29 UTC on 2026-11-15 (agent clock), Codex session `01a1dfe3-7bc5-7962-8146-d9228e343751`. Codex saved its image from `~/.codex/generated_images/01a1dfe3-7bc5-7962-8146-d9228e343751/` into `concept/`; the two files have the same sha256. No generation for this item was rejected: this is the only image generated for it.

## Clean-up (W-3)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/frame/honey_comb/work/honey_comb_cleanup.py honey_comb assets-src/cosmetic/frame/honey_comb/concept/20261115-01.png assets-src/cosmetic/frame/honey_comb/work
```

The clean-up printed background `[208.0, 207.0, 207.0]`, box `(12, -17, 1240, 1211)`, opening radius 0.628 of the half box. A second run of every script into an empty folder wrote byte-identical files.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | The set 1 frame clean-up (`frame/ember_crown/work/ember_crown_cleanup.py`) with two changes: the script writes only the master (the static and the loop come from the loop script), and background pockets inside the ornament are found by flatness, so light metal close to the grey is kept. A pocket is a region within 14 of the grey whose local spread (standard deviation of luminance over 5 x 5 px) has a median under 1.2, or pixels within 6 of the grey with a spread under 1.2, in parts of at least 20 px; it grows over neighbouring pixels within 40 of the grey with a saturation under 12 and a spread under 4. Everything else is as in set 1: border median colour, edge-connected background within 30, the open centre, specks under 200 px dropped, 1 px erosion, feather sigma 0.8, fringe unmixed. The loop script places the master on a square centred on its bounding box with 2% margin on each side, as set 1 did, so the opening keeps the share printed above |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | In part | No separate layer: the drip regions are a displacement field in `work/honey_comb_loop.py`, and the falling drops are drawn by it |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen in the concept, the master or the frames; the ambient detail is drawn without glow. The static shows the metal and gems at rest with no aura, sheen or ambient detail (`docs/art-direction.md` section 7) |
| 5. Silhouette and smallest-size tests | In part | The agent drew the static and loop frame 5 around `avatar.court_poet` set inside the opening at 256, 128 and 64 px on `color.background.base`. No avatar size is set in `docs/ui-motion.md` or the client (`AvatarView` fits the avatar and frame to its box), so these three sizes are the task's default, not a spec. At 64 px a cell cluster on top and a hexagon ring with hanging drops still read. The smallest display size is open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-15 by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/frame/honey_comb/work/honey_comb_loop.py assets-src/cosmetic/frame/honey_comb/work assets-src/cosmetic/frame/honey_comb/export
```

It writes `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. Each frame is composed at 1024 x 1024 (the work canvas) in premultiplied alpha and Lanczos-resized to the canvas. A second run into an empty folder wrote byte-identical files.

| Choice | Value | Source |
|---|---|---|
| Canvas | 256 x 256 | Open (O-6). The set 1 frames are 512 x 512; 17 sprites at 512 would be 2,011,712 bytes by the ASTC 6x6 rule, about seven times the 300,000-byte limit of an animated avatar. 256 is the largest size of the three-size test, so the frame is never scaled up there and stays sharp at 128 and 64 (checked on the test sheet). The framing (2% margin around the bounding box, pivot at the centre) is the set 1 framing |
| Frames | 16 | ADR 0012 first measurement (16 frames per loop) |
| Frame rate | 12 fps, a loop of 1.33 s | ADR 0012 first measurement. The rate is not stored with the loop; the frame rate per class is open (O-5, O-6) |
| Loop content | Honey: the drips stretch and drops fall. Moving parts: 12 drips (tips measured on the master and listed in `DRIPS`) stretch downward by up to 14 work px: 9 swell and shrink with sin^2(pi (f - 8) / 16), 3 creep over the whole loop and snap back at frame 9. Ambient detail: a drop drawn by the script (amber teardrop, radius 10 work px, with a highlight) leaves each of the 3 creeping drips at frame 9 and 3 of the swelling lower drips at frame 1, falling 75 work px with gravity over 5 frames while it fades. The 5 drips of the top cluster and the bottom centre drip only stretch, so no drop falls over the face or off the canvas edge | Sheen and aura stay runtime (`docs/art-direction.md` sections 4 and 5.2); the moving parts and the ambient detail are drawn into the frames as 5.2 allows for frames. f counts frames from 0 |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 0.29 of 255, against 0.60 on average and 0.92 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 without the ambient detail | The rest frame: every moving part is at its rest position at frame 9. `export/static.png` is frame 9 drawn without the ambient detail; it differs from `export/anim/0009.png` by 0.053 of 255 on average |
| Size | 17 sprites, 502,928 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 29,584); frames have no limit yet (O-5) |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `c021ed60edfa66267550891ec9ee1fcd53e1c39246928ee09e2281febf44250e` | 1254 x 1254 RGB, the generator's output, unchanged | Saved by Codex from `~/.codex/generated_images/01a1dfe3-7bc5-7962-8146-d9228e343751/` |
| work/honey_comb_cleanup.py | `5e34e5d2919f7f5755d87a536bbbcd8801444203795050beb35f7d390fcd208a` | W-3 script | Written by Claude Code, 2026-11-15; the same script for every frame of this PR |
| work/honey_comb_master.png | `e1434389787f9908dfc91444077884df8d5bc7044a73e5d6552e2d548c921db2` | 1254 x 1254 RGBA, the cut-out master | `honey_comb_cleanup.py` |
| work/honey_comb_loop.py | `0504ad9394606cca90c9c246234086c8ede61289d79092f0e1bad3af4544197a` | W-4 and W-5 script | Written by Claude Code, 2026-11-15 |
| export/static.png | `e90616fa80f582a92ae3bd3334dfdf5ca9a72e15c8b5fdd22db2beb72ec3f77e` | 256 x 256 RGBA static | `honey_comb_loop.py`, frame 9 without ambient detail |
| export/anim/0001.png to 0016.png | per file in git | 256 x 256 RGBA loop frames | `honey_comb_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-4 with the style key, quoted below. Sent 15:46:29 UTC by Claude Code; `c2pa.created` 2026-11-15T15:47:18Z |
| 2026-11-15 | work/, export/ | The scripts above |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Ornamental round frame for a circular portrait, a ring built from golden honeycomb cells, round amber honey drops hanging from the lower cells, a larger cluster of cells at the top, symmetric, open centre left empty, thin enough not to cover a face. Palette: honey amber, beeswax gold and dark walnut brown.
```

Codex was given the prompt inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

RC-2 against this prompt and the Inputs line: it names no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-15.
