# avatar.pearl_comb

Animated avatar of set 1 (#171), batch A, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: `concept/20261115-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261115-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a woman in her twenties with medium tan skin, long wavy dark brown hair swept over one shoulder, a tall carved ivory comb in her hair with short strands of seed pearls hanging from it, and a dusty rose silk gown with fine silver embroidery"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt. Not asked for but present: ruby and pearl drop earrings, a ruby, pearl and crystal necklace, and burgundy velvet bands with gold embroidery on the shoulders.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.14, 0.17 px, scale 1.0001 and rotation 0.0000 rad (mean squared grey difference outside the eyes 0.00137 before, 0.00134 after), and the eye patch was offset by [-8.0, -2.6, -0.4] in RGB to meet the canonical skin. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `legendary`, source `shop:offer.avatar_pearl_comb`. The offer id is a placeholder until the shop record exists (V-15). Animated avatars are proposed at `epic` and above, as the batch leans to the higher tiers of `docs/rarity.md`; this batch proposes 5 epic, 4 legendary, 1 mythic.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 2 animated details (pearl strands, earrings (pair)), so `legendary`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signatures were not validated with a C2PA tool.

| Field | `concept/20261115-01.png` | `concept/20261115-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T16:45:24Z | `c2pa.created`, 2026-11-15T16:46:41Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 16:44:46 | 2026-11-15 16:45:46 |
| Codex session | `01a1e018-d70c-74b1-a28e-d7ab28664885` | `01a1e019-c1c8-7623-9e17-cc72872b5032` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/pearl_comb/work/pearl_comb_cleanup.py pearl_comb assets-src/cosmetic/avatar/pearl_comb/concept/20261115-01.png assets-src/cosmetic/avatar/pearl_comb/work
```

It printed background [200.0, 200.0, 200.0], pockets 7870 px, crop (116, 0, 1147, 1031), portrait 768. The script is `amethyst_courtier_cleanup.py` of batch A of the static avatars with three changes, all for the same reason: on these concepts grey background stayed opaque between loose strands and the neck, and hair edges kept a light outline. (1) It writes the master and a square 768 x 768 portrait crop (`work/pearl_comb_portrait.png`, the same crop rule, no circle) instead of the 512 static, because the static of an animated avatar is a loop key frame. (2) Enclosed background pockets are taken within 24 of the grey (10 before) and up to 140 px inside the outline (80 before), and a pocket counts only if its median colour is within 12 of the grey, so pearls, crystal and silver that come close to the grey stay. (3) In a 6 px band along the background the alpha is capped by projecting each pixel between the grey and the mean colour of the nearby solid subject (Gaussian sigma 6); where that colour is itself within 40 of the grey, by the colour distance to the grey (0 at 14, 1 at 60). The colour of edge pixels is unmixed from the grey and blended toward the nearby subject colour by one minus the alpha, so faint edge pixels take the subject's colour instead of a light speck.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | As in the static batch A with the changes above. The loop script puts a circular alpha (soft edge of 2.25 px at 768, 0.56 px at 192) on every frame, because the portrait is shown in a circle (`docs/art-direction.md` 5.1); the class canvas is open (O-6) and D-1 sets 192 x 192 |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Yes | Written by the loop script into `work/` (table below) |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen in the concept, the master or the frames |
| 5. Silhouette and smallest-size tests | In part | The agent looked at every frame at 192 px and at 2x on a dark backdrop and at frame 1 on a light one; face, expression and the moving part read. The smallest display size and frame band of an avatar are open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-15 by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/avatar/pearl_comb/work/pearl_comb_loop.py assets-src/cosmetic/avatar/pearl_comb/concept assets-src/cosmetic/avatar/pearl_comb/work assets-src/cosmetic/avatar/pearl_comb/export
```

It printed `pearl_comb_loop: blink edit registered (tx, ty, scale, angle) = (0.1386, 0.1673, 1.0001, 0.0000), residual 0.00137 -> 0.00134, eye colour offset [-8.0, -2.6, -0.4], parts 3, filled 19862 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run into empty folders wrote files byte-identical to each other and to the committed ones.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes. Half-closed frames made by cross-fading showed two sets of lids and were dropped | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `pearl_strands` (layer `work/pearl_comb_pearl_strands.png`, pivot (212, 160)) turns about its pivot by up to 4.0 degrees with sin(lag) - sin(2 pi (f - 8) / 16 + lag), lag 0.5, the turn growing from 0 at the pivot to full at 150 px (768 scale), so the top stays attached and the end swings; the part also follows the head or torso warp at its pivot; `earring_left` (layer `work/pearl_comb_earring_left.png`, pivot (310, 380)) turns about its pivot by up to 4.0 degrees with sin(lag) - sin(2 pi (f - 8) / 16 + lag), lag 0.3, the turn growing from 0 at the pivot to full at 70 px (768 scale), so the top stays attached and the end swings; the part also follows the head or torso warp at its pivot; `earring_right` (layer `work/pearl_comb_earring_right.png`, pivot (537, 352)) turns about its pivot by up to 4.0 degrees with sin(lag) - sin(2 pi (f - 8) / 16 + lag), lag 0.8, the turn growing from 0 at the pivot to full at 70 px (768 scale), so the top stays attached and the end swings; the part also follows the head or torso warp at its pivot | The part lies over hair or skin, so the hole it leaves in the base layer is filled by normalized convolution from its surroundings (Gaussian sigma 48 down to 3); only a sliver of it shows while the part is off its rest position |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 3.55 of 255, against 2.88 on average and 3.58 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `pearl_comb: background [200.0, 200.0, 200.0], pockets 7870 px, crop (71, -177, 1192, 944), portrait 768`
- Crop `(116, 0, 1147, 1031)` to `(71, -177, 1192, 944)` on the concept
- Scale k = 0.9197. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- `register` now clips a crop edge above the concept to 0 when it marks the registration region, as the later loop scripts of this batch do; with the old slice a negative top would have marked the wrong rows
- Printed by the loop: `pearl_comb_loop: blink edit registered (tx, ty, scale, angle) = (0.1516, 0.1818, 1.0002, 0.0001), residual 0.00115 -> 0.00112, eye colour offset [-7.9, -2.6, -0.4], parts 3, filled 17157 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.016 before, 0.172 after; highest over the 16 loop frames 0.010 before, 0.161 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/pearl_comb/work/pearl_comb_cleanup.py pearl_comb assets-src/cosmetic/avatar/pearl_comb/concept/20261115-01.png assets-src/cosmetic/avatar/pearl_comb/work
python -I assets-src/cosmetic/avatar/pearl_comb/work/pearl_comb_loop.py assets-src/cosmetic/avatar/pearl_comb/concept assets-src/cosmetic/avatar/pearl_comb/work assets-src/cosmetic/avatar/pearl_comb/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 27 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/pearl_comb` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `f7c9d6a7a84c000ff38d5f73a0446c3bfb830e798a07084bf6dc558b3ccbc9e9` |
| export/anim/0002.png | `b3a25ed4a27de943393968e3b65a747001130377aaeee6dd6a62b6b097a87f90` |
| export/anim/0003.png | `a0dcf460eaa6bf602eba4e2c38fe24d7d2e082561e267a83650d2d03d6c4c313` |
| export/anim/0004.png | `6b3bcc66cf0bcac757c1f617fad452fecfd2079f6bb9f8c44c094be97ae215a6` |
| export/anim/0005.png | `172b073129b653907169baa27f5551e50e8ef8a83e49d0eb5d44955b1b303777` |
| export/anim/0006.png | `3aa1d323bf672dd16ece0c5c13818df23a83de4270ccb1fdf54f82ee5eae39ae` |
| export/anim/0007.png | `11cb3891a2be10feaf8749a27cbee39f8c73eb9a3b9ee50126db36c338332584` |
| export/anim/0008.png | `a68ed68c329c55c114d6534cd2646e19f9eb1d54838a175dff6d5e323b8a8046` |
| export/anim/0009.png | `17249ccf2c7903009e01f6a60ce754356f8be358e02d7d35d21ddda9ad4f0233` |
| export/anim/0010.png | `b851ecede9ed17217a65b457e7f38cb5e7f62fe7664638f43706d132c732075d` |
| export/anim/0011.png | `ed2eec5a19ea4f591ca0abfe2beb06f9fd0e1b7aace2148b70ec947ef5b60a08` |
| export/anim/0012.png | `fb4000a8b3728129bed1c831ccdbed82567722923781f07e584a3e53eac4ac04` |
| export/anim/0013.png | `7be02cfd59d436befbc5f92b928fd17e2bd6a623eb51bdc15a1fa3bfb0519300` |
| export/anim/0014.png | `b6adfb5025b74b13647157b31acff166126acb419fb918bf92f3130b822f6a45` |
| export/anim/0015.png | `91d5ba5030c649880c332703a8351f7b75720eee50bfb966e052984715623076` |
| export/anim/0016.png | `9a6760aa43059e4fc08b2ccb439b20643e4df3c598c429e80cdce48462727ddc` |
| export/static.png | `17249ccf2c7903009e01f6a60ce754356f8be358e02d7d35d21ddda9ad4f0233` |
| work/pearl_comb_base.png | `47e3ee10310954c93691ef1b9139471eba0305d14aaea0336e04b6e7c1bb886b` |
| work/pearl_comb_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` |
| work/pearl_comb_earring_left.png | `12cc3ed32b74c1f4fbc24e35cc68184a2ef73620cdf570b7fc30484a16384862` |
| work/pearl_comb_earring_right.png | `16ef58af356b3e4824b87ce864b2d6e9e758beb8dfee4c185d5630884265e6b0` |
| work/pearl_comb_eyes_closed.png | `5409ee81ebb13809a92dcf43660776d32ccbec1c088c2a0976c81118e8f5bd78` |
| work/pearl_comb_eyes_open.png | `8f86870e98a807c9fadd86ba2a3e525090a2437ea2750b37a66830e38de62d6a` |
| work/pearl_comb_head.png | `adc42d434335b5305a444b4a06bc8542ebe27174ca2f04c630d018944588ad84` |
| work/pearl_comb_loop.py | `7f39c38e588da2371a94ecd9879620bf31665ed3b3de18ff61acd7a68ee340ab` |
| work/pearl_comb_pearl_strands.png | `fd6e8c0cadf0576962dbe520e5e7f35ed17678efb3fbcef6b57fd098ab4b1cf1` |
| work/pearl_comb_portrait.png | `ef624204f1f43fe66322df1b327e4ae63ac3934df744fa3308bc19217932bbf6` |
| work/pearl_comb_torso.png | `0df1b545fe9a74a5710186bb32c8f156229fde14dea6b02c9db90283518bf4ce` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `f2806d0be3d2a9f3f1a777bede32c10603f3d1686505845ebf7f025c329c1354` | 1312 x 1199 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e018-d70c-74b1-a28e-d7ab28664885` |
| concept/20261115-02.png | `6c672b7d50239ff3e3e37a70207bad6d0c4819f4fbe0c899427d4b5eda8e8834` | 1312 x 1199 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e019-c1c8-7623-9e17-cc72872b5032` |
| work/pearl_comb_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-15, from `amethyst_courtier_cleanup.py` |
| work/pearl_comb_loop.py | `f2840c1690d6be18cb1d9668585b1c2e8750a7fe6e2a9162d994d8e761a67ba2` | W-4 and W-5 script | Written by Claude Code, 2026-11-15; the same script for every avatar of this batch with the item's geometry |
| work/pearl_comb_master.png | `883d34c8d709310fd3d2fb92a17558a287b790aca58805d3c331c451c310d133` | 1312 x 1199 RGBA, the cut-out master | `pearl_comb_cleanup.py` |
| work/pearl_comb_base.png | `c3a2cb841498981b221ca2f7a89639fac2f6e0cc11391245085585b8daa912f4` | The portrait with the moving parts taken out (filled or cleared) | `pearl_comb_loop.py` |
| work/pearl_comb_earring_left.png | `346c105adaab827e8d89d0066e5e8e05c949444c575211fa58d7d330ff6a2a3f` | Moving part `earring_left`: the portrait through its feathered polygon | `pearl_comb_loop.py` |
| work/pearl_comb_earring_right.png | `b9de14465431e5c8279825a6daef43e4e0e8e88f85371e6f2b4b0312179e1c29` | Moving part `earring_right`: the portrait through its feathered polygon | `pearl_comb_loop.py` |
| work/pearl_comb_eyes_closed.png | `5fc012147d45afdf914c58af3b04a1740a1b41b35b58eb050af1bae8ad3806cd` | Eye region, closed (aligned edit) through the same mask | `pearl_comb_loop.py` |
| work/pearl_comb_eyes_open.png | `101fe89d25c8d003498fca7adc5f6ade5d63d7a318760fd6a2a59bcd16b16095` | Eye region, open (canonical concept) through the feathered eye mask | `pearl_comb_loop.py` |
| work/pearl_comb_head.png | `8f1e19e9a75584cb5ebbaa14cd846aa87294b8f4fc68a2b4e2c9adca94586420` | Head and hair: the portrait weighted by the head field of the warp | `pearl_comb_loop.py` |
| work/pearl_comb_pearl_strands.png | `db72d5e5941db57d04a31bb13e1615920a35da57419884a65a4acca7e9c0bf3c` | Moving part `pearl_strands`: the portrait through its feathered polygon | `pearl_comb_loop.py` |
| work/pearl_comb_portrait.png | `82a4fcae7bb37092a3bd25390c4ba3ad488be669bdb7bd4fca1259239f1d8e3a` | 768 x 768 RGBA portrait crop, no circle | `pearl_comb_cleanup.py` |
| work/pearl_comb_torso.png | `bc2d88eba20c6169017904c13b0a35ea876986fd9dfbe269b5e8787c54f7c223` | Torso and costume: the portrait weighted by one minus the head field | `pearl_comb_loop.py` |
| export/static.png | `e8766507e5b1c86b4c3011cdb05527355fd8a70c7fb2a0245a3596c04475aa54` | 192 x 192 RGBA static, loop frame 9 | `pearl_comb_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `pearl_comb_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted below. Sent 16:44:46 UTC; `c2pa.created` 2026-11-15T16:45:24Z |
| 2026-11-15 | concept/20261115-02.png | Edit of `concept/20261115-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 16:45:46 UTC; `c2pa.created` 2026-11-15T16:46:41Z |
| 2026-11-15 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/pearl_comb_base.png, work/pearl_comb_cleanup.py, work/pearl_comb_earring_left.png, work/pearl_comb_earring_right.png, work/pearl_comb_eyes_closed.png, work/pearl_comb_eyes_open.png, work/pearl_comb_head.png, work/pearl_comb_loop.py, work/pearl_comb_pearl_strands.png, work/pearl_comb_portrait.png, work/pearl_comb_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(116, 0, 1147, 1031)` to `(71, -177, 1192, 944)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her twenties with medium tan skin, long wavy dark brown hair swept over one shoulder, a tall carved ivory comb in her hair with short strands of seed pearls hanging from it, and a dusty rose silk gown with fine silver embroidery, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261115-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her twenties with medium tan skin, long wavy dark brown hair swept over one shoulder, a tall carved ivory comb in her hair with short strands of seed pearls hanging from it, and a dusty rose silk gown with fine silver embroidery, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261115-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261115-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-15.
