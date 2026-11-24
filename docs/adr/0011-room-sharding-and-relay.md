# 0011. Room sharding: one process per node, routing in Redis, relay for resumes

Status: Accepted
Date: 2026-10-28
Issue: #51

## Context
A room lives in the memory of exactly one node (ADR 0003) and players reach nodes through one public load balancer (ADR 0010). With more than one node a resume may land on a node that does not host the player's room, matches must survive a rolling deploy, and the operator needs a signal that says when to add or remove nodes. Budgets: p99 move round-trip under 100 ms in region, 10k concurrent rooms per node (CLAUDE.md). Ephemeral coordination state belongs in Redis (ADR 0005).

## Decision

### Gateway and game node are one process
- There is no separate gateway tier. Every server pod terminates WebSockets, hosts rooms and runs matchmaking. A split would add a hop to every move and a second deployable to operate; the relay below covers the one case a split would solve.

### Room to node assignment
- A room is opened on the node that forms it (matchmaking tick, custom lobby start). Public traffic is spread by the load balancer; a node whose `rooms` reached `HD_NODE_CAPACITY` or that is draining reports it in its heartbeat and stops forming new matches (`cluster.Accepting`), so new rooms land on nodes with free capacity. `cluster.Least` names the node with the most free capacity for callers that place work explicitly.
- Every node writes a heartbeat to Redis (`hd:node:<id>`, JSON with address, capacity, rooms, connections, draining; TTL `HD_NODE_TTL`) and its id into the set `hd:nodes`. A node whose heartbeat expired is dropped from the set on the next read. Membership changes are published on `hd:members`.

### Player to node routing
- When a session is seated in a room, the hosting node claims `hd:route:<sha256(token)[:16]>` = node id with TTL `HD_ROUTE_TTL`; the claim is refused if another live node owns it. The hosting node re-claims the routes of its routed sessions every half `HD_ROUTE_TTL`, so a long match or a long-open friend lobby stays resumable. The route is released when the session leaves the room after the reconnect grace or is revoked. The raw session token never reaches Redis (`docs/environments.md` S-3).
- A hello with a token the node does not know is looked up in the routes. If another live node owns it, the node dials that node's internal address over the cluster network with the `X-HD-Relay` header, replays the hello and pipes frames both ways for the life of the connection. The player keeps one public endpoint and TLS stays on the load balancer. A node accepts relayed connections while draining, so a player whose connection dropped during a deploy resumes into the old pod and finishes the match there.
- A relayed connection is never relayed again: a hosting node answers a foreign hello that arrived through a relay with `UNAUTHENTICATED`.

### Node loss
- Rooms are not migrated. When a node dies, its heartbeat expires within `HD_NODE_TTL`, lookups of its routes return "node gone" and the route is deleted, and clients get `UNAUTHENTICATED` on resume. The match is over for everyone at the table; no result is written, no rating moves and no penalty is applied, because the fault is the server's. Players return to the lobby and queue again. A planned stop is never a node loss: the drain of ADR 0010 lets rooms finish.

### Auto-scale signal
- `GET /loadz` on any node returns the node's own load and the cluster summary: live nodes, rooms, connections, capacity, utilisation (`rooms / capacity` over non-draining nodes) and an advice: `out` when utilisation is at or above `HD_SCALE_OUT_AT` or no node is live, `in` when more than one node is live and utilisation is at or below `HD_SCALE_IN_AT`, otherwise `hold`. The autoscaler of #25 reads this endpoint; thresholds are configuration, not code.

## Alternatives considered and why rejected
- Per-pod public addresses (one hostname or port per node): the client would learn the topology and a certificate per pod; the relay keeps one endpoint.
- Frames over Redis pub/sub instead of a direct relay: every move of a relayed player would cross Redis twice, against the 100 ms p99 budget; pub/sub carries only membership events.
- Moving room state between nodes on loss: the room's in-memory state, timers and bot controllers would need a serialised form and a hand-over protocol; the reconnect grace plus the drain already cover planned stops, and an unplanned loss is rare enough that ending the match is the cheaper, honest outcome.
- A fleet manager that allocates one process per match: see ADR 0010; our process hosts thousands of rooms.

## Consequences
- Easier: horizontal scaling by adding pods; deploys and reconnects during a drain need no client change; the operator has one number for scale decisions.
- Harder: Redis is on the resume path, so a Redis outage turns cross-node resumes into `UNAUTHENTICATED` until it returns (same-node resumes still work). Route keys of a node that stops renewing them without a clean release expire after `HD_ROUTE_TTL`.
- Must be verified: the two-node relay and drain tests of `server/internal/cluster` run against a real Redis 7 in CI; the 10k rooms per node figure waits for the reference node of #192 (`docs/performance.md` section 9 holds the developer-machine run).
