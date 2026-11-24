# frame.star_compass

Static frame of set 1 (#173), an ornamental ring made with template P-4 of `docs/asset-workflow.md` section 2. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-14 while the work was done.

- artSource: ai_assisted
- Authors: W-1 concept generated in the project owner's second ChatGPT account on 2026-11-14 by an AI agent (Claude Code) driving the owner's browser. W-3 clean-up and W-5 export by the same agent with `work/star_compass_cleanup.py`. No person has done W-3
- Generator: ChatGPT in the browser, Create image tool, model `gpt-image` as named by the C2PA manifest embedded in the concept (below). Account holder: the project owner, second ChatGPT account (the first one reached its image limit during #172), signed in to the second browser on the owner's computer. Plan: Free (shown under the account name in the ChatGPT sidebar on 2026-11-14)
- Terms: OpenAI Terms of Use, `https://openai.com/policies/terms-of-use/`, not read (RC-1 open for the terms, O-2). The account holder reads the terms in force on 2026-11-14 for the Free plan and records the date here
- Canonical concept: concept/20261114-01.png
- Inputs uploaded to the generator: none; one prompt, one image, no edits, in a new conversation (`chatgpt.com/c/6af85621...`, titled "Create Astrolabe Portrait Frame" by ChatGPT)
- Prompt: P-4 with the style key of section 2.2 pasted verbatim; `{theme in plain words}` = "a ring of interlocking brass astrolabe circles with small star-shaped studs and sapphire cabochons", `{colour words}` = "polished brass, deep sapphire blue and ivory"; quoted below
- Concept against the prompt: Matches the prompt: a brass ring with interlocking outer arcs, four sapphire cabochons in four-point star mounts at the top, bottom and sides, small four-point brass star studs between them, and blue enamel and ivory panels in the arcs. The stars are four-pointed, not a symbol.
- Similarity check: not done (RC-3 open, O-7)
- Human work: none. Clean-up and export are scripted image operations run by an AI agent, so RC-5 stays open; the master and the script are in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. The asset author signs it, and W-3 has no human author yet. Notes from the agent for whoever signs: no text, letters, numbers, flags, emblems, religious symbols, weapons, casino cues or team hues seen in the art; no glow or drop shadow painted in
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

Proposed for the admin draft, not decided: rarity `legendary`, source `shop:offer.frame_star_compass`. The offer id is a placeholder until the shop record exists (V-15).

## Embedded C2PA manifest

`concept/20261114-01.png` carries a C2PA manifest in a `caBX` PNG chunk. Read as text on 2026-11-14; the signature was not validated with a C2PA tool.

| Field | Value |
|---|---|
| Action | `c2pa.created`, 2026-11-14T12:38:16Z |
| Software agent | `ChatGPT`, version `gpt-image` |
| Digital source type | `trainedAlgorithmicMedia` (IPTC) |
| Further actions | `c2pa.converted`, `c2pa.watermarked.unbound`, same time |
| Claim generator | `OpenAI Media Service API` |
| Signer | OpenAI OpCo, LLC |

The prompt was sent at 12:38:06 UTC on 2026-11-14 (agent clock), shortly before `c2pa.created`.

## Clean-up (W-3) and export (W-5)

Run on 2026-11-14 by Claude Code with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/cosmetic/frame/star_compass/work/star_compass_cleanup.py star_compass assets-src/cosmetic/frame/star_compass/concept/20261114-01.png assets-src/cosmetic/frame/star_compass/work assets-src/cosmetic/frame/star_compass/export
```

It printed background `[204.0, 204.0, 206.0]`, box `(-20, -35, 1272, 1256)`, opening radius 0.716 of half canvas, canvas 512. A second run into an empty folder wrote byte-identical files.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Background, trim, class canvas and pivot | Yes, canvas open | The flat grey background is the median of a 6 px strip on all four edges. Background is: pixels within 30 of it (RGB distance) connected to an edge, the region of such pixels that holds the image centre (the open centre), and every region of at least 20 px within 12 of it (gaps in the filigree). Parts under 200 px are dropped as specks; the rest is eroded by 1 px, feathered (Gaussian sigma 0.8) and unmixed from the grey at the edge. Export: a square centred on the frame's bounding box with 2% margin on each side, scaled to 512 x 512, pivot at the centre. The opening radius is printed as a share of half the canvas so the avatar circle can be matched to it; the class canvas and the frame band are open (O-6, #168) |
| 2. Paint-over, AI artefacts | No | No paint-over |
| 3. Moving parts on their own layers | Not needed | Static item; the animated variant is later work |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow, drop shadow, rarity colour or team hue seen; the static shows the metal and gems at rest (`docs/art-direction.md` section 7) |
| 5. Silhouette and smallest-size tests | In part | The agent drew the static around an avatar portrait (`avatar.court_poet` set inside the opening) at 256, 120 and 64 px on a dark backdrop. At 64 px the double ring and the four blue stones read; the small studs disappear. The smallest display size is open (#168, O-6), so the test is not final |

`npm run asset:check -- assets-src/cosmetic/frame/star_compass`: 1 sprite, ASTC 6x6 estimate 118,336 bytes, no limit set for a static frame (O-5).

Not done: W-6. `npm run asset:add` refuses the item until the Cultural review line above ends in `approved`, so nothing is copied to `client/Assets/Cosmetics/` and no Unity import was run.

## Files

| File | sha256 | What it is | How it was made |
|---|---|---|---|
| concept/20261114-01.png | `094d8706116900a8bb2ea246810471406b3451cea3db4547ef92a89ad0adaf96` | 1254 x 1254 RGB, the generator's download, unchanged | ChatGPT image viewer Download button, copied byte for byte |
| work/star_compass_cleanup.py | `826b01ffc14fd90cfe16c3271cfd1375648ffce4ea6dd2217eaf30798b9e4ed6` | W-3 and W-5 script | Written by Claude Code, 2026-11-14; the same script for every frame of this PR |
| work/star_compass_master.png | `8a68be64ce7842ec03fa4cd7d34b1acec0cbe616f5f2c0c9943e6a5aaa3f36e9` | 1254 x 1254 RGBA, the cut-out master | `star_compass_cleanup.py` |
| export/static.png | `537ceb822b9eb1c69ae9ce7efeb701d7718534b22dc935a92e3171af13cea87a` | 512 x 512 RGBA static | `star_compass_cleanup.py` |

| Date | File | Prompt or edit |
|---|---|---|
| 2026-11-14 | concept/20261114-01.png | P-4 with the style key, the only prompt of the conversation, quoted below. Sent 12:38:06 UTC by Claude Code; `c2pa.created` 2026-11-14T12:38:16Z |
| 2026-11-14 | work/star_compass_master.png, export/static.png | `star_compass_cleanup.py` on the concept |

Prompt of `concept/20261114-01.png`, verbatim:

```text
Original 2D game art for an all-ages mobile game with a premium, aristocratic look. Painted, semi-realistic rendering with soft volume and rich materials: polished gold and brass, ivory, deep lacquer, gemstone, crystal, silk and wool. Fine ornament inside a clear, readable outer shape. One warm light from above. No text, letters, numbers, logos, flags, emblems or religious symbols. No weapons. No glow, no drop shadow, no cast shadow, no background scenery. Isolated on a plain flat light grey background that does not appear anywhere in the subject. Ornamental round frame for a circular portrait, a ring of interlocking brass astrolabe circles with small star-shaped studs and sapphire cabochons, symmetric, open centre left empty, thin enough not to cover a face. Palette: polished brass, deep sapphire blue and ivory.
```

RC-2 against this prompt and the Inputs line: it names no artist, studio, game, brand, mascot, logo or real person, and no image was uploaded. Checked by Claude Code on 2026-11-14.
