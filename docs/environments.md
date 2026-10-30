# Environments, secrets and migrations

The environments the game runs in, every environment variable the code reads, where secrets live and how they are rotated, the secret scan, and how database schema changes reach each environment (#24).

Scope boundaries: deploy, rollback and orchestration are #25 and `docs/launch-runbook.md` section 5. Backups and restore are #188. Observability is #28. Admin config (every number a designer tunes) is never an environment variable; it is versioned in admin (ADR 0006).

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

Hosting: the primary data region is Germany (EU) with Hetzner as hosting provider (`docs/privacy.md` section 2, OQ-3). Orchestration is ADR 0010: k3s on Hetzner Cloud in Germany, one cluster per environment, manifests in `deploy/k8s`, infrastructure as code with OpenTofu and its state in a Hetzner Object Storage bucket per environment.

## 2. Environment variables

The server reads its configuration from the environment only, with the `HD_` prefix. A variable with a default is optional; a bad value stops the start with every bad variable named and exit code 2. A secret is never given a default and never logged (`docs/privacy.md` P-9).

Today no code on `main` reads an environment variable. The server variables below come from the open #264 (service skeleton, #43) and are marked so; when it merges, `server/README.md` holds the same table and the two stay in sync.

### 2.1 Server runtime

| Name | Required | Service | Default | Example placeholder | Secret | Source |
|---|---|---|---|---|---|---|
| `HD_LISTEN_ADDR` | No | server | `:8080` | `:8080` | No | from #264 |
| `HD_LOG_LEVEL` | No | server | `info` | `info` | No | from #264 |
| `HD_ORIGINS` | No | server | none | `play.example.com,admin.example.com` | No | from #264 |
| `HD_SHUTDOWN_TIMEOUT` | No | server | `10s` | `10s` | No | from #264 |
| `HD_DRAIN_TIMEOUT` | No | server | `15m` | `15m` | No | from #264 |
| `HD_MIN_APP_VERSION` | No | server | `1.0.0` | `1.0.0` | No | from #264 |
| `HD_CATALOG_VERSION` | No | server | empty | `<catalog-version>` | No | from #264 |
| `HD_MAX_FRAME_BYTES` | No | server | `1024` | `1024` | No | from #264 |
| `HD_WRITE_QUEUE` | No | server | `32` | `32` | No | from #264 |
| `HD_HELLO_TIMEOUT` | No | server | `5s` | `5s` | No | from #264 |
| `HD_WRITE_TIMEOUT` | No | server | `5s` | `5s` | No | from #264 |
| `HD_CLOSE_TIMEOUT` | No | server | `2s` | `2s` | No | from #264 |
| `HD_COMPRESSION_THRESHOLD` | No | server | `256` | `256` | No | from #264 |
| `HD_INTENTS_PER_SECOND` | No | server | `10` | `10` | No | from #264 |
| `HD_INTENT_BURST` | No | server | `20` | `20` | No | from #264 |
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
| `HD_COSMETICS_MANIFEST` | No | server | empty (built-in manifest) | `/etc/hypedomino/cosmetics.json` | No | from #61 |
| `HD_EMOJI_SLOTS` | No | server | `6` | `6` | No | from #61 |
| `HD_ACTION_SLOTS` | No | server | `4` | `4` | No | from #61 |
| `HD_RARITY_ORDINALS` | No | server | `common:1,rare:2,epic:3,legendary:4,mythic:5` | same | No | from #98 |
| `HD_SHOP_CATALOG` | No | server | empty (shop closed until a publish) | `/etc/hypedomino/shop.json` | No | from #104 |
| `HD_SHOP_DAILY_SLOTS` | No | server | `4` | `4` | No | from #104 |
| `HD_SHOP_WEEKLY_SLOTS` | No | server | `2` | `2` | No | from #104 |
| `HD_SHOP_RESET_HOUR_UTC` | No | server | `0` | `0` | No | from #104 |
| `HD_CONFIG_POLL` | No | server | `10s` | `10s` | No | from #104 |
| `HD_SEASON_POLL` | No | server | `1m` | `1m` | No | from #115 |
| `HD_DUPLICATE_COMPENSATION_PERCENT` | No | server | `20` | `20` | No | from #104 |
| `HD_PLAY_PACKAGE` | No | server | empty (Google purchases off) | `com.example.hypedomino` | No | from #106 |
| `HD_PLAY_SERVICE_ACCOUNT` | With `HD_PLAY_PACKAGE` | server | empty | `/run/secrets/play-service-account.json` | Yes, the file it names | from #106 |
| `HD_PLAY_NOTIFICATION_TOKEN` | With `HD_PLAY_PACKAGE` | server | empty | `<random-32-bytes>` | Yes | from #106 |
| `HD_APPLE_BUNDLE_ID` | No | server | empty (Apple purchases off) | `com.example.hypedomino` | No | from #106 |
| `HD_STORE_SANDBOX` | No | server | `false` (`true` in dev and staging) | `false` | No | from #106 |
| `HD_REFUNDS_BEFORE_PURCHASE_BLOCK` | No | server | `3` | `3` | No | from #106 |
| `HD_REFUND_WINDOW_DAYS` | No | server | `365` | `365` | No | from #106 |
| `HD_PROMO_KEY` | No | server | empty (promo redemption off, 503 `promo.unavailable`) | `<random-32-bytes>` | Yes | from #108; `cmd/promo` needs the same key |
| `HD_PROMO_CODE_LENGTH` | No | server | `12` | `12` | No | from #108 |
| `HD_PROMO_MAX_VALIDITY_DAYS` | No | server | `90` | `90` | No | from #108 |
| `HD_PROMO_ATTEMPTS_PER_HOUR` | No | server | `10` | `10` | No | from #108 |
| `HD_PROMO_FAILED_ATTEMPTS_BEFORE_LOCK` | No | server | `5` | `5` | No | from #108 |
| `HD_PROMO_LOCK_MINUTES` | No | server | `60` | `60` | No | from #108 |
| `HD_PROMO_COINS_APPROVAL_THRESHOLD` | No | server | `2000` | `2000` | No | from #108 |

The gateway limits and the reconnect grace are environment variables only until admin config serves them (#210, `docs/penalties.md` section 8); then the admin value wins and the variable is removed in the same PR.

### 2.2 CI

| Name | Required | Service | Example placeholder | Secret | Stored as |
|---|---|---|---|---|---|
| `UNITY_LICENSE` | Yes, for the `client` job | CI, Unity test runner | `<unity-license-file-contents>` | Yes | Repository secret |
| `UNITY_EMAIL` | Yes, for the `client` job | CI, Unity test runner | `<unity-account-email>` | Yes | Repository secret |
| `UNITY_PASSWORD` | Yes, for the `client` job | CI, Unity test runner | `<unity-account-password>` | Yes | Repository secret |

### 2.3 Not read yet

The admin panel and the client read no environment variable. Crash reporting (#182) adds its variables when the code that reads them lands. That PR adds the row here with all columns filled and the secret to section 3 for each environment.

## 3. Where secrets live

| Secret use | Lives in | Reaches the consumer as |
|---|---|---|
| CI only (Unity license) | GitHub Actions repository secrets | Job environment variables, masked in logs |
| `staging` runtime | GitHub Actions environment secrets of the `staging` environment | Environment variables injected by the staging deploy (#25) |
| `production` runtime | GitHub Actions environment secrets of the `production` environment, deployable only from protected refs | Environment variables injected by the production deploy (#25) |
| `dev` | A local `.env` file, ignored by `.gitignore` | The local stack (#22) |
| Redis password (`HD_REDIS_URL`) | The `server-env` secret of each cluster (ADR 0010), from the environment secrets above | Environment variable of the server pods |
| Database runtime role password (`HD_DATABASE_URL`) | The `server-env` secret of each cluster, from the environment secrets above; the migration role (M-7) is a separate secret of the migration runner | Environment variable of the server pods |
| Access token signing keys (`HD_TOKEN_KEYS`) | The `server-env` secret of each cluster; generated with `openssl rand -base64 32` per key id, never derived from a password | Environment variable of the server pods; rotated with overlap (section 5 step 3) |

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
- M-5 Migrations run once per deploy, from one runner holding the tool's lock, before nodes of the new version take traffic; server nodes never migrate on start concurrently (#25).
- M-6 Order per release: `staging` first, a deploy drill on it, then `production` with the same migration files.
- M-7 The migration role may change the schema; the server's runtime role may not. Each has its own secret per environment.
- M-8 Ledger, purchase and inventory rows are never corrected by a migration or direct SQL; corrections are ledger entries through admin tools (`docs/launch-runbook.md` 5.4, #144).
- M-9 `matches`, `match_participants` and `match_events` are partitioned by month of `started_at`. `match_partitions_ensure(from, months)` creates the months ahead and is idempotent; a deploy or a scheduled job calls it for the coming six months, and a `_default` partition catches any row outside the created months so a write never fails. A partition is created before rows for its month exist.
- M-10 A match result is written once, in one transaction: the `matches` row is closed, participant results, the replay and the event log are stored, and one `outbox` row per downstream topic (`rating`, `xp`, `missions`, `economy`) is inserted with the key `match:<id>`. A second finish of the same match is refused, so a crash between the write and the publish never scores a match twice; the relay publishes pending outbox rows at least once and consumers deduplicate by key (`server/internal/matchstore`, #50).

## 7. Open items

- Infrastructure as code: the tool and state location are fixed by ADR 0010; the OpenTofu code needs the Hetzner Cloud project and its API token per environment, which the owner creates. GN-8 evidence (IaC apply log) depends on it.
- Staging automatic deploy (acceptance criterion of #24) needs the staging cluster above and the release workflow that pushes the image (#27).
- The GitHub environments `staging` and `production`, their protection rules and their secrets are created by a repository admin when the first deploy exists (#25).
- Encryption at rest of PostgreSQL volumes and backups is a hosting setting delivered with the IaC (#25, #188, `docs/privacy.md` 9.1).
- Migration tool: #55.
