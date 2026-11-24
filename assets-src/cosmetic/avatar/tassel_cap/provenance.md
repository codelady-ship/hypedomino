# avatar.tassel_cap

Animated avatar of set 1 (#171), a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. The item was re-made on 2026-11-16 from a new concept by owner decision (avatars show adults only, and new headwear); the first concept and everything built from it are recorded under "Superseded, 2026-11-16" below. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-16 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15; `codex login status` on 2026-11-16 printed `Logged in using ChatGPT`)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-16 for the Plus plan and records the date here
- Canonical concept: concept/20261116-01.png
- Inputs uploaded to the generator: `concept/20261116-01.png`, our own canonical concept of this item, attached with `codex exec --image` to the one edit that made `concept/20261116-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "an adult man in his thirties with warm tan skin, short wavy black hair and a neatly trimmed short beard, a soft burgundy velvet beret worn tilted to one side with a long gold silk tassel cord hanging from its edge down beside his ear, an ivory shirt and a mustard yellow silk waistcoat with gold piping"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt: a soft burgundy velvet beret tilted to his right (the viewer's left) with a gold-embroidered band; the gold tassel hangs on a twisted cord from a ruby and crystal ornament on the beret beside his ear. Not asked for: a burgundy velvet over-waistcoat with gold embroidery and gold buttons and cords on the shoulders, and a ruby and crystal pin on a ruffled ivory cravat. The beret is a soft, wide crown with a band, not a small round brimless cap; the agent saw no cultural or religious meaning in it, left to the cultural review.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of -0.14, -0.23 px, scale 1.0001 and rotation 0.0002 rad (mean squared grey difference outside the eyes 0.00123 before, 0.00117 after), and the eye patch was offset by [-9.3, -4.0, -1.1] in RGB to meet the canonical skin. The edit also redrew the folds of the beret and the cravat a little; those changes are outside the eye mask and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: the person is an adult (a man in his thirties with a short beard); no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, source `shop:offer.avatar_tassel_cap`. The offer id is a placeholder until the shop record exists (V-15). Animated avatars are proposed at `epic` and above, as the batch leans to the higher tiers of `docs/rarity.md`; this batch proposes 5 epic, 4 legendary, 1 mythic.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 1 animated detail (tassel), so `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-16; the signatures were not validated with a C2PA tool.

| Field | `concept/20261116-01.png` | `concept/20261116-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-16T13:32:09Z | `c2pa.created`, 2026-11-16T13:33:43Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-16 13:31:15 | 2026-11-16 13:32:58 |
| Codex session | `01a1e48e-06b9-7760-8781-3e986116cf55` | `01a1e48f-9968-79c3-96f3-193043554e43` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). No generation for this item was rejected: these are the only two images generated for the new concept.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/tassel_cap/work/tassel_cap_cleanup.py tassel_cap assets-src/cosmetic/avatar/tassel_cap/concept/20261116-01.png assets-src/cosmetic/avatar/tassel_cap/work
```

It printed `tassel_cap: background [193.0, 192.0, 192.0], pockets 13408 px, crop (0, -192, 1172, 980), portrait 768`. The script is `cornflower_ribbon_cleanup.py` after the re-crop of 2026-11-16 (#733) unchanged: `CROP_SHARE` 0.935, `HEADROOM` 0.17 and the crop may reach above the top of the concept (`upper = min(top - HEADROOM * side, height - side)`), so the portrait has the same size and head height in the circle as the other re-cropped avatars. It writes the master and a square 768 x 768 portrait crop (`work/tassel_cap_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

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
python -I assets-src/cosmetic/avatar/tassel_cap/work/tassel_cap_loop.py assets-src/cosmetic/avatar/tassel_cap/concept assets-src/cosmetic/avatar/tassel_cap/work assets-src/cosmetic/avatar/tassel_cap/export
```

It printed `tassel_cap_loop: blink edit registered (tx, ty, scale, angle) = (-0.1398, -0.2252, 1.0001, 0.0002), residual 0.00123 -> 0.00117, eye colour offset [-9.3, -4.0, -1.1], parts 1, ripples 0, filled 11309 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha.

The script is `cornflower_ribbon_loop.py` after the re-crop (#733) with this item's file names and geometry (`CANONICAL`, `BLINK_EDIT`, `CROP`, `HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `EYES`, `PARTS`). The geometry was read by the agent from the 768 portrait on a coordinate grid; the eye ellipses from the difference between the canonical and the edit.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 4.6 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `tassel` (layer `work/tassel_cap_tassel.png`, pivot (242, 356)) turns about its pivot by 4.0 degrees x (sin(lag) - sin(t + lag)), lag 0.4, t = 2 pi (f - 8) / 16, the turn growing from 0 at the pivot to full at 170 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | Cord, knot and fringe below the ornament on the beret; the cut is at the ornament, where the turn is 0. The cord lies over the beret and hair, the fringe over the background |
| Inside the circle | `tassel` 0.00% outside at rest and in every frame, farthest pixel at 0.727 of the radius | Measured by the agent on 2026-11-16: the share of each part layer's alpha that lies outside the circle of radius 384 at 768, at rest and in every one of the 16 frames after its swing |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 2.96 of 255, against 2.66 on average and 3.54 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Head top | First row of `export/static.png` with alpha above 0.5 at 0.172 of the canvas height, highest over the 16 frames 0.167 | The same as the re-cropped avatars of #733 |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Determinism: two more runs of both commands into two empty folders wrote the same 25 files byte for byte, and both match the files committed here.

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261116-01.png | `5841c09717b50b0b6bff648831438adfa7e4665f1cb731c5d7d874327e6c6865` | 1254 x 1254 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e48e-06b9-7760-8781-3e986116cf55` |
| concept/20261116-02.png | `3e15ddcb258b17f1bccf747bf3a68745e186e92474931370d982ba76db470d2b` | 1254 x 1254 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e48f-9968-79c3-96f3-193043554e43` |
| work/tassel_cap_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` | W-3 script | `cornflower_ribbon_cleanup.py` after #733, unchanged |
| work/tassel_cap_loop.py | `c6785f4d9ed99516d4711b5dd34db329eae9183f9d4ac0ae30bc4fc60b2ecbad` | W-4 and W-5 script | Written by Claude Code, 2026-11-16, from `cornflower_ribbon_loop.py` after #733 with this item's geometry |
| work/tassel_cap_master.png | `60273847cd28f198b9c05b47a1d69458c2de18a2f34975be04e55294943e50cc` | 1254 x 1254 RGBA, the cut-out master | `tassel_cap_cleanup.py` |
| work/tassel_cap_portrait.png | `f83d17b21aca0a885781aacf5fe803fccd66545c97f31a3ed166a7931870f4e5` | 768 x 768 RGBA portrait crop, no circle | `tassel_cap_cleanup.py` |
| work/tassel_cap_base.png | `0744ce166b22da02f9eb53f0f7010ec1706bf3fcc512dd102cc3b0d329a60d00` | The portrait with the moving parts taken out (filled or cleared) | `tassel_cap_loop.py` |
| work/tassel_cap_head.png | `cb35581594d539c783ee9839dff84842883393389b0357b508f1a7afa297a6b6` | Head and hair: the portrait weighted by the head field of the warp | `tassel_cap_loop.py` |
| work/tassel_cap_torso.png | `2bb56ac82c5f8d6a841960615aa2a18039c54b709853cc73a45041db568145f7` | Torso and costume: the portrait weighted by one minus the head field | `tassel_cap_loop.py` |
| work/tassel_cap_eyes_open.png | `3f37a0159ca4fea4685578d0dda253ea6c88ce230bf80b975c6c3351338844e0` | Eye region, open (canonical concept) through the feathered eye mask | `tassel_cap_loop.py` |
| work/tassel_cap_eyes_closed.png | `43387ea0ced47b7bc6676d5a097186f19d181ec302e605265fa9886c819420c4` | Eye region, closed (aligned edit) through the same mask | `tassel_cap_loop.py` |
| work/tassel_cap_tassel.png | `026b7a81dd9b7e194659ede6592d524360ef76432852a46236e8d301ef285539` | Moving part `tassel`: the portrait through its feathered polygon | `tassel_cap_loop.py` |
| export/static.png | `12b2f26ebabec75404c465a3ccce3678f8ac5f89bd89cfe0dac82cc277b42c67` | 192 x 192 RGBA static, loop frame 9 | `tassel_cap_loop.py` |
| export/anim/0001.png | `eb99b1bd922d0982595bcc2816f770e4bb28de4e02406c0d2124d5742dccc669` | 192 x 192 RGBA loop frame 1 | `tassel_cap_loop.py` |
| export/anim/0002.png | `1b1f89b1f053ce33cea81e9cb7a2fa4798081449e16c7efe53824be93b78fc99` | 192 x 192 RGBA loop frame 2 | `tassel_cap_loop.py` |
| export/anim/0003.png | `c856dbb78144d68cd024e3cf3d789e10150ebcf81b574740b6601e2b5d9f35ac` | 192 x 192 RGBA loop frame 3 | `tassel_cap_loop.py` |
| export/anim/0004.png | `c72f8937bb99f913337bcbed4540c19b7f48f8b560ae0f06782e37c0b931ee8f` | 192 x 192 RGBA loop frame 4 | `tassel_cap_loop.py` |
| export/anim/0005.png | `b648f7e5f2fe54b517a2f46d82509543b495bf9449a0ab25ee149df8ce7a8b33` | 192 x 192 RGBA loop frame 5 | `tassel_cap_loop.py` |
| export/anim/0006.png | `f5fda821c33ee10023304643bffb6b2bc2a00cbe2f912b93990d5b8b0dcd08dc` | 192 x 192 RGBA loop frame 6 | `tassel_cap_loop.py` |
| export/anim/0007.png | `d08bf8732b2c12fa7e81ed4d6859bdf0e84f796c0f2451ef5fabfe2f37f1b890` | 192 x 192 RGBA loop frame 7 | `tassel_cap_loop.py` |
| export/anim/0008.png | `693baa2a590ce28885afe4a377d62d7e0e7a0d6b0b1c10a9bdace04e075983dc` | 192 x 192 RGBA loop frame 8 | `tassel_cap_loop.py` |
| export/anim/0009.png | `12b2f26ebabec75404c465a3ccce3678f8ac5f89bd89cfe0dac82cc277b42c67` | 192 x 192 RGBA loop frame 9 | `tassel_cap_loop.py` |
| export/anim/0010.png | `43c2d5dcd4a88cff6fc403634f06fdde9d4a0f47c057c3ceb0556d399c3a40f4` | 192 x 192 RGBA loop frame 10 | `tassel_cap_loop.py` |
| export/anim/0011.png | `c6b8952bafb70e492802fea4d18bc3c0cc47151a902acd2fc7983f6d82bb6ddd` | 192 x 192 RGBA loop frame 11 | `tassel_cap_loop.py` |
| export/anim/0012.png | `5884db7870328f9fc1a7506e8540c4cc273fc9c9d14e42dab464192ac482aadc` | 192 x 192 RGBA loop frame 12 | `tassel_cap_loop.py` |
| export/anim/0013.png | `bd51c6a13e150dab48c63f275f45958381cf440a28d011322dfc2157de068e1c` | 192 x 192 RGBA loop frame 13 | `tassel_cap_loop.py` |
| export/anim/0014.png | `113f43c6fddc8fbb4e2a7d90aa797ad5e61644428711db748e9485a02a061fbb` | 192 x 192 RGBA loop frame 14 | `tassel_cap_loop.py` |
| export/anim/0015.png | `a53a02c9cae96bf54c618c3503c7f993493fcb827c63774ad0458073de40c935` | 192 x 192 RGBA loop frame 15 | `tassel_cap_loop.py` |
| export/anim/0016.png | `35a3c76e0b2fbce5916de86d6ff1e92a4bdb5499879a9ecac0a24c77d4d77152` | 192 x 192 RGBA loop frame 16 | `tassel_cap_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-16 | concept/20261116-01.png | P-3 with the style key, quoted below. Sent 13:31:15 UTC; `c2pa.created` 2026-11-16T13:32:09Z |
| 2026-11-16 | concept/20261116-02.png | Edit of `concept/20261116-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 13:32:58 UTC; `c2pa.created` 2026-11-16T13:33:43Z |
| 2026-11-16 | work/, export/ | The scripts above |

Prompt of `concept/20261116-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of an adult man in his thirties with warm tan skin, short wavy black hair and a neatly trimmed short beard, a soft burgundy velvet beret worn tilted to one side with a long gold silk tassel cord hanging from its edge down beside his ear, an ivory shirt and a mustard yellow silk waistcoat with gold piping, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it, with standard input closed (`< /dev/null`), inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261116-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261116-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of an adult man in his thirties with warm tan skin, short wavy black hair and a neatly trimmed short beard, a soft burgundy velvet beret worn tilted to one side with a long gold silk tassel cord hanging from its edge down beside his ear, an ivory shirt and a mustard yellow silk waistcoat with gold piping, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `--image 20261116-01.png --` and standard input closed, inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261116-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.

## Superseded, 2026-11-16

Owner decision on 2026-11-16: avatars show adults only, and the round brimless cap with a tassel read like a traditional cap with cultural or religious meaning in some markets. The first concept showed a boy of about fourteen in a small round burgundy felt cap; it was re-made as an adult man in a soft burgundy velvet beret, with the same costume colours (burgundy headwear, ivory shirt, mustard yellow silk waistcoat with gold piping) and the same ambient part, a long gold silk tassel swinging beside his ear. The slug is a stable id (`docs/asset-workflow.md` 4.1), so the item keeps `tassel_cap`; the new concept replaced the old files instead of a `_v2`, and git keeps the old ones (last at commit `4e040451`, together with their full provenance, the re-crop of #733 included).

- First canonical concept: `concept/20261115-01.png`, P-3 with the style key and `{identity}` = "a boy of about fourteen with warm tan skin and short wavy black hair, a small round burgundy felt cap with a long gold silk tassel hanging down beside his ear, an ivory shirt and a mustard yellow silk waistcoat with gold piping"; the closed-eyes edit of it was made with the same edit sentence as above
- Removed from `concept/` on 2026-11-16: `concept/20261115-01.png`, `concept/20261115-02.png`. Every `work/` and `export/` file built from them was re-made from the new concept; no other file was left over

Their records, kept as they were:

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `98943ffa99a7af7f8bd418e7a24718670d73cb85d7880b66cd087d7f08632540` | 1254 x 1254 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e016-fac0-7992-9360-e593494fb21e` |
| concept/20261115-02.png | `2b04fe7784bc1c7e14cfaef6aa619b26da5f03fee6cbf6ad866bdd89fd8a527e` | 1254 x 1254 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e017-f7d3-76c3-b8cf-723a74428734` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted in this file at commit `4e040451`. Sent 16:42:44 UTC; `c2pa.created` 2026-11-15T16:43:27Z |
| 2026-11-15 | concept/20261115-02.png | Edit of `concept/20261115-01.png` (attached): eyes closed, everything else identical, quoted in this file at commit `4e040451`. Sent 16:43:49 UTC; `c2pa.created` 2026-11-15T16:44:25Z |

| Field | `concept/20261115-01.png` | `concept/20261115-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T16:43:27Z | `c2pa.created`, 2026-11-15T16:44:25Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 16:42:44 | 2026-11-15 16:43:49 |
| Codex session | `01a1e016-fac0-7992-9360-e593494fb21e` | `01a1e017-f7d3-76c3-b8cf-723a74428734` |
