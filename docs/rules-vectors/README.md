# Rule test vectors

Hand-computed input/output cases for the rules in [`docs/rules.md`](../rules.md). The Go rules engine (`server/internal/rules`) and any other implementation of the rules must pass every vector. A vector that disagrees with `rules.md` is a bug in the vector; fix it in the same PR that changes the rule.

## Files

| File | Covers |
|---|---|
| `schema.json` | JSON Schema (draft 2020-12) for every vector file |
| `deal.json` | Tile set and deal shape (R-1 to R-3) |
| `seats.json` | Teams and clockwise turn order (R-6 to R-8) |
| `opening.json` | First-hand opener: 2-3, the 1v1 double order, redeal (R-9 to R-11, R-14) |
| `opening-score.json` | The opening placement score (R-29, R-30) |
| `board-points.json` | Open end sums, doubles on ends, legal attachment (R-15, R-16, R-25 to R-28) |
| `spinner.json` | Spinner creation and arms (R-17 to R-19, R-27) |
| `multiple-doubles.json` | Several doubles in one turn (R-24, R-31) |
| `draw.json` | Place, pass and draw decisions (R-20 to R-23) |
| `hand-status.json` | Domino and block detection (R-32, R-36) |
| `rounding.json` | Rounding hand points up to a multiple of 5 (R-33, R-37, R-38) |
| `domino.json` | Hand end by domino (R-33 to R-35, R-39, R-12) |
| `block.json` | Hand end by block, equal totals and carry-over (R-37 to R-40, R-13) |
| `rule-300.json` | The 300 rule on hand points and board points (R-41, R-42) |
| `match-end.json` | Match end checks (R-43 to R-48) |
| `timer.json` | Turn timer, AFK and leavers (R-49 to R-52) |
| `redeal.json` | Redeal when one player is dealt six or more doubles, in every hand (R-59) |

## Vector format

A vector file is `{ "$schema": "./schema.json", "vectors": [ ... ] }`. Every vector has:

| Field | Meaning |
|---|---|
| `id` | Stable, unique across all files, kebab-case. Never reused or renamed |
| `kind` | Which engine function the vector drives (table below) |
| `rules` | The `R-n` ids the vector checks |
| `description` | One line a human can verify against `rules.md` |
| `mode` | `1v1` or `2v2` |
| `given` | Input state |
| `expect` | Exact output. Implementations compare it field by field |

Conventions:

- Seats: 2v2 uses `N`, `E`, `S`, `W` in clockwise order; team `A` is N and S, team `B` is E and W. 1v1 uses `X` (team `A`) and `Y` (team `B`), each a team of one (R-8).
- Tiles are `"a-b"` strings. Outside a board they are written low-high (`"2-5"`, never `"5-2"`).
- A board is `{ "line", "spinner", "top", "bottom" }`. `line` lists tiles from the left end to the right end, each oriented so its first half faces left. `spinner` is the spinner tile (it sits inside `line`) or `null`. `top` and `bottom` list arm tiles from the spinner outward, each oriented so its first half faces the spinner; both are empty while there is no spinner.
- A placement `end` is `left`, `right`, `top` or `bottom`. `top` and `bottom` exist only once there is a spinner.
- `openEnds` maps each open end to what it counts: the exposed half, or both halves when the outermost tile is a double (R-26). Empty arms are left out (R-27).
- States are minimal: they hold only what the function reads. Tile counts need not add up to 28 except in `deal`, `first-opener` and `doubles-redeal` vectors. In 1v1 the boneyard is listed in draw order, first tile first.

## Kinds

| Kind | Input (`given`) | Output (`expect`) |
|---|---|---|
| `deal` | `hands`, `boneyard` | `valid`: the deal is a full double-six set with 7 tiles per player and the right boneyard size |
| `seat` | `seat` | `team`, `next` (the seat that plays after it) |
| `first-opener` | full first-hand deal: `hands`, `boneyard` | `opener`, `tile`, `next`, or `redeal: true` with the others `null` |
| `opening-score` | `firstHand`, opening `tile`, `teamScoreAtHandStart` of the opener's team | `score` of the opening placement |
| `place` | `board`, `tile`, `end`, optional `teamScoreAtHandStart` | `legal: false`, or `legal: true` with the new `board`, `openEnds`, `total` and `score` |
| `turn` | `board`, `placements` tried in order within one turn | per placement `placed` (with `total` and `score` when placed) and the `turnScore`. The list stops at the first placement that is refused |
| `draw` | `board`, the player's `hand`, `boneyard` | `drawn` tiles and `outcome`: `place` (the player must place; after drawing, the last drawn tile) or `pass` |
| `hand-status` | `board`, all `hands`, `boneyard` | `status`: `domino`, `block` or `continue` |
| `round` | `pips` | `points` rounded up to a multiple of 5 |
| `hand-end` | `end` (`domino` or `block`), `lastPlacedBy`, remaining `hands`, `boneyard`, `scoresAtHandStart`, `scores` before hand points, incoming `carryOver` | `pips` per team, `winner` (`null` on an equal block), `awarded` (hand points plus any carry-over taken), outgoing `carryOver`, final `scores`, `nextOpeningTeam` |
| `match-end` | `end` (`domino`, `block`, or `none` for a check in the middle of a hand), `scores` after the hand | `outcome`: `won` with `winner`, `next-hand`, or `continue` |
| `turn-timer` | `turnSeconds` from admin config, `actedAtSeconds` (`null` if the player never acted) | `movedBy`: `player` or `bot` |
| `doubles-redeal` | full deal of any hand: `hands`, `boneyard` | `redeal`: some single player holds `RedealDoublesThreshold` or more doubles |
| `afk` | `afkTimeouts` from admin config, `events` per seat (`timeout`, `move`, `reconnect`, `leave`) | `controllers`: who plays each seat afterwards |

## Use by the engine

The engine's table-driven test in `server/internal/rules` reads every `*.json` file in this directory except `schema.json`, dispatches each vector on `kind` to the matching pure function, and compares the result with `expect`. An unknown `kind` fails the test. Vector ids appear in test names so a failure points at one vector.

Numbers that `rules.md` makes admin-tunable (the turn time and the AFK limit) are inputs of the vector. The other vectors use the default `rules.Config` values from `rules.md` section 13.

## Validation

`npm run test:tools` runs `tools/rules-vectors/rules-vectors.test.mjs`, which checks:

- every file matches `schema.json`;
- ids are unique and every `R-n` exists in `docs/rules.md`;
- every rule that is not excluded below has at least 3 vectors;
- states are consistent: tiles appear once, board chains match, the spinner is an inner double, seats belong to the mode, only the domino player's hand is empty;
- every `expect` agrees with an independent re-derivation of the arithmetic in `rules.md` (sums, rounding, open ends, scoring, carry-over, match end). This is a cross-check of the hand-computed values, not the engine.

## Adding a vector

1. Pick the topic file and the `kind`. Compute `expect` by hand from `rules.md`, then check it a second time.
2. Give it a new `id` and list every rule it checks in `rules`.
3. Run `npm run test:tools`. If the validator disagrees, recheck the hand computation against `rules.md` before touching anything else.
4. If a rule is ambiguous for the case, do not add the vector. Raise the gap on the rules issue instead.

## Excluded rules

These rules are not checkable by a deterministic rules-engine function. The validator reads this list and does not require vectors for them.

- **R-4** Shuffling uses a secure random source and tile visibility is a server transport property; neither is a pure function result.
- **R-5** `mode` is a closed enum (`1v1`, `2v2`) in the schema and the engine API, so a third mode cannot be expressed as input.
- **R-53** Ratings, ranks and leaderboards per mode belong to the ranking spec (#8, #52), outside `server/internal/rules`.
- **R-54** Partner assignment in ranked 2v2 is random matchmaking, outside `server/internal/rules`.
- **R-55** Custom rooms and team splitting are room management, outside `server/internal/rules`.
- **R-56** Casual queues and bot fill are matchmaking, outside `server/internal/rules`.
- **R-57** The loading phase is room start-up, outside `server/internal/rules`; the room tests cover it.
- **R-58** Rejoining and the queue block for a player with an unfinished match are session and matchmaking, outside `server/internal/rules`.
