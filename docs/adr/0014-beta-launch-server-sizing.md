# 0014. Server sizing for closed beta and launch: shared vCPU, dedicated game node on demand

Status: Accepted
Date: 2026-11-16
Issue: #566

## Context
ADR 0010 fixes the platform (k3s on Hetzner Cloud in Germany, one cluster per environment) and leaves node size to the load test (#192). The load test and `docs/performance.md` 6.1 assumed a dedicated-vCPU CCX23 for every host, which makes the smallest production cluster cost several times what the closed beta (`docs/closed-beta.md`) needs.

The owner decided on 2026-11-16 to run the closed beta and the launch on shared-vCPU servers (the CX line, x86, and the CAX line, arm64) at roughly €6 to €10 per server and month, and to move only the game node to dedicated vCPU (CCX) when latency demands it. Accepted with that decision:
- Shared vCPU can add latency spikes when a neighbour on the same host is busy (CPU steal).
- Some CX and CAX plans have been shown as not available in some locations at times in 2026.
- One game node and one database node means no redundancy. Data stays safe through WAL-G (ADR 0013): WAL and base backups go to a bucket in another Hetzner location than the database.
- 4 to 8 GB of RAM per node is enough for the beta.

Constraints that still hold: Germany only (`docs/privacy.md` section 2, OQ-3); the move round-trip budget p99 < 100 ms in region (`docs/performance.md` 6.2); rollback under 5 minutes (ADR 0010); the PostgreSQL image `ghcr.io/alievs-corp/hypedomino-postgres-walg` is linux/amd64 only, like the WAL-G release binary (`docs/backup-restore.md` 3.6).

## Decision

### Nodes per role
One k3s cluster per environment with three nodes and a load balancer, all in `fsn1` on one private network, created by OpenTofu (ADR 0010). Each node carries the label `hypedomino.io/role` with its role; the environment overlays of `deploy/k8s` pin workloads to it with a `nodeSelector`, so a game pod never lands on the database node and the reverse.

| Role (`hypedomino.io/role`) | Runs | Type | vCPU / RAM | Approx. € per month |
|---|---|---|---|---|
| `control` | k3s server (control plane), admin web and `admin-api`, Prometheus of the `monitoring` namespace (`docs/launch-runbook.md` OI-7) | CX23 | 2 shared x86 / 4 GB | 5.49 |
| `game` | k3s agent; `server` Deployment, the `migrate` Job and the `account-deletion` CronJob | CX33, or CAX21 (arm64) | 4 shared / 8 GB | 8.49 (CAX21 10.49) |
| `db` | k3s agent; PostgreSQL 16 StatefulSet with WAL-G (`deploy/k8s/postgres`), Redis 7, the backup CronJobs (`deploy/k8s/backup`) | CX33, x86 only, plus a Hetzner Volume for the PostgreSQL data (40 GB to start) | 4 shared / 8 GB | 8.49 + about 2 for the Volume |
| Load balancer | TLS termination and the public WebSocket endpoint (ADR 0010, ADR 0011) | LB11 | | about 6 to 7 |
| Object storage | Buckets below | | | 6.49 base, includes the first TB |
| Primary IPv4 | One per server | | | about 0.5 to 0.6 each |

About €38 to €45 per month for one environment, VAT excluded. Prices are 2026 figures from the Hetzner price adjustment page and third-party listings; verify in the console before ordering. The database node stays x86 until the PostgreSQL image is multi-arch. The game node may be CAX because the server image is built for linux/amd64 and linux/arm64 (`.github/workflows/server-image.yml`); a cluster with a CAX game node is mixed-architecture, which k3s supports, and the role pin keeps the amd64-only PostgreSQL and WAL-G pods on the `db` node.

Buckets per environment, in the names `docs/launch-runbook.md` OI-3 and `docs/backup-restore.md` use, a suffix added if a name is taken: OpenTofu state `hypedomino-tofu-state-<environment>` (versioning on); content `hypedomino-content-<environment>` in `fsn1`; in a second German location (`nbg1`): WAL-G backups `hypedomino-walg-<environment>`, deletion log `hypedomino-deletion-log-<environment>` and the content mirror `hypedomino-content-mirror-<environment>`.

`production` runs this layout for the closed beta and the launch. `staging` uses the same node classes and is created from the same OpenTofu code only for drills and load runs, then destroyed, so it is paid by the hour (E-1 still holds while it exists).

When a type is not available: the same type in `nbg1` (same network zone), then for the game node the CAX equivalent, then CCX13, the smallest dedicated type, for any role. Never a type with less RAM than the table.

### Upgrade trigger
The game node moves to dedicated vCPU (CCX23, the reference node of `docs/performance.md` 6.1) when either holds:
1. Production: the alert `GameNodeMoveRoundTripOverBudget` (`deploy/k8s/monitoring/server-latency.rules.yaml`) fires, that is the p99 of the server histogram `move_roundtrip_seconds` over 5 minutes stays above 100 ms for 15 minutes at more than one move per second, and rolling back the latest release (`docs/launch-runbook.md` 5.4) does not clear it. The histogram is only the in-process share of the round-trip, so players already see more than the budget when it fires.
2. Measurement: the beta run of `docs/performance.md` 6.1 on the shared game node type fails the bot-measured move round-trip gate of 6.2 (p99 under 100 ms overall and in every 1-minute window) at the beta peak.

The same alert firing is a SEV2 incident (`docs/launch-runbook.md` 6.1). The control and database nodes are resized only for their own reasons (memory, disk, query latency), not by this trigger.

### Upgrade and rollback
A new node of the target type joins the cluster with the `game` role, the old one is cordoned and `kubectl rollout restart deployment/server` moves the pods: the rolling deploy of ADR 0010 brings the new pods up on the new node before old pods drain, so no live match ends and capacity never drops. The old node is removed after the new one has held the budget for 30 minutes. Rollback is the same move in the other direction. This works between architectures (CAX to CCX), which an in-place resize does not. The steps are `docs/launch-runbook.md` 5.6.

## Alternatives considered and why rejected
- CCX23 for every node from the start (the previous assumption): about €86 per node and month by the owner's 2026 figure; no evidence yet that the beta load needs dedicated vCPU anywhere but possibly the game node.
- One shared node for everything: cheapest, but the game server, PostgreSQL and the control plane would compete for the same 4 vCPU and a noisy database query would show up as move latency.
- In-place resize (`hcloud server change-type`) as the upgrade path: needs the server powered off, so every live room on the only game node ends; impossible from CAX to CCX.
- Managed database or another provider: rejected by ADR 0010 and ADR 0013 for the same reasons.

## Consequences
- Easier: the beta runs for about a tenth of the dedicated layout; the multi-arch image lets the game node use whichever 4 vCPU / 8 GB type is available.
- Harder: latency on shared vCPU is not reproducible run to run, so a shared-type load run proves the beta peak at that time, not a capacity number; the 10,000-room node target and run-to-run comparisons stay on the dedicated CCX23 reference. One node per role means a node failure is an outage until it is replaced; a `db` node loss is a restore from WAL-G (`docs/backup-restore.md`, RTO 1 hour).
- Must be verified: the beta run of `docs/performance.md` 6.1 on the chosen shared type before the closed beta opens; the alert reaching the pager (OI-9) with one test alert (GN-12); the `nodeSelector` pins in the `staging` and `production` overlays when they are written (OI-1).
