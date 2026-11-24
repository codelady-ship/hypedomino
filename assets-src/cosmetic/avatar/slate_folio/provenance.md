# avatar.slate_folio

Animated avatar of set 1 (#171), batch C, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-16 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, final runs on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-16 for the Plus plan and records the date here
- Canonical concept: concept/20261116-01.png
- Inputs uploaded to the generator: `concept/20261116-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261116-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a man in his seventies with fair skin, swept-back silver hair and a neat short grey beard, a slate-grey frock coat over a mustard-yellow silk waistcoat and a white cravat, holding a small open leather-bound book against his chest, the top corner of the right page curling up, the whole book and his hand fully visible"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt: he holds a small brown leather book open against his chest with his left hand (the viewer's right), and the top corner of a page curls up. Not asked for: a sapphire pin on the cravat, a crystal fob on a gold chain at the waistcoat, gold embroidery on the lapels and a jewelled oval ornament on the book cover.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.02, -0.17 px, scale 1.0000 and rotation -0.0003 rad (mean squared grey difference outside the eyes 0.00105 before, 0.00103 after), and the eye patch was offset by [-5.1, -2.4, -0.4] in RGB to meet the canonical skin. The changes outside the eye mask are small and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, source `shop:offer.avatar_slate_folio`. The offer id is a placeholder until the shop record exists (V-15). One ambient motion, the curled page corner lifting and settling. Animated avatars stay at `epic` and above as in batches A and B; batch C proposes 5 epic, 3 legendary, 2 mythic, which brings the 30 animated avatars of set 1 to 15 epic, 11 legendary and 4 mythic.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 1 animated detail (page curl), so `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-16; the signatures were not validated with a C2PA tool.

| Field | `concept/20261116-01.png` | `concept/20261116-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-16T08:25:15Z | `c2pa.created`, 2026-11-16T08:41:49Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-16 08:23:58 | 2026-11-16 08:40:45 |
| Codex session | `01a1e374-b282-7bf1-86f7-3affb1e2de57` | `01a1e384-115a-7a31-8156-48f1e79d85f9` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked by a byte comparison). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/slate_folio/work/slate_folio_cleanup.py slate_folio assets-src/cosmetic/avatar/slate_folio/concept/20261116-01.png assets-src/cosmetic/avatar/slate_folio/work
```

It printed background [189.0, 190.0, 191.0], pockets 547 px, crop (58, 0, 1136, 1078), portrait 768. The script is `cornflower_ribbon_cleanup.py` of batch B unchanged (the batch A clean-up of `avatar.ruby_drop` with its 768 crop, grey-pocket rule and edge re-matte). It writes the master and a square 768 x 768 portrait crop (`work/slate_folio_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

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
python -I assets-src/cosmetic/avatar/slate_folio/work/slate_folio_loop.py assets-src/cosmetic/avatar/slate_folio/concept assets-src/cosmetic/avatar/slate_folio/work assets-src/cosmetic/avatar/slate_folio/export
```

It printed `slate_folio_loop: blink edit registered (tx, ty, scale, angle) = (0.0167, -0.1699, 1.0000, -0.0003), residual 0.00105 -> 0.00103, eye colour offset [-5.1, -2.4, -0.4], parts 1, ripples 0, filled 4689 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run of the clean-up and the loop into empty folders wrote files byte-identical to each other and to the committed ones.

The script is the batch B loop script (`cornflower_ribbon_loop.py`, which carries the batch A rig of `ruby_drop_loop.py` with `flap`, `RIPPLES`, the clearing hole fill, clipped glints and the clipped registration region) with this item's geometry and one addition, the same for every avatar of batch C: a part can spin (`spin`, whole turns per loop added to its turn angle, so a full turn ends where it started). Parts without `spin` move exactly as in batch B.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `page_curl` (layer `work/slate_folio_page_curl.png`, pivot (640, 508)) turns about its pivot by 12.0 degrees x (sin(lag) - sin(t + lag) + 0.4 sin(2t)), lag 0.0, the turn growing from 0 at the pivot to full at 70 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | The curled page stands over the background above the book; the turn grows from the gutter, so the page stays attached to the book and its tip lifts and settles, inside the circle |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 2.40 of 255, against 2.27 on average and 3.02 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `slate_folio: background [189.0, 190.0, 191.0], pockets 547 px, crop (11, -184, 1183, 988), portrait 768`
- Crop `(58, 0, 1136, 1078)` to `(11, -184, 1183, 988)` on the concept
- Scale k = 0.9198. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- Printed by the loop: `slate_folio_loop: blink edit registered (tx, ty, scale, angle) = (0.0154, -0.1713, 1.0000, -0.0004), residual 0.00095 -> 0.00093, eye colour offset [-5.0, -2.4, -0.4], parts 1, ripples 0, filled 4068 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.016 before, 0.172 after; highest over the 16 loop frames 0.010 before, 0.167 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/slate_folio/work/slate_folio_cleanup.py slate_folio assets-src/cosmetic/avatar/slate_folio/concept/20261116-01.png assets-src/cosmetic/avatar/slate_folio/work
python -I assets-src/cosmetic/avatar/slate_folio/work/slate_folio_loop.py assets-src/cosmetic/avatar/slate_folio/concept assets-src/cosmetic/avatar/slate_folio/work assets-src/cosmetic/avatar/slate_folio/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 25 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/slate_folio` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `bd608a9781534b9d0080b5d69efc391f324b6490d71a138be88c8e862cfb9f7c` |
| export/anim/0002.png | `a2e64acf30da7cd1da12236035c75fd4b8d1f223897f4709b7061975e8dfcaa4` |
| export/anim/0003.png | `3d18d1794ed8a54fc64feee42be6a3919bfc71d17f9d99ca0a9e323e8db6000f` |
| export/anim/0004.png | `11f6046e8746878649a9c40a09cdad87fad4238b0b980d9ad492f64ccb445f7d` |
| export/anim/0005.png | `a9fa93bdf46017c9fb7f7868a6223d054edfbc69b15a014271147958f1b69aaf` |
| export/anim/0006.png | `ea1fdc5c0f4ec975a6a512a6ebb9f8a558bf1d2f882bb87d41c86f8fa4988476` |
| export/anim/0007.png | `c06193d2614b71af796f2335fb80070ecee13c66e49eb24ac7521c714564ca4f` |
| export/anim/0008.png | `614111d7bf46336dd7ccc526c7d6bdca742919ae065be05080e7c39dc35b5871` |
| export/anim/0009.png | `1a162ea55f4cbe060cb2b48ed20f8527b6158703ca188377c2ed279a1c952d56` |
| export/anim/0010.png | `6b580ee3c9abea24a726a5d00c2864973052f2c6e01a674cd5023f7b6afaee77` |
| export/anim/0011.png | `92b8c8596a8228e2b17a8a2cbecb05236db0b67a1cb268a4439eeb905f47b578` |
| export/anim/0012.png | `d93ea0987da7f238b51d5c467ea7618ab8d03d4e9929b74c606b43da618f3986` |
| export/anim/0013.png | `bfcd29908a7bbb7b47b8dcfaa131a009e896b400edbfcb149747abbae7c834c5` |
| export/anim/0014.png | `a2eacccebcec423b2415c971fbdddb5fa9f7a274aab32e1fc78b2e6c902960cf` |
| export/anim/0015.png | `4ec57b76cc8eab7cfa3cc6dd590721a5de598d09f0e3251ea5e6a511e2bd4bcd` |
| export/anim/0016.png | `02c077a2c476f5ec00ad4a236e70be7e322e1125cd5708d52d7cd5358e6462d4` |
| export/static.png | `1a162ea55f4cbe060cb2b48ed20f8527b6158703ca188377c2ed279a1c952d56` |
| work/slate_folio_base.png | `b55a987d6c082e0b676bed4a2a9e6b0a64d1028513828a85877cf1a66e20ad23` |
| work/slate_folio_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` |
| work/slate_folio_eyes_closed.png | `fde2ac68393ebef3fe193b7ad398fa54d4cdaaf1ca4984adf50109f6eaf720ed` |
| work/slate_folio_eyes_open.png | `b8361633f9a80c698d8fed4893a86e94ec91680cd98ae418318769e58e95b487` |
| work/slate_folio_head.png | `ba31e90e9729748fd8df7426b2b3735a02784738a5cd0d8fbc84742b571c68ed` |
| work/slate_folio_loop.py | `e6682cf9595610f7391d39121bc9a97391eed1fbe1a42965fdde3ec092f47ed4` |
| work/slate_folio_page_curl.png | `7ac01a41caceac20f237595ea32d9dac23b626e994f19b1286b9e55fc371ba9b` |
| work/slate_folio_portrait.png | `1dc671ee0f7bf4e0dd50bc0cfcfc7c2a7c27fff3538fe4ca8a228c8221584451` |
| work/slate_folio_torso.png | `f34e178eeb18e9ae89b70fa13bccbab770136494510a6888ce21bb0c17a210fc` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261116-01.png | `4b68e748031aa9dc8c92a7883268b71fb9aa7584dcdb29b682c4b5c7039ab931` | 1254 x 1254 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e374-b282-7bf1-86f7-3affb1e2de57` |
| concept/20261116-02.png | `a5cb7f947c2f7f9b0d0a61da797471804c154abac63ec26755ad17a51817bec1` | 1254 x 1254 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e384-115a-7a31-8156-48f1e79d85f9` |
| work/slate_folio_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-16, from `cornflower_ribbon_cleanup.py` |
| work/slate_folio_loop.py | `0a8c490785a1c5e59d99744bd5c6e86c2e27ff86b1e63021577a847d5c1d7cb6` | W-4 and W-5 script | Written by Claude Code, 2026-11-16, from `cornflower_ribbon_loop.py`; the same script for every avatar of this batch with the item's geometry |
| work/slate_folio_master.png | `06ca422abe99867fbafb84152f8d4a52ce4b3f2bca4b653def450782ad4b2d1b` | 1254 x 1254 RGBA, the cut-out master | `slate_folio_cleanup.py` |
| work/slate_folio_base.png | `f59ac501a6a63e3c3963bbca3b7c55b2f6653ae26608e8fc25bf3f51a3ade9ba` | The portrait with the moving parts taken out (filled or cleared) | `slate_folio_loop.py` |
| work/slate_folio_eyes_closed.png | `929f40f34bcb660b357ebcd870349d5e9abf2346c7b02924408fe8cdee5198d0` | Eye region, closed (aligned edit) through the same mask | `slate_folio_loop.py` |
| work/slate_folio_eyes_open.png | `ff5191365c0342de38a7e6d522cab6d92c8756f7b92f7e39c392a10aaa09cf35` | Eye region, open (canonical concept) through the feathered eye mask | `slate_folio_loop.py` |
| work/slate_folio_head.png | `755bb3d1a4827d1593a3800eecc4f2ef685da74acbc2a4bbd942a14645d2f476` | Head and hair: the portrait weighted by the head field of the warp | `slate_folio_loop.py` |
| work/slate_folio_page_curl.png | `6aea87b987bbe5d26c9e1c28fa07fd3837061f7cbbd86d89654f8f7790fe1511` | Moving part `page_curl`: the portrait through its feathered polygon | `slate_folio_loop.py` |
| work/slate_folio_portrait.png | `4362975897bd4de6c21bbd1ba5c16f6b015b9b9b77d853d4e2a53e4aee1cf85d` | 768 x 768 RGBA portrait crop, no circle | `slate_folio_cleanup.py` |
| work/slate_folio_torso.png | `b232dc56c6cc5272e5605257bb60477f34885b0d644b1ae58f4d94c2712285bf` | Torso and costume: the portrait weighted by one minus the head field | `slate_folio_loop.py` |
| export/static.png | `5b87991a1c28ab070739716a02098ce937eac2bafc977638ed0d94a5eee7b3dd` | 192 x 192 RGBA static, loop frame 9 | `slate_folio_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `slate_folio_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-16 | concept/20261116-01.png | P-3 with the style key, quoted below. Sent 08:23:58 UTC; `c2pa.created` 2026-11-16T08:25:15Z |
| 2026-11-16 | concept/20261116-02.png | Edit of `concept/20261116-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 08:40:45 UTC; `c2pa.created` 2026-11-16T08:41:49Z |
| 2026-11-16 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/slate_folio_base.png, work/slate_folio_cleanup.py, work/slate_folio_eyes_closed.png, work/slate_folio_eyes_open.png, work/slate_folio_head.png, work/slate_folio_loop.py, work/slate_folio_page_curl.png, work/slate_folio_portrait.png, work/slate_folio_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(58, 0, 1136, 1078)` to `(11, -184, 1183, 988)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261116-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his seventies with fair skin, swept-back silver hair and a neat short grey beard, a slate-grey frock coat over a mustard-yellow silk waistcoat and a white cravat, holding a small open leather-bound book against his chest, the top corner of the right page curling up, the whole book and his hand fully visible, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261116-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261116-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his seventies with fair skin, swept-back silver hair and a neat short grey beard, a slate-grey frock coat over a mustard-yellow silk waistcoat and a white cravat, holding a small open leather-bound book against his chest, the top corner of the right page curling up, the whole book and his hand fully visible, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261116-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261116-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.
