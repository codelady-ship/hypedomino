# avatar.watch_chain

Animated avatar of set 1 (#171), batch B, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 and 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, final runs on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-02.png
- Inputs uploaded to the generator: `concept/20261115-02.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261115-03.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a man in his fifties with light olive skin, thick wavy grey hair combed back and a short trimmed grey beard, a chocolate-brown velvet waistcoat over a cream shirt with a rust-orange silk tie, and a polished brass watch chain looping from a buttonhole high on the waistcoat just below the collar, clearly visible across the upper chest"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt: the brass chain hangs from a small ruby pin on the left of the waistcoat to a button on the right, across the upper chest. Not asked for: the ruby pin, a cream pocket square and brass buttons.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.07, 0.15 px, scale 0.9998 and rotation -0.0001 rad (mean squared grey difference outside the eyes 0.00067 before, 0.00066 after), and the eye patch was offset by [-7.2, 0.0, 3.3] in RGB to meet the canonical skin. The changes outside the eye mask are small and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `legendary`, source `shop:offer.avatar_watch_chain`. The offer id is a placeholder until the shop record exists (V-15). A glint runs along the whole chain while the chain sways. Animated avatars stay at `epic` and above as in batch A; batch B proposes 5 epic, 4 legendary, 1 mythic, the same split as batch A.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 2 animated details (chain glint, chain sway), so `legendary`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signatures were not validated with a C2PA tool.

| Field | `concept/20261115-02.png` | `concept/20261115-03.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T17:36:32Z | `c2pa.created`, 2026-11-15T17:52:53Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 17:35:46 | 2026-11-15 17:51:53 |
| Codex session | `01a1e047-8a2a-7902-ba76-a376cf238005` | `01a1e056-48ba-74b2-9a80-fb915c09af49` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). One earlier generation for this item was rejected and is not committed: `20261115-01.png` (Codex session `01a1e03d-6d0e-7b30-95c9-bdb1ac770eef`, prompt sent 2026-11-15 17:24:44 UTC), with the first prompt quoted below. It met the prompt, but the watch chain hung at the bottom edge of the concept, below the portrait crop, so the ambient part would not show. The prompt was changed in one place, the position of the chain, and the next generation became the canonical; the file numbers of this item therefore start at 02.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/watch_chain/work/watch_chain_cleanup.py watch_chain assets-src/cosmetic/avatar/watch_chain/concept/20261115-02.png assets-src/cosmetic/avatar/watch_chain/work
```

It printed background [188.0, 187.0, 187.0], pockets 1274 px, crop (115, 0, 1146, 1031), portrait 768. The script is `ruby_drop_cleanup.py` of batch A unchanged. It writes the master and a square 768 x 768 portrait crop (`work/watch_chain_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | As in batch A. The loop script puts a circular alpha (soft edge of 2.25 px at 768, 0.56 px at 192) on every frame, because the portrait is shown in a circle (`docs/art-direction.md` 5.1); the class canvas is open (O-6) and D-1 sets 192 x 192 |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Yes | Written by the loop script into `work/` (table below); the ripples are warps of the base layer and need no layer of their own |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen in the concept, the master or the frames |
| 5. Silhouette and smallest-size tests | In part | The agent looked at every frame at 192 px and enlarged on a dark backdrop; face, expression and the moving part read. The smallest display size and frame band of an avatar are open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-16 (UTC) by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/avatar/watch_chain/work/watch_chain_loop.py assets-src/cosmetic/avatar/watch_chain/concept assets-src/cosmetic/avatar/watch_chain/work assets-src/cosmetic/avatar/watch_chain/export
```

It printed `watch_chain_loop: blink edit registered (tx, ty, scale, angle) = (0.0703, 0.1546, 0.9998, -0.0001), residual 0.00067 -> 0.00066, eye colour offset [-7.2, 0.0, 3.3], parts 0, ripples 1, filled 0 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run into empty folders wrote files byte-identical to each other and to the committed ones.

The script is the batch A loop script (`ruby_drop_loop.py`) with this item's geometry and these additions, the same for every avatar of this batch: the concept file names are constants; a part can open and close (`flap`, a horizontal scale about its pivot); `RIPPLES` add a smooth sideways wave to the warp inside a feathered polygon, so cloth, ribbon and chain move without a cut-out layer and without seams; a hole behind a part is cleared where the fill estimate is less than half covered, which keeps a filled hole from leaving a smear over the background; a glint is clipped to the bright pixels of the portrait and its peak opacity is a constant (`GLINT_PEAK`, 0.55 as in batch A unless the item says otherwise); and the registration region is clipped to the image when the crop reaches past its edge.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient ripple | The lower loop of the watch chain, between the pin and the button, so both ends stay fixed: inside a polygon feathered by Gaussian sigma 8 (768 scale), the pixels shift sideways by 3 x (sin(2 pi y / 400 - 1 t) - sin(2 pi y / 400)) px at 768, faded in over 60 px below y 600, so the wave runs down once per loop and the attached top stays still | Warp only, so no seams and no hole to fill; the polygon ends before the circle edge |
| Ambient glint | Nine glints in a row along the chain, each four frames long and one frame after the one before, so one highlight travels from the pin down the loop and up to the button: (362, 615), ellipse 30 x 30 px at 768, frames 9, 10, 11, 12; (385, 645), ellipse 30 x 30 px at 768, frames 10, 11, 12, 13; (415, 675), ellipse 30 x 30 px at 768, frames 11, 12, 13, 14; (450, 695), ellipse 30 x 30 px at 768, frames 12, 13, 14, 15; (485, 702), ellipse 30 x 30 px at 768, frames 13, 14, 15, 16; (530, 695), ellipse 30 x 30 px at 768, frames 14, 15, 16, 1; (560, 680), ellipse 30 x 30 px at 768, frames 15, 16, 1, 2; (585, 655), ellipse 30 x 30 px at 768, frames 16, 1, 2, 3; (600, 625), ellipse 30 x 30 px at 768, frames 1, 2, 3, 4 | A painted highlight: a soft streak that crosses the chain links, clipped to its ellipse and to the bright pixels of the portrait (mean channel value faded in from 0.3 to 0.5), warm ivory at up to 1.0 opacity, normal blending, no halo outside the ellipse (not a glow, `docs/art-direction.md` section 8) |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 2.34 of 255, against 2.10 on average and 2.53 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `watch_chain: background [188.0, 187.0, 187.0], pockets 1274 px, crop (70, -189, 1191, 932), portrait 768`
- Crop `(115, 0, 1146, 1031)` to `(70, -189, 1191, 932)` on the concept
- Scale k = 0.9197. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- Printed by the loop: `watch_chain_loop: blink edit registered (tx, ty, scale, angle) = (0.0691, 0.1485, 0.9998, -0.0001), residual 0.00060 -> 0.00059, eye colour offset [-7.1, 0.1, 3.4], parts 0, ripples 1, filled 0 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.000 before, 0.172 after; highest over the 16 loop frames 0.000 before, 0.167 after
- The concept cuts the subject at its top edge (42 px wide at the first opaque row of the master). That cut lay outside the circle before; it now shows as a short straight edge at the head top inside the circle, under the frame's top band in the four frames checked. An artist paints it over (RC-5)

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/watch_chain/work/watch_chain_cleanup.py watch_chain assets-src/cosmetic/avatar/watch_chain/concept/20261115-02.png assets-src/cosmetic/avatar/watch_chain/work
python -I assets-src/cosmetic/avatar/watch_chain/work/watch_chain_loop.py assets-src/cosmetic/avatar/watch_chain/concept assets-src/cosmetic/avatar/watch_chain/work assets-src/cosmetic/avatar/watch_chain/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 24 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/watch_chain` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `19a33772d60bb1050142be739d11c36464709a79899b5d1652b63021c54af54b` |
| export/anim/0002.png | `51d2d4bc04afb27aee56df2fe6ab740d7e00d2e3a8d7276932015cf81b1800d9` |
| export/anim/0003.png | `a70a69cc301794119067d8e62f5448d77cc24eba11730af8858340a43daaf7ac` |
| export/anim/0004.png | `3cde7a05be086f3c34ee5a4e217cad5db052da4bc0febb18b117d4c2593966e1` |
| export/anim/0005.png | `93a56d6145bfdb859d21938d5fa2e797532fbf7d482401e3fe485bd894acded1` |
| export/anim/0006.png | `f285f1e1b52d1275900c93e3483699ee2ce9a95416318b0a6cec82fd8f56ce35` |
| export/anim/0007.png | `ae25634d92e5487aa9a0a808fe320c17f2e1f52dfba38371632c738ffba73fae` |
| export/anim/0008.png | `c11a6c48960c284d1f551438e1d6368e2731f72cc2835eb0b478bf68985c1a60` |
| export/anim/0009.png | `3164f81bea09ed443ff89f8f8905eb4b9ea1bffead65323d163c11981a4554ce` |
| export/anim/0010.png | `bf822ae33d420e8eebeef0fcc3f7473d36289f051dc58fba1269bc7f6dfd140b` |
| export/anim/0011.png | `95d04b9100c3dd9dca85c5b193902f096dd9526546ecdae7c96e74b20991c543` |
| export/anim/0012.png | `7ae5c81993e0d2e29fa9913d3527d29d4415844e07e099c626904125817319a9` |
| export/anim/0013.png | `09346d01ebda7580893c0d143874ee359f94d19965f4b7de53f9fbcecdac2781` |
| export/anim/0014.png | `f363f0b024ded597975cc526244a917682ebb2149a89b454fe5071b52bc291b1` |
| export/anim/0015.png | `fdb0ef3a0398a4907a416caa0437fab2ccceeb566f224054fe545cfeec903cae` |
| export/anim/0016.png | `95c7a86d52fb873a986ecfc878bd8dc6773b72607e283368e58305d717ff7592` |
| export/static.png | `3164f81bea09ed443ff89f8f8905eb4b9ea1bffead65323d163c11981a4554ce` |
| work/watch_chain_base.png | `e10fe83a61606ee3df7de372b30782b99d532ed3317de6b97702181dc25f8fb9` |
| work/watch_chain_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` |
| work/watch_chain_eyes_closed.png | `167495ec2666def4381b13673a52ed4a82315d5c5cc6534cfdf152b1e36796e5` |
| work/watch_chain_eyes_open.png | `67cb3e8fb5131b9962d7558790e6b9ac2f0b8dcdde927b633ad820b1b5aa50f2` |
| work/watch_chain_head.png | `57c14c3e612e4b33631a59aff1c8cd73f51dd1fee450eeb65c792eb8995cbece` |
| work/watch_chain_loop.py | `d894eea4a69870df910955a9fe48370695028001bd933ddbfa9d8dc4b4c729e5` |
| work/watch_chain_portrait.png | `0c2287c9a850483b7735752f966d77a51caccd39363e4523f1ad93a220661299` |
| work/watch_chain_torso.png | `d1d8bc71af6afc90481361c28547d8298b10caa265ce126ee79e385415290132` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-02.png | `b19b60fcea1089bf1e402b3312ed974f00429d96e95d175dac7a9681734a1c4c` | 1312 x 1199 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e047-8a2a-7902-ba76-a376cf238005` |
| concept/20261115-03.png | `c99a48e4ae1e385b4c2a42f71ed9b6469f31b1cc3bfb1493b0ce26ed6fe4df57` | 1312 x 1199 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e056-48ba-74b2-9a80-fb915c09af49` |
| work/watch_chain_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_cleanup.py` |
| work/watch_chain_loop.py | `ed32fcec53d495157f0488dc09e32604ba78fa2a8ac4093247dbd7b9c7c19e48` | W-4 and W-5 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_loop.py`; the same script for every avatar of this batch with the item's geometry |
| work/watch_chain_master.png | `c754e36182adcad09c5d1a75edd0bfaf90786d807ce747d3bc79a13af14b7f09` | 1312 x 1199 RGBA, the cut-out master | `watch_chain_cleanup.py` |
| work/watch_chain_base.png | `718edb96e8fc59e314e895851f437aedd5df85c910c5dac389b2d09c415bfe99` | The portrait with the moving parts taken out (filled or cleared) | `watch_chain_loop.py` |
| work/watch_chain_eyes_closed.png | `74e5fb5ea5950bca245c13bd35b4dba9771a738e28438b38dc45a69f46ae9380` | Eye region, closed (aligned edit) through the same mask | `watch_chain_loop.py` |
| work/watch_chain_eyes_open.png | `4d5dde646f8f50d22e7ef92b13592e284f1d6a186eecdedf671bbe7e209e3eba` | Eye region, open (canonical concept) through the feathered eye mask | `watch_chain_loop.py` |
| work/watch_chain_head.png | `51437506b2b62b84b9099e6eca0453689490f3675a2f1c09f6a08dd34ab744bc` | Head and hair: the portrait weighted by the head field of the warp | `watch_chain_loop.py` |
| work/watch_chain_portrait.png | `f1ba65a07e83c38e370292d26dafb67d64ad0de4c49c599f8df9b79c59c08316` | 768 x 768 RGBA portrait crop, no circle | `watch_chain_cleanup.py` |
| work/watch_chain_torso.png | `109f14d342ce0ce366969d10bda7842d49b15e9f45be01e42d82ce685dacda36` | Torso and costume: the portrait weighted by one minus the head field | `watch_chain_loop.py` |
| export/static.png | `193346e46fcdf9b0f6ca24892d04a66f96f8cc533d249c80eaba03e787ce4018` | 192 x 192 RGBA static, loop frame 9 | `watch_chain_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `watch_chain_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png (rejected, not committed) | P-3 with the style key and the first identity, quoted below. Sent 17:24:44 UTC; Codex session `01a1e03d-6d0e-7b30-95c9-bdb1ac770eef` |
| 2026-11-15 | concept/20261115-02.png | P-3 with the style key, quoted below. Sent 17:35:46 UTC; `c2pa.created` 2026-11-15T17:36:32Z |
| 2026-11-15 | concept/20261115-03.png | Edit of `concept/20261115-02.png` (attached): eyes closed, everything else identical, quoted below. Sent 17:51:53 UTC; `c2pa.created` 2026-11-15T17:52:53Z |
| 2026-11-16 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/watch_chain_base.png, work/watch_chain_cleanup.py, work/watch_chain_eyes_closed.png, work/watch_chain_eyes_open.png, work/watch_chain_head.png, work/watch_chain_loop.py, work/watch_chain_portrait.png, work/watch_chain_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(115, 0, 1146, 1031)` to `(70, -189, 1191, 932)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261115-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his fifties with light olive skin, thick wavy grey hair combed back and a short trimmed grey beard, a chocolate-brown velvet waistcoat over a cream shirt with a rust-orange silk tie, and a polished brass watch chain looping from a buttonhole high on the waistcoat just below the collar, clearly visible across the upper chest, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-02.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261115-03.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his fifties with light olive skin, thick wavy grey hair combed back and a short trimmed grey beard, a chocolate-brown velvet waistcoat over a cream shirt with a rust-orange silk tie, and a polished brass watch chain looping from a buttonhole high on the waistcoat just below the collar, clearly visible across the upper chest, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261115-02.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261115-03.png. Do not use any API key.`

Prompt of the rejected `concept/20261115-01.png`, verbatim (the identity differs only in the chain):

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his fifties with light olive skin, thick wavy grey hair combed back and a short trimmed grey beard, a chocolate-brown velvet waistcoat over a cream shirt with a rust-orange silk tie, and a polished brass pocket-watch chain draped high across the waistcoat, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.
