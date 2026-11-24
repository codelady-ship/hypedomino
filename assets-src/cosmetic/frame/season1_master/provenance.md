# frame.season1_master

Animated rank frame of #173 (Rank çərçivələri, 10 pillə), the Master tier of `docs/ranking.md` section 3 (rank index 28, apex tier). `docs/economy.md` section 4 names `frame.season1_gold` in the example bundle `bundle.season1_rank_gold`, the reward for a final Gold rank in `season_1`; this item is the same slot for Master, so its slug follows that id, as the season 1 medals of #711 did. The ten rank frames are one family that escalates with the tier: plain metal rings for Iron to Silver, gold, gems and ornament from Gold to Emerald, and an animated frame with its own motion for Diamond and the three apex tiers; each has its own outline (`docs/art-direction.md` 5.2). It is in no catalog, and no bundle grants it. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept generated on 2026-11-16 by an AI agent (Claude Code) running Codex CLI on the project owner's computer. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifest in the PNG names ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-16, `codex login status`: Logged in using ChatGPT)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-16 for the Plus plan and records the date here
- Canonical concept: concept/20261116-01.png
- Inputs uploaded to the generator: none; one prompt, one image, no edits, in a new Codex session (`01a1e36e-35b1-7701-8f78-694fa5ec7df3`)
- Prompt: P-4 with the style key of section 2.2 pasted verbatim; `{theme in plain words}` = "a ring of black lacquer banded in polished gold and set with deep violet amethysts, with one large sweeping gold acanthus scroll on each side curling upward and outward like a pair of handles, and one tall pointed amethyst in a gold mount at the top", `{colour words}` = "black lacquer, polished gold and deep violet"; quoted below
- Concept against the prompt: Matches the prompt: a black lacquer band with gold collars, seven round amethysts in gold mounts, a marquise amethyst in a gold acanthus mount at the top, a small acanthus mount at the bottom, and a large gold acanthus scroll on each side. The rendering is flatter and more saturated than the other items of this PR.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not applicable, no person is depicted
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; points for the cultural review are named under Concept against the prompt
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `legendary`, source `rank_reward:bundle.season1_rank_master`. The bundle id follows the example of `docs/economy.md` section 4 and is a placeholder until the `season_1` rank rewards exist (#120, V-15); `RankRewards` holds a bundle per mode and final tier (`docs/ranking.md` 7.1), so the ids per mode are #120's call. Rarity across the ten frames rises with the tier: Iron and Bronze `common`, Silver, Gold and Platinum `rare`, Emerald and Diamond `epic`, Master and Grandmaster `legendary`, Challenger `mythic`; static frames stay at `epic` or below, as `docs/economy.md` section 3 expects a `legendary` or `mythic` item to be animated.

2026-11-16: owner decision — the ten rank frames keep the tier ladder stated above, an exception to the animated-detail rule of `docs/rarity.md` 1.1, so `legendary`. The earlier note of the same day reached the same tier from its 2 animated details (scroll sway, amethyst glints); that rule no longer applies to this frame.

## Embedded C2PA manifest

`concept/20261116-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-16; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-16T08:17:43Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 08:16:52 UTC on 2026-11-16 (agent clock), Codex session `01a1e36e-35b1-7701-8f78-694fa5ec7df3`. Codex saved its image from `~/.codex/generated_images/01a1e36e-35b1-7701-8f78-694fa5ec7df3/` into the agent's scratch folder, and the agent copied it into `concept/`; the files have the same sha256. No generation for this item was rejected: this is the only image generated for it.

## Clean-up (W-3)

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/frame/season1_master/work/season1_master_cleanup.py season1_master assets-src/cosmetic/frame/season1_master/concept/20261116-01.png assets-src/cosmetic/frame/season1_master/work
```

It printed `season1_master: background [185.0, 184.0, 184.0], box (7, -6, 1245, 1233), opening radius 0.710 of half box`. A second run of every script into an empty folder wrote byte-identical files.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | The frame clean-up of the animated set 1 frames (`frame/storm_halo/work/storm_halo_cleanup.py`, #703). Unchanged. The loop script places the master on a square centred on its bounding box with 2% margin on each side, as set 1 did |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | In part | No separate layer: the scroll region is a weight field in `work/season1_master_loop.py`, and the sparkles are drawn by it |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen in the concept, the master or the export and the loop frames; the ambient detail is drawn without glow. The static shows the metal and gems at rest with no aura, sheen or ambient detail (`docs/art-direction.md` section 7) |
| 5. Silhouette and smallest-size tests | In part | The agent drew the static and loop frame 5 around `avatar.court_poet` set inside the opening (the avatar scaled to 1.04 times the opening diameter, the frame on top) at 256, 128 and 64 px on `color.background.base`. No avatar size is set in `docs/ui-motion.md`, `docs/art-direction.md` (5.1 and section 10 leave the frame band and the smallest display size to #168) or the client (`AvatarView` fits the avatar and frame to its box, `docs/cosmetics.md` 8.3), so these three sizes are the task's default, not a spec, and no size variant is exported: the one 256 x 256 export is scaled down by the view. At 64 px the two side scrolls and the marquise at the top read; the black band stays visible on `color.background.base` through its gold edges. The smallest display size is open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-16 by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/frame/season1_master/work/season1_master_loop.py assets-src/cosmetic/frame/season1_master/work assets-src/cosmetic/frame/season1_master/export
```

It printed `season1_master_loop: 8 amethysts; wrote 16 frames and the static (frame 9 without ambient detail) at 256x256`. It writes `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. Each frame is composed at 1024 x 1024 (the work canvas) in premultiplied alpha and Lanczos-resized to the canvas. A second run into an empty folder wrote byte-identical files.

| Choice | Value | Source |
|---|---|---|
| Canvas | 256 x 256 | The canvas of the eight animated set 1 frames (#703), whose provenance gives the reason (17 sprites at 512 would be about seven times the animated avatar limit). Open (O-6) |
| Frames | 16 | ADR 0012 first measurement (16 frames per loop), as #703 |
| Frame rate | 12 fps, a loop of 1.33 s | ADR 0012 first measurement. The rate is not stored with the loop; the frame rate per class is open (O-5, O-6) |
| Loop content | Swaying scrolls and amethyst glints. Moving part: the outer side scrolls (work columns more than 410 px from the opening centre, full weight beyond 505 px, smoothstep between) lift and drop by up to 14 work px with sin(2 pi f / 16), both sides together, so the band and the top and bottom mounts stay still. Ambient detail: a four-point sparkle (white-lilac, up to 34 work px) on each of the 8 amethysts the script finds (blue over green by 0.15, red over green by 0.05, opaque, at least 150 px after a 2 px opening, 50 px apart), lit for 4 frames in turn clockwise from the top, two frames apart, moving with the sway where a stone sits on a scroll | Decision D-3 of the Epic 15 plan: an offline 2D rig, parts and drawn ambient detail moved by a deterministic script. Sheen and aura stay runtime (`docs/art-direction.md` sections 4 and 5.2). f counts frames from 0 |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 0.75 of 255, against 0.53 on average and 0.75 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 without the ambient detail: the scrolls are at rest at frame 9 | `export/static.png` differs from `export/anim/0009.png` by 0.101 of 255 on average |
| Size | 17 sprites, 502,928 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 29,584); frames have no limit yet (O-5) |

Flashing: one or two sparkles show at a time, well under 1% of the canvas; nothing flashes the whole frame. The photosensitivity limits are admin config (`docs/localization.md` section 6) and this loop was not measured against them.

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261116-01.png | `99b55ce7af7701e3ebcaf06b4b0f0e2a94cb1805df1082d845d39b4f0206a6d7` | 1254 x 1254 RGB, the generator's output, unchanged | Codex CLI `$imagegen`; byte-identical to the file in `~/.codex/generated_images/01a1e36e-35b1-7701-8f78-694fa5ec7df3/` (checked with sha256) |
| work/season1_master_cleanup.py | `5e34e5d2919f7f5755d87a536bbbcd8801444203795050beb35f7d390fcd208a` | W-3 script | Copied unchanged from `frame/storm_halo/work/storm_halo_cleanup.py` (#703) |
| work/season1_master_master.png | `27d2afa900b7bf185e69cf6e0894922c1bd37aaa0a1ee376c36ca1bca4860a70` | 1254 x 1254 RGBA, the cut-out master | `season1_master_cleanup.py` |
| work/season1_master_loop.py | `5155482295d2cf1e5e398f8b9096c9bca3bd19b47d1d6b8379747a3c88e7ce70` | W-4 and W-5 script | Written by Claude Code, 2026-11-16, from the loop scripts of #703 |
| export/static.png | `10a267d89a52435de5d0132db521434c27342a83311f9cffc48addea5e91fdfb` | 256 x 256 RGBA static | `season1_master_loop.py`, frame 9 without ambient detail |
| export/anim/0001.png to 0016.png | per file in git | 256 x 256 RGBA loop frames | `season1_master_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-16 | concept/20261116-01.png | P-4 with the style key, quoted below. Sent 08:16:52 UTC by Claude Code; `c2pa.created` 2026-11-16T08:17:43Z |
| 2026-11-16 | work/, export/ | The scripts above |

Prompt of `concept/20261116-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Ornamental round frame for a circular portrait, a ring of black lacquer banded in polished gold and set with deep violet amethysts, with one large sweeping gold acanthus scroll on each side curling upward and outward like a pair of handles, and one tall pointed amethyst in a gold mount at the top, symmetric, open centre left empty, thin enough not to cover a face. Palette: black lacquer, polished gold and deep violet.
```

Codex was given the prompt inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261116-01.png. Do not use any API key and do not edit the image.`

RC-2 against this prompt and the Inputs line: it names no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-16.
