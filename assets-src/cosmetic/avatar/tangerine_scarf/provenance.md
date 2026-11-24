# avatar.tangerine_scarf

Animated avatar of set 1 (#171), batch B, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 and 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, final runs on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: `concept/20261115-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261115-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a woman in her fifties with warm medium-brown skin, black hair with silver streaks in a smooth low chignon, a tangerine silk scarf knotted loosely at the side of her neck with two short fringed ends, and a dove-grey silk gown with silver embroidery"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt; the fringed ends hang below the knot and the fringe falls outside the portrait crop. Not asked for: crystal drop earrings, a gold and amber hair ornament and an amber brooch on the gown.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.15, 0.10 px, scale 1.0000 and rotation -0.0001 rad (mean squared grey difference outside the eyes 0.00126 before, 0.00124 after), and the eye patch was offset by [-5.9, -0.4, 3.6] in RGB to meet the canonical skin. The changes outside the eye mask are small and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `legendary`, source `shop:offer.avatar_tangerine_scarf`. The offer id is a placeholder until the shop record exists (V-15). The scarf end ripples and both crystal earrings swing. Animated avatars stay at `epic` and above as in batch A; batch B proposes 5 epic, 4 legendary, 1 mythic, the same split as batch A.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 2 animated details (scarf end ripple, earrings (pair)), so `legendary`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signatures were not validated with a C2PA tool.

| Field | `concept/20261115-01.png` | `concept/20261115-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T17:32:10Z | `c2pa.created`, 2026-11-15T17:48:09Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 17:31:06 | 2026-11-15 17:47:05 |
| Codex session | `01a1e043-4228-7ec0-b7e8-474d6297e907` | `01a1e051-e49b-77b2-9df7-784e5c084ce0` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/tangerine_scarf/work/tangerine_scarf_cleanup.py tangerine_scarf assets-src/cosmetic/avatar/tangerine_scarf/concept/20261115-01.png assets-src/cosmetic/avatar/tangerine_scarf/work
```

It printed background [209.0, 209.0, 210.0], pockets 1717 px, crop (50, 0, 1128, 1078), portrait 768. The script is `ruby_drop_cleanup.py` of batch A unchanged. It writes the master and a square 768 x 768 portrait crop (`work/tangerine_scarf_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

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
python -I assets-src/cosmetic/avatar/tangerine_scarf/work/tangerine_scarf_loop.py assets-src/cosmetic/avatar/tangerine_scarf/concept assets-src/cosmetic/avatar/tangerine_scarf/work assets-src/cosmetic/avatar/tangerine_scarf/export
```

It printed `tangerine_scarf_loop: blink edit registered (tx, ty, scale, angle) = (0.1542, 0.0981, 1.0000, -0.0001), residual 0.00126 -> 0.00124, eye colour offset [-5.9, -0.4, 3.6], parts 2, ripples 1, filled 6883 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run into empty folders wrote files byte-identical to each other and to the committed ones.

The script is the batch A loop script (`ruby_drop_loop.py`) with this item's geometry and these additions, the same for every avatar of this batch: the concept file names are constants; a part can open and close (`flap`, a horizontal scale about its pivot); `RIPPLES` add a smooth sideways wave to the warp inside a feathered polygon, so cloth, ribbon and chain move without a cut-out layer and without seams; a hole behind a part is cleared where the fill estimate is less than half covered, which keeps a filled hole from leaving a smear over the background; a glint is clipped to the bright pixels of the portrait and its peak opacity is a constant (`GLINT_PEAK`, 0.55 as in batch A unless the item says otherwise); and the registration region is clipped to the image when the crop reaches past its edge.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `earring_left` (layer `work/tangerine_scarf_earring_left.png`, pivot (256, 364)) turns about its pivot by 5.0 degrees x (sin(lag) - sin(t + lag)), lag 0.4, the turn growing from 0 at the pivot to full at 60 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | A small part well inside the circle, so its motion never meets the circle edge |
| Ambient part | `earring_right` (layer `work/tangerine_scarf_earring_right.png`, pivot (556, 400)) turns about its pivot by 5.0 degrees x (sin(lag) - sin(t + lag)), lag 0.9, the turn growing from 0 at the pivot to full at 60 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | A small part well inside the circle, so its motion never meets the circle edge |
| Ambient ripple | The scarf end below the knot: inside a polygon feathered by Gaussian sigma 14 (768 scale), the pixels shift sideways by 5 x (sin(2 pi y / 180 - 1 t) - sin(2 pi y / 180)) px at 768, faded in over 70 px below y 570, so the wave runs down once per loop and the attached top stays still | Warp only, so no seams and no hole to fill; the polygon ends before the circle edge |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 2.77 of 255, against 2.61 on average and 3.19 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `tangerine_scarf: background [209.0, 209.0, 210.0], pockets 1717 px, crop (3, -195, 1175, 977), portrait 768`
- Crop `(50, 0, 1128, 1078)` to `(3, -195, 1175, 977)` on the concept
- Scale k = 0.9198. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- Printed by the loop: `tangerine_scarf_loop: blink edit registered (tx, ty, scale, angle) = (0.1547, 0.0955, 1.0000, -0.0001), residual 0.00101 -> 0.00099, eye colour offset [-6.1, -0.5, 3.6], parts 2, ripples 1, filled 6051 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.005 before, 0.172 after; highest over the 16 loop frames 0.000 before, 0.167 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/tangerine_scarf/work/tangerine_scarf_cleanup.py tangerine_scarf assets-src/cosmetic/avatar/tangerine_scarf/concept/20261115-01.png assets-src/cosmetic/avatar/tangerine_scarf/work
python -I assets-src/cosmetic/avatar/tangerine_scarf/work/tangerine_scarf_loop.py assets-src/cosmetic/avatar/tangerine_scarf/concept assets-src/cosmetic/avatar/tangerine_scarf/work assets-src/cosmetic/avatar/tangerine_scarf/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 26 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/tangerine_scarf` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `87315821e48039f679a8520a89409eefb024501f3321db8e42af5c02890c0a8c` |
| export/anim/0002.png | `d75adf59c5aa0e88cb43c44a0a88d7f755408cb6eee17e0140af41bc893e1813` |
| export/anim/0003.png | `96be11d0e3d7510be0fe9c8cc29a3c7dd4c30397ce6aba5c47b26eea60b85b9c` |
| export/anim/0004.png | `d2f7beb351a20ce8fee1225a264108ac7607d230220b61990d2644edc171c60d` |
| export/anim/0005.png | `77e535b1cd13e76107581281c9502e09218749545eeafd764c30ff21d6d283b9` |
| export/anim/0006.png | `8d517e6b35b78a5a239a689d091aec6a75c725c72690d629a879364f0c328764` |
| export/anim/0007.png | `f37c3740ea8375db334987150250175063bcf63372074c3a48ca16113c1d7861` |
| export/anim/0008.png | `fc5a010b1f18017b8495d96225d06cfc091cd643b597afab76632f51dbb980d0` |
| export/anim/0009.png | `e5da9ed7ac4dcdbd0af2d94a469c326d2f1ea5e45267985b85c3a15a6cf1a061` |
| export/anim/0010.png | `478aae80a67fa6e8fa04cf9045acd8119a03e10e361a65d7bef1d1826d4e4506` |
| export/anim/0011.png | `6c19d0f75b0bf4b0382839fbb52e9f6afb187a93c1c0c350958c01836760ea59` |
| export/anim/0012.png | `a155062ae85d0eeb5af5d58606ba6f0a79939f5a4ad4c0a0d3dfab77fdf441ea` |
| export/anim/0013.png | `339d197495886852fe98a6efcac2fdab9b69cd0eabfbd691712b650baaa37ac2` |
| export/anim/0014.png | `81ed6523e6ecd80fe300e222b3efa20693ff18f59b0065ca00a3a54e0ae31d6b` |
| export/anim/0015.png | `1cc92d4b122d93113144bd966ff6fd5ebf7d8aa64decd9aa754b5ca7d166c42e` |
| export/anim/0016.png | `c2a0726dde1094a566a8cf9775521da3f92752081427f6cede6826c5a7198559` |
| export/static.png | `e5da9ed7ac4dcdbd0af2d94a469c326d2f1ea5e45267985b85c3a15a6cf1a061` |
| work/tangerine_scarf_base.png | `202b0ad046c3dab322161bfe2fa6c405cd6e0a078f6396d5d06db2a12eb3baae` |
| work/tangerine_scarf_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` |
| work/tangerine_scarf_earring_left.png | `110da3e4eda6c6af9f4a88bf62a6dd05652c3da32d76a12a2d688f5ba83f830a` |
| work/tangerine_scarf_earring_right.png | `dd3459a9c963b1168eb129154ba9cb09a1d106193ba4fd11ae120239c7030455` |
| work/tangerine_scarf_eyes_closed.png | `35052d0a5bb3a78dd319e2e56aa66766c1a978aa694e2ebadf0fe1636c4c379c` |
| work/tangerine_scarf_eyes_open.png | `78d03a497cea97c8d90cd44a3e8947775444ece8219f82f17b3bf1b90320bf20` |
| work/tangerine_scarf_head.png | `15c8d09fba5082f28dee4676f806b1e32d7c6876e4c964da2673d5552d02a172` |
| work/tangerine_scarf_loop.py | `08dd20e5e2c68ddd37a78a8a2d9b7b56b848342f0769a5f1ec7420adc1e765f0` |
| work/tangerine_scarf_portrait.png | `c893518a0868a28c6bb10bb0274a7ec7cdca2c2e522d5e4980f4e5fecfe20168` |
| work/tangerine_scarf_torso.png | `bca1659e726e9ab366d04c0f33e65fa2bf48d6a440e9ef0b822862b36c69646b` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `544f033bfe3f671272ce4d8d0590e7e21fbd300951bd7db07f8b9588adba3f79` | 1254 x 1254 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e043-4228-7ec0-b7e8-474d6297e907` |
| concept/20261115-02.png | `c63ce8c5a2bd6b30d6972c8ba68d4894775b1cffe5baf38231e688788df1c131` | 1254 x 1254 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e051-e49b-77b2-9df7-784e5c084ce0` |
| work/tangerine_scarf_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_cleanup.py` |
| work/tangerine_scarf_loop.py | `69585ce7e88dc89db0528fe74be79af6d559d541fbd4a78c4b7043afd5cfb3a6` | W-4 and W-5 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_loop.py`; the same script for every avatar of this batch with the item's geometry |
| work/tangerine_scarf_master.png | `5ab24a67d96d82ff77bf2b9a0b5b897dccf3a67a584576421b301ffe46f04990` | 1254 x 1254 RGBA, the cut-out master | `tangerine_scarf_cleanup.py` |
| work/tangerine_scarf_base.png | `d8575e1e78eec499ef492c66cb6146de97ac1c1135ee402dc04a0174694ac054` | The portrait with the moving parts taken out (filled or cleared) | `tangerine_scarf_loop.py` |
| work/tangerine_scarf_earring_left.png | `37e02062f289a08287e2f1bc95b99d65823ad1c0ea626620d4dbbd4754557dd6` | Moving part `earring_left`: the portrait through its feathered polygon | `tangerine_scarf_loop.py` |
| work/tangerine_scarf_earring_right.png | `c26c215bcafe7d856b1810c81eb132c1cbdf4e2aa7d1252b3290121e0462b5f1` | Moving part `earring_right`: the portrait through its feathered polygon | `tangerine_scarf_loop.py` |
| work/tangerine_scarf_eyes_closed.png | `c750f1f8e9d7f7ddc6c98857500610b8979594658144317318308a7709471d04` | Eye region, closed (aligned edit) through the same mask | `tangerine_scarf_loop.py` |
| work/tangerine_scarf_eyes_open.png | `c71fb3767801d735041297b334a4dcbb024550fefd9fa33f6351927482758354` | Eye region, open (canonical concept) through the feathered eye mask | `tangerine_scarf_loop.py` |
| work/tangerine_scarf_head.png | `07ca00365f30a0b3e2b04d3aee19eee27fa1a44fee05f07e1f5810bac3f370f7` | Head and hair: the portrait weighted by the head field of the warp | `tangerine_scarf_loop.py` |
| work/tangerine_scarf_portrait.png | `e7b62a82cb9af99f154e3ea731758b24f7f61421e8a04af9271acfc8aeba539f` | 768 x 768 RGBA portrait crop, no circle | `tangerine_scarf_cleanup.py` |
| work/tangerine_scarf_torso.png | `4c74b56de0c982cbbf9a213c71ca65c98565c5cfeed58d6147fcf88e4e44c29a` | Torso and costume: the portrait weighted by one minus the head field | `tangerine_scarf_loop.py` |
| export/static.png | `7fd95af04074303a3d30f3c151612958bf8f8e796102431b93495430b689f978` | 192 x 192 RGBA static, loop frame 9 | `tangerine_scarf_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `tangerine_scarf_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted below. Sent 17:31:06 UTC; `c2pa.created` 2026-11-15T17:32:10Z |
| 2026-11-15 | concept/20261115-02.png | Edit of `concept/20261115-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 17:47:05 UTC; `c2pa.created` 2026-11-15T17:48:09Z |
| 2026-11-16 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/tangerine_scarf_base.png, work/tangerine_scarf_cleanup.py, work/tangerine_scarf_earring_left.png, work/tangerine_scarf_earring_right.png, work/tangerine_scarf_eyes_closed.png, work/tangerine_scarf_eyes_open.png, work/tangerine_scarf_head.png, work/tangerine_scarf_loop.py, work/tangerine_scarf_portrait.png, work/tangerine_scarf_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(50, 0, 1128, 1078)` to `(3, -195, 1175, 977)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her fifties with warm medium-brown skin, black hair with silver streaks in a smooth low chignon, a tangerine silk scarf knotted loosely at the side of her neck with two short fringed ends, and a dove-grey silk gown with silver embroidery, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261115-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her fifties with warm medium-brown skin, black hair with silver streaks in a smooth low chignon, a tangerine silk scarf knotted loosely at the side of her neck with two short fringed ends, and a dove-grey silk gown with silver embroidery, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261115-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261115-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.
