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
- The first client frame is `ClientHello`. A new session (not a resume of a known session id) is checked for device integrity: the hello carries `integrity`, either the Play Integrity token or an App Attest assertion (key id, nonce, assertion) for a `session_start` nonce from `POST /integrity/nonce`. Its `session_token` is the access token of `POST /auth/signin` or `POST /auth/refresh` (`server/README.md`, Sign-in); the resume session is keyed by the token's session id, so a refreshed token resumes the same seat. The server answers `ServerHello` with `HELLO_STATUS_ACCEPTED` or `HELLO_STATUS_UPDATE_REQUIRED`; on the latter the client blocks play and shows the store update screen. Server time, the active catalog version and `content_base_url`, the CDN root the client downloads the catalog and bundles from (`docs/content-delivery.md` 2), arrive in the same message. An accepted hello also carries `string_bundles`: language, version and content hash of every active string bundle. It also carries `fallback_chains`: the lookup order per language (`docs/localization.md` 1.2, 2.2). A language without a bundle uses the built-in one. `in_match` tells a resuming client that its session sits in a running match, so it returns to the match scene instead of the lobby (R-58). `session_nonce` is a random value the server draws for this connection; every later client frame carries it in `Envelope.session_nonce`. A reconnect gets a new one, so frames captured from an earlier connection or sent into another session are refused (`docs/anti-cheat.md` V-8).
- A client-to-server payload is an intent: `PlaceTileIntent`, `EndTurnIntent`, `DrawIntent`, `PassIntent`. `LoadingIntent` reports loading progress (0 to 100) and `ready` during `MATCH_STATE_LOADING`; it carries no sequence and is never acknowledged or rejected (R-57). `EmojiIntent` sends one emoji by catalog id (#151); like `LoadingIntent` it carries no sequence and is never acknowledged or rejected. `QuickMessageIntent` sends one quick message by key with `team_only` (#152) and is handled the same way. A `PlaceTileIntent` with `BOARD_END_UNSPECIFIED` is the opening placement of a hand. A rejected intent is answered with `Error` carrying `rejected_sequence`; the gateway answers `ILLEGAL_INTENT` for a payload it cannot decode and `RATE_LIMITED` for an intent above the per-connection budget, and drops it.
- The server-to-client gameplay payload is `MatchView`: the whole visible state of one seat after every accepted intent, timer event, join or reconnect. It carries the seat's own hand, every seat's hand size and controller, the board, the legal moves of the seat whose turn it is, the turn deadline in server time, the last hand result and the winner. During loading it carries the `roster` (nickname, character, frame, avatar, rank label, emblems and the bot flag per seat), each seat's `loading_progress` and `ready`, and `loading_deadline_unix_ms`. `table` is the table background item id the server chose for the whole match (`docs/rarity.md` 5); the client renders it and never computes it. The client renders it and never derives a score or a legal move itself.
- `EmojiShown` relays an accepted emoji to every connected seat, the sender included, as `seat` and `item_id`. The room accepts an emoji only from a connected human seat whose loadout, frozen at match start (`docs/anti-cheat.md` NT-9), equips that id, and only within the seat's spam limit: a bucket of `HD_EMOJI_BURST` emoji (default 3) that refills one every `HD_EMOJI_REFILL` (default 2 s). Anything else is dropped without an answer, so the sender sees an emoji only when it comes back as `EmojiShown`. Emoji are relayed from memory and never stored or recorded in the replay (`docs/privacy.md` D-14).
- `QuickMessageShown` relays an accepted quick message as `seat`, `key` and `team_only`, to every connected seat or, with `team_only`, to the sender's team only, the sender included. The room accepts a key listed in the published `quick_messages` catalog from a connected human seat that is not muted, within the same spam limit as emoji (`docs/localization.md` 5.7). Like emoji they are never stored.
- `JoinQueueIntent` (mode and `QueueKind`) and `LeaveQueueIntent` enter and leave matchmaking outside a match; the server answers with `QueueState`: status, mode, queue, join time, the time the search starts, and for a refusal its `QueueRefusal` and, for a matchmaking ban, the time it ends (`docs/matchmaking.md` section 4). The server handling follows in #478.
- Tiles are `Tile { low, high }` with `low <= high` outside the board. Ends are `BoardEnd`; `TOP` and `BOTTOM` exist only once there is a spinner (`docs/rules.md` R-17, R-18).

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
| Hello frame size `MaxHelloBytes` (room for a Play Integrity token) | 8192 | Close 1009 |
| Frame size `MaxFrameBytes` | 1024 | Close 1009 |
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

The generated C# parser reuses an existing instance for a singular message field (`if (tile_ == null) Tile = new Tile(); input.ReadMessage(Tile);`) but allocates a fresh instance for every `oneof` member on each parse. `Envelope.payload` is a `oneof`, so the client never parses an `Envelope` with the generated parser. It reads the three varints and the payload tag with `CodedInputStream` by hand, keeps one long-lived instance per payload type, and calls `input.ReadMessage(instance)` on the matching one. Sending works the same way: one `Envelope` per intent type is kept and only its scalar fields change. Repeated fields are cleared and refilled, not reallocated. This is a usage rule for `client/` (#67), checked by the allocation tests of `docs/test-strategy.md` section 2.5; generation cannot enforce it.
