# effect.tile_shadow

The soft drop shadow drawn under every tile at runtime (#169, `docs/tiles.md` section 5, `docs/art-direction.md` sections 2 and 4). Authored white; the runtime tints it with `color.shadow` and sets its alpha and Y offset from `elevation`. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: handmade (no image generator was used; the sprite is computed by `work/tile_shadow.py` from the alpha of a rendered tile face)
- Authors: script written and run by an AI agent (Claude Code) on 2026-11-15. No person has done W-3
- Generator: none
- Terms: not applicable, no generator
- Canonical concept: none; the shape is the tile silhouette of `assets-src/tile/ivory/export/face/0_0.png`
- Inputs uploaded to the generator: none
- Similarity check: not applicable (a blurred rounded rectangle)
- Human work: none. RC-5 does not apply to a sprite with no generated content; the script is in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

## Export (W-5)

Run on 2026-11-15 with Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/effect/tile_shadow/work/tile_shadow.py assets-src/tile/ivory/export/face/0_0.png assets-src/effect/tile_shadow/export
```

| Property | Value |
|---|---|
| File | `export/static.png`, 176 x 304 RGBA, straight alpha |
| Colour | RGB white everywhere; the shape is in the alpha only |
| Shape | The alpha of the 128 x 256 tile face, padded by 24 px on every side and blurred with a Gaussian of sigma 6 px |
| Pivot | Centre, the same point as the tile's pivot. Drawn at the tile's scale (1 shadow pixel = 1 face pixel), so its centre sits under the tile's centre before the `elevation` Y offset |
| Runtime | Tint `color.shadow`, alpha and Y offset from `elevation.e1` for a tile at rest; a lifted tile may use a higher level. No baked colour, no baked offset |

A second run into an empty folder wrote byte-identical output. Not done: the shared effects atlas and its address (#176), import in the client.
