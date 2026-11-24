# avatar.topaz_brooch

Animated avatar of set 1 (#171), batch A, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: `concept/20261115-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261115-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a man in his fifties with rich dark brown skin, a full neatly rounded black beard with a few grey hairs, a deep chocolate brown velvet frock coat with gold frogging, and a large golden topaz brooch at the collar"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt. Not asked for: grey in the hair and a brocade waistcoat with gold buttons. The gold frogging is a coat fastening, not an insignia; no rank, badge or emblem is shown.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.08, 0.28 px, scale 0.9996 and rotation -0.0003 rad (mean squared grey difference outside the eyes 0.00097 before, 0.00093 after), and the eye patch was offset by [-4.7, -2.0, -1.6] in RGB to meet the canonical skin. The edit also changed the smile slightly; outside the eye mask and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, source `shop:offer.avatar_topaz_brooch`. The offer id is a placeholder until the shop record exists (V-15). Animated avatars are proposed at `epic` and above, as the batch leans to the higher tiers of `docs/rarity.md`; this batch proposes 5 epic, 4 legendary, 1 mythic.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 1 animated detail (topaz glint), so `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signatures were not validated with a C2PA tool.

| Field | `concept/20261115-01.png` | `concept/20261115-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T16:53:38Z | `c2pa.created`, 2026-11-15T16:54:40Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 16:52:57 | 2026-11-15 16:54:04 |
| Codex session | `01a1e020-56c8-7b91-9080-7b48a0212c29` | `01a1e021-5be9-75a3-a42a-feb40536615a` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/topaz_brooch/work/topaz_brooch_cleanup.py topaz_brooch assets-src/cosmetic/avatar/topaz_brooch/concept/20261115-01.png assets-src/cosmetic/avatar/topaz_brooch/work
```

It printed background [186.0, 186.0, 185.0], pockets 258 px, crop (92, 26, 1170, 1104), portrait 768. The script is `amethyst_courtier_cleanup.py` of batch A of the static avatars with three changes, all for the same reason: on these concepts grey background stayed opaque between loose strands and the neck, and hair edges kept a light outline. (1) It writes the master and a square 768 x 768 portrait crop (`work/topaz_brooch_portrait.png`, the same crop rule, no circle) instead of the 512 static, because the static of an animated avatar is a loop key frame. (2) Enclosed background pockets are taken within 24 of the grey (10 before) and up to 140 px inside the outline (80 before), and a pocket counts only if its median colour is within 12 of the grey, so pearls, crystal and silver that come close to the grey stay. (3) In a 6 px band along the background the alpha is capped by projecting each pixel between the grey and the mean colour of the nearby solid subject (Gaussian sigma 6); where that colour is itself within 40 of the grey, by the colour distance to the grey (0 at 14, 1 at 60). The colour of edge pixels is unmixed from the grey and blended toward the nearby subject colour by one minus the alpha, so faint edge pixels take the subject's colour instead of a light speck.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | As in the static batch A with the changes above. The loop script puts a circular alpha (soft edge of 2.25 px at 768, 0.56 px at 192) on every frame, because the portrait is shown in a circle (`docs/art-direction.md` 5.1); the class canvas is open (O-6) and D-1 sets 192 x 192 |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Yes | Written by the loop script into `work/` (table below) |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen in the concept, the master or the frames; the glint is a painted highlight clipped to the stone, not a glow |
| 5. Silhouette and smallest-size tests | In part | The agent looked at every frame at 192 px and at 2x on a dark backdrop and at frame 1 on a light one; face, expression and the moving part read. The smallest display size and frame band of an avatar are open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-15 by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/avatar/topaz_brooch/work/topaz_brooch_loop.py assets-src/cosmetic/avatar/topaz_brooch/concept assets-src/cosmetic/avatar/topaz_brooch/work assets-src/cosmetic/avatar/topaz_brooch/export
```

It printed `topaz_brooch_loop: blink edit registered (tx, ty, scale, angle) = (0.0841, 0.2793, 0.9996, -0.0003), residual 0.00097 -> 0.00093, eye colour offset [-4.7, -2.0, -1.6], parts 0, filled 0 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run into empty folders wrote files byte-identical to each other and to the committed ones.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes. Half-closed frames made by cross-fading showed two sets of lids and were dropped | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | a glint on the stone at (433, 532), ellipse 27 x 37 px at 768, frames 11 to 15 | A painted highlight on the gem: a soft streak that crosses the stone, clipped to the stone's ellipse, warm ivory at up to 0.55 opacity, normal blending, no halo outside the stone (not a glow, `docs/art-direction.md` section 8) |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 1.39 of 255, against 1.6 on average and 2.06 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `topaz_brooch: background [186.0, 186.0, 185.0], pockets 258 px, crop (45, -98, 1217, 1074), portrait 768`
- Crop `(92, 26, 1170, 1104)` to `(45, -98, 1217, 1074)` on the concept
- Scale k = 0.9198. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- `register` now clips a crop edge above the concept to 0 when it marks the registration region, as the later loop scripts of this batch do; with the old slice a negative top would have marked the wrong rows
- Printed by the loop: `topaz_brooch_loop: blink edit registered (tx, ty, scale, angle) = (-0.1894, 0.2223, 0.9997, -0.0000), residual 0.00089 -> 0.00084, eye colour offset [-4.4, -1.5, -1.2], parts 0, filled 0 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.068 before, 0.172 after; highest over the 16 loop frames 0.062 before, 0.167 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/topaz_brooch/work/topaz_brooch_cleanup.py topaz_brooch assets-src/cosmetic/avatar/topaz_brooch/concept/20261115-01.png assets-src/cosmetic/avatar/topaz_brooch/work
python -I assets-src/cosmetic/avatar/topaz_brooch/work/topaz_brooch_loop.py assets-src/cosmetic/avatar/topaz_brooch/concept assets-src/cosmetic/avatar/topaz_brooch/work assets-src/cosmetic/avatar/topaz_brooch/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 24 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/topaz_brooch` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `1eae55da1e7e7e6bddc155faccbc930da271b7bcb387708f208324f9c5b90311` |
| export/anim/0002.png | `e0d57f07367d6bc5f625efb10e02f23d8dbc0e9e898e6371e546e5567560f62a` |
| export/anim/0003.png | `8caa0322b439b2f63b3ef72bd3fafc1941136e9baca3c9eb2bd27c3b77333544` |
| export/anim/0004.png | `c8728ad68d5d60e5c3355e5934f1593f92bb724ca7789e1150cc5e2fc79bbdbd` |
| export/anim/0005.png | `285bfb184a9ca6adbf6356e8fd3b3f978ba0856a9296e1eddb5d0ec254ad1a5e` |
| export/anim/0006.png | `df35bd8cc9ed457b8dd7ad804fdee0fa2f80d65df38984660121815fffe48675` |
| export/anim/0007.png | `e7bef1f647c2c3de5f3992a0f49285ff44c2f02da3b0dac7cf178182c578e570` |
| export/anim/0008.png | `d845cf9418f95ac654bcae548e2444abdadd6b0f85ee9d0edbcde2632b92cd24` |
| export/anim/0009.png | `a79a939c41ad13a18dd2feaffd620279c377845ddaed138d1289abff16235819` |
| export/anim/0010.png | `2cc2b64b24d9b2e5c6cfe437c8a394e1d4eb2a319b1cb9755d5e8b73bfbb22e3` |
| export/anim/0011.png | `811bb9770289fcee7b5424a0c674d9377a24cee6a9464c2a501cae6f30ace49e` |
| export/anim/0012.png | `98ee29c1c2c5c89d6584b84263c2645bc2c4974403be542944bf225857fb6ab0` |
| export/anim/0013.png | `1f5a864efc9afa7a10e44e0942f98486961770770d93f3d5f67518c1108a5d57` |
| export/anim/0014.png | `2395d14ee788da42028b9ff7d317a6e72f7cf21c3a8da8f1abab53f742d76a2f` |
| export/anim/0015.png | `400a7fa16cbc32618a59cbd58b17f1cdd3a8d5a2631d813e5ed463e1b5707000` |
| export/anim/0016.png | `24cdef293d703736266d5255821dfb43610c64d400610b67d50f403e7cb07c63` |
| export/static.png | `a79a939c41ad13a18dd2feaffd620279c377845ddaed138d1289abff16235819` |
| work/topaz_brooch_base.png | `3d2e39656ca0dc4ad9959afb10ff48f5e4678a17f38e057699a4873315be7fc1` |
| work/topaz_brooch_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` |
| work/topaz_brooch_eyes_closed.png | `16086b4bd0936c9f72d6876bc69011e5a729b6bc1a703096fef0c3e3564636cd` |
| work/topaz_brooch_eyes_open.png | `2ff3400d274afb84ced469917f9a0408bf72c71eb1d3a58b0df7959324035041` |
| work/topaz_brooch_head.png | `aef4a013fd06ba8639fb7b1867a3d11ea6da8c1e5e9ed1446ea77ea4cc051b5d` |
| work/topaz_brooch_loop.py | `2fc67c96cf3f5a117c1f8b0be90647e286533409273f0de2c9025cb506641bc3` |
| work/topaz_brooch_portrait.png | `30d7451a4300fd5c6eb564c3bfec3e599f5d8fcaa45f89d456103e929f74c48f` |
| work/topaz_brooch_torso.png | `e865ec808010b7f2a01f11b51c7894bb269fef618986d47698d89341c6ce54cd` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `069ecf6f4cd5951dc3e60e6c966b40a337ef96f4ac9f87bc3db84213253e7632` | 1254 x 1254 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e020-56c8-7b91-9080-7b48a0212c29` |
| concept/20261115-02.png | `a73eee05f21caed1f7cf21cee10e5dece7ce5d01f4eeb4d238dc5285b9840e5c` | 1254 x 1254 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e021-5be9-75a3-a42a-feb40536615a` |
| work/topaz_brooch_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-15, from `amethyst_courtier_cleanup.py` |
| work/topaz_brooch_loop.py | `22723c560d0a0eb0546c5b1bed5795bc3c3da894779a5abbeb2ec6f962a54e46` | W-4 and W-5 script | Written by Claude Code, 2026-11-15; the same script for every avatar of this batch with the item's geometry |
| work/topaz_brooch_master.png | `d713d8d0d35bdda87d97b60b8337549c6de44f04706e9ab8eb9edfe1a808fc60` | 1254 x 1254 RGBA, the cut-out master | `topaz_brooch_cleanup.py` |
| work/topaz_brooch_base.png | `08788a2436d49470af336b1858efc78465ec5e644168342dbe57601a2f525925` | The portrait with the moving parts taken out (filled or cleared) | `topaz_brooch_loop.py` |
| work/topaz_brooch_eyes_closed.png | `559bef6c3b286a74255fab9853600249fa0418b79904568e833972190b597497` | Eye region, closed (aligned edit) through the same mask | `topaz_brooch_loop.py` |
| work/topaz_brooch_eyes_open.png | `52373ecde281f89272ae5e218a5bb401b7c9840e394f5f9f90371d3ade862c51` | Eye region, open (canonical concept) through the feathered eye mask | `topaz_brooch_loop.py` |
| work/topaz_brooch_head.png | `14ffde84f1b935e93d02e9dfe5f40ad495542445c887a0ae85db8100f4dd3668` | Head and hair: the portrait weighted by the head field of the warp | `topaz_brooch_loop.py` |
| work/topaz_brooch_portrait.png | `96869f8d3845a82d99c1a1687e8ca57dae0fe96b30757b4585e34e6793854df8` | 768 x 768 RGBA portrait crop, no circle | `topaz_brooch_cleanup.py` |
| work/topaz_brooch_torso.png | `2b24d483aed388973a3106c1e6b1b01e879e3f293dfb5f0f1e21e0ae4d823bba` | Torso and costume: the portrait weighted by one minus the head field | `topaz_brooch_loop.py` |
| export/static.png | `8ae20f3db217b6a10e7f2bf1159a628db1b5cce6c1e3fa98737e54a18cb0d2e1` | 192 x 192 RGBA static, loop frame 9 | `topaz_brooch_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `topaz_brooch_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted below. Sent 16:52:57 UTC; `c2pa.created` 2026-11-15T16:53:38Z |
| 2026-11-15 | concept/20261115-02.png | Edit of `concept/20261115-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 16:54:04 UTC; `c2pa.created` 2026-11-15T16:54:40Z |
| 2026-11-15 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/topaz_brooch_base.png, work/topaz_brooch_cleanup.py, work/topaz_brooch_eyes_closed.png, work/topaz_brooch_eyes_open.png, work/topaz_brooch_head.png, work/topaz_brooch_loop.py, work/topaz_brooch_portrait.png, work/topaz_brooch_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(92, 26, 1170, 1104)` to `(45, -98, 1217, 1074)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his fifties with rich dark brown skin, a full neatly rounded black beard with a few grey hairs, a deep chocolate brown velvet frock coat with gold frogging, and a large golden topaz brooch at the collar, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261115-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his fifties with rich dark brown skin, a full neatly rounded black beard with a few grey hairs, a deep chocolate brown velvet frock coat with gold frogging, and a large golden topaz brooch at the collar, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261115-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261115-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-15.
