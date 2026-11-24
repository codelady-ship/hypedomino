# frame.season1_bronze

Static rank frame of #173 (Rank çərçivələri, 10 pillə), the Bronze tier of `docs/ranking.md` section 3 (rank index 4 to 7). `docs/economy.md` section 4 names `frame.season1_gold` in the example bundle `bundle.season1_rank_gold`, the reward for a final Gold rank in `season_1`; this item is the same slot for Bronze, so its slug follows that id, as the season 1 medals of #711 did. The ten rank frames are one family that escalates with the tier: plain metal rings for Iron to Silver, gold, gems and ornament from Gold to Emerald, and an animated frame with its own motion for Diamond and the three apex tiers; each has its own outline (`docs/art-direction.md` 5.2). It is in no catalog, and no bundle grants it. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept generated on 2026-11-16 by an AI agent (Claude Code) running Codex CLI on the project owner's computer, as one sheet of the six static rank frames. W-3 clean-up and W-5 export by the same agent with `work/season1_bronze_cleanup.py`. No person has done W-3
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifest in the PNG names ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-16, `codex login status`: Logged in using ChatGPT)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-16 for the Plus plan and records the date here
- Canonical concept: concept/20261116-01.png, a shared sheet (sha256 `2c69e85351ed26d62d1120546b9a9c74925008c1a113c39cfa3ee7efce80f63c`), cell 2 of 2 x 3 (row 1, column 2)
- Inputs uploaded to the generator: none; one prompt, one image, no edits, in a new Codex session (`01a1e36a-d8a9-75c2-8158-564bae5affc1`)
- Prompt: the style key of `docs/asset-workflow.md` 2.2 pasted verbatim, then a sheet prompt for six frames, quoted below. It repeats the wording of template P-4 (round frame for a circular portrait, symmetric, open centre left empty, thin enough not to cover a face) and gives each cell its theme and palette in plain words; one sheet instead of six P-4 calls saves image turns, as the medal sheets of #711 did
- Concept against the prompt: Matches the prompt: a band of dark olive-patinated bronze between two twisted rope borders, with three round bronze studs grouped at the top. The rope borders are a warmer gold-bronze than the deep brown asked for.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not applicable, no person is depicted
- Human work: none. Clean-up and export are scripted image operations run by an AI agent, so RC-5 stays open; the master and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; points for the cultural review are named under Concept against the prompt
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `common`, source `rank_reward:bundle.season1_rank_bronze`. The bundle id follows the example of `docs/economy.md` section 4 and is a placeholder until the `season_1` rank rewards exist (#120, V-15); `RankRewards` holds a bundle per mode and final tier (`docs/ranking.md` 7.1), so the ids per mode are #120's call. Rarity across the ten frames rises with the tier: Iron and Bronze `common`, Silver, Gold and Platinum `rare`, Emerald and Diamond `epic`, Master and Grandmaster `legendary`, Challenger `mythic`; static frames stay at `epic` or below, as `docs/economy.md` section 3 expects a `legendary` or `mythic` item to be animated.

2026-11-16: owner decision — the ten rank frames keep the tier ladder stated above, an exception to the animated-detail rule of `docs/rarity.md` 1.1, so `common`.

## Embedded C2PA manifest

`concept/20261116-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-16; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-16T08:14:39Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 08:13:12 UTC on 2026-11-16 (agent clock), Codex session `01a1e36a-d8a9-75c2-8158-564bae5affc1`. Codex saved its image from `~/.codex/generated_images/01a1e36a-d8a9-75c2-8158-564bae5affc1/` into the agent's scratch folder, and the agent copied it into `concept/`; the files have the same sha256. No generation of the sheet was rejected. An earlier call with the same sheet prompt at 08:08:59 UTC never reached the generator: the Codex CLI waited for input on stdin, started no session and made no image, and the agent stopped it; the call above closed stdin.

## Clean-up (W-3) and export (W-5)

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/frame/season1_bronze/work/season1_bronze_cleanup.py season1_bronze assets-src/cosmetic/frame/season1_bronze/concept/20261116-01.png 2 3 2 assets-src/cosmetic/frame/season1_bronze/work assets-src/cosmetic/frame/season1_bronze/export
```

It printed `season1_bronze: background [210.0, 209.0, 210.0], cell 2 of 2x3, crop (438, 121, 805, 495), box (-4, 0, 370, 373), opening radius 0.673 of half box, canvas 256`; crop is the master's place on the sheet, box the square around the frame inside the master. A second run into an empty folder wrote byte-identical files and the same line.

The script is the frame clean-up of the animated set 1 frames (`frame/storm_halo/work/storm_halo_cleanup.py`, #703) changed for a sheet, the way the medal clean-up of #711 cuts a cell. Background, pockets and growth are as there: border median colour, edge-connected background within 30, flat pockets within 14 of the grey with a median local spread under 1.2, grown over smooth near-grey pixels. The parts of the subject of at least 200 px whose centre lies in this frame's cell of the 2 x 3 grid are the frame; the near-grey region at the centre of the frame's bounding box is its opening; the mask is eroded by 1 px, feathered (Gaussian sigma 0.8) and unmixed from the grey; the master is the frame cropped with 4 px around it, at the resolution of the sheet. The static is the master on a square centred on its bounding box with 2% margin on each side (the set 1 framing), Lanczos-resized on premultiplied alpha to 256 x 256.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | As above. Canvas 256 x 256, the canvas of the eight animated set 1 frames (#703), so a rank frame and a shop frame swap at one size; the four static set 1 frames are 512 x 512. The class canvas is open (O-6). The frame is about 370 to 490 px on the sheet, so the static is a reduction, never an enlargement |
| 2. Paint-over, AI artefacts | No | No paint-over. An artist checks the edge and the ornament |
| 3. Moving parts on their own layers | Not needed | Static item: the lower tiers of the family carry no animation, by the design of this PR |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen in the concept, the master or the export. The static shows the metal and gems at rest with no aura, sheen or ambient detail (`docs/art-direction.md` section 7) |
| 5. Silhouette and smallest-size tests | In part | The agent drew the static around `avatar.court_poet` set inside the opening (the avatar scaled to 1.04 times the opening diameter, the frame on top) at 256, 128 and 64 px on `color.background.base`. No avatar size is set in `docs/ui-motion.md`, `docs/art-direction.md` (5.1 and section 10 leave the frame band and the smallest display size to #168) or the client (`AvatarView` fits the avatar and frame to its box, `docs/cosmetics.md` 8.3), so these three sizes are the task's default, not a spec, and no size variant is exported: the one 256 x 256 export is scaled down by the view. At 64 px it reads as a round ring with a twisted edge and a small bump of studs on top; it is the simplest outline of the family. The smallest display size is open (#168, O-6), so the test is not final |

Size: 1 sprite, 29,584 bytes (`npm run asset:check`, ASTC 6x6 estimate); frames have no limit yet (O-5).

Not done: W-6, and the upload and publish of W-7 and W-8. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261116-01.png | `2c69e85351ed26d62d1120546b9a9c74925008c1a113c39cfa3ee7efce80f63c` | 1254 x 1254 RGB, the generated sheet of six frames, unchanged; this frame is cell 2 | Codex CLI `$imagegen`; byte-identical to the file in `~/.codex/generated_images/01a1e36a-d8a9-75c2-8158-564bae5affc1/` (checked with sha256) |
| work/season1_bronze_cleanup.py | `33a5856c09103b37d6a00660855f68005cc95e819a0ef08e246cd186eb7a0e31` | W-3 and W-5 script | Written by Claude Code, 2026-11-16; the same script for every static frame of this PR |
| work/season1_bronze_master.png | `396e445dd7276fe25838f15107cb7918b949ebe3c1cf2c31a998789fec8c9c69` | The cut-out frame at the resolution of the sheet, RGBA | `season1_bronze_cleanup.py` |
| export/static.png | `6a4679452b7a156c76737aeb82f78ba244f4e54891d1605156c141778fb8c6f8` | 256 x 256 RGBA static | `season1_bronze_cleanup.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-16 | concept/20261116-01.png | Style key and sheet prompt, quoted below. Sent 08:13:12 UTC by Claude Code; `c2pa.created` 2026-11-16T08:14:39Z |
| 2026-11-16 | work/season1_bronze_master.png, export/static.png | `season1_bronze_cleanup.py` on cell 2 of the concept |

Prompt of `concept/20261116-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Sheet of six separate ornamental round frames for circular portraits, arranged in an even grid of three columns and two rows, front view, each frame symmetric left to right, fully inside its own cell with wide empty background between neighbours, nothing touching or overlapping. Each frame is a round ring with its open centre left empty, thin enough not to cover a face, and each frame has its own outer silhouette. All six share one family look: the same warm light, the same painted finish, readable at thumbnail size, getting richer from the first to the last. No crowns, no stars, no crosses, no crescents, no fleur-de-lis, no heraldic lilies, no wings, no feathers. Row 1, left to right: first, a heavy plain ring of dark forged iron with a wide flat band, a row of round rivets around it and four small square iron clamps at the top, bottom, left and right, palette dark iron grey, gunmetal and warm steel highlights; second, a ring of dark aged bronze with a twisted rope border along both edges and three small rounded bronze studs grouped at the top, palette deep bronze brown, soft olive patina and warm highlights; third, a polished silver ring whose outer edge is an octagon with a small round silver knob on each of the eight corners and a fine beaded border, palette polished silver, pale grey and cool white highlights. Row 2, left to right: fourth, a polished gold ring whose outer edge is a scalloped rosette of twelve rounded petals, with one small ruby in a little gold mount at the top, palette polished gold, warm yellow and ruby red; fifth, a pale platinum ring with fine engraved scrollwork and four tapering curled scroll flourishes reaching outward at the four diagonals, each set with a small pearl, palette pale platinum, ivory and soft pearl; sixth, a polished gold ring set all around with small square-cut emeralds in gold bezels, an arched gold crest at the top holding a large cushion-cut emerald between small gold leaf scrolls, and a smaller emerald drop at the bottom, palette rich gold, deep emerald green and ivory.
```

Codex was given the prompt inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261116-01.png. Do not use any API key and do not edit the image.`

RC-2 against this prompt and the Inputs line: it names no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-16.
