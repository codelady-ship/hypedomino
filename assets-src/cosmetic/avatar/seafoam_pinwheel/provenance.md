# avatar.seafoam_pinwheel

Animated avatar of set 1 (#171), batch C, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-16 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, final runs on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-16 for the Plus plan and records the date here
- Canonical concept: concept/20261116-01.png
- Inputs uploaded to the generator: `concept/20261116-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261116-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a woman in her twenties with medium-brown skin and short dark curls, small gold hoop studs, a seafoam-green silk dress with puffed sleeves, holding a small silk pinwheel of seafoam and pale peach petals on a slim gold stick up beside her shoulder, the whole pinwheel and her hand fully visible"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt: a four-bladed silk pinwheel of seafoam and pale peach on a gold stick is held up in her left hand (the viewer's right) beside her shoulder. Not asked for: a silk flower and ribbon hair ornament with pearls and a crystal drop, and a jewelled brooch on the bodice.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.24, -0.01 px, scale 1.0004 and rotation -0.0002 rad (mean squared grey difference outside the eyes 0.00140 before, 0.00134 after), and the eye patch was offset by [-3.9, -0.3, 1.1] in RGB to meet the canonical skin. The changes outside the eye mask are small and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, source `shop:offer.avatar_seafoam_pinwheel`. The offer id is a placeholder until the shop record exists (V-15). The strongest motion of the batch: the pinwheel spins a full turn every loop. Animated avatars stay at `epic` and above as in batches A and B; batch C proposes 5 epic, 3 legendary, 2 mythic, which brings the 30 animated avatars of set 1 to 15 epic, 11 legendary and 4 mythic.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 1 animated detail (pinwheel spin), so `epic`, was `mythic`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-16; the signatures were not validated with a C2PA tool.

| Field | `concept/20261116-01.png` | `concept/20261116-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-16T08:33:37Z | `c2pa.created`, 2026-11-16T08:48:24Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-16 08:32:51 | 2026-11-16 08:47:18 |
| Codex session | `01a1e37c-d5a1-7b60-93e7-2ce486f8f9d6` | `01a1e38a-10e4-7ea3-b52e-9003116eeb38` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked by a byte comparison). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/seafoam_pinwheel/work/seafoam_pinwheel_cleanup.py seafoam_pinwheel assets-src/cosmetic/avatar/seafoam_pinwheel/concept/20261116-01.png assets-src/cosmetic/avatar/seafoam_pinwheel/work
```

It printed background [202.0, 201.0, 201.0], pockets 3794 px, crop (137, 0, 1168, 1031), portrait 768. The script is `cornflower_ribbon_cleanup.py` of batch B unchanged (the batch A clean-up of `avatar.ruby_drop` with its 768 crop, grey-pocket rule and edge re-matte). It writes the master and a square 768 x 768 portrait crop (`work/seafoam_pinwheel_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | As in batches A and B. The loop script puts a circular alpha (soft edge of 2.25 px at 768, 0.56 px at 192) on every frame, because the portrait is shown in a circle (`docs/art-direction.md` 5.1); the class canvas is open (O-6) and D-1 sets 192 x 192 |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Yes | Written by the loop script into `work/` (table below); the ripples are warps of the base layer and need no layer of their own |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen in the concept, the master or the frames |
| 5. Silhouette and smallest-size tests | In part | The agent looked at every frame at 192 px and enlarged on a dark backdrop; face, expression and the moving part read. The smallest display size and frame band of an avatar are open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-16 (UTC) by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/avatar/seafoam_pinwheel/work/seafoam_pinwheel_loop.py assets-src/cosmetic/avatar/seafoam_pinwheel/concept assets-src/cosmetic/avatar/seafoam_pinwheel/work assets-src/cosmetic/avatar/seafoam_pinwheel/export
```

It printed `seafoam_pinwheel_loop: blink edit registered (tx, ty, scale, angle) = (0.2397, -0.0065, 1.0004, -0.0002), residual 0.00140 -> 0.00134, eye colour offset [-3.9, -0.3, 1.1], parts 1, ripples 0, filled 57860 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run of the clean-up and the loop into empty folders wrote files byte-identical to each other and to the committed ones.

The script is the batch B loop script (`cornflower_ribbon_loop.py`, which carries the batch A rig of `ruby_drop_loop.py` with `flap`, `RIPPLES`, the clearing hole fill, clipped glints and the clipped registration region) with this item's geometry and one addition, the same for every avatar of batch C: a part can spin (`spin`, whole turns per loop added to its turn angle, so a full turn ends where it started). Parts without `spin` move exactly as in batch B.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `pinwheel` (layer `work/seafoam_pinwheel_pinwheel.png`, pivot (632, 397)) turns 1 whole turn per loop anticlockwise about its pivot as one rigid layer (22.5 degrees per frame), so it spins without a seam at the loop point; the part also follows the head or torso warp at its pivot; the base layer under it is cleared, because it sits over the background | The wheel is cut by a disc around the hub that stops above the stick and left of her hair, so the stick and the hair stay still; at every angle the blades stay inside the circle |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 8.40 of 255, against 8.18 on average and 8.76 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `seafoam_pinwheel: background [202.0, 201.0, 201.0], pockets 3794 px, crop (78, -132, 1199, 989), portrait 768`
- Crop `(137, 0, 1168, 1031)` to `(78, -132, 1199, 989)` on the concept
- Scale k = 0.9197. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- Printed by the loop: `seafoam_pinwheel_loop: blink edit registered (tx, ty, scale, angle) = (0.2446, -0.0117, 1.0004, -0.0001), residual 0.00130 -> 0.00125, eye colour offset [-3.8, -0.3, 1.1], parts 1, ripples 0, filled 50222 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.062 before, 0.177 after; highest over the 16 loop frames 0.052 before, 0.167 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/seafoam_pinwheel/work/seafoam_pinwheel_cleanup.py seafoam_pinwheel assets-src/cosmetic/avatar/seafoam_pinwheel/concept/20261116-01.png assets-src/cosmetic/avatar/seafoam_pinwheel/work
python -I assets-src/cosmetic/avatar/seafoam_pinwheel/work/seafoam_pinwheel_loop.py assets-src/cosmetic/avatar/seafoam_pinwheel/concept assets-src/cosmetic/avatar/seafoam_pinwheel/work assets-src/cosmetic/avatar/seafoam_pinwheel/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 25 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/seafoam_pinwheel` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `56e1a1e8bb9046746962c9c7e79e39a427b0f999c73c98940e92e15575e33be7` |
| export/anim/0002.png | `37fc9caa09394b7128650fb33eeea6fd126146e1b742f1076b0fe96600b490fe` |
| export/anim/0003.png | `9eed484864ed5a322d7ef67268eb0422391cc97b95fc5ef0b55b56ffa41009e5` |
| export/anim/0004.png | `d93050fdfcc89a6cee0529862c38f450bed40293a78371ce57edf507b2c95710` |
| export/anim/0005.png | `419f2131cfd70a961f1062576afd6edfb62c189bba90ab8dedbbebfd83000677` |
| export/anim/0006.png | `9ef5998bdd5a697d17f245b389977e731def4d679c025bd2f4cdeb0639d66566` |
| export/anim/0007.png | `8a30ad0d8ff0e84d66b7a6843d86282822e96ada7bf8f2c5163e92e1980fa4af` |
| export/anim/0008.png | `022be8d89eff278c2bbea5eef4a46e7e86137e07ddd963c4e8750aadd714323b` |
| export/anim/0009.png | `e40bba52a5ed76d15ff288f59f164291e67893e8f1e50b5213e1cd7b09f5f983` |
| export/anim/0010.png | `5232544cfde0b21e705a1ac70691fff1e3fb6aceb34178f7927cb1f5fa2c2ef6` |
| export/anim/0011.png | `090c426b5599ae9abba83399a6cfcae4e5500a3c044a3e0ec2e5fbc84f24628a` |
| export/anim/0012.png | `f9e7c59695e22b0cf8bcb0278b4f0a36e5a350c92ba191b0f503b3e4aac06c3a` |
| export/anim/0013.png | `518b1b8012826ca4c605cd780720b6b83eea70db9b40a0009300d1253a2e7e97` |
| export/anim/0014.png | `211165df64c92b87507f4cf2ab08714b6134edd00f5130b3856fc8725c2738d4` |
| export/anim/0015.png | `dbdb484fd2db5cd22bf80360abb5c866d99c56ce8e398839e61efe9a4caadbc0` |
| export/anim/0016.png | `69cb9a6c26163bab87175b7cce1f596f04777a546e4e28a84797118c9ee97d8e` |
| export/static.png | `e40bba52a5ed76d15ff288f59f164291e67893e8f1e50b5213e1cd7b09f5f983` |
| work/seafoam_pinwheel_base.png | `9b267b28b921db9b3d801171785cd809931515ea8958cd4cddbfad0a6cb64fa1` |
| work/seafoam_pinwheel_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` |
| work/seafoam_pinwheel_eyes_closed.png | `5417a3de5652bdaf7ea4e0e666d0d69eb5bb57452c319f8e05a3482dde7c1b60` |
| work/seafoam_pinwheel_eyes_open.png | `14c3d39536787a70a7ebf12ed4fb2fc288769c9c1135f541a77d18327312642b` |
| work/seafoam_pinwheel_head.png | `9f3fe0be32393bc335dfea51ac3fa1076be40b2562c82b3f7914eac7f595c950` |
| work/seafoam_pinwheel_loop.py | `3e31ace07a2a56b0d737e751c602a444ca669320864b9e59b98b06156e06f8aa` |
| work/seafoam_pinwheel_pinwheel.png | `9110707d133e6e9054dee264cbd51b3f4bb5728efedc01f0772b2a247e5b23ab` |
| work/seafoam_pinwheel_portrait.png | `b72125d8ab1fd81108be99648c6e0908d1e0be66631e890390da9d20eae43f61` |
| work/seafoam_pinwheel_torso.png | `787b3790ea10ee3692c6e997ea6516198ef44393965600e9e01fe97128c94fc6` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261116-01.png | `43149977fb5ab66f126d763805e99e9ffc166bf56bb64f6939090c04355da96c` | 1199 x 1312 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e37c-d5a1-7b60-93e7-2ce486f8f9d6` |
| concept/20261116-02.png | `ed552c8f829bcbb6d2cd241a113ea3b9907f1f952be8a0761e0a85516b7255eb` | 1199 x 1312 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e38a-10e4-7ea3-b52e-9003116eeb38` |
| work/seafoam_pinwheel_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-16, from `cornflower_ribbon_cleanup.py` |
| work/seafoam_pinwheel_loop.py | `5bf6733cfd4e9bc98ddb99564751c0a1936bc750d5070032ba2d2379dd25371d` | W-4 and W-5 script | Written by Claude Code, 2026-11-16, from `cornflower_ribbon_loop.py`; the same script for every avatar of this batch with the item's geometry |
| work/seafoam_pinwheel_master.png | `b9cfb7260f074351dafac390a92b117929ee19f8a79d4d00a55f6b55250a131e` | 1199 x 1312 RGBA, the cut-out master | `seafoam_pinwheel_cleanup.py` |
| work/seafoam_pinwheel_base.png | `befb16ed999459d2a7e046e8041c2c336f30e4a46085b0f262bf39772675d25b` | The portrait with the moving parts taken out (filled or cleared) | `seafoam_pinwheel_loop.py` |
| work/seafoam_pinwheel_eyes_closed.png | `0fb593f0f485bcebf53090df2a7188049d64270ebca47ddbb9062fa6d8c48ed8` | Eye region, closed (aligned edit) through the same mask | `seafoam_pinwheel_loop.py` |
| work/seafoam_pinwheel_eyes_open.png | `c37491151df84fa55f322f7fa5c8b235944d70b1333c8217db318467d95bd6be` | Eye region, open (canonical concept) through the feathered eye mask | `seafoam_pinwheel_loop.py` |
| work/seafoam_pinwheel_head.png | `1821a2810eb7c772a28f93468bc251eeb81d8f36ee650c077ca2ccc392abb94f` | Head and hair: the portrait weighted by the head field of the warp | `seafoam_pinwheel_loop.py` |
| work/seafoam_pinwheel_pinwheel.png | `0e695406bbfb3223daad18790508943ae25e563010ad083660ce605dd2de385e` | Moving part `pinwheel`: the portrait through its feathered polygon | `seafoam_pinwheel_loop.py` |
| work/seafoam_pinwheel_portrait.png | `bb95e239c3d976b6e85e225838005be9b4eb5956a76a8c155c7ba3f36aa6ef47` | 768 x 768 RGBA portrait crop, no circle | `seafoam_pinwheel_cleanup.py` |
| work/seafoam_pinwheel_torso.png | `b32329df059346d34343117648be75f4500a35f38eeda3a2c5c330541e192f8e` | Torso and costume: the portrait weighted by one minus the head field | `seafoam_pinwheel_loop.py` |
| export/static.png | `ef825f44fc43018ed0f5c2548ca9a189d2e107d1beff421e55c8260dbd6647e7` | 192 x 192 RGBA static, loop frame 9 | `seafoam_pinwheel_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `seafoam_pinwheel_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-16 | concept/20261116-01.png | P-3 with the style key, quoted below. Sent 08:32:51 UTC; `c2pa.created` 2026-11-16T08:33:37Z |
| 2026-11-16 | concept/20261116-02.png | Edit of `concept/20261116-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 08:47:18 UTC; `c2pa.created` 2026-11-16T08:48:24Z |
| 2026-11-16 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/seafoam_pinwheel_base.png, work/seafoam_pinwheel_cleanup.py, work/seafoam_pinwheel_eyes_closed.png, work/seafoam_pinwheel_eyes_open.png, work/seafoam_pinwheel_head.png, work/seafoam_pinwheel_loop.py, work/seafoam_pinwheel_pinwheel.png, work/seafoam_pinwheel_portrait.png, work/seafoam_pinwheel_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(137, 0, 1168, 1031)` to `(78, -132, 1199, 989)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261116-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her twenties with medium-brown skin and short dark curls, small gold hoop studs, a seafoam-green silk dress with puffed sleeves, holding a small silk pinwheel of seafoam and pale peach petals on a slim gold stick up beside her shoulder, the whole pinwheel and her hand fully visible, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261116-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261116-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her twenties with medium-brown skin and short dark curls, small gold hoop studs, a seafoam-green silk dress with puffed sleeves, holding a small silk pinwheel of seafoam and pale peach petals on a slim gold stick up beside her shoulder, the whole pinwheel and her hand fully visible, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261116-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261116-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.
