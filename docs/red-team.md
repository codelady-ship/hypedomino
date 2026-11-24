# Red team plan and release gate

The plan for #212: we attack our own game with the tool classes a cheater uses and record, per attack, which rule of `docs/anti-cheat.md` defeats it, how it is tested and the result. The release gate (GN-5 in `docs/launch-runbook.md`) passes only when no attack yields currency, cosmetics, pass progress, rating or score outside legal play, and no critical finding is open.

Three layers, each with its own owner and date:

| Layer | Proves | Runs | Owner |
|---|---|---|---|
| A. Automated server harness | Every client-reachable input the server trusts is checked against server state; forged, replayed, malformed and flooded inputs change nothing | Every pull request (`server` CI job) and on demand | Server lead |
| B. Device and APK attacks | The client-side tool classes reach only layer A's surface; client detections raise signals | Release candidate build on a real device or emulator | Security lead, after a signed candidate build exists |
| C. Third-party penetration test | An outside team finds what we missed | Before launch, on staging | Product owner chooses the vendor |

## 1. Tool classes

Every tool class ends at the same place: whatever it does on the device, it can only send intents and HTTP requests to the server, and layer A attacks exactly those. The Result column is the state on 2026-11-14.

| Tool class | Attack | Defeated by | How it is tested | Result |
|---|---|---|---|---|
| Memory editor (GameGuardian and similar) | Rewrite balance, ownership flags, timers, tile values; read the opponent's hand | NT-1, NT-3, NT-4: nothing worth editing is on the device. An edited tile or balance becomes an intent the rules engine or the economy refuses (NT-2). Hidden tiles never reach the device (E-1) | A: tiles not held, out of turn, illegal placements, forged balance, price and reward fields in every write. B: edit the shown balance and a hand tile, then act | A: pass. B: not run |
| Billing patcher and license bypass (Lucky Patcher and similar) | Fake the store's success answer; submit forged, reused, test or other-account receipts | NT-5, V-3: the client's success grants nothing; `/shop/complete` validates with the store, binds the receipt to the account and the transaction id grants once | A: receipts the store never issued, another account's receipt, one receipt resent, a cheap receipt for an expensive offer, a test purchase in production, a cancelled purchase, a forged App Store certificate chain, forged store notifications. B: patcher against the candidate build | A: pass. B: not run |
| Hooking and instrumentation (Frida, Xposed and similar) | Replace functions: skip checks, script play, forge attestation, read memory | NT-7, NT-8, NT-10: client checks are risk signals only; no secret in the client; attestation is verified on the server against a server nonce (V-5) | A: everything a hooked client can send, as for the other classes. B: hook signals reported (`instrumentation`, `hook_framework`, `debugger`, `docs/anti-cheat.md` 5.4) and recorded; scripted play | A: pass. B: not run |
| Modded or re-signed APK | Unpack, patch (unlock menus, local premium, disabled checks), re-sign and share | V-2: Play Integrity signing digest and install source, App Attest, minimum version and build allowlist (#205, #210). Unowned cosmetics stay invisible to others (NT-9) | A: hello below the minimum version, unknown build, forged identities; equip of unowned items. B: a re-signed build against staging with Play Integrity on | A: pass. B: not run |
| Intercepting proxy (Burp Suite, mitmproxy) | Read and edit WebSocket and HTTPS traffic; inject, replay and reorder frames and requests | NT-1, V-4, V-8: values the server never reads; request MACs on critical writes; session nonce on every frame; idempotent writes; pinning (#207) | A: edited bodies with and without a valid MAC, MACs of another body, path, key or token, frames without or with a foreign nonce, a captured frame replayed after reconnect, a duplicated frame, unknown fields, server-only payloads. B: pinning against a user-installed certificate | A: pass. B: not run |
| Emulator and multi-instance | Run many accounts, farm rewards, enable the other classes | V-6, V-9, V-10, NT-12: device verdict and client device signals are risk input; rate limits per IP and account; rewards only from server-recorded matches | A: floods of intents, sign-ins, requests and purchases are rate limited; concurrent purchases against one balance grant once. B: emulator and root signals recorded, never a ban alone | A: pass. B: not run |

## 2. Layer A: the automated harness

`server/cmd/server/redteam_*_test.go`. Run it with Docker available:

```
cd server
go test -count=1 -run RedTeam -v ./cmd/server/
```

It starts PostgreSQL 16 and Redis 7 through testcontainers like every other integration test (`HD_TEST_DATABASE_URL` and `HD_TEST_REDIS_URL` point it at existing ones), migrates a fresh database, publishes the example configs of `tools/dev/publish-configs.sh` (the pass season moved to cover today) and starts the game server through the same `build` function `cmd/server` runs in production, with the production defaults for rate limits, MACs, frame limits and store environment (`HD_STORE_SANDBOX` off). Attacks go over real WebSocket frames (`internal/botclient` plus raw frames) and real HTTP. The only fakes are the stores: Google Play is an HTTP server behind `purchase.GoogleConfig.Endpoint` and a generated service account, which no environment variable can set; App Store transactions are verified against the real Apple root, so a forged chain is refused exactly as in production.

Why tests in `cmd/server` rather than a `cmd/redteam` binary: the attack surface is the composition in `build` (limiter, MAC, integrity gate, handlers, session manager), and package tests are the only place that runs it unchanged without adding a test-only switch to production config. They run in `go test -race ./...`, so the existing `server` job runs them on every pull request and `CI complete` needs no change. A binary against a deployed stack would need a staging identity path, which does not exist yet (`docs/test-strategy.md` 13).

Every attack asserts the documented refusal and that the account's wallets, ledger, inventory, pass progress, purchases, promo redemptions, daily claims and loadout are unchanged (`economy` in `redteam_test.go`); match attacks assert that the table (board, hand, scores, turn, boneyard) is unchanged and that the match record holds only the legal intents. The server log must show no recovered panic.

| Test | Attacks | Refusal asserted |
|---|---|---|
| `TestRedTeamHelloRefusesEveryForgedIdentityAndVersion` | Missing and future protocol version; app below the minimum; no token; token signed with another key; unsigned (`alg: none`) token; expired token; the bare session id of a live session; a refresh token; an intent before the hello; a hello above `MaxHelloBytes`; a claimed acked sequence and a device clock two days ahead | `UNSUPPORTED_PROTOCOL` 1002, `UPDATE_REQUIRED` 1000, `UNAUTHENTICATED` 1008, 1008, 1009; the server keeps its own sequence and time and records `clock_skew` |
| `TestRedTeamFramesAreBoundToTheirSessionAndShape` | Frames without and with a foreign or shifted session nonce; queue entry with unknown mode and queue; leader-only lobby commands from a member; garbage, text, oversized frames; a flood of 60 intents | `REPLAY_REJECTED` and a `replay_reject` signal; `QueueState REFUSED`; lobby refusal with the leader unchanged; close 1008, 1003, 1009; `RATE_LIMITED` |
| `TestRedTeamMatchRefusesIllegalIntentsAndReplays` | In a live 1v1 room: opening out of turn, with a tile not held, at an end of an empty board; pass, draw and end turn at the opening; a legal opening carrying an unknown field (accepted once); the same frame sent again; a tile not held that fits the board; a held tile at an end it does not fit or that does not exist; pass and draw while holding a legal placement; a placement out of turn; tiles and ends outside the set; payloads only the server sends (a finished `MatchView` with forged scores, `ServerHello`, `QueueState MATCH_FOUND`, `EmojiShown`, a second hello, no payload) from the seat whose turn it is; the captured opening replayed after a reconnect | `ILLEGAL_INTENT` with the rejected sequence and an unchanged table; the duplicate is dropped; `REPLAY_REJECTED`; the record holds one intent |
| `TestRedTeamForgedValuesInEveryWriteAreIgnored` | Every client write (`/shop/buy`, `/shop/begin`, `/shop/complete`, `/inventory/equip` owned and unowned, `/pass/levels`, `/daily-login/claim`, `/medals/showcase`, `/missions/reroll`, `/promo/redeem`) sent by one account as is and by another with balance, price, cost, amount, reward, bundle, score, rating, XP, level, rank, inventory, ownership, premium and state fields added, plus another account's id | Both accounts get the same answer and end in the same state; a body the endpoint refuses as too large changes nothing; the named account is untouched |
| `TestRedTeamSpendsAreBoundedIdempotentAndSigned` | Pass level counts 0, -1, minimum int, 1000, 2^62, maximum int and maximum int - 1 in coins and crystals; one purchase request sent twice; a second purchase of a one-per-account offer; five concurrent purchases with coins for one; writes without a MAC, with the MAC of another body, path, key or token, and without an access token | 409 `pass.level_count`; one charge; 409 `offer.limit_reached`; one grant and a zero balance; 401 `bad_mac` and a `mac_failure` signal |
| `TestRedTeamFloodsAreRateLimited` | Ten sign-ins from one address, forty requests from one account, ten purchases from one account | 429 `rate_limited` |
| `TestRedTeamReceiptsGrantOnlyWhatTheStoreSold` | See the billing patcher row of section 1 | 400 `receipt.unknown`, 403 `receipt.account`, one grant for one receipt, 409 `receipt.product`, `receipt.environment`, `receipt.cancelled`, 400 `receipt.signature`, `receipt.malformed`, 401 on forged notifications; `fake_receipt` signals |
| `TestRedTeamPromoCodesResistGuessingAndReuse` | One code redeemed twice; twelve random codes in a row from one account | Second redemption refused; 429 `rate_limited`; a `promo_lock` signal and `promo.locked` for the real code afterwards |

Not covered by layer A, and why: Play Integrity and App Attest verdicts need tokens from Google and Apple (their verification has unit tests in `internal/integrity`; the gate is off in the harness because no attestation provider is configured); TLS pinning is client code (layer B); collusion and automated play are statistical (V-9, V-10, `internal/rating` tests); the admin API is out of the player's reach and has its own RBAC tests.

## 3. Findings

| Id | Severity | Finding | State |
|---|---|---|---|
| F-1 | High | `session.Manager` looked the hello token up as a session key before authenticating it. Authenticated sessions are keyed by their session id (`sid`), which anyone can read from the access token or the refresh token. So a hello carrying a bare `sid` resumed the session, with its seat, without a signature or expiry check: an expired or leaked access token kept granting the WebSocket session for the life of the node, and an account whose refresh family was revoked elsewhere (admin API, theft detection) kept playing. Fixed: with an authenticator set, every hello must verify as an access token (`session.sessionFor`, the route lookup too). Test: `TestTheBareSessionIdOfAnAuthenticatedSessionIsNotACredential` and the harness hello case | Fixed in the #212 pull request |
| F-2 | Medium | `POST /pass/levels` checked `level + count > levels`, which overflows for a `count` near the maximum int, so the request reached the price: `count × levelCoinPrice` wrapped to a credit (for example +200 coins). Only the ledger's sign rule stopped it, with a 500. Fixed: the count is compared as `count > levels - level`. Test: `TestBoughtLevelsKeepProgressInsideTheLevelAndChargeOnce` and the harness level counts | Fixed in the #212 pull request |
| F-3 | Medium | NT-2 and V-1 say an illegal intent is counted and `IllegalIntentThreshold` in `IllegalIntentWindowMinutes` raises a signal. Illegal intents are refused, but nothing counts them: no metric, no risk signal. Detection of memory-editor use therefore rests on client signals only. Needs owner values for `IllegalIntentThreshold`, `IllegalIntentWindowMinutes` and `IllegalIntentWeight` (no launch defaults in `docs/anti-cheat.md` 7) | Open, owner decision |
| F-4 | Low | `POST /pass/levels` spends coins or crystals but is not in the spend bucket of NT-12; only the account bucket limits it. It is server-priced and idempotent per request id, so this is not an economy hole | Open, informational |
| F-5 | Low | The promo throttle counts attempts before the redemption and records them after it, so parallel requests can pass `PromoAttemptsPerHour` together. The spend bucket bounds it to its burst (5) per account, and codes are 12 characters of a 31-letter alphabet | Open, informational |

## 4. Layer B: device and APK (owner, later)

State of this machine on 2026-11-14, checked for #212: the Android emulator and an AVD `Pixel_7` (Android 37, x86_64, Google Play image, 16 KB pages) exist and `adb` works, with no device attached; Unity 6000.3.27f1 is not installed, so no APK can be built here. A development build needs no signing secret (`androidUseCustomKeystore` is off, so Unity signs with its debug keystore), but the client targets ARM64 only (`client/README.md`), so an x86_64 emulator runs it only through the image's ARM translation, and the native Google and Apple sign-in plugins are not built yet, so no build can sign in to a server. Layer B therefore waits for a candidate build with sign-in. Rules: test devices and staging only, accounts made for the test, no tool downloaded onto a personal device, every result recorded in the table of section 1 and in the #212 report.

| Case | Steps | Pass when |
|---|---|---|
| B-1 Memory editor | Rooted test device or emulator; search and change the shown coin balance, an owned flag and a hand tile; buy, equip, play the tile | The next server value replaces the display; buy and equip are refused or charged at the server price; the tile intent is refused |
| B-2 Billing patcher | Patch the store response on a test device; buy a crystal pack | Nothing is granted; `/shop/complete` refuses the receipt; a `fake_receipt` signal appears |
| B-3 Hooking | Attach an instrumentation agent and a hook framework; skip pinning, script moves | `instrumentation` or `hook_framework` reaches `risk_signals`; scripted moves gain nothing layer A did not already refuse |
| B-4 Re-signed APK | Unpack, patch a premium flag, re-sign with another key, install outside Play | Play Integrity fails on the signing digest and install source; purchases and ranked are blocked; the patched flag shows nothing to other players |
| B-5 Proxy | Install a user CA, route through an intercepting proxy | Pinning refuses the connection; with pinning patched out, edited requests behave as in layer A |
| B-6 Emulator and multi-instance | Run the build on an emulator and in several instances | `emulator` and the device verdict reach `risk_signals`; rate limits hold; no ban from these signals alone |

## 5. Layer C: third-party penetration test

Scope for the vendor (choosing the vendor and budget is the owner's decision, #212):

- Game server on staging: WebSocket protocol (`proto/`), every player HTTP route, rate limits, session and token handling, store notification endpoints.
- Purchase and promo flows end to end with store test accounts.
- Android and iOS release candidate builds: reverse engineering, tampering, pinning, local storage (`docs/anti-cheat.md` Device storage), attestation.
- Admin API and panel from outside: authentication, RBAC, approval flow (`docs/admin.md`); no production data.
- Out of scope: production, denial of service beyond rate limit checks, social engineering of staff.
- Deliverable: findings with severity and reproduction, retested after fixes; criticals block GN-5.

## 6. Gate

GN-5 is green when: layer A is green on the release candidate commit (`CI complete` of GN-7 includes it); every layer B case passed on the candidate build or has an owner waiver on #212; the layer C report has no open critical finding; and section 3 has no open finding of high severity or above.
