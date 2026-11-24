# frame.season1_diamond

Animated rank frame of #173 (Rank çərçivələri, 10 pillə), the Diamond tier of `docs/ranking.md` section 3 (rank index 24 to 27). `docs/economy.md` section 4 names `frame.season1_gold` in the example bundle `bundle.season1_rank_gold`, the reward for a final Gold rank in `season_1`; this item is the same slot for Diamond, so its slug follows that id, as the season 1 medals of #711 did. The ten rank frames are one family that escalates with the tier: plain metal rings for Iron to Silver, gold, gems and ornament from Gold to Emerald, and an animated frame with its own motion for Diamond and the three apex tiers; each has its own outline (`docs/art-direction.md` 5.2). It is in no catalog, and no bundle grants it. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept generated on 2026-11-16 by an AI agent (Claude Code) running Codex CLI on the project owner's computer. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifest in the PNG names ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-16, `codex login status`: Logged in using ChatGPT)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-16 for the Plus plan and records the date here
- Canonical concept: concept/20261116-01.png
- Inputs uploaded to the generator: none; one prompt, one image, no edits, in a new Codex session (`01a1e36d-3ebf-7df3-8c88-fa278ae6f65a`)
- Prompt: P-4 with the style key of section 2.2 pasted verbatim; `{theme in plain words}` = "a ring of white gold set all around with small clear faceted diamonds, with a fan of tall clear crystal shards rising from the outer edge over the whole upper half, the tallest shards at the top and the shards getting shorter toward the sides, the lower half of the ring plain and slim", `{colour words}` = "white gold, colourless crystal and soft champagne"; quoted below
- Concept against the prompt: Matches the prompt in shape: a slim band paved all around with small round diamonds and a fan of eleven clear crystal shards over the upper half, tallest at the top. The generator painted the band in a warm champagne gold, not white gold. The crystal fan reads close to a tiara; no crown was asked for, and whether it reads as a royal emblem is left to the cultural review.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not applicable, no person is depicted
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; points for the cultural review are named under Concept against the prompt
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, source `rank_reward:bundle.season1_rank_diamond`. The bundle id follows the example of `docs/economy.md` section 4 and is a placeholder until the `season_1` rank rewards exist (#120, V-15); `RankRewards` holds a bundle per mode and final tier (`docs/ranking.md` 7.1), so the ids per mode are #120's call. Rarity across the ten frames rises with the tier: Iron and Bronze `common`, Silver, Gold and Platinum `rare`, Emerald and Diamond `epic`, Master and Grandmaster `legendary`, Challenger `mythic`; static frames stay at `epic` or below, as `docs/economy.md` section 3 expects a `legendary` or `mythic` item to be animated.

2026-11-16: owner decision — the ten rank frames keep the tier ladder stated above, an exception to the animated-detail rule of `docs/rarity.md` 1.1, so `epic`. It replaces the earlier note of the same day that proposed `legendary` from its 2 animated details (shard glints, orbiting sparkles).

## Embedded C2PA manifest

`concept/20261116-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-16; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-16T08:16:30Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 08:15:49 UTC on 2026-11-16 (agent clock), Codex session `01a1e36d-3ebf-7df3-8c88-fa278ae6f65a`. Codex saved its image from `~/.codex/generated_images/01a1e36d-3ebf-7df3-8c88-fa278ae6f65a/` into the agent's scratch folder, and the agent copied it into `concept/`; the files have the same sha256. No generation for this item was rejected: this is the only image generated for it.

## Clean-up (W-3)

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/frame/season1_diamond/work/season1_diamond_cleanup.py season1_diamond assets-src/cosmetic/frame/season1_diamond/concept/20261116-01.png assets-src/cosmetic/frame/season1_diamond/work
```

It printed `season1_diamond: background [186.0, 186.0, 186.0], box (11, 0, 1235, 1225), opening radius 0.640 of half box`. A second run of every script into an empty folder wrote byte-identical files.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | The frame clean-up of the animated set 1 frames (`frame/storm_halo/work/storm_halo_cleanup.py`, #703). One change for this item: a background pocket must also be neutral (saturation under 4 of 255, `POCKET_SATURATION`), because the flat facets of the crystals lie within 14 of the grey and were cut out as holes in the first run; the facets carry a faint warm tint (about 8 of 255 red over green) that the grey does not. The loop script places the master on a square centred on its bounding box with 2% margin on each side, as set 1 did |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Not needed | Nothing in the master moves; the sparkles are drawn by `work/season1_diamond_loop.py` |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen in the concept, the master or the export and the loop frames; the ambient detail is drawn without glow. The static shows the metal and gems at rest with no aura, sheen or ambient detail (`docs/art-direction.md` section 7) |
| 5. Silhouette and smallest-size tests | In part | The agent drew the static and loop frame 5 around `avatar.court_poet` set inside the opening (the avatar scaled to 1.04 times the opening diameter, the frame on top) at 256, 128 and 64 px on `color.background.base`. No avatar size is set in `docs/ui-motion.md`, `docs/art-direction.md` (5.1 and section 10 leave the frame band and the smallest display size to #168) or the client (`AvatarView` fits the avatar and frame to its box, `docs/cosmetics.md` 8.3), so these three sizes are the task's default, not a spec, and no size variant is exported: the one 256 x 256 export is scaled down by the view. At 64 px the jagged crystal crest and the light band read; the glints are 2 to 3 px there. The smallest display size is open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-16 by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/frame/season1_diamond/work/season1_diamond_loop.py assets-src/cosmetic/frame/season1_diamond/work assets-src/cosmetic/frame/season1_diamond/export
```

It printed `season1_diamond_loop: 9 crystal tips, band radius 348.4 work px; wrote 16 frames and the static (frame 9 without ambient detail) at 256x256`. It writes `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. Each frame is composed at 1024 x 1024 (the work canvas) in premultiplied alpha and Lanczos-resized to the canvas. A second run into an empty folder wrote byte-identical files.

| Choice | Value | Source |
|---|---|---|
| Canvas | 256 x 256 | The canvas of the eight animated set 1 frames (#703), whose provenance gives the reason (17 sprites at 512 would be about seven times the animated avatar limit). Open (O-6) |
| Frames | 16 | ADR 0012 first measurement (16 frames per loop), as #703 |
| Frame rate | 12 fps, a loop of 1.33 s | ADR 0012 first measurement. The rate is not stored with the loop; the frame rate per class is open (O-5, O-6) |
| Loop content | Crystal glints. No part of the master moves, so no painted light turns. Ambient detail: a four-point sparkle (warm white, astroid shape, up to 50 work px, the `paint_glint` of #703) at the tip of each of the 9 shards the script finds (bright, low-saturation, opaque parts above the band and outside 1.12 times the opening radius, at least 300 px after a 2 px opening, tips at least 60 px apart; the two smallest shards at the ends are not found), lit for 5 frames once per loop at a start frame from seed 20261116; and two small sparkles (24 work px) that run around the paved band at its median radius, opposite each other, half a turn per loop | Decision D-3 of the Epic 15 plan: an offline 2D rig, parts and drawn ambient detail moved by a deterministic script. Sheen and aura stay runtime (`docs/art-direction.md` sections 4 and 5.2). f counts frames from 0 |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 0.64 of 255, against 0.46 on average and 0.67 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 without the ambient detail, which is the cleaned master as it is | `export/static.png` differs from `export/anim/0009.png` by 0.243 of 255 on average |
| Size | 17 sprites, 502,928 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 29,584); frames have no limit yet (O-5) |

Flashing: sparkles cover well under 1% of the canvas and at most four show at once; nothing flashes the whole frame. The photosensitivity limits are admin config (`docs/localization.md` section 6) and this loop was not measured against them.

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261116-01.png | `bc1ddf38d54e6b367adf1f26e2f45175ec95514e04c14ae29cb51db1607c021d` | 1254 x 1254 RGB, the generator's output, unchanged | Codex CLI `$imagegen`; byte-identical to the file in `~/.codex/generated_images/01a1e36d-3ebf-7df3-8c88-fa278ae6f65a/` (checked with sha256) |
| work/season1_diamond_cleanup.py | `ad498811b78082790b3cbd9fe1d7961b86687ff222d10b77a7bd01a99abe7b54` | W-3 script | Copied from `frame/storm_halo/work/storm_halo_cleanup.py` (#703) with the pocket change above |
| work/season1_diamond_master.png | `abc752c4078edcf75ce0e454d35b935aa2c60f9edda60df0da49d762f8eca51d` | 1254 x 1254 RGBA, the cut-out master | `season1_diamond_cleanup.py` |
| work/season1_diamond_loop.py | `9300396df367e52931aede6bd100edcd598f86ce4708def532021062b7b728d9` | W-4 and W-5 script | Written by Claude Code, 2026-11-16, from the loop scripts of #703 |
| export/static.png | `10b3654c290961fb7a75138910f11faa72fef1dd8b991640771cc5f2576ccf1a` | 256 x 256 RGBA static | `season1_diamond_loop.py`, frame 9 without ambient detail |
| export/anim/0001.png to 0016.png | per file in git | 256 x 256 RGBA loop frames | `season1_diamond_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-16 | concept/20261116-01.png | P-4 with the style key, quoted below. Sent 08:15:49 UTC by Claude Code; `c2pa.created` 2026-11-16T08:16:30Z |
| 2026-11-16 | work/, export/ | The scripts above |

Prompt of `concept/20261116-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Ornamental round frame for a circular portrait, a ring of white gold set all around with small clear faceted diamonds, with a fan of tall clear crystal shards rising from the outer edge over the whole upper half, the tallest shards at the top and the shards getting shorter toward the sides, the lower half of the ring plain and slim, symmetric, open centre left empty, thin enough not to cover a face. Palette: white gold, colourless crystal and soft champagne.
```

Codex was given the prompt inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261116-01.png. Do not use any API key and do not edit the image.`

RC-2 against this prompt and the Inputs line: it names no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-16.
