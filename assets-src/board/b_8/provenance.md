# b_8

- Panel: B-8 Effects storyboard (`docs/art-direction.md` section 9)
- File: export/b_8.jpg, 1080 x 2328, on `color.background.base`
- Layout: two columns (left the low effect level, right the high level) of 504 x 360 cells, 24 px apart, 24 px margin. Rows, top to bottom: `tilePlace`; `scoreCascade` at step 1, step 3 and step 6; `domino`; `block`. Each cell is one frame near the peak of the effect, drawn at the 1080-unit short side.
- Ground of each cell: the centre 504 x 360 crop of `assets-src/cosmetic/table/crimson_carpet/export/static.png`, with a chain of three tiles (`assets-src/tile/ivory/export/face/4_6.png`, `6_6.png`, `1_6.png` at half scale) on the runtime tile shadow (`assets-src/effect/tile_shadow`, tint `color.shadow`, offset and alpha of `elevation.e1`).
- Effects: the sprites of `assets-src/effect/vfx/export/`, counts and tints read from `client/Assets/HypeDomino/UI/Effects/effects.json` (the definitions of `docs/vfx.md`), score step counts from `scoreCascade.steps` and `effects.budgets.scoreCascade` in `docs/design-tokens.json`, glow alpha and scale from `glow`. Additive sprites are added, the stamp is alpha blended. Winning team: `color.role.teamA.fill`.
- Board dim of `block`: `color.scrim` at `opacity.disabled` (0.4). The amount is a proposal (`docs/vfx.md` section 4), not decided.
- Not shown: the score popup numerals, the localised `domino` banner and the team meter, because a board panel carries no text and the banner art is not made yet. Particle positions are random (seed 176), not the runtime's motion; the panel shows look and density, not timing.
- Made by `work/b_8.py`, written and run by an AI agent (Claude Code) on 2026-11-16 with Python 3.12.10, NumPy 2.4.6 and Pillow 12.1.1, JPEG quality 82: `python -I assets-src/board/b_8/work/b_8.py . assets-src/board/b_8/export/b_8.jpg`. A second run wrote a byte-identical file. No image generator was used.
- Reference only; never shipped.
