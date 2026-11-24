# ui.icons

The interface icon set of `docs/art-direction.md` section 5.7 (#168): Iconsax, bulk style only, exported white with the bulk second tone kept as reduced alpha. The list, sizes and use of every icon are in `docs/ui-kit.md`. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 while the work was done.

- artSource: handmade (no image generator was used; the PNGs are rasterized from the vendor's vector icons by `work/icons_rasterize.py`)
- Authors: icon artwork by the Iconsax vendor (npm package author field: `Iconsax`). Selection and scripts by an AI agent (Claude Code) on 2026-11-16. No person has reviewed the selection yet
- Generator: none
- Source package: npm `iconsax` 0.1.1, tarball `https://registry.npmjs.org/iconsax/-/iconsax-0.1.1.tgz`, integrity `sha512-/HXomG+SuhytKH/Yw4ZdunDGyWP02Osuk/pjony1k9CCMOsg7dUpnomu7HZUOHNu0yK0b3phErjH2IgUTJFjPA==`, SHA-256 of the tarball `812d011ad515f0b0f1bec5830998a7f056785129c072e66938124818e0981dd5`. Fetched with `npm pack iconsax` into a scratch folder on 2026-11-16; it is not a dependency of this repository
- Licence: the package's `package.json` declares `"license": "MIT"`; the tarball ships no LICENSE file. Its README says the free icons (1200+) need no key; the Pro tier is not used. The terms for the icon artwork on the vendor's site were not read: an owner item before the icons ship (`docs/art-direction.md` section 10)
- Terms: not read (see Licence)
- Canonical concept: none
- Inputs uploaded to the generator: none
- Similarity check: not applicable (third-party icon set used as published)
- Human work: none on the artwork. The vendor SVGs are kept verbatim in `work/icons_source.json`; selection in `work/icons_list.json`
- Self-check (docs/localization.md 6.1 step 1): not done. Icons with letters, digits or currency signs were left out on purpose (a `$` coin, a `24` support badge, a `%` offer and promo ticket, a language icon with a CJK glyph), as were brand logos and seasonal or religious motifs of the package
- Cultural review: not done

## Export (W-5)

Run on 2026-11-16 with Python 3.12.10, NumPy 2.4.6 and Pillow 12.1.1, from the repository root, with the tarball unpacked to `<scratch>/package`:

```bash
python -I assets-src/ui/icons/work/icons_extract.py <scratch>/package assets-src/ui/icons/work/icons_list.json assets-src/ui/icons/work/icons_source.json
python -I assets-src/ui/icons/work/icons_rasterize.py assets-src/ui/icons/work/icons_source.json assets-src/ui/icons/export
```

| Property | Value |
|---|---|
| Files | `export/<key>.png`, 127 icons, keys and sources in `work/icons_list.json` |
| Canvas | 96 x 96 RGBA, straight alpha, the vendor's 24-unit view box at 4 px per unit |
| Colour | RGB white everywhere. The solid tone has alpha 1, the bulk second tone keeps the vendor's opacity (0.4 for most icons) as alpha, so a vertex-colour gold gradient tints both tones |
| Rasterizer | Own script, no SVG library: `work/icons_rasterize.py` parses the path data (move, line, cubic and quadratic curves, arcs), flattens curves, fills each path with its fill rule (non-zero, or even-odd where the SVG says so) at 16 x 16 samples per pixel and composites the paths with their opacity. It refuses a view box other than 24 x 24, a stroke, a transform or a clip path that does not cover the view box, so an icon it cannot draw exactly fails instead of drawing wrong. No resvg, cairosvg or Inkscape was installed on the machine |
| Smallest display size | 36 units: the icon row of the B-1 prototype (emoji, sound and settings, the same three Iconsax bulk icons), measured on `assets-src/board/b_1/export/b_1.jpg` at its 1080-unit short side (`docs/ui-kit.md`) |

Both scripts were run twice into empty folders: byte-identical output, and identical to the committed files. Not done: the shared icon shadow and gold gradient in a screen (the shadow sprite is `ui/kit`), the per-icon motion of art-direction 5.7, the vendor terms, the cultural review.
