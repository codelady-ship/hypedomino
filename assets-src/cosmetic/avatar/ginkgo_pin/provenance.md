# avatar.ginkgo_pin

Animated avatar of set 1 (#171), batch A, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: `concept/20261115-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261115-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a woman in her sixties with fair skin, silver hair in a neat chignon held by a gold hairpin with small gold ginkgo leaves hanging from it on fine chains, and a black silk jacket with gold ginkgo leaf embroidery"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt. Not asked for but present: ginkgo leaf earrings with pearls and crystal drops, a mandarin collar, and a ginkgo and pearl clasp with a yellow gemstone pendant.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.11, -0.13 px, scale 1.0001 and rotation -0.0001 rad (mean squared grey difference outside the eyes 0.00116 before, 0.00114 after), and the eye patch was offset by [-0.3, 2.2, 2.6] in RGB to meet the canonical skin. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, source `shop:offer.avatar_ginkgo_pin`. The offer id is a placeholder until the shop record exists (V-15). Animated avatars are proposed at `epic` and above, as the batch leans to the higher tiers of `docs/rarity.md`; this batch proposes 5 epic, 4 legendary, 1 mythic.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 1 animated detail (ginkgo dangles), so `epic`, was `mythic`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signatures were not validated with a C2PA tool.

| Field | `concept/20261115-01.png` | `concept/20261115-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T16:50:59Z | `c2pa.created`, 2026-11-15T16:52:34Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 16:49:44 | 2026-11-15 16:51:24 |
| Codex session | `01a1e01d-626f-7a02-a825-91c035ab4661` | `01a1e01e-e834-7062-8e8e-488ed9f04db2` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/ginkgo_pin/work/ginkgo_pin_cleanup.py ginkgo_pin assets-src/cosmetic/avatar/ginkgo_pin/concept/20261115-01.png assets-src/cosmetic/avatar/ginkgo_pin/work
```

It printed background [198.0, 198.0, 198.0], pockets 11799 px, crop (36, 0, 1114, 1078), portrait 768. The script is `amethyst_courtier_cleanup.py` of batch A of the static avatars with three changes, all for the same reason: on these concepts grey background stayed opaque between loose strands and the neck, and hair edges kept a light outline. (1) It writes the master and a square 768 x 768 portrait crop (`work/ginkgo_pin_portrait.png`, the same crop rule, no circle) instead of the 512 static, because the static of an animated avatar is a loop key frame. (2) Enclosed background pockets are taken within 24 of the grey (10 before) and up to 140 px inside the outline (80 before), and a pocket counts only if its median colour is within 12 of the grey, so pearls, crystal and silver that come close to the grey stay. (3) In a 6 px band along the background the alpha is capped by projecting each pixel between the grey and the mean colour of the nearby solid subject (Gaussian sigma 6); where that colour is itself within 40 of the grey, by the colour distance to the grey (0 at 14, 1 at 60). The colour of edge pixels is unmixed from the grey and blended toward the nearby subject colour by one minus the alpha, so faint edge pixels take the subject's colour instead of a light speck.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | As in the static batch A with the changes above. The loop script puts a circular alpha (soft edge of 2.25 px at 768, 0.56 px at 192) on every frame, because the portrait is shown in a circle (`docs/art-direction.md` 5.1); the class canvas is open (O-6) and D-1 sets 192 x 192 |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Yes | Written by the loop script into `work/` (table below) |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen in the concept, the master or the frames |
| 5. Silhouette and smallest-size tests | In part | The agent looked at every frame at 192 px and at 2x on a dark backdrop and at frame 1 on a light one; face, expression and the moving part read. The smallest display size and frame band of an avatar are open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-15 by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/avatar/ginkgo_pin/work/ginkgo_pin_loop.py assets-src/cosmetic/avatar/ginkgo_pin/concept assets-src/cosmetic/avatar/ginkgo_pin/work assets-src/cosmetic/avatar/ginkgo_pin/export
```

It printed `ginkgo_pin_loop: blink edit registered (tx, ty, scale, angle) = (0.1131, -0.1306, 1.0001, -0.0001), residual 0.00116 -> 0.00114, eye colour offset [-0.3, 2.2, 2.6], parts 1, filled 23328 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run into empty folders wrote files byte-identical to each other and to the committed ones.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes. Half-closed frames made by cross-fading showed two sets of lids and were dropped | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.5 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `ginkgo_dangles` (layer `work/ginkgo_pin_ginkgo_dangles.png`, pivot (150, 142)) turns about its pivot by up to 3.5 degrees with sin(lag) - sin(2 pi (f - 8) / 16 + lag), lag 0.6, the turn growing from 0 at the pivot to full at 240 px (768 scale), so the top stays attached and the end swings; the part also follows the head or torso warp at its pivot | The part hangs over the background, so the hole it leaves in the base layer is transparent |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 4.43 of 255, against 3.75 on average and 4.6 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `ginkgo_pin: background [198.0, 198.0, 198.0], pockets 11799 px, crop (0, -189, 1172, 983), portrait 768`
- Crop `(36, 0, 1114, 1078)` to `(0, -189, 1172, 983)` on the concept
- Scale k = 0.9198. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- `register` now clips a crop edge above the concept to 0 when it marks the registration region, as the later loop scripts of this batch do; with the old slice a negative top would have marked the wrong rows
- Printed by the loop: `ginkgo_pin_loop: blink edit registered (tx, ty, scale, angle) = (0.1070, -0.1405, 1.0000, -0.0001), residual 0.00094 -> 0.00091, eye colour offset [-0.2, 2.2, 2.6], parts 1, filled 19952 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.010 before, 0.172 after; highest over the 16 loop frames 0.005 before, 0.161 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/ginkgo_pin/work/ginkgo_pin_cleanup.py ginkgo_pin assets-src/cosmetic/avatar/ginkgo_pin/concept/20261115-01.png assets-src/cosmetic/avatar/ginkgo_pin/work
python -I assets-src/cosmetic/avatar/ginkgo_pin/work/ginkgo_pin_loop.py assets-src/cosmetic/avatar/ginkgo_pin/concept assets-src/cosmetic/avatar/ginkgo_pin/work assets-src/cosmetic/avatar/ginkgo_pin/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 25 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/ginkgo_pin` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `61640f7061dd53e93ec69d3c8baefaee74e870fc7ed7b0b6f7f64ce0c6f5315b` |
| export/anim/0002.png | `fd976550e211f4aada61f129bd5d7d38d696e908d025e52e4f40e43159497aa6` |
| export/anim/0003.png | `1999e99874fcf04d615a0a762106c8f9b525596e0be03254f1f080222995616e` |
| export/anim/0004.png | `e1032fc1e0de7cc387cad29149d063f16a3f621e08d8fc303e6e47d35023d97f` |
| export/anim/0005.png | `8237030352a38f4c374c68c209155bb88be4802c87197562e9c031e91503f74a` |
| export/anim/0006.png | `50f64ec24b9570305167e17157deaed198fe49d789f04b9b6cf943f8f01e37ce` |
| export/anim/0007.png | `8a72776faacf652b0d957844f4dac8f0078d92ca8cecc4646019b24b422f64a8` |
| export/anim/0008.png | `4cbc7fa3987a0cbd6e4121aabdd7ce14a85be1c660c60b1225608c8284b93308` |
| export/anim/0009.png | `3960e87e5db681cfac67a27154d5192910dedf1e636bb6ef475483f2be8c5ebc` |
| export/anim/0010.png | `d97cd6c650f0f657f132d2b3aa0e4f0c827969842bd55ca8ba6afff61d80b636` |
| export/anim/0011.png | `6dbe12fa7ef94dd2fbc47df486190bfd452b1aa65393df234af38abd59daffd8` |
| export/anim/0012.png | `e6190057f2efed9bf7d249a41653176d27b010f88eb2e890bdef98e98c35264f` |
| export/anim/0013.png | `38b2aea67bccb77b1493efa6bb2e837051bdd4d5193040953a579c9931138263` |
| export/anim/0014.png | `2cd95ba5be5b62551002a9c668add1c64a0c4194290da24879850740e5ebed9d` |
| export/anim/0015.png | `a375aec77da971a6f5ba5bc3b6af34ca1ebc5de548b4a35226bfb788f4a7c0a6` |
| export/anim/0016.png | `e4a69fc699f07921384ccdacdc4ea2f9a83a331624081859e00737a08cd4033a` |
| export/static.png | `3960e87e5db681cfac67a27154d5192910dedf1e636bb6ef475483f2be8c5ebc` |
| work/ginkgo_pin_base.png | `a5164dbfa9550cf9b75b87d0808b5219e202772fefd8f532365c769ee370eb7b` |
| work/ginkgo_pin_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` |
| work/ginkgo_pin_eyes_closed.png | `69a24174f76053957e25d4722da5a007a799983017c12217f6610b2b4c6e7f43` |
| work/ginkgo_pin_eyes_open.png | `bf7ca67f96521fc32c4f1217551e1b805e37574107d9e69be8684a84d4050b85` |
| work/ginkgo_pin_ginkgo_dangles.png | `6e6a5472a63885bd0b5da7fcf9d11cfab734920f4cc085af90b47f9ccff3b4af` |
| work/ginkgo_pin_head.png | `3c29041f56bb23bd59a69ef808cf8ad579e50a970b16c221f89ddcb2577e16c7` |
| work/ginkgo_pin_loop.py | `0036647b1288cfadbf46309ceee53000c7cd484599ef1d6a203e0e633eb9532e` |
| work/ginkgo_pin_portrait.png | `56ebdcc377ba5082c30e0f9484ddc06f955d6f2bebd0f8f2dd5ba4da8607eda7` |
| work/ginkgo_pin_torso.png | `7f357e07bca2a3fde093829b3b8423baf80630410cc964e10d1a33a6696bbb3d` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `7adb4dfe386653ec67f0befd108fb661409f751c398f8d6a5991079e2d3023ca` | 1254 x 1254 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e01d-626f-7a02-a825-91c035ab4661` |
| concept/20261115-02.png | `9c294cc675c3c82a7e2c988dc4eb95c5e01f2009ca0c8997a4109744024dc1b6` | 1254 x 1254 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e01e-e834-7062-8e8e-488ed9f04db2` |
| work/ginkgo_pin_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-15, from `amethyst_courtier_cleanup.py` |
| work/ginkgo_pin_loop.py | `828dbb0498e83ef047e7e765f9a6b81fd94d9d0fffacaf2b1b75766510b354a8` | W-4 and W-5 script | Written by Claude Code, 2026-11-15; the same script for every avatar of this batch with the item's geometry |
| work/ginkgo_pin_master.png | `d2293202423ac6d0eb4b03ef80f39a3c7753505860677eda95786c69fd25cd0c` | 1254 x 1254 RGBA, the cut-out master | `ginkgo_pin_cleanup.py` |
| work/ginkgo_pin_base.png | `4a323dab2614889c3c79fcf8ab9aa07787323cb3743cf3e873942cc79ae89e26` | The portrait with the moving parts taken out (filled or cleared) | `ginkgo_pin_loop.py` |
| work/ginkgo_pin_eyes_closed.png | `55b1bfe3ba98036429b3cc892b4e3b035f723039fad0eb7918c1796507a84484` | Eye region, closed (aligned edit) through the same mask | `ginkgo_pin_loop.py` |
| work/ginkgo_pin_eyes_open.png | `b5a5bb039231c9dfae2dfbc1a2f1438a6123199fa14a6afe10da5ec8946625b7` | Eye region, open (canonical concept) through the feathered eye mask | `ginkgo_pin_loop.py` |
| work/ginkgo_pin_ginkgo_dangles.png | `e4de6290ecfbeb30365da09afb2b91ed4653816a07f254ff2bd0cf4917cdcea3` | Moving part `ginkgo_dangles`: the portrait through its feathered polygon | `ginkgo_pin_loop.py` |
| work/ginkgo_pin_head.png | `c0de650983ee2c3d2125245cbce46f8358502a10ce1b112d7cd7bcc6caf3ec4f` | Head and hair: the portrait weighted by the head field of the warp | `ginkgo_pin_loop.py` |
| work/ginkgo_pin_portrait.png | `f717b45e950e075d695b273456331a3675ebb5818eeb127df9e4f03101a33559` | 768 x 768 RGBA portrait crop, no circle | `ginkgo_pin_cleanup.py` |
| work/ginkgo_pin_torso.png | `fcca9fe67c77838b6226d40ec44148cf55b603947d67cc10f6b7f2a53abc1ebc` | Torso and costume: the portrait weighted by one minus the head field | `ginkgo_pin_loop.py` |
| export/static.png | `7f13a99fc6282f6327d578a6b6d31f64ff68de1c76c5bb24e735b3e440a2fa0b` | 192 x 192 RGBA static, loop frame 9 | `ginkgo_pin_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `ginkgo_pin_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted below. Sent 16:49:44 UTC; `c2pa.created` 2026-11-15T16:50:59Z |
| 2026-11-15 | concept/20261115-02.png | Edit of `concept/20261115-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 16:51:24 UTC; `c2pa.created` 2026-11-15T16:52:34Z |
| 2026-11-15 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/ginkgo_pin_base.png, work/ginkgo_pin_cleanup.py, work/ginkgo_pin_eyes_closed.png, work/ginkgo_pin_eyes_open.png, work/ginkgo_pin_ginkgo_dangles.png, work/ginkgo_pin_head.png, work/ginkgo_pin_loop.py, work/ginkgo_pin_portrait.png, work/ginkgo_pin_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(36, 0, 1114, 1078)` to `(0, -189, 1172, 983)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her sixties with fair skin, silver hair in a neat chignon held by a gold hairpin with small gold ginkgo leaves hanging from it on fine chains, and a black silk jacket with gold ginkgo leaf embroidery, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261115-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her sixties with fair skin, silver hair in a neat chignon held by a gold hairpin with small gold ginkgo leaves hanging from it on fine chains, and a black silk jacket with gold ginkgo leaf embroidery, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261115-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261115-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-15.
