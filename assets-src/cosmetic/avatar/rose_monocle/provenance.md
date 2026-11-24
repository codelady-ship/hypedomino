# avatar.rose_monocle

Animated avatar of set 1 (#171), batch B, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 and 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, final runs on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: `concept/20261115-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261115-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a woman in her early eighties with fair skin and rosy cheeks, white hair in a soft high updo, a round gold-rimmed monocle over one eye with a fine gold chain looping down to her collar, and a dusty-rose silk gown with a black lace high collar"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt: the monocle sits over her right eye (the viewer's left) and its chain runs down to a ruby brooch on the lace collar. Not asked for: a small gold and ruby hair ornament on the updo, ruby and pearl drop earrings, and pearl buttons and ruffles on the gown.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.27, 0.23 px, scale 0.9999 and rotation -0.0002 rad (mean squared grey difference outside the eyes 0.00124 before, 0.00119 after), and the eye patch was offset by [-11.5, -2.7, 0.8] in RGB to meet the canonical skin. The eye behind the monocle closes with the other; the changes outside the eye mask are small and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly; the hair ornament is a small jewelled band, not a crown
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `mythic`, source `shop:offer.avatar_rose_monocle`. The offer id is a placeholder until the shop record exists (V-15). A glint crosses the monocle lens, its chain sways and both earrings swing. Animated avatars stay at `epic` and above as in batch A; batch B proposes 5 epic, 4 legendary, 1 mythic, the same split as batch A.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 3 animated details (lens glint, chain ripple, earrings (pair)), so `mythic`, was `legendary`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signatures were not validated with a C2PA tool.

| Field | `concept/20261115-01.png` | `concept/20261115-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T17:33:43Z | `c2pa.created`, 2026-11-15T17:49:19Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 17:32:35 | 2026-11-15 17:48:37 |
| Codex session | `01a1e044-9fa2-7cd0-883c-b23a442d9b51` | `01a1e053-4c52-7cc0-a76d-fc321fba17af` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/rose_monocle/work/rose_monocle_cleanup.py rose_monocle assets-src/cosmetic/avatar/rose_monocle/concept/20261115-01.png assets-src/cosmetic/avatar/rose_monocle/work
```

It printed background [203.0, 204.0, 206.0], pockets 1494 px, crop (110, 0, 1141, 1031), portrait 768. The script is `ruby_drop_cleanup.py` of batch A unchanged. It writes the master and a square 768 x 768 portrait crop (`work/rose_monocle_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

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
python -I assets-src/cosmetic/avatar/rose_monocle/work/rose_monocle_loop.py assets-src/cosmetic/avatar/rose_monocle/concept assets-src/cosmetic/avatar/rose_monocle/work assets-src/cosmetic/avatar/rose_monocle/export
```

It printed `rose_monocle_loop: blink edit registered (tx, ty, scale, angle) = (0.2730, 0.2311, 0.9999, -0.0002), residual 0.00124 -> 0.00119, eye colour offset [-11.5, -2.7, 0.8], parts 2, ripples 1, filled 3853 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run into empty folders wrote files byte-identical to each other and to the committed ones.

The script is the batch A loop script (`ruby_drop_loop.py`) with this item's geometry and these additions, the same for every avatar of this batch: the concept file names are constants; a part can open and close (`flap`, a horizontal scale about its pivot); `RIPPLES` add a smooth sideways wave to the warp inside a feathered polygon, so cloth, ribbon and chain move without a cut-out layer and without seams; a hole behind a part is cleared where the fill estimate is less than half covered, which keeps a filled hole from leaving a smear over the background; a glint is clipped to the bright pixels of the portrait and its peak opacity is a constant (`GLINT_PEAK`, 0.55 as in batch A unless the item says otherwise); and the registration region is clipped to the image when the crop reaches past its edge.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `earring_left` (layer `work/rose_monocle_earring_left.png`, pivot (292, 504)) turns about its pivot by 5.0 degrees x (sin(lag) - sin(t + lag)), lag 0.4, the turn growing from 0 at the pivot to full at 60 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | A small part well inside the circle, so its motion never meets the circle edge |
| Ambient part | `earring_right` (layer `work/rose_monocle_earring_right.png`, pivot (566, 500)) turns about its pivot by 5.0 degrees x (sin(lag) - sin(t + lag)), lag 0.9, the turn growing from 0 at the pivot to full at 50 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | A small part well inside the circle, so its motion never meets the circle edge |
| Ambient ripple | The monocle chain, between the monocle and the brooch, so both ends stay fixed: inside a polygon feathered by Gaussian sigma 8 (768 scale), the pixels shift sideways by 4 x (sin(2 pi y / 260 - 1 t) - sin(2 pi y / 260)) px at 768, faded in over 120 px below y 380, so the wave runs down once per loop and the attached top stays still | Warp only, so no seams and no hole to fill; the polygon ends before the circle edge |
| Ambient glint | A glint on the monocle lens: (372, 334), ellipse 92 x 92 px at 768, frames 11, 12, 13, 14, 15 | A painted highlight: a soft streak that crosses the lens, clipped to its ellipse and to the bright pixels of the portrait (mean channel value faded in from 0.3 to 0.5), warm ivory at up to 0.55 opacity, normal blending, no halo outside the ellipse (not a glow, `docs/art-direction.md` section 8) |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 3.17 of 255, against 2.82 on average and 3.45 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `rose_monocle: background [203.0, 204.0, 206.0], pockets 1494 px, crop (65, -173, 1186, 948), portrait 768`
- Crop `(110, 0, 1141, 1031)` to `(65, -173, 1186, 948)` on the concept
- Scale k = 0.9197. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- Printed by the loop: `rose_monocle_loop: blink edit registered (tx, ty, scale, angle) = (0.2756, 0.1857, 0.9999, -0.0003), residual 0.00103 -> 0.00099, eye colour offset [-11.9, -3.0, 0.5], parts 2, ripples 1, filled 3390 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.021 before, 0.172 after; highest over the 16 loop frames 0.010 before, 0.167 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/rose_monocle/work/rose_monocle_cleanup.py rose_monocle assets-src/cosmetic/avatar/rose_monocle/concept/20261115-01.png assets-src/cosmetic/avatar/rose_monocle/work
python -I assets-src/cosmetic/avatar/rose_monocle/work/rose_monocle_loop.py assets-src/cosmetic/avatar/rose_monocle/concept assets-src/cosmetic/avatar/rose_monocle/work assets-src/cosmetic/avatar/rose_monocle/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 26 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/rose_monocle` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `055e00e19f299ab79b082477317c0821e412f72676eeb9f5e12729593ab885c5` |
| export/anim/0002.png | `b9c8c0baacf1b6eea104df2f236975c8efa2578fab6a57e4a979755dcd200276` |
| export/anim/0003.png | `ba92790e49eb59108fe1a8969387cd6de4dba28ed7fc09c74ae675400836bc89` |
| export/anim/0004.png | `cd57eb956cea25f06a4e6b0a47a3214aa0f806c328373ec34b97a97f8d11d5d7` |
| export/anim/0005.png | `e5fd82d5eed776feebc76686863a9e11d496d09132d3bcaf7cd6e5ad0c2560df` |
| export/anim/0006.png | `0bfa13644664ed9310133935b874a3344b81766ec8e5df08d949ebda9c817bc2` |
| export/anim/0007.png | `86f99f258740aefede6da8f9ee2f818952f121a13ee0985f6f85f3d2f10a919a` |
| export/anim/0008.png | `f7e9526ce6f151d2f0c8bba630de115187260a87d04e27148a5a33598f68def0` |
| export/anim/0009.png | `43f89afcf1060bc35d120a942fcac86df84c73bfcf9b8ecc015c2e62f1e442cf` |
| export/anim/0010.png | `a97e6c36c6ea83a7c04176c3ac858a19492b64f2dc04d34d3c893f1151fd2c5f` |
| export/anim/0011.png | `8fe2385abe0be1af716fbdea635ecf6de2d241b528dd33d51ce1117a8d05fe47` |
| export/anim/0012.png | `f4e63e59bdeaccf4707836d6e80ae037cd60a2ca38840a976d05e003158ab243` |
| export/anim/0013.png | `07c8dc1676174bf2dd8cf6bdac6be9330933ac32ed98cdc2886281af0edfcddf` |
| export/anim/0014.png | `8589c8f74c53e95a6201ffe9968e38731db24d4706bc82d02e99d380b8f7999f` |
| export/anim/0015.png | `fa3baa74aee8a8d5ce1d03a9d2b37383291977fe6172d7451084868fa43e293e` |
| export/anim/0016.png | `05f836d2662d7f58246634b7ff259fc015859dd71d8088fbadcab28f2ffb1934` |
| export/static.png | `43f89afcf1060bc35d120a942fcac86df84c73bfcf9b8ecc015c2e62f1e442cf` |
| work/rose_monocle_base.png | `f97526252c1a35bc877f0bf425d8f8882d88fba470f611a1490d7555efb11795` |
| work/rose_monocle_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` |
| work/rose_monocle_earring_left.png | `d244b129175ad43cd2e7161246819cf42b88853631f20acd312a25f11246f9f9` |
| work/rose_monocle_earring_right.png | `37f848e06afc57fc3ce72bbced5a85a6908dd8830adb983b7a5e4681a13fec10` |
| work/rose_monocle_eyes_closed.png | `9624ded87a51588eeb9baabe636477320997b9f46e9c06f88d29312ce0683575` |
| work/rose_monocle_eyes_open.png | `3bbd3b7052842bafea8cc4653ba2aab461b85092424f85b299e94d53010be440` |
| work/rose_monocle_head.png | `d73102346413b8606cd7da30ce14a52b95a8b9a869dfee24a0fad6af5f30f686` |
| work/rose_monocle_loop.py | `e084ea8090d7bf835fad4f8a4b8d52b0c093e665e66943f8468c3808705e2eb6` |
| work/rose_monocle_portrait.png | `d7b8423a240b4bd043af1a8c938987005026ce2a0047c53da4a5b086d518c01b` |
| work/rose_monocle_torso.png | `c73919c56ee1c4c7f1b99b80bdaa25e48a54ad1ff56ab4ec4dd112e4f65174bf` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `a4abca9bcfc38c61db562c8020776dbba824fa4d455e989269e5a13a7a8f1c49` | 1312 x 1199 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e044-9fa2-7cd0-883c-b23a442d9b51` |
| concept/20261115-02.png | `df00095827fcb1f662ba49f086381a3a04b510b582b80c071f724fc3c0fcb9b3` | 1312 x 1199 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e053-4c52-7cc0-a76d-fc321fba17af` |
| work/rose_monocle_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_cleanup.py` |
| work/rose_monocle_loop.py | `183d4fcc672246506c47bfb01d6d60808d265b8d8bf0a695f324946efcf7de46` | W-4 and W-5 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_loop.py`; the same script for every avatar of this batch with the item's geometry |
| work/rose_monocle_master.png | `91758296217c9685d820427df5f66db3b0ebc83b29105a23c214e34b589a25c1` | 1312 x 1199 RGBA, the cut-out master | `rose_monocle_cleanup.py` |
| work/rose_monocle_base.png | `5fd098f41298cba54e7dad4ff3fe01f501bff5c59da7c717a6e982117b635d11` | The portrait with the moving parts taken out (filled or cleared) | `rose_monocle_loop.py` |
| work/rose_monocle_earring_left.png | `12925bbe6a82ceb5ffeedd862dd72a3971b48e669de25c6214284a0d3f787775` | Moving part `earring_left`: the portrait through its feathered polygon | `rose_monocle_loop.py` |
| work/rose_monocle_earring_right.png | `9fd86812bbdd04ec16faa59371e1359174667a6189e429ee261af7f7542d0390` | Moving part `earring_right`: the portrait through its feathered polygon | `rose_monocle_loop.py` |
| work/rose_monocle_eyes_closed.png | `f843bf1f87b39638f0acb394b4556ca19ec7dace2bcbfd75e12c6112a188239c` | Eye region, closed (aligned edit) through the same mask | `rose_monocle_loop.py` |
| work/rose_monocle_eyes_open.png | `a39294fba0fc6ffb57a542c48ba3ece8332a51bba5012949b1415214a2426148` | Eye region, open (canonical concept) through the feathered eye mask | `rose_monocle_loop.py` |
| work/rose_monocle_head.png | `44258508554b083e5bb3c121aad672e8f0570eadcd8695af49095aa2421ef197` | Head and hair: the portrait weighted by the head field of the warp | `rose_monocle_loop.py` |
| work/rose_monocle_portrait.png | `cdda50f33b3fe175f16c868b29b142f7d5e99b732a5c56daf79b9b90efb68249` | 768 x 768 RGBA portrait crop, no circle | `rose_monocle_cleanup.py` |
| work/rose_monocle_torso.png | `ccccf97cf5c29ef6a300d4f52bffabd9e21532515901e3256d4a5fd39a781704` | Torso and costume: the portrait weighted by one minus the head field | `rose_monocle_loop.py` |
| export/static.png | `056174dce83c81e6e61fff3c879bd161ca4713c9830d80d49054704ff00e574c` | 192 x 192 RGBA static, loop frame 9 | `rose_monocle_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `rose_monocle_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted below. Sent 17:32:35 UTC; `c2pa.created` 2026-11-15T17:33:43Z |
| 2026-11-15 | concept/20261115-02.png | Edit of `concept/20261115-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 17:48:37 UTC; `c2pa.created` 2026-11-15T17:49:19Z |
| 2026-11-16 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/rose_monocle_base.png, work/rose_monocle_cleanup.py, work/rose_monocle_earring_left.png, work/rose_monocle_earring_right.png, work/rose_monocle_eyes_closed.png, work/rose_monocle_eyes_open.png, work/rose_monocle_head.png, work/rose_monocle_loop.py, work/rose_monocle_portrait.png, work/rose_monocle_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(110, 0, 1141, 1031)` to `(65, -173, 1186, 948)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her early eighties with fair skin and rosy cheeks, white hair in a soft high updo, a round gold-rimmed monocle over one eye with a fine gold chain looping down to her collar, and a dusty-rose silk gown with a black lace high collar, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261115-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her early eighties with fair skin and rosy cheeks, white hair in a soft high updo, a round gold-rimmed monocle over one eye with a fine gold chain looping down to her collar, and a dusty-rose silk gown with a black lace high collar, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261115-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261115-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.
