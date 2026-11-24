# avatar.magenta_hoops

Animated avatar of set 1 (#171), batch B, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 and 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, final runs on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: `concept/20261115-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261115-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a woman in her thirties with deep dark brown skin, short tapered natural black hair that leaves her ears visible, large polished gold hoop earrings each with a small amethyst drop, and a magenta silk jacket with a stand collar and gold piping"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt; the ears and hoops are fully visible. Not asked for: an ivory shirt collar and gold leaf embroidery on the jacket.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.05, -0.17 px, scale 1.0007 and rotation -0.0004 rad (mean squared grey difference outside the eyes 0.00068 before, 0.00065 after), and the eye patch was offset by [5.9, 5.5, 5.6] in RGB to meet the canonical skin. The changes outside the eye mask are small and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, source `shop:offer.avatar_magenta_hoops`. The offer id is a placeholder until the shop record exists (V-15). One ambient motion, the two hoops. Animated avatars stay at `epic` and above as in batch A; batch B proposes 5 epic, 4 legendary, 1 mythic, the same split as batch A.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 1 animated detail (hoops (pair)), so `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signatures were not validated with a C2PA tool.

| Field | `concept/20261115-01.png` | `concept/20261115-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T17:30:43Z | `c2pa.created`, 2026-11-15T17:46:37Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 17:29:50 | 2026-11-15 17:45:31 |
| Codex session | `01a1e042-1bde-7393-a385-b4e715ec2db8` | `01a1e050-76a4-7511-b3ff-31a4fdffb2ea` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/magenta_hoops/work/magenta_hoops_cleanup.py magenta_hoops assets-src/cosmetic/avatar/magenta_hoops/concept/20261115-01.png assets-src/cosmetic/avatar/magenta_hoops/work
```

It printed background [208.0, 207.0, 207.0], pockets 6895 px, crop (121, 0, 1152, 1031), portrait 768. The script is `ruby_drop_cleanup.py` of batch A unchanged. It writes the master and a square 768 x 768 portrait crop (`work/magenta_hoops_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

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
python -I assets-src/cosmetic/avatar/magenta_hoops/work/magenta_hoops_loop.py assets-src/cosmetic/avatar/magenta_hoops/concept assets-src/cosmetic/avatar/magenta_hoops/work assets-src/cosmetic/avatar/magenta_hoops/export
```

It printed `magenta_hoops_loop: blink edit registered (tx, ty, scale, angle) = (0.0487, -0.1746, 1.0007, -0.0004), residual 0.00068 -> 0.00065, eye colour offset [5.9, 5.5, 5.6], parts 2, ripples 0, filled 11316 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run into empty folders wrote files byte-identical to each other and to the committed ones.

The script is the batch A loop script (`ruby_drop_loop.py`) with this item's geometry and these additions, the same for every avatar of this batch: the concept file names are constants; a part can open and close (`flap`, a horizontal scale about its pivot); `RIPPLES` add a smooth sideways wave to the warp inside a feathered polygon, so cloth, ribbon and chain move without a cut-out layer and without seams; a hole behind a part is cleared where the fill estimate is less than half covered, which keeps a filled hole from leaving a smear over the background; a glint is clipped to the bright pixels of the portrait and its peak opacity is a constant (`GLINT_PEAK`, 0.55 as in batch A unless the item says otherwise); and the registration region is clipped to the image when the crop reaches past its edge.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `hoop_left` (layer `work/magenta_hoops_hoop_left.png`, pivot (276, 388)) turns about its pivot by 6.0 degrees x (sin(lag) - sin(t + lag)), lag 0.4, the turn growing from 0 at the pivot to full at 40 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | A small part well inside the circle, so its motion never meets the circle edge |
| Ambient part | `hoop_right` (layer `work/magenta_hoops_hoop_right.png`, pivot (556, 366)) turns about its pivot by 6.0 degrees x (sin(lag) - sin(t + lag)), lag 0.9, the turn growing from 0 at the pivot to full at 40 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | A small part well inside the circle, so its motion never meets the circle edge |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 2.23 of 255, against 1.96 on average and 2.37 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `magenta_hoops: background [208.0, 207.0, 207.0], pockets 6895 px, crop (76, -125, 1197, 996), portrait 768`
- Crop `(121, 0, 1152, 1031)` to `(76, -125, 1197, 996)` on the concept
- Scale k = 0.9197. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- Printed by the loop: `magenta_hoops_loop: blink edit registered (tx, ty, scale, angle) = (0.0497, -0.1873, 1.0007, -0.0004), residual 0.00060 -> 0.00057, eye colour offset [6.0, 5.5, 5.6], parts 2, ripples 0, filled 9705 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.062 before, 0.172 after; highest over the 16 loop frames 0.057 before, 0.167 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/magenta_hoops/work/magenta_hoops_cleanup.py magenta_hoops assets-src/cosmetic/avatar/magenta_hoops/concept/20261115-01.png assets-src/cosmetic/avatar/magenta_hoops/work
python -I assets-src/cosmetic/avatar/magenta_hoops/work/magenta_hoops_loop.py assets-src/cosmetic/avatar/magenta_hoops/concept assets-src/cosmetic/avatar/magenta_hoops/work assets-src/cosmetic/avatar/magenta_hoops/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 26 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/magenta_hoops` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `3753868a09f29ce2390f72fd64f0ae29f9540b3a89e257daadc45304251c7a98` |
| export/anim/0002.png | `4c8e4268a1f538cca70ac3f7872ff93a0cdc19ec1d547a349115ef870680a40a` |
| export/anim/0003.png | `a523101a2417f3e7d5ef38b5728b346821599613e65e42efc996c7cded01ab7c` |
| export/anim/0004.png | `61c182ea7a376a82a6b4181e236d36d7b1557c32c344b41ca8b4789a35593a34` |
| export/anim/0005.png | `dacd77756b47bee10ff31d17568040eb003457bbb9d2294054270297d1a64d46` |
| export/anim/0006.png | `8e3b65964bd96a94b93630f73cac2187b8b7cad239cddab7efe17500589f51bb` |
| export/anim/0007.png | `75e214f33ed88e5d1416b7698b962a11df7f56a7dd2246e43cfdda7e10d1c1e6` |
| export/anim/0008.png | `13ce5cdfce62a6982bf35a2cc4958140ec817a7228b4f347f1c68e9a6ab90f60` |
| export/anim/0009.png | `80238e3053b2b7c75d82e1427b0dc774dd6c5fc151f07f62a482395dc44cb1ee` |
| export/anim/0010.png | `34f88a2c672ef9936ce91deff7849713c715d795934580eb3a7ec74f9abfef1f` |
| export/anim/0011.png | `fc6bf6784607c8bf2a75420837b8511ea346b8473b8fd10f58bc65bc07cf78bb` |
| export/anim/0012.png | `06cdfa51f2ce768c9f3f1ce4e4645cc5516ff9d47ef0ead3a1630b5a0fdd194f` |
| export/anim/0013.png | `4d747975bffbf326a9cd4e59d2c793f4f3e3ee608f1076a61e274db88f9eb258` |
| export/anim/0014.png | `3491975992b4b1305c30d88fdc0191ab3e3c9b8ddcba08450eb49f8c03050d10` |
| export/anim/0015.png | `568bcb087834ffaf45d1046817f79bfdcceb9facbfeb1f09ae7e8814caca982c` |
| export/anim/0016.png | `71861f7118e8e189eeabd9dc89b957efcdadeaa1c4df2aaa320d4de67dbf60ab` |
| export/static.png | `80238e3053b2b7c75d82e1427b0dc774dd6c5fc151f07f62a482395dc44cb1ee` |
| work/magenta_hoops_base.png | `eb90710e99767ce76ffd97b7b3b930e0d00901706d8d47896de6e46937eb5cf6` |
| work/magenta_hoops_cleanup.py | `d699aa8b38c6c66da5f607232e08ae8c3bd0032a3517e9a07c89d4ef7d5761ab` |
| work/magenta_hoops_eyes_closed.png | `2c0ae3737aa508220aee033ffcce25a4801e465798aa5c673b1426d88544a231` |
| work/magenta_hoops_eyes_open.png | `329f3a7b77064a519f9effe5b4671f356f659ce43267ad4c07eebb7486b6b95a` |
| work/magenta_hoops_head.png | `6a2b60f83467aab147684679a54c061a472e6adc381dd2b2a4c948b79394036e` |
| work/magenta_hoops_hoop_left.png | `ba377aed91af9d5e5911da360351d0597aafae9528cc50df71c750ca0fd17bbb` |
| work/magenta_hoops_hoop_right.png | `c47db4a48f0ce76c6f24d3600e38c8edc09651d4174a5884d90106a92b696264` |
| work/magenta_hoops_loop.py | `01f94f8ffdb03a109a70d2eeb4155ac8868f562d56ce4606c87a9cd7a1ff7c89` |
| work/magenta_hoops_portrait.png | `e73216bd1acc24a49bfe11348ca55b8ee73f6272a1054844feb53d104490bf72` |
| work/magenta_hoops_torso.png | `d9d6dfaafde900f375f208cc7d390305d3d16d5a337e1d32fce9e88709b330bd` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `819e601a2631813cca9cfa2eef5bed778128ed6f13b2c697a45a0d8a595b6a11` | 1312 x 1199 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e042-1bde-7393-a385-b4e715ec2db8` |
| concept/20261115-02.png | `8eb69d62e258c58508a6a433260d4184c7321daa92b3f09998529b1fc818760e` | 1312 x 1199 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e050-76a4-7511-b3ff-31a4fdffb2ea` |
| work/magenta_hoops_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_cleanup.py` |
| work/magenta_hoops_loop.py | `86488c7b904155ab960b7f481ce57608b3de514141d53ff24154e5d3b11f8846` | W-4 and W-5 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_loop.py`; the same script for every avatar of this batch with the item's geometry |
| work/magenta_hoops_master.png | `4d767fa7e3639f399bf85e5b6f3ad6057283267990c1848a10f4f3e5be6ae5e5` | 1312 x 1199 RGBA, the cut-out master | `magenta_hoops_cleanup.py` |
| work/magenta_hoops_base.png | `0832c3df07f7e7774c3b34798aa04feb77f06499edf3b807170e8392bae65d04` | The portrait with the moving parts taken out (filled or cleared) | `magenta_hoops_loop.py` |
| work/magenta_hoops_eyes_closed.png | `63ecd7bafec8b76a5c688bf728b19f38f71779f181a267efb5fa7ee146db2e85` | Eye region, closed (aligned edit) through the same mask | `magenta_hoops_loop.py` |
| work/magenta_hoops_eyes_open.png | `687d23581f8c9e2c0747f9c6909b1efe14607846ee35e3c5425ff810c426f535` | Eye region, open (canonical concept) through the feathered eye mask | `magenta_hoops_loop.py` |
| work/magenta_hoops_head.png | `0ea4176c145a98bfcaaa51fa5dbed43159146d822246f7cee0931993a5cd9af5` | Head and hair: the portrait weighted by the head field of the warp | `magenta_hoops_loop.py` |
| work/magenta_hoops_hoop_left.png | `f24694ca4fbaa42c5cc3e6163c1ccfc6349783e063511a7e49fe95d6838ba066` | Moving part `hoop_left`: the portrait through its feathered polygon | `magenta_hoops_loop.py` |
| work/magenta_hoops_hoop_right.png | `efe5ff75ed2f79a2c19d5a4b92a40563d92896681e30a88526fdb17e7db59254` | Moving part `hoop_right`: the portrait through its feathered polygon | `magenta_hoops_loop.py` |
| work/magenta_hoops_portrait.png | `a48078ca266a72ad070ae87598a8b8a7fc9d71535b71990b2d98f20dd81cd0eb` | 768 x 768 RGBA portrait crop, no circle | `magenta_hoops_cleanup.py` |
| work/magenta_hoops_torso.png | `d2183b61a1930df026742bea61d7d35386d25f6f8d6d9081c2f8a6e9c6e0d758` | Torso and costume: the portrait weighted by one minus the head field | `magenta_hoops_loop.py` |
| export/static.png | `513d3a18535d9063e046c9b59b6f88b10378fe385488cd19745a86620ec6222a` | 192 x 192 RGBA static, loop frame 9 | `magenta_hoops_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `magenta_hoops_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted below. Sent 17:29:50 UTC; `c2pa.created` 2026-11-15T17:30:43Z |
| 2026-11-15 | concept/20261115-02.png | Edit of `concept/20261115-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 17:45:31 UTC; `c2pa.created` 2026-11-15T17:46:37Z |
| 2026-11-16 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/magenta_hoops_base.png, work/magenta_hoops_cleanup.py, work/magenta_hoops_eyes_closed.png, work/magenta_hoops_eyes_open.png, work/magenta_hoops_head.png, work/magenta_hoops_hoop_left.png, work/magenta_hoops_hoop_right.png, work/magenta_hoops_loop.py, work/magenta_hoops_portrait.png, work/magenta_hoops_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.17, crop `(121, 0, 1152, 1031)` to `(76, -125, 1197, 996)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her thirties with deep dark brown skin, short tapered natural black hair that leaves her ears visible, large polished gold hoop earrings each with a small amethyst drop, and a magenta silk jacket with a stand collar and gold piping, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261115-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her thirties with deep dark brown skin, short tapered natural black hair that leaves her ears visible, large polished gold hoop earrings each with a small amethyst drop, and a magenta silk jacket with a stand collar and gold piping, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261115-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261115-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.
