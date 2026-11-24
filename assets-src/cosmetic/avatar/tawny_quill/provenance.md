# avatar.tawny_quill

Animated avatar of set 1 (#171), batch B, a painted portrait made with template P-3 of `docs/asset-workflow.md` section 2 and animated offline (decision D-3 of the Epic 15 plan): the cleaned master split into parts on layers in `work/`, moved by a deterministic script into the sprite-sheet loop of ADR 0012. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 and 2026-11-16 (UTC) while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept and the closed-eyes edit generated in the project owner's ChatGPT account on 2026-11-15 by an AI agent (Claude Code) through Codex CLI. W-3 clean-up, W-4 animation and W-5 export by the same agent with the scripts in `work/`, final runs on 2026-11-16. No person has done W-3 or W-4
- Generator: Codex CLI 0.169.0 built-in image generation (`$imagegen`), model gpt-image-2 per OpenAI docs (learn.chatgpt.com/docs/image-generation); the C2PA manifests in the PNGs name ChatGPT / gpt-image (below). Account holder: the project owner's ChatGPT account, plan Plus (read from the Codex login on 2026-11-15)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-15 for the Plus plan and records the date here
- Canonical concept: concept/20261115-01.png
- Inputs uploaded to the generator: `concept/20261115-01.png`, our own canonical concept of this item, attached with `codex exec -i` to the one edit that made `concept/20261115-02.png` (closed eyes for the blink). Nothing else was uploaded
- Prompt: P-3 with the style key of section 2.2 pasted verbatim; `{identity}` = "a man in his mid-twenties with warm golden-brown skin and short side-swept black hair, clean-shaven, a long tawny-and-cream striped feather quill tucked behind one ear with its feathered end rising above his hair, and a pale ice-blue silk shirt with a high collar under a deep indigo tailcoat with silver piping"; quoted below with the edit prompt
- Concept against the prompt: Matches the prompt: the quill is tucked behind his right ear (the viewer's left) and rises above the hair. Not asked for: a sapphire and crystal pin on the cravat and gold buttons.
- Blink edit against the canonical: Pose, framing, costume and colours kept; the alignment fitted a shift of 0.01, 0.01 px, scale 1.0000 and rotation -0.0001 rad (mean squared grey difference outside the eyes 0.00054 before, 0.00054 after), and the eye patch was offset by [-2.6, 0.0, 1.2] in RGB to meet the canonical skin. The changes outside the eye mask are small and not used. Used: only the two eye ellipses.
- Similarity check: not done (RC-3 open, O-7)
- Real-person check (RC-4): not done by a person. The face is semi-realistic, so `docs/art-direction.md` 5.1 makes this check matter; the reviewer confirms it resembles no real person
- Human work: none. Clean-up, animation and export are scripted image operations run by an AI agent, so RC-5 stays open; the master, the layers and the scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in; the expression is friendly; the quill is a writing quill, not a headdress feather
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `epic`, source `shop:offer.avatar_tawny_quill`. The offer id is a placeholder until the shop record exists (V-15). One ambient motion, the quill sway. Animated avatars stay at `epic` and above as in batch A; batch B proposes 5 epic, 4 legendary, 1 mythic, the same split as batch A.

2026-11-16: owner decision — an animated item's rarity follows its animated detail (`docs/rarity.md` 1.1): 1 animated detail (quill), so `epic`. Any tier split stated above is superseded.

## Embedded C2PA manifests

Both concept files carry a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-15; the signatures were not validated with a C2PA tool.

| Field | `concept/20261115-01.png` | `concept/20261115-02.png` |
|---|---|---|
| Action | `c2pa.created`, 2026-11-15T17:37:43Z | `c2pa.created`, 2026-11-15T17:54:17Z |
| Software agent | `ChatGPT`, version `gpt-image` | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC | OpenAI OpCo, LLC |
| Prompt sent (UTC, agent clock) | 2026-11-15 17:36:55 | 2026-11-15 17:53:15 |
| Codex session | `01a1e048-945a-77a0-bee0-2464072aeeee` | `01a1e057-8905-7181-b340-50cf52e4bf15` |

Codex saved each image under `~/.codex/generated_images/<session>/` and copied it into `concept/`; each pair of files is byte-identical (checked with `cmp`). One earlier generation for this look was rejected and is not committed: it was made under the working slug `ivory_quill` (Codex session `01a1e03e-76a2-7c71-be01-f8ab1d0cf7c7`, prompt sent 2026-11-15 17:25:52 UTC) with the first prompt quoted below. The white quill and the pearl-white coat sat so close to the light grey background that the clean-up keyed out most of the quill and cut holes into the coat. The identity was changed to a tawny striped quill and an indigo coat, the slug to `tawny_quill`, and the next generation became the canonical.

## Clean-up (W-3)

Run on 2026-11-16 (UTC) by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/tawny_quill/work/tawny_quill_cleanup.py tawny_quill assets-src/cosmetic/avatar/tawny_quill/concept/20261115-01.png assets-src/cosmetic/avatar/tawny_quill/work
```

It printed background [204.0, 203.0, 203.0], pockets 1564 px, crop (94, -91, 1172, 987), portrait 768. The script is `ruby_drop_cleanup.py` of batch A with three changes. The first two keep the quill tip inside the circle: `HEADROOM` is 0.1 instead of 0.07, and the top of the crop may lie above the concept (printed as a negative top, filled with transparent pixels) instead of being clipped to row 0. Without them the quill tip touched the top of the circle; with them it is about 330 px from the centre of the 768 crop. A third change keeps two enclosed pockets of the light blue cravat that the pocket rule took for background (they showed as holes at the collar and beside the cravat): `POCKET_KEEP` lists one concept pixel of each, (710, 793) and (761, 699), found by looking at the cut-out on a magenta backdrop, and the pocket that contains such a pixel stays opaque. It writes the master and a square 768 x 768 portrait crop (`work/tawny_quill_portrait.png`, no circle); the enclosed grey pockets, the edge re-matte and the fringe removal work as described in the batch A provenance of `avatar.ruby_drop`.

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
python -I assets-src/cosmetic/avatar/tawny_quill/work/tawny_quill_loop.py assets-src/cosmetic/avatar/tawny_quill/concept assets-src/cosmetic/avatar/tawny_quill/work assets-src/cosmetic/avatar/tawny_quill/export
```

It printed `tawny_quill_loop: blink edit registered (tx, ty, scale, angle) = (0.0103, 0.0090, 1.0000, -0.0001), residual 0.00054 -> 0.00054, eye colour offset [-2.6, 0.0, 1.2], parts 1, ripples 0, filled 14850 px`. It writes the layers below into `work/`, `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, straight alpha. A second and a third run into empty folders wrote files byte-identical to each other and to the committed ones.

The script is the batch A loop script (`ruby_drop_loop.py`) with this item's geometry and these additions, the same for every avatar of this batch: the concept file names are constants; a part can open and close (`flap`, a horizontal scale about its pivot); `RIPPLES` add a smooth sideways wave to the warp inside a feathered polygon, so cloth, ribbon and chain move without a cut-out layer and without seams; a hole behind a part is cleared where the fill estimate is less than half covered, which keeps a filled hole from leaving a smear over the background; a glint is clipped to the bright pixels of the portrait and its peak opacity is a constant (`GLINT_PEAK`, 0.55 as in batch A unless the item says otherwise); and the registration region is clipped to the image when the crop reaches past its edge.

| Choice | Value | Source |
|---|---|---|
| Canvas | 192 x 192 | D-1 of the Epic 15 plan and the ADR 0012 illustration (16 frames of 192 x 192 in a 768 atlas). Every frame is rendered at 768 x 768 in premultiplied alpha from the portrait crop and box-downsampled by 4 |
| Frames | 16 | D-1, ADR 0012 first measurement |
| Frame rate | 12 fps, a loop of 1.33 s | D-1; within the 1 to 2 s of #171. The rate is not stored with the loop (O-5, O-6) |
| Blink | Frames 5 and 6 (two frames, 167 ms) show the closed eyes, every other frame the open eyes | The closed eyes come from the Codex edit, aligned to the canonical by a similarity transform fitted by least squares outside the eyes and blended in through two feathered ellipses (Gaussian sigma 5 at 768) with a colour offset measured in a ring around them |
| Breathing | The shoulders and chest rise by up to 8 px at 768 (2 px at 192) with (1 - cos)/2 over the loop, through a smooth vertical mesh warp that is 1 at the shoulder line, falls to 0 at the bottom edge and carries the head at 0.4; the chest widens by 1.2% at the same time | Warp only, so no seams |
| Head | A tilt of up to 1.6 degrees about the neck with sin over the loop, and a nod of 2 px at 768 twice per loop, weighted by a soft ellipse around head and hair that fades out at the neck | Warp only |
| Ambient part | `quill` (layer `work/tawny_quill_quill.png`, pivot (238, 322)) turns about its pivot by 3.0 degrees x (sin(lag) - sin(t + lag)), lag 0.0, the turn growing from 0 at the pivot to full at 200 px (768 scale), so the end near the pivot stays attached and the far end swings; the part also follows the head or torso warp at its pivot; the hole it leaves in the base layer is filled from the surrounding pixels, and where that estimate is less than half covered it is cleared instead, so no smear shows over the background | A small part well inside the circle, so its motion never meets the circle edge |
| Seamless loop | Mean channel change from frame 16 to frame 1 is 2.24 of 255, against 1.88 on average and 2.30 at most between the other neighbouring frames | Every motion is periodic in the 16 frames |
| Static | Frame 9 | The rest frame: eyes open, no breath, no tilt or nod, every moving part at its painted position. `export/static.png` is byte-identical to `export/anim/0009.png` |
| Size | 17 sprites, 278,528 bytes | `npm run asset:check`, ASTC 6x6 estimate (17 x 16,384), under the 300,000 bytes of #171 |

Not done: the reduce-motion and low-level check of 5.2 (Unity), the board panels B-8 and B-9, the bundle measurement of ADR 0012, and W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Re-crop, 2026-11-16

Owner feedback from the phone on 2026-11-16: inside the frames, the frame's top ornament band covered the top of the head and hair. The owner asked for the portrait "very slightly smaller and a bit lower"; `docs/art-direction.md` 5.1 keeps anything essential out of the outer band the frame covers. Only the crop framing changed. The cut-out master was re-made by the same run and is byte-identical; masks, pockets and the edge re-matte are unchanged.

- `CROP_SHARE` 0.86 to 0.935 (0.86 / 0.92: the subject is 8% smaller in the circle)
- `HEADROOM` 0.1 to 0.17, the same for every avatar except four animated ones whose moving part would leave the circle
- The crop may reach above the top of the concept (transparent canvas), as this item's script already allowed. The bottom and side clamps are as before
- Printed by the clean-up: `tawny_quill: background [204.0, 203.0, 203.0], pockets 1564 px, crop (47, -182, 1219, 990), portrait 768`
- Crop `(94, -91, 1172, 987)` to `(47, -182, 1219, 990)` on the concept
- Scale k = 0.9198. The rig geometry in portrait pixels (`HEAD`, `NECK_Y`, `NECK_BAND`, `SHOULDER_Y`, `BREATH`, `NOD`, `EYES`, `EYE_FEATHER`, the part polygons, pivots, `length` and `grow`, the glints and the ripples) is mapped through the crop change (x' = k x + offset, sizes and pixel amplitudes times k), so every mask stays on the same anatomy and the motion is the same relative to the subject. Angles, timing, blink frames and `STATIC_FRAME` are unchanged
- Printed by the loop: `tawny_quill_loop: blink edit registered (tx, ty, scale, angle) = (0.0085, 0.0039, 1.0000, -0.0001), residual 0.00051 -> 0.00051, eye colour offset [-2.6, 0.1, 1.3], parts 1, ripples 0, filled 12767 px`
- Head top (first row of `export/static.png` with alpha above 0.5, as a share of the canvas height): 0.109 before, 0.177 after; highest over the 16 loop frames 0.094 before, 0.167 after

Run on 2026-11-16 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/avatar/tawny_quill/work/tawny_quill_cleanup.py tawny_quill assets-src/cosmetic/avatar/tawny_quill/concept/20261115-01.png assets-src/cosmetic/avatar/tawny_quill/work
python -I assets-src/cosmetic/avatar/tawny_quill/work/tawny_quill_loop.py assets-src/cosmetic/avatar/tawny_quill/concept assets-src/cosmetic/avatar/tawny_quill/work assets-src/cosmetic/avatar/tawny_quill/export
```

Determinism: two more runs of the same commands into two empty folders wrote the same 25 files byte for byte, and both match the files committed here.

`npm run asset:check -- assets-src/cosmetic/avatar/tawny_quill` after the re-crop: 17 sprites, ASTC 6x6 estimate 278,528 bytes, limit under 300,000 (#171, animated avatars under 300 KB each): pass.

The Files table below keeps the hashes of the first export. The files the re-crop changed now hash as follows:

| File | sha256 after the re-crop |
|---|---|
| export/anim/0001.png | `4482482a199db2a263f4c8d42ed24898cc3536fb8b4331b5b4939c84721b4fb4` |
| export/anim/0002.png | `6a05d716ca03ee5bfde7d5839618ce4f7f62d358a95890d62569950afced9503` |
| export/anim/0003.png | `35d6180ec2683353722b3077b7f83c811a2a97fa90c1cfb9846d59a951188ee1` |
| export/anim/0004.png | `7fc85440952fd97b1e49950c3b6a959feadbef3d37f7f404c9fb7a18a3c9dce5` |
| export/anim/0005.png | `768dfe0d61c850f8f5e8fe3873f0b4095920504e82d165e6c78727a2bd5f0707` |
| export/anim/0006.png | `c01e6df1e2eed14ea43700299fe32d28411686743291226a3713ea8b55323eb0` |
| export/anim/0007.png | `42afb6c15c560249e322b444208282da1287ce001e340ccc86d2fb4c67baaf2f` |
| export/anim/0008.png | `00a081e4317ce49c08ae6e6e5121369ebe329cf54773366b36d9b3cd32714854` |
| export/anim/0009.png | `87b3e3c9cf5c559d926039c620ea8a1318a680a4bf1ac23cc54519389beca2a6` |
| export/anim/0010.png | `f74542bb316127ff528d0d938e78dc1f7d29970a87d94dc4e8229384af37eaca` |
| export/anim/0011.png | `b0cf2e3b21851764c10aaa0c3577a7e286d43ffc1914a9aef30763fe2d245b49` |
| export/anim/0012.png | `a86d8c6253b466a87daa962713357852dae062ca3a7c525eb4b0ed69a21f4d81` |
| export/anim/0013.png | `633ed6cb49924f9440550a37911d88de05989d08396496969d87577092c44981` |
| export/anim/0014.png | `3a6d2868b3e66f6ff94a5af31ec0e0d3e3e5720ee54a34249dc7c25527f1d34d` |
| export/anim/0015.png | `d6f1c1370fd01bd4d7b5a2603f4274058b963139164bdf4238edefb24fa17d8e` |
| export/anim/0016.png | `9e20cb0d1b62bd317ef3adc2c51105748c84a8b2645ff18f35bf922176607829` |
| export/static.png | `87b3e3c9cf5c559d926039c620ea8a1318a680a4bf1ac23cc54519389beca2a6` |
| work/tawny_quill_base.png | `6ae2794963e206831f6ca879b1a914bd2978453d3607c195cba80571014601bf` |
| work/tawny_quill_cleanup.py | `517634391f9b41099a593ee43224ef8f432bb0363f070449f124ae199b06f473` |
| work/tawny_quill_eyes_closed.png | `39f8ad6c94c8cef370024aa2f7ad100f5f32d0d70874168da31a5b7e1e0e2c7b` |
| work/tawny_quill_eyes_open.png | `a23e36b01f19825c1aee9f52b9c0d8f79b8e54eca1ec0804abd26e8172ec1d1a` |
| work/tawny_quill_head.png | `412266fcc282eb4d02eb6e10fc3b1e2b4bbbdd8d074aa6ca1b644560de3b620e` |
| work/tawny_quill_loop.py | `47b154eef45cc8b9f017de3ba7a6cea8efe4ea90de9b776f0a1e1bface3f1ea4` |
| work/tawny_quill_portrait.png | `9ca6fa43bc5a9b0e548c4a513f0719ce7052d0f48fd8f0d2de3a7cea028a059d` |
| work/tawny_quill_quill.png | `8c61787c2deb6232ec2b5943eca75d7bc3446825bc01ad6bec6868e0a58e5362` |
| work/tawny_quill_torso.png | `4d54d3a06374191a9a47d44ba49bdd732a702a40f3fd96707269534b5b4ddef4` |

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261115-01.png | `9e949049c9aa33fd940007a839b74eae67e2a75abc605bf2b954d38fce2ffdb3` | 1254 x 1254 RGB, the canonical concept, unchanged | Codex CLI `$imagegen`, session `01a1e048-945a-77a0-bee0-2464072aeeee` |
| concept/20261115-02.png | `b9b11c6d839f01dd807a28fc4223c1e95a6813fe9a704e73e51897e2da4d0cbf` | 1254 x 1254 RGB, closed-eyes edit of the canonical, unchanged | Codex CLI `$imagegen` with the canonical attached, session `01a1e057-8905-7181-b340-50cf52e4bf15` |
| work/tawny_quill_cleanup.py | `d26383fc3c2751f17e6bf5e5092020d7ef74e3e5979886ad920a647d2f8e211d` | W-3 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_cleanup.py` |
| work/tawny_quill_loop.py | `3091a17b4fa5f0ea2d643c4e8ce0c5211bc39f38fe6b9a93b066c4e3991018d6` | W-4 and W-5 script | Written by Claude Code, 2026-11-15 and 2026-11-16, from `ruby_drop_loop.py`; the same script for every avatar of this batch with the item's geometry |
| work/tawny_quill_master.png | `0f8d33a9cc34118a7eaed761eab7d7cc07cd1d39f094185f8a580358accaa9ff` | 1254 x 1254 RGBA, the cut-out master | `tawny_quill_cleanup.py` |
| work/tawny_quill_base.png | `f62d07e7e0d2e1a773acab7a85ba52433173223c844b388f0f2128a5d0ffadbf` | The portrait with the moving parts taken out (filled or cleared) | `tawny_quill_loop.py` |
| work/tawny_quill_eyes_closed.png | `6768ff6685b3ae374bc25f07852860758d5dfe83d9d486b8531e703223f65e95` | Eye region, closed (aligned edit) through the same mask | `tawny_quill_loop.py` |
| work/tawny_quill_eyes_open.png | `5631f14e43c55bb4ac7cf492837616afcc313c34fea3732a864e2d62603c9dd7` | Eye region, open (canonical concept) through the feathered eye mask | `tawny_quill_loop.py` |
| work/tawny_quill_head.png | `1093572fa77a2bea9f61a37464f1923a5d109e44ab2aaf8d72ca3a5a59d49fca` | Head and hair: the portrait weighted by the head field of the warp | `tawny_quill_loop.py` |
| work/tawny_quill_portrait.png | `6199c43a5885667ecd39f77e4c52e71acbc0f329ad8ddd92861e3145d243fd4e` | 768 x 768 RGBA portrait crop, no circle | `tawny_quill_cleanup.py` |
| work/tawny_quill_quill.png | `e5094a405fb23e96dd26e04590da57f0b875ced6579eab11eeccc3725fe1fbeb` | Moving part `quill`: the portrait through its feathered polygon | `tawny_quill_loop.py` |
| work/tawny_quill_torso.png | `50040e78a109e08eb3244248bd3758fa2b78e4b41e499c35d0310b2842e6d472` | Torso and costume: the portrait weighted by one minus the head field | `tawny_quill_loop.py` |
| export/static.png | `0c0bf6a5912de7f09261f57d6f99e79635ce7ab8879f4f512abcb7f84b719c6f` | 192 x 192 RGBA static, loop frame 9 | `tawny_quill_loop.py` |
| export/anim/0001.png to 0016.png | per file in git | 192 x 192 RGBA loop frames | `tawny_quill_loop.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-15 | `ivory_quill` concept 20261115-01.png (rejected, not committed) | P-3 with the style key and the first identity, quoted below. Sent 17:25:52 UTC; Codex session `01a1e03e-76a2-7c71-be01-f8ab1d0cf7c7` |
| 2026-11-15 | concept/20261115-01.png | P-3 with the style key, quoted below. Sent 17:36:55 UTC; `c2pa.created` 2026-11-15T17:37:43Z |
| 2026-11-15 | concept/20261115-02.png | Edit of `concept/20261115-01.png` (attached): eyes closed, everything else identical, quoted below. Sent 17:53:15 UTC; `c2pa.created` 2026-11-15T17:54:17Z |
| 2026-11-16 | work/, export/ | The scripts above |
| 2026-11-16 | export/static.png, work/tawny_quill_base.png, work/tawny_quill_cleanup.py, work/tawny_quill_eyes_closed.png, work/tawny_quill_eyes_open.png, work/tawny_quill_head.png, work/tawny_quill_loop.py, work/tawny_quill_portrait.png, work/tawny_quill_quill.png, work/tawny_quill_torso.png, export/anim/0001.png to 0016.png | Re-crop: smaller and lower per owner feedback. `CROP_SHARE` 0.86 to 0.935, `HEADROOM` 0.1 to 0.17, crop `(94, -91, 1172, 987)` to `(47, -182, 1219, 990)`; see "Re-crop, 2026-11-16" |

Prompt of `concept/20261115-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his mid-twenties with warm golden-brown skin and short side-swept black hair, clean-shaven, a long tawny-and-cream striped feather quill tucked behind one ear with its feathered end rising above his hair, and a pale ice-blue silk shirt with a high collar under a deep indigo tailcoat with silver piping, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

Codex was given it inside this instruction: `$imagegen <prompt> Generate exactly one image. Save the final PNG in the current working directory as 20261115-01.png. Do not use any API key and do not edit the image.`

Prompt of `concept/20261115-02.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his mid-twenties with warm golden-brown skin and short side-swept black hair, clean-shaven, a long tawny-and-cream striped feather quill tucked behind one ear with its feathered end rising above his hair, and a pale ice-blue silk shirt with a high collar under a deep indigo tailcoat with silver piping, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential. Edit the attached image, which is our own canonical concept of this portrait: change only one thing, the eyes are gently closed as in a relaxed blink. Everything else stays identical: the same face, pose, head position, framing, size, costume, colours, light and plain flat light grey background.
```

Codex was given it with `-i concept/20261115-01.png` inside this instruction: `$imagegen <prompt> Generate exactly one image by editing the attached image. Save the final PNG in the current working directory as 20261115-02.png. Do not use any API key.`

Prompt of the rejected `ivory_quill` concept, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Painted splash-art head and shoulders portrait of a man in his mid-twenties with warm golden-brown skin and short side-swept black hair, clean-shaven, a long white feather quill tucked behind one ear with its feathered end rising above his hair, and a pale ice-blue silk shirt with a high collar under a pearl-white tailcoat with silver piping, face centred, looking slightly toward the viewer, friendly expression, rich costume detail. Keep the outer band of the image empty of anything essential.
```

RC-2 against these prompts and the Inputs line: they name no artist, studio, game, brand, mascot, logo or real person, and the only uploaded image is our own canonical concept of this item. Checked by Claude Code on 2026-11-16.
