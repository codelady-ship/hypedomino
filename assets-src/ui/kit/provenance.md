# ui.kit

Buttons, panels, tabs, the 9-slice elevation shadows, the soft name and hint backdrop, a white pill, the gold divider and the icon shadow of #168 (`docs/ui-motion.md` sections 4 and 5, `docs/art-direction.md` sections 2 to 4). Sprite list, borders and use are in `docs/ui-kit.md`. Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 while the work was done.

- artSource: handmade (no image generator was used; every sprite is computed by `work/kit.py`)
- Authors: script written and run by an AI agent (Claude Code) on 2026-11-16. No person has reviewed the look yet
- Generator: none
- Terms: not applicable, no generator
- Canonical concept: none
- Inputs uploaded to the generator: none
- Similarity check: not applicable (rounded rectangles and pills)
- Human work: none. The script is in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done
- Cultural review: not done

## Export (W-5)

Run on 2026-11-16 with Python 3.12.10, NumPy 2.4.6 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/ui/kit/work/kit.py docs/design-tokens.json assets-src/ui/kit/export
```

| Property | Value |
|---|---|
| Files | 14 sprites `export/<name>.png` and `export/kit.json` (size, 9-slice border in Unity order left, bottom, right, top, pixels per unit, use) |
| Colours | Read from `docs/design-tokens.json` at run time: `color.gold.*`, `color.role.primary.fill`, `color.role.secondary.fill`, `color.surface.*`, `color.background.raised`. Mixes of two tokens give the painted light of the lacquer (top lighter, bottom darker); no colour value is written in the script |
| Sizes | Read from the tokens: `radius.md`, `radius.lg`, `size.touchTargetMin`. Drawn at 2 px per UI unit (pixels per unit 200 against the canvas reference of 100), so corners stay sharp up to a 2160-pixel short side |
| White sprites | `button_tint`, the three shadows, `backdrop_soft`, `pill_fill`, `icon_shadow` are white or greyscale and are tinted through vertex colour (art-direction section 2): a role fill, `color.shadow` at the elevation alpha, `color.scrim` at `opacity.scrim` |
| Not baked | No glow, no drop shadow inside a button or panel sprite, no text (art-direction section 8). The shadow is its own sprite under its owner |
| Anti-aliasing | 4 x 4 samples per pixel, box filtered |

A second run into an empty folder wrote byte-identical output, identical to the committed files. Not done: the look review on the reference board (B-9), press, disabled and glow states in a screen (they are runtime tint and the shared glow sprite, `docs/ui-motion.md` section 5), a measured atlas in a build.
