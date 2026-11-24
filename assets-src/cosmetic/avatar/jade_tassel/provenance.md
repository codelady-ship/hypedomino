# avatar.jade_tassel

Animated avatar of set 1 (#171), a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. The item was re-made on 2026-11-16 from a new concept by owner decision (avatars show adults only); the first concept and everything built from it are recorded under "Superseded, 2026-11-16" below. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-16 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15; `codex login status` on 2026-11-16 printed `Logged in using ChatGPT`)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-16 for the Plus plan and records the date here
- Canonical concept: concept/20261116-01.png
- Inputs uploaded to the generator: `concept/20261116-01.png`, our own canonical concept of this item, attached with `codex exec --image` to the one edit that made `concept/20261116-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "an adult woman in her early thirties with light golden skin, black hair in two round buns tied with jade green silk ribbons that end in small gold tassels hanging beside her cheeks, and a quilted cream silk jacket with jade toggle buttons"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt: two round buns with jade green silk bows; each ribbon end carries a gold bead and a small gold tassel beside the cheeks. Not asked for: jade drop earrings with a crystal drop, cream fleece trim on the shoulders, gold-mounted jade and pearl clasps with crystal drops on the shoulders, green sleeves and embroidered blossoms on the jacket.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.07, 0.04 px, scale 1.0001 and rotation 0.0000 rad (mean squared grey difference outside the eyes 0.00084 before, 0.00084 after), and the eye patch was offset by [-2.9, -1.4, 0.9] in RGB to meet the canonical skin. The changes outside the eye mask are small and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: the person is an adult (a woman in her early thirties); no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, source `shop:offer.avatar_jade_tassel`. The offer id is a placeholder until the shop record exists (V-15). Animated avatars are proposed at `epic` and above, as the batch leans to the higher tiers of `docs/rarity.md`; this batch proposes 5 epic, 4 legendary, 1 mythic.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 1 animated detail (tassels (pair)), so `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-16; the signatures were not validated with a C2PA tool.

| Field | `concept/20261116-01.png` | `concept/20261116-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-16T13:24:01Z | `c2pa.created`, 2026-11-16T13:26:08Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-16 13:22:53 | 2026-11-16 13:25:00 |
| Codex session | `01a1e486-60ff-7e33-b2af-be0ee51cfc67` | `01a1e488-52f1-73e3-bb20-d3d7ff392571` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). No generation for this item was rejected: these are the only two images generated for the new concept.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/jade_tassel/work/jade_tassel_cleanup.py jade_tassel assets-src/cosmetic/avatar/jade_tassel/concept/20261116-01.png assets-src/cosmetic/avatar/jade_tassel/work
```

It printed `jade_tassel: background [198.0, 197.0, 197.0], pockets 6964 px, crop (39, -175, 1211, 997), portrait 768`. The script is `cornflower_ribbon_cleanup.py` after the re-crop of 2026-11-16 (#733) unchanged: `CROP_SHARE` 0.935, `HEADROOM` 0.17 and the crop may reach above the top of the concept (`upper = min(top - HEADROOM * side, height - side)`), so the portrait has the same size and head height in the circle as the other re-cropped avatars. The first concept needed `HEADROOM` 0.12 to keep a tassel in the circle; this one does not. It writes the master and a square 768 x 768 portrait crop (`work/jade_tassel_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | The loop script puts a circular alpha (soft edge of 2.25 px at 768, 0.56 px at 192) on every frame, because the portrait is shown in a circle (`docs/art-direction.md` 5.1); the class canvas is open (O-6) and D-1 sets 192 x 192 |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Yes | Written by the loop script into `work/` (table below) |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen in the concept, the master or the frames |
| 5. Silhouette and smallest-size tests | In part | The agent looked at every frame at 192 px and enlarged on a dark backdrop, and at the static inside the frames `frame.silk_ribbon`, `frame.tide_pearl` and `frame.season1_platinum` at 256 px; face, expression and the moving part read and the head stays clear of the frames' top band. The smallest display size and frame band of an avatar are open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-16 (UTC) by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/avatar/jade_tassel/work/jade_tassel_loop.py assets-src/cosmetic/avatar/jade_tassel/concept assets-src/cosmetic/avatar/jade_tassel/work assets-src/cosmetic/avatar/jade_tassel/export
```

It printed `jade_tassel_loop: blink edit registered (tx, ty, scale, angle) = (0.0655, 0.0446, 1.0001, -0.0000), residual 0.00084 -> 0.00084, eye colour offset [-2.9, -1.4, 0.9], parts 2, ripples 0, filled 10442 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha.

The script is `cornflower_ribbon_loop.py` after the re-crop (#733) with this item's file names and geometry (`CANONICAL`, `BLINK_EDIT`, `CROP`, `HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `EYES`, `PARTS`). The geometry was read by the agent from the 768 portrait on a coordinate grid; the eye ellipses from the difference between the canonical and the edit.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 4.6 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `tassel_left` (layer `work/jade_tassel_tassel_left.png`, pivot (200, 451)) turns about its pivot by 7.0 degrees x (sin(lag) - sin(t + lag)), lag 0.6, t = 2 pi (f - 8) / 16, the turn growing from 0 at the pivot to full at 100 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; it hangs over the background, so the hole it leaves in the base layer is transparent | The bead and tassel below the left ribbon end (the viewer's left); the cut is at the cord under the ribbon, where the turn is 0, so no seam shows |
| Ambient part | `tassel_right` (layer `work/jade_tassel_tassel_right.png`, pivot (577, 446)) turns about its pivot by 7.0 degrees x (sin(lag) - sin(t + lag)), lag 0.9, t = 2 pi (f - 8) / 16, the turn growing from 0 at the pivot to full at 100 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; it hangs over the background, so the hole it leaves in the base layer is transparent | The same for the right ribbon end, with a different lag so the two do not swing in step |
| Inside the circle | `tassel_left` 0.00% outside at rest and in every frame, farthest pixel at 0.713 of the radius; `tassel_right` 0.00%, 0.711 | Measured by the agent on 2026-11-16: the share of each part layer's alpha that lies outside the circle of radius 384 at 768, at rest and in every one of the 16 frames after its swing |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 3.16 of 255, against 2.77 on average and 3.41 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Head top | First row of `export/static.png` with alpha above 0.5 at 0.172 of the canvas height, highest over the 16 frames 0.167 | The same as the re-cropped avatars of #733 |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Determinism: two more runs of both commands into two empty folders wrote the same 26 files byte for byte, and both match the files committed here.

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261116-01.png | `19909038e96d23fcac266789132beac9054d5c583f0277abb577b19b4e68b4bb` | 1254 x 1254 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e486-60ff-7e33-b2af-be0ee51cfc67` |
| concept/20261116-02.png | `89543ccb2f66c232366bbaf1ed1909b6d9289e132edd7af9cef3115716f2ac3a` | 1254 x 1254 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e488-52f1-73e3-bb20-d3d7ff392571` |
| work/jade_tassel_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` | W-3 script | `cornflower_ribbon_cleanup.py` after #733, unchanged |
| work/jade_tassel_loop.py | `565bb2ae54d73eeed912ce3dbdc93dd6b8fea957562bfa99830670cc776cea58` | W-4 and W-5 script | Written by Claude Code, 2026-11-16, from `cornflower_ribbon_loop.py` after #733 with this item's geometry |
| work/jade_tassel_master.png | `6bbd0c1c46f65f7e2c0a3ac9d6c3fd07a4934209f1bf337904ee016664f03fca` | 1254 x 1254 RGBA, the cut-out master | `jade_tassel_cleanup.py` |
| work/jade_tassel_portrait.png | `18a3cf72284a573fc7202c8db52d2e82a745a1ffa8d47d92ebc715a77358156c` | 768 x 768 RGBA portrait crop, no circle | `jade_tassel_cleanup.py` |
| work/jade_tassel_base.png | `8ad381195139ed094c3fb97354978d6ad46d8c473dbd162cba761172963af9cd` | The portrait with the moving parts taken out (filled or cleared) | `jade_tassel_loop.py` |
| work/jade_tassel_head.png | `7f574c4c85e684626c94e7c33f9e5e52513eb59fd9602cc523450c3910254763` | Head and hair: the portrait weighted by the head field of the warp | `jade_tassel_loop.py` |
| work/jade_tassel_torso.png | `158aa1a8373adfacc498013e51152b8b79e43b2608afa09c37c3d1da852f63e6` | Torso and costume: the portrait weighted by one minus the head field | `jade_tassel_loop.py` |
| work/jade_tassel_eyes_open.png | `fe6227b29ff59d1e3161faa330e9493c7a87342f0f17bfafa79099efb769b58e` | Eye region, open (canonical concept) through the feathered eye mask | `jade_tassel_loop.py` |
| work/jade_tassel_eyes_closed.png | `85554fe82ebaf7938dbcb37ccb6728401586f44f1ef739552c59861006faf789` | Eye region, closed (aligned edit) through the same mask | `jade_tassel_loop.py` |
| work/jade_tassel_tassel_left.png | `b1d3046b69de862858f1be51ab600fd7d0fc405c67d09746342d9d0b7250a172` | Moving part `tassel_left`: the portrait through its feathered polygon | `jade_tassel_loop.py` |
| work/jade_tassel_tassel_right.png | `5887991ea31ad6c09fbbc611ee5f929d180d37cb20ce3498ed91c6fcf284c1ea` | Moving part `tassel_right`: the portrait through its feathered polygon | `jade_tassel_loop.py` |
| export/static.png | `da7035beb238a621090b83f070da2602d911ceb887cffebd99c919c0b638084a` | 192 x 192 RGBA static, loop frame 9 | `jade_tassel_loop.py` |
| export/anim/0001.png | `969dbc6be0e34ca7f0c7a15f84c811a7d7ab5343b26b14267ce64575d992a133` | 192 x 192 RGBA loop frame 1 | `jade_tassel_loop.py` |
| export/anim/0002.png | `38c7b52974f057a0eeba55336eacc5983adb58f3c053f758681588e304eef464` | 192 x 192 RGBA loop frame 2 | `jade_tassel_loop.py` |
| export/anim/0003.png | `7ed2db2542a27f5227c74e57f08610be856f1e3491959003c04fceb640db5d40` | 192 x 192 RGBA loop frame 3 | `jade_tassel_loop.py` |
| export/anim/0004.png | `460689386b3e0ba2b2e6703c06be420d832858c1903104f49e5cee64cf2add7d` | 192 x 192 RGBA loop frame 4 | `jade_tassel_loop.py` |
| export/anim/0005.png | `dfa5bef1fad04b60335400c601b5ab7e908f574b682f17d079fa120389a7dd63` | 192 x 192 RGBA loop frame 5 | `jade_tassel_loop.py` |
| export/anim/0006.png | `a504a385c08fa5b5e4a3f34be7a74d1ef847a54353de5a920ace0bc0b7e9eebe` | 192 x 192 RGBA loop frame 6 | `jade_tassel_loop.py` |
| export/anim/0007.png | `7e582a9d02f340f250622d8ab508ca8e788ec51cd9df25665c056e50f3825f95` | 192 x 192 RGBA loop frame 7 | `jade_tassel_loop.py` |
| export/anim/0008.png | `ce8f213abba7998a936111b1327d829f584c4a417b561761e55585aeecf9b8cc` | 192 x 192 RGBA loop frame 8 | `jade_tassel_loop.py` |
| export/anim/0009.png | `da7035beb238a621090b83f070da2602d911ceb887cffebd99c919c0b638084a` | 192 x 192 RGBA loop frame 9 | `jade_tassel_loop.py` |
| export/anim/0010.png | `4c6e0bba0ceb10781bb39708e9a11145c3f8f96b7c2d748492f9c402d4c82eac` | 192 x 192 RGBA loop frame 10 | `jade_tassel_loop.py` |
| export/anim/0011.png | `862cfbfac6490813e0c0b6c082e17224fe7140f388fedd196be1af944d7fb68a` | 192 x 192 RGBA loop frame 11 | `jade_tassel_loop.py` |
| export/anim/0012.png | `ba2aa5e68793cd9d90e8a3505cf21f705fb5dcbb5041e3ad70631115864411b8` | 192 x 192 RGBA loop frame 12 | `jade_tassel_loop.py` |
| export/anim/0013.png | `d83cc0e806e2c7c789082d9bda5ed913427e1513bc6b9b796770a7596c48d006` | 192 x 192 RGBA loop frame 13 | `jade_tassel_loop.py` |
| export/anim/0014.png | `4cd4c5c7acc548c34f2acdccde6484c2c35d2d56681e1535d73d50b94cb56a14` | 192 x 192 RGBA loop frame 14 | `jade_tassel_loop.py` |
| export/anim/0015.png | `cf8726cdc6c570dd6b30acc553d2dddb77ceaae3ef5f1711bbdba65b87e1902c` | 192 x 192 RGBA loop frame 15 | `jade_tassel_loop.py` |
| export/anim/0016.png | `243b96e986543077dbe89adf9cd63ee5b09a720e485f96a8dc433c30318b129b` | 192 x 192 RGBA loop frame 16 | `jade_tassel_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-16 | concept/20261116-01.png | P-3 with the style key, quoted below. Sent 13:22:53 UTC; `c2pa.created` 2026-11-16T13:24:01Z |
| 2026-11-16 | concept/20261116-02.png | Edit of `concept/20261116-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 13:25:00 UTC; `c2pa.created` 2026-11-16T13:26:08Z |
| 2026-11-16 | work/, export/ | The scripts above |

Prompt of `concept/20261116-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of an adult woman in her early thirties with light golden skin, black hair in two round buns tied with jade green silk ribbons that end in small gold tassels hanging beside her cheeks, and a quilted cream silk jacket with jade toggle buttons, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it, with standard input closed (`< /dev/null`), inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261116-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261116-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of an adult woman in her early thirties with light golden skin, black hair in two round buns tied with jade green silk ribbons that end in small gold tassels hanging beside her cheeks, and a quilted cream silk jacket with jade toggle buttons, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `--image 20261116-01.png --` and standard input closed, inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261116-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.

## Superseded, 2026-11-16

Owner decision on 2026-11-16: avatars show adults only. The first concept showed a girl of about eight; it was re-made as an adult woman with the same idea: the same costume colours (jade green ribbons, cream quilted silk jacket, jade toggles) and the same ambient part, the gold tassels at the ends of the hair ribbons. The slug is a stable id (`docs/asset-workflow.md` 4.1), so the item keeps `jade_tassel`; the new concept replaced the old files instead of a `_v2`, and git keeps the old ones (last at commit `4e040451`, together with their full provenance, the re-crop of #733 included).

- First canonical concept: `concept/20261115-01.png`, P-3 with the style key and `{identity}` = "a girl of about eight with light golden skin, black hair in two round buns tied with jade green silk ribbons that end in small gold tassels hanging beside her cheeks, and a quilted cream silk jacket with jade toggle buttons"; the closed-eyes edit of it was made with the same edit sentence as above
- Removed from `concept/` on 2026-11-16: `concept/20261115-01.png`, `concept/20261115-02.png`. Every `work/` and `export/` file built from them was re-made from the new concept; the layers `work/jade_tassel_earring_left.png` and `work/jade_tassel_earring_right.png`, left over from an earlier rig of the first concept and no longer written by its loop script, were removed

Their records, kept as they were:

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `b9e8a81c1128de5dd13807fb2e60bce4027947088d40ed4cb5b705b1d03933f0` | 1254 x 1254 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e00a-834f-7583-b533-bacacdf3e4ce` |
| concept/20261115-02.png | `3c5d5f94e72ffc59bcb71e7da94b58c8c17a96e06fe0ce1c6dea77a89b09fcfb` | 1254 x 1254 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e00c-c7b7-79b3-b05a-290ec17c6f69` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted in this file at commit `4e040451`. Sent 16:29:07 UTC; `c2pa.created` 2026-11-15T16:29:52Z |
| 2026-11-15 | concept/20261115-02.png | Edit of `concept/20261115-01.png` (attached): eyes closed, everything else identical, quoted in this file at commit `4e040451`. Sent 16:31:36 UTC; `c2pa.created` 2026-11-15T16:32:13Z |

| Field | `concept/20261115-01.png` | `concept/20261115-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T16:29:52Z | `c2pa.created`, 2026-11-15T16:32:13Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 16:29:07 | 2026-11-15 16:31:36 |
| Codex session | `01a1e00a-834f-7583-b533-bacacdf3e4ce` | `01a1e00c-c7b7-79b3-b05a-290ec17c6f69` |
