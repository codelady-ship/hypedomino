# HypeDomino game rules

Single source of truth for the game rules. The rules engine (`server/internal/rules`) and its tests implement this document exactly. Change this document first, then the code.

Every rule has a stable id (`R-1`, `R-2`, ...). Test vectors (`docs/rules-vectors`, #4) reference these ids. Ids are never reused or renumbered; a removed rule keeps its id marked as removed.

Owner decisions behind this version: #3 and #4 (2026-10-25); the redeal on six doubles, R-59 (2026-11-14); match start, loading phase and rejoining, R-53, R-57 and R-58, and the friend lobby flow of R-55 (2026-10-28, #85, #154).

## Terms

| Term | Meaning |
|---|---|
| Tile | One domino piece with two halves, each showing 0 to 6 pips |
| Double | A tile with equal halves (0-0, 1-1, ... 6-6) |
| Hand | One deal, played until a domino or a block |
| Match | Hands played until a team wins at the target score |
| Open end | A chain end where a tile may be attached |
| Spinner | The one double per hand that can be extended in four directions |
| Boneyard | Undealt tiles (1v1 only) |
| Domino | A player places their last tile and ends the hand |
| Block | Nobody can place a tile and the hand cannot continue |
| Hand points | Points taken from the pips left in the opponents' hands at the end of a hand |
| Board points | Points scored by placing tiles during play |

## 1. Tiles and deal

- **R-1** The set is a double-six set: 28 tiles, 0-0 through 6-6, each exactly once.
- **R-2** Every player is dealt 7 tiles at the start of each hand.
- **R-3** In 2v2 all 28 tiles are dealt and there is no boneyard. In 1v1 the remaining 14 tiles form the boneyard.
- **R-4** Shuffling happens on the server using a cryptographically secure random source. A client never receives any tile that its player does not hold.
- **R-59** In every hand of every mode, when any one player is dealt `RedealDoublesThreshold` (default 6) or more doubles, all tiles are reshuffled and dealt again before the hand opens. Doubles are counted per player, never summed across partners. The redeal changes no score and no opening team; it repeats until no player holds that many doubles, and in the first hand R-9 to R-11 then apply to the final deal.

## 2. Modes, seats and turn order

- **R-5** Modes: 1v1 and 2v2. There are no other modes.
- **R-6** In 2v2 partners sit opposite each other, so seats alternate between teams around the table.
- **R-7** Turns pass clockwise. In 2v2 every turn therefore passes to the other team.
- **R-8** In 1v1 each player is a team of one. Every rule that says "team" applies to that single player.

## 3. Opening a hand

- **R-9** First hand of a match: the player holding 2-3 opens, and the opening tile must be 2-3.
- **R-10** First hand in 1v1 when 2-3 is in the boneyard: the player holding the lowest double in the order 1-1, 2-2, 3-3, 4-4, 5-5, 6-6, 0-0 opens with that double.
- **R-11** First hand in 1v1 when neither 2-3 nor any double was dealt to a player: all tiles are reshuffled and dealt again.
- **R-12** Later hands after a domino: the team that dominoed opens. The two partners choose between themselves which of them opens, and the opener may open with any tile.
- **R-13** Later hands after a block: the team of the player who placed the last tile before the block opens. The partners choose the opener, and the opener may open with any tile.
- **R-14** After the opening tile, play continues clockwise from the opener.

## 4. Board and spinner

- **R-15** Tiles are attached end to end. The touching halves must show the same number.
- **R-16** The chain starts with two open ends: the two halves of the opening tile.
- **R-17** The first double in a hand to have tiles attached on both of its sides becomes the spinner, whichever double was placed first. There is exactly one spinner per hand at most.
- **R-18** Once a double becomes the spinner, its top and bottom arms open and tiles may be attached there too, giving up to four open ends.
- **R-19** Any other double only continues the line it sits in. It never gets arms.

## 5. Legal moves, drawing and passing

- **R-20** On their turn a player must place a tile if they have a legal placement.
- **R-21** In 2v2, a player with no legal placement passes.
- **R-22** In 1v1, a player draws from the boneyard only when they have no legal placement. They draw one tile at a time until they draw a tile they can place, and then place that tile.
- **R-23** In 1v1, the last tile in the boneyard can never be drawn. A player with no legal placement when only that tile remains passes.
- **R-24** A turn normally has one placement. Several placements in one turn are allowed only as described in R-31.

## 6. Board points

- **R-25** After every placement, add up the values of all open ends. If the total is a positive multiple of 5, the placing player's team scores that total.
- **R-26** A double on an open end counts both halves (4-4 on an end counts 8).
- **R-27** The spinner's top and bottom arms count as open ends only after a tile has been placed on them. Before that they add nothing.
- **R-28** If the total is not a multiple of 5, or is 0, nothing is scored.
- **R-29** The opening placement is the only exception to R-25. It scores only when the opening tile is 5-5, which scores 10. Every other opening tile scores nothing, even when its pips add up to a multiple of 5 (for example 2-3 or 6-4).
- **R-30** The 5-5 opening scores nothing when the opener's team had 300 or more points at the start of the hand (R-41), and always scores nothing in the first hand of a match.
- **R-31** Multiple doubles in one turn: when a player places a double and that placement scores, the player may place another double from their hand in the same turn, but only if that placement also scores. This can repeat. Each placement is scored on its own under R-25. A double that would not score cannot be placed as a continuation.

## 7. End of hand: domino

- **R-32** A hand ends in a domino when a player places their last tile.
- **R-33** The team that dominoed scores the total pips left in the opposing team's hands, rounded up to the next multiple of 5 (22 → 25, 25 → 25, 26 → 30). The pips in the partner's hand are not counted.
- **R-34** In 1v1 the boneyard tiles, including the last undrawable one, are not counted.
- **R-35** The team that dominoed is the winner of the hand.

## 8. End of hand: block

- **R-36** A hand ends in a block when no player can place a tile: in 2v2 all four players pass in a row; in 1v1 only the last, undrawable boneyard tile remains and both players pass in a row.
- **R-37** Each team adds up the pips left in its own hands. The team with the lower total scores the other team's total, rounded up to the next multiple of 5. The team with the higher total scores nothing from hand points. The team with the lower total is the winner of the hand.
- **R-38** If both totals are equal, nobody scores and the hand has no winner. Both totals are added together, rounded up to the next multiple of 5, and kept as a carry-over.
- **R-39** The carry-over is added to the score of the next hand's winner, together with that winner's own hand points. If the next hand also ends in an equal block, its rounded sum is added to the carry-over, and the total moves to the next winner. If the next winner cannot take hand points (R-41), the carry-over is discarded.
- **R-40** R-13 decides who opens the next hand after any block. This does not depend on which team won the block.

## 9. The 300 rule

- **R-41** A team that has 300 or more points at the start of a hand cannot score hand points in that hand: no points from a domino (R-33), no points from a block (R-37) and no carry-over (R-39).
- **R-42** The 300 rule never blocks board points (R-25), except for the 5-5 opening (R-30).

## 10. End of match

- **R-43** The target score is 365.
- **R-44** The match is checked only at the end of a hand, never in the middle of one.
- **R-45** A hand that ends in a block never ends the match, even when a team has reached 365. The next hand is played. If that hand also ends in a block, play continues again.
- **R-46** At the end of a hand that ends in a domino, if at least one team has 365 or more, the team with more points wins the match. This applies even when that team did not win the last hand.
- **R-47** If both teams have 365 or more with equal points, another hand is played.
- **R-48** If no team has reached 365, a new hand is dealt.

## 11. Timer, AFK and leavers

- **R-49** Every turn has a time limit from admin config (default 20 seconds).
- **R-50** When the time runs out, the server bot makes a legal move for that player. An action that arrives at or after the limit is late.
- **R-51** After a number of consecutive timeouts set in admin config, the player is marked AFK and the bot plays the rest of the match for them. The player can take back control by reconnecting; the consecutive-timeout count then starts again from zero. Penalties for going AFK are defined in `docs/penalties.md`.
- **R-52** A player who leaves the match is replaced by the bot. If they come back before the match ends, they take back control from the bot; the consecutive-timeout count then starts again from zero. Their partner keeps playing either way. A leaver who stays away too long is penalised as defined in `docs/penalties.md`.

The turn timer shows R-49 to the players and changes nothing about it: the match view carries the turn length (`TurnSeconds`) next to the turn deadline, and the timer ring turns to the warning colour once `TurnWarningSeconds` (admin config, default 5 seconds) or less remain, by server time (owner decision 2026-11-14, `docs/ui-motion.md` Turn timer).

## 12. Ranked, casual and custom play; match start

- **R-53** 1v1 and 2v2 are both ranked. Each mode has its own rating, rank and leaderboard. A ranked match opens directly once matchmaking has formed it (`docs/ranking.md`): there is no accept step, room, seat choice or team split, only the loading phase of R-57.
- **R-54** In ranked 2v2 the partner is assigned at random. A player cannot choose a partner or queue for ranked play together with a chosen partner.
- **R-55** Custom rooms are unranked and meant for friends. A custom room starts as a friend lobby, the creator then splits the teams in team setup and starts the match, and after the match the players return to the lobby (`docs/friend-lobby.md`). The creator of the room can remove a player from it and add a player to a free seat, before or during a match. During a match the seat of a removed player is played by the bot until the creator adds a player to it; a removed player comes back only if the creator adds them again. Nobody is penalised in a custom room for timing out, leaving or being removed.
- **R-56** Casual play is an unranked matchmaking queue for 1v1 and 2v2 under the same game rules. It changes no rating, rank or leaderboard. Only casual queues may fill an empty seat with a bot after a long wait; ranked queues never do.
- **R-57** Every match, ranked, casual or custom, opens with a loading phase. Each player's client loads the match and reports when it is ready. The first hand is dealt as soon as every player is ready, or when `MatchLoadingSeconds` (admin config, default 10 seconds) have passed, whichever comes first. No turn timer runs during the loading phase. A player who is not ready when the phase ends starts the match as disconnected: the bot plays for them (R-52), and their reconnect grace starts at that moment. A match is never cancelled or delayed further because a player did not load.
- **R-58** A player whose match has not ended can rejoin it at any time until it ends, and takes back control from the bot when they do (R-52). Until the match ends they cannot join a matchmaking queue or a custom room or start another match; opening the game takes them back into the running match.

## 13. Engine configuration

Rule values live in `rules.Config` so they can change without code changes. The defaults are the values in this document.

| Field | Default | Rules |
|---|---|---|
| `HandSize` | 7 | R-2 |
| `RedealDoublesThreshold` | 6 | R-59 |
| `TargetScore` | 365 | R-43 |
| `HandPointsLockThreshold` | 300 | R-30, R-41 |
| `ScoringMultiple` | 5 | R-25, R-33, R-37, R-38 |
| `FirstHandOpeningTile` | 2-3 | R-9 |
| `ScoringOpeningTile` | 5-5 | R-29 |
| `ScoringOpeningPoints` | 10 | R-29 |
| `TurnSeconds` | admin config, 20 | R-49 |
| `TurnWarningSeconds` | admin config, 5 | R-49 (display only) |
| `AfkConsecutiveTimeouts` | admin config, 3 | R-51 |
| `MatchLoadingSeconds` | admin config, 10 | R-57 |

The admin config fields form the `rules` domain (`docs/admin.md` 3.1, 5.14). The admin edits them within these bounds:

| Key | Type | Unit | Default | Min | Max |
|---|---|---|---|---|---|
| `TurnSeconds` | integer | seconds | 20 | 10 | 60 |
| `TurnWarningSeconds` | integer | seconds | 5 | 1 | 10 |
| `AfkConsecutiveTimeouts` | integer | timeouts | 3 | 1 | 10 |
| `MatchLoadingSeconds` | integer | seconds | 10 | 5 | 60 |

- **Refused snapshots.** A snapshot is refused if it has an unknown key, a value that is not an integer, or a value outside its bounds. The server then keeps its current values.
- **Missing keys.** A key that the snapshot leaves out keeps its default.
- **Warning bound.** `TurnWarningSeconds` is at most 10, the lowest `TurnSeconds`, so the warning never starts before the turn does.
- **When a new value applies.** Only to matches that open after the server loads it. Matches in progress keep the values they started with.

This is implemented in `server/internal/rulesconfig` (#145). The admin API edits, publishes and rolls back the domain (`docs/admin.md` 9.1), and `config-publish -domain rules` checks a snapshot the same way before publishing it.

## 14. Worked examples

All 2v2 examples use seats North, East, South and West in clockwise order. Team A is North and South, Team B is East and West.

### Example 1: opening, board points and the spinner (R-9, R-17, R-18, R-25 to R-29)

First hand. East holds 2-3.

| # | Player | Tile | Placed on | Open ends | Total | Score |
|---|---|---|---|---|---|---|
| 1 | East | 2-3 | opening | 2, 3 | 5 | none, only 5-5 scores as an opening (R-29) |
| 2 | South | 3-3 | the 3 | 2, 3-3 | 2 + 6 = 8 | none |
| 3 | West | 2-4 | the 2 | 4, 3-3 | 4 + 6 = 10 | B +10 |
| 4 | North | 3-1 | the other side of 3-3 | 4, 1 | 5 | A +5. 3-3 now has tiles on both sides and becomes the spinner. Its arms are empty, so they add nothing (R-27) |
| 5 | East | 3-5 | the spinner's top arm | 4, 1, 5 | 10 | B +10 |

### Example 2: opening a later hand (R-12, R-29, R-30)

Team B dominoed the previous hand. B has 140 points, A has 95. B decides that West opens.

- West opens with 5-5: B +10, now 150.
- If B had 300 points at the start of the hand, the 5-5 opening would score nothing.
- If West opened with 6-4 instead, nothing is scored, even though 6 + 4 = 10.

### Example 3: multiple doubles in one turn (R-31)

The open ends are 0 and 5. South holds 0-0, 5-5 and 1-1.

| # | Tile | Placed on | Open ends | Total | Score |
|---|---|---|---|---|---|
| 1 | 0-0 | the 0 | 0-0, 5 | 0 + 5 = 5 | A +5, so South may continue with a double |
| 2 | 5-5 | the 5 | 0-0, 5-5 | 0 + 10 = 10 | A +10, so South may continue with a double |
| 3 | 1-1 | no 1 end | | | cannot be placed, the turn ends |

A scores 15 in one turn. A counter-example: with open ends 2 and 6, a player places 2-2 (4 + 6 = 10, +10). 6-6 cannot follow, because it would make 4 + 12 = 16, which does not score.

### Example 4: domino and rounding (R-33, R-35)

At the start of the hand A has 180 and B has 220. North places their last tile.

- East holds 6-4 and 1-0, which is 11 pips. West holds 5-2, which is 7 pips.
- B's total is 18, rounded up to 20. A scores 20 and now has 200.
- South's remaining tiles are not counted.

### Example 5: the 300 rule on a domino (R-41, R-42)

A starts the hand with 310. During the hand A scores 15 board points and reaches 325. North then dominoes, and B's hands hold 23 pips.

- 23 would round up to 25, but A had 300 or more at the start of the hand, so A gets no hand points.
- A ends the hand with 325. The board points still count.

### Example 6: block with unequal totals (R-37, R-40)

The hand is blocked. West placed the last tile.

- Team A: North holds 6-5 (11) and South holds 4-3 (7), so A's total is 18.
- Team B: East holds 6-6 (12) and West holds 5-4 (9), so B's total is 21.
- A has the lower total and scores B's 21, rounded up to 25 (A started the hand under 300). A is the winner of the hand.
- Team B opens the next hand, because West placed the last tile. B chooses the opener and the tile.

### Example 7: block with equal totals and carry-over (R-38, R-39)

Hand 5 is blocked, with A holding 17 pips and B holding 17. North placed the last tile.

- Nobody scores. 17 + 17 = 34 rounds up to 35, which is kept as the carry-over.
- A opens hand 6 (R-13).
- In hand 6, B dominoes while A holds 12 pips, which rounds up to 15. B started hand 6 with 150, so B scores 15 + 35 = 50.
- If hand 6 had instead been blocked with 8 and 8, then 16 rounds up to 20, and the carry-over would grow to 35 + 20 = 55 for the winner of hand 7.
- If B had started hand 6 with 300 or more, B would score neither the 15 nor the carry-over, and the 35 would be discarded.

### Example 8: a block that crosses 365 (R-41, R-45, R-46)

B starts the hand with 290. The hand is blocked, B has the lower total, and A holds 78 pips.

- 78 rounds up to 80, so B scores 80 and has 370. The match does not end, because the hand was a block.
- The next hand is played. B started it with 370, so B cannot take hand points, but can still score board points.
- That hand ends with a domino by A, and A finishes with 240. B has 365 or more and more points than A, so B wins the match.
- If that hand had also been blocked, another hand would be played.

### Example 9: both teams cross 365 (R-46, R-47)

- A domino ends a hand with A at 370 and B at 375. B wins, even though A dominoed.
- A domino ends a hand with both teams at 370. Another hand is played.

### Example 10: 1v1 first hand without 2-3 (R-10, R-11, R-29, R-30)

2-3 is in the boneyard.

- X holds 4-4, 6-6 and 0-0. Y holds 1-1. Y opens with 1-1, the lowest double in the order. The opening scores nothing.
- X holds 0-0, Y holds 5-5, and neither holds another double. 0-0 is last in the order, so Y opens with 5-5. It scores nothing, because it is the first hand (R-30).
- Neither player holds 2-3 or any double: all tiles are reshuffled and dealt again.

### Example 11: other doubles never become spinners (R-17, R-19, R-26)

The spinner already exists. A player places 1-1 on an open 1 end.

- 1-1 counts 2 while it sits on an open end.
- It never gets arms. The next tile on it continues the same line.

### Example 12: drawing in 1v1 (R-22, R-23, R-36)

The open ends are 6 and 2. X holds no 6 or 2, and the boneyard has 4 tiles.

- X draws 1-4: it cannot be placed, so X draws again.
- X draws 5-0: it cannot be placed, so X draws again.
- X draws 6-1: it can be placed, so X places it on the 6. The open ends are 1 and 2, total 3, so nothing is scored.
- One tile is left in the boneyard and nobody can draw it. If both players now have no legal placement, the hand is blocked.

### Example 13: timer and AFK (R-49 to R-52)

The turn time is 20 seconds.

- South does not act in time, so the bot places a legal tile for South.
- South keeps timing out until the admin-configured limit is reached. South is marked AFK, and the bot plays South's remaining turns. North keeps playing.
- If South leaves the match, the bot plays South's turns. If South comes back before the match ends, South takes back control. If South stayed away longer than the admin-configured grace time, or never comes back, South is penalised for leaving (`docs/penalties.md`).

### Example 14: redeal on six doubles (R-59)

- 2v2, a later hand. North is dealt 0-0, 1-1, 2-2, 3-3, 5-5, 6-6 and 2-4: six doubles, so all 28 tiles are reshuffled and dealt again. The team that was to open still opens, and the scores do not change.
- 2v2. North holds four doubles and South, North's partner, holds three. No single player holds six, so the hand is played.
- 1v1, first hand. X is dealt all seven doubles: the tiles are dealt again, and R-9 to R-11 decide the opener from the new deal.
