# effect.vfx

The shared effect sprites of `docs/art-direction.md` section 6 and the rarity halos of `docs/rarity.md` 2.2 (#176): every glow, burst, trail, impact and confetti shape the effects of `docs/ui-motion.md` section 7 draw. Each is white or greyscale and is tinted at runtime with a token through vertex colour. How the effects use them, and the budget table, are `docs/vfx.md`. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 while the work was done.

- artSource: handmade (no image generator was used; every sprite is computed by `work/vfx.py`)
- Authors: script written and run by an AI agent (Claude Code) on 2026-11-16. No person has done W-3
- Generator: none
- Terms: not applicable, no generator
- Canonical concept: none
- Inputs uploaded to the generator: none
- Similarity check: not applicable (soft discs, rings, rays, a four-point star, a padlock seal, flakes)
- Human work: none. RC-5 does not apply to sprites with no generated content; the script is in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

## Export (W-5)

Run on 2026-11-16 with Python 3.12.10, NumPy 2.4.6 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/effect/vfx/work/vfx.py assets-src/effect/vfx/export
```

Every shape is drawn at 4x and box-filtered down; alpha under 2 of 255 is set to 0, so a tight sprite mesh hugs the shape. Straight alpha, sRGB, RGB white unless the row says greyscale. Pivot: centre for all.

| File | Pixels | Shape | Used for |
|---|---|---|---|
| `glow.png` | 128 x 128 | Soft disc: Gaussian falloff (sigma 0.38 of the radius) shifted to reach 0 at the edge | Event glow behind a score popup, banner or badge; the primary button's `glow.soft` (`docs/ui-kit.md` 2) |
| `halo.png` | 256 x 256 | Soft ring: rises from 0.40 to its peak at 0.66 of the radius (the owner's edge at `glow.intense` scale 1.5) and falls to 0 at the edge | Rarity halo around a frame or behind an item card |
| `flow_halo.png` | 256 x 256 | The halo at 60% with one bright arc at the top (Gaussian, sigma 0.38 rad), screen-combined | Rarity flow halo; the arc's rest angle is the top (`docs/rarity.md` 2.2) |
| `star.png` | 64 x 64 | Four-point star: two tapered rays on the axes and a soft core | Score and win bursts |
| `spark.png` | 32 x 32 | Bright core with a soft surround | Bursts, sparkles |
| `dust.png` | 32 x 32 | Very soft mote, 80% peak alpha, no core | Warm dust of `tilePlace` and `block`; tinted warm by the effect |
| `streak.png` | 128 x 32 | Tapered streak: thin faded tail on the left, full head on the right | Trail of points flying to the meter and of a travelling tile |
| `ring.png` | 128 x 128 | Thin ring at 0.8 of the radius | Ring pulse of `spinnerFormation` and the `block` impact |
| `rays.png` | 256 x 256 | Sixteen rays, alternating long and short, hollow centre | `rankUp` rays behind the badge |
| `stamp.png` | 256 x 256 | Round seal with a notched rim and a padlock with a keyhole cut out; solid white | `block` lock stamp |
| `flake_paper.png` | 24 x 32 | Rounded paper rectangle with a greyscale curl across it | `matchWin` confetti in the winning team fill |
| `flake_leaf.png` | 32 x 32 | Irregular eleven-point leaf with greyscale facets | `matchWin` gold-leaf confetti in `color.gold.default` |

A second run into an empty folder wrote byte-identical files.

## In the client

The twelve files are copied unchanged into `client/Assets/HypeDomino/UI/Effects/Sprites/`, with `assets-src/effect/sheen/export/static.png` as `sheen.png` and `assets-src/effect/tile_shadow/export/static.png` as `tile_shadow.png`, and packed into the one shared effects atlas (`docs/vfx.md` section 3).

Not done: the reference board review on `color.surface.default` and `color.surface.raised`, the smallest-display-size check (section 3 of art direction), and the cultural review.
