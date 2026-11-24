# Launch checklist and runbook

How a release goes from candidate to players, and what happens when it goes wrong: the go/no-go checklist, the launch timeline, deploy, staged rollout and rollback procedures, incident management, on-call, player communication and support. It applies to the soft launch (#196), to the global launch and, from section 5 on, to every later release.

Scope boundaries: `docs/roadmap.md` section 4 is the release gate and is not restated here; this document adds the operational evidence needed to pass it. Budgets are in `docs/performance.md`. Kill switches, the build allowlist and ban responses are in `docs/anti-cheat.md` and #210. Architecture is ADR 0002 to 0006.

The repository holds the tooling; the hosted environments do not exist yet. In the repository: the server image and the `deploy/k8s` base manifests with the drain-aware rolling deploy and rollback measured by the `image` CI job (ADR 0010, #25), the backup manifests and the local restore drill of the `backup-drill` CI job (#188), the Android `Release build` workflow (#27), the content upload tool `content-publish` (#26), structured logs and Prometheus metrics (#28) and the admin support tools (#144). Not yet: the staging and production clusters, their overlays and OpenTofu code, the content bucket and CDN, dashboards, a Prometheus and Alertmanager to load the alert rules of `deploy/k8s/monitoring`, pager, status page, crash reporting, the iOS pipeline and the store accounts. A step that needs one of these names it as owner input `OI-<n>` (section 10); engineering work still missing is in section 11. Numbers marked *default* are starting values: operational ones change by PR to this file, game ones are admin config (ADR 0006). Roles, never names; one person may hold several roles, except where a section forbids it.

## 1. Roles

| Role | Owns |
|---|---|
| Product owner | Final go/no-go decision, waivers (an owner comment on the issue concerned, `docs/roadmap.md` section 4), player compensation decisions |
| Release manager | This checklist, the release issue, the timeline, store consoles (submission, rollout, halt) |
| Server lead | Server and data deploys, server rollback, capacity, database restore (#188) |
| Client lead | Release builds, client hotfixes, forced minimum version |
| Live-ops lead | Production admin config, catalog, banners, maintenance mode, kill switches (#145, #146, #210) |
| QA lead | Release candidate regression and device smoke (#190, #191) |
| Security lead | Red team result (#212), anti-cheat config, security incidents, secret rotation |
| Support lead | Support channel, FAQ, templates, store review replies, ticket-to-incident escalation |
| Analyst | Health and KPI dashboards during launch (#147, #181, #196) |

Incident roles (section 6) are assigned per incident: incident commander, comms lead, scribe, responders.

## 2. Release record

Every release has one GitHub issue labelled `area:release` that links its release candidate commit and tag. The release manager copies section 3 into it, links each item's evidence, and records the go/no-go decision as an owner comment. A release without this issue does not ship.

## 3. Go/no-go checklist

Rules: every item is green, or waived by an owner comment on the issue concerned. Missing or unrecorded evidence is red. Evidence is recorded against the release candidate commit or the production environment it ships to, not an earlier build.

### 3.1 Release gate (`docs/roadmap.md` section 4)

| Id | Item | Owner | Evidence |
|---|---|---|---|
| GN-1 | Gate 1: no open `priority:p0` or `priority:p1` issue | Release manager | Output of both `gh issue list` commands from the gate, pasted in the release issue |
| GN-2 | Gate 2: performance budgets pass on the device matrix | Client lead, Server lead | Measurement reports in #191, #192, #161, #164 per `docs/performance.md` |
| GN-3 | Gate 3: content gate | Live-ops lead | `green`, non-stale production content gate report run after the config freeze starts, or a `red` one with an owner waiver on #198 (`docs/launch-content-gate.md` 4.3, 4.4) |
| GN-4 | Gate 4: legal documents published and linked | Product owner | Links open from the store pages and in-app settings (#195, #187); AI art terms and store disclosure rules reviewed (`docs/asset-workflow.md` O-2 to O-4) |
| GN-5 | Gate 5: red team has no open critical finding | Security lead | #212 report and the result table of `docs/red-team.md` (gate in its section 6) |
| GN-6 | Gate 6: this checklist green; backup and restore exercised; closed beta and soft launch KPI tracking done or waived | Release manager | This list, #188 drill record, #193 closed or waived after its exit record (`docs/closed-beta.md` 7.1), #196 closed or waived |
| GN-7 | Gate 7: `CI complete` green on the release commit | Release manager | Link to the run on the tagged commit |

### 3.2 Operational readiness

| Id | Item | Owner | Evidence | Link |
|---|---|---|---|---|
| GN-8 | Production environment separate from staging (`docs/environments.md` E-1 to E-6); secrets only in the `production` environment secrets (section 3 there); secret scan green | Server lead | OpenTofu apply log of the production environment (ADR 0010), owner input OI-1; `secret-scan` job green in the `CI complete` run of the release commit | #24, `docs/environments.md` |
| GN-9 | Server deploy drill on staging: matches in progress finish or move during a deploy; rollback measured under 5 minutes | Server lead | Drill record with timings; needs owner input OI-1. Until then the `image` CI job measures drain and rollback on kind (`tools/deploy/kind-rollout.sh`) without live matches | #25, #43 |
| GN-10 | Catalog switch drill: previous Addressables catalog re-activated; an older client still loads with its catalog | Live-ops lead | Drill record on staging with timings: two catalog versions uploaded with `content-publish` (5.3), the newer one active, then the older one re-activated with the content step of 5.4; the `catalog_version` an older client build loads afterwards. Needs owner inputs OI-1 and OI-3 | #26, ADR 0002, `docs/content-delivery.md` 8 |
| GN-11 | Release builds are produced and signed by the pipeline from the tag; version and build numbers recorded; symbols and mapping files uploaded | Client lead | Run of the `Release build` workflow (`.github/workflows/release-build.yml`) for the tag `client-v<major>.<minor>.<patch>`: version from the tag, build number is the run number, catalog version `<version>-<run number>` in the run summary, `android-release-<catalog version>` artifact with the AAB and the install size report, signed AAB on the Play internal testing track; build numbers copied into the release issue. Needs owner input OI-4. The iOS build and TestFlight upload (OI-5) and the symbol and mapping upload (OI-6) are not built yet | #27, #182, `client/README.md` Release build |
| GN-12 | Dashboards live in production; alerts route to the on-call pager; one test alert reached the primary | Server lead | Screenshot of the test page and its acknowledgement; needs owner inputs OI-7, OI-8 and OI-9 | #28, #566 |
| GN-13 | Crash reporting shows release health per app version; a test crash from the release build is visible | Client lead | Dashboard link; needs owner input OI-6 | #182 |
| GN-14 | Funnel events (install, login, first match, purchase) visible on the launch dashboard | Analyst | Dashboard link | #181, #147, #196 |
| GN-15 | Backup restore drill done; RTO and RPO recorded | Server lead | Drill record | #188 |
| GN-16 | Rate limits and WebSocket abuse protection active in production | Server lead | Config dump and a test against staging | #184 |
| GN-17 | Kill switch drill in production: purchase and ranked switches each off and on, a test build blocked and unblocked, each under 5 minutes; the forced-update screen shown on a device | Live-ops lead, Client lead | Drill record with timings, screen recording | #210, #73, ADR 0004 |
| GN-18 | Maintenance mode and an in-game banner shown on a device in all four launch languages | Live-ops lead | Screen recordings | #146, `docs/localization.md` |
| GN-19 | Production config reviewed and published, including every value in `docs/anti-cheat.md` section 7; config rollback drill (publish the previous version) applies to new matches | Live-ops lead, Security lead | Config version id, audit log entries | #145, ADR 0006 |
| GN-20 | Admin accounts least privilege; audit log on; hardening passed | Security lead | Account and role list, #148 report | #134, #148 |
| GN-21 | Support tools work in production: player lookup, grant with reason, ban and mute; moderation and appeal queue staffed | Support lead | Audit log entries of test actions on a staff account: lookup at `/players`, grant, coin adjustment, ban and mute from `/players/{id}`, the second approval at `/approvals` (`docs/admin.md` 6.1, 6.2, section 8) | #144, #186 |
| GN-22 | Store products match admin shop config, and the country prices in both stores match the published `store_prices` table: the push plan of the Store prices page shows no `change`, `error` or `failed` row for either store, or, without store credentials, the consoles match the CSV export (`docs/economy.md` 7.4, `docs/admin.md` 5.6), Azerbaijan and Turkey at the `mid` tier; a licensed test purchase on the release build is granted once and restores | Live-ops lead | Ledger entry, restore record, store push plan | #106, #107, #139, #204, #581, #585 |
| GN-23 | Store listings complete, age ratings and data safety forms submitted, release approved and held for manual release | Release manager | Console screenshots; pre-submission checklist ST-1 to ST-23 of `docs/store-listing.md` section 9 checked | #194, #187 |
| GN-24 | Release candidate regression passed; smoke passed on every device tier | QA lead | Test run report per `docs/test-strategy.md` section 8 | #190, #191 |
| GN-25 | On-call schedule published from T-1d to T+14d; contact sheet complete; escalation path tested end to end | Release manager | Schedule link, test page record | section 7 |
| GN-26 | Support channel live; FAQ and templates published in all four languages | Support lead | Links, translation review | sections 8, 9 |
| GN-27 | Status page live with incident and maintenance components | Support lead | Link; owner input OI-11 | section 8 |
| GN-28 | At least two people hold release, halt and pause rights in both store consoles | Release manager | Console user lists | section 5 |

## 4. Launch timeline

T-0 is the moment the release is made available in the stores. Server and config changes ship before T-0; the client never depends on a server change that is not already deployed everywhere (ADR 0004: additive protocol within a major version).

| When | Action | Owner |
|---|---|---|
| T-7d | Release candidate tagged from `main`; only fixes for release-blocking issues merge after this. Builds submitted for review with manual release (App Store) and managed publishing (Play) so approval does not publish. First full pass of section 3; open items get owners and dates in the release issue | Release manager |
| T-7d | Load and soak test re-run against the production config version (#192) | Server lead |
| T-5d | Support templates, FAQ and banner texts sent for translation (`docs/localization.md`) | Support lead |
| T-3d | Server release deployed to production (section 5.1). Go/no-go dry run: every item has evidence or a named blocker | Server lead, Release manager |
| T-3d | Config freeze: production admin publishes only through the release manager until T+72h, except incident mitigation | Live-ops lead |
| T-3d | Content gate report run in production (GN-3); run again after any later content or config publish, since that makes it stale | Live-ops lead |
| T-1d | Go/no-go meeting. Product owner decides go, no-go or slip; decision recorded in the release issue. War room channel opened, on-call launch shifts start | Product owner |
| T-1d | Backups verified fresh; production smoke with the store-approved build | Server lead, QA lead |
| T-2h | Final health check of all dashboards; status page set to operational; on-call primary and secondary confirmed online | Release manager |
| T-0 | Release made available in both stores; announcement published | Release manager, Support lead |
| T+15m | First installs, logins and matches visible on the launch dashboard | Analyst |
| T+1h | Health review against section 4.1; then hourly until T+6h | Release manager |
| T+24h | Daily review: section 4.1 signals, support ticket themes, store reviews, KPIs. Decide whether to continue, hold or widen availability (#196) | Product owner |
| T+48h | Daily review | Product owner |
| T+72h | Launch exit: back to the normal on-call rotation, config freeze lifted, retrospective scheduled, this runbook updated with what was learned | Release manager |

### 4.1 Launch health signals

| Signal | Source | Threshold |
|---|---|---|
| Crash-free sessions per app version | #182 | At least 99.5% (the #193 beta exit level, `docs/closed-beta.md` BX-1) and not below the previous version |
| Move round-trip p99 | `move_roundtrip_seconds` on the server's `/metrics` (#28) | Under 100 ms (`docs/performance.md`); alert `GameNodeMoveRoundTripOverBudget` (`deploy/k8s/monitoring/server-latency.rules.yaml`) after 15 minutes over, see 5.6 |
| Error rate, database connections | `hd_http_responses_total{class}`, `hd_db_connections_*` and `hd_db_acquire_*` on `/metrics` (`server/README.md`, #28) | Alert thresholds: owner input OI-8 |
| Login success | `hd_signins_total{provider,result}` on `/metrics` (`server/README.md`, #643); success rate per provider is `result="ok"` over all results | Alert thresholds: owner input OI-8 |
| Matchmaking wait | `hd_matchmaking_wait_seconds{queue,mode}` and `hd_matchmaking_left_total` on `/metrics` (#643). The server has no matchmaking timeout (`docs/matchmaking.md`), so players who give up show as queue leaves | Alert thresholds: owner input OI-8 |
| Receipt validation success per store | `hd_receipt_validations_total{store,result}` and `hd_receipt_validation_seconds{store}` on `/metrics` (#106, #643) | Alert thresholds: owner input OI-8 |
| Android vitals crash and ANR rates | Play Console | Below the console's bad behaviour thresholds |
| Support tickets per symptom | Support tool | Section 9.3 escalation rule |

A threshold breach with player impact is an incident (section 6).

## 5. Procedures

### 5.1 Server deploy

1. The release is a tagged commit with `CI complete` green; the image is built once and promoted from staging to production unchanged: `ghcr.io/alievs-corp/hypedomino-server:<sha>`, holding `/server` and `/migrate` (ADR 0010). The workflow `.github/workflows/server-image.yml` builds `server/Dockerfile` for linux/amd64 and linux/arm64 on every push to `main` that changes `server/`, and on demand (`workflow_dispatch` on `main`); it runs the container smoke test on both platforms (arm64 under QEMU), then pushes `ghcr.io/alievs-corp/hypedomino-server:<full commit sha>` (immutable) and the moving tag `main` with `GITHUB_TOKEN` (`packages: write`) as one multi-platform image, so x86 (CX, CCX) and arm64 (CAX) nodes pull the same tag (ADR 0014); nothing is pushed from a pull request or a laptop. Pin the SHA in the overlay with `cd deploy/k8s/overlays/<environment> && kustomize edit set image ghcr.io/alievs-corp/hypedomino-server=ghcr.io/alievs-corp/hypedomino-server:<full commit sha>` and commit the result; the base names the image `ghcr.io/alievs-corp/hypedomino-server` with no tag, and a manifest never uses `main` or `latest`. If the package stays private, the clusters need an image pull secret for `ghcr.io`. Then run from the repository root `bash tools/deploy/apply-release.sh deploy/k8s/overlays/<environment> <namespace>`. It runs the migration `Job` to completion before the rollout, and a failed job stops the deploy before the server is touched. Never `kubectl apply -k` the overlay directly: that starts the job and the rollout at once. The script does, and by hand is:
   ```bash
   kubectl kustomize deploy/k8s/overlays/<environment> > release.yaml
   kubectl -n <namespace> delete job/migrate --ignore-not-found --cascade=foreground --wait=true
   kubectl apply -f release.yaml -l 'app.kubernetes.io/name!=server'
   kubectl -n <namespace> wait --for=condition=complete job/migrate --timeout=960s
   kubectl -n <namespace> logs job/migrate
   kubectl apply -f release.yaml -l 'app.kubernetes.io/name!=migrate'
   kubectl -n <namespace> rollout status deployment/server
   ```
   The job (`deploy/k8s/base/migrate.yaml`) runs `/migrate up` from the release image with `HD_DATABASE_URL` of the migration role from the `migrate-env` secret (`docs/environments.md` M-7), as user 65532 like the server. Jobs are immutable, so it keeps the fixed name `migrate` and each deploy deletes the previous one first; one namespace has at most one migration job, its pods and logs stay 24 hours (`ttlSecondsAfterFinished`) or until the next deploy. Limits: 3 attempts (`backoffLimit: 2`) and 15 minutes in total (`activeDeadlineSeconds: 900`), which covers the 5 minutes `migrate` waits for the advisory lock of another runner; a migration expected to run longer raises the deadline and the `wait` timeout in its PR. `migrate up` on an up-to-date schema applies nothing, so running a deploy again is safe. If the job fails, read its log, fix forward with a new migration (M-1) and deploy again; the server keeps running the previous release.
2. Migrations follow expand then contract: a migration never breaks the previous server version, so the binary can roll back without touching the schema (ADR 0005). A contract step ships only after the previous version is no longer deployed. Migrations are forward only and run once, staging first (`docs/environments.md` section 6).
3. Canary: new rooms are routed to one new node first, *default* 10% of new rooms for 30 minutes, while rooms in progress stay on old nodes (ADR 0003: a room lives on one node).
4. Watch section 4.1 on the canary. On a breach, stop and roll back (5.4).
5. Roll the remaining nodes: new rooms go to new nodes, old nodes drain until their rooms finish, then stop (#25, #43). Never kill a node with live rooms.
6. Record the deployed version and config version in the release issue.

Order with a client release: server first (backward compatible with every client at or above the minimum version), then client.

### 5.2 Client rollout

First store release: percentage rollouts are not used. Exposure is limited by country availability (soft launch regions, #196) and by server-side controls (feature flags, matchmaking, kill switches).

Every later update:

| Store | Mechanism | Default plan |
|---|---|---|
| Google Play | Staged rollout by percentage, halt and resume in Play Console | *Default* 1%, 5%, 20%, 50%, 100%, at least 24 hours per step |
| App Store | Phased release over 7 days with a schedule fixed by Apple (1%, 2%, 5%, 10%, 20%, 50%, 100% of users with automatic updates); can be paused, total pause up to 30 days, or released to everyone early | Phased release on; release to all early only after the Play rollout reaches 50% without incident |

Phased release covers automatic updates only: new installs and manual updates get the new version at once. Feature flags (#145) are the real limit for risky features on iOS.

Advance a step only when, for the new version: crash-free sessions meet section 4.1, no open SEV1 or SEV2 is caused by it, and support tickets show no new symptom for it. Otherwise hold; on a SEV1 or SEV2 caused by the version, halt (Play) and pause (App Store) at once.

### 5.3 Config and content publish

- Admin config is published as a new version; matches started after the publish use it and every match records its version (ADR 0005, ADR 0006). Publishing is audited (#134).
- Content goes live by activating a new Addressables catalog version (ADR 0002, #26), in two steps (`docs/content-delivery.md` 8):
  1. Upload. The `Release build` workflow runs `content-publish` for a `client-v*` tag when the repository variable `CONTENT_STORE_URL` is set (owner input OI-3). By hand, from `server/` with the `HD_CONTENT_STORE_*` variables of the target environment set: `go run ./cmd/content-publish -dir ../client/ServerData/Android -environment <environment> -platform Android -catalog-version <catalog_version> -dry-run`, then the same without `-dry-run`. It uploads bundles before the catalog and refuses a version that already exists.
  2. Activate. Set `HD_CATALOG_VERSION` to the uploaded version as in the content step of 5.4. Clients load it after their next `ServerHello` (`docs/content-delivery.md` CD-5).
  3. Collect unreferenced bundles (`content-gc`, CD-33 to CD-35), only after the minimum app version has risen (5.5) and never while `content-publish` runs for the same bucket. Per platform, from `server/` with the `HD_CONTENT_STORE_*` variables of the target environment set (a dry run needs them too): `go run ./cmd/content-gc -environment <environment> -platform <Android|iOS> -active-catalog-version <catalog_version> -min-app-version <version>`. The two versions default to `HD_CATALOG_VERSION` and `HD_MIN_APP_VERSION`; pass the values the environment announces now, and for the minimum the `minimum_app_version` of the published `live_controls` snapshot. Nothing is deleted without `-delete`. Review the dry run: every catalog marked `in scope` or `unsupported` with its reason, the `missing` lines (a bundle a supported catalog needs that the store lacks: stop and find out why), and the `unreferenced` list against what you expect to go. It refuses to run when the active catalog is not in the store or a catalog in scope cannot be read. Then run the same command with `-delete`; a bundle younger than `-grace` (default `24h`) is kept. Record the counts in the release issue.
- During the config freeze (section 4) every publish is approved by the release manager and noted in the release issue.

### 5.4 Rollback

Mitigate first, find the root cause after. Pick the fastest step that stops the harm.

| What broke | Step | Target | Notes |
|---|---|---|---|
| Server release | `kubectl rollout undo deployment/server`, or deploy the previous SHA through 5.1; same drain as 5.1 (ADR 0010) | Under 5 minutes, measured by the `image` CI job and drill GN-9 (#25) | Works because migrations are expand only during a release. Never roll back below the version a live client needs; fix forward instead |
| Database data | Restore from point-in-time recovery | RTO and RPO from #188 | Only for corruption or loss, decided by the incident commander with the product owner. Ledger mistakes are corrected with ledger entries, never by restore or direct SQL (#101, #144) |
| Config | Publish the previous config version in admin | Under 5 minutes | Applies to new matches; matches in progress finish on their recorded version |
| Content | Re-activate the previous catalog version: `kubectl patch secret server-env --type merge -p '{"stringData":{"HD_CATALOG_VERSION":"<previous catalog_version>"}}'`, then `kubectl rollout restart deployment/server` and `kubectl rollout status deployment/server`; the same value goes into the environment's secret source (`docs/environments.md` section 3) so the next deploy keeps it | Under 5 minutes, measured by drill GN-10 | Old content URLs stay valid because files are immutable (ADR 0002). Only versions at or above the minimum app version are rollback targets: after a `content-gc` run (5.3) a catalog below it may have lost bundles. The restart drains like 5.1; a client on a draining pod gets the version on its next connection (`docs/content-delivery.md` CD-8) |
| Client feature | Turn off its feature flag or kill switch (5.5) | Under 5 minutes (#210) | First choice for any client-side defect |
| Client build | Halt the Play rollout and pause the App Store phased release | Immediate in the consoles | Players who already updated keep the version |
| Client build, no flag helps | Ship a hotfix with a higher version and build number, request expedited App Store review; then raise the minimum version once the fix is available in both stores | Hours to days, store dependent | Raising the minimum version before a fixed build exists locks those players out |
| Leaked, modded or compromised build | Remove its build hash from the allowlist or raise the minimum version | Under 5 minutes (#210) | Security only; players on it see the forced-update screen (ADR 0004) |

Store builds cannot be withdrawn from devices. That is why every client feature with risk ships behind a flag.

### 5.5 Kill switches and server-side controls

| Control | Effect | Changed by | Defined in |
|---|---|---|---|
| Purchase kill switch | Store purchases disabled in the shop; already validated receipts are still granted | Live-ops lead or incident commander, with the `live_ops` or `super_admin` admin role and the two-person rule of `docs/admin.md` section 8 | #210 |
| Ranked kill switch | Ranked queues closed; matches in progress finish | Same | #210 |
| Minimum version and build allowlist | Older or unlisted builds get `UPDATE_REQUIRED` | Same, with the client lead | #210, ADR 0004 |
| Maintenance mode | Client shows the maintenance screen; no new matches | Same | #146 |
| Feature flags | Any flagged feature off or on without a release | Same | #145 |
| Promo code disable | One code stops redeeming | Live-ops lead | #142, `docs/anti-cheat.md` V-11 |

Every change is audited with a reason (#134). Switching back follows the same path and is noted in the incident record.

### 5.6 Game node to dedicated vCPU

When: the alert `GameNodeMoveRoundTripOverBudget` fires (p99 of `move_roundtrip_seconds` over 5 minutes above 100 ms for 15 minutes, at more than one move per second) and rolling back the latest release (5.4) does not clear it, or the beta run of `docs/performance.md` 6.1 fails its move round-trip gate (ADR 0014). It is a SEV2 incident (6.1) until the new node holds the budget. Before moving, look at the node: CPU steal (`st` of `vmstat 10` on the node) of several percent while the server's own CPU is moderate points at a neighbour, which this step fixes; high CPU of the server process points at load or a regression, which it only buys time for.

Steps, by the server lead, no maintenance window needed:

1. Add a node of the target type (CCX23, `docs/performance.md` 6.1) with the label `hypedomino.io/role=game` through OpenTofu, joined to the cluster as a k3s agent; wait until `kubectl get node <new>` is `Ready`.
2. `kubectl cordon <old game node>`: no new pod lands there, live rooms keep running.
3. `kubectl -n <namespace> rollout restart deployment/server` and `kubectl -n <namespace> rollout status deployment/server`: new pods start on the new node and become ready before old pods drain (ADR 0010), so no live match ends. The status returns when the last old pod has drained, up to `HD_DRAIN_TIMEOUT` per pod.
4. Watch section 4.1 for 30 minutes; the alert resolves when the 5-minute p99 is back under 100 ms.
5. `kubectl drain <old game node> --ignore-daemonsets --delete-emptydir-data`, `kubectl delete node <old game node>`, then remove the old server from the OpenTofu code and apply.
6. Record the node types, the time and the 4.1 numbers before and after in the incident record and in #566.

Rollback, when the new node does not help or the shared type is wanted back: the same steps with a node of the shared type (ADR 0014) as the new node. An in-place resize (`hcloud server change-type`) is not used: it powers the only game node off, which ends every live room, and it cannot change between CAX (arm64) and CCX (x86).

## 6. Incident management

### 6.1 Severity

| Severity | Definition | Examples |
|---|---|---|
| SEV1 | Most players cannot play, or money, data or security is at risk | Login or matchmaking down; a path that mints currency or items; data loss; leaked secret; mass crash of the current version |
| SEV2 | A large group cannot play or pay, or a budget is broken for everyone | One region or one platform down; purchases failing on one store; p99 move round-trip over budget for 15 minutes (*default*) |
| SEV3 | A feature is broken with a workaround, or a small group is affected | Leaderboard stale; one device model crashing; banner text missing in one language |
| SEV4 | Cosmetic or no player impact | Typo; internal dashboard broken |

When unsure, pick the higher severity and lower it later.

### 6.2 Response targets

*Default* values, changed by PR to this file.

| Severity | Acknowledge | Incident commander named | First public update | Update cadence | Postmortem |
|---|---|---|---|---|---|
| SEV1 | 5 min | 15 min | 30 min | Every 30 min | Required, within 5 business days |
| SEV2 | 15 min | 30 min | 60 min | Every 60 min | Required, within 5 business days |
| SEV3 | Next business day | Not required | Only if players ask | On change | Optional |
| SEV4 | Backlog | Not required | None | None | No |

### 6.3 Roles

| Role | Does | Does not |
|---|---|---|
| Incident commander | Owns the incident: severity, decisions, roles, when to roll back, when it is resolved. The on-call primary until handed over | Debug or type fixes in SEV1 |
| Comms lead | Status page, banner, store replies, internal updates on the cadence in 6.2, using section 8 | Promise times or compensation not approved by the incident commander or product owner |
| Scribe | Timestamped log of signals, decisions and actions in the incident channel; drafts the postmortem timeline | Act on systems |
| Responders | Investigate and mitigate as asked by the incident commander | Change production without telling the incident commander |

In SEV1 the incident commander, comms lead and scribe are three different people.

### 6.4 Flow

1. Detect: alert (#28), crash spike (#182), ticket pattern (9.3), store reviews, internal report.
2. Declare: anyone may declare. Open the incident channel (owner input OI-10) and set the severity.
3. Assign roles (6.3).
4. Mitigate with section 5.4 and 5.5 before the root cause is known.
5. Communicate on the cadence in 6.2 (section 8).
6. Resolve when the signals in 4.1 are back to normal for 30 minutes (*default*); post the resolved update.
7. Postmortem: blameless; timeline, impact (players, duration, money), root cause, what helped, what hurt. Every action item becomes an issue with `type:`, `area:` and `priority:` labels.

### 6.5 Special cases

- Economy integrity (currency or items created outside legal play): purchase kill switch on, evidence preserved, ledger reconciliation (#202). Corrections go through the ledger with a reason through admin tools (#144). Compensation is a product owner decision.
- Security (leaked secret, admin compromise, modded build at scale): security lead joins as responder. Rotate secrets (`docs/environments.md` section 5), block builds (#210), disable affected admin accounts (#134). Personal data breaches follow the notification duties in #187 (GDPR: 72 hours to the supervisory authority).
- Cheating wave: responses follow `docs/anti-cheat.md` section 3; public messages never reveal detection details (#186, #209).
- Store outage (billing or review): nothing on our side to roll back; purchase kill switch if receipts fail, comms only.

## 7. On-call

| Item | Rule |
|---|---|
| Rotation | Weekly, a primary and a secondary, handoff on a fixed weekday with a written note of open issues |
| Coverage | Production server, data, admin config and release consoles. The primary carries the pager (owner input OI-9) |
| Launch window | T-1d to T+72h: primary and secondary both on shift, release manager reachable, shifts no longer than 12 hours (*default*) |
| Acknowledge | Within the section 6.2 target; no acknowledgement in 10 minutes (*default*) pages the secondary |
| Escalation | Primary, then secondary, then the server or client lead for the area, then the product owner. The product owner is told of every SEV1. The security lead joins every security or economy incident |
| Access | Every person on the rotation has production read access, admin roles for 5.5 and the runbook; store console rights per GN-28 |
| Contact sheet | Pager, phone and backup channel for every role, kept outside the systems it is needed to repair (owner input OI-12) |

## 8. Communication

Channels: status page (owner input OI-11), in-game banner and maintenance screen (#146), store review replies, push only for SEV1 lasting more than an hour (*default*, #132). English is the source; every template is published in all four launch languages (`docs/localization.md`) before T-3d. Times are given in UTC.

Rules: state what players see and what to do, not internal causes. No blame on vendors or stores. No time we cannot hold: give the next update time instead. No compensation before the product owner decides. No detection detail.

### 8.1 Status page

| State | Text |
|---|---|
| Investigating | Some players cannot {symptom}. We are looking into it. Next update by {time} UTC. |
| Identified | We found the cause of {symptom} and are fixing it. Next update by {time} UTC. |
| Monitoring | A fix is live for {symptom}. We are watching to make sure it holds. |
| Resolved | {symptom} is fixed since {time} UTC. Thanks for your patience. |
| Maintenance | Planned maintenance from {start} to {end} UTC. Matches cannot start during this time. |

### 8.2 In-game banner and maintenance screen

| Use | Text |
|---|---|
| Degraded | Some players are having trouble with {feature}. We are on it. |
| Purchases paused | The shop is paused for a moment. Purchases you already made are safe. |
| Ranked paused | Ranked play is paused for a moment. Casual matches are open. |
| Maintenance screen | We are making HypeDomino better. Back at {end} UTC. |
| Update required | A new version is ready. Update to keep playing. |

"Purchases you already made are safe" is shown only when the incident commander has confirmed it.

### 8.3 Store review replies

| Case | Text |
|---|---|
| Crash or bug | Thanks for telling us. We are fixing this. Please update to the latest version, and if it still happens, use Settings > Support so we can look at your case. |
| Purchase not received | Thanks for reporting this. Every purchase is checked with the store. Please use Settings > Support > Purchases and include your store order number, and we will sort it out. |
| Ban or penalty | Penalties follow our fair play rules. If you think it was a mistake, please send an appeal from Settings > Support. |
| During an incident | Sorry for the trouble. We know about this issue and are working on it. Follow {status page} for updates. |

## 9. Support

### 9.1 Channels

| Channel | Use |
|---|---|
| In-game support form (Settings > Support) | Primary channel. Attaches player id, app version, build, platform, OS, device model, language and last match id automatically; nothing more personal than #187 allows |
| Support email (owner input OI-13) | Players who cannot open the game |
| Store review replies | Public answers with section 8.3; move the case to the form |

All actions on accounts go through admin tools with a reason in the audit log (#144). Refunds are handled by the stores; revoked purchases are reversed through the ledger (#204).

### 9.2 Ticket categories and first response

| Category | Handled by | First response (*default*) |
|---|---|---|
| Purchase missing or wrong | Support, live-ops lead for ledger corrections | 24 h |
| Account and login | Support, server lead | 24 h |
| Ban, penalty or appeal | Support with the moderation queue (#186) | Per #186 |
| Bug or crash | Support, triaged to an issue | 48 h |
| Privacy and account deletion | Support, product owner | Within the legal deadline (#187, #63) |
| Other | Support | 48 h |

### 9.3 Escalation to on-call

Page the on-call primary when 5 tickets with the same new symptom arrive within 60 minutes (*default*), when any ticket reports currency or items appearing or disappearing, or when a ticket suggests a security or privacy breach.

### 9.4 FAQ structure

One page per category, each answer in all four launch languages and taken from its source document, never written from memory.

Hosting (decided 2026-10-28): the FAQ is one set of web pages, and the game shows the same pages inside the game from its help screen, in the game's language, without sending the player to an outside browser. The web pages also serve as the support URL of the store listings (#194). One source means an answer is changed once and is current in both places.

| Category | Questions it answers | Source |
|---|---|---|
| How to play | Rules, scoring, 365 target, 1v1 and 2v2 | `docs/rules.md` |
| Account and login | Sign-in methods, switching devices, lost account | #56, #57, #58 |
| Purchases | Missing purchase, restore, refunds, pass purchase | #106, #107, #125, #195 |
| Coins and rewards | How coins and rewards are earned, daily rewards, promo codes | #7, #103, #108, #131 |
| Ranked and seasons | Ranks, placement, season reset, leaderboards | `docs/ranking.md` |
| Penalties and fair play | AFK, leaving, bans, appeals | `docs/penalties.md`, #186 |
| Connection | Reconnecting, lag, timer behaviour | #45, #87, `docs/rules.md` R-49 to R-52 |
| Devices and performance | Supported devices, graphics settings, storage | `docs/performance.md`, #157, #159 |
| Privacy and data | Data use, analytics opt-out, account deletion | #187, #63 |
| Languages | Changing the language | `docs/localization.md` |

## 10. Owner inputs before launch

Accounts, addresses, vendors and decisions only the owner can provide. A step above that needs one names it as `owner input OI-<n>`. Credentials are stored as `docs/environments.md` section 3 says, never in this file or an issue; this table names them only.

| Id | Input | Owner role | Issue | Unblocks |
|---|---|---|---|---|
| OI-1 | Hetzner Cloud project and API token per environment (`staging`, `production`) and the game domain; node types, load balancer and bucket names per ADR 0014; then the OpenTofu code, the `staging` and `production` overlays of `deploy/k8s`, ingress and TLS are written against them (ADR 0010) | Server lead, Product owner | #25, #24, #192 | GN-8, GN-9, GN-10, GN-15 to GN-21 on staging and production, the staging step-up check of #144, the reference load and soak run of #192 (`tools/loadtest/hetzner.sh`, `docs/performance.md` 6.1) |
| OI-2 | GitHub environments `staging` and `production` with protection rules and their secrets (`docs/environments.md` section 3, section 7) | Server lead | #24, #25 | Every deploy of section 5.1, secret rotation |
| OI-3 | Content bucket `hypedomino-content-<environment>`, CDN and DNS per environment (`docs/content-delivery.md` CD-27, CD-31); repository variables `CONTENT_STORE_URL`, `CONTENT_STORE_REGION`, `CONTENT_ENVIRONMENT`; secrets `CONTENT_STORE_ACCESS_KEY`, `CONTENT_STORE_SECRET_KEY`; server `HD_CONTENT_BASE_URL` per environment | Server lead | #26 | 5.3, the content step of 5.4, GN-10 |
| OI-4 | Play Console app created by hand; repository variable `ANDROID_APPLICATION_ID`; secrets `ANDROID_KEYSTORE_BASE64`, `ANDROID_KEYSTORE_PASS`, `ANDROID_KEYALIAS_NAME`, `ANDROID_KEYALIAS_PASS`, `GOOGLE_PLAY_SERVICE_ACCOUNT_JSON` (`docs/environments.md` 2.3). The tag path has not run with real secrets yet | Client lead, Release manager | #27 | GN-11 (Android), GN-22, GN-23 |
| OI-5 | Apple developer account: distribution certificate, provisioning profile, App Store Connect API key and a macOS runner for the iOS build and TestFlight upload | Client lead, Release manager | #27 | GN-11 (iOS), GN-22, GN-23 |
| OI-6 | Crash reporting account (EU region), its DSN and a CI token for symbol and mapping upload | Client lead | #182, #566 | GN-11 (symbols), GN-13, section 4.1 crash-free sessions |
| OI-7 | Prometheus in the `monitoring` namespace of each cluster scraping the server debug port (already in the manifests and admitted only from `monitoring`, `server/README.md` Container and deploy) and loading the rule files of the ConfigMap `prometheus-rules-server` (`kubectl apply -k deploy/k8s/monitoring`), and Grafana dashboards in production | Server lead | #28, #566 | GN-12, section 4.1 |
| OI-8 | Alert thresholds for error rate, database connections, login success, matchmaking wait and receipt validation per store (section 4.1) | Server lead, Product owner | #28, #566 | Section 4.1, GN-12 |
| OI-9 | Pager service and alert routing to it, economy alerts (`docs/economy.md` 10) and the backup alerts (`docs/backup-restore.md` 3.3) included; one test alert to the primary | Server lead | #28, #110, #566 | GN-12, GN-25, section 7 |
| OI-10 | Incident channel and launch war room channel | Release manager | #28 | Section 6.4, section 4 T-1d |
| OI-11 | Status page with incident and maintenance components, and its URL | Support lead | #28 | GN-27, section 8, store review replies (8.3) |
| OI-12 | Contact sheet location, outside the systems it is needed to repair | Release manager | #28 | GN-25, section 7 |
| OI-13 | Support email address and a monitored inbox | Support lead | #144 | Section 9.1, `docs/store-listing.md` SF-10 and ST-8, `docs/closed-beta.md` FB-3 |
| OI-14 | Deletion log bucket per environment in the backup location, an access key limited to it, and Secret `account-deletion-env` with `HD_DATABASE_URL`, `HD_DELETION_LOG_URL`, `HD_DELETION_LOG_REGION`, `HD_DELETION_LOG_ACCESS_KEY`, `HD_DELETION_LOG_SECRET_KEY` (`docs/backup-restore.md` 3.7). Until it exists the `account-deletion` CronJob fails and no account past its grace is erased | Server lead | #601, #25 | GN-15, `docs/privacy.md` L-15 and L-19 |

## 11. Open items

- All *default* numbers are unvalidated until the closed beta (#193) and soft launch (#196) produce data.
- The go/no-go checklist turns green only at launch; this document is complete when the release issue records it so.
