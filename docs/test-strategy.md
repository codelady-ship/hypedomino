# Test strategy

What is tested at which level, by what, where the test lives, which CI job runs it and who owns it, for every component of the monorepo. It also fixes the end-to-end bot client design, the regression list re-run before every release, test data rules and the flaky test policy. Epic 17 (#189), issue #190.

Scope boundaries: budgets and their measurement are `docs/performance.md` and ADR 0008; device certification is #191; load and soak is #192 (tooling #53); the red team is #212 against `docs/anti-cheat.md`; release go/no-go is `docs/launch-runbook.md` and the release gate is `docs/roadmap.md` section 4. This document links them and does not restate their numbers except where a test asserts them.

Status: no server, client, admin or proto code exists yet. Every test below is a plan; the "Lands in" column names the issue that writes it. Numbers marked *proposal* are starting values changed by PR to this file.

## 1. Principles

| Id | Rule |
|---|---|
| TS-1 | Tests assert behaviour stated in a spec: `docs/rules.md` rule ids, `docs/economy.md`, `docs/ranking.md`, `docs/penalties.md`, `docs/anti-cheat.md` NT and V ids. A test name carries the id it checks (`R-31`, `NT-4`, `V-8`), so a failure points at the spec line. |
| TS-2 | The server is the only judge (ADR 0003, ADR 0007). Correctness of rules, scores, rewards, balances and ratings is tested on the server. Client tests check rendering, input, state application and allocation, never legality or values. |
| TS-3 | Everything is deterministic: injected randomness with a logged seed, injected clock, no sleeps, ports from `:0`, one isolated database per test package. A test that cannot be made deterministic is not merged. |
| TS-4 | Every bug fix lands with a test that fails before the fix. |
| TS-5 | Rules, scoring, economy and ranking changes need tests before merge (CLAUDE.md). A PR that changes one of their specs changes its tests in the same PR. |
| TS-6 | Tests run in the existing CI jobs where possible. A new CI job joins `CI complete` (section 10). A failing check is fixed, never skipped or disabled. |
| TS-7 | No production data in any test, fixture or non-production environment (section 7). |

## 2. Test levels per component

The pyramid: most checks are fast pure tests (rules vectors, property tests, client edit-mode), fewer are integration tests against real PostgreSQL and Redis, and a small set of end-to-end bot scenarios drives the real server over the real protocol. Device runs, load and soak sit outside PR CI.

| Level | Server (Go) | Proto | Client (Unity) | Admin (Next.js) |
|---|---|---|---|---|
| Unit, pure | Rules vectors, property and fuzz tests, ranking and economy formulas | Lint, format, breaking change | Edit-mode tests | Unit and component tests |
| Allocation and benchmark | `-benchmem` benchmarks, `benchstat` | n/a | Allocation constraint tests, play-mode 0 B/frame check | n/a |
| Integration | Services against PostgreSQL and Redis in containers | Golden frames decoded in Go and C# | Play-mode with a fake transport fed by recorded frames | Admin API against seeded PostgreSQL |
| End to end | Bot clients play full matches over WebSocket (section 3) | Covered by bot scenarios | Device smoke against staging (section 4) | Browser e2e of admin flows |
| Non-functional | Load and soak (#192) | n/a | Device matrix (#191), battery (#164) | Lighthouse and security (#148) |

### 2.1 Server: rules package

`server/internal/rules` is pure (ADR 0003): no I/O, no clock, randomness injected. That makes it the cheapest and most complete layer.

| Test | What it asserts | Lands in |
|---|---|---|
| Vector runner | One table-driven test reads every `docs/rules-vectors/*.json` except `schema.json`, dispatches on `kind`, compares with `expect` field by field; the vector id is the subtest name; an unknown `kind` fails (`docs/rules-vectors/README.md`). Non-tunable values use the `rules.Config` defaults of `docs/rules.md` section 13 | #30 to #38, #39 |
| Property tests | From random seeded deals and random legal play: the 28 tiles are conserved across hands, board and boneyard (R-1 to R-3); every move the engine lists as legal is accepted and every other one is refused (R-20 to R-23); every score and hand total is a multiple of `ScoringMultiple` (R-25, R-33, R-37, R-38); no team score changes in the middle of a hand except by placement scores; a match ends only at a hand end by domino with a team at or above `TargetScore` and strictly ahead (R-43 to R-48); every match terminates | #39 |
| Fuzz | `go test -fuzz` targets on intent application and state decoding: no panic, an illegal intent returns an error and leaves state unchanged | #39 |
| Benchmarks | Processing one move under 5 microseconds and 0 allocs/op (`BenchmarkPlace`, `BenchmarkLegalMoves`, `BenchmarkMatch` per step); the `bench` CI job runs `benchstat` against `main` and reports without gating, per `docs/performance.md` section 6 "Hot-path cost" | #39 |
| Bot (#40) | Never plays an illegal move under fuzz; the hardest level wins at least 80% against random legal play; decision within 50 ms | #40 |
| Replay (#41) | Replaying a record (seed and moves) gives the same hands, scores and result | #41 |

The vectors are the only hand-computed rule fixtures. A vector that disagrees with `rules.md` is a bug in the vector (`docs/rules-vectors/README.md`); the engine is never changed to match a wrong vector. `npm run test:tools` already validates the vectors themselves (schema, coverage, independent arithmetic) before any engine exists.

### 2.2 Server: economy, ranking and penalties

Formulas are pure functions tested table-driven; stateful invariants are property tests against the ledger in a real PostgreSQL (section 2.3).

| Area | Fixtures | Invariants (property tests) | Lands in |
|---|---|---|---|
| Ledger (`docs/economy.md` section 5) | Every reason row of the ledger table with its idempotency key | Balance equals the sum of entries after any sequence of writes; repeating a write with the same key returns the first result and changes nothing (V-8); concurrent spends of one account never drive the balance below zero; only `refund_revoke` can make it negative (section 9); `purchase` keys are unique across accounts | #101 |
| Reward bundle (section 4) | Bundle examples from the spec | A grant writes ledger, inventory, medals and pass XP together or not at all (fault injected between writes); the grant records source, bundle id, revision and config version | #102 |
| Caps and sources (sections 2, 10) | `docs/economy/balance.csv` as the config seed | An account's daily earn never exceeds the legitimate maximum of its reasons under the config in force (section 10); first win, daily login and mission keys grant once per period | #103, #110 |
| Authoritative economy (#202) | Generated intents | Fuzzed and mutated economy intents never increase a balance; no client-to-server message has a balance, price or reward field (the proto lint of #23) | #202, #23 |
| Refunds (section 9) | Store notice samples | Processing a notice twice revokes once; `REFUND_REVERSED` re-grants through `refund_reversal` | #204 |
| Ranking (`docs/ranking.md` section 4) | Worked examples of section 4.6 as a table test, config defaults of section 10 | LP change is within [`LpChangeMin`, `LpChangeMax`]; the sign of `dMMR` follows `S`; a result in one mode never changes the other; applying the same match end twice changes nothing; ranked queues never seat a bot (section 9) | #112, #113, #118 |
| Penalties (`docs/penalties.md`) | Offense event sequences | Offense level and ladder per section 3 and 4; result mapping per section 5; replaying a match end records no second offense | #52 |

### 2.3 Server: integration

Packages that touch PostgreSQL or Redis test against real instances started per test package in containers (testcontainers-go on the CI runner's Docker), at the major versions production runs (#24). Each package migrates a fresh database with the real migrations (#55); no mocks of SQL.

| Suite | Asserts | Lands in |
|---|---|---|
| Migrations | Up from empty to head; every expand migration keeps the previous server version working (runbook 5.1) | #55 |
| Match persistence | Result, participants and events written atomically; a crash between steps never loses or doubles a result (#50 acceptance) | #50 |
| Ledger and inventory | Section 2.2 invariants against real transactions and isolation levels | #101, #61 |
| Redis state | Presence, queue, routing and session keys carry TTLs; losing Redis loses no balance or purchase (ADR 0005) | #45, #48, #51 |
| Config snapshot | A config version published through the admin API is loaded by the game server; a match records the version it used; publishing an earlier version applies to new matches only (ADR 0006) | #145 |

Integration tests run in the default `go test ./...` and therefore in the existing `server` CI job. They skip only under `go test -short` for local quick runs; in CI a missing Docker is a failure, not a skip.

### 2.4 Proto contracts

| Check | How | Lands in |
|---|---|---|
| Lint, format, breaking change against the PR base | `buf lint`, `buf format --diff --exit-code`, `buf breaking` (exists in the `proto` CI job) | exists |
| Breaking change against the last released tag | `buf breaking` against the release tag, because released clients stay installed (ADR 0004) | release workflow, #27 |
| Generated code drift | Regenerate Go, C# and TS and fail on a diff | #23 |
| No values from the client | A lint over client-to-server messages rejects balance, price, reward, score, rating and inventory fields (NT-1) | #23, #202 |
| Golden frames | `proto/testdata/` holds binary `Envelope` frames for every message type; the Go test and a Unity edit-mode test both decode them and compare with the expected fields | #23 |

### 2.5 Client: Unity

Unity Test Framework, one test assembly per runtime assembly (#65 assembly definitions).

| Test | What it asserts | Lands in |
|---|---|---|
| Edit-mode | Pure C# logic: chain layout geometry for line, spinner arms and auto-scale (#80); applying a server snapshot or event to the view state; reconnect state machine (#67); localization key coverage (#70); golden frame decoding (section 2.4) | #65, #67, #80 |
| Play-mode | Scene flows driven by a fake transport replaying recorded server frames (section 7.3): deal, place, draw, pass, hand end, match end, reconnect, `UPDATE_REQUIRED` screen. Asserts what is shown equals what the frames say; the client never shows a value it did not receive | #66, #78 to #87, #73 |
| Allocation, per call | `Assert.That(() => step(), Allocates.Nothing)` on every per-frame and per-input method after warm-up. `Allocates.Nothing` (`client/Assets/HypeDomino/Tests/Support`) counts `GC.Alloc` samples with a `ProfilerRecorder` that collects only on the calling thread and requires 0. The engine's `Is.Not.AllocatingGCMemory()` is not used: its shared recorder sometimes reports one allocation for an empty step (#593); `tools/allocation-constraint` fails `npm run test:tools` if a client file uses it | #72, #593 |
| Allocation, per frame | Play-mode replay of a recorded 2v2 match reads `ProfilerRecorder` "GC Allocated In Frame" each frame from first deal to result screen and requires 0 B, with `GC.CollectionCount(0)` unchanged | #72 |
| Performance capture | Unity Performance Testing package records frame time and markers during the same replay; results are trend data only | #72, #161 |
| Build checks | Build size report per PR with a warning when the budget would be exceeded | #76 |
| Assembly layout | Every asmdef is in the Layout table of `client/README.md` with exactly the project references it lists, references point only up the table, and Core references nothing. Runs in the `lint` job (section 2.7), so it needs no Unity license | #76 |

Editor numbers do not represent devices (ADR 0008). The CI allocation test catches code-path regressions early; the gates in `docs/performance.md` section 3 are passed only by device runs (#191, #161).

### 2.6 Admin: Next.js

Unit and component tests run with `pnpm test`, browser e2e as a separate suite. The test runner choice is made in #134 (*proposal*: Vitest and Playwright).

| Test | What it asserts | Lands in |
|---|---|---|
| Unit | Form validation matches the spec bounds of each config key; config diff view | #134 to #145 |
| RBAC matrix | Every role against every write action: allowed or refused as the admin spec says | #134 |
| Audit | Every write produces an audit entry with who, what, before, after and reason | #134 |
| Config publish e2e | Create a value, publish, see a new version id; publish an earlier version; the game server loads it (section 2.3) | #145 |
| Catalog e2e | Cosmetic CRUD produces a manifest that passes the cosmetics schema (`tools/cosmetics-schema`) | #136 |
| Hardening | Lighthouse above 90 in performance, accessibility and best practices (SEO excluded, the pages are `noindex`: `docs/admin.md` O-9), no critical vulnerability (#148 acceptance); procedure and last result in `admin/README.md`; the vulnerability half runs in CI (section 10.1) | #148 |

### 2.7 Spec tooling (exists)

`npm run lint` runs the comment lint; `npm run test:tools` runs `node --test "tools/**/*.test.mjs"`: comment lint, rules vectors, economy balance sheet, pass sample season, design tokens, art direction and cosmetics schema validators, and the client assembly layout (`tools/assembly-layout`: every asmdef under `client/Assets` matches the Layout table of `client/README.md`, and references point only up it). They run in the `lint` CI job on every PR and keep the spec fixtures that server tests consume (vectors, balance CSV, manifests) valid before the code exists.

## 3. End-to-end bot clients

Goal (#190 acceptance): bot clients play a full match on the real server automatically, headless, in CI.

### 3.1 Design

| Part | Decision |
|---|---|
| Client | A Go protocol client in `server/internal/botclient`: connects over WebSocket, sends the client hello, authenticates, wraps intents in `Envelope` with sequence numbers, tracks the last acknowledged sequence and resumes with the session token (ADR 0004). It uses the generated Go code from `proto/`, the same contract the Unity client uses |
| Brain | The bot sees only what its seat receives: its own hand and the public board. It chooses moves with the #40 bot or a seeded random legal choice, computed by `server/internal/rules` on that visible state. The server still validates every intent |
| In CI | Scenario tests in `server/internal/e2e` start the real server (app, gateway, session manager, room) in process on `127.0.0.1:0`, then connect N bot clients over real WebSocket frames. A scenario starts PostgreSQL and Redis containers (section 2.3) once it asserts on stored state. They run in `go test ./...`, so the existing `server` job runs them |
| Against staging | `server/cmd/botclient` runs the same scenarios against a deployed environment: release smoke at T-1d (runbook section 4) and opponents for device smoke (section 4) |
| Determinism | The deal source is injected (ADR 0003). Production wires a cryptographically secure source (R-4). E2E wires a seeded source (`math/rand/v2` ChaCha8) that exists only in test code, so no production binary can be seeded. The seed is logged on every failure. The server clock is injected, so timer scenarios advance time instead of waiting |
| Identity | In process, the server is built with a fake ID-token verifier. On staging, bot accounts use a staging-only identity path that the production config refuses to start with (#57, #24) |
| Load reuse | The load bot (`server/cmd/loadbot`, #192, #53) reuses `botclient` for the protocol so load and E2E speak one implementation |

### 3.2 Assertions on every scenario

| Id | Assertion | Source |
|---|---|---|
| E-1 | No frame sent to a seat contains a tile of another seat or of the boneyard | R-4, NT-4 |
| E-2 | The final scores and result equal a replay of the match record through the rules package | #41 |
| E-3 | The match ends exactly as R-43 to R-48 say: at a domino hand end, a team at or above the target and ahead | `docs/rules.md` |
| E-4 | Exactly one match result row; exactly one `match_reward` ledger entry per rewarded player with key `match:<matchId>`; ranked MMR and LP applied once | #50, `docs/economy.md` section 5, `docs/ranking.md` section 1 |
| E-5 | Every refused intent leaves server state unchanged and increments the illegal intent counter | NT-2, V-1 |
| E-6 | No server error log line and no goroutine left after the room closes | #46 |
| E-7 | Every seat sends and receives under 200 KB of frame payload in a whole match (`docs/performance.md` 6.2) | #163 |

### 3.3 Scenarios

Seed counts are *proposals*.

| Id | Scenario | Runs in |
|---|---|---|
| S-1 | 1v1 full match to the target, #40 bots, fixed seeds | PR CI |
| S-2 | 2v2 full match to the target, #40 bots, fixed seeds | PR CI |
| S-3 | Seed sweep: random legal bots, 1v1 and 2v2 | PR CI 20 seeds, nightly 5,000 |
| S-4 | Illegal intents: tile not in hand, wrong end, out of turn, pass or draw while holding a legal placement (R-20 to R-23), action at or after the turn limit (R-50) | PR CI |
| S-5 | Duplicate sequence answered idempotently; frame with a foreign session nonce refused | PR CI, V-8, #207 |
| S-6 | Turn timeout: bot moves for the player (R-50); after `AfkConsecutiveTimeouts` the player is AFK (R-51); reconnect takes control back and resets the count | PR CI |
| S-7 | Leaver replaced by the bot (R-52); coming back before the end takes control back and resets the timeout count; penalty recorded once (`docs/penalties.md` section 7) | PR CI |
| S-8 | Disconnect mid-turn and resume with session token and last acknowledged sequence | PR CI, #45 |
| S-9 | Server stops mid-match and restarts: result not lost and not doubled | PR CI, #50 |
| S-10 | App version below the configured minimum gets `UPDATE_REQUIRED` | PR CI, ADR 0004, #210 |
| S-11 | Ranked queue with an empty seat never seats a bot; casual queue fills it after the configured wait (R-56) | PR CI, #48 |
| S-12 | Mutated and random frames from an authenticated bot: no panic, no balance or state change | Nightly fuzz, #202 |

#190 is closed when S-1 and S-2 run green in the `server` job on `main`.

S-1 and S-2 live in `server/internal/e2e/match_test.go` (`TestS1OneVersusOneFullMatch`, `TestS2TwoVersusTwoFullMatch`). Each seat is a `botclient.Brain`: the #40 bot at the hard level deciding on a `rules.Board` rebuilt from the seat's own view. The room clock never fires, so no turn times out and every intent in the record comes from a bot client. They assert E-1, E-2, E-3, E-6, E-7 and that no intent is refused (E-5). E-4 joins when match results are stored from the room (#50).

## 4. Device farm smoke

Owned by #191 on the `docs/performance.md` section 1 matrix (at least 12 devices, Android 8+ and iOS 14+, per #191). The farm provider and the UI automation driver are chosen in #191 and #76.

Smoke run, per device, on the release candidate build against staging, with `botclient` (section 3) filling the other seats of a custom room (R-55, unranked):

1. Install, cold start, interactive lobby (cold start is measured separately, `docs/performance.md` section 3).
2. Sign in with a staging test account.
3. Create a custom room, play one full hand to its end against bot clients.
4. Toggle the network off and on mid-turn; the match resumes (#87).
5. Background and foreground the app; state matches the server.
6. Open shop, collection and settings; no error.
7. On designated devices only: one licensed test purchase is granted once (GN-22).
8. Crash reporting shows the session (#182).

Pass: every step on every device tier, recorded per device in the release issue (GN-24). Budget gates (fps, draw calls, GC, size, battery) are certification runs in #191, not smoke.

## 5. Load and soak

Owned by #192 with tooling from #53; method and pass criteria are `docs/performance.md` section 6. The numbers the run must show:

| Check | Target | Source |
|---|---|---|
| Move round-trip | p99 under 100 ms under load, overall and per minute window | CLAUDE.md, #192 |
| Rooms per node | 10,000 active rooms (about 40,000 connections), error and disconnect rate under 0.1%, average CPU under 80%, heap flat within 5% | `docs/performance.md`, #53 |
| Peak | Three times the expected launch peak | #192 |
| Connections | 50,000 concurrent connections in the load test; average move message under 64 bytes | #44 |
| Reconnect storm | Mass disconnect and reconnect within the resume window | #192 |
| Soak | 12 hours, no memory growth | #53 |
| Data stores | PostgreSQL and Redis bottlenecks found and recorded | #192 |

Load runs on staging sized like production, never on CI runners. It is re-run at T-7d against the production config version (runbook section 4).

## 6. Security regression

The red team (#212) attacks the release candidate with the `docs/anti-cheat.md` vectors V-1 to V-11; the plan, results and findings are `docs/red-team.md`. Its automatable part becomes permanent tests, among them the red team harness `server/cmd/server/redteam_*_test.go`, which attacks the server as `cmd/server` wires it over real frames and requests in the `server` job:

| Vector | Permanent test |
|---|---|
| V-1 memory editor | S-4 and E-5: illegal intents refused and counted |
| V-3 fake purchase | Forged, reused, sandbox-in-production and other-account receipts refused by #204 integration tests |
| V-4 proxy | Edited economy intents change nothing (#202 fuzz, S-12) |
| V-7 clock | Every time window computed from the injected server clock; a client time field changes nothing (#208) |
| V-8 replay | S-5 and the ledger idempotency properties (section 2.2) |
| V-11 unowned items | Equip and claim intents for unowned items refused (#211) |
| Signals | A simulated cracked client is flagged (#209 acceptance) |

## 7. Test data

### 7.1 Sources

| Data | Source of truth | Used by |
|---|---|---|
| Rule cases | `docs/rules-vectors/` | Rules engine tests, E2E replay checks |
| Ranking cases | `docs/ranking.md` section 4.6 and section 10 defaults | Ranking tests |
| Economy config | `docs/economy/balance.csv` | Seeded config snapshot, economy tests |
| Rules config | `docs/rules.md` section 13 defaults | `rules.Config` in tests |
| Cosmetics | Example manifests validated by `tools/cosmetics-schema` | Admin catalog tests, client collection tests |
| Protocol | `proto/testdata/` golden frames | Go and C# decoders |
| Matches | Seeds and replay records (#41) | E2E, client play-mode, performance replay |

### 7.2 Seeded environments

- One seed command creates the local stack (#22, `make dev` under 2 minutes), the CI databases and staging from the same seed: synthetic accounts, a published config version built from the sources above, and a cosmetic catalog from the example manifests.
- A test that needs a different value sets it explicitly in its own config version (for example a short turn time in a timer scenario); it never edits the seed.
- Synthetic accounts only. Store purchases use store test accounts and licensed testers (#106); production rejects sandbox receipts (V-3).

### 7.3 Recorded frames

The E2E suite writes the frame stream each seat received for selected seeds. These recordings feed the client play-mode tests (section 2.5) and the scripted match replay of `docs/performance.md` section 2. They are regenerated, never edited by hand, when the protocol or a rule changes, in the same PR.

### 7.4 No production data

No production database dump, player record, receipt or log is copied into a test, fixture, local stack or staging. A production incident is reproduced from its replay record (seed and moves, `docs/privacy.md` D-13) and the config version it used, which identify no player. Secrets come from the environment only.

## 8. Regression list

Re-run on the release candidate commit before every release; the result is the evidence for GN-24 and GN-7 in `docs/launch-runbook.md`. Owner: QA lead unless stated.

| Id | Run | Evidence |
|---|---|---|
| RG-1 | Every CI job on the candidate commit, path filters ignored (release workflow, #27) | Run link |
| RG-2 | Rules vectors, property tests and the fuzz seed corpus; benchmarks against the previous release with `benchstat` | Job log, benchstat output |
| RG-3 | Full E2E scenario set S-1 to S-12, nightly seed counts | Job log |
| RG-4 | Integration suite on the production PostgreSQL and Redis versions | Job log |
| RG-5 | `buf breaking` against the previous release tag; golden frames of the previous release decode on the new server | Job log |
| RG-6 | Migrations: expand on a staging copy; the previous server binary runs on the new schema (runbook 5.1) | Drill record |
| RG-7 | Unity edit-mode, play-mode and allocation tests | Job log |
| RG-8 | Admin unit, RBAC, audit and config publish and rollback e2e | Job log |
| RG-9 | Device smoke on every tier (section 4) | Per device record, #191 |
| RG-10 | Budget measurements for every changed performance path; load and soak at T-7d | `docs/performance.md` reports, #192 |
| RG-11 | Security regression (section 6); red team on the candidate | #212 report, Security lead |
| RG-12 | Licensed test purchase granted once and restored (GN-22) | Ledger entry, Live-ops lead |
| RG-13 | Every test added for a bug fixed since the previous release passes (TS-4) | Included in RG-1 |

## 9. Flaky tests

A test that fails and then passes on the same commit is flaky. A flaky test is a failing test.

- One re-run is allowed to confirm flakiness. A PR does not re-run until green.
- The person who sees it opens a `type:test` issue with the seed, the log and the job link, labelled with the area and `priority:p1`.
- The fix is a root-cause fix (shared state, real clock, real randomness, ordering, port or timing assumption) or deleting a test that has no value and replacing it with a deterministic one. No retry wrapper, no skip, no quarantine of a required check (CLAUDE.md).
- Until it is fixed, merges in that area wait for the fix.
- Nightly jobs that fail open a `type:test` issue the same way; they never block PRs.

## 10. CI mapping

### 10.1 What exists (`.github/workflows/ci.yml`)

| Job | Runs | Trigger |
|---|---|---|
| `changes` | Path filter for `server/`, `admin/`, `client/`, `proto/`; an area counts as changed only if its project file exists (`server/go.mod`, `admin/package.json`, `client/ProjectSettings/ProjectVersion.txt`, `proto/buf.yaml`) | Every PR and push to `main` |
| `lint` | `npm run lint` (comment lint), `npm run test:tools` (spec validators) | Always |
| `secret-scan` | Pinned gitleaks over the PR's commits, or the full history on `main` (`docs/environments.md` section 4) | Always |
| `pr-meta` | One `type:`, `area:` and `priority:` label; `Closes #N` or `Part of #N` in the body | PRs |
| `server` | `gofmt`, `go vet`, `go build`, `go test -race ./...`, `govulncheck` (pinned) over `./cmd/admin-api/...`, failing on every reachable vulnerability (#148, `docs/admin.md` O-10) | Server changed |
| `image` | Builds `server/Dockerfile`; `tools/deploy/container-smoke.sh` (non-root user, bad config exits 2, health, drain on `SIGTERM`); `deploy/k8s/base`, `deploy/k8s/overlays/ci` and `deploy/k8s/monitoring` render and pass kubeconform; `promtool check rules` and `promtool test rules` on the alert rules of `deploy/k8s/monitoring` (the latency alert of ADR 0014 fires after 15 minutes over budget and stays quiet within budget or at low traffic); `tools/deploy/kind-rollout.sh` on a kind cluster (the release through `tools/deploy/apply-release.sh`: the `migrate` Job migrates a throwaway PostgreSQL to the newest migration before the rollout, a second release applies nothing, a failed job leaves the server untouched; rolling deploy keeps every replica available, old pods drain and stop, rollback under 5 minutes, ADR 0010); `tools/deploy/kind-metrics.sh` (the debug port `6060` answers the `monitoring` namespace, is refused from another namespace, is not in the `server` Service and answers pprof with 404; the loopback pprof listener `6061` is refused from the pod network and reached through `kubectl port-forward`) | Server, `deploy/` or `tools/deploy/` changed |
| `admin` | `pnpm install --frozen-lockfile`, `typecheck`, `test`, `build` (scripts if present), `pnpm audit --audit-level=critical` (#148) | Admin changed |
| `client` | Unity test runner, `testMode: all` (edit-mode and play-mode) | Client changed |
| `proto` | `buf lint`, `buf format --diff --exit-code`, `buf breaking` against the PR base | Proto changed |
| `CI complete` | Fails if any needed job is neither `success` nor `skipped`, or if `changes`, `lint` or `secret-scan` did not succeed | Always |

Today no project file exists, so `server`, `admin`, `client` and `proto` are skipped on every run.

### 10.2 Where each test runs

| Tests | Job | New job needed |
|---|---|---|
| Rules vectors, property, fuzz seed corpus, economy, ranking, penalties, integration, E2E S-1 to S-11 | `server` | No |
| Coverage report (section 11) | `server` | No |
| Benchmarks with `benchstat` against `main` | Report only, not in `CI complete`: shared runners are too noisy to gate; the PR measurement rule of `docs/performance.md` section 8 is the gate | Yes, non-blocking (#39) |
| Proto lint, breaking, generated code drift, value-field lint, golden frames | `proto` and `server` | No (#23) |
| Unity edit-mode, play-mode, allocation tests | `client` | No |
| Build size report | Client build job | Yes (#76, #27) |
| Admin unit, RBAC, audit | `admin` | No |
| Admin browser e2e | Needs a database and a browser | Yes (#134) |
| Spec validators | `lint` | No |
| Long fuzz, S-3 nightly sweep, S-12 | Scheduled workflow | Yes, separate workflow |
| Device smoke, load, soak | Staging and device farm | Not CI (#191, #192) |
| Full regression (section 8) | Release workflow on the tag | Yes (#27) |

### 10.3 Adding a job

Per CLAUDE.md: a new job that may block a merge is added to the `needs` list of `ci-complete`. A job that must run on every PR (not path-gated) is also added to the list of jobs that must succeed in its check step, next to `changes`, `lint` and `secret-scan`. A path-gated job only joins `needs`, where `skipped` counts as passing. Scheduled and release workflows are separate files and never part of `CI complete`. This document does not edit `ci.yml`; each issue in the table above adds its own job.

## 11. Coverage targets

All *proposal*; measured, reported in CI and raised or lowered by PR to this file. Coverage is a floor check, not a goal: a test without an assertion on behaviour does not count in review.

| Code | Line coverage | Measured by |
|---|---|---|
| `server/internal/rules` | 95%, enforced by the `server` CI job | `go test -coverprofile` |
| Economy, ledger, ranking, penalties packages | 90% | `go test -coverprofile` |
| Rest of `server/` | 75% | `go test -coverprofile` |
| Client logic assemblies (no scene or view code) | 60% | Unity Code Coverage package |
| `admin/` | 70% | Admin test runner coverage |

A PR may not lower the coverage of `server/internal/rules` or the economy packages; elsewhere a drop needs a reason in the PR body.

## 12. Ownership

Roles as in `docs/launch-runbook.md` section 1. The author of a change writes its tests; review follows `.github/CODEOWNERS`.

| Area | Owner |
|---|---|
| This document, regression list, release regression run, flaky test triage | QA lead |
| Rules, economy, ranking, penalties, integration and E2E bot tests | Server lead |
| Proto contract checks | Server lead and client lead |
| Unity tests, allocation tests, device smoke | Client lead |
| Admin tests | Client lead (`/admin/` owners) |
| Security regression and red team | Security lead |
| Load and soak | Server lead (#192) |
| Seed data and config snapshot | Live-ops lead for values, server lead for the seed command |

## 13. Open items

- The E2E acceptance of #190 (S-1 and S-2 green in CI) needs the rules engine (#30 to #40), the protocol (#23, #44), rooms (#46), sessions (#45) and the schema (#55). #190 stays open until then.
- The staging-only identity path for bot accounts (section 3.1) needs the token design of #57 and #58 and the environment separation of #24.
- Device farm provider and UI automation driver: #191, #76.
- Admin test runner: #134.
- On a push to `main` the path filter compares with the previous commit, so a release commit does not run every job by itself; RG-1 needs the release workflow of #27.
