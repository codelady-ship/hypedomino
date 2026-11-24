# avatar.teal_boa

Animated avatar of set 1 (#171), batch C, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-16 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, final runs on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-16 for the Plus plan and records the date here
- Canonical concept: concept/20261116-01.png
- Inputs uploaded to the generator: `concept/20261116-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261116-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a woman in her fifties with warm golden-brown skin and black hair with one silver streak in a smooth updo, small diamond studs, a deep teal silk gown, and a soft smoky-grey ostrich-feather boa draped around her shoulders, fully visible"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt: a smoky-grey ostrich-feather boa lies around her shoulders over a deep teal gown. Not asked for: a jewelled hair ornament with cream flowers and pearl strands, a diamond necklace with a teal drop and embroidered flowers on the gown.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.24, -0.14 px, scale 1.0000 and rotation -0.0002 rad (mean squared grey difference outside the eyes 0.00164 before, 0.00157 after), and the eye patch was offset by [-6.4, -2.1, -0.9] in RGB to meet the canonical skin. The changes outside the eye mask are small and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, source `shop:offer.avatar_teal_boa`. The offer id is a placeholder until the shop record exists (V-15). One ambient motion, the feather boa rippling on both shoulders. Animated avatars stay at `epic` and above as in batches A and B; batch C proposes 5 epic, 3 legendary, 2 mythic, which brings the 30 animated avatars of set 1 to 15 epic, 11 legendary and 4 mythic.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 1 animated detail (boa ripple (both shoulders)), so `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-16; the signatures were not validated with a C2PA tool.

| Field | `concept/20261116-01.png` | `concept/20261116-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-16T08:38:47Z | `c2pa.created`, 2026-11-16T08:54:32Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-16 08:38:09 | 2026-11-16 08:53:45 |
| Codex session | `01a1e381-afcf-7ca0-a8e8-edd35002d996` | `01a1e38f-f927-72e2-a412-79e4c0b280c6` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked by a byte comparison). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/teal_boa/work/teal_boa_cleanup.py teal_boa assets-src/cosmetic/avatar/teal_boa/concept/20261116-01.png assets-src/cosmetic/avatar/teal_boa/work
```

It printed background [196.0, 195.0, 195.0], pockets 15704 px, crop (56, 0, 1053, 997), portrait 768. The script is `cornflower_ribbon_cleanup.py` of batch B unchanged (the batch A clean-up of `avatar.ruby_drop` with its 768 crop, grey-pocket rule and edge re-matte). It writes the master and a square 768 x 768 portrait crop (`work/teal_boa_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

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
python -I assets-src/cosmetic/avatar/teal_boa/work/teal_boa_loop.py assets-src/cosmetic/avatar/teal_boa/concept assets-src/cosmetic/avatar/teal_boa/work assets-src/cosmetic/avatar/teal_boa/export
```

It printed `teal_boa_loop: blink edit registered (tx, ty, scale, angle) = (0.2421, -0.1400, 1.0000, -0.0002), residual 0.00164 -> 0.00157, eye colour offset [-6.4, -2.1, -0.9], parts 0, ripples 2, filled 0 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run of the clean-up and the loop into empty folders wrote files byte-identical to each other and to the committed ones.

The script is the batch B loop script (`cornflower_ribbon_loop.py`, which carries the batch A rig of `ruby_drop_loop.py` with `flap`, `RIPPLES`, the clearing hole fill, clipped glints and the clipped registration region) with this item's geometry and one addition, the same for every avatar of batch C: a part can spin (`spin`, whole turns per loop added to its turn angle, so a full turn ends where it started). Parts without `spin` move exactly as in batch B.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient ripple | The feather boa on her right shoulder (the viewer's left): inside a polygon feathered by Gaussian sigma 14 (768 scale), the pixels shift sideways by 9.0 x (sin(2 pi y / 160 - 1 t) - sin(2 pi y / 160)) px at 768, faded in over 120 px below y 430, so the wave runs down once per loop and the attached top stays still | Warp only, so no seams and no hole to fill; the short wavelength moves the feather tips over the background |
| Ambient ripple | The feather boa on her left shoulder (the viewer's right): inside a polygon feathered by Gaussian sigma 14 (768 scale), the pixels shift sideways by 9.0 x (sin(2 pi y / 160 - 1 t) - sin(2 pi y / 160)) px at 768, faded in over 120 px below y 440, so the wave runs down once per loop and the attached top stays still | Warp only, so no seams and no hole to fill; the short wavelength moves the feather tips over the background |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 3.44 of 255, against 3.05 on average and 3.56 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `teal_boa: background [196.0, 195.0, 195.0], pockets 15704 px, crop (12, -178, 1096, 906), portrait 768`
- Crop `(56, 0, 1053, 997)` to `(12, -178, 1096, 906)` on the concept
- Scale k = 0.9197. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- Printed by the loop: `teal_boa_loop: blink edit registered (tx, ty, scale, angle) = (0.2491, -0.1323, 1.0000, -0.0002), residual 0.00138 -> 0.00131, eye colour offset [-6.3, -2.1, -0.8], parts 0, ripples 2, filled 0 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.005 before, 0.172 after; highest over the 16 loop frames 0.000 before, 0.161 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/teal_boa/work/teal_boa_cleanup.py teal_boa assets-src/cosmetic/avatar/teal_boa/concept/20261116-01.png assets-src/cosmetic/avatar/teal_boa/work
python -I assets-src/cosmetic/avatar/teal_boa/work/teal_boa_loop.py assets-src/cosmetic/avatar/teal_boa/concept assets-src/cosmetic/avatar/teal_boa/work assets-src/cosmetic/avatar/teal_boa/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 24 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/teal_boa` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `eaf562fdf7a69bda963f53fbf2635d824193fab73b88678b6d7c034593cbb2be` |
| export/anim/0002.png | `4b2fafa14a90394389de36ec72f722f2f0ce6d1bd0e0af407e21d51b844d0821` |
| export/anim/0003.png | `3602e80c953acc8aa9a2d46bc8e3a7a85003989b860e66104665356668d98e74` |
| export/anim/0004.png | `653551b51bbc684d8f802557f64327c278a2cbcd9365d2ed40fdec16411dd580` |
| export/anim/0005.png | `9be3a16437291945c5f3aa3d19c7287956007d0872573a09d0f6294c80c31cd3` |
| export/anim/0006.png | `21394b81f92687d415c2bc626ba6238c1a7658837d551a52e22c3a0d7a2daabd` |
| export/anim/0007.png | `5deb91037a683d7c1f1b2ff9e71139c47a9c9ad8d576e31c5a83abcb41ddcfa6` |
| export/anim/0008.png | `c5fbfd0a9f4a97606da2f021d005934a2d5a6b0c8cf4acf0fb374d9d5476e917` |
| export/anim/0009.png | `b4610138843f0157cf986ad60d850e56a46940ee1a9b4b76385039d4d2127544` |
| export/anim/0010.png | `33247162d62f4f8d52590f13ee83d365ac44cb4e5bfada11e24f00ac73e4a76b` |
| export/anim/0011.png | `f5a76c809baaa353217f6c7cb6f0c65095af21f3ec4ede0492099a603518cb61` |
| export/anim/0012.png | `f72979a9c31f099ace76e3ff51cd0622b5f94735fd8928a16c3bd1e555925664` |
| export/anim/0013.png | `108de1d4bfe20099c7e994d731b2a8cd7ba11fddf9cdb79644512dce1784dda5` |
| export/anim/0014.png | `03180a9b111e6b7600c7f6221d79637b8caa28e8584de8f117c47128e0642e00` |
| export/anim/0015.png | `3fe58d594c0ed9913dc6be49f03636ffcca313a1bf58e66fe3e86cb915371921` |
| export/anim/0016.png | `9d710004e625a662f29ab15201e2a34cada9733aa0c0ef2022ebd616b2e0be1b` |
| export/static.png | `b4610138843f0157cf986ad60d850e56a46940ee1a9b4b76385039d4d2127544` |
| work/teal_boa_base.png | `72e814a39e22fac4339d96ca7a479d4ff93d057a6003bfee695616c9753e6564` |
| work/teal_boa_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` |
| work/teal_boa_eyes_closed.png | `742760064272a5e6f67c3e421f2231f55ca3d75382ea704ef0acabb338d51a6c` |
| work/teal_boa_eyes_open.png | `1efbebcbbf8198b3f3517f547ec46932fbf55d52648c071872bbc294af024721` |
| work/teal_boa_head.png | `665fa00d796831a667d9977558a1f2b77c91fadd9d257eb0b2528e21cee6e303` |
| work/teal_boa_loop.py | `64a3ee782279e117619b5c8963fddf11efa806e0788c605307569298a315880c` |
| work/teal_boa_portrait.png | `e65186e546156ac85d49913b8050a94ff5a5e1e11bb7adad0ff986db5f28323a` |
| work/teal_boa_torso.png | `357a101cadcf37c89bf4e7299162e98dba0ae07472ad900899d0875dfd2f91ed` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261116-01.png | `daa5476d1510cf86ac96b00616fee08ccd46d06ea2d9c63e2c127170c83d749d` | 1160 x 1356 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e381-afcf-7ca0-a8e8-edd35002d996` |
| concept/20261116-02.png | `425c8bccc62829274da645f4ca94e68fce1d6f6e27c234f67bbc0ef811d9d3ad` | 1159 x 1356 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e38f-f927-72e2-a412-79e4c0b280c6` |
| work/teal_boa_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-16, from `cornflower_ribbon_cleanup.py` |
| work/teal_boa_loop.py | `fe0c313d512a77db2b88be94ee1cfd6b209ef765d2a8f69d3f98362613760f1b` | W-4 and W-5 script | Written by Claude Code, 2026-11-16, from `cornflower_ribbon_loop.py`; the same script for every avatar of this batch with the item's geometry |
| work/teal_boa_master.png | `903337df9d34326261cab5f5b708dbdbed10a0d091e7b484dcac50621c971e13` | 1160 x 1356 RGBA, the cut-out master | `teal_boa_cleanup.py` |
| work/teal_boa_base.png | `1967c2ab49d6f587476ab07363e81a0167babcd09874a362e28e1e39b6056ce7` | The portrait with the moving parts taken out (filled or cleared) | `teal_boa_loop.py` |
| work/teal_boa_eyes_closed.png | `b7f29bd52e0b8db38f7ffc82a7763bb48fd561e4e095adf884c7512a447aae6c` | Eye region, closed (aligned edit) through the same mask | `teal_boa_loop.py` |
| work/teal_boa_eyes_open.png | `8129c1a227c84769cd2dc340330d924ce5783a03591ad3f0a2aa6b19a237f9db` | Eye region, open (canonical concept) through the feathered eye mask | `teal_boa_loop.py` |
| work/teal_boa_head.png | `f73cd59b89c9f71ff53efd6a67d3fb2de5c72b63c4e57ddaf937e1db802c3c09` | Head and hair: the portrait weighted by the head field of the warp | `teal_boa_loop.py` |
| work/teal_boa_portrait.png | `7a2e8834f53e8bd9d0e9613d507a629a70b1cb249bf8a2bee6aac791cccb371b` | 768 x 768 RGBA portrait crop, no circle | `teal_boa_cleanup.py` |
| work/teal_boa_torso.png | `9e61c4a0865390a74462c455f7ec5cb9b002b03e1a4bd1b281d4328855a5fb9e` | Torso and costume: the portrait weighted by one minus the head field | `teal_boa_loop.py` |
| export/static.png | `7a6a814f3c860614071fdc4076034b8dfa96e8a7165ff59f138d928500d972f6` | 192 x 192 RGBA static, loop frame 9 | `teal_boa_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `teal_boa_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-16 | concept/20261116-01.png | P-3 with the style key, quoted below. Sent 08:38:09 UTC; `c2pa.created` 2026-11-16T08:38:47Z |
| 2026-11-16 | concept/20261116-02.png | Edit of `concept/20261116-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 08:53:45 UTC; `c2pa.created` 2026-11-16T08:54:32Z |
| 2026-11-16 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/teal_boa_base.png, work/teal_boa_cleanup.py, work/teal_boa_eyes_closed.png, work/teal_boa_eyes_open.png, work/teal_boa_head.png, work/teal_boa_loop.py, work/teal_boa_portrait.png, work/teal_boa_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(56, 0, 1053, 997)` to `(12, -178, 1096, 906)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261116-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her fifties with warm golden-brown skin and black hair with one silver streak in a smooth updo, small diamond studs, a deep teal silk gown, and a soft smoky-grey ostrich-feather boa draped around her shoulders, fully visible, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261116-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261116-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her fifties with warm golden-brown skin and black hair with one silver streak in a smooth updo, small diamond studs, a deep teal silk gown, and a soft smoky-grey ostrich-feather boa draped around her shoulders, fully visible, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261116-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261116-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.
