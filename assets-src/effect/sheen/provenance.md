# effect.sheen

The one shared white sheen streak of `docs/art-direction.md` section 4: masked by the asset it crosses (gold rims, frames, tile back skins) and moved by a transform tween; no per-item material (#169, `docs/tiles.md` section 5). Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: handmade (no image generator was used; the sprite is computed by `work/sheen.py`)
- Authors: script written and run by an AI agent (Claude Code) on 2026-11-15. No person has done W-3
- Generator: none
- Terms: not applicable, no generator
- Canonical concept: none
- Inputs uploaded to the generator: none
- Similarity check: not applicable (two soft bands)
- Human work: none. The script is in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done
- Cultural review: not done

## Export (W-5)

Run on 2026-11-15 with Python 3.12.10, NumPy 2.4.6 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/effect/sheen/work/sheen.py assets-src/effect/sheen/export
```

| Property | Value |
|---|---|
| File | `export/static.png`, 64 x 256 RGBA, straight alpha |
| Colour | RGB white everywhere; greyscale in the alpha only, so a tint from a token colours it (art-direction section 2) |
| Shape | A vertical streak: a wide soft band (Gaussian, centre at 40% of the width, sigma 11% of the width, full strength) beside a thin one (centre 72%, sigma 3.5%, 55% strength), faded to nothing over the top and bottom 18% with a smoothstep. Drawn at 4x and box-filtered down |
| Pivot | Centre. The runtime rotates it to its sweep angle and moves it across the masked asset; length and angle are the tween's, not the sprite's |

A second run into an empty folder wrote byte-identical output. Not done: the shared effects atlas, its address and the tween timing (#176, `docs/ui-motion.md` section 7), import in the client.
