# Anti-cheat threat model

Attack vectors against HypeDomino, and for each one the risk, the countermeasure, how it is detected and how the server responds. The stance is fixed by ADR 0007 (zero trust in the client); this document applies it vector by vector and links every control to the Epic 18 (#200) sub-issue that implements it. Related: ADR 0003 (authoritative server), ADR 0004 (transport and envelope), ADR 0005 (data), `docs/rules.md`, `docs/penalties.md`.

Every weight, threshold, duration and switch named here is admin config (section 7). This document fixes the structure, not the values.

## 1. What the attacker wants

| Target | Why it matters |
|---|---|
| Coin balance and purchases | Free currency or free paid items break the economy and revenue |
| Inventory, cosmetics, pass progress | Unowned premium items devalue what paying players hold |
| Rating, rank, leaderboards | Ranked integrity (R-53) is what competitive players pay attention to |
| Hidden information | Opponent and partner hands and the boneyard (R-4) decide matches |
| Time-gated rewards | Daily rewards, pass windows, cooldowns and season boundaries |

Out of scope here: denial of service and WebSocket abuse (#184), account takeover and login security (Epic 5), moderation of names and chat (#186).

## 2. Never trust the client

Code review checks every change against this list. A change that breaks a rule is rejected, whatever the client-side protection around it.

- **NT-1** The client sends intents only: play tile X at end Y, pass, draw, buy offer Z, claim reward W, equip item V. It never sends results, balances, prices, rewards, scores, ratings or inventory (ADR 0007).
- **NT-2** The server validates every intent against authoritative state: moves through `server/internal/rules` (ADR 0003), economy intents through the server's own prices and balances (#202). An illegal intent is rejected, changes nothing and is counted.
- **NT-3** No balance, price, reward, inventory, score or rating is stored authoritatively on the device. PlayerPrefs, files and caches are display caches that server state replaces. Editing them changes nothing on the server (#203).
- **NT-4** Hidden information never leaves the server: a client receives only the tiles its player holds and the public board (R-4). Deal order, boneyard contents and random seeds stay on the server.
- **NT-5** A purchase is granted only after server-side receipt validation with the store. The store transaction id is unique, and the receipt is bound to the buying account (ADR 0005, #106, #204).
- **NT-6** Every time-based decision uses server UTC: turn timers (R-49, R-50), daily rewards, offers, pass windows, cooldowns, season boundaries. The device clock is display only (#208).
- **NT-7** Everything the client says about itself is a claim: app version, build hash, signature hash, device model, root or emulator status. Trust comes only from attestation tokens verified on the server, and even those are risk signals, never proof (#205, #185).
- **NT-8** A client-side detection (hook, debugger, root, emulator) only raises the risk score. It never replaces a server check and is never the sole basis of a permanent ban.
- **NT-9** The cosmetic loadout is checked against server inventory and frozen on the server at match start. Other players see the server's loadout, not what a client claims (#211).
- **NT-10** The client holds no secret that grants anything: no server keys, no admin tokens, no reward or receipt signing keys. Anything shipped in the client is public (#206).
- **NT-11** Every economy write goes through the idempotent ledger with a reason and a source (#101, #202). No code path changes a balance outside it.
- **NT-12** Intents, logins, promo redemptions and purchases are rate limited and counted per account (ADR 0007, #184).

## 3. Responses

| Response | What happens | Who triggers it |
|---|---|---|
| Block | The request, intent or session is rejected and server state is unchanged. A counter is incremented | Server, immediately |
| Flag | A risk signal with its evidence is recorded and the account's risk score rises | Server, immediately |
| Shadowban | Ranked matchmaking pairs the account only with other shadowbanned accounts (cheater pool). It is hidden from public leaderboards. The player is not notified; a data export or access request shows it (`docs/privacy.md` 8.3) | Risk score at `ShadowbanScore`, applied in the next ban wave, or an admin |
| Temporary ban | No login for `TempBanHours` | Risk score at `TempBanScore`, applied in the next ban wave, or an admin. An account under 18 is sent to review instead of a score-based ban (`docs/privacy.md` 12.2, OQ-9) |
| Permanent ban | No login, permanently | An admin from the review queue only |
| Review | The account enters the admin review queue with all its signals and evidence; the admin decides and the player can appeal | Risk score at `ReviewScore`, any hard-evidence signal, or a penalty ladder review step (`docs/penalties.md`) |

- Blocks are immediate because they protect state. Shadowbans and temporary bans from the risk score are applied in ban waves every `BanWaveIntervalHours`, so an attacker cannot tie a ban to the action that caused it (#209).
- The risk score is the sum of signal weights with decay over `RiskDecayDays` (section 5). It is separate from the AFK and leaver offense level in `docs/penalties.md`; both feed the same admin review queue.
- Every response and every admin change is audited with a mandatory reason (#209).
- Owner-confirmed on #201 (2026-10-26): permanent bans only by an admin from review, soft signals never ban alone, shadowbanned accounts hidden from public leaderboards, every hard signal goes to review.

## 4. Attack vectors

Risk is the impact if the countermeasures did not exist. Every vector is verified by the red team before release (#212).

### V-1. Memory editor

| | |
|---|---|
| Attack | A memory editing tool on the device searches for and rewrites values: coin balance, item ownership flags, timers, tile values, the opponent's hand |
| Risk | High: memory editors are the most common mobile cheat and need no skill |
| Countermeasure | Nothing worth editing exists on the client (NT-1, NT-3, NT-4). The displayed balance is replaced by the next server value. An edited tile or move becomes an intent the server rejects. Opponent tiles are never in memory to read |
| Detection | Illegal intents per match and per account (tile not in hand, buy without enough balance, equip of an unowned item). Debugger and memory-tool signals from the client (#206) |
| Response | Block every illegal intent. Flag when illegal intents exceed `IllegalIntentThreshold` in `IllegalIntentWindowMinutes` |
| Issues | #202, #203, #206, #209, #212 |

### V-2. Repacked or modded app

| | |
|---|---|
| Attack | The APK or IPA is unpacked, patched (unlock menus, auto-play, disabled checks, local "premium") and re-signed with another key, then shared |
| Risk | High: one modded build reaches many players and removes all client-side checks at once |
| Countermeasure | Android: the signing certificate digest from the Play Integrity verdict (`appIntegrity.certificateSha256Digest`) is checked on the server against an allowlist; install source must be `PLAY_RECOGNIZED`. iOS: App Attest. Tokens are requested at session start, purchase and match start, bound to a server nonce and verified on the server, never trusted from a client-reported hash (#205). The client hello is checked against the minimum version and build allowlist (ADR 0004, #210). Unowned cosmetics stay invisible to others (NT-9, #211) |
| Detection | Attestation failure, unknown signing digest, unrecognised install source, build hash not on the allowlist |
| Response | Per `IntegrityFailureAction`: block login or flag. Ranked play and purchases are blocked when attestation fails. A leaked or weak build is cut off by removing it from the allowlist or raising the minimum version; purchases and ranked have separate kill switches (#210) |
| Issues | #205, #210, #211, #212 |

### V-3. Fake in-app purchase

| | |
|---|---|
| Attack | A billing patcher fakes the store's success response on the device, or the attacker submits a forged, reused, sandbox or other-account receipt |
| Risk | High: direct revenue loss and unlimited premium currency |
| Countermeasure | The client's "purchase succeeded" grants nothing. The server validates every receipt with the store API (#106) and rejects forged, unsigned and sandbox receipts in production. The transaction id is unique in PostgreSQL, so a receipt grants once (ADR 0005). The receipt is bound to the account (`obfuscatedAccountId` on Android, `appAccountToken` on iOS); use from another account is rejected. Refund and chargeback notifications revoke the goods through the ledger (#101, #204) |
| Detection | Validation failures, transaction id reuse, account mismatch, refund and chargeback rate per account |
| Response | Block the grant. Forged receipt, reuse on another account and account mismatch are hard evidence: flag and review. Refund or chargeback: revoke and flag |
| Issues | #204, #202, #209, #212 |

### V-4. Packet tampering through a proxy

| | |
|---|---|
| Attack | An intercepting proxy with a user-installed certificate reads and edits WebSocket and HTTPS traffic: changes amounts, injects intents, looks for hidden data |
| Risk | Medium: the server already decides everything, but readable traffic speeds up every other attack |
| Countermeasure | TLS with certificate pinning and a rotation plan (a backup pin shipped in every build) (#207). Edited amounts are ignored because the server never reads them (NT-1, #202). Every move is re-checked in the rules engine (NT-2). Hidden tiles are never sent (R-4), so reading traffic reveals nothing the player does not already see. Critical operations (purchase, reward claim, equip) carry a message authentication code with the session key (#207) |
| Detection | Pinning failure reported at the next good connection, MAC failures, malformed frames, intents inconsistent with state |
| Response | Block the frame. Flag when MAC failures exceed `MacFailureThreshold` in `MacFailureWindowMinutes` |
| Issues | #207, #202, #212 |

### V-5. Runtime hooking and instrumentation

| | |
|---|---|
| Attack | A dynamic instrumentation framework or hooking module attaches to the running app, replaces functions (skip pinning, fake attestation calls, auto-play, read state) and scripts the client |
| Risk | Medium: it defeats any client-side check, but the server still decides outcomes |
| Countermeasure | IL2CPP with managed stripping and encrypted metadata, symbol and string obfuscation (#206). Critical logic and keys stay on the server (NT-10). Attestation tokens are bound to a server nonce and verified on the server, so a hook cannot mint a valid token (#205). Performance budgets still hold (#206, `docs/performance.md`) |
| Detection | Hook, injected library, debugger and instrumentation-server signals from the client (#206). Attestation failures (#205). Action timing patterns of scripted play (V-10) |
| Response | Flag. A hook signal alone never bans (NT-8); together with other signals it moves the account toward shadowban or review |
| Issues | #206, #205, #209, #212 |

### V-6. Emulator, root and jailbreak

| | |
|---|---|
| Attack | The game runs on an emulator or a rooted or jailbroken device, which enables V-1, V-2 and V-5 and multi-instance farming |
| Risk | Medium: an enabler rather than a cheat; many honest players have rooted devices |
| Countermeasure | The Play Integrity device verdict and App Attest give a server-verified device signal (#205). Client root, jailbreak and emulator checks add risk input only (#185). Nothing gained on such a device bypasses NT-1 to NT-12 |
| Detection | Device integrity verdict, client root and emulator signals, many accounts on one device |
| Response | Per `DeviceRiskAction`: warn, block ranked entry, or flag. Never a ban by itself; false positives are measured on the device matrix (#185) |
| Issues | #205, #209, #212; device signals in #185 (Epic 16) |

### V-7. Clock manipulation

| | |
|---|---|
| Attack | The device clock or time zone is moved to collect daily rewards early, finish cooldowns or reopen expired offers |
| Risk | Medium: trivial to do and directly farms rewards when a client clock is trusted |
| Countermeasure | Every schedule, timer and reward window is computed in server UTC (NT-6, #208). Turn time limits and late actions are judged on the server (R-49, R-50). The client clock only formats countdowns the server sent |
| Detection | Skew between the client-reported time and server time |
| Response | None needed: the attack gains nothing. Skew is a low-weight signal only, because honest devices are often wrong |
| Issues | #208, #212 |

### V-8. Replay

| | |
|---|---|
| Attack | Captured frames or requests are sent again: a reward claim, a purchase, a move, or a whole recorded session |
| Risk | High: replayed claims and purchases mint currency if writes are not idempotent |
| Countermeasure | Every frame carries the session's sequence number (ADR 0004) and a session nonce; a frame from another session or with an old nonce is rejected (#207). A repeated sequence number inside the session is answered idempotently with the earlier result, so reconnect retries stay safe. Reward claims and ledger writes are idempotent per server period and source (#101). A receipt grants once (V-3) |
| Detection | Frames with foreign or expired nonces, out-of-window sequence numbers, duplicate claim and transaction ids |
| Response | Block. Flag when rejected replays exceed `ReplayRejectThreshold` in `ReplayRejectWindowMinutes` |
| Issues | #207, #202, #204, #212 |

### V-9. Collusion and multi-account farming

| | |
|---|---|
| Attack | Players share hidden hands through an outside channel in 2v2, trade wins, or run several accounts to farm rewards and rank |
| Risk | Medium: random partners in ranked 2v2 (R-54) make arranged teams hard, but win trading and farming accounts still distort ratings and the economy |
| Countermeasure | Partners are assigned at random and cannot be chosen (R-54). Custom rooms are unranked (R-55). Rating abuse rules (#121) and farm signals (#110) feed the risk score |
| Detection | The same accounts meeting more often than matchmaking explains, play that matches the partner's hidden tiles more often than chance, repeated losses to the same opponent, accounts sharing devices or networks (#183, #110) |
| Response | Flag and review. Confirmed collusion: shadowban or temporary ban, rating changes from the affected matches reverted by an admin |
| Issues | #209, #212; detection details in #183 (Epic 16), #121 (Epic 10), #110 (Epic 9) |

- Implemented in `server/internal/rating` (#121): the repeat limit of the same pair in ranked matchmaking and the `pair_frequency`, `feeding` and `win_trade` pair signals of `docs/ranking.md` section 11, stored in `rating_signals` and published on the `rating_signals` outbox topic with `CollusionWeight` applied by the risk score of #209.

### V-10. Automated play

| | |
|---|---|
| Attack | A bot, through a scripted client (V-5) or input automation, plays ranked or farms rewards around the clock |
| Risk | Medium: legal moves cannot be rejected, so only behaviour gives it away |
| Countermeasure | Rate limits on intents and matchmaking entries (NT-12, #184). Rewards come only from completed server-recorded matches |
| Detection | Action timing with too little variance, sessions longer than a person plays, perfect play rates, matches started without pauses (#209) |
| Response | Flag and review. Confirmed: shadowban or temporary ban |
| Issues | #209, #212 |

### V-11. Unowned cosmetics and pass content

| | |
|---|---|
| Attack | A modified client marks premium items or the premium pass track as owned, or a leaked promo code is shared publicly |
| Risk | Medium: only the attacker's own screen changes, unless the server accepts the claim |
| Countermeasure | Equip, pass track and shop actions are checked against server inventory every time; the loadout is frozen on the server at match start (NT-9). Every grant records its reason (#211). Promo redemption is rate limited and brute-force protected (#108) |
| Detection | Equip or claim intents for unowned items, redemption spikes for one promo code |
| Response | Block the intent. Flag repeated attempts. A leaked promo code is disabled by an admin |
| Issues | #211, #202, #209, #212 |

## 5. Signals and risk score

| Signal | Source | Kind | Weight (config) |
|---|---|---|---|
| Illegal intent rate over threshold | Rules engine, economy service | Soft | `IllegalIntentWeight` |
| Attestation failure, unknown signing digest, unrecognised install source | #205 | Soft | `IntegrityFailureWeight` |
| Build not on the allowlist | #210 | Soft | `UnknownBuildWeight` |
| Hook, debugger or instrumentation signal | #206 | Soft | `HookSignalWeight` |
| Root, jailbreak or emulator | #185 | Soft | `DeviceRiskWeight` |
| MAC failure rate over threshold | #207 | Soft | `MacFailureWeight` |
| Replay rejection rate over threshold | #207 | Soft | `ReplayRejectWeight` |
| Clock skew | #208 | Soft | `ClockSkewWeight` |
| Forged receipt, receipt reuse on another account, account mismatch | #204 | Hard | `FakeReceiptWeight` |
| Refund or chargeback | #106, #204 | Soft | `RefundWeight` |
| Balance or inventory change with no matching ledger source | #202 | Hard | `ImpossibleBalanceWeight` |
| Collusion, win trading, farm accounts | #183, #121, #110 | Soft | `CollusionWeight` |
| Automated play pattern | #209 | Soft | `AutomationWeight` |

- Hard signals are facts an honest client cannot produce. Each one sends the account to review at once, whatever its score.
- Soft signals happen to honest players too (bad networks, rooted devices, wrong clocks). They act only through the score.
- A simulated cracked client must be flagged; this is the acceptance test of #209.

### 5.1 Attestation verdict (#205)

`server/internal/integrity` turns one attestation token into a verdict with named failure reasons and a decision. It applies no penalty itself: the caller blocks the request, records the signal and feeds the score.

- **Nonce.** The server issues a random, single-use nonce bound to the account, the session and the purpose: session start, purchase, ranked match start or casual match start. It expires after `IntegrityNonceTtlSeconds`. Redeeming removes it, so an unknown, already used, expired or differently bound nonce fails the check.
- **Android.** Google decodes the Play Integrity token server to server (`decodeIntegrityToken`); a verdict or hash reported by the client is never read. The nonce comes from `requestDetails.nonce` (classic request) or `requestDetails.requestHash` (standard request). The check passes only if the nonce redeems; `requestDetails.requestPackageName` and `appIntegrity.packageName` equal the configured package; `requestDetails.timestampMillis` is within `IntegrityTokenMaxAgeSeconds` of server time; `appIntegrity.appRecognitionVerdict` is `PLAY_RECOGNIZED`; every `appIntegrity.certificateSha256Digest` entry is on the signing digest allowlist (a build re-signed with another key fails); the highest `deviceIntegrity.deviceRecognitionVerdict` level (basic, then device, then strong) reaches `IntegrityMinDeviceLevel`; and, when `IntegrityRequireLicensed` is on, `accountDetails.appLicensingVerdict` is `LICENSED`. A token Google rejects fails the check. An outage of the decoding service is an error, not a verdict; purchases and ranked entry then fail closed.
- **iOS attestation.** The app attests one Secure Enclave key with App Attest, using the nonce as client data. It passes only if the format is `apple-appattest`; the `x5c` chain verifies to the Apple App Attest root CA at server time; the credential certificate extension `1.2.840.113635.100.8.2` holds SHA-256(authData ‖ SHA-256(nonce)); the SHA-256 of the certificate's public key (uncompressed point) equals both the key id and the credential id; the RP ID hash equals SHA-256(`<team id>.<bundle id>`); the counter is 0; and the AAGUID is the production one (`appattest` and seven zero bytes), or the development one (`appattestdevelop`) only where the develop environment switch is on, which it never is in production. The server stores the public key, the counter and the receipt.
- **iOS assertion.** Later checks sign a fresh nonce with the attested key. An assertion passes only if the nonce redeems, the signature over SHA-256(authenticatorData ‖ SHA-256(nonce)) verifies with the stored key, the RP ID hash matches and the counter is strictly greater than the stored one.
- **Reasons.** Each failure has a stable name stored with the signal as evidence, for example `unknown_signing_certificate`, `unrecognized_app`, `token_stale`, `nonce_unknown`, `app_id_mismatch`, `environment_mismatch`, `counter_replay`.
- **Decision.** A device verdict below `IntegrityMinDeviceLevel` is a V-6 device signal handled by `DeviceRiskAction`: warn (no signal), flag (signal with `DeviceRiskWeight`) or block ranked (ranked match start blocked, other purposes flagged). Every other failure is a V-2 integrity failure with a signal of `IntegrityFailureWeight`: purchase and ranked match start are always blocked, session start and casual match start follow `IntegrityFailureAction`. The most severe action wins, and a blocked request still records its signal. An unknown `IntegrityFailureAction` is treated as block, an unknown `DeviceRiskAction` as block ranked.
- **Tightening.** `IntegrityFailureAction`, `IntegrityMinDeviceLevel`, `IntegrityRequireLicensed` and `DeviceRiskAction` are read on every check, so publishing stricter config applies to the next check without a release.

## 6. Residual risk

Complete protection is impossible. A determined attacker controls the device and can bypass any client-side check, obfuscation or detection given time. The goal is not to make cheating impossible but to make it worthless where it matters: no path may create currency, items or pass progress, change a balance, or move a rating outside legal play. The release gate checks exactly that (#212, `docs/roadmap.md`).

Accepted residual risks:

- Assistance from legal information: a tool that tracks played tiles or suggests moves uses only what the player already sees. It cannot be prevented, only partly detected as automated play (V-10).
- Collusion over an outside channel between players who meet by chance cannot be prevented, only detected statistically after the fact (V-9).
- Client-side detections (V-2, V-5, V-6) will be bypassed on some devices; they raise the cost and feed the score.
- A modified client can show anything on its own screen. That changes nothing for other players or on the server.
- Detection has false positives. That is why soft signals never ban alone, permanent bans need an admin, and every ban can be appealed.

## 7. Config

| Field | Meaning |
|---|---|
| `IllegalIntentThreshold`, `IllegalIntentWindowMinutes` | Illegal intents within the window that raise a flag |
| `MacFailureThreshold`, `MacFailureWindowMinutes` | MAC failures within the window that raise a flag |
| `ReplayRejectThreshold`, `ReplayRejectWindowMinutes` | Rejected replays within the window that raise a flag |
| `IntegrityFailureAction` | Block login or flag on failed attestation |
| `DeviceRiskAction` | Warn, block ranked or flag on a risky device |
| `IntegrityNonceTtlSeconds` | Lifetime of an attestation nonce (section 5.1) |
| `IntegrityTokenMaxAgeSeconds` | How far a Play Integrity token timestamp may be from server time |
| `IntegrityMinDeviceLevel` | Minimum Play Integrity device verdict: none, basic, device or strong |
| `IntegrityRequireLicensed` | Require the `LICENSED` Play licensing verdict |
| Package name, signing certificate digest allowlist, App Attest team id and bundle id, App Attest root CA, develop environment switch | Deployment config of #205. The root CA is Apple's public certificate; the switch is off in production |
| `IllegalIntentWeight`, `IntegrityFailureWeight`, `UnknownBuildWeight`, `HookSignalWeight`, `DeviceRiskWeight`, `MacFailureWeight`, `ReplayRejectWeight`, `ClockSkewWeight`, `FakeReceiptWeight`, `RefundWeight`, `ImpossibleBalanceWeight`, `CollusionWeight`, `AutomationWeight` | Risk score weight of each signal |
| `RiskDecayDays` | How long a signal keeps counting toward the score |
| `ReviewScore`, `ShadowbanScore`, `TempBanScore` | Risk score at which each response starts |
| `TempBanHours` | Temporary ban length |
| `BanWaveIntervalHours` | Interval between ban waves |
| Minimum version, build allowlist, purchase and ranked kill switches | Managed in the admin panel (#210) |

Values are set in the admin panel before launch. Every config change is audited.
