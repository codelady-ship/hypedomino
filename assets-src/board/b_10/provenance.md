# b_10

- Panel: B-10 Do and don't
- File: export/b_10.jpg
- Layout: six columns on `color.background.base`, 300 x 560 cells, 24 px apart, 36 px margin. Top row: do. Bottom row: the matching don't of `docs/art-direction.md` section 8. Columns, left to right:
  1. Baked text: the clean `tile.sema` back; the same back with the prototype's mock hint text `BİRİNCİ ƏL` painted into the skin (Josefin Sans, weight 600, `color.gold.hi`, 3 px stroke in `color.background.base`).
  2. Baked glow: the clean back; the same render with a halo painted into the sprite (alpha dilated 9 px, Gaussian blur 12 px, `color.gold.default`) and a flat `color.gold.default` wash over the tile at 27% alpha.
  3. Decoration in a team hue: the clean back; the skin's gold ornament (HSV hue 14 to 46 of 255, saturation above 80, value above 60) moved to the hue of `color.role.teamA`.
  4. Casino cues: the clean back; the four card suits painted over the skin's four-point stars (Segoe UI Symbol glyphs, `color.gold.default`).
  5. Recoloured copy: a skin with its own shape (`back_storm`); the `tile.sema` skin with every hue rotated by 200 of 256 and offered as a new skin.
  6. Frame 0 fallback: the finished static; a blank first frame (the skin at 6% brightness, as at the start of a fade in).
- Sources: `assets-src/cosmetic/tile/sema/work/sema_render.png` (columns 1 to 4 and 6, top, and column 2 bottom), prototype `out/back_storm.png` (column 5, top). The other bottom tiles are new renders: each changed texture was made from `assets-src/cosmetic/tile/sema/work/sema_texture.png` with Pillow and rendered with the prototype script `render_tiles.py` (mode `back`) in Blender 5.2.2 LTS, the same way as `sema_render.png`.
- Generator, account and prompts of the painted skins: `tile.sema` in `assets-src/cosmetic/tile/sema/provenance.md`; the storm skin (prototype `gen/hd-backs.png`) is not recorded yet, like the other prototype art of B-1 to B-7.
- Board panel of `docs/art-direction.md` section 9. Reference only; never shipped.
- Processing for this panel: Python 3.12 with Pillow 12.1, JPEG quality 82, on 2026-11-12.
