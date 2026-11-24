# tile.sema

The sample item of `docs/asset-workflow.md` section 6: the sky tile back skin of the approved prototype (`table-v2`, owner decision 2026-11-12, board panel B-4). Every line below is either read from a file or a record, or marked as not recorded. Recorded on 2026-11-12.

- artSource: ai_assisted
- Authors: W-1 concept generated in the project owner's ChatGPT account on 2026-11-12. W-3 clean-up and W-4 animation were done by an AI agent (Claude Code) on 2026-11-12 with the two scripts under Clean-up and Animation below; no person has done W-3 or W-4
- Generator: ChatGPT in the browser, image generation, model `gpt-image` as named by the C2PA manifest embedded in `concept/20261112-01.png` (see below). Account holder: the project owner, own ChatGPT account. Plan: Free
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read yet: an automated fetch on 2026-11-12 was refused by the site (HTTP 403). The account holder reads the terms in force on 2026-11-12 for the Free plan and records the date here (RC-1 open for the terms, O-2)
- Canonical concept: concept/20261112-01.png
- Inputs uploaded to the generator: none; one prompt, one image, no edits (read from the conversation history on 2026-11-12)
- Prompt: in the log below. It was written before P-8 and the style key of `docs/asset-workflow.md` section 2 existed, so neither was used. It asks for a "soft violet glow", which `docs/art-direction.md` section 8 keeps out of sprites; the clean-up below reduces it only in part
- Similarity check: not done (RC-3 open, O-7)
- Human work: none. Clean-up and animation are scripted image operations run by an AI agent, not human work, so RC-5 stays open; `docs/asset-workflow.md` gives no exception for scripted work. The layered master and both scripts are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 and W-4 have no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, casino cues or team hues in the art; baked glow remains around the gem (Clean-up, step 4); the crescent emblem, which the owner reviewed below
- Cultural review: 2026-11-12, the project owner (owner decision: the crescent-with-gem emblem among four-point stars is acceptable for the four launch markets under docs/localization.md section 6), approved

About the cultural review: the owner's decision was given in chat on 2026-11-12 and relayed to the agent that wrote this file; the owner confirms it by merging the pull request that adds this line. It covers the art in `concept/`, `work/` and `export/` as listed under Files. The owner is also the account holder of the concept, and `docs/asset-workflow.md` W-7 asks for the review by another account than the author's; the owner's decision is recorded as given. The self-check above is still open.

## Embedded C2PA manifest

`concept/20261112-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-12; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-12T11:29:26Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC, certificate issued by SSL.com C2PA ICA R1 2025 |

The conversation that made it is titled "Create Celestial Domino Design" and was created 2026-11-12 15:29 local time (UTC+4), which is 11:29 UTC and matches `c2pa.created`.

## Clean-up (W-3)

Run on 2026-11-12 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/tile/sema/work/sema_cleanup.py assets-src/cosmetic/tile/sema/concept/20261112-01.png assets-src/cosmetic/tile/sema/work
```

A second run into an empty folder wrote byte-identical files. The steps of `docs/asset-workflow.md` 5.1:

| Step | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | In part | The concept fills the image edge to edge, so there is no background to remove and nothing to trim. The tile canvas and pivot are open (O-6); the master keeps the concept's 887 x 1774 (1:2) and the export canvas is chosen under Animation |
| 2. Paint-over, AI artefacts | In part | No paint-over. One artefact fix: the broad violet haze around the emblem is reduced. Gold is masked (hue 15 to 60 degrees, saturation over 0.35, value over 0.35, plus bright low-saturation highlights, grown by a 9 px maximum filter) and so is the gem (radius 128 px around 478, 878). The sky luminance is low-passed (Gaussian sigma 10, normalised over the sky mask); within 300 px of 450, 870, fading to none at 360 px, sky above the median level of the ring 380 to 520 px keeps 35% of its excess. The left-right mean difference of the concept is 26.2 of 255, mostly highlight placement; the ornament was not made symmetric |
| 3. Moving parts on their own layers | Yes | `work/sema_stars.png`: the seven four-point gold stars (centres and half-sizes in `STARS` of the script), cut with a feathered star-shaped mask. `work/sema_sparkles.png`: the small sky stars, alpha from the positive fine detail (luminance minus a sigma 2 blur, above 0.04) inside the sky mask |
| 4. Nothing art-direction section 8 forbids | No | No text, numbers, rarity colour or team hue found. Glow is not fully removed: the lens flare at the gem's upper right and the bloom along the crescent remain, as the highlight mask protects them. An artist paints them out |
| 5. Silhouette and smallest-size tests | No | The smallest display size of a tile back is open (O-6, #169) |

## Animation (W-4) and export (W-5)

Run on 2026-11-12 by Claude Code with the same versions, after the clean-up:

```bash
python -I assets-src/cosmetic/tile/sema/work/sema_loop.py assets-src/cosmetic/tile/sema/work assets-src/cosmetic/tile/sema/export assets-src/tile/ivory/export/face/0_0.png
```

It writes `export/anim/0001.png` to `0016.png` and `export/static.png`: RGBA PNG, 8 bit, each frame composed at 887 x 1774 and Lanczos-resized to the canvas. The alpha of every frame is the alpha of the ivory face `assets-src/tile/ivory/export/face/0_0.png` (the third argument), so the back has the face's silhouette with its transparent rounded corners and the two swap without a jump (`docs/tiles.md` section 1); all 28 ivory faces share that alpha byte for byte. A second run into an empty folder wrote byte-identical files.

Until 2026-11-16 the export had alpha 255 everywhere, so the back showed square opaque corners next to the rounded faces (found on the phone in the Art Bench, 2026-11-16). On 2026-11-16 Claude Code added the silhouette mask to `sema_loop.py` and ran it again with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1; the colour of every pixel is unchanged, only the alpha differs (pixel 0, 0 now has alpha 0). `sema_cleanup.py` and the `work/` layers were not changed. The copy in `client/Assets/Cosmetics/remote/tile/sema/` was replaced with the new export, keeping its `.meta` files.

| Choice | Value | Source |
|---|---|---|
| Canvas | 128 x 256 | Open (O-6). Chosen so that the static and 16 frames fit the 768 x 768 ASTC 6x6 atlas that ADR 0012 measured (six by three cells of 128 x 256) and the estimate stays under the only animated-cosmetic limit the docs set, 300,000 bytes for an avatar (#171) |
| Frames | 16 | ADR 0012 first measurement (16 frames per loop) |
| Frame rate | 12 fps, a loop of 1.33 s | ADR 0012 first measurement. The rate is not stored with the loop (`SpriteLoop` holds frames only); the frame rate per class is open (O-5, O-6) |
| Loop content | Star twinkle | `docs/art-direction.md` 5.3 names the sheen sweep and a glow pulse for skins, and section 4 makes both runtime sprites, so neither is drawn into the frames. The loop carries the skin's own ambient detail, as 5.2 allows for frames: the seven gold stars grow to 118% and back in a cosine pulse 5 frames wide (the star above the gem at frame 3, the side pair at 6, the inner pair on the axis at 12, the outer pair at 15), and the sky stars dim by up to 70% in two groups of 12 px cells (seed 20261112), one with sin²(pi f / 8), one with sin²(pi f / 4), f counting from 0 |
| Static | Frame 9 | The rest frame: no star grown, no sky star dimmed, so it equals the cleaned master. `export/static.png` and `export/anim/0009.png` are the same bytes. Frame 1 is 0.017 of 255 away on average, as one pulse fades over the loop seam |
| Size | 17 sprites, 257,312 bytes | `npm run asset:check -- assets-src/cosmetic/tile/sema`, ASTC 6x6 estimate; tile skins have no limit yet (O-5) |

Not done: the reduce-motion and low-level check of 5.2 (Unity), board panel B-9, and a look at the loop at the tile's real display size.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261112-01.png | `f6fb7abb6f8148d76c89be1b3a099f6afe342911f06b7c7b86876dec62673ef3` | 887 x 1774 RGB, the generator's download, unchanged | Byte copy of prototype `gen/hd-back-sema.png` |
| work/sema_texture.png | `0605711fe48cdf668c9aa619187267bb5f930e1e88d7185dd8f6808224c72dc0` | 400 x 800 RGB texture for the render | Lanczos downscale of the concept, no other change; identical pixel for pixel to a Pillow 12.1 `LANCZOS` resize of the concept. Byte copy of prototype `gen/skin-sema.png` |
| work/sema_render.png | `f091c3f9b6c01c8d5db78c68f3634c901bd3aee4e2016f0c87d17ad536e994ee` | 300 x 560 RGBA render of a tile with the skin on its back | Blender Cycles, prototype script `render_tiles.py` (sha256 `16cdd236056576dc42438a2179db3e792f874f315cc9c582eac9b9f5475340ec`), mode `back`, texture `work/sema_texture.png`: 96 samples, denoised, orthographic camera, transparent film, the modelled ivory tile body. File metadata: rendered 2026-11-12 15:30:32 local time. Byte copy of prototype `out/back_sema.png` |
| work/sema_cleanup.py | `cc184ed7f75b4cca4df832d0f1fd1c4726e14ad3db45cf4660113ca7545101a1` | W-3 script | Written by Claude Code, 2026-11-12 |
| work/sema_master.png | `ffb3e2fb382a392b7ff7e7d21ee9e443af6c04308fe6c884c138b0142e19c518` | 887 x 1774 RGB, the cleaned master (base layer) | `sema_cleanup.py` |
| work/sema_stars.png | `524e9e8d7257ccf30b7f7ab2fbdd2f5108a5b84172447b0f31c048a2e70e207f` | 887 x 1774 RGBA, the gold star layer | `sema_cleanup.py` |
| work/sema_sparkles.png | `d05969fc7b98f773bdac31c83d92733e863d3cd75d3934e44305e1ed4a782e6c` | 887 x 1774 RGBA, the sky star layer | `sema_cleanup.py` |
| work/sema_loop.py | `6e8fdcabf0f1376ac07d17494916c2643ae2925d30cb36c6a923eddeaeb8cbc7` | W-4 and W-5 script | Written by Claude Code, 2026-11-12; silhouette mask added 2026-11-16 |
| export/static.png | `ab3d67654b96755a5ad640c68f47ed6424263fe4efb7ddb6b099c906b8c3ff59` | 128 x 256 RGBA static, alpha of the ivory face | `sema_loop.py`, frame 9 |
| export/anim/0001.png to 0016.png | per file in git | 128 x 256 RGBA loop frames, alpha of the ivory face | `sema_loop.py` |

Reproduced on 2026-11-12 with Blender 5.2.2 LTS (`blender -b --factory-startup -P render_tiles.py -- <out> back work/sema_texture.png sema`): the new render matches `work/sema_render.png` with a mean channel difference of 0 and a maximum of 9 of 255 (denoiser noise).

Not imported: prototype `gen/preview-sema.jpg`, a 300 x 600 JPEG of the same image (no other use), and `web/img/back_sema.png`, the render cropped to 264 x 514 for the prototype page.

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-12 | concept/20261112-01.png | ChatGPT `gpt-image`, conversation "Create Celestial Domino Design", the only prompt, quoted below |
| 2026-11-12 | work/sema_texture.png | Lanczos downscale of the concept to 400 x 800 |
| 2026-11-12 | work/sema_render.png | `render_tiles.py` mode `back` with `work/sema_texture.png` |
| 2026-11-12 | work/sema_master.png, work/sema_stars.png, work/sema_sparkles.png | `sema_cleanup.py` on the concept |
| 2026-11-12 | export/static.png, export/anim/ | `sema_loop.py` on the `work/` layers |
| 2026-11-16 | export/static.png, export/anim/ | `sema_loop.py` again, now masked by the alpha of `assets-src/tile/ivory/export/face/0_0.png` |

Prompt of `concept/20261112-01.png`, verbatim:

```text
Create an image: one luxury domino tile back design, a flat, front-on, perfectly straight vertical panel with an exact 1:2 width-to-height ratio, filling the whole image edge to edge (no tile edges, no perspective, no shadow, no background). Legendary-tier premium game skin art, glossy, extremely detailed, symmetrical ornament centred. Theme "Celestial": deep amethyst-violet night-sky lacquer with tiny scattered stars, antique gold filigree, a large golden crescent moon emblem in the centre holding a faceted violet amethyst gem, small gold four-point stars around it, soft violet glow. No text, no letters, no numbers, no dots or pips.
```

RC-2 against this prompt and the Inputs line: it names no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-12.
