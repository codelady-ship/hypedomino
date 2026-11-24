# avatar.lemon_spaniel

Animated avatar of set 1 (#171), batch C, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-16 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, final runs on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-16 for the Plus plan and records the date here
- Canonical concept: concept/20261116-01.png
- Inputs uploaded to the generator: `concept/20261116-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261116-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a man in his fifties with light brown skin, wavy dark hair greying at the temples and a short beard, a dove-grey morning coat with a lemon-yellow silk cravat, holding a small cream spaniel puppy with long floppy ears against his chest, the whole puppy fully visible"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt: he holds a cream spaniel puppy with long ears against his chest. Not asked for: a gold collar with a yellow jewel tag on the puppy, a jewelled lapel brooch with chains, a lemon pocket square and a jewelled cravat pin.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.15, 0.12 px, scale 0.9999 and rotation 0.0000 rad (mean squared grey difference outside the eyes 0.00124 before, 0.00122 after), and the eye patch was offset by [-3.4, -0.4, 3.1] in RGB to meet the canonical skin. The changes outside the eye mask are small and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `mythic`, source `shop:offer.avatar_lemon_spaniel`. The offer id is a placeholder until the shop record exists (V-15). A companion in his arms: the puppy sways, its long ears swing and its collar tag swings. Animated avatars stay at `epic` and above as in batches A and B; batch C proposes 5 epic, 3 legendary, 2 mythic, which brings the 30 animated avatars of set 1 to 15 epic, 11 legendary and 4 mythic.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 3 animated details (puppy sway and ears (a creature counts 2), collar tag), so `mythic`, was `legendary`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-16; the signatures were not validated with a C2PA tool.

| Field | `concept/20261116-01.png` | `concept/20261116-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-16T08:37:42Z | `c2pa.created`, 2026-11-16T08:53:19Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-16 08:36:57 | 2026-11-16 08:52:19 |
| Codex session | `01a1e380-9542-7220-abf3-33c368febe22` | `01a1e38e-a77d-7471-a4e6-5aa5355182db` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked by a byte comparison). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/lemon_spaniel/work/lemon_spaniel_cleanup.py lemon_spaniel assets-src/cosmetic/avatar/lemon_spaniel/concept/20261116-01.png assets-src/cosmetic/avatar/lemon_spaniel/work
```

It printed background [208.0, 207.0, 207.0], pockets 142 px, crop (72, 0, 1023, 951), portrait 768. The script is `cornflower_ribbon_cleanup.py` of batch B unchanged (the batch A clean-up of `avatar.ruby_drop` with its 768 crop, grey-pocket rule and edge re-matte). It writes the master and a square 768 x 768 portrait crop (`work/lemon_spaniel_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | As in batches A and B. The loop script puts a circular alpha (soft edge of 2.25 px at 768, 0.56 px at 192) on every frame, because the portrait is shown in a circle (`docs/art-direction.md` 5.1); the class canvas is open (O-6) and D-1 sets 192 x 192 |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Yes | Written by the loop script into `work/` (table below); the ripples are warps of the base layer and need no layer of their own |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen in the concept, the master or the frames |
| 5. Silhouette and smallest-size tests | In part | The agent looked at every frame at 192 px and enlarged on a dark backdrop; face, expression and the moving part read. The smallest display size and frame band of an avatar are open (#168, O-6), so the test is not final |

## Animation (W-4) and export (W-5)

Run on 2026-11-16 (UTC) by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/avatar/lemon_spaniel/work/lemon_spaniel_loop.py assets-src/cosmetic/avatar/lemon_spaniel/concept assets-src/cosmetic/avatar/lemon_spaniel/work assets-src/cosmetic/avatar/lemon_spaniel/export
```

It printed `lemon_spaniel_loop: blink edit registered (tx, ty, scale, angle) = (0.1495, 0.1199, 0.9999, 0.0000), residual 0.00124 -> 0.00122, eye colour offset [-3.4, -0.4, 3.1], parts 1, ripples 3, filled 3148 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run of the clean-up and the loop into empty folders wrote files byte-identical to each other and to the committed ones.

The script is the batch B loop script (`cornflower_ribbon_loop.py`, which carries the batch A rig of `ruby_drop_loop.py` with `flap`, `RIPPLES`, the clearing hole fill, clipped glints and the clipped registration region) with this item's geometry and one addition, the same for every avatar of batch C: a part can spin (`spin`, whole turns per loop added to its turn angle, so a full turn ends where it started). Parts without `spin` move exactly as in batch B.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `tag` (layer `work/lemon_spaniel_tag.png`, pivot (346, 666)) turns about its pivot by 10.0 degrees x (sin(lag) - sin(t + lag)), lag 0.6, the turn growing from 0 at the pivot to full at 30 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | A small part well inside the circle, so its motion never meets the circle edge |
| Ambient ripple | The whole puppy, with a wave so long that it reads as a sideways sway of head and body: inside a polygon feathered by Gaussian sigma 18 (768 scale), the pixels shift sideways by 5.0 x (sin(2 pi y / 1600 - 1 t) - sin(2 pi y / 1600)) px at 768, faded in over 60 px below y 420, so the wave runs down once per loop and the attached top stays still | Warp only, so no seams; the wide feather lets the coat around the puppy follow softly |
| Ambient ripple | The puppy's left ear (the viewer's left): inside a polygon feathered by Gaussian sigma 10 (768 scale), the pixels shift sideways by 8.0 x (sin(2 pi y / 200 - 1 t) - sin(2 pi y / 200)) px at 768, faded in over 120 px below y 560, so the wave runs down once per loop and the attached top stays still | Warp only, so no seams and no hole to fill; the attached top of the ear stays with the head |
| Ambient ripple | The puppy's right ear (the viewer's right): inside a polygon feathered by Gaussian sigma 10 (768 scale), the pixels shift sideways by 8.0 x (sin(2 pi y / 200 - 1 t) - sin(2 pi y / 200)) px at 768, faded in over 120 px below y 560, so the wave runs down once per loop and the attached top stays still | Warp only, so no seams and no hole to fill; the attached top of the ear stays with the head |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 2.54 of 255, against 2.59 on average and 3.19 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.12, not the 0.17 of the other avatars: at 0.17 the tag layer was 4.4% outside the circle; at 0.12 it is 0.0%
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `lemon_spaniel: background [208.0, 207.0, 207.0], pockets 142 px, crop (30, -107, 1064, 927), portrait 768`
- Crop `(72, 0, 1023, 951)` to `(30, -107, 1064, 927)` on the concept
- Scale k = 0.9197. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- Printed by the loop: `lemon_spaniel_loop: blink edit registered (tx, ty, scale, angle) = (0.1379, 0.1293, 0.9999, -0.0000), residual 0.00115 -> 0.00113, eye colour offset [-3.7, -0.5, 3.0], parts 1, ripples 3, filled 2756 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.016 before, 0.120 after; highest over the 16 loop frames 0.010 before, 0.115 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/lemon_spaniel/work/lemon_spaniel_cleanup.py lemon_spaniel assets-src/cosmetic/avatar/lemon_spaniel/concept/20261116-01.png assets-src/cosmetic/avatar/lemon_spaniel/work
python -I assets-src/cosmetic/avatar/lemon_spaniel/work/lemon_spaniel_loop.py assets-src/cosmetic/avatar/lemon_spaniel/concept assets-src/cosmetic/avatar/lemon_spaniel/work assets-src/cosmetic/avatar/lemon_spaniel/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 25 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/lemon_spaniel` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `f30968ae3d5f43559809f2dfed565ada139d12238cde84e453ef6490a5343044` |
| export/anim/0002.png | `761d3b6f8695dba04ba8df40a3c5d399ae144fc5b8c5b4db916eae3d1ed49600` |
| export/anim/0003.png | `0b5912720ba6777c627688fa4f15a054f3256adec459ac9aaa6170891b2f316b` |
| export/anim/0004.png | `c15bf1707c47684b125ba5f69ac441044d17258edb20bc9711f0e1b48d73a1f9` |
| export/anim/0005.png | `e344e51621bf82b9d70f6d97b3fad6c847b7487fd729031c89db6ebd440907a0` |
| export/anim/0006.png | `6719331255cb2972eb99ef52fee4273b255d64890aa57340fc7dcd16a8d494a2` |
| export/anim/0007.png | `486028e25ae17dc58b00d9b97416386934a8863fd971d19ff01d4b2f3d1b1bab` |
| export/anim/0008.png | `0ce6d5aa254ae91e3a555be431e691e6b08ece70a90d9a117f747285637505bc` |
| export/anim/0009.png | `e9678b3aaf557681c585d96f3a6deff10872279df052274f52b659326f6bab70` |
| export/anim/0010.png | `9fe22c14c4ea9711b4c4c4767633e8e25213131141ce51de2deab52a8bfcb5cb` |
| export/anim/0011.png | `02b11dbf71cbd0b879328266ec33783b18f483f079deea32cf167d6e1647ab81` |
| export/anim/0012.png | `4b46296ca138b7f421ad4fda268e04cac82f25814c600cfd3cff0704852f1e29` |
| export/anim/0013.png | `47001a72fc6f0fdf4f1e6b99e8c39053f66ed1c95a6950aac363b4f687a809ed` |
| export/anim/0014.png | `22b3cfb412bd50cee43661ea34a6c4d6ac268861d78be06a86e320dbeed6924e` |
| export/anim/0015.png | `cd09aab4df2ae70c7d6b98d58fc7125980974d4a54b1577231e8207580305a41` |
| export/anim/0016.png | `d48264bf11f4bc883a8e4c0b7f492d0bbe4fcca878de0e069c19ccf6d9821a06` |
| export/static.png | `e9678b3aaf557681c585d96f3a6deff10872279df052274f52b659326f6bab70` |
| work/lemon_spaniel_base.png | `f9f9f3ae19c34297748a4b758ad71511462cb1a4706461ba76a6ca16f778dbfa` |
| work/lemon_spaniel_cleanup.py | `18a09c5cb88e9deea6bacde53bf5efc038ab7d778bc75123651cf1298b91226b` |
| work/lemon_spaniel_eyes_closed.png | `2d8c096135524560d6f7bba1b4f17a53943e90924cbbbcd080db462c6b89c9f2` |
| work/lemon_spaniel_eyes_open.png | `b17bd7d8a60eb3aca6559f6c9d61c70fd8ebea90244ba9019638fa6d73ac03de` |
| work/lemon_spaniel_head.png | `250f3f72cdf000edf1df0d27e725be40b7fcf10d64d88e2d5efc022bf92d9fc3` |
| work/lemon_spaniel_loop.py | `1b3d8e5212f12c900904b991c6a6211e64696b40f144277a7ac951da26ef4e8d` |
| work/lemon_spaniel_portrait.png | `792db1c16da369dba8a6161a6c4c24a3a68440eb0b36a03d7f268b0d9f618b84` |
| work/lemon_spaniel_tag.png | `1ae7a6887327b5dc3ee7f4b1f8aa073729bbb3cb156172a3832e6b309bc526db` |
| work/lemon_spaniel_torso.png | `dac5142487cc5b92af1f72a9dcf13604ef0d38731a130e2334adea1b1ec27da9` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261116-01.png | `d932bcfb971e9e7190f59dff0ef4089eb18c2e2c71060ded360c34559dee03ef` | 1106 x 1422 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e380-9542-7220-abf3-33c368febe22` |
| concept/20261116-02.png | `a8216c23570a9a7c1845de96271c787262a6b5a0cdd052f625fd8bd0442fa41c` | 1106 x 1422 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e38e-a77d-7471-a4e6-5aa5355182db` |
| work/lemon_spaniel_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-16, from `cornflower_ribbon_cleanup.py` |
| work/lemon_spaniel_loop.py | `32825b4998de3aee63dca0638b4dc3f5ae144b34926798a583aa477513ffc1d3` | W-4 and W-5 script | Written by Claude Code, 2026-11-16, from `cornflower_ribbon_loop.py`; the same script for every avatar of this batch with the item's geometry |
| work/lemon_spaniel_master.png | `83773034dd8281a1e3b6966c10a90b7ff2c81161af0e9e8e7285ae2a3c4e7536` | 1106 x 1422 RGBA, the cut-out master | `lemon_spaniel_cleanup.py` |
| work/lemon_spaniel_base.png | `06fd8be40bfde729f56916da1202c329e80540653d15895cd3773cceda0b5e29` | The portrait with the moving parts taken out (filled or cleared) | `lemon_spaniel_loop.py` |
| work/lemon_spaniel_eyes_closed.png | `e1a2062780d6732a3b4f93a3350f57dbdf97a4a7068063ff67987c858feca39f` | Eye region, closed (aligned edit) through the same mask | `lemon_spaniel_loop.py` |
| work/lemon_spaniel_eyes_open.png | `47534ff579bfc85849d410532c8cf2e51b47696da24d2d0617deee90d06513e6` | Eye region, open (canonical concept) through the feathered eye mask | `lemon_spaniel_loop.py` |
| work/lemon_spaniel_head.png | `1b727cdb82740f4bbd1072d54b1d22f796a6b5848861e0ae370e5a5a1738f31e` | Head and hair: the portrait weighted by the head field of the warp | `lemon_spaniel_loop.py` |
| work/lemon_spaniel_portrait.png | `125d65b0a191883049f490de64a1ec341c4716a1631d5a18dd36d3b4c14b4a98` | 768 x 768 RGBA portrait crop, no circle | `lemon_spaniel_cleanup.py` |
| work/lemon_spaniel_tag.png | `8ea754791326b6ee89cfc6c17c679db733d76f234919d5eae152ce1dee384d25` | Moving part `tag`: the portrait through its feathered polygon | `lemon_spaniel_loop.py` |
| work/lemon_spaniel_torso.png | `a263b0f06a9aa0467aebc6a78c3cb79e7f5e53ac304804f37fee6221536cff17` | Torso and costume: the portrait weighted by one minus the head field | `lemon_spaniel_loop.py` |
| export/static.png | `8aac6c22c42262d358ed915cf6dec054b489f18d18dcf375d1a0827c3e5944b5` | 192 x 192 RGBA static, loop frame 9 | `lemon_spaniel_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `lemon_spaniel_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-16 | concept/20261116-01.png | P-3 with the style key, quoted below. Sent 08:36:57 UTC; `c2pa.created` 2026-11-16T08:37:42Z |
| 2026-11-16 | concept/20261116-02.png | Edit of `concept/20261116-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 08:52:19 UTC; `c2pa.created` 2026-11-16T08:53:19Z |
| 2026-11-16 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/lemon_spaniel_base.png, work/lemon_spaniel_cleanup.py, work/lemon_spaniel_eyes_closed.png, work/lemon_spaniel_eyes_open.png, work/lemon_spaniel_head.png, work/lemon_spaniel_loop.py, work/lemon_spaniel_portrait.png, work/lemon_spaniel_tag.png, work/lemon_spaniel_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.12, crop `(72, 0, 1023, 951)` to `(30, -107, 1064, 927)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261116-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his fifties with light brown skin, wavy dark hair greying at the temples and a short beard, a dove-grey morning coat with a lemon-yellow silk cravat, holding a small cream spaniel puppy with long floppy ears against his chest, the whole puppy fully visible, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261116-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261116-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his fifties with light brown skin, wavy dark hair greying at the temples and a short beard, a dove-grey morning coat with a lemon-yellow silk cravat, holding a small cream spaniel puppy with long floppy ears against his chest, the whole puppy fully visible, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261116-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261116-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.
