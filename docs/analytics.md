# Analytics events

First-party product analytics (#181). Privacy rules are `docs/privacy.md` section 6.4 and data item D-24; this document lists the events and how the client sends them.

## 1. Rules

- Nothing is tracked, queued or written to disk before the `analytics` consent purpose is on (`docs/privacy.md` 6.2). An event tracked before consent is dropped, not held.
- Withdrawing consent drops every queued event at once and stops sending; the caller rotates the analytics id.
- An event carries the analytics id, its name, one optional label, one optional integer value and the device time. Never an account id, email, nickname, IP, install id or free text.
- Events go only to the game's own server. No third-party analytics SDK ships.
- Analytics never decides anything: revenue and economy KPIs (#147) come from the ledger and purchase records, not from these events.

## 2. Client events

| Event | Label | Value | Funnel step |
|---|---|---|---|
| `app_open` | none | 0 | Install (first `app_open` of an analytics id), session |
| `sign_in` | Provider: `google` or `apple` | 0 | Login |
| `nickname_set` | none | 0 | Login |
| `screen_view` | Screen id | 0 | |
| `queue_join` | Queue mode | 0 | First match |
| `purchase_start` | Product id | 0 | Purchase |
| `purchase_cancel` | Product id | 0 | Purchase |

The client sends only names on this list (`AnalyticsEvents.IsClientEvent`); any other name is ignored.

Where the client tracks them (`Game/Analytics/AppAnalytics`, `Game/Profile/NicknameSaver`, `Game/Shop/ShopPurchase`). An event tracked while `analytics` is off is dropped (section 1), so every call site runs only through `AnalyticsQueue.Track`:

- `app_open`: once per app run, on the first frame the queue holds the `analytics` consent. Consent is known only after the account's choices are read, so the event cannot fire at the first frame of the run. A withdrawal and a new grant in the same run do not track it again; the new analytics id gets its first `app_open` on the next run.
- `sign_in`: when a provider sign-in (`AuthSession.SignIn`) ends signed in, labeled `google` or `apple` (`AuthSession.ProviderName`). Resuming a stored session is not a sign-in. A sign-in on a fresh install happens before the account's consent is read, so it is dropped unless the queue already holds consent from earlier in the run.
- `nickname_set`: when the server accepts a nickname save (`NicknameSaver`, `PUT /profile/nickname`). The nickname itself is never tracked.
- `queue_join`: when a join intent is sent (`MatchQueue.TryJoin` returns true). The label is the queue and the mode as `docs/matchmaking.md` names them: `ranked_1v1`, `ranked_2v2`, `casual_1v1` or `casual_2v2` (draft form, owner to confirm). A join that is not sent is not tracked.
- `purchase_start`: when the store purchase of an offer opens (`docs/economy.md` 7.5 step 2), that is when `ShopPurchase` hands the store product id from the `POST /shop/begin` answer to the billing SDK (`IStoreBilling.Purchase`). The label is that store product id, never the offer id, price or currency. A purchase that never reaches the store (no store billing on the device, a refused or unreadable begin, no response) is not tracked, and a retry of an unsettled receipt (`ShopPurchase.Retry`) does not track it again.
- `purchase_cancel`: when the store answers that the player cancelled that purchase (`StoreBillingOutcome.Cancelled`), labeled with the same store product id. A deferred, failed or unavailable store answer is not a cancel; a completed purchase is the server's `purchase_complete` (section 3).

Not tracked yet: `screen_view` (no list of screen ids is decided).

## 3. Server events

Match end, purchase, reward, mission and pass events are written by the server from its own records (`docs/privacy.md` 6.4). For a player without `analytics` they carry no analytics id and only feed aggregates. All five are built.

Every server event is written through `analytics.RecordServerIn` (or `analytics.RecordServerLabeledIn` for an event with a label), in the same transaction as its source, and reads the player's consent there. When the latest `analytics` record of the account (`consent_records`, `docs/privacy.md` 6.3) is a grant, the row carries the account's analytics id first seen (`analytics_ids`) most recently after its latest `analytics` withdrawal. When the account has sent several analytics ids since that withdrawal, for example from several devices, the row carries the newest of them (owner decision of 2026-11-20). Otherwise, or when the account has sent no analytics id since that withdrawal, the row is written with no analytics id. An id from before a withdrawal is never bound again: the client rotates it then (section 1), and server events carry the new id once the client has sent it. A server event never has a device time.

| Event | Source |
|---|---|
| `match_end` | Match record (D-12) |
| `purchase_complete` | Validated purchase (D-17) |
| `reward_grant` | Ledger entry (D-16) |
| `mission_complete` | Mission progress (D-16) |
| `pass_level` | Pass progress (D-16) |

`match_end`: `matchstore.Finish` writes the match outcome to the `analytics` outbox topic in the same transaction as the result, for every match kind (`docs/matchmaking.md` section 4). The game server's relay on that topic (`HD_OUTBOX_POLL`) writes one `match_end` row per human player of the outcome, cancelled matches included, to `analytics_events`. Each row carries the player's analytics id as above, with the consent as it stands when the relay writes the row; it has no label and value 0, and its time is the match end. `analytics_sources` keeps the topic and key of every applied outbox row, so a redelivered match writes nothing twice; it holds no account and no link to the events. The label and value of `match_end` are not decided.

`mission_complete`: `mission.Store.ApplyMatch` writes one row per mission the match completes, in the same transaction as the mission's `completed_at` and its reward grant (`docs/pass-missions-medals.md` section 10). The match is applied once per player (`mission_matches`) and a completed mission is never completed again, so the row needs no `analytics_sources` entry. Like `match_end` the row carries the player's analytics id only with `analytics` on, has no label and value 0, and its time is the match end. The label and value of `mission_complete` are not decided.

`pass_level`: every pass XP gain (match XP, bundle XP, level purchase, Premium+) goes through `pass.Store.addXpIn`, which writes one row per level the new XP total crosses, in the same transaction as `pass_progress` and the level grants (`docs/pass-missions-medals.md` section 10). An XP key is applied once (`pass_xp_events`), so a repeated gain writes nothing twice; after a Premium revoke (`RevokePremiumIn`) lowers the XP, levels reached again write new rows. Like `match_end` the row carries the player's analytics id only with `analytics` on, has no label and value 0, and its time is the server time of the XP gain (the match end for match XP). The label and value of `pass_level` are not decided.

`reward_grant`: every reward bundle (missions, pass levels, daily login, promo codes, rank rewards, shop and chest offers, store purchases, admin grants) goes through `reward.Engine.ApplyIn`, which writes one row when it records a new grant in `reward_grants`, in the same transaction as the grant's ledger entries (`docs/economy.md` sections 4 and 5). A store purchase therefore also writes a `reward_grant` row; `purchase_complete` stays its own event. A repeated idempotency key writes nothing, and a revoke or a reinstatement is not a grant. Like `match_end` the row carries the player's analytics id only with `analytics` on, and its time is the server time of the grant. Its label is the grant's source type as the reward engine records it in `reward_grants.source_type` (`mission`, `pass`, `daily_login`, `promo`, `rank_reward`, `shop`, `chest`, `first_purchase`, `event`), never the source ref; an admin grant is labeled with the source type the admin chose, not `admin` (owner decision of 2026-11-20). Its value is 0; the value of `reward_grant` is not decided.

`purchase_complete`: `purchase.Store` writes one row when a receipt-validated purchase moves to `granted` (`docs/economy.md` section 7), from `POST /shop/complete` or from the store notification that settles a pending purchase, in the same transaction as the purchase state and its grants. A repeated completion of a granted purchase, a refund and a reversal write nothing. Like `match_end` the row carries the player's analytics id only with `analytics` on, has no label and value 0, and its time is the server time of the grant. The price, currency and product of the purchase are never recorded, so the row keeps no label and value 0 (owner decision of 2026-11-20); revenue comes from the purchase records (section 1).

## 4. Sending

`Core/Analytics/AnalyticsQueue` holds tracked events in memory while the device is offline; they are never written to disk. `Flush` hands at most `AnalyticsBatchSize` events to an `IAnalyticsSender`, one send at a time. When the queue holds `AnalyticsQueueLimit` events, the oldest is dropped.

`Game/Analytics/ApiAnalyticsSender` posts the batch to `POST /analytics/events` (section 5) through the signed API client, with the access token. It reports the batch as settled, so it leaves the queue, when the server answers 200, and when it refuses the batch for good: 400 (`analytics.malformed`), 403 `analytics.consent_required` and 409 `analytics.id_taken` (section 5). Every other answer keeps the batch for the next flush: no response (offline), a lost session, 429, 5xx and any other refusal. An expired access token is refreshed once by the API client before the answer counts.

`Core/Analytics/AnalyticsFlushTimer` flushes every `AnalyticsFlushSeconds` of real time (section 6) and at once when the app pauses. `Game/AnalyticsBoot` builds the queue, the sender and the timer when the network layer starts and drives the timer from a hidden object (`Update` and `OnApplicationPause`). The device time of an event is the UTC Unix time in milliseconds. A full batch does not trigger a send of its own: the docs define only the timer and the pause, and a size trigger would repeat a failing send while offline.

## 5. Server ingest

`POST /analytics/events` (`server/internal/analytics`) takes the access token and `{analyticsId, events: [{name, label, value, clientTimeMs}]}` as JSON.

- `analyticsId` is a random UUID in its 36-character form. The first account that sends it owns it (`analytics_ids`); the same id from another account answers 409 `analytics.id_taken` and stores nothing. This table is the only link from an analytics id to an account (`docs/privacy.md` P-7) and is what deletion and export use.
- The account must have the `analytics` consent on (`docs/privacy.md` 6.3). Otherwise the answer is 403 `analytics.consent_required` and nothing is stored, so a batch that races a withdrawal is dropped; the sender drops such a batch, since events from before consent are not held.
- An event is kept only if its name is on the client list in section 2, its value is 0, `clientTimeMs` is positive and its label matches the event: none for `app_open` and `nickname_set`, `google` or `apple` for `sign_in`, and for the others an id of at most 64 letters, digits, `_`, `.` or `-` starting with a letter or digit. Any other event is dropped and the rest of the batch is kept, so a client that knows a newer event does not lose the batch. Server event names are never accepted from the client.
- The answer is 200 `{accepted}`, the number of events stored. A body that is not valid JSON, is larger than 64 KiB or has no valid analytics id answers 400 `analytics.malformed`; a 400 or 409 batch will never be accepted, so the sender drops it rather than retrying.
- `analytics_events` stores the analytics id, name, label, value, the device time as sent and the server receive time. The source IP, the account id and the request headers are never stored.
- Erasure: `analytics_ids` is an erase-rows table in account deletion and its events go with it. Export: the `analytics` section lists the account's events with their analytics id.
- The global per-IP and per-account rate limits (`HD_RATE_*`) cover the route.
- Retention: every run of the hourly deletion job (`server/cmd/account-deletion run`, `docs/backup-restore.md` 3.7) deletes the `analytics_events` rows whose server receive time is older than `HD_ANALYTICS_RAW_RETENTION`, default 9480 h, which is `AnalyticsRawRetentionDays` (395) of `docs/privacy.md` section 4. The device time is never used for it. `analytics_ids` rows stay: they are the account's link for deletion and export and go with the account. The purge runs even when the deletion log is unreachable, and a failure of either fails the job. The same run deletes the server events of section 3 by their time and the `analytics_sources` rows older than the cutoff.

Retention cohorts (#147, `GET /admin/analytics/kpis`, `docs/admin.md` 9.1) are computed from these rows when asked and never stored: an analytics id joins the cohort of the economy day of its first `app_open` by receive time, and counts as retained on D1 or D7 with an `app_open` received on the day after or seven days after. A cohort older than the raw retention only shows the rows still kept.

Not built yet: aggregates without ids (none exist yet, so nothing is kept past the purge).

## 6. Config

The three keys are in the `liveops` domain of remote config (`docs/admin.md` 3.1, 5.14), edited, published and rolled back at `/config/liveops`. Defaults are drafts; the values are owner decisions not yet made, and so are the bounds below.

| Key | Default | Min | Max | Meaning |
|---|---|---|---|---|
| `AnalyticsBatchSize` | 50 | 1 | 400 | Most events in one send |
| `AnalyticsQueueLimit` | 500 | 1 | 5000 | Most events held while offline |
| `AnalyticsFlushSeconds` | 60 | 10 | 3600 | Time between flushes, in seconds |

The maximum batch is `AnalyticsSettings.MaxBatchSize` (400): 400 of the longest event the server accepts (a 64-character label) fit the 64 KiB body limit of section 5, so no batch is refused as too large. `AnalyticsQueueLimit` must be at least `AnalyticsBatchSize`, a key left out counting as its default; `server/internal/liveops` refuses any other snapshot, in the admin API and in `config-publish -domain liveops`.

Delivery: the game server follows the published `liveops` snapshot and sends the three values, with the defaults for keys not published, in `ServerHello.analytics_config` of every accepted hello (`proto/README.md`). The client starts with `AnalyticsSettings.Draft` (the same defaults) and `Game/Analytics/AnalyticsTuning` applies the values of each accepted hello: the queue takes the new batch size and limit and drops its oldest events above the new limit, and the flush timer restarts its interval when the interval changes. Values outside the client limits (a batch of 0 or above 400, a queue below the batch, an interval of 0) are ignored and the current settings stay. A server without a database sends no values, and the client keeps its current settings.
