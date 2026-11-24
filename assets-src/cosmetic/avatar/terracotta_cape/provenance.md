# avatar.terracotta_cape

Animated avatar of set 1 (#171), batch C, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-16 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, final runs on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-16 for the Plus plan and records the date here
- Canonical concept: concept/20261116-01.png
- Inputs uploaded to the generator: `concept/20261116-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261116-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a man in his twenties with deep brown skin, short twisted hair and a clean-shaven face, a cream linen shirt and a terracotta velvet half-cape thrown over one shoulder and fastened with a small brass clasp, the lower edge of the cape fully visible"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt: the terracotta velvet half-cape lies over his left shoulder (the viewer's right), fastened with a brass clasp at the chest. Not asked for: a high standing collar and gold embroidered bands on the shirt and the cape hem.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.02, 0.13 px, scale 0.9996 and rotation -0.0003 rad (mean squared grey difference outside the eyes 0.00080 before, 0.00078 after), and the eye patch was offset by [-10.0, -6.0, -2.8] in RGB to meet the canonical skin. The changes outside the eye mask are small and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, source `shop:offer.avatar_terracotta_cape`. The offer id is a placeholder until the shop record exists (V-15). One ambient motion, the velvet cape rippling over the shoulder. Animated avatars stay at `epic` and above as in batches A and B; batch C proposes 5 epic, 3 legendary, 2 mythic, which brings the 30 animated avatars of set 1 to 15 epic, 11 legendary and 4 mythic.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 1 animated detail (cape ripple), so `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-16; the signatures were not validated with a C2PA tool.

| Field | `concept/20261116-01.png` | `concept/20261116-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-16T08:30:39Z | `c2pa.created`, 2026-11-16T08:46:57Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-16 08:29:46 | 2026-11-16 08:46:20 |
| Codex session | `01a1e37a-031b-7930-aa96-de90c9f06c14` | `01a1e389-2c5f-7b80-ba79-b63887b65dc3` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked by a byte comparison). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/terracotta_cape/work/terracotta_cape_cleanup.py terracotta_cape assets-src/cosmetic/avatar/terracotta_cape/concept/20261116-01.png assets-src/cosmetic/avatar/terracotta_cape/work
```

It printed background [183.0, 183.0, 183.0], pockets 147 px, crop (34, 0, 1112, 1078), portrait 768. The script is `cornflower_ribbon_cleanup.py` of batch B unchanged (the batch A clean-up of `avatar.ruby_drop` with its 768 crop, grey-pocket rule and edge re-matte). It writes the master and a square 768 x 768 portrait crop (`work/terracotta_cape_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

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
python -I assets-src/cosmetic/avatar/terracotta_cape/work/terracotta_cape_loop.py assets-src/cosmetic/avatar/terracotta_cape/concept assets-src/cosmetic/avatar/terracotta_cape/work assets-src/cosmetic/avatar/terracotta_cape/export
```

It printed `terracotta_cape_loop: blink edit registered (tx, ty, scale, angle) = (0.0161, 0.1304, 0.9996, -0.0003), residual 0.00080 -> 0.00078, eye colour offset [-10.0, -6.0, -2.8], parts 0, ripples 1, filled 0 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run of the clean-up and the loop into empty folders wrote files byte-identical to each other and to the committed ones.

The script is the batch B loop script (`cornflower_ribbon_loop.py`, which carries the batch A rig of `ruby_drop_loop.py` with `flap`, `RIPPLES`, the clearing hole fill, clipped glints and the clipped registration region) with this item's geometry and one addition, the same for every avatar of batch C: a part can spin (`spin`, whole turns per loop added to its turn angle, so a full turn ends where it started). Parts without `spin` move exactly as in batch B.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient ripple | The velvet half-cape below the clasp, from the shoulder to the lower edge of the circle: inside a polygon feathered by Gaussian sigma 14 (768 scale), the pixels shift sideways by 10.0 x (sin(2 pi y / 240 - 1 t) - sin(2 pi y / 240)) px at 768, faded in over 160 px below y 460, so the wave runs down once per loop and the attached top stays still | Warp only, so no seams and no hole to fill; the clasp lies outside the polygon and stays still, and the cape's outer edge over the background moves with the folds |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 1.39 of 255, against 1.64 on average and 2.10 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `terracotta_cape: background [183.0, 183.0, 183.0], pockets 147 px, crop (0, -182, 1172, 990), portrait 768`
- Crop `(34, 0, 1112, 1078)` to `(0, -182, 1172, 990)` on the concept
- Scale k = 0.9198. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- Printed by the loop: `terracotta_cape_loop: blink edit registered (tx, ty, scale, angle) = (0.0090, 0.1338, 0.9996, -0.0003), residual 0.00071 -> 0.00070, eye colour offset [-10.0, -5.9, -2.8], parts 0, ripples 1, filled 0 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.016 before, 0.172 after; highest over the 16 loop frames 0.010 before, 0.167 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/terracotta_cape/work/terracotta_cape_cleanup.py terracotta_cape assets-src/cosmetic/avatar/terracotta_cape/concept/20261116-01.png assets-src/cosmetic/avatar/terracotta_cape/work
python -I assets-src/cosmetic/avatar/terracotta_cape/work/terracotta_cape_loop.py assets-src/cosmetic/avatar/terracotta_cape/concept assets-src/cosmetic/avatar/terracotta_cape/work assets-src/cosmetic/avatar/terracotta_cape/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 24 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/terracotta_cape` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `d08ffe18ca0a7d35513029dbeaa148bfbc73514ecadce1351beebb368cd7a323` |
| export/anim/0002.png | `f37838ba7b95b6848d9469b842a5990ed85af65626d77a81bb551e66fde91feb` |
| export/anim/0003.png | `462bcb82c0b41f20a3db0fd0638ec7e3d8413657cf44dc850a3923556189a455` |
| export/anim/0004.png | `0254611c06604fc375fd86452792551deff9095b0841d2cc0c46d0b4e70eaaa8` |
| export/anim/0005.png | `20f9d9af237de5ac28918de83f110f0ce123022c0af6d5ae850215279c6b5956` |
| export/anim/0006.png | `217d341270a06764877698d8ebd30a2d0fe036818b64c7d0eb43a991e510fb7b` |
| export/anim/0007.png | `c59ee40cbadb7c09b60ae4e4ce0f4e287715cd4b862956d636edd580644ecfbc` |
| export/anim/0008.png | `0fab3dab7cbfd0351350b5cff75daa1ab93306fb7b7cd3e494301f6804c96f97` |
| export/anim/0009.png | `b38bd98f65e1a8d7492edcc6d885c7a5fb43a0c893c55f06b7fc67c9271d4b72` |
| export/anim/0010.png | `6314bcf42e87455c01199e9887f09e59c9ab401cb2e14670c3aadd69b77fdbbc` |
| export/anim/0011.png | `c71c919dba5e63460cf9f48e9e02db5e10b4c5d72abcfe94c44243d029b48c72` |
| export/anim/0012.png | `918745c04d81131adde8fa1bb990e4d2e0663ecb29846529d3a548dc74865408` |
| export/anim/0013.png | `7381591f0b35f675bd24486412eb3179694c567c023d69bddd3aa731b68658be` |
| export/anim/0014.png | `4db5f341aa9e67177d54f070595368a869a2988a0e11c6d8a923de27746c8b93` |
| export/anim/0015.png | `e5a590ae24b7d0c92e8c04a2ccca8e9092c63e4b8eba2b6a9400e2d83ea67e98` |
| export/anim/0016.png | `6f10ccae1900be25d4b2fce786d80272ba3bf8fa6c88669bb00120644150c0ec` |
| export/static.png | `b38bd98f65e1a8d7492edcc6d885c7a5fb43a0c893c55f06b7fc67c9271d4b72` |
| work/terracotta_cape_base.png | `411317d0c4472973b9672291e240978de2efc64f92e4ca591b9f963bdff1ec02` |
| work/terracotta_cape_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` |
| work/terracotta_cape_eyes_closed.png | `0c5a5718a594ccb209577137b2bcdfdb67217897428361af25c37e7574878aeb` |
| work/terracotta_cape_eyes_open.png | `3215bdd1af7670a01ce81eb376c0ea628e912b49a29d3ba554eebef49355202b` |
| work/terracotta_cape_head.png | `fc19f610b26b5bfb2a622dff33dded5613d6f18927a263740f1ac35f852fbfa6` |
| work/terracotta_cape_loop.py | `c9923a0a90ccc50327879340d25af89f7c3f38a174f393d8c1603543f58c1800` |
| work/terracotta_cape_portrait.png | `feed9b9f2fa47dc6e7d85abf1bdb806a26e33529d1d6cac7b9107c88302e276f` |
| work/terracotta_cape_torso.png | `9588d181920991f0ed4df4396c1581b5d28ba816984e1f58315c18f7865028d0` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261116-01.png | `c8bccdefa5e9ba4976f3b2ebaea3f6c58e68bab582b8f0126c597cbe277d14ba` | 1254 x 1254 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e37a-031b-7930-aa96-de90c9f06c14` |
| concept/20261116-02.png | `b894a4610a0a39fca78da1113a5313d3bdad5b2392708e0ba75aa86d8c84a1e3` | 1254 x 1254 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e389-2c5f-7b80-ba79-b63887b65dc3` |
| work/terracotta_cape_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-16, from `cornflower_ribbon_cleanup.py` |
| work/terracotta_cape_loop.py | `c066ef96986736c23f0332b234a93595f26028835752a71c0650f176ad53c1ba` | W-4 and W-5 script | Written by Claude Code, 2026-11-16, from `cornflower_ribbon_loop.py`; the same script for every avatar of this batch with the item's geometry |
| work/terracotta_cape_master.png | `bec9e2385df1906bf2347e640327d689d26b9853a29d50d3ae7f49202721902e` | 1254 x 1254 RGBA, the cut-out master | `terracotta_cape_cleanup.py` |
| work/terracotta_cape_base.png | `9da270bf24125c7ae4e2ff9f05b44c7ce8fcaae2b6d0cdda19fe2af920d340b3` | The portrait with the moving parts taken out (filled or cleared) | `terracotta_cape_loop.py` |
| work/terracotta_cape_eyes_closed.png | `d8b941795c75b029acfc323438dcccbec33034824014027a845cb21ad9bab37a` | Eye region, closed (aligned edit) through the same mask | `terracotta_cape_loop.py` |
| work/terracotta_cape_eyes_open.png | `80b05727e3cc4cfc93b092352255597358513aacdadee01467d7d52ca326184c` | Eye region, open (canonical concept) through the feathered eye mask | `terracotta_cape_loop.py` |
| work/terracotta_cape_head.png | `c6e7a0fea2a24f1a53f1fc36631d28674a50d3b47a61adf3ce1c14585f814bf4` | Head and hair: the portrait weighted by the head field of the warp | `terracotta_cape_loop.py` |
| work/terracotta_cape_portrait.png | `5c63804c051ca6d7664bf8086af70dd26b0ed4032014615907852620f574d240` | 768 x 768 RGBA portrait crop, no circle | `terracotta_cape_cleanup.py` |
| work/terracotta_cape_torso.png | `497e25be8785e37e358a7f56dea153913b46df5f3b0d941f96cb138bcfd29fc2` | Torso and costume: the portrait weighted by one minus the head field | `terracotta_cape_loop.py` |
| export/static.png | `61254b349ce00852836465965afe8f419bb29bf17bb706ba36c211fc5a3837c4` | 192 x 192 RGBA static, loop frame 9 | `terracotta_cape_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `terracotta_cape_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-16 | concept/20261116-01.png | P-3 with the style key, quoted below. Sent 08:29:46 UTC; `c2pa.created` 2026-11-16T08:30:39Z |
| 2026-11-16 | concept/20261116-02.png | Edit of `concept/20261116-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 08:46:20 UTC; `c2pa.created` 2026-11-16T08:46:57Z |
| 2026-11-16 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/terracotta_cape_base.png, work/terracotta_cape_cleanup.py, work/terracotta_cape_eyes_closed.png, work/terracotta_cape_eyes_open.png, work/terracotta_cape_head.png, work/terracotta_cape_loop.py, work/terracotta_cape_portrait.png, work/terracotta_cape_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(34, 0, 1112, 1078)` to `(0, -182, 1172, 990)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261116-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his twenties with deep brown skin, short twisted hair and a clean-shaven face, a cream linen shirt and a terracotta velvet half-cape thrown over one shoulder and fastened with a small brass clasp, the lower edge of the cape fully visible, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261116-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261116-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his twenties with deep brown skin, short twisted hair and a clean-shaven face, a cream linen shirt and a terracotta velvet half-cape thrown over one shoulder and fastened with a small brass clasp, the lower edge of the cape fully visible, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261116-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261116-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.
