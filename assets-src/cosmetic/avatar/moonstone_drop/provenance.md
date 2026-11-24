# avatar.moonstone_drop

Animated avatar of set 1 (#171), batch A, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: `concept/20261115-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261115-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a young man in his twenties with pale skin, long straight platinum blond hair tied low at the nape, a single long silver earring with a moonstone drop, and an ivory and silver brocade coat with a high collar"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt: one earring, on the right ear in the picture. Not asked for: a moonstone brooch on the cream silk cravat and a silver cuff on the ponytail.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.28, 0.20 px, scale 1.0002 and rotation -0.0002 rad (mean squared grey difference outside the eyes 0.00075 before, 0.00071 after), and the eye patch was offset by [-10.1, -4.5, -0.0] in RGB to meet the canonical skin. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, source `shop:offer.avatar_moonstone_drop`. The offer id is a placeholder until the shop record exists (V-15). Animated avatars are proposed at `epic` and above, as the batch leans to the higher tiers of `docs/rarity.md`; this batch proposes 5 epic, 4 legendary, 1 mythic.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 1 animated detail (earring), so `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signatures were not validated with a C2PA tool.

| Field | `concept/20261115-01.png` | `concept/20261115-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T16:47:48Z | `c2pa.created`, 2026-11-15T16:49:17Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 16:47:08 | 2026-11-15 16:48:14 |
| Codex session | `01a1e01b-01a4-7461-af15-99d63785b2f3` | `01a1e01c-02a9-7923-96f6-b0a91f1b1832` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-15 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/moonstone_drop/work/moonstone_drop_cleanup.py moonstone_drop assets-src/cosmetic/avatar/moonstone_drop/concept/20261115-01.png assets-src/cosmetic/avatar/moonstone_drop/work
```

It printed background [202.0, 202.0, 202.0], pockets 11151 px, crop (88, 0, 1119, 1031), portrait 768. The script is `amethyst_courtier_cleanup.py` of batch A of the static avatars with three changes, all for the same reason: on these concepts grey background stayed opaque between loose strands and the neck, and hair edges kept a light outline. (1) It writes the master and a square 768 x 768 portrait crop (`work/moonstone_drop_portrait.png`, the same crop rule, no circle) instead of the 512 static, because the static of an animated avatar is a loop key frame. (2) Enclosed background pockets are taken within 24 of the grey (10 before) and up to 140 px inside the outline (80 before), and a pocket counts only if its median colour is within 12 of the grey, so pearls, crystal and silver that come close to the grey stay. (3) In a 6 px band along the background the alpha is capped by projecting each pixel between the grey and the mean colour of the nearby solid subject (Gaussian sigma 6); where that colour is itself within 40 of the grey, by the colour distance to the grey (0 at 14, 1 at 60). The colour of edge pixels is unmixed from the grey and blended toward the nearby subject colour by one minus the alpha, so faint edge pixels take the subject's colour instead of a light speck.

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
python -I assets-src/cosmetic/avatar/moonstone_drop/work/moonstone_drop_loop.py assets-src/cosmetic/avatar/moonstone_drop/concept assets-src/cosmetic/avatar/moonstone_drop/work assets-src/cosmetic/avatar/moonstone_drop/export
```

It printed `moonstone_drop_loop: blink edit registered (tx, ty, scale, angle) = (0.2762, 0.1951, 1.0002, -0.0002), residual 0.00075 -> 0.00071, eye colour offset [-10.1, -4.5, -0.0], parts 1, filled 5136 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run into empty folders wrote files byte-identical to each other and to the committed ones.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes. Half-closed frames made by cross-fading showed two sets of lids and were dropped | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.5 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `earring` (layer `work/moonstone_drop_earring.png`, pivot (550, 324)) turns about its pivot by up to 3.5 degrees with sin(lag) - sin(2 pi (f - 8) / 16 + lag), lag 0.6, the turn growing from 0 at the pivot to full at 100 px (768 scale), so the top stays attached and the end swings; the part also follows the head or torso warp at its pivot | The part lies over hair or skin, so the hole it leaves in the base layer is filled by normalized convolution from its surroundings (Gaussian sigma 48 down to 3); only a sliver of it shows while the part is off its rest position |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 3.06 of 255, against 2.55 on average and 3.18 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `moonstone_drop: background [202.0, 202.0, 202.0], pockets 11151 px, crop (43, -176, 1164, 945), portrait 768`
- Crop `(88, 0, 1119, 1031)` to `(43, -176, 1164, 945)` on the concept
- Scale k = 0.9197. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- `register` now clips a crop edge above the concept to 0 when it marks the registration region, as the later loop scripts of this batch do; with the old slice a negative top would have marked the wrong rows
- Printed by the loop: `moonstone_drop_loop: blink edit registered (tx, ty, scale, angle) = (0.2148, -0.3415, 1.0002, 0.0001), residual 0.00058 -> 0.00053, eye colour offset [-10.6, -5.0, -0.6], parts 1, filled 4532 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.016 before, 0.172 after; highest over the 16 loop frames 0.010 before, 0.167 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/moonstone_drop/work/moonstone_drop_cleanup.py moonstone_drop assets-src/cosmetic/avatar/moonstone_drop/concept/20261115-01.png assets-src/cosmetic/avatar/moonstone_drop/work
python -I assets-src/cosmetic/avatar/moonstone_drop/work/moonstone_drop_loop.py assets-src/cosmetic/avatar/moonstone_drop/concept assets-src/cosmetic/avatar/moonstone_drop/work assets-src/cosmetic/avatar/moonstone_drop/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 25 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/moonstone_drop` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `640199ad567a6108c2290703497be083dd3c1830ab4c8b5583f6314bc985dc70` |
| export/anim/0002.png | `a6ff6de14c04f3489521177a556e7aa84a30b84f31e2be711a316d2158db7981` |
| export/anim/0003.png | `1f179dc7a26f98e11fdfcfc69d6aa1ecb09c7ea327a81d887d4c7f66d95b7d93` |
| export/anim/0004.png | `641bc428f8a5daa0bbb3bd4f73a157b45e7f2150b4e9fa347a8aad2cf38598a4` |
| export/anim/0005.png | `47ab28298c07e14ed86b7d9c408088a78c57b6ec6a0f2b9971e81be907c9d86a` |
| export/anim/0006.png | `26ece5fcf95e6e603e922d0481ad22216fcca587c9f3801f4f4f38b761772b58` |
| export/anim/0007.png | `14e896b2d7ac0667ab1ee7c1c2c4fb2411d8065631dc64ef31b1446dce307faf` |
| export/anim/0008.png | `9100ae8d60dfaa07935c763992eb0b658df44f621a1b16f1e70078efc3246bf8` |
| export/anim/0009.png | `ac2d8e8e68feae84d9dc850877d4395151a1d02bb00930d1ce69bd6912cb0c61` |
| export/anim/0010.png | `18976b944324792a70963679f389f205a130b21838558cdf24feeaab679c215c` |
| export/anim/0011.png | `25f7daba72f403b0aff5e707213c4c6a816bdc7b94a17a6b60bab4ce01b1f6af` |
| export/anim/0012.png | `511720f8764b93f0f990a7d626d78fd5214ac032e2a6f25858654352a8a4fe0c` |
| export/anim/0013.png | `d3161f94bf86551a8a6567580f1be4029a100f5c5f4d04c78899c9d28c98ca04` |
| export/anim/0014.png | `0b39ed3f76825ed5905d7fe4630f27daaf8dfb683edbec211fdb4b46eecee1f2` |
| export/anim/0015.png | `6e914b82ebfeca86e74198b3f102568a8fae5cd2920fa4622911e11bcb5296ce` |
| export/anim/0016.png | `0ac66e8c3b9f983b3197a70a5c91524a2333bf81064d914fc74d1178bb68edb6` |
| export/static.png | `ac2d8e8e68feae84d9dc850877d4395151a1d02bb00930d1ce69bd6912cb0c61` |
| work/moonstone_drop_base.png | `558e84bec4ddf2b603eb151c2143894276b0d384b9a982f0a05a0c02b89b6e7a` |
| work/moonstone_drop_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` |
| work/moonstone_drop_earring.png | `2ee07cfe7bd14dc7516df7829437a0fa782af231f9a4c306dab483a7009ed7a2` |
| work/moonstone_drop_eyes_closed.png | `65e1dbaf627fb4ac504143820c21a57045843d4cdb1e95aaabe23b556d755bcb` |
| work/moonstone_drop_eyes_open.png | `068ffc5c2d31bfafa024decbaeb931274ed65a7f88a983e2a6a9d065854bb130` |
| work/moonstone_drop_head.png | `628d6767c4797e884e57c45009a44fe16b5829380b15ca592558e52a45203193` |
| work/moonstone_drop_loop.py | `c8484056a6d249b79bf92a7df3e323fea3df6a16aa8099a8f578b8acf8d9a157` |
| work/moonstone_drop_portrait.png | `fd471d8a7aba4ac559633e3a7c30e802116117ba694d76e8d53984978936c5b8` |
| work/moonstone_drop_torso.png | `8eb0130cb98609ca791f26d208815a0b37f8d2ed8ea7eece396e54bb0585cbc3` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `7b0bce4b8524d7bc13a99aa66baf3e91df397bdf9a6c9d316f7935b707f30317` | 1312 x 1199 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e01b-01a4-7461-af15-99d63785b2f3` |
| concept/20261115-02.png | `ceb90cd423dde56a6a8a72fa94cefa112df06650153666c79ede8d049b286bdc` | 1312 x 1199 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e01c-02a9-7923-96f6-b0a91f1b1832` |
| work/moonstone_drop_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-15, from `amethyst_courtier_cleanup.py` |
| work/moonstone_drop_loop.py | `0ccba679f808fdd8fce4bc1bdce104998f9931db536a17f1445c2a4813605fbe` | W-4 and W-5 script | Written by Claude Code, 2026-11-15; the same script for every avatar of this batch with the item's geometry |
| work/moonstone_drop_master.png | `bab3f1e62b07a7da84a0334019401d566f3df88fe8643ddd9ba5dd24d8dae0ad` | 1312 x 1199 RGBA, the cut-out master | `moonstone_drop_cleanup.py` |
| work/moonstone_drop_base.png | `d8d9267e43142d94b0ad53956fba787b9c15916b51c0f52eb78d8c6c99fe2844` | The portrait with the moving parts taken out (filled or cleared) | `moonstone_drop_loop.py` |
| work/moonstone_drop_earring.png | `3905524c408a1702913c0f6e6402f0860dbcdbaea54f4a064d69f13f70a20142` | Moving part `earring`: the portrait through its feathered polygon | `moonstone_drop_loop.py` |
| work/moonstone_drop_eyes_closed.png | `0b83708311cc5ea61f836281ec2e878af1ec595ded602864ed8c00d73bee5544` | Eye region, closed (aligned edit) through the same mask | `moonstone_drop_loop.py` |
| work/moonstone_drop_eyes_open.png | `c8c16f8048e17d9bfcd8abd5cdef01d92e07fba73306b1163c390701b50e1955` | Eye region, open (canonical concept) through the feathered eye mask | `moonstone_drop_loop.py` |
| work/moonstone_drop_head.png | `11732b921ee60bfa5c672fcad105b030457a14d9e4fbe9f28772914e70b22f08` | Head and hair: the portrait weighted by the head field of the warp | `moonstone_drop_loop.py` |
| work/moonstone_drop_portrait.png | `ce11786d86a176ecdc3cdc47a662e112681821cccec1bfa5bb780b3c7ddaa347` | 768 x 768 RGBA portrait crop, no circle | `moonstone_drop_cleanup.py` |
| work/moonstone_drop_torso.png | `f4b6568cdeed26b69daa08b3bb5f54f704486cc57502a0fc6dd2c6c2b86a0ac1` | Torso and costume: the portrait weighted by one minus the head field | `moonstone_drop_loop.py` |
| export/static.png | `f938711d4c846fa03c16fc1e8705cb55f5af4404f3336c4a682b0683ec22c2b8` | 192 x 192 RGBA static, loop frame 9 | `moonstone_drop_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `moonstone_drop_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted below. Sent 16:47:08 UTC; `c2pa.created` 2026-11-15T16:47:48Z |
| 2026-11-15 | concept/20261115-02.png | Edit of `concept/20261115-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 16:48:14 UTC; `c2pa.created` 2026-11-15T16:49:17Z |
| 2026-11-15 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/moonstone_drop_base.png, work/moonstone_drop_cleanup.py, work/moonstone_drop_earring.png, work/moonstone_drop_eyes_closed.png, work/moonstone_drop_eyes_open.png, work/moonstone_drop_head.png, work/moonstone_drop_loop.py, work/moonstone_drop_portrait.png, work/moonstone_drop_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(88, 0, 1119, 1031)` to `(43, -176, 1164, 945)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a young man in his twenties with pale skin, long straight platinum blond hair tied low at the nape, a single long silver earring with a moonstone drop, and an ivory and silver brocade coat with a high collar, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261115-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a young man in his twenties with pale skin, long straight platinum blond hair tied low at the nape, a single long silver earring with a moonstone drop, and an ivory and silver brocade coat with a high collar, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261115-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261115-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-15.
