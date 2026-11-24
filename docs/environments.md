# Environments, secrets and migrations

The environments the game runs in, every environment variable the code reads, where secrets live and how they are rotated, the secret scan, and how database schema changes reach each environment (#24).

Scope boundaries: deploy, rollback and orchestration are #25 and `docs/launch-runbook.md` section 5. Backups and restore are #188 and `docs/backup-restore.md`. Observability is #28. Admin config (every number a designer tunes) is never an environment variable; it is versioned in admin (ADR 0006).

## 1. Environments

| Environment | Purpose | Data | Who reaches it |
|---|---|---|---|
| `dev` | One developer's machine: the local stack of #22 and the in-process tests | Synthetic seed only (`docs/test-strategy.md` 7.2) | The developer |
| `staging` | Release candidates, drills (`docs/launch-runbook.md` GN-9 to GN-19), device smoke, load and soak, admin change sets tried on staff test accounts (`docs/admin.md` 3.4) | Synthetic seed and staff test accounts; sized like production for load runs | Team accounts with staging roles |
| `production` | Players | Player data under `docs/privacy.md` | Players; the `eng` role and admin roles granted for production only (`docs/admin.md` section 2, `docs/privacy.md` section 3) |

Rules:
- E-1 Each environment has its own database, Redis, object storage bucket, admin accounts and secrets. No credential is valid in two environments.
- E-2 No production data is copied into `staging` or `dev` (`docs/test-strategy.md` 7.4). A production incident is reproduced from its replay record and config version.
- E-3 Staging-only paths (the bot identity path of `docs/test-strategy.md` section 3.1) are refused by the server at start when it runs as `production` (#57, #58).
- E-4 An image is built once and promoted from `staging` to `production` unchanged; environments differ only by environment variables and secrets (`docs/launch-runbook.md` 5.1, #25).
- E-5 Admin change sets move from `staging` to `production` by export and import, approved again in production (`docs/admin.md` 3.6).
- E-6 PostgreSQL and Redis run the same major version in CI, `staging` and `production` (`docs/test-strategy.md` 2.3). PostgreSQL is 16 (#55, `server/internal/db/dbtest` runs `postgres:16-alpine`); Redis is 7 (#51, `server/internal/cluster/redistest` runs `redis:7-alpine`); the hosting is #25 (ADR 0010).

Hosting: the primary data region is Germany (EU) with Hetzner as hosting provider (`docs/privacy.md` section 2, OQ-3). Orchestration is ADR 0010: k3s on Hetzner Cloud in Germany, one cluster per environment, manifests in `deploy/k8s`, infrastructure as code with OpenTofu and its state in a Hetzner Object Storage bucket per environment. Node types per role (`control`, `game`, `db`), the load balancer and the bucket names are ADR 0014: shared-vCPU CX/CAX servers for the closed beta and the launch, the game node moved to dedicated vCPU (CCX23) when the latency trigger of ADR 0014 fires.

## 2. Environment variables

The server reads its configuration from the environment only, with the `HD_` prefix. A variable with a default is optional; a bad value stops the start with every bad variable named and exit code 2. A secret is never given a default and never logged (`docs/privacy.md` P-9).

Today no code on `main` reads an environment variable. The server variables below come from the open #264 (service skeleton, #43) and are marked so; when it merges, `server/README.md` holds the same table and the two stay in sync.

### 2.1 Server runtime

| Name | Required | Service | Default | Example placeholder | Secret | Source |
|---|---|---|---|---|---|---|
| `HD_LISTEN_ADDR` | No | server | `:8080` | `:8080` | No | from #264 |
| `HD_DEBUG_ADDR` | No | server | empty (off) | `127.0.0.1:6060` | No | #192 |
| `HD_PPROF_ADDR` | No | server | empty (off) | `127.0.0.1:6061` | No | #642 |
| `HD_LOG_LEVEL` | No | server | `info` | `info` | No | from #264 |
| `HD_ORIGINS` | No | server | none | `play.example.com,admin.example.com` | No | from #264 |
| `HD_SHUTDOWN_TIMEOUT` | No | server | `10s` | `10s` | No | from #264 |
| `HD_DRAIN_TIMEOUT` | No | server | `15m` | `15m` | No | from #264 |
| `HD_MIN_APP_VERSION` | No | server | `1.0.0` | `1.0.0` | No | from #264 |
| `HD_CATALOG_VERSION` | No | server | empty | `2026.11.08-1` | No | from #264; format: `docs/content-delivery.md` 2 |
| `HD_CONTENT_BASE_URL` | No | server | empty | `https://content.example.com/production` | No | #68, `docs/content-delivery.md` 2 |
| `HD_MAX_FRAME_BYTES` | No | server | `1024` | `1024` | No | from #264 |
| `HD_MAX_HELLO_BYTES` | No | server | `8192` | `8192` | No | from #205 |
| `HD_WRITE_QUEUE` | No | server | `32` | `32` | No | from #264 |
| `HD_HELLO_TIMEOUT` | No | server | `5s` | `5s` | No | from #264 |
| `HD_WRITE_TIMEOUT` | No | server | `5s` | `5s` | No | from #264 |
| `HD_CLOSE_TIMEOUT` | No | server | `2s` | `2s` | No | from #264 |
| `HD_COMPRESSION_THRESHOLD` | No | server | `256` | `256` | No | from #264 |
| `HD_INTENTS_PER_SECOND` | No | server | `10` | `10` | No | from #264 |
| `HD_INTENT_BURST` | No | server | `20` | `20` | No | from #264 |
| `HD_RATE_IP_PER_MINUTE` | No | server | `60` | `60` | No | from #184 |
| `HD_RATE_IP_BURST` | No | server | `20` | `20` | No | from #184 |
| `HD_RATE_ACCOUNT_PER_MINUTE` | No | server | `30` | `30` | No | from #184 |
| `HD_RATE_ACCOUNT_BURST` | No | server | `10` | `10` | No | from #184 |
| `HD_RATE_SIGNIN_PER_MINUTE` | No | server | `5` | `5` | No | from #184 |
| `HD_RATE_SIGNIN_BURST` | No | server | `5` | `5` | No | from #184 |
| `HD_RATE_SPEND_PER_MINUTE` | No | server | `5` | `5` | No | from #184 |
| `HD_RATE_SPEND_BURST` | No | server | `5` | `5` | No | from #184 |
| `HD_READ_TIMEOUT` | No | server, admin-api | `15s` | `15s` | No | from #184 |
| `HD_IDLE_TIMEOUT` | No | server, admin-api | `60s` | `60s` | No | from #184 |
| `HD_CLIENT_IP_HEADER` | No | server | empty (connection address) | `CF-Connecting-IP` | No | from #184; only a header the edge overwrites on every request |
| `HD_PING_INTERVAL` | No | server | `15s` | `15s` | No | from #264 (default from #262) |
| `HD_PONG_TIMEOUT` | No | server | `5s` | `5s` | No | from #264 (default from #262) |
| `HD_RECONNECT_GRACE` | No | server | `60s` | `60s` | No | from #264 |
| `HD_REDIS_URL` | No; required for more than one node | server | empty (single node, no routing) | `redis://:<password>@redis:6379/0` | Yes | from #51 |
| `HD_NODE_ID` | No | server | hostname (the pod name) | `server-6d4f9c7b8-x2k9q` | No | from #51 |
| `HD_NODE_ADDR` | With `HD_REDIS_URL` | server | empty | `10.42.0.17:8080` (pod IP and port) | No | from #51 |
| `HD_NODE_CAPACITY` | No | server | `10000` | `10000` | No | from #51 |
| `HD_NODE_HEARTBEAT` | No | server | `2s` | `2s` | No | from #51 |
| `HD_NODE_TTL` | No | server | `6s` | `6s` | No | from #51 |
| `HD_ROUTE_TTL` | No | server | `2h` | `2h` | No | from #51 |
| `HD_SCALE_OUT_AT` | No | server | `0.7` | `0.7` | No | from #51 |
| `HD_SCALE_IN_AT` | No | server | `0.3` | `0.3` | No | from #51 |
| `HD_RELAY_DIAL_TIMEOUT` | No | server | `5s` | `5s` | No | from #51 |
| `HD_DATABASE_URL` | No; required for sign-in | server | empty (no database, no `/auth` routes) | `postgres://hd_server:<password>@postgres.hypedomino.svc:5432/hypedomino?sslmode=require` | Yes | from #57 |
| `HD_GOOGLE_AUDIENCES` | One of the two with `HD_DATABASE_URL` | server | empty | `<android-client-id>.apps.googleusercontent.com,<ios-client-id>.apps.googleusercontent.com` | No | from #57 |
| `HD_APPLE_AUDIENCES` | One of the two with `HD_DATABASE_URL` | server | empty | `com.example.hypedomino` (the app bundle id) | No | from #57 |
| `HD_NONCE_TTL` | No | server | `10m` | `10m` | No | from #57 |
| `HD_TOKEN_KEYS` | With `HD_DATABASE_URL` | server | none | `2026-10:<base64 of 32 random bytes>,2026-07:<previous key>` | Yes | from #58 |
| `HD_ACCESS_TTL` | No | server | `15m` | `15m` | No | from #58 |
| `HD_REFRESH_TTL` | No | server | `720h` | `720h` | No | from #58 |
| `HD_REFRESH_REUSE_GRACE` | No | server | `30s` | `30s` | No | from #58 |
| `HD_NICKNAME_MIN_LENGTH` | No | server | `3` | `3` | No | from #59 |
| `HD_NICKNAME_MAX_LENGTH` | No | server | `16` | `16` | No | from #59 |
| `HD_NICKNAME_ALLOW_MIXED_SCRIPT` | No | server | `false` | `false` | No | from #59 |
| `HD_NICKNAME_RENAME_COOLDOWN` | No | server | `720h` | `720h` | No | from #59 |
| `HD_NICKNAME_RELEASE` | No | server | `720h` | `720h` | No | from #59 |
| `HD_COSMETICS_MANIFEST` | No | server, admin-api | empty (built-in manifest) | `/etc/hypedomino/cosmetics.json` | No | from #61; admin-api from #142, for the `mythic` check of promo codes |
| `HD_EMOJI_SLOTS` | No | server | `6` | `6` | No | from #61 |
| `HD_ACTION_SLOTS` | No | server | `4` | `4` | No | from #61 |
| `HD_EMOJI_BURST` | No | server | `3` | `3` | No | from #151 |
| `HD_EMOJI_REFILL` | No | server | `2s` | `2s` | No | from #151 |
| `HD_REPORT_AUTO_RENAME_THRESHOLD` | No | server | `5` (0 turns it off) | `5` | No | from #155 |
| `HD_REPORT_AUTO_RENAME_WINDOW` | No | server | `168h` | `168h` | No | from #155 |
| `HD_REPORT_DAILY_LIMIT` | No | server | `10` | `10` | No | from #155 |
| `HD_REPORT_REJECTED_LIMIT` | No | server | `5` | `5` | No | from #155 |
| `HD_REPORT_REJECTED_WINDOW` | No | server | `720h` | `720h` | No | from #155 |
| `HD_REPORT_REJECTED_SILENCE` | No | server | `168h` | `168h` | No | from #155 |
| `HD_RARITY_ORDINALS` | No | server | `common:1,rare:2,epic:3,legendary:4,mythic:5` | same | No | from #98 |
| `HD_SHOP_CATALOG` | No | server | empty (shop closed until a publish) | `/etc/hypedomino/shop.json` | No | from #104 |
| `HD_SHOP_DAILY_SLOTS` | No | server, admin-api | `4` | `4` | No | from #104; admin-api from #139 |
| `HD_SHOP_WEEKLY_SLOTS` | No | server, admin-api | `2` | `2` | No | from #104; admin-api from #139 |
| `HD_SHOP_RESET_HOUR_UTC` | No | server, admin-api | `0` | `0` | No | from #104; admin-api from #139 |
| `HD_SHOP_REFERENCE_PRICE_DAYS` | No | admin-api | `30` (at least 1) | `30` | No | from #139 |
| `HD_CONFIG_POLL` | No | server | `10s` | `10s` | No | from #104 |
| `HD_SEASON_POLL` | No | server | `1m` | `1m` | No | from #115 |
| `HD_OUTBOX_POLL` | No | server | `2s` (at least 100ms) | `2s` | No | from #440 |
| `HD_MATCHMAKING_TICK` | No | server | `1s` (at least 100ms) | `1s` | No | from #478 |
| `HD_MATCHMAKING_LEADER_TTL` | No | server | `5s` (at least two ticks) | `5s` | No | from #478 |
| `HD_CLOCK_SKEW_TOLERANCE` | No | server | `5m` | `5m` | No | from #208 |
| `HD_CLOCK_SKEW_WEIGHT` | No | server | `1` | `1` | No | from #208 |
| `HD_REPLAY_REJECT_THRESHOLD` | No | server | `5` | `5` | No | from #207 |
| `HD_REPLAY_REJECT_WINDOW` | No | server | `10m` | `10m` | No | from #207 |
| `HD_REPLAY_REJECT_WEIGHT` | No | server | `5` | `5` | No | from #207 |
| `HD_MAC_FAILURE_THRESHOLD` | No | server | `5` | `5` | No | from #207 |
| `HD_MAC_FAILURE_WINDOW` | No | server | `10m` | `10m` | No | from #207 |
| `HD_MAC_FAILURE_WEIGHT` | No | server | `5` | `5` | No | from #207 |
| `HD_UNOWNED_EQUIP_THRESHOLD` | No | server | `5` | `5` | No | from #211 |
| `HD_UNOWNED_EQUIP_WINDOW` | No | server | `10m` | `10m` | No | from #211 |
| `HD_UNOWNED_EQUIP_WEIGHT` | No | server | `5` | `5` | No | from #211 |
| `HD_INTEGRITY_NONCE_TTL` | No | server | `5m` | `5m` | No | from #205 |
| `HD_FAKE_RECEIPT_WEIGHT` | No | server | `20` | `20` | No | from #204 |
| `HD_REFUND_WEIGHT` | No | server | `3` | `3` | No | from #204 |
| `HD_COUNTRY_HEADER` | No | server | empty (no IP country fallback) | `CF-IPCountry` | No | from #117; only a header the edge overwrites on every request |
| `HD_COUNTRY_CHANGE_COOLDOWN` | No | server | `4320h` | `4320h` | No | from #117 |
| `HD_ACCOUNT_DELETION_GRACE` | No | server, account-deletion | `720h` | `720h` | No | from #63; `AccountDeletionGraceDays` of `docs/privacy.md` section 4; the same value in both |
| `HD_DELETION_LOG_URL` | Yes | account-deletion | none: the job refuses to start | `https://<object-store-host>/<deletion-log-bucket>` | No | from #601; its own bucket in the backup location (`docs/backup-restore.md` 3.7), never readable or writable by the server or admin-api keys |
| `HD_DELETION_LOG_REGION` | Yes | account-deletion | none | `<store-region>` | No | from #601 |
| `HD_DELETION_LOG_ACCESS_KEY` | Yes | account-deletion | none | `<deletion-log-access-key>` | Yes | from #601 |
| `HD_DELETION_LOG_SECRET_KEY` | Yes | account-deletion | none | `<deletion-log-secret-key>` | Yes | from #601 |
| `HD_DELETION_LOG_RETENTION` | No | account-deletion | `840h` | `840h` | No | from #601; `BackupRetentionDays` of `docs/privacy.md` section 4 |
| `HD_ANALYTICS_RAW_RETENTION` | No | account-deletion | `9480h` | `9480h` | No | from #181; `AnalyticsRawRetentionDays` of `docs/privacy.md` section 4 |
| `HD_EXPORT_STORE_URL` | No | server | none: `POST /account/export` is not served | `https://<object-store-host>/<private-export-bucket>` | No | from #63; a private bucket, never the public asset bucket |
| `HD_EXPORT_STORE_REGION` | With `HD_EXPORT_STORE_URL` | server | none | `<store-region>` | No | from #63 |
| `HD_EXPORT_STORE_ACCESS_KEY` | With `HD_EXPORT_STORE_URL` | server | none | `<export-store-access-key>` | Yes | from #63 |
| `HD_EXPORT_STORE_SECRET_KEY` | With `HD_EXPORT_STORE_URL` | server | none | `<export-store-secret-key>` | Yes | from #63 |
| `HD_EXPORT_LINK_TTL` | No | server | `72h`, at most `168h` | `72h` | No | from #63; `ExportLinkTtlHours` of `docs/privacy.md` section 4 |
| `HD_DATA_EXPORT_COOLDOWN` | No | server | `168h` | `168h` | No | from #63; `DataExportCooldownDays` of `docs/privacy.md` section 11 |
| `HD_DUPLICATE_COMPENSATION_PERCENT` | No | server | `20` | `20` | No | from #104 |
| `HD_PLAY_PACKAGE` | No | server | empty (Google purchases off) | `com.example.hypedomino` | No | from #106 |
| `HD_PLAY_SERVICE_ACCOUNT` | With `HD_PLAY_PACKAGE` | server | empty | `/run/secrets/play-service-account.json` | Yes, the file it names | from #106 |
| `HD_PLAY_NOTIFICATION_TOKEN` | With `HD_PLAY_PACKAGE` | server | empty | `<random-32-bytes>` | Yes | from #106 |
| `HD_APPLE_BUNDLE_ID` | No | server | empty (Apple purchases off) | `com.example.hypedomino` | No | from #106 |
| `HD_STORE_SANDBOX` | No | server | `false` (`true` in dev and staging) | `false` | No | from #106 |
| `HD_APP_ATTEST_TEAM_ID` | No | server | empty (App Attest off) | Apple team id | No | from #205 |
| `HD_APP_ATTEST_ROOT_CA` | With team id | server | empty | path of the Apple App Attestation Root CA PEM | No | from #205 |
| `HD_APP_ATTEST_DEVELOPMENT` | No | server | `false` (`true` in dev) | `false` | No | from #205 |
| `HD_PLAY_CERT_DIGESTS` | Production | server | empty (Play Integrity off) | SHA-256 of the release signing certificates, base64url, comma separated | No | from #205 |
| `HD_PLAY_INTEGRITY_MAX_AGE` | No | server | `2m` | `2m` | No | from #205 |
| `HD_INTEGRITY_FAILURE_ACTION` | No | server | `flag` | `flag` | No | from #205 |
| `HD_DEVICE_RISK_ACTION` | No | server | `flag` | `flag` | No | from #205 |
| `HD_INTEGRITY_MIN_DEVICE_LEVEL` | No | server | `device` | `device` | No | from #205 |
| `HD_INTEGRITY_REQUIRE_LICENSED` | No | server | `false` | `false` | No | from #205 |
| `HD_INTEGRITY_FAILURE_WEIGHT` | No | server | `20` | `20` | No | from #205 |
| `HD_DEVICE_RISK_WEIGHT` | No | server | `5` | `5` | No | from #205 |
| `HD_REFUNDS_BEFORE_PURCHASE_BLOCK` | No | server | `3` | `3` | No | from #106 |
| `HD_REFUND_WINDOW_DAYS` | No | server | `365` | `365` | No | from #106 |
| `HD_PROMO_KEY` | No | server, admin-api | empty (promo redemption off, 503 `promo.unavailable`; admin promo routes answer `not_configured`) | `<random-32-bytes>` | Yes | from #108; `cmd/promo` and admin-api (#142) need the same key |
| `HD_PROMO_CODE_LENGTH` | No | server, admin-api | `12` | `12` | No | from #108 |
| `HD_PROMO_MAX_VALIDITY_DAYS` | No | server, admin-api | `90` | `90` | No | from #108 |
| `HD_PROMO_ATTEMPTS_PER_HOUR` | No | server | `10` | `10` | No | from #108 |
| `HD_PROMO_FAILED_ATTEMPTS_BEFORE_LOCK` | No | server | `5` | `5` | No | from #108 |
| `HD_PROMO_LOCK_MINUTES` | No | server | `60` | `60` | No | from #108 |
| `HD_PROMO_COINS_APPROVAL_THRESHOLD` | No | server, admin-api | `2000` | `2000` | No | from #108 |
| `HD_PROMO_LOCK_SIGNAL_WEIGHT` | No | server | `5` | `5` | No | from #211 |
| `HD_PROMO_SPIKE_PERCENT` | No | server | `50` | `50` | No | from #211 |
| `HD_PROMO_SPIKE_WINDOW_MINUTES` | No | server | `60` | `60` | No | from #211 |
| `HD_SENTRY_DSN` | No | server, admin-api | empty (panics are logged and counted only, nothing leaves the node) | `https://<public-key>@o<org-id>.ingest.de.sentry.io/<project-id>` | Yes | #182; only a Sentry EU data region DSN (host ending in `.ingest.de.sentry.io`) is accepted (`docs/privacy.md` section 2), any other value stops the start without echoing it |
| `HD_SENTRY_ENVIRONMENT` | No | server, admin-api | empty | `staging` | No | #182; the environment name of section 1 the reports are filed under |

The gateway limits and the reconnect grace are environment variables only until admin config serves them (#210, `docs/penalties.md` section 8); then the admin value wins and the variable is removed in the same PR.

### 2.2 Admin API runtime

`server/cmd/admin-api` (`docs/admin.md` 9.1). It also reads `HD_LOG_LEVEL` and `HD_DATABASE_URL` as above, `HD_REDIS_URL` for its sign-in nonces and rate-limit buckets when more than one instance runs, `HD_READ_TIMEOUT` and `HD_IDLE_TIMEOUT` for its slow client cut-offs, and `HD_SHOP_DAILY_SLOTS`, `HD_SHOP_WEEKLY_SLOTS` and `HD_SHOP_RESET_HOUR_UTC`, which must match the game servers so the shop catalog is checked the same way, and `HD_SHOP_REFERENCE_PRICE_DAYS`, the struck-through price window of `docs/economy.md` 6.3. For the promo routes (#142) it reads `HD_PROMO_KEY`, `HD_PROMO_CODE_LENGTH`, `HD_PROMO_MAX_VALIDITY_DAYS`, `HD_PROMO_COINS_APPROVAL_THRESHOLD` and `HD_COSMETICS_MANIFEST`, which must match the game servers so a code it stores redeems there. For the store price push (`docs/economy.md` 7.4, #585) it reads `HD_PLAY_PACKAGE` and `HD_PLAY_SERVICE_ACCOUNT` as the server does, and the App Store Connect API key below; a store without its variables is shown as not configured and nothing is pushed to it. The Play service account needs the Play Console permission to manage store presence and pricing of the app; the App Store Connect key needs the App Manager role. Its recovered panics go to Sentry with `HD_SENTRY_DSN` and `HD_SENTRY_ENVIRONMENT` as for the server (2.1), under the same EU data region rule. Each environment has its own values (E-1).

| Name | Required | Service | Default | Example placeholder | Secret | Source |
|---|---|---|---|---|---|---|
| `HD_ADMIN_LISTEN_ADDR` | No | admin-api | `:8090` | `:8090` | No | from #134 |
| `HD_ADMIN_ENVIRONMENT` | Yes | admin-api | none | `staging` | No | from #134 |
| `HD_ADMIN_GOOGLE_DOMAIN` | Yes | admin-api | none | `example.com` | No | from #134 |
| `HD_ADMIN_GOOGLE_AUDIENCES` | Yes | admin-api | none | `<admin-web-client-id>.apps.googleusercontent.com` | No | from #134 |
| `HD_ADMIN_SESSION_IDLE` | No | admin-api | `30m` (`AdminSessionIdleMinutes`) | `30m` | No | from #134 |
| `HD_ADMIN_SESSION_MAX` | No | admin-api | `12h` (`AdminSessionMaxHours`) | `12h` | No | from #134 |
| `HD_ADMIN_STEP_UP` | No | admin-api | `10m` (`AdminStepUpMinutes`) | `10m` | No | from #134 |
| `HD_ADMIN_APPROVAL_TTL` | No | admin-api | `24h` (`AdminApprovalTtlHours`) | `24h` | No | from #134 |
| `HD_ADMIN_GRANT_APPROVAL_COINS` | No | admin-api | `2000` (`AdminGrantApprovalCoins`) | `2000` | No | from #144 |
| `HD_ADMIN_RATE_ACCOUNT_PER_MINUTE` | No | admin-api | `120` (`AdminRateAccountPerMinute`) | `120` | No | from #148 |
| `HD_ADMIN_RATE_ACCOUNT_BURST` | No | admin-api | `40` (`AdminRateAccountBurst`) | `40` | No | from #148 |
| `HD_ADMIN_RATE_SIGNIN_PER_MINUTE` | No | admin-api | `10` (`AdminRateSignInPerMinute`) | `10` | No | from #148 |
| `HD_ADMIN_RATE_SIGNIN_BURST` | No | admin-api | `10` (`AdminRateSignInBurst`) | `10` | No | from #148 |
| `HD_ASSET_STORE_URL` | No | admin-api | none: `/admin/assets` lists rows and answers `asset_store_off` to uploads | `https://<bucket>.<object-store-host>` or `https://<object-store-host>/<bucket>` | No | from #135 |
| `HD_ASSET_STORE_REGION` | With `HD_ASSET_STORE_URL` | admin-api | none | `<store-region>` | No | from #135 |
| `HD_ASSET_STORE_ACCESS_KEY` | With `HD_ASSET_STORE_URL` | admin-api | none | `<asset-store-access-key>` | Yes | from #135 |
| `HD_ASSET_STORE_SECRET_KEY` | With `HD_ASSET_STORE_URL` | admin-api | none | `<asset-store-secret-key>` | Yes | from #135 |
| `HD_ASSET_STORE_URL_TTL` | No | admin-api | `15m`, at most `168h` | `15m` | No | from #135 |
| `HD_CONTENT_STORE_URL` | No | admin-api | none: publishing a language bundle or a cosmetics manifest answers `content_store_off` | `https://<object-store-host>/hypedomino-content-<environment>`; it must name `HD_ADMIN_ENVIRONMENT` and no other environment | No | from #214, `docs/content-delivery.md` CD-27 |
| `HD_CONTENT_STORE_REGION` | With `HD_CONTENT_STORE_URL` | admin-api | none | `<store-region>` | No | from #214 |
| `HD_CONTENT_STORE_ACCESS_KEY` | With `HD_CONTENT_STORE_URL` | admin-api | none | `<content-store-access-key>` | Yes | from #214 |
| `HD_CONTENT_STORE_SECRET_KEY` | With `HD_CONTENT_STORE_URL` | admin-api | none | `<content-store-secret-key>` | Yes | from #214 |
| `HD_PLAY_PACKAGE` | No | admin-api | empty (store price push to Google Play not configured) | `com.example.hypedomino` | No | from #585 |
| `HD_PLAY_SERVICE_ACCOUNT` | With `HD_PLAY_PACKAGE` | admin-api | empty | `/run/secrets/play-service-account.json` | Yes, the file it names | from #585 |
| `HD_ASC_ISSUER_ID` | With the other `HD_ASC_*` | admin-api | empty (store price push to the App Store not configured) | `<app-store-connect-issuer-id>` | No | from #585 |
| `HD_ASC_KEY_ID` | With the other `HD_ASC_*` | admin-api | empty | `<app-store-connect-key-id>` | No | from #585 |
| `HD_ASC_PRIVATE_KEY` | With the other `HD_ASC_*` | admin-api | empty | `/run/secrets/app-store-connect-key.p8` | Yes, the file it names (the `.p8` API key) | from #585 |
| `HD_ASC_APP_ID` | With the other `HD_ASC_*` | admin-api | empty | `<numeric-apple-app-id>` | No | from #585 |
| `HD_TEMP_BAN_HOURS` | No | admin-api | none: temporary bans answer `not_configured` until set (`TempBanHours`, value owed before launch) | `<temp-ban-hours>` | No | from #144 |
| `HD_MUTE_HOURS` | No | admin-api | 24 (`ModerationMuteHours`) | `<mute-hours>` | No | from #144 |
| `HD_RANKED_BAN_HOURS` | No | admin-api | 72 (`RankedBanHours`, placeholder until the owner sets it) | `<ranked-ban-hours>` | No | from #186 |

The admin web server (`admin/`) reads three variables on the server side only; the browser never sees the API address. A missing one stops the page that needs it with an error naming the variable.

| Name | Required | Service | Default | Example placeholder | Secret | Source |
|---|---|---|---|---|---|---|
| `HD_ADMIN_API_URL` | Yes | admin | none | `http://admin-api:8090` | No | from #364 |
| `HD_ADMIN_GOOGLE_CLIENT_ID` | Yes | admin | none | `<admin-web-client-id>.apps.googleusercontent.com`, one of `HD_ADMIN_GOOGLE_AUDIENCES` | No | from #134 |
| `HD_ADMIN_ASSET_ORIGIN` | With `HD_ASSET_STORE_URL` | admin | none | `https://<object-store-host>` | No | from #135 |

### 2.3 CI

| Name | Required | Service | Example placeholder | Secret | Stored as |
|---|---|---|---|---|---|
| `UNITY_LICENSE` | Yes, for the `client` job | CI, Unity test runner | `<unity-license-file-contents>` | Yes | Repository secret |
| `UNITY_EMAIL` | Yes, for the `client` job | CI, Unity test runner | `<unity-account-email>` | Yes | Repository secret |
| `UNITY_PASSWORD` | Yes, for the `client` job | CI, Unity test runner | `<unity-account-password>` | Yes | Repository secret |
| `ANDROID_APPLICATION_ID` | Yes, for a `client-v*` tag | `Release build` workflow | `<play-package-name>` | No | Repository variable |
| `ANDROID_KEYSTORE_BASE64` | Yes, for a `client-v*` tag | `Release build` workflow, upload key | `<base64-of-upload-keystore>` | Yes | Repository secret |
| `ANDROID_KEYSTORE_PASS` | Yes, for a `client-v*` tag | `Release build` workflow | `<keystore-password>` | Yes | Repository secret |
| `ANDROID_KEYALIAS_NAME` | Yes, for a `client-v*` tag | `Release build` workflow | `<upload-key-alias>` | Yes | Repository secret |
| `ANDROID_KEYALIAS_PASS` | Yes, for a `client-v*` tag | `Release build` workflow | `<upload-key-password>` | Yes | Repository secret |
| `GOOGLE_PLAY_SERVICE_ACCOUNT_JSON` | Yes, for a `client-v*` tag | `Release build` workflow, Play internal testing upload | `<play-publisher-service-account-json>` | Yes | Repository secret |
| `CONTENT_STORE_URL` | No: without it a tag build skips the CDN upload | `Release build` workflow, `content-publish` | `https://<object-store-host>/hypedomino-content-<environment>` | No | Repository variable |
| `CONTENT_STORE_REGION` | With `CONTENT_STORE_URL` | `Release build` workflow, `content-publish` | `<store-region>` | No | Repository variable |
| `CONTENT_ENVIRONMENT` | With `CONTENT_STORE_URL` | `Release build` workflow, `content-publish` | `staging` | No | Repository variable |
| `CONTENT_STORE_ACCESS_KEY` | With `CONTENT_STORE_URL` | `Release build` workflow, `content-publish` | `<content-store-access-key>` | Yes | Repository secret |
| `CONTENT_STORE_SECRET_KEY` | With `CONTENT_STORE_URL` | `Release build` workflow, `content-publish` | `<content-store-secret-key>` | Yes | Repository secret |
| `CLIENT_SERVER_URL` | Yes, for a `client-v*` tag | `Release build` workflow, passed as `-serverUrl` (2.5) | `https://play.example.com` | No | Repository variable |
| `CLIENT_HTTP_TIMEOUT_SECONDS` | Yes, for a `client-v*` tag | `Release build` workflow, passed as `-httpTimeoutSeconds` (2.5) | not set: an owner decision (section 7) | No | Repository variable |
| `CLIENT_SENTRY_DSN` | No: without it a tag build warns and sends no crash reports | `Release build` workflow, written to the `-crashReporting` file (2.5) for a `client-v*` tag | `https://<public-key>@o<org-id>.ingest.de.sentry.io/<project-id>` | Yes | Repository secret |
| `CLIENT_SENTRY_ENVIRONMENT` | No | `Release build` workflow, written to the `-crashReporting` file (2.5) | `staging` | No | Repository variable |

### 2.4 Not read yet

The client reads no environment variable; its crash reporting DSN is a build input (2.5). The server and the admin API read `HD_SENTRY_DSN` and `HD_SENTRY_ENVIRONMENT` (2.1, 2.2).

### 2.5 Client build inputs

Where the player connects is fixed when it is built. `Editor/ReleaseBuild` takes these arguments and writes them to `Resources/ServerEndpoint.json` for that build only; the file is removed after the build and never committed (`client/README.md`, App flow). Neither is a secret.

| Argument | Required | Without it | Example placeholder | Used for |
|---|---|---|---|---|
| `-serverUrl` | Together with `-httpTimeoutSeconds`; one without the other fails the build | The player has no server, logs `Net no server endpoint` and stays at boot. In the editor: `http://localhost:8080`, the `server` port of `docker-compose.yml` | `https://play.example.com` | The base of `/auth` and the player API. The WebSocket is `<serverUrl>/ws` (`GET /ws`, `server/internal/app`), `wss` for `https` and `ws` for `http` |
| `-httpTimeoutSeconds` | Together with `-serverUrl`; a whole number from 1 | In the editor: no timeout (`UnityWebRequest.timeout` 0) | not set: an owner decision (section 7) | The `UnityWebRequest` timeout of every `/auth` and player API request |
| `-crashReporting` | No | No crash reports: nothing is hooked and nothing leaves the device | `Signing/crash-reporting.json` with `{"dsn": "https://<public-key>@o<org-id>.ingest.de.sentry.io/<project-id>", "environment": "staging"}` | Client crash reports (#182, `client/README.md`, Error reports), written to `Resources/CrashReporting.json` for that build only. Only a Sentry EU data region DSN is accepted and a refused one is not echoed; a file without `dsn` fails the build. The DSN is a secret in CI (2.3) although it ships inside the build |

## 3. Where secrets live

| Secret use | Lives in | Reaches the consumer as |
|---|---|---|
| CI only (Unity license) | GitHub Actions repository secrets | Job environment variables, masked in logs |
| `staging` runtime | GitHub Actions environment secrets of the `staging` environment | Environment variables injected by the staging deploy (#25) |
| `production` runtime | GitHub Actions environment secrets of the `production` environment, deployable only from protected refs | Environment variables injected by the production deploy (#25) |
| `dev` | A local `.env` file, ignored by `.gitignore` | The local stack (#22) |
| Redis password (`HD_REDIS_URL`) | The `server-env` secret of each cluster (ADR 0010), from the environment secrets above | Environment variable of the server pods |
| Database runtime role password (`HD_DATABASE_URL`) | The `server-env` secret of each cluster, from the environment secrets above; the migration role (M-7) is the `HD_DATABASE_URL` of the separate `migrate-env` secret of each cluster | Environment variable of the server pods; `migrate-env` only of the `migrate` Job pods |
| PostgreSQL superuser and `walg` role passwords, PostgreSQL TLS key | Secrets `postgres-superuser`, `walg-credentials` and `postgres-tls` of each cluster (`docs/backup-restore.md` 3.4), from the environment secrets above | Environment variables and mounted files of the `postgres` pod and the backup CronJobs |
| Deletion job: database URL and deletion log key (`HD_DATABASE_URL`, `HD_DELETION_LOG_ACCESS_KEY`, `HD_DELETION_LOG_SECRET_KEY`) | The `account-deletion-env` secret of each cluster, from the environment secrets above; never in `server-env` (`docs/backup-restore.md` 3.7) | Environment variables of the `account-deletion` CronJob only |
| Access token signing keys (`HD_TOKEN_KEYS`) | The `server-env` secret of each cluster; generated with `openssl rand -base64 32` per key id, never derived from a password | Environment variable of the server pods; rotated with overlap (section 5 step 3) |
| Client crash report DSN (`CLIENT_SENTRY_DSN`) | GitHub Actions repository secret; one DSN for the builds of a tag, from the Sentry EU data region project the owner creates | Written by the `Release build` workflow to `client/Signing/crash-reporting.json`, removed after the build, and built into the player as `Resources/CrashReporting.json`; unset until the Sentry account exists (section 7) |
| Server crash report DSN (`HD_SENTRY_DSN`) | The `server-env` secret of each cluster, from the environment secrets above; one DSN per environment, taken from the Sentry EU data region project the owner creates | Environment variable of the server pods and of the admin API, which gets the same DSN; unset until the Sentry account exists (section 7) |

Rules:
- S-1 No secret is committed: not in code, docs, fixtures, images, `.env` files or CI files. `.gitignore` ignores `.env`, `.env.*` (except `.env.example`), keystores, signing certificates, provisioning profiles and `service-account*.json`. Only `.env.example` with placeholders may be committed.
- S-2 Test keys are generated at test time, never committed (CLAUDE.md).
- S-3 A secret is read by the process from its environment at start and kept in memory; it is never written to the image, a log, a metric label, a crash report or a replay record.
- S-4 `production` secrets are visible to the `production` GitHub environment only; workflows for pull requests never receive them.
- S-5 Values are entered in the GitHub settings or with `gh secret set --env <environment>` reading from standard input, never typed into a command line, chat or issue.

## 4. Secret scan

The `secret-scan` job of `.github/workflows/ci.yml` runs on every pull request and every push to `main`, whatever paths changed. It runs gitleaks, pinned by version and by the SHA-256 of the release archive, with findings redacted in the log:

- On a pull request it scans every commit from the merge base to the head of the branch.
- On a push to `main` it scans the full git history.

`CI complete` fails unless `secret-scan` succeeded, so a finding blocks the merge. A finding is never allowlisted to pass: a real secret is rotated (section 5); a false positive (generated code, test vectors) gets a narrow path allowlist in `.gitleaks.toml`, reviewed in its PR, never a disabled rule.

Local run, before pushing a branch that touched credentials or config:

```bash
gitleaks git --redact --no-banner .
```

## 5. Rotation

The security lead owns rotation (`docs/launch-runbook.md` section 1). Every rotation is recorded in an issue labelled `area:infra` with the secret name, environment, reason and time, never the value.

When: on any suspected leak, when a person with access to it leaves the team or changes role, when the issuer expires it, and at the regular cadence below (decided 2026-10-28). The cadence counts from the last rotation; the rotation issue of a secret states its next due date.

| Secret type | Regular cadence | Why |
|---|---|---|
| Session token signing keys (#58) | 90 days | Signs every session; rotation is cheap because of the overlap in step 3 |
| PostgreSQL and Redis passwords, runtime and migration roles (M-7) | 180 days | Reachable only inside the cluster network; each rotation needs a deploy |
| Hosting and infrastructure tokens: cloud API, object storage keys, registry and deploy tokens (ADR 0010) | 180 days | Broad power over the environment |
| Store and vendor API credentials: Play service account key, App Store Connect API key, crash reporting (#106, #182) | 365 days | Narrow scope, issued and revoked in the vendor console |
| CI-only credentials (Unity license and account) | 365 days, or when the license is renewed | CI only, never in a runtime environment |
| App signing keys: Play upload key, Apple distribution certificate | No cadence: only on a leak or when the issuer expires it | A new key changes the signing digest; the integrity allowlist (#205) is updated in the same change |

Procedure for one secret in one environment:
1. Create the new value at its issuer (database role password, key pair, store or vendor credential). Keep the old one valid.
2. Store the new value in the environment's secrets (section 3).
3. Deploy, so every node runs with the new value (#25). Signing keys rotate with an overlap: the new key signs, the old key still verifies until every token it signed has expired (#58).
4. Verify on that environment: health and readiness pass, logins and matches work, no authentication errors in the logs (#28).
5. Revoke the old value at the issuer.
6. Rotate `staging` first, then `production`, unless the leak is in `production`.

On a leak: it is a SEV1 incident (`docs/launch-runbook.md` 6.1, 6.5). Revoke first when the exposed value is in active use by someone else, accepting the outage, then follow the procedure. A secret found in git history is rotated; rewriting history does not undo the exposure and is not a substitute. A personal data breach follows `docs/privacy.md` and #187.

## 6. Database migrations

Schema changes are versioned migrations reviewed in PRs (ADR 0005). The tool is goose (#55): SQL files under `server/internal/db/migrations`, embedded in the binaries, applied by `server/cmd/migrate` under a PostgreSQL session advisory lock. PostgreSQL major version: 16 (E-6).

Rules:
- M-1 Forward only in `staging` and `production`. A mistake is fixed by a new migration. Down migrations, if the tool writes them, run only in `dev` and CI.
- M-2 Expand, then contract (`docs/launch-runbook.md` 5.1). A migration never breaks the server version currently deployed, so the binary rolls back without touching the schema. A contract step ships in a later release, after the previous version is no longer deployed anywhere.
- M-3 A migration is never edited after it merged; its version number is unique and ordered.
- M-4 Every migration PR passes the migrations suite of `docs/test-strategy.md` 2.3 in CI: up from empty to head on a fresh database, and the previous server version working on the new schema. Before a release the expand step is drilled on a staging copy (`docs/test-strategy.md` RG-6).
- M-5 Migrations run once per deploy, from one runner holding the tool's lock, before nodes of the new version take traffic; server nodes never migrate on start concurrently (#25). The runner is the `migrate` Job of `deploy/k8s/base/migrate.yaml`, started from the release image by `tools/deploy/apply-release.sh` before the rollout (`docs/launch-runbook.md` 5.1).
- M-6 Order per release: `staging` first, a deploy drill on it, then `production` with the same migration files.
- M-7 The migration role may change the schema; the server's runtime role may not. Each has its own secret per environment.
- M-8 Ledger, purchase and inventory rows are never corrected by a migration or direct SQL; corrections are ledger entries through admin tools (`docs/launch-runbook.md` 5.4, #144).
- M-9 `matches`, `match_participants` and `match_events` are partitioned by month of `started_at`. `match_partitions_ensure(from, months)` creates the months ahead and is idempotent; a deploy or a scheduled job calls it for the coming six months, and a `_default` partition catches any row outside the created months so a write never fails. A partition is created before rows for its month exist.
- M-10 A match result is written once, in one transaction: the `matches` row is closed, participant results, the replay and the event log are stored, and one `outbox` row per downstream topic (`rating`, `xp`, `missions`, `economy`, `analytics`) is inserted with the key `match:<id>`. A second finish of the same match is refused, so a crash between the write and the publish never scores a match twice; the relay publishes pending outbox rows at least once and consumers deduplicate by key (`server/internal/matchstore`, #50).

## 7. Open items

- Infrastructure as code: the tool and state location are fixed by ADR 0010; the OpenTofu code needs the Hetzner Cloud project and its API token per environment, which the owner creates. GN-8 evidence (IaC apply log) depends on it.
- Staging automatic deploy (acceptance criterion of #24) needs the staging cluster above; the image it deploys is pushed by `.github/workflows/server-image.yml` (`docs/launch-runbook.md` 5.1).
- The GitHub environments `staging` and `production`, their protection rules and their secrets are created by a repository admin when the first deploy exists (#25).
- Encryption at rest of PostgreSQL volumes and backups is a hosting setting delivered with the IaC (#25, #188, `docs/privacy.md` 9.1).
- Migration tool: #55.
- Crash reporting (#182): the Sentry account in the EU data region, its project and one DSN per environment are created by the owner. Until `HD_SENTRY_DSN` is set, panics of the server and the admin API are only logged and counted; until `CLIENT_SENTRY_DSN` is set, release builds send no crash reports. The CI token for the IL2CPP symbol and mapping upload is also the owner's (`docs/launch-runbook.md` OI-6).
- Client HTTP timeout: no document or config sets the timeout of the player's `/auth` and API requests, so `-httpTimeoutSeconds` (2.5) and `CLIENT_HTTP_TIMEOUT_SECONDS` (2.3) have no value. A tag release fails before the build until the owner sets it (#66).
