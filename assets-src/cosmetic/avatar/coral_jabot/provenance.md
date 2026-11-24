# avatar.coral_jabot

Animated avatar of set 1 (#171), batch B, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 and 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, final runs on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: `concept/20261115-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261115-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a man in his early thirties with fair skin, light freckles and wavy chestnut-brown hair to the jaw, clean-shaven, a coral-red velvet frock coat with silver embroidery and a full, soft ivory lace jabot cascading from the collar"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt. Not asked for: silver buttons and a high embroidered collar on the coat.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.17, 0.17 px, scale 0.9998 and rotation -0.0001 rad (mean squared grey difference outside the eyes 0.00137 before, 0.00134 after), and the eye patch was offset by [-7.4, -1.7, 0.0] in RGB to meet the canonical skin. The changes outside the eye mask are small and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, source `shop:offer.avatar_coral_jabot`. The offer id is a placeholder until the shop record exists (V-15). One ambient motion, the lace ripple. Animated avatars stay at `epic` and above as in batch A; batch B proposes 5 epic, 4 legendary, 1 mythic, the same split as batch A.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 1 animated detail (jabot ripple), so `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signatures were not validated with a C2PA tool.

| Field | `concept/20261115-01.png` | `concept/20261115-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T17:24:21Z | `c2pa.created`, 2026-11-15T17:42:43Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 17:23:41 | 2026-11-15 17:41:36 |
| Codex session | `01a1e03c-7a77-7420-b8cc-524bddf0e938` | `01a1e04c-df2c-78f0-b53a-f146f42ee05c` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/coral_jabot/work/coral_jabot_cleanup.py coral_jabot assets-src/cosmetic/avatar/coral_jabot/concept/20261115-01.png assets-src/cosmetic/avatar/coral_jabot/work
```

It printed background [192.0, 191.0, 191.0], pockets 2275 px, crop (70, 0, 1148, 1078), portrait 768. The script is `ruby_drop_cleanup.py` of batch A unchanged. It writes the master and a square 768 x 768 portrait crop (`work/coral_jabot_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

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
python -I assets-src/cosmetic/avatar/coral_jabot/work/coral_jabot_loop.py assets-src/cosmetic/avatar/coral_jabot/concept assets-src/cosmetic/avatar/coral_jabot/work assets-src/cosmetic/avatar/coral_jabot/export
```

It printed `coral_jabot_loop: blink edit registered (tx, ty, scale, angle) = (0.1658, 0.1730, 0.9998, -0.0001), residual 0.00137 -> 0.00134, eye colour offset [-7.4, -1.7, 0.0], parts 0, ripples 1, filled 0 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run into empty folders wrote files byte-identical to each other and to the committed ones.

The script is the batch A loop script (`ruby_drop_loop.py`) with this item's geometry and these additions, the same for every avatar of this batch: the concept file names are constants; a part can open and close (`flap`, a horizontal scale about its pivot); `RIPPLES` add a smooth sideways wave to the warp inside a feathered polygon, so cloth, ribbon and chain move without a cut-out layer and without seams; a hole behind a part is cleared where the fill estimate is less than half covered, which keeps a filled hole from leaving a smear over the background; a glint is clipped to the bright pixels of the portrait and its peak opacity is a constant (`GLINT_PEAK`, 0.55 as in batch A unless the item says otherwise); and the registration region is clipped to the image when the crop reaches past its edge.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient ripple | The lace jabot, below the knot at the collar: inside a polygon feathered by Gaussian sigma 14 (768 scale), the pixels shift sideways by 5 x (sin(2 pi y / 150 - 2 t) - sin(2 pi y / 150)) px at 768, faded in over 90 px below y 470, so the wave runs down twice per loop and the attached top stays still | Warp only, so no seams and no hole to fill; the polygon ends before the circle edge |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 2.91 of 255, against 2.91 on average and 3.40 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `coral_jabot: background [192.0, 191.0, 191.0], pockets 2275 px, crop (23, -175, 1195, 997), portrait 768`
- Crop `(70, 0, 1148, 1078)` to `(23, -175, 1195, 997)` on the concept
- Scale k = 0.9198. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- Printed by the loop: `coral_jabot_loop: blink edit registered (tx, ty, scale, angle) = (0.1669, 0.1665, 0.9998, -0.0001), residual 0.00117 -> 0.00114, eye colour offset [-7.3, -1.7, 0.1], parts 0, ripples 1, filled 0 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.021 before, 0.172 after; highest over the 16 loop frames 0.016 before, 0.167 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/coral_jabot/work/coral_jabot_cleanup.py coral_jabot assets-src/cosmetic/avatar/coral_jabot/concept/20261115-01.png assets-src/cosmetic/avatar/coral_jabot/work
python -I assets-src/cosmetic/avatar/coral_jabot/work/coral_jabot_loop.py assets-src/cosmetic/avatar/coral_jabot/concept assets-src/cosmetic/avatar/coral_jabot/work assets-src/cosmetic/avatar/coral_jabot/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 24 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/coral_jabot` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `90d63796374526768e5ee9621b6bcc2554c2359bc9812ba984e85aa8446664f7` |
| export/anim/0002.png | `64c1b118c543dacc30f2a2ecb8196d74b48e57e14561471dbb31fc644cfbb3a9` |
| export/anim/0003.png | `5905eb15f4228b1c31d7bcc40a12235850de8c5387161ac1d7afd0e91d32c0db` |
| export/anim/0004.png | `7bb3b4ba6a105336d275f08ef0dd504fe4131b5f5c84e54d8627630be2fca17a` |
| export/anim/0005.png | `90e622404a612037880a35e1686ee441f1cca1d0c256d3a492ff684864a40886` |
| export/anim/0006.png | `4a8c288ee86b61e1501963dc577a7e0f5368d150622f05f201accdec9eec876e` |
| export/anim/0007.png | `5d117b7d4225939b32437ae1e038570890dffdd0ebff770deb1e02a59562077c` |
| export/anim/0008.png | `281b4aa45554aa706137ea24cf6a43e7a481efdedf1b2018b8c6c4ae76edaa90` |
| export/anim/0009.png | `5f1190cb651960e246118d9d2ca2b2e02a363c250da48e503623dbcfb7d4319a` |
| export/anim/0010.png | `9c00f797a27250f2133e8518f50285a3440e53f66e71b233add2bbba3bd7096b` |
| export/anim/0011.png | `84b8eb9fa4573523d5f0a0cc9ad514df5991c0ecb51c3a60c40b15614bd3ded3` |
| export/anim/0012.png | `0720bfe566bf891ebd1c56f48a2c5eb946c08253e19f9297590a3b52fb878ae8` |
| export/anim/0013.png | `d71d993a5d69c4d7e8c4b77db4731402e47447fbd54e1edcf1ce7bcb1252e1a1` |
| export/anim/0014.png | `a67fdc8bacdd43820a02d275bb42f8b8cdaf56a5cf70bc26d155e58f05aab134` |
| export/anim/0015.png | `a7c7ce37d46cf0c82e38ca62c90a736d38ee2c8a7e50f6c5535c985b5c2a4eac` |
| export/anim/0016.png | `755b183bc39be1b3a203390e28e14923f7d0ff8316ba42a2ec1e3d88382b8db5` |
| export/static.png | `5f1190cb651960e246118d9d2ca2b2e02a363c250da48e503623dbcfb7d4319a` |
| work/coral_jabot_base.png | `9cda40e3f5d920459cc860e5ec0bc1b0afb90cd2dc0f64bdb1d7d81799ffd933` |
| work/coral_jabot_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` |
| work/coral_jabot_eyes_closed.png | `23b88f462a92e109c1eaf7967558d721ea68d1ebce46164876149876f3a514f5` |
| work/coral_jabot_eyes_open.png | `24ea620ca2c03b902d89a3d9bd90755684bdbdf77152e89185f6e38f09ca25eb` |
| work/coral_jabot_head.png | `ee230a336349910368f83c247ce87d61907a4ad6cb961a3b5e80fb9ad5405be4` |
| work/coral_jabot_loop.py | `485465da61dcfab3b8280ff61a0681551790dd04c2e1c45b03c1df350130889d` |
| work/coral_jabot_portrait.png | `aca8a4e95614659c2be4329e191fca58a50e7d8cd4c49b69dd9d8cc96ebc05cf` |
| work/coral_jabot_torso.png | `26640ee0190235363a2b445ce0b4492f1bb36c626db3e9a8d22dbc40675adbb3` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `36e2d3dfbb273a6a5adc48d4861f518b9ce108ccf28f14e569ed261b0907dd3b` | 1254 x 1254 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e03c-7a77-7420-b8cc-524bddf0e938` |
| concept/20261115-02.png | `6d611020397e4a3efae175489b2aa6f6a002c1f55bd297d40c22f4e17b23fa49` | 1254 x 1254 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e04c-df2c-78f0-b53a-f146f42ee05c` |
| work/coral_jabot_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_cleanup.py` |
| work/coral_jabot_loop.py | `e6bc9df091fc0dc8eaf87ff1bcce433d9fd9d819459a66c41f020e71c4f62383` | W-4 and W-5 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_loop.py`; the same script for every avatar of this batch with the item's geometry |
| work/coral_jabot_master.png | `c067a76db24b1ee300d8dd4933f5c161ef9ee814415af393956a2380591d6293` | 1254 x 1254 RGBA, the cut-out master | `coral_jabot_cleanup.py` |
| work/coral_jabot_base.png | `745b518bf41f291bf085e88402e214893f60b5e3c6f56dcd15ac9ec2ca8d25ea` | The portrait with the moving parts taken out (filled or cleared) | `coral_jabot_loop.py` |
| work/coral_jabot_eyes_closed.png | `1c4408f45d6709b561501c42844da50ed23f72c504f715223d6fcfd917dccaf8` | Eye region, closed (aligned edit) through the same mask | `coral_jabot_loop.py` |
| work/coral_jabot_eyes_open.png | `b5ca9951e1cde2ce2ad5f0feb03287e8c929dc4f65cff7c7a4682045840925b9` | Eye region, open (canonical concept) through the feathered eye mask | `coral_jabot_loop.py` |
| work/coral_jabot_head.png | `641b64aec57ad368af61798187220c0b1954ee4e326fd2092c6b400362b8d635` | Head and hair: the portrait weighted by the head field of the warp | `coral_jabot_loop.py` |
| work/coral_jabot_portrait.png | `2e38b4b759408379d50e8cc19c4ca51b62d9fd9e9cccf356f3670f79b7622068` | 768 x 768 RGBA portrait crop, no circle | `coral_jabot_cleanup.py` |
| work/coral_jabot_torso.png | `2a9f3850f80f3fbec5edd4ab3cbea954ab9f883a9a2ff6de58b59d036403e8ae` | Torso and costume: the portrait weighted by one minus the head field | `coral_jabot_loop.py` |
| export/static.png | `1acb2e55182ecff864cb15e9dedce78f8e411e4ef6d97927c36de264ca4e2948` | 192 x 192 RGBA static, loop frame 9 | `coral_jabot_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `coral_jabot_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted below. Sent 17:23:41 UTC; `c2pa.created` 2026-11-15T17:24:21Z |
| 2026-11-15 | concept/20261115-02.png | Edit of `concept/20261115-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 17:41:36 UTC; `c2pa.created` 2026-11-15T17:42:43Z |
| 2026-11-16 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/coral_jabot_base.png, work/coral_jabot_cleanup.py, work/coral_jabot_eyes_closed.png, work/coral_jabot_eyes_open.png, work/coral_jabot_head.png, work/coral_jabot_loop.py, work/coral_jabot_portrait.png, work/coral_jabot_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(70, 0, 1148, 1078)` to `(23, -175, 1195, 997)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his early thirties with fair skin, light freckles and wavy chestnut-brown hair to the jaw, clean-shaven, a coral-red velvet frock coat with silver embroidery and a full, soft ivory lace jabot cascading from the collar, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261115-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his early thirties with fair skin, light freckles and wavy chestnut-brown hair to the jaw, clean-shaven, a coral-red velvet frock coat with silver embroidery and a full, soft ivory lace jabot cascading from the collar, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261115-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261115-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.
