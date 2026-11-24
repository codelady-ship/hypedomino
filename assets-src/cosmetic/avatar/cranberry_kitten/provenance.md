# avatar.cranberry_kitten

Animated avatar of set 1 (#171), a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. The item was re-made on 2026-11-16 from a new concept by owner decision (the kitten inside the portrait circle); the first concept and everything built from it are recorded under "Superseded, 2026-11-16" below. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-16 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15; `codex login status` on 2026-11-16 printed `Logged in using ChatGPT`)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-16 for the Plus plan and records the date here
- Canonical concept: concept/20261116-03.png
- Inputs uploaded to the generator: `concept/20261116-03.png`, our own canonical concept of this item, attached with `codex exec --image` to the one edit that made `concept/20261116-04.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a woman in her eighties with deep brown skin, short white curls, small gold earrings, a cream and cranberry striped silk gown with a high lace collar, and a small ginger kitten sitting high on her shoulder close to her cheek with its tail curled around its front paws, the whole kitten fully visible and compact, nothing hanging down"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt: the ginger kitten sits high on her left shoulder (the viewer's right) against her cheek, with its tail curled round its front paws; nothing hangs down. The earrings are gold with a pearl. Not asked for: a ruby brooch with a crystal drop and a pearl strand on the lace jabot, pearl buttons and gold scroll ornaments on the bodice.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of -0.12, 0.10 px, scale 0.9997 and rotation 0.0001 rad (mean squared grey difference outside the eyes 0.00105 before, 0.00103 after), and the eye patch was offset by [-10.7, -3.1, 1.1] in RGB to meet the canonical skin. The edit also redrew the lace and the folds of the gown a little; those changes are outside the eye mask and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: the person is an adult (a woman in her eighties); no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `legendary`, source `shop:offer.avatar_cranberry_kitten`. The offer id is a placeholder until the shop record exists (V-15). A living companion moving on its own on two layers: the kitten turns its head and swishes its tail. Animated avatars stay at `epic` and above as in batches A and B; batch C proposes 5 epic, 3 legendary, 2 mythic, which brings the 30 animated avatars of set 1 to 15 epic, 11 legendary and 4 mythic.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 2 animated details (kitten head and tail (a creature counts 2)), so `legendary`, was `mythic`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-16; the signatures were not validated with a C2PA tool.

| Field | `concept/20261116-03.png` | `concept/20261116-04.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-16T13:35:22Z | `c2pa.created`, 2026-11-16T13:37:06Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-16 13:34:29 | 2026-11-16 13:35:59 |
| Codex session | `01a1e490-fe18-7832-8d63-ece683581b82` | `01a1e492-5db9-7cc3-b581-ea4451fb3bb3` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). No generation for this item was rejected: these are the only two images generated for the new concept.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/cranberry_kitten/work/cranberry_kitten_cleanup.py cranberry_kitten assets-src/cosmetic/avatar/cranberry_kitten/concept/20261116-03.png assets-src/cosmetic/avatar/cranberry_kitten/work
```

It printed `cranberry_kitten: background [206.0, 205.0, 205.0], pockets 1181 px, crop (54, -161, 1226, 1011), portrait 768`. The script is `cornflower_ribbon_cleanup.py` after the re-crop of 2026-11-16 (#733) unchanged: `CROP_SHARE` 0.935, `HEADROOM` 0.17 and the crop may reach above the top of the concept (`upper = min(top - HEADROOM * side, height - side)`), so the portrait has the same size and head height in the circle as the other re-cropped avatars. The first concept needed `CROP_SHARE` 1.0 and `HEADROOM` 0.12 and still cut the tail; this one uses the values of the other avatars. It writes the master and a square 768 x 768 portrait crop (`work/cranberry_kitten_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

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
python -I assets-src/cosmetic/avatar/cranberry_kitten/work/cranberry_kitten_loop.py assets-src/cosmetic/avatar/cranberry_kitten/concept assets-src/cosmetic/avatar/cranberry_kitten/work assets-src/cosmetic/avatar/cranberry_kitten/export
```

It printed `cranberry_kitten_loop: blink edit registered (tx, ty, scale, angle) = (-0.1226, 0.0975, 0.9997, 0.0001), residual 0.00105 -> 0.00103, eye colour offset [-10.7, -3.1, 1.1], parts 2, ripples 0, filled 20874 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha.

The script is `cornflower_ribbon_loop.py` after the re-crop (#733) with this item's file names and geometry (`CANONICAL`, `BLINK_EDIT`, `CROP`, `HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `EYES`, `PARTS`). The geometry was read by the agent from the 768 portrait on a coordinate grid; the eye ellipses from the difference between the canonical and the edit.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 4.6 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `kitten_head` (layer `work/cranberry_kitten_kitten_head.png`, pivot (545, 500)) turns about its pivot by 7.0 degrees x (sin(lag) - sin(t + lag) + 0.5 sin(2t)), lag 0.0, t = 2 pi (f - 8) / 16, the turn growing from 0 at the pivot to full at 48 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | The kitten's head with its ears; the turn grows from the kitten's neck, so the head tilts while the body stays on the shoulder. It lies over her hair and cheek and the background |
| Ambient part | `kitten_tail` (layer `work/cranberry_kitten_kitten_tail.png`, pivot (564, 621)) turns about its pivot by 6.0 degrees x (sin(lag) - sin(t + lag) + 0.4 sin(2t)), lag 1.0, t = 2 pi (f - 8) / 16, the turn growing from 0 at the pivot to full at 60 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | The curled end of the tail in front of the paws; the turn grows from the middle of the curl, so the tip flicks while the root stays at the body. It lies over the paws and the gown |
| Inside the circle | `kitten_head` 0.00% outside at rest and in every frame, farthest pixel at 0.617 of the radius; `kitten_tail` 0.00%, 0.812 (the first concept's tail: 26.2% outside, 1.107) | Measured by the agent on 2026-11-16: the share of each part layer's alpha that lies outside the circle of radius 384 at 768, at rest and in every one of the 16 frames after its swing |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 2.87 of 255, against 2.34 on average and 2.95 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Head top | First row of `export/static.png` with alpha above 0.5 at 0.172 of the canvas height, highest over the 16 frames 0.167 | The same as the re-cropped avatars of #733 |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Determinism: two more runs of both commands into two empty folders wrote the same 26 files byte for byte, and both match the files committed here.

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261116-03.png | `a32c2a4c1d85cac7ce9cd1c5e291ba9b67bd2f87419a9cc76f9a9e95d98932f4` | 1254 x 1254 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e490-fe18-7832-8d63-ece683581b82` |
| concept/20261116-04.png | `5f4ffc89776156be887f24d60219d905165561efdc1a72b3fbcfe64b43c17a3c` | 1254 x 1254 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e492-5db9-7cc3-b581-ea4451fb3bb3` |
| work/cranberry_kitten_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` | W-3 script | `cornflower_ribbon_cleanup.py` after #733, unchanged |
| work/cranberry_kitten_loop.py | `277720c6e96937c4865f17c009403c10980e0424a1b69843fa8795df843c011c` | W-4 and W-5 script | Written by Claude Code, 2026-11-16, from `cornflower_ribbon_loop.py` after #733 with this item's geometry |
| work/cranberry_kitten_master.png | `95280ccb04cb66f53b1a7d6a358c964f20c2349e2b7a77b0a4b55599e54612b1` | 1254 x 1254 RGBA, the cut-out master | `cranberry_kitten_cleanup.py` |
| work/cranberry_kitten_portrait.png | `f2fc5a47cd055acb66740d2a2482614bbf010d975252988d4e07d9543c5a30d3` | 768 x 768 RGBA portrait crop, no circle | `cranberry_kitten_cleanup.py` |
| work/cranberry_kitten_base.png | `047ea4d9d08b6db7191818265e844bd4733185fc360e99055796e8cbdbe54147` | The portrait with the moving parts taken out (filled or cleared) | `cranberry_kitten_loop.py` |
| work/cranberry_kitten_head.png | `a79831dd388aeb16659fb1dceb3a1f782210bba4457d8c9a69fb10593182368c` | Head and hair: the portrait weighted by the head field of the warp | `cranberry_kitten_loop.py` |
| work/cranberry_kitten_torso.png | `5c9d2bc7663a784ae23fd6f8a0e1f21d51c0dcbb25e89cf3d389844f08b81677` | Torso and costume: the portrait weighted by one minus the head field | `cranberry_kitten_loop.py` |
| work/cranberry_kitten_eyes_open.png | `eafafb7d29a83c438c0410cb8e7a6ee16076440d44216841e7a162a3b124b5f1` | Eye region, open (canonical concept) through the feathered eye mask | `cranberry_kitten_loop.py` |
| work/cranberry_kitten_eyes_closed.png | `c363c2c63f05f9a2d9777b33ef441c84af3b421b7103222936198a273fec5b03` | Eye region, closed (aligned edit) through the same mask | `cranberry_kitten_loop.py` |
| work/cranberry_kitten_kitten_head.png | `aeadb0a2d0368563d6c7f8ae1d221380c9b9f543d90130b608e8451d43cd5272` | Moving part `kitten_head`: the portrait through its feathered polygon | `cranberry_kitten_loop.py` |
| work/cranberry_kitten_kitten_tail.png | `8a24913e9690abd4bc79ab05558c38ea19ac7f74f1c01ed83f94c50bffd61a9d` | Moving part `kitten_tail`: the portrait through its feathered polygon | `cranberry_kitten_loop.py` |
| export/static.png | `21c2c11d7992213f2e72a89707cc788847e06ea8b5c90e3428a42647133ceb7f` | 192 x 192 RGBA static, loop frame 9 | `cranberry_kitten_loop.py` |
| export/anim/0001.png | `342f4a24fac6b50c6ab0191aef50e6887513f48cd1fd10e38e756f172ac7c62e` | 192 x 192 RGBA loop frame 1 | `cranberry_kitten_loop.py` |
| export/anim/0002.png | `9f4609967033fff969bf6b58018df68e8a545a37a441003040626d4d4b10030d` | 192 x 192 RGBA loop frame 2 | `cranberry_kitten_loop.py` |
| export/anim/0003.png | `e3c379d0e8e84a1cc4fbcfeca2082437aff8b4d2579303e959c9cd7f586845d6` | 192 x 192 RGBA loop frame 3 | `cranberry_kitten_loop.py` |
| export/anim/0004.png | `f77345a7fbf7010c0d99de52f3e8d060f7fe1690974116d71df002e1528f46a1` | 192 x 192 RGBA loop frame 4 | `cranberry_kitten_loop.py` |
| export/anim/0005.png | `378d1c8b02c34206fc883d82a25b9de9c3151168f96497281b80932fadd4a429` | 192 x 192 RGBA loop frame 5 | `cranberry_kitten_loop.py` |
| export/anim/0006.png | `70554a93bf9a85202f1317f403bd7c0f1a82cc1f188ca08813b201ee0e285fbf` | 192 x 192 RGBA loop frame 6 | `cranberry_kitten_loop.py` |
| export/anim/0007.png | `0214ab7a080d65567564f1ab113316d0e282185cbe6b15d24c24e30c4747f169` | 192 x 192 RGBA loop frame 7 | `cranberry_kitten_loop.py` |
| export/anim/0008.png | `66e8daa019f05b260e0cd6db7ce41f3be492a3663510e43a9b215255fe098917` | 192 x 192 RGBA loop frame 8 | `cranberry_kitten_loop.py` |
| export/anim/0009.png | `21c2c11d7992213f2e72a89707cc788847e06ea8b5c90e3428a42647133ceb7f` | 192 x 192 RGBA loop frame 9 | `cranberry_kitten_loop.py` |
| export/anim/0010.png | `3699a5dc056501bebb86dd15d310280c580135c26d86251d4149a1a892d51e6d` | 192 x 192 RGBA loop frame 10 | `cranberry_kitten_loop.py` |
| export/anim/0011.png | `b83f90eb600d687296b7cdab2ce880638a4e54c5c4e08a54b622244d0c7f67cb` | 192 x 192 RGBA loop frame 11 | `cranberry_kitten_loop.py` |
| export/anim/0012.png | `ee247d8e1ff3d395e884097e032d1d10e78987eb4080961c6def588f9f7cce8c` | 192 x 192 RGBA loop frame 12 | `cranberry_kitten_loop.py` |
| export/anim/0013.png | `a04c247bf65206cff81b89caebfd2da2e23cfb04f26d379395012c194fad4d2e` | 192 x 192 RGBA loop frame 13 | `cranberry_kitten_loop.py` |
| export/anim/0014.png | `652e653368b5e7601c4f362012803e1a31e63a141d9602a3778ce72c28bdb364` | 192 x 192 RGBA loop frame 14 | `cranberry_kitten_loop.py` |
| export/anim/0015.png | `76346f1a6bf06e57d069579c4a2c5ce38a6b4683f54fcf3d7e7d3d6428a29bcc` | 192 x 192 RGBA loop frame 15 | `cranberry_kitten_loop.py` |
| export/anim/0016.png | `56811c2bd3ec3008a6dfdad723eafaf20ed51338ac6f01e1a5b9082de88527b2` | 192 x 192 RGBA loop frame 16 | `cranberry_kitten_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-16 | concept/20261116-03.png | P-3 with the style key, quoted below. Sent 13:34:29 UTC; `c2pa.created` 2026-11-16T13:35:22Z |
| 2026-11-16 | concept/20261116-04.png | Edit of `concept/20261116-03.png` (attached): eyes closed, everything else identical, quoted below. Sent 13:35:59 UTC; `c2pa.created` 2026-11-16T13:37:06Z |
| 2026-11-16 | work/, export/ | The scripts above |

Prompt of `concept/20261116-03.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her eighties with deep brown skin, short white curls, small gold earrings, a cream and cranberry striped silk gown with a high lace collar, and a small ginger kitten sitting high on her shoulder close to her cheek with its tail curled around its front paws, the whole kitten fully visible and compact, nothing hanging down, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it, with standard input closed (`< /dev/null`), inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261116-03.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261116-04.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her eighties with deep brown skin, short white curls, small gold earrings, a cream and cranberry striped silk gown with a high lace collar, and a small ginger kitten sitting high on her shoulder close to her cheek with its tail curled around its front paws, the whole kitten fully visible and compact, nothing hanging down, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `--image 20261116-03.png --` and standard input closed, inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261116-04.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.

## Superseded, 2026-11-16

Owner decision on 2026-11-16: a new concept with the kitten fully inside the portrait circle. In the first concept the kitten sat at the bottom right of the square with its tail hanging down, and at the framing of #733 the tail layer was 26.2% outside the circle. The new concept keeps the woman, the costume colours (cream and cranberry striped silk gown, high lace collar) and the living companion that moves on its own, but the kitten sits high on her shoulder against her cheek with its tail curled round its front paws. The slug is a stable id (`docs/asset-workflow.md` 4.1), so the item keeps `cranberry_kitten`; the new concept replaced the old files instead of a `_v2`, and git keeps the old ones (last at commit `4e040451`, together with their full provenance, the re-crop of #733 included).

- First canonical concept: `concept/20261116-01.png`, P-3 with the style key and `{identity}` = "a woman in her eighties with deep brown skin, short white curls, small gold earrings, a cream and cranberry striped silk gown with a high lace collar, and a small ginger kitten sitting on her shoulder close to her neck with its tail hanging down, the whole kitten fully visible"; the closed-eyes edit of it was made with the same edit sentence as above
- Removed from `concept/` on 2026-11-16: `concept/20261116-01.png`, `concept/20261116-02.png`. Every `work/` and `export/` file built from them was re-made from the new concept; no other file was left over

Their records, kept as they were:

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261116-01.png | `a8d72b23c50a71574b3ba7cd12f62d5b5c10eb064182526466f03331c7e6c547` | 1254 x 1254 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e376-3c51-7502-ac0e-3f521c9b6ac6` |
| concept/20261116-02.png | `7d7c47ff225abf1ddaf158162a5cbb258964dbf351150bda15abd520debd69ec` | 1254 x 1254 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e385-704e-7fc3-a5f2-4b3457a87dbb` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-16 | concept/20261116-01.png | P-3 with the style key, quoted in this file at commit `4e040451`. Sent 08:25:38 UTC; `c2pa.created` 2026-11-16T08:26:33Z |
| 2026-11-16 | concept/20261116-02.png | Edit of `concept/20261116-01.png` (attached): eyes closed, everything else identical, quoted in this file at commit `4e040451`. Sent 08:42:15 UTC; `c2pa.created` 2026-11-16T08:43:00Z |

| Field | `concept/20261116-01.png` | `concept/20261116-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-16T08:26:33Z | `c2pa.created`, 2026-11-16T08:43:00Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-16 08:25:38 | 2026-11-16 08:42:15 |
| Codex session | `01a1e376-3c51-7502-ac0e-3f521c9b6ac6` | `01a1e385-704e-7fc3-a5f2-4b3457a87dbb` |
