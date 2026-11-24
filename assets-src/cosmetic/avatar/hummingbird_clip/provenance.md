# avatar.hummingbird_clip

Animated avatar of set 1 (#171), a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. The item was re-made on 2026-11-16 from a new concept by owner decision (avatars show adults only); the first concept and everything built from it are recorded under "Superseded, 2026-11-16" below. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-16 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15; `codex login status` on 2026-11-16 printed `Logged in using ChatGPT`)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-16 for the Plus plan and records the date here
- Canonical concept: concept/20261116-01.png
- Inputs uploaded to the generator: `concept/20261116-01.png`, our own canonical concept of this item, attached with `codex exec --image` to the one edit that made `concept/20261116-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "an adult woman in her late twenties with deep brown skin, a voluminous round afro held back by a wide ivory silk headband with a small enamel hummingbird clip, small crystal drop earrings, and a high-collared sage green silk blouse with pearl buttons"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt. Not asked for: freckles, pearl studs above the crystal drops, an emerald and pearl brooch on the collar, gold embroidery, cream floral panels and puffed sleeves on the bodice. The enamel hummingbird has a red throat and green wings; no colour of it dominates.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of -0.15, -0.17 px, scale 1.0002 and rotation 0.0001 rad (mean squared grey difference outside the eyes 0.00102 before, 0.00099 after), and the eye patch was offset by [-7.9, -1.4, 1.0] in RGB to meet the canonical skin. The edit also redrew the folds of the bodice below the collar and the hummingbird clip a little; those changes are outside the eye mask and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: the person is an adult (a woman in her late twenties); no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, source `shop:offer.avatar_hummingbird_clip`. The offer id is a placeholder until the shop record exists (V-15). Animated avatars are proposed at `epic` and above, as the batch leans to the higher tiers of `docs/rarity.md`; this batch proposes 5 epic, 4 legendary, 1 mythic.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 1 animated detail (earrings (pair)), so `epic`, was `legendary`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-16; the signatures were not validated with a C2PA tool.

| Field | `concept/20261116-01.png` | `concept/20261116-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-16T13:28:00Z | `c2pa.created`, 2026-11-16T13:30:33Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-16 13:26:43 | 2026-11-16 13:29:32 |
| Codex session | `01a1e489-e079-7cc2-9c7b-c4f8a70231f7` | `01a1e48c-7728-7503-9ef3-8c2c782751de` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). No generation for this item was rejected: these are the only two images generated for the new concept.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/hummingbird_clip/work/hummingbird_clip_cleanup.py hummingbird_clip assets-src/cosmetic/avatar/hummingbird_clip/concept/20261116-01.png assets-src/cosmetic/avatar/hummingbird_clip/work
```

It printed `hummingbird_clip: background [199.0, 199.0, 198.0], pockets 1020 px, crop (63, -186, 1184, 935), portrait 768`. The script is `cornflower_ribbon_cleanup.py` after the re-crop of 2026-11-16 (#733) unchanged: `CROP_SHARE` 0.935, `HEADROOM` 0.17 and the crop may reach above the top of the concept (`upper = min(top - HEADROOM * side, height - side)`), so the portrait has the same size and head height in the circle as the other re-cropped avatars. It writes the master and a square 768 x 768 portrait crop (`work/hummingbird_clip_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

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
python -I assets-src/cosmetic/avatar/hummingbird_clip/work/hummingbird_clip_loop.py assets-src/cosmetic/avatar/hummingbird_clip/concept assets-src/cosmetic/avatar/hummingbird_clip/work assets-src/cosmetic/avatar/hummingbird_clip/export
```

It printed `hummingbird_clip_loop: blink edit registered (tx, ty, scale, angle) = (-0.1535, -0.1680, 1.0002, 0.0001), residual 0.00102 -> 0.00099, eye colour offset [-7.9, -1.4, 1.0], parts 2, ripples 0, filled 4160 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha.

The script is `cornflower_ribbon_loop.py` after the re-crop (#733) with this item's file names and geometry (`CANONICAL`, `BLINK_EDIT`, `CROP`, `HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `EYES`, `PARTS` and `TILT` 1.4 as in the first rig, because the wide hair moves more at its edge). The geometry was read by the agent from the 768 portrait on a coordinate grid; the eye ellipses from the difference between the canonical and the edit.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 4.6 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.4 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `earring_left` (layer `work/hummingbird_clip_earring_left.png`, pivot (280, 544)) turns about its pivot by 6.0 degrees x (sin(lag) - sin(t + lag)), lag 0.4, t = 2 pi (f - 8) / 16, the turn growing from 0 at the pivot to full at 46 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | The crystal drop below the pearl stud on her right ear (the viewer's left); it hangs over hair, the stud stays still |
| Ambient part | `earring_right` (layer `work/hummingbird_clip_earring_right.png`, pivot (537, 519)) turns about its pivot by 6.0 degrees x (sin(lag) - sin(t + lag)), lag 0.8, t = 2 pi (f - 8) / 16, the turn growing from 0 at the pivot to full at 46 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | The same for the other ear, with a different lag |
| Inside the circle | `earring_left` 0.00% outside at rest and in every frame, farthest pixel at 0.629 of the radius; `earring_right` 0.00%, 0.651 | Measured by the agent on 2026-11-16: the share of each part layer's alpha that lies outside the circle of radius 384 at 768, at rest and in every one of the 16 frames after its swing |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 2.82 of 255, against 2.32 on average and 3.10 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Head top | First row of `export/static.png` with alpha above 0.5 at 0.172 of the canvas height, highest over the 16 frames 0.161 | The same as the re-cropped avatars of #733 |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Determinism: two more runs of both commands into two empty folders wrote the same 26 files byte for byte, and both match the files committed here.

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261116-01.png | `97af53afeb94921a1e3dba404c991822944a78668fc793ad3f8858e8a255d216` | 1312 x 1199 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e489-e079-7cc2-9c7b-c4f8a70231f7` |
| concept/20261116-02.png | `4aa58c4ff12e4e11f827c29393eb15f8c39cf5bb6ac08b3e19c96706f7b9c11e` | 1312 x 1199 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e48c-7728-7503-9ef3-8c2c782751de` |
| work/hummingbird_clip_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` | W-3 script | `cornflower_ribbon_cleanup.py` after #733, unchanged |
| work/hummingbird_clip_loop.py | `3e11903fde8b2b891f743560bd6aa940d24e27def20a540989ac8720fc5eafcd` | W-4 and W-5 script | Written by Claude Code, 2026-11-16, from `cornflower_ribbon_loop.py` after #733 with this item's geometry |
| work/hummingbird_clip_master.png | `bfc4e2cd19b23a1f958bca5d9dcccb0168cc5f0b7e3cd2cd80bd1711bea6fa6f` | 1312 x 1199 RGBA, the cut-out master | `hummingbird_clip_cleanup.py` |
| work/hummingbird_clip_portrait.png | `e8172cebf97fed7d3d430408fbf985e00489a1f99b6ddca9c4e58292751c1feb` | 768 x 768 RGBA portrait crop, no circle | `hummingbird_clip_cleanup.py` |
| work/hummingbird_clip_base.png | `68479e2251b4a458904d3bbe94128ff5e441f4cbde87f1d29e7173bb80a86f4d` | The portrait with the moving parts taken out (filled or cleared) | `hummingbird_clip_loop.py` |
| work/hummingbird_clip_head.png | `bf81d40e7714393e00b19bb49dcf43731e8ade48fdcb3ae3985d3c265583e08c` | Head and hair: the portrait weighted by the head field of the warp | `hummingbird_clip_loop.py` |
| work/hummingbird_clip_torso.png | `9835678486ffff8c36043106b3f85926fdd31fa1ebf2ed887d2d9e507521f2f3` | Torso and costume: the portrait weighted by one minus the head field | `hummingbird_clip_loop.py` |
| work/hummingbird_clip_eyes_open.png | `9053d7adb07a7c08d1ed7d28e5d8f3224f48b2c8b37d402320fe0003bd5c56df` | Eye region, open (canonical concept) through the feathered eye mask | `hummingbird_clip_loop.py` |
| work/hummingbird_clip_eyes_closed.png | `fa645ed794b076555e81254646b12c1706b8adb6ea103259a178e32506a1ae7c` | Eye region, closed (aligned edit) through the same mask | `hummingbird_clip_loop.py` |
| work/hummingbird_clip_earring_left.png | `24236b72cf9daa939a9a854620c5f18ef57b0995c3ee6cbf723b8da76f0c4756` | Moving part `earring_left`: the portrait through its feathered polygon | `hummingbird_clip_loop.py` |
| work/hummingbird_clip_earring_right.png | `a42541eb0ab5f22a7276dc48d4d5fb397e7da62c525ca1bb83c0a41017e69762` | Moving part `earring_right`: the portrait through its feathered polygon | `hummingbird_clip_loop.py` |
| export/static.png | `941de4a658cc2affe9ca6573ce5b8a6020a7f5304d1679593378e7b631a3dd8b` | 192 x 192 RGBA static, loop frame 9 | `hummingbird_clip_loop.py` |
| export/anim/0001.png | `75e0395b97c9c66a00569257e8821f745649d6919169d571ec78e8641514ee53` | 192 x 192 RGBA loop frame 1 | `hummingbird_clip_loop.py` |
| export/anim/0002.png | `931edfd3bc35ae56513c25338b552cd6bf8cb669fa573644fbd528e1d877af5b` | 192 x 192 RGBA loop frame 2 | `hummingbird_clip_loop.py` |
| export/anim/0003.png | `ae30d094de3e41831d7f81c5f210fde17185ee02744582dd208716e427ac5d6b` | 192 x 192 RGBA loop frame 3 | `hummingbird_clip_loop.py` |
| export/anim/0004.png | `bfcf4af39fdee1d720ca84ee9c76c2604f360e6be11d4b31e363fbc13c5ff2b0` | 192 x 192 RGBA loop frame 4 | `hummingbird_clip_loop.py` |
| export/anim/0005.png | `0e146a3fb26655676af71e8de64a11a0bf26fab45714727662af60334144040a` | 192 x 192 RGBA loop frame 5 | `hummingbird_clip_loop.py` |
| export/anim/0006.png | `2a679d6e6240f745b6a233deb44b552922a5e2dc9986388003b18591a6efb0ae` | 192 x 192 RGBA loop frame 6 | `hummingbird_clip_loop.py` |
| export/anim/0007.png | `162c76cda249b285fdc7087543a04df10fc8f0059ee7ec9703649ffac61f1203` | 192 x 192 RGBA loop frame 7 | `hummingbird_clip_loop.py` |
| export/anim/0008.png | `fc314b77b6e89950ad24bd9d99c84af6e43eba5059e4486f0bea0944f4bf7a82` | 192 x 192 RGBA loop frame 8 | `hummingbird_clip_loop.py` |
| export/anim/0009.png | `941de4a658cc2affe9ca6573ce5b8a6020a7f5304d1679593378e7b631a3dd8b` | 192 x 192 RGBA loop frame 9 | `hummingbird_clip_loop.py` |
| export/anim/0010.png | `196e0bead972703adbacafad089f0731c1a88af69156e24774849c44c867162b` | 192 x 192 RGBA loop frame 10 | `hummingbird_clip_loop.py` |
| export/anim/0011.png | `33042cc1d42b59ad771887a8e7db6b155639eba229ac0a8d458c0dbe69f5e2ac` | 192 x 192 RGBA loop frame 11 | `hummingbird_clip_loop.py` |
| export/anim/0012.png | `971bbc5b577e80d0f209946bfb519ec6ce3d5378120e6b8099f60dfebe112fd9` | 192 x 192 RGBA loop frame 12 | `hummingbird_clip_loop.py` |
| export/anim/0013.png | `b9c2c3927a03f997153d6ae2328a472e2e5688e6485fa3f4f33554a1720c9439` | 192 x 192 RGBA loop frame 13 | `hummingbird_clip_loop.py` |
| export/anim/0014.png | `16ac22c40b308dbbf6031dc48a76e2a6bb0852eeed80319e14afb21337f80cd0` | 192 x 192 RGBA loop frame 14 | `hummingbird_clip_loop.py` |
| export/anim/0015.png | `e8f200a9414f7950ad1e0107fef63fb9400cd392e5d9b1baac0bb6e1bf876184` | 192 x 192 RGBA loop frame 15 | `hummingbird_clip_loop.py` |
| export/anim/0016.png | `9ab2d5872b1339a8072f36b2d9158bbaa5068b1b5fbef4837f7887d06392f0f9` | 192 x 192 RGBA loop frame 16 | `hummingbird_clip_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-16 | concept/20261116-01.png | P-3 with the style key, quoted below. Sent 13:26:43 UTC; `c2pa.created` 2026-11-16T13:28:00Z |
| 2026-11-16 | concept/20261116-02.png | Edit of `concept/20261116-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 13:29:32 UTC; `c2pa.created` 2026-11-16T13:30:33Z |
| 2026-11-16 | work/, export/ | The scripts above |

Prompt of `concept/20261116-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of an adult woman in her late twenties with deep brown skin, a voluminous round afro held back by a wide ivory silk headband with a small enamel hummingbird clip, small crystal drop earrings, and a high-collared sage green silk blouse with pearl buttons, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it, with standard input closed (`< /dev/null`), inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261116-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261116-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of an adult woman in her late twenties with deep brown skin, a voluminous round afro held back by a wide ivory silk headband with a small enamel hummingbird clip, small crystal drop earrings, and a high-collared sage green silk blouse with pearl buttons, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `--image 20261116-01.png --` and standard input closed, inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261116-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.

## Superseded, 2026-11-16

Owner decision on 2026-11-16: avatars show adults only. The first concept showed a girl of about sixteen; it was re-made as an adult woman with the same idea: the same costume colours (ivory silk headband with the enamel hummingbird clip, sage green high-collared silk blouse with pearl buttons) and the same ambient part, the crystal drop earrings. The slug is a stable id (`docs/asset-workflow.md` 4.1), so the item keeps `hummingbird_clip`; the new concept replaced the old files instead of a `_v2`, and git keeps the old ones (last at commit `4e040451`, together with their full provenance, the re-crop of #733 included).

- First canonical concept: `concept/20261115-01.png`, P-3 with the style key and `{identity}` = "a girl of about sixteen with deep brown skin, a voluminous round afro held back by a wide ivory silk headband with a small enamel hummingbird clip, small crystal drop earrings, and a high-collared sage green silk blouse with pearl buttons"; the closed-eyes edit of it was made with the same edit sentence as above
- Removed from `concept/` on 2026-11-16: `concept/20261115-01.png`, `concept/20261115-02.png`. Every `work/` and `export/` file built from them was re-made from the new concept; no other file was left over

Their records, kept as they were:

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `f730ea001abfff7fac435d45efef15f28b2a7182d58d36def1b177b87ac79f0f` | 1312 x 1199 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e025-472a-7023-8a4b-ee9666b5b76b` |
| concept/20261115-02.png | `d0f5fb117b4976916b9522681b36887714f1c57f4614b2d9b0f86b91a7465d71` | 1312 x 1199 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e026-d07f-7f33-ab9a-db5c53ae6d3a` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted in this file at commit `4e040451`. Sent 16:58:21 UTC; `c2pa.created` 2026-11-15T16:59:29Z |
| 2026-11-15 | concept/20261115-02.png | Edit of `concept/20261115-01.png` (attached): eyes closed, everything else identical, quoted in this file at commit `4e040451`. Sent 17:00:02 UTC; `c2pa.created` 2026-11-15T17:01:15Z |

| Field | `concept/20261115-01.png` | `concept/20261115-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T16:59:29Z | `c2pa.created`, 2026-11-15T17:01:15Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 16:58:21 | 2026-11-15 17:00:02 |
| Codex session | `01a1e025-472a-7023-8a4b-ee9666b5b76b` | `01a1e026-d07f-7f33-ab9a-db5c53ae6d3a` |
