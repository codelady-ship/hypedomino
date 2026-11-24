# b_4

- Panel: B-4 Tile back skins
- File: export/b_4.jpg
- Prototype files: `out/back_storm.png`, `back_phoenix`, `back_frost`, `back_sema`: Blender Cycles renders of the same tile with the skin images `gen/skin-*.png` on the back (`render_tiles.py`, mode `back`).
- Steps: Laid out in one row on `color.background.base`, 60 px apart, 60 px margin.
- Board panel of `docs/art-direction.md` section 9. Reference only; never shipped.
- Source: the owner-approved prototype `table-v2` (owner decision 2026-11-12), kept by the art team outside this repository.
- Generator, account and prompts of the prototype's painted source art (portraits, frames, skins, carpet): not recorded in the prototype. #167 records them before any of this art becomes a shipped item (`docs/asset-workflow.md` section 3).
- Processing for this panel: Python 3.12 with Pillow 12.1, JPEG quality 82, on 2026-11-12.
