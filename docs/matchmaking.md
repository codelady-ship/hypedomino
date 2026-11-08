# Matchmaking

Owner scope: #48 (queue), #52 (leaver rules), #118 (quality tuning). Rules referenced: R-53 to R-58 in `docs/rules.md`; the ranked side of section 9 of `docs/ranking.md`.

## 1. Queues

- Four queues: ranked 1v1, ranked 2v2, casual 1v1, casual 2v2. Entries of different queues, modes or regions never meet.
- An entry is one account with its mode, queue, region, MMR of that mode (`InitialMmr` until #112 writes real values) and the server time it joined. One account holds at most one entry at a time, and none while a match it plays in has not ended (R-58).
- A penalty queue delay (`docs/penalties.md` steps 2) sets the time the search starts; the entry keeps its join time for ordering and window growth.
- There are no parties. An entry names one account and nothing else; a client cannot name a partner, and in 2v2 the four players are split into teams at random (R-54). Playing with a friend is what custom rooms are for (R-55, #49).

## 2. Forming a match

- The pool is scanned on every tick, oldest entry first. For the oldest entry the server collects, in join order, the other entries whose MMR is within the narrower of the two players' windows. When the mode's number of players is reached the match forms; otherwise the entry waits and the scan continues with the next oldest.
- A player's window starts at `MmrWindowStart` and grows by `MmrWindowStep` every `MmrWindowStepSeconds` of waiting, up to `MmrWindowMax`.
- Ranked queues only (#118): two placed players also need rank indexes (`docs/ranking.md` section 2) at most `RankWindowDivisions` apart, so a Gold II player never meets a Diamond player however close their MMR. A player still in placement has no rank and is matched by MMR alone; 0 disables the rank check.
- Only a match formed from a ranked queue reaches the rating store (`Match.Ranked`); casual matches and custom rooms (R-55, R-56) change nothing in `docs/ranking.md`.
- Ranked queues only (#121): an entry carries the accounts it must not meet again (`rating.Store.Avoid`: met `RepeatPairMax` times within `RepeatPairWindowHours`, `docs/ranking.md` section 11); the scan skips a pair when either side lists the other.
- Casual queues only: an entry that has waited `CasualBotFillSeconds` forms a match with the players found so far and server bots in the remaining seats. Ranked queues never seat a bot (section 9 of `docs/ranking.md`); a value of 0 disables bot fill.
- Seats: the players are shuffled and seated in seat order (1v1: X, Y; 2v2: N, E, S, W, so N and S are one team and E and W the other). Bots take the seats left.
- A formed match is never cancelled because a player did not load; it starts after the loading phase without them (R-57).

## 3. Config

| Field | Default | Meaning |
|---|---|---|
| `MmrWindowStart` | 100 | MMR distance accepted at join |
| `MmrWindowStep` | 50 | Growth of the window per step |
| `MmrWindowStepSeconds` | 10 | Length of a step |
| `MmrWindowMax` | 400 | Largest window |
| `RankWindowDivisions` | 8 | Largest rank index distance between two placed players in a ranked queue; 0 disables |
| `CasualBotFillSeconds` | 30 | Wait before a casual entry is completed with bots; 0 disables |

The engine is `server/internal/matchmaking`: `Form` is a pure function over the entries of one tick (table tests in `matchmaking_test.go`), `Pool` is the in-process queue around it. The cluster wiring is section 4 (#478). A ranked entry's MMR, rank and placement state come from `rating.Store.Queue` (#118), `InitialMmr` and unplaced for an account without a rating in the mode; the test in `server/internal/rating/queue_test.go` runs a formed ranked match through `rating.Store.Apply` and checks that a casual match is never applied.

## 4. Queue protocol and placement

Owner decision 2026-11-08 (#478): one pool per region, run by a leader node.

- Any game server pod accepts the queue intents of its connected players. `JoinQueueIntent` names the mode and the queue (`QUEUE_KIND_RANKED` or `QUEUE_KIND_CASUAL`); `LeaveQueueIntent` leaves. The server answers with `QueueState`.
- `QueueState.status`: `SEARCHING` with the join time and the time the search starts (later than the join time during a penalty queue delay); `LEFT` after a leave or a disconnect; `MATCH_FOUND` when a match formed, followed by the match itself (R-57 loading phase); `REFUSED` with a reason and, for a ban, the time it ends.
- Refusals: `IN_MATCH` while a match of the player has not ended (R-58); `MATCHMAKING_BAN` during penalty step 3 (`docs/penalties.md`); `RANKED_CLOSED` while the ranked kill switch is on (`docs/admin.md`); `UNAVAILABLE` while the pod drains or no leader is reachable. A second join while queued replaces the entry.
- One pod at a time holds the leader lock in Redis and runs the region's only `Pool` and its `Tick`. A lost lock ends leadership; the next leader starts with an empty pool and the pods re-send the entries they hold, so no player has to queue again.
- The leader picks the pod that opens the room of a formed match. Players whose sessions live on other pods join it through the cross-node relay (#51).
