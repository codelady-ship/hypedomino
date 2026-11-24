# 0004. Transport and protocol: WebSocket with binary protobuf

Status: Accepted
Date: 2026-10-25
Issue: #10

## Context
Turn-based real-time play with small messages, mobile networks that change IP and drop, and a p99 move round-trip budget of 100 ms in region. Client and server are in different languages (C#, Go) and must share one contract. Old clients stay installed for a long time, so the protocol must evolve safely.

## Decision
- Transport: one WebSocket over TLS per client session, carrying binary protobuf frames. Gameplay, presence and notifications share the connection. Account and purchase flows that are not latency-critical may use HTTPS endpoints served by the same backend.
- Contracts live in `proto/`, the only source for message definitions. C# and Go code are generated from it; handwritten message classes are forbidden.
- CI runs `buf lint` and `buf breaking` against `main` on every change under `proto/`. A breaking change fails CI.
- Envelope: every frame is one `Envelope` with the protocol version, a client-assigned sequence number and a oneof payload. Server frames carry the last acknowledged client sequence, so intents are idempotent and retriable after reconnect.
- Versioning:
  - The package is versioned by major (`hypedomino.v1`). A breaking redesign is a new package (`v2`) served in parallel for a transition window, and needs its own ADR.
  - Within a major version only additive changes are allowed. Field numbers are never reused or renumbered; removed fields are `reserved` with their names. Every enum has an `UNSPECIFIED` zero value, and unknown values must be handled safely.
  - The first message after connect is a client hello carrying the app version and protocol major. The server compares the app version with an admin-managed minimum supported version and answers accept, `UPDATE_REQUIRED` or `MAINTENANCE`. On `UPDATE_REQUIRED` the client blocks play and shows a store-update screen; on `MAINTENANCE` it shows the maintenance screen with the end time from the hello (#146). The minimum version is config, not a constant.
- Reconnect: a session token plus the last acknowledged sequence lets the server resume the room or resend a snapshot. Timeouts and grace windows are admin-tunable.

## Alternatives considered and why rejected
- JSON over WebSocket: easy to debug. Rejected because of larger frames, parsing allocations on the client (conflicts with 0 B/frame) and no enforced schema or breaking-change check.
- gRPC / gRPC-Web: strong typing and tooling. Rejected because mobile Unity support is awkward (HTTP/2 stack, IL2CPP issues), server push to many idle rooms is heavier than a WebSocket, and gRPC-Web needs a proxy.
- Raw UDP / ENet / KCP: lowest latency, no head-of-line blocking. Rejected because a turn-based game with small messages does not need it, carrier NAT and filtering hurt reachability, and reliability and ordering would be rebuilt by hand. Revisit only if measurements show TCP head-of-line blocking violates the budget.
- MessagePack: compact and schemaless. Rejected because without an IDL there is no cross-language contract and no `buf breaking` equivalent; protobuf gives both, with mature C# and Go generators.
- FlatBuffers / Cap'n Proto: zero-copy decoding. Rejected as complexity not justified by message sizes; protobuf with reused messages is expected to meet the budget. Reconsider if profiling shows decode cost.

## Consequences
- Easier: one contract, generated code on both sides, mechanical compatibility checks, small frames.
- Harder: binary frames need tooling to inspect, so a debug dump tool is part of the server work. Generated C# allocations must be controlled in the client by message reuse; this is checked in ADR 0008.
- TCP head-of-line blocking on bad networks is accepted; resume logic hides it from the player.
- Every `proto/` PR states whether it is additive.
