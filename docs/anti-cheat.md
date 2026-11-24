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
- **NT-4** Hidden information never leaves the server: during a hand a client receives only the tiles its player holds and the public board (R-4). Once a hand has ended, the tiles left in each hand are shown for the hand summary (#83); the boneyard never is. Deal order, boneyard contents and random seeds stay on the server.
- **NT-5** A purchase is granted only after server-side receipt validation with the store. The store transaction id is unique, and the receipt is bound to the buying account (ADR 0005, #106, #204).
- **NT-6** Every time-based decision uses server UTC: turn timers (R-49, R-50), daily rewards, offers, pass windows, cooldowns, season boundaries. The device clock is display only (#208).
- **NT-7** Everything the client says about itself is a claim: app version, build hash, signature hash, device model, root or emulator status. Trust comes only from attestation tokens verified on the server, and even those are risk signals, never proof (#205, #185).
- **NT-8** A client-side detection (hook, debugger, root, emulator) only raises the risk score. It never replaces a server check and is never the sole basis of a permanent ban.
- **NT-9** The cosmetic loadout is checked against server inventory and frozen on the server at match start. Other players see the server's loadout, not what a client claims (#211).
- **NT-10** The client holds no secret that grants anything: no server keys, no admin tokens, no reward or receipt signing keys. Anything shipped in the client is public (#206).
- **NT-11** Every economy write goes through the idempotent ledger with a reason and a source (#101, #202). No code path changes a balance outside it.
- **NT-12** Intents, logins, promo redemptions and purchases are rate limited and counted per account (ADR 0007, #184).

HTTP requests of the game server pass a rate limiter before any handler (#184). Each request takes a token from its client IP bucket; a request with a valid access token also takes one from its account bucket. `POST /auth/signin` and `POST /auth/refresh` take one more from a sign-in bucket per IP; `POST /shop/buy`, `POST /shop/begin`, `POST /shop/complete`, `POST /promo/redeem` and `POST /pass/levels` take one more from a spend bucket per account. An empty bucket answers 429 `rate_limited` with `Retry-After`, adds one to the `rateLimited` counter of `/loadz` and runs no handler. Buckets live in Redis when `HD_REDIS_URL` is set, so every node shares them, and in node memory otherwise; when Redis fails the request passes and a warning is logged, so an outage does not lock out players. Health probes and the store notification endpoints, which the stores call from their own servers, are not limited. The client IP is the connection address, or the header named by `HD_CLIENT_IP_HEADER` when the edge overwrites it on every request. The server closes a connection that sends its headers and body slower than `HD_READ_TIMEOUT` or idles longer than `HD_IDLE_TIMEOUT`; a WebSocket leaves these deadlines at upgrade and keeps its own frame size, intent rate and ping limits (`proto/README.md`, Gateway limits). The edge WAF and CDN layer is deployment config (`docs/launch-runbook.md` GN-16).

NT-1 is checked automatically (#202). `tools/proto-lint` refuses a value field (balance, coin, crystal, price, cost, amount, reward, score, points, rating, MMR, LP, XP, rank, tier, inventory, owned, result) in a client message of `proto/`. `server/internal/contract` scans every server handler that takes an `*http.Request` and fails the build when a JSON body it decodes has a field with one of those words, nested structs included. The admin API is left out: staff set prices there under RBAC and audit (`docs/admin.md`). A fuzz test sends tampered `POST /shop/buy` bodies (extra price, coin, balance and reward fields, wrong types, duplicate keys) and checks that no balance rises, that a fall is a catalog price and that the ledger audit stays clean.

### Device storage (NT-3, #203)

The client keeps only device settings and content bookkeeping on the device. Every PlayerPrefs key the client uses is listed here; `RuntimeCodeRulesTests` in the client fails when runtime code uses a key that is not in this table, when a key here is no longer used, when a key names a server value (the word list of `tools/proto-lint`), or when runtime code outside the files named here writes a file. Content bundles are written only by Addressables into its cache. Three runtime files write files themselves: `Content/AddressablesCatalogSource.cs` moves each downloaded content catalog into its version folder (`docs/content-delivery.md`), `Content/UnityStringBundleSource.cs` keeps the last downloaded language bundle of each language in `content/strings/<language>/<version>_<hash>.json` (`docs/localization.md` 2.4), and `Content/UnityCosmeticManifestSource.cs` keeps the last downloaded cosmetics manifest in `content/cosmetics/<version>_<hash>.json` (`docs/cosmetics.md` 7.3). A catalog lists art and its addresses, never what an account owns; a language bundle holds display text only; the cosmetics manifest describes what exists, never ownership, price or chance (`docs/cosmetics.md` 10).

| Key | Holds | Why editing it gains nothing |
|---|---|---|
| `content.catalogVersion`, `content.baseUrl` | The last content catalog the server announced and where it is served | It picks which art to download; what an account owns and shows to others is server state (NT-9) |
| `content.clearAtStart` | Clear the content cache at the next start | Cache housekeeping only |
| `graphics.choice`, `graphics.frameRate`, `graphics.batterySaver` | Graphics level, frame rate and battery saver chosen in settings | Device settings |
| `graphics.probeDrops`, `graphics.probeSettled`, `graphics.probeVersion` | The startup frame probe result for this app version | Device settings |
| `motion.reduce` | Reduce motion chosen in settings, or follow the system (`docs/ui-motion.md`, Reduce motion) | Device settings |
| `haptics.enabled` | Haptics turned on or off in settings (`docs/ui-motion.md`, Haptics) | Device settings |
| `audio.musicVolume`, `audio.sfxVolume`, `audio.uiVolume` | Volume level of each sound group chosen in settings (`docs/ui-motion.md`, Sound) | Device settings |
| `tutorial.finished` | This device finished the tutorial, skipped or completed, so it is not offered on its own again (`docs/tutorial.md` TU-7) | It only decides whether the tutorial is offered (TU-9); the tutorial grants nothing (TU-8) |
| `language.choice` | Game language, picked from the device locales at first launch or chosen in settings (`docs/localization.md` 1.1) | Display setting only; the server never trusts it for game logic |
| `content/strings/<language>/<version>_<hash>.json` (file) | The last downloaded language bundle of the selected language and `en` (`docs/localization.md` 2.4) | Display text only; it is refused unless its SHA-256 matches the hash in its name, the next hello replaces it when the server announces another hash, and the server never reads it |
| `content/cosmetics/<version>_<hash>.json` (file) | The last downloaded cosmetics manifest (`docs/cosmetics.md` 7.3) | It lists items and their art only; it is refused unless its SHA-256 matches the hash in its name, ownership and equip are server checks (`docs/cosmetics.md` 10), and the server never reads it |

- The client has no offline mode and grants nothing on its own: rewards, balances, inventory, scores and ratings come only from server messages and are never written to the device.
- A new key, file or cache is added to this table in the same pull request, with why editing it gains nothing. A display cache of server values (for example the last balance shown) may only be shown until the server answers and is replaced by it; it never feeds an intent.

## 3. Responses

| Response | What happens | Who triggers it |
|---|---|---|
| Block | The request, intent or session is rejected and server state is unchanged. A counter is incremented | Server, immediately |
| Flag | A risk signal with its evidence is recorded and the account's risk score rises | Server, immediately |
| Shadowban | Ranked matchmaking pairs the account only with other shadowbanned accounts (cheater pool). It is hidden from public leaderboards. The player is not notified; a data export or access request shows it (`docs/privacy.md` 8.3) | Risk score at `ShadowbanScore`, applied in the next ban wave, or an admin |
| Temporary ban | No login for `TempBanHours` | Risk score at `TempBanScore`, applied in the next ban wave, or an admin. An account under 18 is sent to review instead of a score-based ban (`docs/privacy.md` 12.2, OQ-9) |
| Permanent ban | No login, permanently | An admin from the review queue only |
| Ranked ban | No ranked queue and no custom rooms for `RankedBanHours`; login and casual matches stay open | An admin (`docs/admin.md` 6.2) |
| Review | The account enters the admin review queue with all its signals and evidence; the admin decides and the player can appeal | Risk score at `ReviewScore`, any hard-evidence signal, or a penalty ladder review step (`docs/penalties.md`) |

- Blocks are immediate because they protect state. Shadowbans and temporary bans from the risk score are applied in ban waves every `BanWaveIntervalHours`, so an attacker cannot tie a ban to the action that caused it (#209).
- The risk score is the sum of signal weights with decay over `RiskDecayDays` (section 5). It is separate from the AFK and leaver offense level in `docs/penalties.md`; both feed the same admin review queue.
- Every response and every admin change is audited with a mandatory reason (#209).
- Every ban carries one ban reason category (`docs/admin.md` 6.2): `cheating`, `collusion`, `afk_leaving`, `offensive_name`, `payment_fraud` or `other`. The refusals the player gets name only that category: sign-in 403 `banned` with `category`, and `ban_reason` on the `MATCHMAKING_BAN` and `RANKED_BAN` queue refusals and the `RANKED_BAN` lobby refusal. Signals, scores and evidence are never sent.
- Owner-confirmed on #201 (2026-10-26): permanent bans only by an admin from review, soft signals never ban alone, shadowbanned accounts hidden from public leaderboards, every hard signal goes to review.

## 4. Attack vectors

Risk is the impact if the countermeasures did not exist. Every vector is verified by the red team before release (#212, `docs/red-team.md`).

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

Implemented in `server/internal/purchase` with `server/internal/risk` (#204). A `fake_receipt` signal of weight `FakeReceiptWeight` (20) is recorded in `risk_signals` when a completed purchase is refused for an account mismatch, a transaction already bound to another account, a bad store signature (`receipt.signature`), a token the store does not know (`receipt.unknown`) or a sandbox or test receipt where they are not allowed (`receipt.environment`). Evidence holds the platform, offer, refusal code and, when known, the transaction id; the dedupe key is the transaction id or a SHA-256 of the token, so resending one receipt counts once. A refund or chargeback notice that revokes a purchase records a `refund` signal of weight `RefundWeight` (3), once per transaction, in the same transaction as the revoke. A reversal does not remove the signal. Every refusal is also logged with the account and the code.

### V-4. Packet tampering through a proxy

| | |
|---|---|
| Attack | An intercepting proxy with a user-installed certificate reads and edits WebSocket and HTTPS traffic: changes amounts, injects intents, looks for hidden data |
| Risk | Medium: the server already decides everything, but readable traffic speeds up every other attack |
| Countermeasure | TLS with certificate pinning and a rotation plan (a backup pin shipped in every build) (#207). Edited amounts are ignored because the server never reads them (NT-1, #202). Every move is re-checked in the rules engine (NT-2). Hidden tiles are never sent (R-4), so reading traffic reveals nothing the player does not already see. Critical operations (purchase, reward claim, equip) carry a message authentication code with the session key (#207) |
| Detection | Pinning failure reported at the next good connection, MAC failures, malformed frames, intents inconsistent with state |
| Response | Block the frame. Flag when MAC failures exceed `MacFailureThreshold` in `MacFailureWindowMinutes` |
| Issues | #207, #202, #212 |

Request MACs are implemented in `server/internal/token` and `server/internal/risk` (#207). Sign-in and refresh return a `request_key` next to the access token: 32 bytes, standard base64, derived on the server as HMAC-SHA256 with the key that signed the access token over `hd-request-mac`, a zero byte and the session id. It is never stored and stays the same for the session until the signing key rotates; the next refresh then returns the new one. Every POST, PUT, PATCH or DELETE to `/shop`, `/shop/complete`, `/inventory`, `/daily-login`, `/pass`, `/medals`, `/missions` and `/promo/redeem` carries `X-HD-Mac`: unpadded base64url of HMAC-SHA256 with the request key over the method, a newline, the request path with its query, a newline, the access token, a newline and the raw body. A request with a valid access token and a missing or wrong MAC is answered 401 `bad_mac` before the handler runs; a request without a valid access token goes on to the handler, which refuses it as before. Each failure counts for the account; `MacFailureThreshold` (5) failures within `MacFailureWindowMinutes` (10) record a `mac_failure` signal of weight `MacFailureWeight` (5) with the count and window as evidence, at most once per window. Counts live in node memory. Binding the access token limits a captured request to the token lifetime; repeating it inside that time gains nothing because the writes are idempotent (V-8). The client signs every write it sends, on any route (`Net/Api/ApiClient`), and keeps the request key in memory only.

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

- Implemented in `server/internal/risk` (#208): `ClientHello.client_time_unix_ms` carries the device clock at connect. When it is set and differs from server time by more than `ClockSkewToleranceSeconds` (300), the server records a `clock_skew` signal of weight `ClockSkewWeight` (1) in `risk_signals` with the client time, the server time and the skew as evidence, at most once per account per UTC day, so reconnects do not stack it. The signal never blocks the connection and nothing the server decides reads the device clock.

### V-8. Replay

| | |
|---|---|
| Attack | Captured frames or requests are sent again: a reward claim, a purchase, a move, or a whole recorded session |
| Risk | High: replayed claims and purchases mint currency if writes are not idempotent |
| Countermeasure | Every frame carries the session's sequence number (ADR 0004) and a session nonce; a frame from another session or with an old nonce is rejected (#207). A repeated sequence number inside the session is answered idempotently with the earlier result, so reconnect retries stay safe. Reward claims and ledger writes are idempotent per server period and source (#101). A receipt grants once (V-3) |
| Detection | Frames with foreign or expired nonces, out-of-window sequence numbers, duplicate claim and transaction ids |
| Response | Block. Flag when rejected replays exceed `ReplayRejectThreshold` in `ReplayRejectWindowMinutes` |
| Issues | #207, #202, #204, #212 |

Implemented in `server/internal/gateway` and `server/internal/risk` (#207). An accepted `ServerHello` carries a `session_nonce` the server draws from a secure random source for that connection; every later client frame must carry it in `Envelope.session_nonce` (`proto/README.md`). A frame with another or no nonce is answered `Error REPLAY_REJECTED` and dropped; the connection stays open. Each rejection counts for the account; `ReplayRejectThreshold` (5) rejections within `ReplayRejectWindowMinutes` (10) record a `replay_reject` signal of weight `ReplayRejectWeight` (5) with the count and window as evidence, at most once per window. A repeated sequence number inside the session is not a replay: the room drops it silently, so reconnect retries stay safe. Counts live in node memory; a reconnect to another node starts a fresh count.

### V-9. Collusion and multi-account farming

| | |
|---|---|
| Attack | Players share hidden hands through an outside channel in 2v2, trade wins, or run several accounts to farm rewards and rank |
| Risk | Medium: random partners in ranked 2v2 (R-54) make arranged teams hard, but win trading and farming accounts still distort ratings and the economy |
| Countermeasure | Partners are assigned at random and cannot be chosen (R-54). Custom rooms are unranked (R-55). Rating abuse rules (#121) and farm signals (#110) feed the risk score |
| Detection | The same accounts meeting more often than matchmaking explains, play that matches the partner's hidden tiles more often than chance, repeated losses to the same opponent, accounts sharing devices or networks (#183, #110) |
| Response | Flag and review. Confirmed collusion: shadowban or temporary ban, rating changes from the affected matches reverted by an admin |
| Issues | #209, #212; detection details in #183 (Epic 16), #121 (Epic 10), #110 (Epic 9) |

- Implemented in `server/internal/rating` (#121): the repeat limit of the same pair in ranked matchmaking and the `pair_frequency`, `feeding` and `win_trade` pair signals of `docs/ranking.md` section 11, stored in `rating_signals` and published on the `rating_signals` outbox topic with `CollusionWeight` applied by the risk score of #209. The partner signals `partner_frequency` and `partner_fit` of the same section (#183) watch teammates in 2v2: the first counts how often two accounts were drawn as partners, the second compares how often a player's placements suit the partner's hidden tiles with that partner and with others. Both are stored and published the same way; the matches behind a signal are the ranked 2v2 matches of the pair in the window, which the admin opens in the replay view (D-13) for review.
- Collusion reaches the risk score in the same transaction (#183): a `win_trade`, `feeding` or `partner_fit` signal records a `collusion` risk signal of `CollusionWeight` (launch default 10, owner decision 2026-11-08) on each account of the pair, with the pair signal, counterpart, mode, matches and wins as evidence. It is recorded at most once per UTC day for the same pair signal, counterpart and mode. `pair_frequency` and `partner_frequency` only show that two accounts meet often and stay out of the score. `HD_COLLUSION_WEIGHT` 0 turns it off.
- Farm accounts reach the risk score through the daily economy monitor (#110, `docs/economy.md` 10): an account whose `match_reward`, `first_win`, `daily_login` and `mission` coins of one economy day exceed `AccountDailyEarnAlertCoins` gets one `farm` signal of `CollusionWeight` for that day. Device and IP links between accounts are not signals yet: no threshold for shared devices is decided.
- The admin player record lists the account's pair signals and marks each match played with the counterpart of a `win_trade`, `feeding` or `partner_fit` signal inside that signal's window, with a link to its replay (#183). `GET /admin/matches/suspicious` (players view permission) lists the same marked matches across all accounts, newest first and 50 a page, filtered by signal and searched by account id (`docs/admin.md`), each with its signal kinds, the accounts of the flagged pairs and a link to its replay; the admin opens it as Suspicious matches.

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

Unowned equips are flagged in `server/internal/risk`, `server/internal/inventory` and `server/internal/medal` (#211). An equip (`PUT /inventory/equip`) or a medal showcase (`PUT /medals/showcase`) refused because the account does not own the item counts for the account; `UnownedEquipThreshold` (5) refusals within `UnownedEquipWindowMinutes` (10) record an `unowned_equip` signal of weight `UnownedEquipWeight` (5) with the count and window as evidence, at most once per window. Counts live in node memory. The refusal itself does not change.

The matchmade room roster takes each player's loadout from `inventory.Store.Freeze` when the room opens (`server/internal/matchstart`, #211): an item the account no longer owns falls back to its default, so other players see and the match record keeps only owned cosmetics.

Promo locks and spikes are implemented in `server/internal/promo` (#211, `docs/economy.md` 8): a lock records a `promo_lock` signal of weight `PromoLockSignalWeight` (5), and a `multi_use` code whose redemptions within `PromoSpikeWindowMinutes` (60) reach `PromoSpikePercent` (50) of its cap is marked `spike_at` once and logged, so an admin can disable it.

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
| Equip or showcase of an unowned item over threshold | #211 | Soft | `UnownedEquipWeight` |
| Promo redemption lock | #211 | Soft | `PromoLockSignalWeight` |
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

- **Nonce.** The server issues a random, single-use nonce bound to the account, the session and the purpose: session start, purchase, ranked match start or casual match start. It expires after `IntegrityNonceTtlSeconds`. Redeeming removes it, so an unknown, already used, expired or differently bound nonce fails the check. The client asks for one with `POST /integrity/nonce` `{"purpose"}` (`session_start`, `purchase`, `ranked_match` or `casual_match`) and its access token, which supplies the account and session, and gets `{"nonce","expires_in"}`. Nonces live in Redis with the TTL, so any node can redeem them; a node without Redis keeps them in memory. On the client `Net/Integrity/IntegrityProofs` asks for a nonce per purpose (a ranked queue `ranked_match`, any other queue `casual_match`) and hands it to the device attestor (`IDeviceAttestor`, Play Integrity or App Attest), which returns the `IntegrityProof`. A device without an attestor, or a nonce the server did not issue, gives no proof: the client then sends none and the server's policy above decides. `Net/Integrity/SessionStartProof` asks for a `session_start` proof before a new connection dials and puts it in `ClientHello.integrity`; redials before the hello is accepted resend it, and reconnects after it carry none. When a hello without a proof is answered `INTEGRITY_FAILED` (the server no longer knows the session, for example after a restart), it asks for one proof and dials again only if it got one.
- **Android.** Google decodes the Play Integrity token server to server (`decodeIntegrityToken`); a verdict or hash reported by the client is never read. The nonce comes from `requestDetails.nonce` (classic request) or `requestDetails.requestHash` (standard request). The check passes only if the nonce redeems; `requestDetails.requestPackageName` and `appIntegrity.packageName` equal the configured package; `requestDetails.timestampMillis` is within `IntegrityTokenMaxAgeSeconds` of server time; `appIntegrity.appRecognitionVerdict` is `PLAY_RECOGNIZED`; every `appIntegrity.certificateSha256Digest` entry is on the signing digest allowlist (a build re-signed with another key fails); the highest `deviceIntegrity.deviceRecognitionVerdict` level (basic, then device, then strong) reaches `IntegrityMinDeviceLevel`; and, when `IntegrityRequireLicensed` is on, `accountDetails.appLicensingVerdict` is `LICENSED`. A token Google rejects fails the check. An outage of the decoding service is an error, not a verdict; purchases and ranked entry then fail closed. `integrity.PlayDecoder` calls `v1/{package}:decodeIntegrityToken` with a token for the Play service account (`HD_PLAY_SERVICE_ACCOUNT`, scope `playintegrity`); an answer of 400 means Google refused the token, any other failure is an outage.
- **iOS attestation.** The app attests one Secure Enclave key with App Attest, using the nonce as client data. It passes only if the format is `apple-appattest`; the `x5c` chain verifies to the Apple App Attest root CA at server time; the credential certificate extension `1.2.840.113635.100.8.2` holds SHA-256(authData ‖ SHA-256(nonce)); the SHA-256 of the certificate's public key (uncompressed point) equals both the key id and the credential id; the RP ID hash equals SHA-256(`<team id>.<bundle id>`); the counter is 0; and the AAGUID is the production one (`appattest` and seven zero bytes), or the development one (`appattestdevelop`) only where the develop environment switch is on, which it never is in production. The server stores the public key, the counter and the receipt. The client sends `POST /integrity/attest` `{"nonce","key_id","attestation"}` (key id and attestation in standard base64) with a `session_start` nonce; a verified key is stored in `app_attest_keys` for the account (migration 00032) and answered 204, a failed check 403 `attestation_failed` (the reasons are logged, not returned), a key id already stored 409 `key_exists`.
- **iOS assertion.** Later checks sign a fresh nonce with the attested key. An assertion passes only if the nonce redeems, the signature over SHA-256(authenticatorData ‖ SHA-256(nonce)) verifies with the stored key, the RP ID hash matches and the counter is strictly greater than the stored one.
- **Reasons.** Each failure has a stable name stored with the signal as evidence, for example `unknown_signing_certificate`, `unrecognized_app`, `token_stale`, `nonce_unknown`, `app_id_mismatch`, `environment_mismatch`, `counter_replay`.
- **Decision.** A device verdict below `IntegrityMinDeviceLevel` is a V-6 device signal handled by `DeviceRiskAction`: warn (no signal), flag (signal with `DeviceRiskWeight`) or block ranked (ranked match start blocked, other purposes flagged). Every other failure is a V-2 integrity failure with a signal of `IntegrityFailureWeight`: purchase and ranked match start are always blocked, session start and casual match start follow `IntegrityFailureAction`. The most severe action wins, and a blocked request still records its signal. An unknown `IntegrityFailureAction` is treated as block, an unknown `DeviceRiskAction` as block ranked.
- **Tightening.** `IntegrityFailureAction`, `IntegrityMinDeviceLevel`, `IntegrityRequireLicensed` and `DeviceRiskAction` are read on every check, so publishing stricter config applies to the next check without a release. Until they are admin config they come from `HD_INTEGRITY_FAILURE_ACTION`, `HD_INTEGRITY_MIN_DEVICE_LEVEL`, `HD_INTEGRITY_REQUIRE_LICENSED` and `HD_DEVICE_RISK_ACTION`, so tightening needs a restart, not a release.
- **Session start.** A new session (not a reconnect to a known session id) is checked from `ClientHello.integrity` with a `session_start` nonce before the hello is accepted. It follows `IntegrityFailureAction` and `DeviceRiskAction`; a block is answered `Error INTEGRITY_FAILED` and the connection closes. A decoding outage lets the session start: the spec fails closed only for purchases and ranked entry. The hello may be up to `MaxHelloBytes` (8 KiB) so a Play Integrity token fits; later frames keep `MaxFrameBytes`, except `JoinQueueIntent` (match start below).
- **Match start.** `JoinQueueIntent.integrity` carries the same proof as the hello for a `ranked_match` nonce (ranked queue) or a `casual_match` nonce (any other queue). The session checks it before the entry reaches matchmaking, once a session is not already in a match. A blocked entry is answered `QueueState REFUSED` with `QUEUE_REFUSAL_INTEGRITY_FAILED` (reasons logged, not returned). A decoding outage refuses a ranked entry with `QUEUE_REFUSAL_INTEGRITY_UNAVAILABLE` and lets a casual entry through. A `JoinQueueIntent` may be up to `MaxHelloBytes` so a Play Integrity token fits; every other frame keeps `MaxFrameBytes`.
- **Purchase gate.** `POST /shop/complete` runs the check with a `purchase` nonce before the receipt is read. Android sends the Play Integrity token in `X-HD-Play-Integrity`; iOS sends `X-HD-Attest-Key` (key id, standard base64), `X-HD-Attest-Nonce` and `X-HD-Attest-Assertion` (standard base64) for a key attested earlier, and the stored counter moves forward only. No proof, or a proof for a platform the server is not configured for, is `token_rejected`. A blocked purchase is answered 403 `integrity_failed` (reasons logged, not returned), a decoding outage 503 `integrity_unavailable`. On the client `Game/Shop/ShopPurchase` asks `IntegrityProofs` for a `purchase` proof before every `/shop/complete`, a retry included, and `Net/Integrity/IntegrityHeaders` turns it into these headers; without a proof it sends none. Every signal is stored in `risk_signals` as `integrity_failure` or `device_risk` with the purpose and reason names as evidence, once per purpose per UTC day. The gate is on when Play Integrity (`HD_PLAY_CERT_DIGESTS`) or App Attest (`HD_APP_ATTEST_TEAM_ID`) is configured; production must configure both.

### 5.2 Live controls (#210)

`server/internal/livecontrol` holds the live controls of `docs/admin.md` section 4 that this document names, as one config snapshot of domain `live_controls`: `minimum_app_version`, `build_allowlist_enforced`, `build_allowlist`, `purchases_disabled`, `ranked_disabled` and `maintenance_ends_at` (maintenance mode, `docs/admin.md` section 4, #146). Unknown fields, an invalid version and an enforced empty allowlist are refused. Nodes read the snapshot at start and refresh it every `HD_CONFIG_POLL` (default 10 s); a node that cannot read a published snapshot at start does not start. Until the first publish `HD_MIN_APP_VERSION` is the minimum version and nothing else is restricted.

- **Hello.** A client below the minimum version, or with a `build_hash` off an enforced allowlist, gets `UPDATE_REQUIRED` and is closed before its session token is read.
- **Connected clients.** When a node applies a new snapshot it checks every open connection against it the same way. A client that no longer passes gets `UPDATE_REQUIRED` and is closed, also in a match; the match treats it as a disconnect (`docs/rules.md`). So publish a raised minimum version only to cut a build off, or when older clients must stop now.
- **Purchase kill switch.** With `purchases_disabled`, `POST /shop/begin` is answered 503 `purchases.disabled`. Coin and crystal offers stay open, and `POST /shop/complete` and the store notifications still grant validated receipts (`docs/economy.md` 7.5).
- **Ranked kill switch.** `ranked_disabled` is stored; the matchmaking entry reads it once matchmaking is wired to the network.
- **Maintenance mode.** While `maintenance_ends_at` is set, a hello whose session has no running match gets `MAINTENANCE` with `maintenance_ends_unix_ms` and is closed after its session token is read, and a node applying the snapshot does the same to every open connection outside a match. A player in a match keeps playing and can reconnect to it; once it ends, the next hello is turned away. With no connection left outside a match, no new match starts. The end time is shown, not enforced: maintenance stays on until the field is cleared.

`GET /admin/live-controls` shows the controls to every role with view rights. `PUT /admin/live-controls` `{"controls","reason","ref"}` needs the `switch` permission, a reason and a recent sign-in (`docs/admin.md` 9). In staging it publishes at once. In production it answers 202 with `approval_id`, and the switch happens when a second admin approves (`docs/admin.md` 8). During a declared SEV1 or SEV2 incident, `break_glass` `{"severity","id"}` switches at once and a second admin confirms afterwards; `GET /admin/live-controls/break-glass` lists the unconfirmed ones (`docs/admin.md` 8).

`build_hash` is reported by the client. The allowlist stops builds we distributed, which report their real hash; a re-signed build that lies about it is stopped by the signing digest of section 5.1.

### 5.3 Risk score and ban waves (#209)

The score of an account is the sum of the weights of its signals from the last `RiskDecayDays`. Values are owner decisions of 2026-11-05.

- **Hard signals.** A `fake_receipt` signal opens a review in the same write, whatever the score. The ledger mismatch signal joins it when `cmd/wallet-audit` records one.
- **Ban wave.** Every `BanWaveIntervalHours` one node wins the interval's row in `risk_waves` and runs the wave. Slots are fixed UTC multiples of the interval. A node down during a wave loses nothing: the next wave reads the same signals.
- **Review.** A score at `ReviewScore` opens a review with cause `score` at the next wave. An account has at most one open review.
- **Shadowban.** A score at `ShadowbanScore` also sets `accounts.shadowbanned_at` and stores a `risk_actions` row with the score and wave. Soft signals may reach this step alone, because a shadowban does not block login (section 3, `docs/privacy.md` OQ-9).
- **Temporary ban.** The wave applies none yet. A score-based temporary ban needs a hard signal among the counted signals (owner decision, 2026-11-05) and must skip accounts under 18 (section 3), and the server does not store the age gate result yet. Until it does, such an account stays in review, where an admin can ban it.

Wave actions have no admin actor, so `risk_actions` is their record rather than the admin audit log. A shadowbanned account leaves the live boards at the wave, and a later rating change does not bring it back. At season end it takes the last positions of the frozen snapshot, so nobody is pushed down, stays off the frozen board and earns no emblem; its rank reward is still granted, so the sanction stays hidden (`docs/ranking.md` 8.3). A ranked queue entry reads the shadowban when it joins (`rating.Store.Queue`), and matchmaking forms shadowbanned and other entries in separate buckets, so a shadowbanned account meets only shadowbanned accounts in ranked, as opponent and as 2v2 partner. Ranked seats no bots, so it waits until another shadowbanned account queues. Casual entries are not split. An entry already in the queue keeps the state it joined with. A shadowban is appealed only through support; temporary, permanent, ranked and penalty ladder bans and forced renames are appealed with `POST /appeals` and decided in the appeal queue (`docs/admin.md` 6.4, Appeals). The appeal token a ban answer carries is valid for that one appeal only and never as an access token; the appeal text is the player's own words and is shown only to queue roles.

`GET /admin/reviews` lists open reviews, oldest first, 50 a page (at most 200 through `limit`), with each account's signals since its last decision, its earlier decisions, and its `due_at` (`opened_at` plus `ReviewSlaHours`, default 24) and `overdue` flag from the server clock (`docs/admin.md` 6.4); `cause` keeps one cause, `account` is an account id prefix search and `after=<review id>` asks for the next page (#148). `POST /admin/reviews/{id}/clear` and `/confirm` take a reason and a recent sign-in, and are audited on the `queues` object. Either decision closes the review, and the signals it covered stop counting, so only a new signal can bring the account back. Clear also lifts the shadowban; confirm keeps it. A cleared account returns to the boards at its next rating change or board rebuild. An admin who wants a temporary ban uses the player action (`docs/admin.md` 6.2), then confirms. `POST /admin/reviews/{id}/permanent-ban` `{"reason","ref","category"}` proposes a permanent ban on the review's account with a ban reason category (`docs/admin.md` section 8 and 6.2; 400 `invalid_category` without one). The review stays open until a second admin approves; the approval confirms the review, sets the account state to `banned`, records a `permanent_ban` sanction with that category (`other` for a proposal made before categories), revokes its sessions and is refused with `decided` when the review was closed meanwhile. A permanent ban is lifted only by lifting an appeal of it in the appeal queue, which sets the account back to `active`. `POST` and `DELETE /admin/players/{id}/shadowban` set and lift a shadowban by hand (`docs/admin.md` 6.2), with a reason and a recent sign-in, audited on the player; a manual shadowban leaves the boards at once when the admin API has Redis, and a lifted one returns like a cleared review.

### 5.4 Client device signals (#185)

The client collects root, jailbreak and emulator signals and reports their names; it never acts on them (NT-7, NT-8). `client/Assets/HypeDomino/Core/DeviceRisk/DeviceRiskSignals` names them: `root` (a `su` or Magisk binary or a superuser app on a known path), `test_keys` (Android build tags `test-keys`), `emulator` (emulator markers in the Android build fingerprint, model, product, hardware or manufacturer) and `jailbreak` (a jailbreak app, package manager or substrate path on iOS). `Game/DeviceRiskProbe` reads the Android `Build` fields and the paths on the device. The signals are claims: the server weighs them with `DeviceRiskWeight` and `DeviceRiskAction` like the attestation device verdict of 5.1. The client fills `Net/Session.DeviceRiskSignals` once from `DeviceRiskProbe.Collect`, and every `ClientHello` carries the names in `device_risk_signals`. The server (`server/internal/integrity/devicereport.go`) keeps only these four names, each once, and ignores any other name, so the hello size limit of 5.1 bounds the field. A claim only raises risk and never blocks by itself: with `DeviceRiskAction` warn nothing is recorded; with flag or block ranked one `device_risk` signal of `DeviceRiskWeight` is stored in `risk_signals` with purpose `client_report` and the names as evidence, once per account per UTC day, so reconnects add nothing. The false positive rate is measured on the device matrix (#191).

Hook signals (#206, V-5) work the same way. `Core/DeviceRisk/HookSignals` reads the libraries mapped into the process (`/proc/self/maps` on Android, read by `DeviceRiskProbe.CollectHooks`) and names `instrumentation` (a Frida agent or gadget), `hook_framework` (Xposed, LSPosed, Riru, EdXposed, Substrate, SandHook or Pine) and `debugger` (a managed or Android debugger attached). The client collects `Net/Session.HookSignals` again from `DeviceRiskProbe.CollectHooks` before each `ClientHello` (`Session.CollectHookSignals`), once per dial and never per frame, so a debugger attached after start is reported on the next connection; every `ClientHello` carries the names in `hook_signals`. The server keeps only these three names, each once, and ignores any other name. A hook signal is never silenced by `DeviceRiskAction` and never blocks: one `hook_signal` signal of `HookSignalWeight` is stored in `risk_signals` with purpose `client_report` and the names as evidence, once per account per UTC day. `HookSignalWeight` has no launch default; until it is decided it is 0 (`HD_HOOK_SIGNAL_WEIGHT`), so the evidence is kept for review without moving the score. Keys and secrets stay out of the client (NT-10); the `secret-scan` CI job runs gitleaks over every pull request.

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
| `MacFailureThreshold`, `MacFailureWindowMinutes` | MAC failures within the window that raise a flag (launch defaults 5 and 10) |
| `ReplayRejectThreshold`, `ReplayRejectWindowMinutes` | Rejected replays within the window that raise a flag (launch defaults 5 and 10) |
| `UnownedEquipThreshold`, `UnownedEquipWindowMinutes` | Refused equips or showcases of unowned items within the window that raise a flag (launch defaults 5 and 10) |
| `IntegrityFailureAction` | Block login or flag on failed attestation (launch default flag; purchase and ranked match start block regardless) |
| `DeviceRiskAction` | Warn, block ranked or flag on a risky device (launch default flag) |
| `IntegrityNonceTtlSeconds` | Lifetime of an attestation nonce (section 5.1, launch default 300) |
| `IntegrityTokenMaxAgeSeconds` | How far a Play Integrity token timestamp may be from server time (launch default 120) |
| `IntegrityMinDeviceLevel` | Minimum Play Integrity device verdict: none, basic, device or strong (launch default device) |
| `IntegrityRequireLicensed` | Require the `LICENSED` Play licensing verdict (launch default off) |
| Package name, signing certificate digest allowlist, App Attest team id and bundle id, App Attest root CA, develop environment switch | Deployment config of #205. The root CA is Apple's public certificate; the switch is off in production |
| `IllegalIntentWeight`, `IntegrityFailureWeight` (launch default 20), `UnknownBuildWeight`, `HookSignalWeight`, `DeviceRiskWeight` (launch default 5), `MacFailureWeight` (launch default 5), `UnownedEquipWeight` (launch default 5), `PromoLockSignalWeight` (launch default 5, `docs/economy.md` 8), `ReplayRejectWeight` (launch default 5), `ClockSkewWeight` (launch default 1), `FakeReceiptWeight` (launch default 20), `RefundWeight` (launch default 3), `ImpossibleBalanceWeight`, `CollusionWeight` (launch default 10), `AutomationWeight` | Risk score weight of each signal |
| `ClockSkewToleranceSeconds` | How far the device clock may be from server time before a `clock_skew` signal (launch default 300) |
| `HD_RATE_IP_PER_MINUTE`, `HD_RATE_IP_BURST`, `HD_RATE_ACCOUNT_PER_MINUTE`, `HD_RATE_ACCOUNT_BURST`, `HD_RATE_SIGNIN_PER_MINUTE`, `HD_RATE_SIGNIN_BURST`, `HD_RATE_SPEND_PER_MINUTE`, `HD_RATE_SPEND_BURST` | Refill per minute and bucket size of the NT-12 limits. Launch: IP 60 and 20, account 30 and 10, sign-in 5 and 5, spend 5 and 5 |
| `HD_READ_TIMEOUT`, `HD_IDLE_TIMEOUT` | Slow client cut-offs of NT-12. Launch: 15s and 60s |
| `RiskDecayDays` | How long a signal keeps counting toward the score (launch default 30) |
| `ReviewScore`, `ShadowbanScore`, `TempBanScore` | Risk score at which each response starts (launch defaults 20, 40 and 60) |
| `TempBanHours` | Temporary ban length |
| `BanWaveIntervalHours` | Interval between ban waves (launch default 24) |
| Minimum version, build allowlist, purchase and ranked kill switches | Managed in the admin panel (#210) |

Values are set in the admin panel before launch. Every config change is audited.
