# avatar.citrine_braid

Animated avatar of set 1 (#171), batch A, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: `concept/20261115-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261115-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a man in his thirties with warm medium brown skin, long black hair in a single braid over one shoulder bound with small gold rings and ending in a gold tassel, a neat short beard, and a deep violet silk jacket with gold embroidery at the collar"; quoted below with the edit prompt
- Concept against the prompt: Differs from the prompt: the hair covers the ears, so no citrine earrings show, and the gold tassel at the end of the braid falls below the portrait crop. Not asked for: a ruby and emerald pendant at the collar and ruby shoulder clasps with gold cord swags and a pearl. The item keeps its slug; the ambient part is a glint on the two rubies instead of the earrings.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.11, 0.05 px, scale 0.9992 and rotation -0.0002 rad (mean squared grey difference outside the eyes 0.00123 before, 0.00116 after), and the eye patch was offset by [-6.2, -1.7, -1.2] in RGB to meet the canonical skin. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, source `shop:offer.avatar_citrine_braid`. The offer id is a placeholder until the shop record exists (V-15). Animated avatars are proposed at `epic` and above, as the batch leans to the higher tiers of `docs/rarity.md`; this batch proposes 5 epic, 4 legendary, 1 mythic.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 1 animated detail (ruby glints), so `epic`, was `legendary`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signatures were not validated with a C2PA tool.

| Field | `concept/20261115-01.png` | `concept/20261115-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T16:56:30Z | `c2pa.created`, 2026-11-15T16:58:02Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 16:55:41 | 2026-11-15 16:57:00 |
| Codex session | `01a1e022-d411-7f53-b1ed-8a9ca5238d28` | `01a1e024-0bf0-7122-ba59-997eee7c0f21` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/citrine_braid/work/citrine_braid_cleanup.py citrine_braid assets-src/cosmetic/avatar/citrine_braid/concept/20261115-01.png assets-src/cosmetic/avatar/citrine_braid/work
```

It printed background [199.0, 199.0, 200.0], pockets 4989 px, crop (81, 0, 1128, 1047), portrait 768. The script is `amethyst_courtier_cleanup.py` of batch A of the static avatars with three changes, all for the same reason: on these concepts grey background stayed opaque between loose strands and the neck, and hair edges kept a light outline. (1) It writes the master and a square 768 x 768 portrait crop (`work/citrine_braid_portrait.png`, the same crop rule, no circle) instead of the 512 static, because the static of an animated avatar is a loop key frame. (2) Enclosed background pockets are taken within 24 of the grey (10 before) and up to 140 px inside the outline (80 before), and a pocket counts only if its median colour is within 12 of the grey, so pearls, crystal and silver that come close to the grey stay. (3) In a 6 px band along the background the alpha is capped by projecting each pixel between the grey and the mean colour of the nearby solid subject (Gaussian sigma 6); where that colour is itself within 40 of the grey, by the colour distance to the grey (0 at 14, 1 at 60). The colour of edge pixels is unmixed from the grey and blended toward the nearby subject colour by one minus the alpha, so faint edge pixels take the subject's colour instead of a light speck.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | As in the static batch A with the changes above. The loop script puts a circular alpha (soft edge of 2.25 px at 768, 0.56 px at 192) on every frame, because the portrait is shown in a circle (`docs/art-direction.md` 5.1); the class canvas is open (O-6) and D-1 sets 192 x 192 |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Yes | Written by the loop script into `work/` (table below) |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen in the concept, the master or the frames; the glint is a painted highlight clipped to the stone, not a glow |
| 5. Silhouette and smallest-size tests | In part | The agent looked at every frame at 192 px and at 2x on a dark backdrop and at frame 1 on a light one; face, expression and the moving part read. The smallest display size and frame band of an avatar are open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-15 by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/avatar/citrine_braid/work/citrine_braid_loop.py assets-src/cosmetic/avatar/citrine_braid/concept assets-src/cosmetic/avatar/citrine_braid/work assets-src/cosmetic/avatar/citrine_braid/export
```

It printed `citrine_braid_loop: blink edit registered (tx, ty, scale, angle) = (0.1075, 0.0503, 0.9992, -0.0002), residual 0.00123 -> 0.00116, eye colour offset [-6.2, -1.7, -1.2], parts 0, filled 0 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run into empty folders wrote files byte-identical to each other and to the committed ones.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes. Half-closed frames made by cross-fading showed two sets of lids and were dropped | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | a glint on the stone at (411, 682), ellipse 21 x 27 px at 768, frames 11 to 15; a glint on the stone at (139, 659), ellipse 15 x 15 px at 768, frames 14 to 17 | A painted highlight on the gem: a soft streak that crosses the stone, clipped to the stone's ellipse, warm ivory at up to 0.55 opacity, normal blending, no halo outside the stone (not a glow, `docs/art-direction.md` section 8) |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 2.27 of 255, against 2.22 on average and 2.69 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `citrine_braid: background [199.0, 199.0, 200.0], pockets 4989 px, crop (36, -180, 1174, 958), portrait 768`
- Crop `(81, 0, 1128, 1047)` to `(36, -180, 1174, 958)` on the concept
- Scale k = 0.9200. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- `register` now clips a crop edge above the concept to 0 when it marks the registration region, as the later loop scripts of this batch do; with the old slice a negative top would have marked the wrong rows
- Printed by the loop: `citrine_braid_loop: blink edit registered (tx, ty, scale, angle) = (0.1086, 0.0662, 0.9993, -0.0002), residual 0.00105 -> 0.00099, eye colour offset [-6.2, -1.7, -1.2], parts 0, filled 0 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.016 before, 0.172 after; highest over the 16 loop frames 0.005 before, 0.167 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/citrine_braid/work/citrine_braid_cleanup.py citrine_braid assets-src/cosmetic/avatar/citrine_braid/concept/20261115-01.png assets-src/cosmetic/avatar/citrine_braid/work
python -I assets-src/cosmetic/avatar/citrine_braid/work/citrine_braid_loop.py assets-src/cosmetic/avatar/citrine_braid/concept assets-src/cosmetic/avatar/citrine_braid/work assets-src/cosmetic/avatar/citrine_braid/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 24 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/citrine_braid` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `d141c02ec1b0765d7349af7e22a157f249680bcb8f1bfe15a18268926f1ea7fd` |
| export/anim/0002.png | `e186daf0910d97be43f8a5b3bcc739a16e0d886f4802a67d1eb651b7fa9ab215` |
| export/anim/0003.png | `c6a064edaea0222aa205aa81dd4223f1cbe90e596735579d2e5c462d877814ef` |
| export/anim/0004.png | `3d66b5b66cb2b6b0378268e312de57513cb0b415762958a5a92166258ba72c71` |
| export/anim/0005.png | `9dae2c5be4523dde7d1372e3d897e3db167efa8f6369b3775d7d2f82bb5da8d0` |
| export/anim/0006.png | `fdeeec810099b1f6144a5e261ceb416d8fe9abc3f32d74351e9f26690bdd901c` |
| export/anim/0007.png | `48a7e4390b45e1a64a1f1db92427068a7847d11d653bfe0fe8d4e8ba224597ac` |
| export/anim/0008.png | `a3288b3088d0a4952c0f22024c6aa5b42582bdfcb89773a116923487e435d460` |
| export/anim/0009.png | `ba847783805669885ec8674dc14305cde41dcbeaced8851a8c2f35a44c6d7766` |
| export/anim/0010.png | `d8435ce45999a15016768ded823b38e17353b0e1c7cb1336f143c1808a35873b` |
| export/anim/0011.png | `9a1533dcef9f89c2fc1cf034816d7638bb93b6a1d8cdb318bdf6f5f8dc3f641a` |
| export/anim/0012.png | `6c0c7de66e107e9c2f5c5a1d334357d00b4ef7e293369fb9b4f373e52636918d` |
| export/anim/0013.png | `736aa5872b90e7568eb4aae7c70c262cf0610f489fa6b3cceea5d2d71fccc03e` |
| export/anim/0014.png | `29175afc234e4f617cace03e5b148a2c5c9ac0ef1f0a03ea492a871f41e01169` |
| export/anim/0015.png | `43e1682094b82f4f6ab19598283256b312913023cc0778f1505bad939723d90d` |
| export/anim/0016.png | `7e7c338a9e74e18f79ffb57773baa5c752ebcea3037b0d415714c5125fb03c99` |
| export/static.png | `ba847783805669885ec8674dc14305cde41dcbeaced8851a8c2f35a44c6d7766` |
| work/citrine_braid_base.png | `a4b7614c20be757ce3f5272a9fd542f27ac4cd3fa41b7394419aae765bca0174` |
| work/citrine_braid_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` |
| work/citrine_braid_eyes_closed.png | `4ab6a2cebae1a3fe27784f5ab807b83ac48df0b0a116691b82d31b999a73186f` |
| work/citrine_braid_eyes_open.png | `4467d826b62c828f9c913e87c671bc0ffc8cca86a9b66458407241be98af4409` |
| work/citrine_braid_head.png | `889a0b232bb5eed9cd1caa15c849d1a796f229e54b3d0f8400b6ee37bec93cb6` |
| work/citrine_braid_loop.py | `9a500bee308f1eee7bd4ad24506edbb77b4ddc14fc19aae866f6519a68a6c878` |
| work/citrine_braid_portrait.png | `193c1dc29d7d103999223e2690438416e0d165d901d135ff022ca9031b26bef9` |
| work/citrine_braid_torso.png | `b13a3c18bf655c97b610fa660bd2cc90afdff34de208668ae13c9c95d345a914` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `2fad1d50706f36a23514a13c34e54fa5d3a4c404a8e60a6b0af83f1ea9603ead` | 1218 x 1292 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e022-d411-7f53-b1ed-8a9ca5238d28` |
| concept/20261115-02.png | `496617eb3e5379307b82106692666210dc7d3cfec38247ed67990584ff52a7d2` | 1218 x 1292 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e024-0bf0-7122-ba59-997eee7c0f21` |
| work/citrine_braid_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-15, from `amethyst_courtier_cleanup.py` |
| work/citrine_braid_loop.py | `da101d9c2857de62c74cd3e2bb9e0e855ebede282438a1e16354902d7f207428` | W-4 and W-5 script | Written by Claude Code, 2026-11-15; the same script for every avatar of this batch with the item's geometry |
| work/citrine_braid_master.png | `e6c6e3e440ea63ad4649fad70065c196fafc78c2ca8108157e539df3e977cf01` | 1218 x 1292 RGBA, the cut-out master | `citrine_braid_cleanup.py` |
| work/citrine_braid_base.png | `101c42043e1f33fdece9a8a06e55eaa925c33efef94bf5d757139f32281505e1` | The portrait with the moving parts taken out (filled or cleared) | `citrine_braid_loop.py` |
| work/citrine_braid_eyes_closed.png | `12a1cf204d30ab9c7adc55607fff8576816770d4bf97efec54a04da93ed9c676` | Eye region, closed (aligned edit) through the same mask | `citrine_braid_loop.py` |
| work/citrine_braid_eyes_open.png | `e44bcace65766c58e6f6b8d31df6419664ea0311d05073b5b179dc1e11e79f57` | Eye region, open (canonical concept) through the feathered eye mask | `citrine_braid_loop.py` |
| work/citrine_braid_head.png | `4a623256c67a5d59f85b6dc50163e9ccaccd556b778db1308ff4d6375e6f009c` | Head and hair: the portrait weighted by the head field of the warp | `citrine_braid_loop.py` |
| work/citrine_braid_portrait.png | `5168dec5689d29284544078c24d58021dd1d43d9d6ad99fe196ab09f2b6aa9ad` | 768 x 768 RGBA portrait crop, no circle | `citrine_braid_cleanup.py` |
| work/citrine_braid_torso.png | `b492d11154a7cc82111679f7361da17af4099f58c03d313111f44f7f9784fc25` | Torso and costume: the portrait weighted by one minus the head field | `citrine_braid_loop.py` |
| export/static.png | `de95fafabc5215ca5491903cf149f0bd7234ab5e9f1b39f5212573867591bc03` | 192 x 192 RGBA static, loop frame 9 | `citrine_braid_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `citrine_braid_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted below. Sent 16:55:41 UTC; `c2pa.created` 2026-11-15T16:56:30Z |
| 2026-11-15 | concept/20261115-02.png | Edit of `concept/20261115-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 16:57:00 UTC; `c2pa.created` 2026-11-15T16:58:02Z |
| 2026-11-15 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/citrine_braid_base.png, work/citrine_braid_cleanup.py, work/citrine_braid_eyes_closed.png, work/citrine_braid_eyes_open.png, work/citrine_braid_head.png, work/citrine_braid_loop.py, work/citrine_braid_portrait.png, work/citrine_braid_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(81, 0, 1128, 1047)` to `(36, -180, 1174, 958)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his thirties with warm medium brown skin, long black hair in a single braid over one shoulder bound with small gold rings and ending in a gold tassel, a neat short beard, and a deep violet silk jacket with gold embroidery at the collar, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261115-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his thirties with warm medium brown skin, long black hair in a single braid over one shoulder bound with small gold rings and ending in a gold tassel, a neat short beard, and a deep violet silk jacket with gold embroidery at the collar, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261115-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261115-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-15.
