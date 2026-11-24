# ui.rarity

The status diamond of `docs/rarity.md` section 2.1 (#168): one white diamond sprite tinted with the tier's `fill` and one white pip overlay per pip count tinted with the tier's `on`, both through vertex colour, as `RarityDiamond` draws them (`client/Assets/HypeDomino/UI/Rarity/RarityDiamond.cs`: an `Image` for the diamond, an `Image` for the pips, `Sprite[] pipOverlays` indexed by pip count). Every line below is read from a file or a record, or marked as not done. Recorded on 2026-11-16 while the work was done.

- artSource: handmade (no image generator was used; the sprites are computed by `work/rarity.py`)
- Authors: script written and run by an AI agent (Claude Code) on 2026-11-16. No person has reviewed the look yet
- Generator: none
- Terms: not applicable, no generator
- Canonical concept: none
- Inputs uploaded to the generator: none
- Similarity check: not applicable (a square on its corner and dots)
- Human work: none. The script is in `work/`
- Self-check (docs/localization.md 6.1 step 1): not done
- Cultural review: not done

## Export (W-5)

Run on 2026-11-16 with Python 3.12.10, NumPy 2.4.6 and Pillow 12.1.1, from the repository root:

```bash
python -I assets-src/ui/rarity/work/rarity.py docs/design-tokens.json assets-src/ui/rarity/export
```

| Property | Value |
|---|---|
| Files | `export/diamond.png`, `export/pips_1.png` to `export/pips_6.png`, all 192 x 192 RGBA on one canvas, pivot at the centre, so the overlay lies on the diamond without offsets |
| Diamond | A square on its corner, half diagonal 92 px. Greyscale facets lit from above (upper left 1.0, upper right 0.9, lower left 0.84, lower right 0.74) around a lighter table (0.97) at 55% of the size; white enough that the tint gives the tier `fill` |
| Pips | White dots, radius 14% of the half diagonal, on the layout of a domino half turned 45 degrees with the square (`docs/rarity.md` 2.1): 1 centre; 2 top and bottom corner; 3 vertical line; 4 the four corners; 5 four corners and centre; 6 two rows of three, the domino half's left and right columns turned with the square (reserved for a sixth tier) |
| Counts | The pip counts of `rarity.tiers` in `docs/design-tokens.json` (1 to 5) plus 6, the reserved sixth tier of `docs/rarity.md` 2.1, so `RarityDiamond.pipOverlays` covers every pip count the admin table allows (`pips` 1 to 6, rarity section 7) |
| Size | Drawn at 192 px for display sizes from `rarity.diamondMinSize` (72 units) up to about 96 units at 2 px per unit; the client hides the diamond under 72 units |
| Not baked | No rarity colour, glow or halo (art-direction section 8). The halo and flow halo sprites of `RarityHalo` belong to the shared effects atlas (#176) and are not here |

A second run into an empty folder wrote byte-identical output, identical to the committed files. Not done: the review on the reference board (the five item cards of `docs/rarity.md` section 8), the halo sprites (#176), a prefab that sets these sprites on `RarityDiamond`.
