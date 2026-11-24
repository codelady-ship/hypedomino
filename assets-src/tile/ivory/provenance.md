# tile.ivory

The shared face set of the double-six set (#169): 28 ivory faces, `0_0` to `6_6`, the default and so far the only face set. Class spec: `docs/tiles.md`. Every face is rendered from a 3D model by one deterministic Blender script; no image generator was used. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-15 while the work was done.

- artSource: handmade (no image in `work/` or `export/` came from a generator)
- Authors: the renderer is adapted by an AI agent (Claude Code) on 2026-11-15 from `render_tiles.py` of the owner-approved prototype `table-v2` (board panels B-1 and B-3, owner decision 2026-11-12), kept by the art team outside this repository. No person has done W-3
- Generator: none
- Terms: not applicable, no generator
- Canonical concept: none; the reference is board panel B-3 (`assets-src/board/b_3/`), the prototype's renders of seven faces
- Inputs uploaded to the generator: none
- Similarity check: not applicable to the pattern (domino pips are a common layout); the face is our own model and render
- Human work: none. The model, materials and light are code in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done. Notes from the agent for whoever signs: plain ivory faces with black pips, a black divider and a brass pin; no text, no symbols, no team or role hue
- Cultural review: not done. W-7 asks for it by another account than the author's (docs/localization.md 6.1)

## From the prototype to this set

`work/ivory_render.py` keeps the prototype's tile model: the same tile proportions, bevel, pip grid, pip size, divider and pin, the same pip layout per value, and the prototype's materials and two area lights. Changes, and why:

| Change | Why |
|---|---|
| Pips and divider are flat inlays on the face, not discs at the bottom of carved holes; their material has no specular and roughness 1 | The carved holes rendered as grey discs with a light ring in a top-down view; B-3 and art-direction section 4 ask for pure dark pips |
| Standard view transform, no look, exposure 0, gamma 1 (as `docs/asset-workflow.md` 5.3 sets for Blender exports); the prototype used a contrast look and exposure -0.8 | One documented output setting. The face colour and light energies were tuned to match B-3 under it: face base colour 0.86, 0.82, 0.72 (the prototype's 0.82, 0.68, 0.45 rendered beige under this transform, about RGB 220, 199, 164 mid-face, against about 227, 226, 224 on B-3), key 0.72, fill 0.1, world 0.2 (the prototype's 1.1, 0.12, 0.22 at exposure -0.8) |
| Canvas 128 x 256 with the tile filling it edge to edge (camera ortho scale = tile length); the prototype rendered 300 x 560 with a margin | The class canvas of `docs/tiles.md`, shared with tile back skins |
| Cycles on the CPU, 64 samples at 4x the canvas, fixed seed 169, 8 fixed threads, no adaptive sampling, no denoiser, image stamps off; the prototype used a GPU, 96 samples and a denoiser | Deterministic output: the same pixels on every run |
| The look (materials, light) moved to `work/ivory_set.json` | A new face set changes that file only (`docs/tiles.md` 1.2) |

## Render (W-3) and export (W-5)

Run on 2026-11-15 by Claude Code with Blender 5.2.2 LTS, Python 3.12.10, NumPy 2.4.6, SciPy 1.15.3 and Pillow 12.1.1, from the repository root:

```bash
blender --background --factory-startup --python-exit-code 1 --python assets-src/tile/ivory/work/ivory_render.py -- assets-src/tile/ivory/work/ivory_set.json <renders>
python -I assets-src/tile/ivory/work/ivory_export.py <renders> assets-src/tile/ivory/export
```

It printed `exported 28 faces at 128x256`. Rendering took 14 min 17 s for the 28 faces on the CPU (about 31 s a face). A second render into an empty folder on 2026-11-15 gave the same pixels for all 28 faces (compared as decoded RGBA arrays), and exporting it gave faces byte-identical to `export/face/` (`diff -r`). No stamp is drawn into the image, as the render script turns stamps off; the raw renders still carry Cycles text chunks with the render time, so they differ as files, and the export writes none of them.

| Step of `docs/asset-workflow.md` 5.1 | Done | How, or what is missing |
|---|---|---|
| 1. Class canvas and pivot | Yes | 128 x 256, centre (`docs/tiles.md` section 1) |
| 2. Paint-over | Not needed for generated art; none was generated | The face is a render of our own model. An artist may still tune `work/ivory_set.json` |
| 3. Parts on layers | Not applicable | The face does not animate; shadow and sheen are their own sprites (`assets-src/effect/`) |
| 4. Nothing art-direction section 8 forbids | Checked by the agent | No text, glow or drop shadow; the shadow is a separate runtime sprite |
| 5. Silhouette and smallest-size tests | Yes | Below |

## Readability check (acceptance of #169)

```bash
python -I assets-src/tile/ivory/work/ivory_check.py assets-src/tile/ivory/export/face assets-src/cosmetic/table/starter/export/static.png assets-src/effect/tile_shadow/export/static.png docs/design-tokens.json <out> --width 32 28
```

The smallest display size is a tile short side of 41.7 canvas units, 32 x 64 px on the iPhone XR, the smallest phone of the device matrix (`docs/tiles.md` section 3 has the derivation). Output on 2026-11-15:

```text
32x64 px: pip diameter 5.6 px, gap between neighbouring pips 3.0 px, ivory luminance 225, darkest pip 11, faces with a wrong pip count 0 of 56
28x56 px: pip diameter 4.9 px, gap between neighbouring pips 2.6 px, ivory luminance 225, darkest pip 11, faces with a wrong pip count 0 of 56
```

56 = the 28 faces, each sampled with a box filter (as with mipmaps) and with plain bilinear sampling of the 128 px sprite (no mipmaps). Every half of every face showed exactly its number of separate pips at both sizes and both samplings. The same check at 24, 20 and 16 px also found no wrong count, so the pass at 32 px has margin. The check image of the 28 faces at 32 x 64 px on `table.starter` with the runtime shadow (`color.shadow`, `elevation.e1`) was looked at by the agent at real size and at 4x: every value reads by its pattern; the divider softens to a grey line at this size, the pins to a small warm dot. The check images are not committed; the command above makes them again.

On `table.starter`, the centre field (pixels 200 to 1720 by 200 to 880) has a relative luminance median of 0.034 (90th percentile 0.044); the ivory face 0.765; contrast of the face against the median field 9.7:1.

Not done: a look at the faces on a real iPhone XR or Galaxy A14 (#191).

## Files

The scripts and `work/ivory_set.json` are the source; the 4x renders are not kept, as the render script makes them again pixel for pixel. Each export face is 128 x 256 RGBA, straight alpha.

| File | sha256 |
|---|---|
| export/face/0_0.png | `a341418db3a63119dbcfbeb4501e6fee16c28be7f3863af8789e903806476f27` |
| export/face/0_1.png | `2ce5aed3ca46b4ac94e715b8082960849f95ece644dbd92ae00fa5613ba233cf` |
| export/face/0_2.png | `47f59d104a8724bec433a804a095ffc5a7660610e3e2d77970f69e98b5e2efd8` |
| export/face/0_3.png | `fea2c2eec8af9821f73e7869bb2dc947bfe887bf4ef3a375ad2c999369d594d8` |
| export/face/0_4.png | `e97e8227165f1a76046f14979853f34b53180e10565a725637215f72629257ec` |
| export/face/0_5.png | `fdb3e733ed816fd83a3977e997b4733a02b87cdb12a1dff2afd52dd7d921a1dd` |
| export/face/0_6.png | `6c71cf36f170a0193b23a85fbd96664689fffb8064469f2457fc10ba7c2fa73a` |
| export/face/1_1.png | `d53a91538b24b7c17f57c592476be2ac2fcaf17919859d892d1618ceb0ae1cae` |
| export/face/1_2.png | `d8af35ffd6d405f2a2e52d7f4b8c5e97dbce27d54e670dbafc7aef7d6c6f1ee4` |
| export/face/1_3.png | `acc7df79ff1d76f2c2eaf90adee8f9406fa3dc518b80397feb7d5baa675474ad` |
| export/face/1_4.png | `95a52d257766fb3132b4ea922b9976307cc8ac33096846acdc74d0974be5e2e2` |
| export/face/1_5.png | `93597e108c5197e71eba6315faefe7bc85898512e05144f6576b11734a26d03b` |
| export/face/1_6.png | `0570246cb6365e433e8efa02612e2fc267ea9e8ff48aa5f877c94d0afe9eddfa` |
| export/face/2_2.png | `7dc2ab6081f8343485aad58cb7a34b814d5441b4cb6aa6e3a9123775cdc01977` |
| export/face/2_3.png | `58e0a9cd1123474f2762507b8b5f5d99843894cc7fc68866b2614849e293943e` |
| export/face/2_4.png | `947325e3ecabe3563a0b6318157d11249ef27c13addbe312f13c52669fc01f45` |
| export/face/2_5.png | `de851347b69408c00fd646b8f580f98492684c2e22ec03970f97224866bb592b` |
| export/face/2_6.png | `166307daf60a17e64d740ef73000551840703e355244e7c1fd36fd0cfd9d5e6d` |
| export/face/3_3.png | `2a5a66389217335a09b1d26be5a20d5fc98a0ea848f48b54abe69f4dd0b65dcb` |
| export/face/3_4.png | `64868567e5d9d786078a4804243f6ab974afd70d7f44fe124b2c7384e27bd86b` |
| export/face/3_5.png | `2f5d8dae2e6e06b3b6570f26743c5b9df1356c62f9689491a5ddea7b6571b52e` |
| export/face/3_6.png | `5c2f17879d6cc38360969ec9bd03886a7310a9b84c1e69b22ae7a2cfdb93f53f` |
| export/face/4_4.png | `74387c00ac87a30f841a63a972dbdb7c13a3a33c6276bbf65ede1ab881fa7dff` |
| export/face/4_5.png | `08d29b6d7a725f96cd2d12dbf1b9c32d6ea861458b836ea490f75bbf45743d00` |
| export/face/4_6.png | `6519e75a55d9bcb4bc5eb5b20a9f6d05b43bf3c8dc6d79062e76e27d7caf3e00` |
| export/face/5_5.png | `659ca517e6569e9125c1939711a50ca51ae7731eec7a1a71145126685acbdc6b` |
| export/face/5_6.png | `9bd1a538f4e46136446cf75136b9bbc444e5761ee1be9e164f8945bd6be43609` |
| export/face/6_6.png | `56b7bdf6e487042bff658cd615394d60c0f0ee8faffe5a518efa94f3a91beef2` |

