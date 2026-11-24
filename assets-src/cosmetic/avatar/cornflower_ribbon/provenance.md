# avatar.cornflower_ribbon

Animated avatar of set 1 (#171), batch B, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 and 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, final runs on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: `concept/20261115-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261115-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a woman in her forties with light tan skin, long wavy dark brown hair in a high ponytail tied with a big cornflower-blue satin bow whose two long ribbon ends hang down beside her face, small pearl stud earrings, and a cream silk blouse under a cornflower-blue velvet bodice with fine gold trim"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt: one ribbon end hangs over the hair on her right (the viewer's left), the other beside her neck on the left. Not asked for: a sapphire and pearl brooch on the ruffled collar.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.10, -0.22 px, scale 1.0002 and rotation 0.0001 rad (mean squared grey difference outside the eyes 0.00101 before, 0.00098 after), and the eye patch was offset by [-1.6, 0.7, 1.0] in RGB to meet the canonical skin. The changes outside the eye mask are small and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, source `shop:offer.avatar_cornflower_ribbon`. The offer id is a placeholder until the shop record exists (V-15). One ambient motion, the two ribbon ends. Animated avatars stay at `epic` and above as in batch A; batch B proposes 5 epic, 4 legendary, 1 mythic, the same split as batch A.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 1 animated detail (ribbon ends (pair)), so `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signatures were not validated with a C2PA tool.

| Field | `concept/20261115-01.png` | `concept/20261115-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T17:29:28Z | `c2pa.created`, 2026-11-15T17:45:09Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 17:28:19 | 2026-11-15 17:44:06 |
| Codex session | `01a1e040-b6a0-70a1-831b-70e5c8fdbcd7` | `01a1e04f-2a9e-7b90-9495-47a73c7eefb3` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/cornflower_ribbon/work/cornflower_ribbon_cleanup.py cornflower_ribbon assets-src/cosmetic/avatar/cornflower_ribbon/concept/20261115-01.png assets-src/cosmetic/avatar/cornflower_ribbon/work
```

It printed background [201.0, 201.0, 202.0], pockets 8592 px, crop (37, 0, 1068, 1031), portrait 768. The script is `ruby_drop_cleanup.py` of batch A unchanged. It writes the master and a square 768 x 768 portrait crop (`work/cornflower_ribbon_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | As in batch A. The loop script puts a circular alpha (soft edge of 2.25 px at 768, 0.56 px at 192) on every frame, because the portrait is shown in a circle (`docs/art-direction.md` 5.1); the class canvas is open (O-6) and D-1 sets 192 x 192 |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Yes | Written by the loop script into `work/` (table below); the ripples are warps of the base layer and need no layer of their own |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen in the concept, the master or the frames |
| 5. Silhouette and smallest-size tests | In part | The agent looked at every frame at 192 px and enlarged on a dark backdrop; face, expression and the moving part read. The smallest display size and frame band of an avatar are open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-16 (UTC) by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/avatar/cornflower_ribbon/work/cornflower_ribbon_loop.py assets-src/cosmetic/avatar/cornflower_ribbon/concept assets-src/cosmetic/avatar/cornflower_ribbon/work assets-src/cosmetic/avatar/cornflower_ribbon/export
```

It printed `cornflower_ribbon_loop: blink edit registered (tx, ty, scale, angle) = (0.1004, -0.2192, 1.0002, 0.0001), residual 0.00101 -> 0.00098, eye colour offset [-1.6, 0.7, 1.0], parts 0, ripples 2, filled 0 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run into empty folders wrote files byte-identical to each other and to the committed ones.

The script is the batch A loop script (`ruby_drop_loop.py`) with this item's geometry and these additions, the same for every avatar of this batch: the concept file names are constants; a part can open and close (`flap`, a horizontal scale about its pivot); `RIPPLES` add a smooth sideways wave to the warp inside a feathered polygon, so cloth, ribbon and chain move without a cut-out layer and without seams; a hole behind a part is cleared where the fill estimate is less than half covered, which keeps a filled hole from leaving a smear over the background; a glint is clipped to the bright pixels of the portrait and its peak opacity is a constant (`GLINT_PEAK`, 0.55 as in batch A unless the item says otherwise); and the registration region is clipped to the image when the crop reaches past its edge.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient ripple | Each of the two hanging ribbon ends, from below the bow: inside a polygon feathered by Gaussian sigma 10 (768 scale), the pixels shift sideways by 6 x (sin(2 pi y / 280 - 1 t) - sin(2 pi y / 280)) px at 768, faded in over 220 px below y 220, so the wave runs down once per loop and the attached top stays still | Warp only, so no seams and no hole to fill; the polygon ends before the circle edge |
| Ambient ripple | Each of the two hanging ribbon ends, from below the bow: inside a polygon feathered by Gaussian sigma 10 (768 scale), the pixels shift sideways by 6 x (sin(2 pi y / 280 - 1 t) - sin(2 pi y / 280)) px at 768, faded in over 180 px below y 340, so the wave runs down once per loop and the attached top stays still | Warp only, so no seams and no hole to fill; the polygon ends before the circle edge |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 3.04 of 255, against 2.83 on average and 3.37 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `cornflower_ribbon: background [201.0, 201.0, 202.0], pockets 8592 px, crop (0, -177, 1121, 944), portrait 768`
- Crop `(37, 0, 1068, 1031)` to `(0, -177, 1121, 944)` on the concept
- Scale k = 0.9197. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- Printed by the loop: `cornflower_ribbon_loop: blink edit registered (tx, ty, scale, angle) = (0.0991, -0.1840, 1.0001, 0.0000), residual 0.00079 -> 0.00077, eye colour offset [-1.5, 0.9, 1.2], parts 0, ripples 2, filled 0 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.016 before, 0.172 after; highest over the 16 loop frames 0.010 before, 0.167 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/cornflower_ribbon/work/cornflower_ribbon_cleanup.py cornflower_ribbon assets-src/cosmetic/avatar/cornflower_ribbon/concept/20261115-01.png assets-src/cosmetic/avatar/cornflower_ribbon/work
python -I assets-src/cosmetic/avatar/cornflower_ribbon/work/cornflower_ribbon_loop.py assets-src/cosmetic/avatar/cornflower_ribbon/concept assets-src/cosmetic/avatar/cornflower_ribbon/work assets-src/cosmetic/avatar/cornflower_ribbon/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 24 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/cornflower_ribbon` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `86641294540577fdb002541259187e947b80e698e9ca78832c0a756d39517db6` |
| export/anim/0002.png | `18870b775f2e8ad83a17b146daaddd59a8ff26b03d6ff4dce364157c57dcac5c` |
| export/anim/0003.png | `3698663c21d5a93ffd6e37bcc55a46e745d62efed0a5687d1c2bc682e594584b` |
| export/anim/0004.png | `2b5968e450a5f592e0ff4bbe67c9337dacf56f47e8262a6bf73e86bc3bd71f2e` |
| export/anim/0005.png | `14b20afe63fe79ff6e2e3b3f446c33f8164e86b6c0c57d2875bc1ee8fdc2f324` |
| export/anim/0006.png | `192140e05f80792f9a90a751e510c448e30fb17a44f188c9ef63920308921985` |
| export/anim/0007.png | `9f07851a721cce8a75b31e9c54b8c0d1118978fa83931e5b7d0c206fe851c045` |
| export/anim/0008.png | `65cfe7a5bcd42153d0d7761d371cfe9ad4f11c6012fef0d93404bcdda0c84dc3` |
| export/anim/0009.png | `46a544b60b0b979420d87dd1a847c97083d0275ecffe42c7bb02b97c97f0daf4` |
| export/anim/0010.png | `146bd34c5b0cbd295d262618a518ed077c63f96cc668759c77ee18a19e62a5cf` |
| export/anim/0011.png | `ee054e9b4c58e26b750fffdb8f84941d0bc9c1d8fa60d7eb1c473935eb7a814e` |
| export/anim/0012.png | `a105c6507f00d096181700673e3799e61be3722889072a6b2496cf09b85d95bf` |
| export/anim/0013.png | `897d1280d38525c62f66615902cc635aefb2126bf2b07685403cc8a2c8a283bc` |
| export/anim/0014.png | `c2283f28362f9c32d9bb8028e1b7831b449a9a955dfdf6bae42b994ab6d1e49c` |
| export/anim/0015.png | `7ff710ad2317dc15950e6ab266cb58cef639f162ecec75ccf535efd42347a454` |
| export/anim/0016.png | `7641042674e3261b22e4b5bc6a40059ab8dc27e0dea4d7cf3c71b6e0b0f28642` |
| export/static.png | `46a544b60b0b979420d87dd1a847c97083d0275ecffe42c7bb02b97c97f0daf4` |
| work/cornflower_ribbon_base.png | `77e087da2b262e07697f36b09c2c2b77420ecccaf44b854701e5d313ed46078f` |
| work/cornflower_ribbon_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` |
| work/cornflower_ribbon_eyes_closed.png | `2f6fae8fecd5835a58b91156664d571519d5f0f47409235ec4ae4511be67bad3` |
| work/cornflower_ribbon_eyes_open.png | `8651dd81e0e4079165f15622105d58b8ae2d1d5502b561030f441d3d5ede6c81` |
| work/cornflower_ribbon_head.png | `e32fbac43b4ed1c3bf3690d9164e2de40ac789f7e0b66abed71c2def319acf3f` |
| work/cornflower_ribbon_loop.py | `0b8a38c9ef8e38536b61d89be350af0698a187ffccdccd51d90b741dc91a7b33` |
| work/cornflower_ribbon_portrait.png | `9849f8991b8e74d8f2d56c846d7beb96a7d54310bfc7a65fca867af93b81cc20` |
| work/cornflower_ribbon_torso.png | `d5817aae60a2cf4a43efe4ebdba6f6615fc8a0706702eb503fddadfe0d534f1c` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `2a3c8ac9c03e3ab86d175fdd27cc7e4720f057590c93f6db20b35d899b1aa904` | 1199 x 1312 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e040-b6a0-70a1-831b-70e5c8fdbcd7` |
| concept/20261115-02.png | `769ea8ea7cd3f29d5ec882fb46f32098a29753728e1f306a5dc2a2aaab20f45e` | 1199 x 1312 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e04f-2a9e-7b90-9495-47a73c7eefb3` |
| work/cornflower_ribbon_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_cleanup.py` |
| work/cornflower_ribbon_loop.py | `650da5edf7b9622c667f9f7ad4d22a56df6bfd7f502720564e48934700cd2237` | W-4 and W-5 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_loop.py`; the same script for every avatar of this batch with the item's geometry |
| work/cornflower_ribbon_master.png | `66ef93384bcf156f7ec098e2b5d32ebdf35b7f49233ef03a6e1325bfd3cf01b2` | 1199 x 1312 RGBA, the cut-out master | `cornflower_ribbon_cleanup.py` |
| work/cornflower_ribbon_base.png | `7cdf08fe48ee260a5e893f501d460468c0cc2f444d4e0dcc762467ec587b8b7e` | The portrait with the moving parts taken out (filled or cleared) | `cornflower_ribbon_loop.py` |
| work/cornflower_ribbon_eyes_closed.png | `5fd1437b4d1ad0516e4821a49d1f363c7823d59dbc17c66e453483a7f98c6269` | Eye region, closed (aligned edit) through the same mask | `cornflower_ribbon_loop.py` |
| work/cornflower_ribbon_eyes_open.png | `0ac46307a419da897f98169375558f325f7ac4f028bbc53ac7bf74076168b97f` | Eye region, open (canonical concept) through the feathered eye mask | `cornflower_ribbon_loop.py` |
| work/cornflower_ribbon_head.png | `bc694e9702e830de3dee5a93e0ac1a0b804b0c12ca7383452438d56c115c972b` | Head and hair: the portrait weighted by the head field of the warp | `cornflower_ribbon_loop.py` |
| work/cornflower_ribbon_portrait.png | `6402fdc61ff873cfc640cd05ff24f2af22b02aa50f67c96ea2f079f5496f5ee2` | 768 x 768 RGBA portrait crop, no circle | `cornflower_ribbon_cleanup.py` |
| work/cornflower_ribbon_torso.png | `d17e5c711f0d71d976f84d9c02d01fc52ffd93b35d627682a91d0b73c28b6b99` | Torso and costume: the portrait weighted by one minus the head field | `cornflower_ribbon_loop.py` |
| export/static.png | `1c50f2db044e9b7807f96fe69ffd642d970f9bf49f03864fc90a7251d88fef11` | 192 x 192 RGBA static, loop frame 9 | `cornflower_ribbon_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `cornflower_ribbon_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted below. Sent 17:28:19 UTC; `c2pa.created` 2026-11-15T17:29:28Z |
| 2026-11-15 | concept/20261115-02.png | Edit of `concept/20261115-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 17:44:06 UTC; `c2pa.created` 2026-11-15T17:45:09Z |
| 2026-11-16 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/cornflower_ribbon_base.png, work/cornflower_ribbon_cleanup.py, work/cornflower_ribbon_eyes_closed.png, work/cornflower_ribbon_eyes_open.png, work/cornflower_ribbon_head.png, work/cornflower_ribbon_loop.py, work/cornflower_ribbon_portrait.png, work/cornflower_ribbon_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(37, 0, 1068, 1031)` to `(0, -177, 1121, 944)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her forties with light tan skin, long wavy dark brown hair in a high ponytail tied with a big cornflower-blue satin bow whose two long ribbon ends hang down beside her face, small pearl stud earrings, and a cream silk blouse under a cornflower-blue velvet bodice with fine gold trim, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261115-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her forties with light tan skin, long wavy dark brown hair in a high ponytail tied with a big cornflower-blue satin bow whose two long ribbon ends hang down beside her face, small pearl stud earrings, and a cream silk blouse under a cornflower-blue velvet bodice with fine gold trim, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261115-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261115-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.
