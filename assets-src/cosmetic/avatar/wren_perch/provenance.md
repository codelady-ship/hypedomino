# avatar.wren_perch

Animated avatar of set 1 (#171), batch B, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 and 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, final runs on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: `concept/20261115-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261115-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a man in his late sixties with deep brown skin, close-cropped white hair and a short white beard, a cobalt-blue brocade waistcoat with silver buttons over a crisp white shirt and a cobalt silk cravat, and a small brown songbird perched on his shoulder close to his neck, fully visible"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt: the songbird is a small brown bird standing on his left shoulder (the viewer's right), fully visible. Not asked for: silver piping on the waistcoat edges and white shirt sleeves.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.10, 0.11 px, scale 0.9995 and rotation -0.0005 rad (mean squared grey difference outside the eyes 0.00125 before, 0.00122 after), and the eye patch was offset by [-6.0, -0.5, 2.7] in RGB to meet the canonical skin. The edit kept the bird and the cravat; the changes outside the eye mask are small and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly; the bird is a generic small brown songbird
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `legendary`, source `shop:offer.avatar_wren_perch`. The offer id is a placeholder until the shop record exists (V-15). The only item of the batch with a living companion moving on its own: the bird turns its head and flicks its tail on two layers. Animated avatars stay at `epic` and above as in batch A; batch B proposes 5 epic, 4 legendary, 1 mythic, the same split as batch A.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 2 animated details (wren head and tail (a creature counts 2)), so `legendary`, was `mythic`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signatures were not validated with a C2PA tool.

| Field | `concept/20261115-01.png` | `concept/20261115-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T17:22:55Z | `c2pa.created`, 2026-11-15T17:40:56Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 17:22:13 | 2026-11-15 17:39:41 |
| Codex session | `01a1e03b-22c7-7eb0-a195-74b00d32f245` | `01a1e04b-1e9a-7be2-aa26-54a6029bacc7` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/wren_perch/work/wren_perch_cleanup.py wren_perch assets-src/cosmetic/avatar/wren_perch/concept/20261115-01.png assets-src/cosmetic/avatar/wren_perch/work
```

It printed background [184.0, 184.0, 184.0], pockets 7961 px, crop (17, 0, 1183, 1166), portrait 768. The script is `ruby_drop_cleanup.py` of batch A with one change: `CROP_SHARE` is 0.93 instead of 0.86, so the square crop takes more of the concept around the head. With 0.86 the tip of the bird's tail came to 378 px from the centre of the 768 crop, at the circle edge; with 0.93 the whole bird lies inside the circle (tail tip at about 343 px), and the face stays centred. It writes the master and a square 768 x 768 portrait crop (`work/wren_perch_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

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
python -I assets-src/cosmetic/avatar/wren_perch/work/wren_perch_loop.py assets-src/cosmetic/avatar/wren_perch/concept assets-src/cosmetic/avatar/wren_perch/work assets-src/cosmetic/avatar/wren_perch/export
```

It printed `wren_perch_loop: blink edit registered (tx, ty, scale, angle) = (0.1046, 0.1135, 0.9995, -0.0005), residual 0.00125 -> 0.00122, eye colour offset [-6.0, -0.5, 2.7], parts 2, ripples 0, filled 8146 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run into empty folders wrote files byte-identical to each other and to the committed ones.

The script is the batch A loop script (`ruby_drop_loop.py`) with this item's geometry and these additions, the same for every avatar of this batch: the concept file names are constants; a part can open and close (`flap`, a horizontal scale about its pivot); `RIPPLES` add a smooth sideways wave to the warp inside a feathered polygon, so cloth, ribbon and chain move without a cut-out layer and without seams; a hole behind a part is cleared where the fill estimate is less than half covered, which keeps a filled hole from leaving a smear over the background; a glint is clipped to the bright pixels of the portrait and its peak opacity is a constant (`GLINT_PEAK`, 0.55 as in batch A unless the item says otherwise); and the registration region is clipped to the image when the crop reaches past its edge.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `bird_head` (layer `work/wren_perch_bird_head.png`, pivot (596, 374)) turns about its pivot by 12.0 degrees x (sin(lag) - sin(t + lag) + 0.7 sin(2t)), lag 0.0, the turn growing from 0 at the pivot to full at 45 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is cleared, because the part sits over the background | The bird stands well inside the circle after the wider crop; head and tail hang over the background, so their holes are cleared and the turn grows from the neck and the tail root |
| Ambient part | `bird_tail` (layer `work/wren_perch_bird_tail.png`, pivot (642, 410)) turns about its pivot by 6.0 degrees x (sin(lag) - sin(t + lag) + 0.5 sin(2t)), lag 1.2, the turn growing from 0 at the pivot to full at 40 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is cleared, because the part sits over the background | The bird stands well inside the circle after the wider crop; head and tail hang over the background, so their holes are cleared and the turn grows from the neck and the tail root |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 2.13 of 255, against 2.20 on average and 2.66 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.93 to 1.0 (0.93 / 0.92 = 1.011 is wider than the concept, so it is clamped to 1.0, the shorter side of the concept; the subject is 7.0% smaller in the circle instead of 8%)
- `HEADROOM` 0.07 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `wren_perch: background [184.0, 184.0, 184.0], pockets 7961 px, crop (0, -193, 1254, 1061), portrait 768`
- Crop `(17, 0, 1183, 1166)` to `(0, -193, 1254, 1061)` on the concept
- Scale k = 0.9298. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- Printed by the loop: `wren_perch_loop: blink edit registered (tx, ty, scale, angle) = (0.1312, 0.1285, 0.9995, -0.0004), residual 0.00115 -> 0.00111, eye colour offset [-5.7, -0.2, 2.9], parts 2, ripples 0, filled 7149 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.021 before, 0.172 after; highest over the 16 loop frames 0.010 before, 0.167 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/wren_perch/work/wren_perch_cleanup.py wren_perch assets-src/cosmetic/avatar/wren_perch/concept/20261115-01.png assets-src/cosmetic/avatar/wren_perch/work
python -I assets-src/cosmetic/avatar/wren_perch/work/wren_perch_loop.py assets-src/cosmetic/avatar/wren_perch/concept assets-src/cosmetic/avatar/wren_perch/work assets-src/cosmetic/avatar/wren_perch/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 26 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/wren_perch` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `8626cb4cd6567387c38ca6fecd9b0dbd70e058ec993b1b3ec363a1a42d3f3f29` |
| export/anim/0002.png | `de2718c6e4624181eef27d0c4542600f512ff4f84614e501c0f796858b5fa820` |
| export/anim/0003.png | `0f4259248b17acf0d482be3a3f22188420381f51c1ab9d65acc761b478125e68` |
| export/anim/0004.png | `cc2db8152f5fb6afdb8007d584c3433b6f34bf66d9fc4e626f1d808aeb82fabb` |
| export/anim/0005.png | `5820ae6c68af1d255650f8b9c325af252d2cddeaadbd8281c79d375dcd9081f5` |
| export/anim/0006.png | `523cacc995545846345e3541790cd9f2f7892e560ff3bc90fee6efbc4ab6a023` |
| export/anim/0007.png | `8a742d135c57bddc9165c6ae343a491ca09554a809b058f3a9680f88c42c3e6d` |
| export/anim/0008.png | `2e3325cdc713dcdbd85198e566908e63568b0555f5c3859a35a07bb674c0fded` |
| export/anim/0009.png | `6e91518de4ff200d84a7eb2cc3136a8a0ee2e1fb2fca6dac4cb28c7ebed7ed5f` |
| export/anim/0010.png | `d023b6b4abe00f4f756b902b9940027a81b2d0853c92fd60a3a8d7498476beb7` |
| export/anim/0011.png | `753382bd275c24b6dae636a5b50ec7e75ac55b603f2949c193219885b88ee318` |
| export/anim/0012.png | `99a48901cccb3904d237c4cb5b68df7d18e30427202cb553d9d144ce1897f9c9` |
| export/anim/0013.png | `486dc69e8f5cfc53e82990f371f1404b71933b93104687dea0875199a796ca93` |
| export/anim/0014.png | `2909c7699e11fefd1ed56e6d4ef06e0f2f4d5a440cae3e56929d689eca1def39` |
| export/anim/0015.png | `c6427fc5fcbbf7661794def81bbf10be1669a275b4cb3ba3b14a12d1ffd98b33` |
| export/anim/0016.png | `2877b486690564a0b50fbab7e808213602e979b9cbd3b04659907af886d75185` |
| export/static.png | `6e91518de4ff200d84a7eb2cc3136a8a0ee2e1fb2fca6dac4cb28c7ebed7ed5f` |
| work/wren_perch_base.png | `512f67a63c70b5653e92444a66f128ee9560e1786fe28949c18f23a0a541b501` |
| work/wren_perch_bird_head.png | `655f5725610e0d28e15409094b03e34319ca1fdf8696f5a65e498743c2d6dde2` |
| work/wren_perch_bird_tail.png | `5a1a6b1e5b6d1b237c4fd38a5a894fec4c038c40ab54fdd2bfda319b3ba43752` |
| work/wren_perch_cleanup.py | `1da8c45fbe97b8b5a0001a0c1e81daa1619d2b115c4966c30e71d5d8112591df` |
| work/wren_perch_eyes_closed.png | `be0dedce9cacac6df2df1f79c6d468425b846fe3610eb2b85bfc54a7c07dc4b7` |
| work/wren_perch_eyes_open.png | `ee101239a78652c36e90bb96dfb67f5209ecd530bc8bb136ff54b1f64e43a356` |
| work/wren_perch_head.png | `99575efb54ba1a35f613812d6b0f2ab8a41fe7131e46f076b77a8d1a0a1e5b5b` |
| work/wren_perch_loop.py | `d21058723cd9cce5aeac8c0f8714a18b8fbdd12ec0060e3dbb36ec6a46891108` |
| work/wren_perch_portrait.png | `490ed580ec9be13cb0209362063c1fcc6485813c11fde89b2fe2a1da6bb1c237` |
| work/wren_perch_torso.png | `32e8493534640de58d604931b18d0b3165bb04f151d932429d6a1b2ee6c39b9e` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `802017d162e343bb3bc68ff10718adca31eb66a9f3d5fcd9f1539f2bcb70e181` | 1254 x 1254 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e03b-22c7-7eb0-a195-74b00d32f245` |
| concept/20261115-02.png | `44602c8681b2ab5971d48a6bbbb14d1a696faaf1fa0af4f074d47bf5574149a5` | 1254 x 1254 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e04b-1e9a-7be2-aa26-54a6029bacc7` |
| work/wren_perch_cleanup.py | `ffbfdb03279134f873d3d5dbc90bd956e313a94ca71a5dc68d42103de56e5a8a` | W-3 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_cleanup.py` |
| work/wren_perch_loop.py | `7f602438be6ebcf3169920320ec8ef3f772f1a0c72000d6a9673b1adf2ce42a9` | W-4 and W-5 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_loop.py`; the same script for every avatar of this batch with the item's geometry |
| work/wren_perch_master.png | `616fe81f81c56d9c397f6ee84438b515ddbfa9804d2bdae19a0d5f84094ae841` | 1254 x 1254 RGBA, the cut-out master | `wren_perch_cleanup.py` |
| work/wren_perch_base.png | `600ea738a26dd4bb73dc0e99845b47235ff6cbd7f588a6425f4a2b224c14b826` | The portrait with the moving parts taken out (filled or cleared) | `wren_perch_loop.py` |
| work/wren_perch_bird_head.png | `17f83583908bf8ff7103428df9cc0546f6aaacaf7c2fcc09c520bda0fead1c71` | Moving part `bird_head`: the portrait through its feathered polygon | `wren_perch_loop.py` |
| work/wren_perch_bird_tail.png | `7060b26b6ed56fb198e553bfe618c06bcf2f19fe758151d56b9f6dc7d2ca20c1` | Moving part `bird_tail`: the portrait through its feathered polygon | `wren_perch_loop.py` |
| work/wren_perch_eyes_closed.png | `24832b0943d2dfba6621d759392fd818968b1cbc40318702dac914c002948f34` | Eye region, closed (aligned edit) through the same mask | `wren_perch_loop.py` |
| work/wren_perch_eyes_open.png | `885de9b214002e403f32a645aa92317a6d1ed19c2a425fd679a5971874688761` | Eye region, open (canonical concept) through the feathered eye mask | `wren_perch_loop.py` |
| work/wren_perch_head.png | `643b7323134c6458d6dd1435e6bcb5dfb9a04c40d8917919432a5db8ba96942e` | Head and hair: the portrait weighted by the head field of the warp | `wren_perch_loop.py` |
| work/wren_perch_portrait.png | `f36fab1ae98ac6a5c868c0692f9a55a8103eb777b6252d23d42d3fec534e18a8` | 768 x 768 RGBA portrait crop, no circle | `wren_perch_cleanup.py` |
| work/wren_perch_torso.png | `3f7e8c2c8ab6510868a018aa7da212c275dd387074c9c5f78df42e665a103852` | Torso and costume: the portrait weighted by one minus the head field | `wren_perch_loop.py` |
| export/static.png | `5062dbf6042ff5c822ea8dedbb9896c9e558ba5717f57dcd8591b6fae69a8afd` | 192 x 192 RGBA static, loop frame 9 | `wren_perch_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `wren_perch_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted below. Sent 17:22:13 UTC; `c2pa.created` 2026-11-15T17:22:55Z |
| 2026-11-15 | concept/20261115-02.png | Edit of `concept/20261115-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 17:39:41 UTC; `c2pa.created` 2026-11-15T17:40:56Z |
| 2026-11-16 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/wren_perch_base.png, work/wren_perch_bird_head.png, work/wren_perch_bird_tail.png, work/wren_perch_cleanup.py, work/wren_perch_eyes_closed.png, work/wren_perch_eyes_open.png, work/wren_perch_head.png, work/wren_perch_loop.py, work/wren_perch_portrait.png, work/wren_perch_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.93 to 1.0, `HEADROOM` 0.07 to 0.17, crop `(17, 0, 1183, 1166)` to `(0, -193, 1254, 1061)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his late sixties with deep brown skin, close-cropped white hair and a short white beard, a cobalt-blue brocade waistcoat with silver buttons over a crisp white shirt and a cobalt silk cravat, and a small brown songbird perched on his shoulder close to his neck, fully visible, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261115-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his late sixties with deep brown skin, close-cropped white hair and a short white beard, a cobalt-blue brocade waistcoat with silver buttons over a crisp white shirt and a cobalt silk cravat, and a small brown songbird perched on his shoulder close to his neck, fully visible, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261115-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261115-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.
