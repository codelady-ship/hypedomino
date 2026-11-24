# avatar.turquoise_pendant

Animated avatar of set 1 (#171), batch C, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-16 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, final runs on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-16 for the Plus plan and records the date here
- Canonical concept: concept/20261116-01.png
- Inputs uploaded to the generator: `concept/20261116-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261116-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a woman in her forties with light tan skin, a sleek chin-length auburn bob, a chocolate-brown satin gown with a square neckline, and one long turquoise drop pendant hanging on a fine gold chain to the middle of her chest, fully visible"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt: the long turquoise drop hangs on a fine gold chain to the middle of her chest. Not asked for: matching turquoise drop earrings, pearl and gold trim on the bodice and a pearl ornament at the neckline.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.08, -0.08 px, scale 0.9995 and rotation 0.0000 rad (mean squared grey difference outside the eyes 0.00090 before, 0.00087 after), and the eye patch was offset by [-5.4, -2.2, -0.5] in RGB to meet the canonical skin. The changes outside the eye mask are small and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly; the square neckline shows the upper chest, which the reviewer judges for the all-ages audience
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `legendary`, source `shop:offer.avatar_turquoise_pendant`. The offer id is a placeholder until the shop record exists (V-15). One swinging pendant on a long chain, with the earrings swinging in step. Animated avatars stay at `epic` and above as in batches A and B; batch C proposes 5 epic, 3 legendary, 2 mythic, which brings the 30 animated avatars of set 1 to 15 epic, 11 legendary and 4 mythic.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 2 animated details (pendant, earrings (pair)), so `legendary`, was `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-16; the signatures were not validated with a C2PA tool.

| Field | `concept/20261116-01.png` | `concept/20261116-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-16T08:27:47Z | `c2pa.created`, 2026-11-16T08:44:28Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-16 08:27:01 | 2026-11-16 08:43:25 |
| Codex session | `01a1e377-7ee5-7311-96a4-fa02e7512c56` | `01a1e386-8176-7521-a820-ed81567daa67` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked by a byte comparison). No generation for this item was rejected: these are the only two images generated for it.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/turquoise_pendant/work/turquoise_pendant_cleanup.py turquoise_pendant assets-src/cosmetic/avatar/turquoise_pendant/concept/20261116-01.png assets-src/cosmetic/avatar/turquoise_pendant/work
```

It printed background [191.0, 191.0, 191.0], pockets 693 px, crop (76, 0, 1122, 1046), portrait 768. The script is `cornflower_ribbon_cleanup.py` of batch B unchanged (the batch A clean-up of `avatar.ruby_drop` with its 768 crop, grey-pocket rule and edge re-matte). It writes the master and a square 768 x 768 portrait crop (`work/turquoise_pendant_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

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
python -I assets-src/cosmetic/avatar/turquoise_pendant/work/turquoise_pendant_loop.py assets-src/cosmetic/avatar/turquoise_pendant/concept assets-src/cosmetic/avatar/turquoise_pendant/work assets-src/cosmetic/avatar/turquoise_pendant/export
```

It printed `turquoise_pendant_loop: blink edit registered (tx, ty, scale, angle) = (0.0810, -0.0831, 0.9995, 0.0000), residual 0.00090 -> 0.00087, eye colour offset [-5.4, -2.2, -0.5], parts 3, ripples 0, filled 24506 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run of the clean-up and the loop into empty folders wrote files byte-identical to each other and to the committed ones.

The script is the batch B loop script (`cornflower_ribbon_loop.py`, which carries the batch A rig of `ruby_drop_loop.py` with `flap`, `RIPPLES`, the clearing hole fill, clipped glints and the clipped registration region) with this item's geometry and one addition, the same for every avatar of batch C: a part can spin (`spin`, whole turns per loop added to its turn angle, so a full turn ends where it started). Parts without `spin` move exactly as in batch B.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `pendant` (layer `work/turquoise_pendant_pendant.png`, pivot (374, 440)) turns about its pivot by 6.0 degrees x (sin(lag) - sin(t + lag)), lag 0.0, the turn growing from 0 at the pivot to full at 300 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | The chain and the drop are one thin layer; the pivot sits above the neck between the two chain ends, so the chain ends barely move and the drop swings like a pendulum over the skin, inside the circle |
| Ambient part | `earring_left` (layer `work/turquoise_pendant_earring_left.png`, pivot (275, 284)) turns about its pivot by 6.0 degrees x (sin(lag) - sin(t + lag)), lag 0.5, the turn growing from 0 at the pivot to full at 50 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | A small part well inside the circle, so its motion never meets the circle edge |
| Ambient part | `earring_right` (layer `work/turquoise_pendant_earring_right.png`, pivot (516, 308)) turns about its pivot by 6.0 degrees x (sin(lag) - sin(t + lag)), lag 0.9, the turn growing from 0 at the pivot to full at 45 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | A small part well inside the circle, so its motion never meets the circle edge |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 1.91 of 255, against 1.80 on average and 2.10 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.07 to 0.12, not the 0.17 of the other avatars: at 0.17 the pendant layer was 1.2% outside the circle; at 0.12 it is 0.7%, the tip touches the circle at the bottom
- The crop may now reach above the top of the concept: `upper = min(top - HEADROOM * side, height - side)` instead of clipping it at 0, the rule `avatar.tawny_quill` already used. The subject starts within a few pixels of the concept's top edge, so a lower head needs transparent canvas above it; the bottom and side clamps are as before
- Printed by the clean-up: `turquoise_pendant: background [191.0, 191.0, 191.0], pockets 693 px, crop (31, -130, 1168, 1007), portrait 768`
- Crop `(76, 0, 1122, 1046)` to `(31, -130, 1168, 1007)` on the concept
- Scale k = 0.9200. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- Printed by the loop: `turquoise_pendant_loop: blink edit registered (tx, ty, scale, angle) = (0.0980, -0.1397, 0.9996, -0.0001), residual 0.00073 -> 0.00071, eye colour offset [-5.1, -2.0, -0.3], parts 3, ripples 0, filled 21148 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.005 before, 0.120 after; highest over the 16 loop frames 0.000 before, 0.115 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/turquoise_pendant/work/turquoise_pendant_cleanup.py turquoise_pendant assets-src/cosmetic/avatar/turquoise_pendant/concept/20261116-01.png assets-src/cosmetic/avatar/turquoise_pendant/work
python -I assets-src/cosmetic/avatar/turquoise_pendant/work/turquoise_pendant_loop.py assets-src/cosmetic/avatar/turquoise_pendant/concept assets-src/cosmetic/avatar/turquoise_pendant/work assets-src/cosmetic/avatar/turquoise_pendant/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 27 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/turquoise_pendant` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `fa8d1ea9e980dd19d3a948587cfd638785250a22f3109c88e33d395c5a584b47` |
| export/anim/0002.png | `944ebfebc4e7c4f5067371c25d70d9d18fa848a65618f4db0bd271159cf5eeb6` |
| export/anim/0003.png | `7c980e34b6c92460360905607987d67159d906aec60cebca610a33f09594e1ea` |
| export/anim/0004.png | `f10496a9496b4c77b0d8a77d6fec131f7a028fc75e29b4ab2f0cc64ab84f1967` |
| export/anim/0005.png | `78ec627c803e6bf731ee7b8044231615e89b6727145b81d2584606a6a21afde1` |
| export/anim/0006.png | `08805295a66973e24815e1e56fbd57dcb892027f645b0f12fc2cb7063581869d` |
| export/anim/0007.png | `cbc7466ee9ea98e2a9fddc842f63202ec9dee29b3430a9ec60b663956ba97db9` |
| export/anim/0008.png | `343c88a518d8d5d02db49ea91c894542fd197fce011868d664868f8cae84a3fb` |
| export/anim/0009.png | `c0d83425c3adf9ffd353b044a5f6c676765bc17e022424e4f977c73fe5f3de4c` |
| export/anim/0010.png | `f7124407d2748e8a951790c85b6d8a8302aee97c55d816efee3e668ea8985efc` |
| export/anim/0011.png | `95c2f873ec981393a889221887f8655e1505a2a2118e76fd8e2a7e98352ec21f` |
| export/anim/0012.png | `4303417f8ad596adb33278b433e9961690813d823b5d860341f96d9bcb2cb022` |
| export/anim/0013.png | `bd36c920fa315bb1054ff1843c56383c9cbd639e1589510cee33bbe74dd83f1e` |
| export/anim/0014.png | `62c263c4f996620973a357f2c3f0b57a57897502cbd34fbd03267429619bbd89` |
| export/anim/0015.png | `57402b1ab2d845b356762bd50a32c6ab646a68595659e6f682e5741e93ce47c8` |
| export/anim/0016.png | `d00efc22db76fc51e62b9c0adb6d970f8c2debb72fac5b02bc340871bef01ed8` |
| export/static.png | `c0d83425c3adf9ffd353b044a5f6c676765bc17e022424e4f977c73fe5f3de4c` |
| work/turquoise_pendant_base.png | `4fb3e0633eb7a888c10c76cdd8b9794f54269fcb0dd8607cb7e168d25bc51efc` |
| work/turquoise_pendant_cleanup.py | `18a09c5cb88e9deea6bacde53bf5efc038ab7d778bc75123651cf1298b91226b` |
| work/turquoise_pendant_earring_left.png | `b2df372d7ac43b85411925ecd0867a20751c7a0b913c32dac7a30d68b63b05e8` |
| work/turquoise_pendant_earring_right.png | `aa10ba8fa4b6d9d880a176cdfa4647216e87b4d9300322129c6e917e30d29a1a` |
| work/turquoise_pendant_eyes_closed.png | `3e7be592108f7655db946b777a7a7ecee750f7b40de5ac2445b2fab5fe6e774d` |
| work/turquoise_pendant_eyes_open.png | `c42b01b806db8cae83e96b3fdbf097673a9bd0598c6249773df73947005ee131` |
| work/turquoise_pendant_head.png | `12319638f08dc6534a9d1fd091c8cd462db7ea353db2a7a1a2b2e378c910eddf` |
| work/turquoise_pendant_loop.py | `73f01d300f03c8a23b5dc76b9b3b27dc0785c81824d47951003b2b66635cd70e` |
| work/turquoise_pendant_pendant.png | `62812a5cbf3162fd7f8c7dbad4023004ff591af07c4c38ffc85e3954384a8a29` |
| work/turquoise_pendant_portrait.png | `3f0f971111db7c3dc6ab4093f51ad1cb830c6f77b196112b62a78445b2a6b3a9` |
| work/turquoise_pendant_torso.png | `46e0286c386f0440321fee9665a71b81d86417db887102e593dcfa7fa0918d32` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261116-01.png | `85d22a7c2299fd837ee1ee02947223426e2c4a3e703a5cc41091cf19f7f5d148` | 1217 x 1293 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e377-7ee5-7311-96a4-fa02e7512c56` |
| concept/20261116-02.png | `093ebebde96a70f58acbf8bcf38a2156360787de2ed1bcae09c44739b05f36c9` | 1216 x 1293 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e386-8176-7521-a820-ed81567daa67` |
| work/turquoise_pendant_cleanup.py | `8c39fa1eb5065d0824cb0eaf53942d85348631f0b7c1a41fc272137a55333af3` | W-3 script | Written by Claude Code, 2026-11-16, from `cornflower_ribbon_cleanup.py` |
| work/turquoise_pendant_loop.py | `34876115f215f4989c5287cfa8c670302d3e65eaafaa3f2083e632ef329bff76` | W-4 and W-5 script | Written by Claude Code, 2026-11-16, from `cornflower_ribbon_loop.py`; the same script for every avatar of this batch with the item's geometry |
| work/turquoise_pendant_master.png | `d9cef5586df03da23db0a6da6fd7a872bbbd585fc34e969a418f5d443fb33140` | 1217 x 1293 RGBA, the cut-out master | `turquoise_pendant_cleanup.py` |
| work/turquoise_pendant_base.png | `b4cf44719913478998d675bf89752a6716746ee4b73f9edcd7ecbebc6ffc72b0` | The portrait with the moving parts taken out (filled or cleared) | `turquoise_pendant_loop.py` |
| work/turquoise_pendant_earring_left.png | `c5ad0108cdb854ddc75bcaae6d6f735cf170bf30a890a521ea51eae732854ea0` | Moving part `earring_left`: the portrait through its feathered polygon | `turquoise_pendant_loop.py` |
| work/turquoise_pendant_earring_right.png | `b77f801388e45169b35615cee2f508b731c07a9d0bfd04eecce99525502481a0` | Moving part `earring_right`: the portrait through its feathered polygon | `turquoise_pendant_loop.py` |
| work/turquoise_pendant_eyes_closed.png | `2c5d0b9aab83621bbe9eeb2568f6bd4f61e642bb6452ba39c0077568000fcdc5` | Eye region, closed (aligned edit) through the same mask | `turquoise_pendant_loop.py` |
| work/turquoise_pendant_eyes_open.png | `06bea17d9e7da4698be5b7ed1b58b6a6ba58fc0b80505bc2ef90afb1aa55bd7c` | Eye region, open (canonical concept) through the feathered eye mask | `turquoise_pendant_loop.py` |
| work/turquoise_pendant_head.png | `bf2a3d620fc3b53d406ee81d8fbf50ee7a738e28f19a2acc0ea775e51502cc15` | Head and hair: the portrait weighted by the head field of the warp | `turquoise_pendant_loop.py` |
| work/turquoise_pendant_pendant.png | `550efe7ef25e17ef5fd8a95a786906fdd49c24568ba437ecc919d01f930d146b` | Moving part `pendant`: the portrait through its feathered polygon | `turquoise_pendant_loop.py` |
| work/turquoise_pendant_portrait.png | `899b59e5f6c6cab82f91742803a87d54f1d80f4fe1c800d38c2eb342a6379dda` | 768 x 768 RGBA portrait crop, no circle | `turquoise_pendant_cleanup.py` |
| work/turquoise_pendant_torso.png | `3db93b6420309e507f9375981d16bc84572150553f5f755aa69ff2f538342a82` | Torso and costume: the portrait weighted by one minus the head field | `turquoise_pendant_loop.py` |
| export/static.png | `0450828b2c3bb3578e8be32db79dabb37a445f3aad1f83fde1d12d8c1b09fa21` | 192 x 192 RGBA static, loop frame 9 | `turquoise_pendant_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `turquoise_pendant_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-16 | concept/20261116-01.png | P-3 with the style key, quoted below. Sent 08:27:01 UTC; `c2pa.created` 2026-11-16T08:27:47Z |
| 2026-11-16 | concept/20261116-02.png | Edit of `concept/20261116-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 08:43:25 UTC; `c2pa.created` 2026-11-16T08:44:28Z |
| 2026-11-16 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/turquoise_pendant_base.png, work/turquoise_pendant_cleanup.py, work/turquoise_pendant_earring_left.png, work/turquoise_pendant_earring_right.png, work/turquoise_pendant_eyes_closed.png, work/turquoise_pendant_eyes_open.png, work/turquoise_pendant_head.png, work/turquoise_pendant_loop.py, work/turquoise_pendant_pendant.png, work/turquoise_pendant_portrait.png, work/turquoise_pendant_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.07 to 0.12, crop `(76, 0, 1122, 1046)` to `(31, -130, 1168, 1007)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261116-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her forties with light tan skin, a sleek chin-length auburn bob, a chocolate-brown satin gown with a square neckline, and one long turquoise drop pendant hanging on a fine gold chain to the middle of her chest, fully visible, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261116-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261116-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a woman in her forties with light tan skin, a sleek chin-length auburn bob, a chocolate-brown satin gown with a square neckline, and one long turquoise drop pendant hanging on a fine gold chain to the middle of her chest, fully visible, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261116-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261116-02.png. Do not use any API key.`

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.
