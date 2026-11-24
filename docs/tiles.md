# Tiles

Status: proposal; the owner approves it by merging. The class spec of the tile (#169): the shared face art, its canvas, pivot, smallest display size, export layout and addresses, how a new face set is made, and the runtime shadow and sheen sprites. Look and material rules are `docs/art-direction.md` sections 3 and 4; the pipeline, naming and provenance are `docs/asset-workflow.md`; the board layout that decides a tile's size on screen is `docs/ui-motion.md` (Match layout, Board); the client side is `UI/Board/TileFaces`, `UI/Board/TileFaceSet`, `UI/Board/BoardTilesView` and the set folders under `UI/Board/Tiles/` (section 2). Tile back skins are cosmetics (`docs/cosmetics.md`); this document sets only the canvas they share with the face.

## 1. Face set

The face is shared art for every player (`docs/cosmetics.md` section 2): one sprite per tile of the set (R-1), 28 in all, upright with the lower value on the top half (`docs/ui-motion.md`, Board).

| Item | Value |
|---|---|
| Default set | `ivory`, in `assets-src/tile/ivory/` |
| Look | Ivory face with a soft bevel highlight, pure dark pips, a dark divider line with a brass pin at its centre, one warm key light from the upper left (`docs/art-direction.md` section 4). Panel B-3 is the reference |
| Canvas | 128 x 256 px per face, the tile's top face filling the canvas edge to edge; the corners outside the bevel are transparent. The same canvas as a tile back skin (`tile.sema`), so face and back swap without a jump |
| Pivot | Centre |
| Outline | None; the bevel separates the tile from the carpet |
| Smallest display size | 41.7 canvas units on the short side (section 3), which is 32 x 64 px on the smallest phone of the device matrix |
| Proportions | Short side W, long side 2W. Bevel 0.112 W. Pip diameter 0.176 W, pip grid step 0.27 W from the centre of each half, divider 0.74 W long and 0.036 W wide, pin diameter 0.15 W |
| Pip layout | A fixed pattern per value on a 3 x 3 grid of each half, so a value reads by pattern (art-direction section 3): 1 centre; 2 and 3 on the falling diagonal from the upper left; 4 the corners; 5 the corners and the centre; 6 two columns of three |
| Order | `0_0` to `0_6`, `1_1` to `1_6`, up to `6_6`: the set order of `TileFaceSet` |
| Palette | The face's own (art-direction section 2): ivory, near-black pips and divider, brass pin. It is painted into the sprite, never tinted |

### 1.1 Files

| Path in `assets-src/tile/<slug>/` | What it is |
|---|---|
| `provenance.md` | The record of `docs/asset-workflow.md` section 3 |
| `work/<slug>_set.json` | The set's look: materials of face, pips, divider and pin, and the light |
| `work/<slug>_render.py` | Blender script: renders the 28 faces at four times the canvas into a folder outside the repository |
| `work/<slug>_export.py` | Box-filters the renders to the canvas, straight alpha, into `export/face/` |
| `work/<slug>_check.py` | The readability check of section 3 |
| `export/face/<low>_<high>.png` | The 28 faces, 128 x 256 RGBA, straight alpha, sRGB |

The render is deterministic: Cycles on the CPU with a fixed seed, fixed thread count, no adaptive sampling and no denoiser, so two runs give the same pixels and the export gives the same bytes.

### 1.2 A new face set

A face set changes material and colour only; the layout above is the same for every set, so a value reads the same on every set.

1. Copy `assets-src/tile/ivory/` to `assets-src/tile/<slug>/` and rename the four work files to `<slug>_*`. Delete the copied `export/`.
2. Edit only `work/<slug>_set.json`:

| Key | Meaning |
|---|---|
| `materials.face` | `color` (linear RGB), `roughness`, `coat`, `subsurface`, `grainScale`, `grainStrength` of the face |
| `materials.pip`, `materials.divider` | `color`, `roughness`, `specular`; keep them near black with `specular` 0 so pips stay pure dark |
| `materials.pin` | `color`, `roughness`, `metallic` of the pin |
| `light.key`, `light.fill` | `energy`, `size`, `location`, `rotation` (degrees) of the two area lights; keep the key at the upper left (art-direction section 4) |
| `light.world` | `color` and `strength` of the ambient light |

3. Render, export and check (Blender 5.2 or later; from the repository root):

```bash
blender --background --factory-startup --python-exit-code 1 --python assets-src/tile/<slug>/work/<slug>_render.py -- assets-src/tile/<slug>/work/<slug>_set.json <renders>
python -I assets-src/tile/<slug>/work/<slug>_export.py <renders> assets-src/tile/<slug>/export
python -I assets-src/tile/<slug>/work/<slug>_check.py assets-src/tile/<slug>/export/face assets-src/cosmetic/table/starter/export/static.png assets-src/effect/tile_shadow/export/static.png docs/design-tokens.json <out> --width 32 28
```

4. The set passes when the check reports no face with a wrong pip count at both widths, and the pips stay the darkest and the face the lightest part of the tile. Record the numbers in `provenance.md`.

Layout constants (canvas, proportions, pip grid) live at the top of `<slug>_render.py`. Changing them changes the layout of every set and is a change to this document first.

## 2. Addresses

| Asset | Address | Group | Notes |
|---|---|---|---|
| Face set `<slug>` | `tile/<slug>/faces`, a `TileFaceSet` asset with the 28 sprites in set order | `tile-<slug>`; the default set `ivory` is built in, in the local group `cosmetic-builtin`, as every match needs it before the first deal | The 28 faces in one Sprite Atlas, `faces.spriteatlasv2`, drawn with the default UI material so the board batches (`docs/ui-motion.md`, Board) |
| Tile shadow | `effect/tile_shadow/static` | The shared effects atlas (#176) | Section 5 |
| Sheen | `effect/sheen/static` | The shared effects atlas (#176) | Section 5 |

In the client a set lives in `client/Assets/HypeDomino/UI/Board/Tiles/<slug>/`: the faces copied from `export/face/` into `face/`, the atlas `faces.spriteatlasv2` and the set `faces.asset`. `HypeDomino.Editor.TileFaceContent` (menu HypeDomino > Import Tile Faces; batch `ImportAllAndExit` through the `unity-editor.yml` workflow) imports the default set: every face a single, full-rect, centre-pivot sprite, sRGB, alpha is transparency, ASTC with the block of `ProjectSetup.SpriteFormat` (6x6) on Android and iOS; the atlas without rotation or tight packing, padding 4, largest page 2048; the set in set order; the address above. `TileFaceContentTests` check it. `UI/Board/BoardTilesView` puts the faces on the laid tiles of `ChainLayout`.

Mipmaps: off, on the faces and the atlas. The board is a screen space overlay at the native resolution (section 3), so a face is drawn at one size per board and never recedes in depth, where mipmaps help most. The readability check of section 3 already samples the 128 px sprite without mipmaps and counts every face right at 32, 28, 24, 20 and 16 px, so mipmaps buy no readability the faces need. Without them the atlas is a third smaller, and the lower mip levels cannot bleed one face into its neighbour across the 4 px padding. This is the import of every UI sprite in the project (`docs/asset-workflow.md` 5.4). If a device check (#191) shows shimmer on a moving tile at the smallest size, mipmaps are the change to try, with the padding raised to match.

Atlas size: 28 sprites of 128 x 256 fill 7 of 8 columns and 4 rows of a 1024 x 1024 page only with no padding; with the default padding they need a 2048 x 1024 page. Texture estimate by the size rule of `docs/asset-workflow.md` 5.4 (each sprite at 16 bytes per ASTC 6x6 block): 28 x 22 x 43 x 16 = 423,808 bytes. No tile limit is set yet (O-5).

## 3. Smallest display size and readability

The smallest phone of the device matrix (`docs/performance.md` section 1) is the iPhone XR: 1792 x 828 px at 326 ppi, 6.06 in diagonal (Apple technical specifications). Every other phone in the matrix has a short side of 1080 px or more, and the XR's physical short side (about 2.54 in) is the smallest in the matrix together with the other 6.1 in iPhones.

The match canvas scales its 1080-unit short side to the display (`docs/ui-motion.md` section 1), so one unit is 828 / 1080 = 0.767 px on the XR. Its safe area, in units, is 2337 x 1025 in landscape (44 pt insets left and right, 21 pt at the bottom) and 1080 x 2134 in portrait (44 pt top, 34 pt bottom). `UI/Match/MatchLayout` gives the board region from the tokens (`spacing.md`, `size.seatPanelWidth`, `size.seatPanelHeight`, `size.handHeight`):

| Mode and orientation | Board region (units) |
|---|---|
| 2v2 landscape | 1516 x 521 |
| 2v2 portrait | 488 x 1630 |
| 1v1 landscape | 2076 x 521 |
| 1v1 portrait | 1048 x 1630 |

The tile is smallest when the board is fullest: at most 25 tiles in 2v2 (all 28 are dealt, R-3, and the hand ends when a player places their last tile, R-32, so the other three still hold at least one each) and 27 in 1v1 (the last boneyard tile is never drawn, R-23). `ChainReadabilityTests` plays 400 full boards per row (seed 169): a random opening, then random legal placements on any open end, with the spinner of R-17 and its arms. The chain layout of `UI/Board/ChainLayout` draws their tiles with these short sides:

| Mode and orientation | Smallest | Median |
|---|---|---|
| 2v2 landscape | 52.1 units (40.0 px) | 73.9 units (56.7 px) |
| 2v2 portrait | 48.8 units (37.4 px) | 61.0 units (46.8 px) |
| 1v1 landscape | 52.1 units (40.0 px) | 86.9 units (66.6 px) |
| 1v1 portrait | 63.9 units (49.0 px) | 87.3 units (67.0 px) |

The test fails when any of these boards draws a tile short side under 41.7 units, the 32 px of the check below. The largest tile size (the layout's scale cap) is not decided, so the tile never gets smaller than this from the cap.

The match canvas is a screen space overlay, which renders at the native resolution on every graphics level (`docs/performance.md` 1.1), so 32 px is the size on screen. The check also runs at 28 px, the size if the board were drawn by the camera at the Low level's 720 px cap.

`work/<slug>_check.py` downsizes each face to the test width two ways: a box filter (a sprite with mipmaps) and plain bilinear sampling of the 128 px sprite (no mipmaps, what the GPU does with the current sprite import of `docs/asset-workflow.md` 5.4). In each half it counts the dark spots inside the pip area; a face passes when the counts equal its two values. It also writes the 28 faces at their real pixel size on the default table with the runtime shadow, and a 4x nearest-neighbour enlargement to look at.

Result for `ivory` on 2026-11-15 (`assets-src/tile/ivory/provenance.md`):

| Width | Pip diameter | Gap between pips | Ivory luminance | Darkest pip | Wrong pip counts |
|---|---|---|---|---|---|
| 32 px | 5.6 px | 3.0 px | 225 | 11 | 0 of 56 |
| 28 px | 4.9 px | 2.6 px | 225 | 11 | 0 of 56 |

56 is the 28 faces under both samplings. The same check at 24, 20 and 16 px also counts every face right. On `table.starter` the ivory face has a contrast of 9.7:1 against the median of the centre field.

## 4. Pip readability on screen

At 32 px, a pip is 5.6 px across and two neighbouring pips are 3.0 px apart. On the XR that is 0.44 mm, about 5 minutes of arc at 30 cm, the size of a letter on the 20/20 line of an eye chart: readable for normal vision and at its limit. 32 px is the floor `ChainReadabilityTests` holds for full boards on the smallest phone; the smallest tile of its boards is 37.4 px, with 6.5 px pips, and the median full board shows pips at 8.2 px or more. A larger floor for the tile is a layout decision (the scale cap and the board region), not an art one. The check on a real XR is part of #191.

## 5. Runtime layers

Authored white or greyscale and tinted at runtime (`docs/art-direction.md` sections 2, 4 and 6). Neither is baked into the face.

| Sprite | Source | Canvas and pivot | Runtime |
|---|---|---|---|
| Tile shadow | `assets-src/effect/tile_shadow/`, made from the alpha of a face | 176 x 304, centre; the tile's silhouette padded by 24 px and blurred (sigma 6 px), drawn at the tile's scale under the tile | Tint `color.shadow`; alpha and Y offset from `elevation.e1` at rest, a higher level while a tile is lifted or travels. `BoardTilesView` gives every tile slot one shadow `Image` from a fixed pool of the same size, with the shadow sprite from the effects atlas, the tile's rotation, and the tile's position moved down by `TokenElevation.E1OffsetY` on the board axis only. The shadow images sit in their own layer below the tile images, so all shadows batch as one draw call from the effects atlas and all faces as one from the face atlas: one overdraw layer for the shadows |
| Sheen | `assets-src/effect/sheen/` | 64 x 256, centre; a soft white streak with a thin second line, faded at both ends | The one shared streak, masked by the asset it crosses and moved by a transform tween (art-direction section 4). Tile faces do not get the sheen; back skins, frames and gold rims do |

## 6. Open items

- Wiring `BoardTilesView` into the match screen with the board's `Image` pool and `TilePlaceLayer`; the match screen is not assembled yet.
- The lifted and travelling shadow levels (`elevation.e2`, `elevation.e3`) during a tile's travel; laid tiles use `e1` (section 5).
- Size and placement of the face-down rows beside each seat and the hand tile size (`docs/art-direction.md` section 10).
- The board's scale cap (`docs/ui-motion.md` section 9).
- The shared effects atlas and the sheen's sweep timing (#176).
