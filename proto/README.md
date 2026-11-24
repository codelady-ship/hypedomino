# proto

Shared protobuf contracts between client, server and admin. This directory is the only source of message definitions (ADR 0004). Generated code is never edited by hand.

## Layout

| Path | What |
|---|---|
| `buf.yaml` | Module, lint (`STANDARD`) and breaking (`FILE`) configuration |
| `buf.gen.yaml` | Code generation: pinned remote plugins and output directories |
| `hypedomino/v1/*.proto` | The `hypedomino.v1` package, one major version |
| `testdata/frames/*.json` | Golden `Envelope` frames, one per payload type, in protobuf JSON |
| `testdata/frames/*.bin` | The same frames in binary wire format, written by the Go test |

Generated output:

| Language | Plugin | Output | Used by |
|---|---|---|---|
| Go | `protocolbuffers/go` | `server/internal/pb/hypedomino/v1` (package `hypedominov1`) | Game server, bot client |
| C# | `protocolbuffers/csharp` | `client/Assets/HypeDomino/Generated/Proto` (namespace `Hypedomino.V1`) | Unity client |
| TypeScript | `bufbuild/es` | `admin/src/gen/proto/hypedomino/v1` | Admin panel |

## Commands

Run from the repository root. `buf` is required; Go is required for the golden frame test.

| Command | What |
|---|---|
| `buf lint proto` | Style and naming rules |
| `buf format --diff --exit-code proto` | Formatting |
| `buf breaking proto --against ".git#branch=main,subdir=proto"` | Wire and source compatibility against `main` |
| `npm run lint:proto` | No value field reaches a client-to-server message (below) |
| `npm run proto:gen` | Regenerate Go, C# and TypeScript. Commit the output |
| `cd server && go test ./internal/pb/...` | Decode every golden frame and compare with its JSON |
| `cd server && go test ./internal/pb/... -update` | Rewrite the binary golden frames from their JSON |

CI runs all of these on every change under `proto/` or a generated directory and fails when the committed generated code differs from the contracts.

## Protocol

- One WebSocket per session carries binary `Envelope` frames in both directions. Every frame has `protocol_version`, the client-assigned `sequence`, the server's `acked_sequence` and exactly one payload.
- The first client frame is `ClientHello`. A new session (not a resume of a known session id) is checked for device integrity: the hello carries `integrity`, either the Play Integrity token or an App Attest assertion (key id, nonce, assertion) for a `session_start` nonce from `POST /integrity/nonce`. Every hello may carry `device_risk_signals`, the device risk names the client collected, and `hook_signals`, the hook, instrumentation and debugger names; the server only weighs them as risk (`docs/anti-cheat.md` 5.4). Its `session_token` is the access token of `POST /auth/signin` or `POST /auth/refresh` (`server/README.md`, Sign-in); the resume session is keyed by the token's session id, so a refreshed token resumes the same seat. The session id alone is not a credential: a hello whose token does not verify as an unexpired access token is refused `UNAUTHENTICATED`. The server answers `ServerHello` with `HELLO_STATUS_ACCEPTED` or `HELLO_STATUS_UPDATE_REQUIRED`; on the latter the client blocks play and shows the store update screen. Server time, the active catalog version and `content_base_url`, the CDN root the client downloads the catalog and bundles from (`docs/content-delivery.md` 2), arrive in the same message. An accepted hello also carries `string_bundles`: language, version and content hash of every active string bundle. It also carries `fallback_chains`: the lookup order per language (`docs/localization.md` 1.2, 2.2). A language without a bundle uses the built-in one. It also carries `cosmetics_manifest`: the `manifestVersion` and content hash of the published cosmetics manifest (`docs/cosmetics.md` 7.3); it is absent until a manifest is published, and the client then keeps its built-in one. It also carries `analytics_config`: `batch_size`, `queue_limit` and `flush_seconds` of the client analytics queue, the `AnalyticsBatchSize`, `AnalyticsQueueLimit` and `AnalyticsFlushSeconds` keys of the published `liveops` snapshot with the defaults for keys it leaves out (`docs/analytics.md` 6); it is absent from a server without a database, and the client then keeps its current settings. `in_match` tells a resuming client that its session sits in a running match, so it returns to the match scene instead of the lobby (R-58). `session_nonce` is a random value the server draws for this connection; every later client frame carries it in `Envelope.session_nonce`. A reconnect gets a new one, so frames captured from an earlier connection or sent into another session are refused (`docs/anti-cheat.md` V-8).
- A client-to-server payload is an intent: `PlaceTileIntent`, `EndTurnIntent`, `DrawIntent`, `PassIntent`. `LoadingIntent` reports loading progress (0 to 100) and `ready` during `MATCH_STATE_LOADING`; it carries no sequence and is never acknowledged or rejected (R-57). `EmojiIntent` sends one emoji by catalog id (#151); like `LoadingIntent` it carries no sequence and is never acknowledged or rejected. `QuickMessageIntent` sends one quick message by key with `team_only` (#152) and is handled the same way. A `PlaceTileIntent` with `BOARD_END_UNSPECIFIED` is the opening placement of a hand. A rejected intent is answered with `Error` carrying `rejected_sequence`; the gateway answers `ILLEGAL_INTENT` for a payload it cannot decode and `RATE_LIMITED` for an intent above the per-connection budget, and drops it.
- The server-to-client gameplay payload is `MatchView`: the whole visible state of one seat after every accepted intent, timer event, join or reconnect. It carries the seat's own hand, every seat's hand size and controller, whether a human seat has no live connection (`disconnected`, false for a bot seat, so a server that does not fill it reads as connected) and its `consecutive_timeouts`, with `afk_consecutive_timeouts`, the number of them that hands the seat to the bot (R-51, `AfkConsecutiveTimeouts` of the match's rules config), the board, the legal moves of the seat whose turn it is, the turn deadline in server time, the last hand result and the winner. During loading it carries the `roster` (nickname, frame, avatar, tile skin, rank label, emblems and the bot flag per seat; `character` is deprecated and never set, since the game has no characters), each seat's `loading_progress` and `ready`, and `loading_deadline_unix_ms`. `table` is the table background item id the server chose for the whole match (`docs/rarity.md` 5); the client renders it and never computes it. The client renders it and never derives a score or a legal move itself. `roster` and `table` never change during a match, so the server sends them only in the first `MatchView` of each connection; a later view without a roster keeps the roster and table the client already has. A reconnect starts a new connection and gets them again. The view also carries `target_score` (R-43) and `hand_points_lock_threshold` (R-41) from the match's rules config, so the score meter marks them without a client constant. `turn_seconds` is the turn length (`TurnSeconds`, R-49) the deadline counts down from, and `turn_warning_seconds` the time left at which the turn timer ring shows the warning colour (`TurnWarningSeconds`, `docs/ui-motion.md` Turn timer); both come from the match's rules config, so the ring needs no client constant. `locked_teams` lists the teams that had reached the lock threshold when the current hand started and so cannot score hand points in it (R-41); the client shows their lock badge from it and never compares scores itself. `last_result` explains the last hand: `ending` (domino or block), `hand_points_locked` when the winner had reached the lock threshold at the start of the hand and got no hand points (R-41), and `carried_in`, the carry-over (R-39) that is part of `awarded`, so `awarded` minus `carried_in` is the hand points of R-33 or R-37. `left` lists the tiles each seat held when the hand ended, for the hand summary. Those tiles are hidden while the hand is played (NT-4 in `docs/anti-cheat.md`) and are sent only once the hand is over, and only in the first view of each connection after that hand ended; later views leave `left` empty.
- `EmojiShown` relays an accepted emoji to every connected seat, the sender included, as `seat` and `item_id`. The room accepts an emoji only from a connected human seat whose loadout, frozen at match start (`docs/anti-cheat.md` NT-9), equips that id, and only within the seat's spam limit: a bucket of `HD_EMOJI_BURST` emoji (default 3) that refills one every `HD_EMOJI_REFILL` (default 2 s). Anything else is dropped without an answer, so the sender sees an emoji only when it comes back as `EmojiShown`. Emoji are relayed from memory and never stored or recorded in the replay (`docs/privacy.md` D-14).
- `QuickMessageShown` relays an accepted quick message as `seat`, `key` and `team_only`, to every connected seat or, with `team_only`, to the sender's team only, the sender included. The room accepts a key listed in the published `quick_messages` catalog from a connected human seat that is not muted, within the same spam limit as emoji (`docs/localization.md` 5.7). Like emoji they are never stored.
- `JoinQueueIntent` (mode, `QueueKind` and `integrity`, the device proof for a `ranked_match` or `casual_match` nonce, `docs/anti-cheat.md` 5.1) and `LeaveQueueIntent` enter and leave matchmaking outside a match; the server answers with `QueueState`: status, mode, queue, join time, the time the search starts, and for a refusal its `QueueRefusal` and, for a matchmaking ban or a ranked ban (`RANKED_BAN`, ranked entries only), the time it ends and `ban_reason`, the `BanReason` category the client shows as `ban.reason.<category>` (`docs/matchmaking.md` section 4, `docs/admin.md` 6.2), and `appeal`, an `AppealGrant` with the `sanction_id`, a `token` valid only for `POST /appeals` and its `expires_unix_ms` (`docs/admin.md` 6.4, Appeals; absent for a ban recorded before sanction rows). `BanReason` is `CHEATING`, `COLLUSION`, `AFK_LEAVING`, `OFFENSIVE_NAME`, `PAYMENT_FRAUD` or `OTHER`; a penalty ladder ban is always `AFK_LEAVING`. A `MATCH_FOUND` on a connection that does not host the match is followed by a close with status 1001; the client reconnects like a resume.
- `LobbyIntent` drives the friend lobby (`docs/friend-lobby.md`, R-55) outside a match: one `LobbyCommand` with the fields it needs: `CREATE`, `JOIN` (`code`), `LEAVE`, `TEAMS` (`mode`), `BACK`, `PLACE` (`account_id`, `seat`), `REMOVE` (`account_id`), `START`, `INVITE` (`account_id`), `REQUEST_SEAT` (`seat`) and `ADD` (`account_id`, `seat`: a member without a seat into a free seat, in team setup or during the match, where the member gets its `MatchView`). Only the leader may change teams, seats, members or start. A `JOIN` whose room lives on another server node is answered by closing the connection with `going away`; the reconnect lands in the room and gets its `LobbyState` right after `ServerHello`, as does any resume of a session that is still in a room. Every change is pushed to each member as `LobbyState`: code, leader, members in join order, phase, mode and the seats of the mode; the client reads nicknames, avatars, frames and rank from the public profile. A refused command is answered with `LobbyState` carrying the current room (empty when the player is in none) and its `LobbyRefusal`. A `CREATE` or `JOIN` during a ranked ban is refused `RANKED_BAN` with `refused_until_unix_ms`, `ban_reason` and `appeal`, as in `QueueState`. An empty `LobbyState` without a refusal means the player is no longer in a lobby (left, removed or disconnected outside a match). `LobbyRelay` carries an accepted emoji (`item_id`) to every member, and a seat request (`requested_seat`) to the leader only. In the lobby phase `EmojiIntent` sends a lobby emoji; it is dropped without an answer like a match emoji. `INVITE` needs the invited account to be a friend (`NOT_FRIEND`) and not blocked; an online friend gets `LobbyInvite` with the code and the inviter. The lobby actions were removed with the characters (2026-11-12): `ACT`, `LobbyIntent.action`, `LobbyRelay.action` and `UNKNOWN_ACTION` are deprecated, kept only because removing a field breaks the wire contract, and the server answers `ACT` like an unknown command (`UNAVAILABLE`). `START` opens a `KindCustom` match and every seated member gets its `MatchView`; when the match ends the members get the lobby `LobbyState` again.
- Tiles are `Tile { low, high }` with `low <= high` outside the board. Ends are `BoardEnd`; `TOP` and `BOTTOM` exist only once there is a spinner (`docs/rules.md` R-17, R-18).
- `BoardView.line` runs from the left end to the right end, each `OrientedTile` with `first` on its left; `top` and `bottom` run outward from the spinner, `first` touching the tile before it. `opening_index` is the place of the hand's opening tile in `line`, so the client can anchor the chain on it before a spinner exists (`docs/ui-motion.md` 5, Board).

## Gateway limits

`server/internal/gateway` serves the WebSocket. Every number below is a `gateway.Config` field, filled from admin config (#210); the defaults are in `DefaultConfig`.

| Limit | Default | On violation |
|---|---|---|
| First frame is a `ClientHello` within `HelloTimeout` | 5 s | Close 1008 |
| `protocol_version` equals the server's | 1 | `Error UNSUPPORTED_PROTOCOL`, close 1002 |
| `app_version` at least `MinimumAppVersion` (numeric `major.minor.patch`) | 1.0.0 | `ServerHello UPDATE_REQUIRED`, close 1000 |
| `build_hash` on the build allowlist, when `build_allowlist_enforced` is on (`docs/anti-cheat.md` 5.2) | off | `ServerHello UPDATE_REQUIRED`, close 1000 |
| Session token accepted by the handler | | `Error UNAUTHENTICATED`, close 1008 |
| Device integrity of a new session (`docs/anti-cheat.md` 5.1) | | `Error INTEGRITY_FAILED`, close 1008 |
| Hello and `JoinQueueIntent` frame size `MaxHelloBytes` (room for a Play Integrity token) | 8192 | Close 1009 |
| Frame size `MaxFrameBytes` for every other frame | 1024 | Close 1009 |
| Binary frames that decode as `Envelope` | | Close 1008 (text frames: 1003) |
| Intents per connection: `IntentsPerSecond`, burst `IntentBurst` | 10, 20 | `Error RATE_LIMITED`, intent dropped |
| Every frame after the hello carries the connection's `session_nonce` | | `Error REPLAY_REJECTED`, frame dropped, connection kept; counted toward `ReplayRejectThreshold` |
| Server write queue `WriteQueue` frames per connection | 32 | Close 1008 (slow consumer) |
| Pong within `PongTimeout` after each server ping every `PingInterval` | 5 s, 15 s | Close 1001 (no pong) |
| One live connection per session token | | The earlier connection is closed 1008 (session replaced) |

Frames above `CompressionThreshold` (256 bytes) are sent with permessage-deflate without context takeover when the client negotiates it; intents stay well under it. Measured over a full 1v1 and 2v2 match in `TestPlayersFinishAMatchOverWebSocket`: 22 to 23 bytes per intent frame on average with the 9 byte `session_nonce` (13 to 14 before #207), against the 64 byte acceptance line of #44.

## Versioning rules

- The package is versioned by major: `hypedomino.v1`. A breaking redesign is a new package served in parallel for a transition window and needs its own ADR.
- Inside a major version only additive changes are allowed. Field numbers are never reused or renumbered. A removed field is kept as `reserved` with its number and its name.
- Every enum has an `_UNSPECIFIED` zero value. Readers handle unknown enum values safely.
- `buf breaking` runs against the PR base and fails the `proto` CI job on a breaking change. Every `proto/` PR states whether it is additive.
- `buf.gen.yaml` pins plugin versions. Bumping one is its own PR and regenerates all three outputs.

## Client-to-server messages carry no values

The client sends intents only (`docs/anti-cheat.md` NT-1). `tools/proto-lint` enforces it:

- A message is client-to-server when its name ends with `Intent`, or it is `ClientHello`. New client-to-server messages must follow this naming.
- Those messages, and every message type reachable from their fields, must not have a field whose snake_case name contains a value token: `balance`, `coin`, `price`, `cost`, `reward`, `score`, `points`, `rating`, `mmr`, `lp`, `xp`, `rank`, `tier`, `inventory`, `owned`, `result` (singular and plural).
- Server-to-client messages may carry any of these. A quantity the client legitimately chooses (for example a pass level count) is named `count`.

## Allocation on the client

The generated C# parser reuses an existing instance for a singular message field (`if (tile_ == null) Tile = new Tile(); input.ReadMessage(Tile);`) but allocates a fresh instance for every `oneof` member on each parse. `Envelope.payload` is a `oneof`, so the client never parses an `Envelope` with the generated parser. It reads the three varints and the payload tag with `CodedInputStream` by hand, keeps one long-lived instance per payload type, and calls `input.ReadMessage(instance)` on the matching one. Sending works the same way: one `Envelope` per intent type is kept and only its scalar fields change. Repeated fields are cleared and refilled, not reallocated. `client/Assets/HypeDomino/Net` (`FrameReader`, `FrameWriter`, `client/README.md` "Network frames") does the envelope part. This is a usage rule for `client/` (#67), checked by the allocation tests of `docs/test-strategy.md` section 2.5; generation cannot enforce it.
