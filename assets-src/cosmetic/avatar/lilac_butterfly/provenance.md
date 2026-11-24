# avatar.lilac_butterfly

Animated avatar of set 1 (#171), batch B, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 and 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, final runs on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: `concept/20261115-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261115-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a woman in her mid-twenties with light skin and East Asian features, glossy black hair in a low side bun, a jewelled butterfly hair pin with delicate open wings of lilac crystal set above one ear, and a pale sea-glass green silk gown with a high lace collar and pearl buttons"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt. Not asked for: pearl and amethyst drop earrings, a sprig of pearl flowers with two hanging pearls under the butterfly, and pearl buttons with lace on the gown.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.23, -0.17 px, scale 0.9998 and rotation -0.0000 rad (mean squared grey difference outside the eyes 0.00147 before, 0.00142 after), and the eye patch was offset by [0.6, -1.5, -0.7] in RGB to meet the canonical skin. The changes outside the eye mask are small and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `mythic`, source `shop:offer.avatar_lilac_butterfly`. The offer id is a placeholder until the shop record exists (V-15). Four moving parts: the butterfly opens and closes its wings, and the hanging pearls of the pin and both earrings swing. Animated avatars stay at `epic` and above as in batch A; batch B proposes 5 epic, 4 legendary, 1 mythic, the same split as batch A.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 4 animated details (butterfly wings (a creature counts 2), pin pearls, earrings (pair)), so `mythic`, was `legendary`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signatures were not validated with a C2PA tool.

| Field | `concept/20261115-01.png` | `concept/20261115-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T17:27:54Z | `c2pa.created`, 2026-11-15T17:43:43Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 17:26:57 | 2026-11-15 17:43:05 |
| Codex session | `01a1e03f-763d-7c12-b88d-ab5392c7627e` | `01a1e04e-3b19-7f71-946a-2e74cf2a522e` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/lilac_butterfly/work/lilac_butterfly_cleanup.py lilac_butterfly assets-src/cosmetic/avatar/lilac_butterfly/concept/20261115-01.png assets-src/cosmetic/avatar/lilac_butterfly/work
```

It printed background [196.0, 195.0, 195.0], pockets 8390 px, patched 1407 px, crop (122, 0, 1200, 1078), portrait 768. The script is `ruby_drop_cleanup.py` of batch A with one change, a fix of an AI artefact: the concept has a small patch of flat background grey inside the gown below the left shoulder (the viewer's right), joined to the background by a hair-thin gap, so the clean-up cut it out as a hole. `patch_gaps` takes the background-grey pixels inside the box (850, 840) to (960, 950) of the concept that a closing of 6 px encloses in the subject (printed as patched px) and paints them from the surrounding fabric by normalised Gaussian blurs (sigma 24, 12, 6, 3) before the cut-out. This is a scripted repair, not a paint-over. It writes the master and a square 768 x 768 portrait crop (`work/lilac_butterfly_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | As in batch A. The loop script puts a circular alpha (soft edge of 2.25 px at 768, 0.56 px at 192) on every frame, because the portrait is shown in a circle (`docs/art-direction.md` 5.1); the class canvas is open (O-6) and D-1 sets 192 x 192 |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Yes | Written by the loop script into `work/` (table below); the ripples are warps of the base layer and need no layer of their own |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen in the concept, the master or the frames; the butterfly is jewellery, not an emblem |
| 5. Silhouette and smallest-size tests | In part | The agent looked at every frame at 192 px and enlarged on a dark backdrop; face, expression and the moving part read. The smallest display size and frame band of an avatar are open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-16 (UTC) by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/avatar/lilac_butterfly/work/lilac_butterfly_loop.py assets-src/cosmetic/avatar/lilac_butterfly/concept assets-src/cosmetic/avatar/lilac_butterfly/work assets-src/cosmetic/avatar/lilac_butterfly/export
```

It printed `lilac_butterfly_loop: blink edit registered (tx, ty, scale, angle) = (0.2339, -0.1716, 0.9998, -0.0000), residual 0.00147 -> 0.00142, eye colour offset [0.6, -1.5, -0.7], parts 4, ripples 0, filled 22212 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run into empty folders wrote files byte-identical to each other and to the committed ones.

The script is the batch A loop script (`ruby_drop_loop.py`) with this item's geometry and these additions, the same for every avatar of this batch: the concept file names are constants; a part can open and close (`flap`, a horizontal scale about its pivot); `RIPPLES` add a smooth sideways wave to the warp inside a feathered polygon, so cloth, ribbon and chain move without a cut-out layer and without seams; a hole behind a part is cleared where the fill estimate is less than half covered, which keeps a filled hole from leaving a smear over the background; a glint is clipped to the bright pixels of the portrait and its peak opacity is a constant (`GLINT_PEAK`, 0.55 as in batch A unless the item says otherwise); and the registration region is clipped to the image when the crop reaches past its edge.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `butterfly` (layer `work/lilac_butterfly_butterfly.png`, pivot (542, 186)) opens and closes about the vertical line through its pivot: its width is scaled by 1 - 0.28 x (1 - cos 2t) / 2, so it narrows to 72% twice per loop; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | A small part well inside the circle, so its motion never meets the circle edge |
| Ambient part | `pin_pearls` (layer `work/lilac_butterfly_pin_pearls.png`, pivot (561, 288)) turns about its pivot by 5.0 degrees x (sin(lag) - sin(t + lag)), lag 0.6, the turn growing from 0 at the pivot to full at 50 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | A small part well inside the circle, so its motion never meets the circle edge |
| Ambient part | `earring_left` (layer `work/lilac_butterfly_earring_left.png`, pivot (240, 380)) turns about its pivot by 5.0 degrees x (sin(lag) - sin(t + lag)), lag 0.4, the turn growing from 0 at the pivot to full at 70 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | A small part well inside the circle, so its motion never meets the circle edge |
| Ambient part | `earring_right` (layer `work/lilac_butterfly_earring_right.png`, pivot (503, 350)) turns about its pivot by 5.0 degrees x (sin(lag) - sin(t + lag)), lag 0.9, the turn growing from 0 at the pivot to full at 70 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | A small part well inside the circle, so its motion never meets the circle edge |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 2.96 of 255, against 2.83 on average and 3.36 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `lilac_butterfly: background [196.0, 195.0, 195.0], pockets 8390 px, patched 1407 px, crop (75, -182, 1247, 990), portrait 768`
- Crop `(122, 0, 1200, 1078)` to `(75, -182, 1247, 990)` on the concept
- Scale k = 0.9198. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- Printed by the loop: `lilac_butterfly_loop: blink edit registered (tx, ty, scale, angle) = (0.2699, 0.0895, 0.9998, 0.0001), residual 0.00130 -> 0.00126, eye colour offset [0.4, -1.7, -1.0], parts 4, ripples 0, filled 19414 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.016 before, 0.172 after; highest over the 16 loop frames 0.010 before, 0.161 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/lilac_butterfly/work/lilac_butterfly_cleanup.py lilac_butterfly assets-src/cosmetic/avatar/lilac_butterfly/concept/20261115-01.png assets-src/cosmetic/avatar/lilac_butterfly/work
python -I assets-src/cosmetic/avatar/lilac_butterfly/work/lilac_butterfly_loop.py assets-src/cosmetic/avatar/lilac_butterfly/concept assets-src/cosmetic/avatar/lilac_butterfly/work assets-src/cosmetic/avatar/lilac_butterfly/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 28 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/lilac_butterfly` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `8492b505764eb77a80e1f2aaaf208fb00a66ab025028d1a6041424012cb7fc82` |
| export/anim/0002.png | `80030e2560335d9389cbdfa21d1a83bc6d7e8f045c62ff03dbcf81bfc59db99c` |
| export/anim/0003.png | `d18ebff2b24686da0ca100bc6ad83de51d5e83ffb9d877362fa9430ac2136d1b` |
| export/anim/0004.png | `c26056a438781ff724a5d025892aa5d2bf84b36a2127741d4a386efbfa6201a5` |
| export/anim/0005.png | `f928ca1b4b7a12eee6a3b4b41c9b28f8c71ad9e4dd595178d021243424b0548a` |
| export/anim/0006.png | `cf8734cf6001559c698c3ffef6944537db25a8853efbfe8197534f25bedcfa34` |
| export/anim/0007.png | `47015302be3492b2d5352278b3403ed339f255821552b04dbf9df18d4f2ae47b` |
| export/anim/0008.png | `247273fda04be1ad1490e0b56ef384fedec6d8482eb8cacc1ceab814fb7a7a0a` |
| export/anim/0009.png | `5b359489612d8c34ba9095a90363f6de92ed212730c28c4e4de267768bade79b` |
| export/anim/0010.png | `7cc6eb52da143b4f776afa67d19afafa1960cbc2bc2ee24d157bafdbfb63d470` |
| export/anim/0011.png | `a3b66d21c89f6b7c68cd511a28e3241f5bafa7dd2dbf86ea418f7166abddda07` |
| export/anim/0012.png | `770b00faabe69b803bb19df78a7d2013473146334496a62c9e350ec1fcbcde45` |
| export/anim/0013.png | `526eb6d37894f18b882f58d84e88bf921916ca9104b39f0d0f92babeeb23ef37` |
| export/anim/0014.png | `7c518cf6c65d3efb388a1fbbef159272fc91f918757eb425669bd975fd2941f3` |
| export/anim/0015.png | `14b3d6de65eb625e5879aa71e5dbca5a422fbd4b7536a9849ef2484b75086b84` |
| export/anim/0016.png | `21878a9fdacc9e113baa582ddcd41eb17ebda57e14081020841e33ad5dcc5ab0` |
| export/static.png | `5b359489612d8c34ba9095a90363f6de92ed212730c28c4e4de267768bade79b` |
| work/lilac_butterfly_base.png | `03ee9dc8611631f4b238ae66d4e2ffcb6c1be68e5831d248820ba0573492bcd7` |
| work/lilac_butterfly_butterfly.png | `0ebcf8403ce4b3858c4ca8dfb860d489ef65c43239b410fd247edb3e5dd35521` |
| work/lilac_butterfly_cleanup.py | `f2485aa2d74f2b9384702b25fe905eca506e838ff8c6e13f7328dc2097413925` |
| work/lilac_butterfly_earring_left.png | `a04bbc6e80059ce09bd26953d611007d693209d569f78ba8d635de5116f6e865` |
| work/lilac_butterfly_earring_right.png | `26dbccbf9fce581fdadc8aa009906f4778785610801b46eebcd3c4ced4d636a4` |
| work/lilac_butterfly_eyes_closed.png | `e39c8b683f65d3f4e905b44c530114a1b01b2f84b42cbd8af3dbbee9038d12ac` |
| work/lilac_butterfly_eyes_open.png | `ccbba0211e36f0665c40e34ebfbd618f4564ea5424fdf65c99861ab3a81a551b` |
| work/lilac_butterfly_head.png | `fe82d945adbdd374f8c51dd4b6516bd3387bda86c320fa99ddfda5323e355421` |
| work/lilac_butterfly_loop.py | `82d015bad0dcc12fadb4fdee7e5f02aa0464df37e6e80ff795179c1975a93583` |
| work/lilac_butterfly_pin_pearls.png | `22807b37fe81bd82d82d462cf688e753098848bcab99a5f8c078eba852ac9556` |
| work/lilac_butterfly_portrait.png | `712232218d6ec2bf6914a94e032be8e45f01a87292b414821de9d4b0202b293b` |
| work/lilac_butterfly_torso.png | `58d5d9fc527dd18c94835c12d1f2970caad46a8eee5b70281a4cd0a5f92163b6` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `b6600989e1662e220e37efff8d14ecd1be860ea0ca80d7e0c65d64a068e09ef9` | 1254 x 1254 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e03f-763d-7c12-b88d-ab5392c7627e` |
| concept/20261115-02.png | `d990ad568dbe62a377e4ac59845f8a3a942bfa7a0e2f215b91b21b717dad9fd1` | 1254 x 1254 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e04e-3b19-7f71-946a-2e74cf2a522e` |
| work/lilac_butterfly_cleanup.py | `7c1afbc8e545a649985f0daf7f52283a9e10f2d4759b1a626829837d0781b45a` | W-3 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_cleanup.py` |
| work/lilac_butterfly_loop.py | `d7621f6411d4320246ee8463ac8104ed56781f4033133ee54aad1e688cce69af` | W-4 and W-5 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_loop.py`; the same script for every avatar of this batch with the item's geometry |
| work/lilac_butterfly_master.png | `176f94580284231147a3dd4ae5f25a668f6f80ada46ba3753803e18c53308c59` | 1254 x 1254 RGBA, the cut-out master | `lilac_butterfly_cleanup.py` |
| work/lilac_butterfly_base.png | `33c18a6624172c8cc8b562ba1846855b908d7859c73aeffedc1557128364a775` | The portrait with the moving parts taken out (filled or cleared) | `lilac_butterfly_loop.py` |
| work/lilac_butterfly_butterfly.png | `5726e8fa1dab17ba7e628fd0ef973b0569cc4212282a93ccf3b4f950f690c617` | Moving part `butterfly`: the portrait through its feathered polygon | `lilac_butterfly_loop.py` |
| work/lilac_butterfly_earring_left.png | `149cf74a0193fd814bcca33b394fb906cc431665bcf3d7740b481107e621da55` | Moving part `earring_left`: the portrait through its feathered polygon | `lilac_butterfly_loop.py` |
| work/lilac_butterfly_earring_right.png | `053e1ded64e0fe884cdb5c31a52e4b0f7f02556d5c4485696461223f6e0e35f9` | Moving part `earring_right`: the portrait through its feathered polygon | `lilac_butterfly_loop.py` |
| work/lilac_butterfly_eyes_closed.png | `c0bbf3772d04f24175c8cca1450fabd4f7592e114316b0b0c2523f8c41ea519c` | Eye region, closed (aligned edit) through the same mask | `lilac_butterfly_loop.py` |
| work/lilac_butterfly_eyes_open.png | `8836405a15ac89a2461b3d05449ab6dde2c740e3b34f27e044b512c93d47360f` | Eye region, open (canonical concept) through the feathered eye mask | `lilac_butterfly_loop.py` |
| work/lilac_butterfly_head.png | `bd2f07298d34319d86720a7d187d9a7f2f64030e5318855da727d21681259798` | Head and hair: the portrait weighted by the head field of the warp | `lilac_butterfly_loop.py` |
| work/lilac_butterfly_pin_pearls.png | `b13cc11285c11b5fb98cc8c8b9eb48df80a1bf0e8f8ba65723d2b5e5181b9874` | Moving part `pin_pearls`: the portrait through its feathered polygon | `lilac_butterfly_loop.py` |
| work/lilac_butterfly_portrait.png | `c308dc9b59ee24561ebf56c9a8b9fb1dbfaa921e4da4301b784c25c54bd7d473` | 768 x 768 RGBA portrait crop, no circle | `lilac_butterfly_cleanup.py` |
| work/lilac_butterfly_torso.png | `967b8dd73cbab3e01418a172cc56566d96c215e8184f8ed0bf7dbe91c99b4753` | Torso and costume: the portrait weighted by one minus the head field | `lilac_butterfly_loop.py` |
| export/static.png | `378021e143fa4918b473aed8940a5a5da12786a291a199334ebcbe5721dd859c` | 192 x 192 RGBA static, loop frame 9 | `lilac_butterfly_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `lilac_butterfly_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted below. Sent 17:26:57 UTC; `c2pa.created` 2026-11-15T17:27:54Z |
| 2026-11-15 | concept/20261115-02.png | Edit of `concept/20261115-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 17:43:05 UTC; `c2pa.created` 2026-11-15T17:43:43Z |
| 2026-11-16 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/lilac_butterfly_base.png, work/lilac_butterfly_butterfly.png, work/lilac_butterfly_cleanup.py, work/lilac_butterfly_earring_left.png, work/lilac_butterfly_earring_right.png, work/lilac_butterfly_eyes_closed.png, work/lilac_butterfly_eyes_open.png, work/lilac_butterfly_head.png, work/lilac_butterfly_loop.py, work/lilac_butterfly_pin_pearls.png, work/lilac_butterfly_portrait.png, work/lilac_butterfly_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(122, 0, 1200, 1078)` to `(75, -182, 1247, 990)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her mid-twenties with light skin and East Asian features, glossy black hair in a low side bun, a jewelled butterfly hair pin with delicate open wings of lilac crystal set above one ear, and a pale sea-glass green silk gown with a high lace collar and pearl buttons, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261115-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her mid-twenties with light skin and East Asian features, glossy black hair in a low side bun, a jewelled butterfly hair pin with delicate open wings of lilac crystal set above one ear, and a pale sea-glass green silk gown with a high lace collar and pearl buttons, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261115-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261115-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.
