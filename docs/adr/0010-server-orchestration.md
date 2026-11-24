# 0010. Server orchestration: k3s on Hetzner Cloud, drain-aware rolling deploys

Status: Accepted
Date: 2026-10-28
Issue: #25

## Context
The game server is one Go binary (ADR 0003) that hosts many rooms in memory, each room on exactly one node, with live WebSocket connections (ADR 0004). A match lasts minutes; killing a node with live rooms ends matches and costs players rank and trust (`docs/launch-runbook.md` 5.1). The binary already drains on `SIGTERM`: `/readyz` turns 503, no new room or WebSocket is accepted, live rooms play to their end bounded by `HD_DRAIN_TIMEOUT` (default 15 minutes), then connections close (`server/README.md`, #43).

Constraints:
- Hosting provider and primary data region are fixed: Hetzner, Germany (`docs/privacy.md` section 2, OQ-3; `docs/environments.md` section 1).
- An image is built once and promoted from `staging` to `production` unchanged (`docs/environments.md` E-4).
- Rollback of a server release under 5 minutes (#25, `docs/launch-runbook.md` 5.4).
- Budgets: p99 move round-trip under 100 ms in region, 10k concurrent rooms per node as a target (CLAUDE.md).
- A small team operates it; every moving part must earn its place.

## Decision

### Platform
- Kubernetes, as k3s on Hetzner Cloud servers in Germany (Falkenstein `fsn1` and Nuremberg `nbg1`), one cluster per environment (`staging`, `production`, E-1).
- Infrastructure as code with OpenTofu and the `hcloud` provider; state in a Hetzner Object Storage bucket per environment with versioning on. Cluster, load balancer, firewall, volumes and DNS are all created by it; nothing is clicked in the console (GN-8 evidence is the apply log).
- Images: `ghcr.io/alievs-corp/hypedomino-server`, tagged with the full commit SHA by `.github/workflows/server-image.yml` (#25). Manifests reference the tag, never `latest`.
- Manifests live in `deploy/k8s`: `base` holds the shape every environment shares, an overlay per environment sets the image tag, replicas, resources and the `server-env` secret source. Environments differ only by overlay values and secrets (E-4).

### Image
- `server/Dockerfile`: static `CGO_ENABLED=0` build on the Go version of `server/go.mod`, runtime `gcr.io/distroless/static-debian12:nonroot` (no shell, no package manager), numeric user `65532`, read-only root filesystem, every capability dropped. The image carries `/server` (the entrypoint), `/migrate` and `/account-deletion`, so the migration runner and the deletion CronJob run the same promoted build (#601).
- Health is checked by the orchestrator over HTTP (`/healthz` liveness, `/readyz` readiness), not by a command inside the image.

### Rolling deploy with drain
- The server runs as a `Deployment` with `maxUnavailable: 0` and `maxSurge: 1`: a new pod must be ready before an old one is told to stop, so capacity never drops during a deploy.
- Termination order of an old pod: it leaves the Service endpoints, a `preStop` sleep of 10 s lets the load balancer and kube-proxy stop sending it new connections, then `SIGTERM` starts the drain of the binary.
- `terminationGracePeriodSeconds` = `preStop` + `HD_DRAIN_TIMEOUT` + `HD_SHUTDOWN_TIMEOUT` + 60 s margin = 980 s with the defaults. Raising `HD_DRAIN_TIMEOUT` requires raising it in the same change; otherwise the kubelet kills rooms the binary would have let finish.
- A `PodDisruptionBudget` of `maxUnavailable: 1` keeps node maintenance from draining several game pods at once.
- Migrations run as one Kubernetes `Job` from the new image before the rollout starts (`docs/environments.md` M-5); a failed job stops the deploy. The job is `deploy/k8s/base/migrate.yaml` and `tools/deploy/apply-release.sh` orders it before the rollout (`docs/launch-runbook.md` 5.1).

### Rollback
- Rollback is `kubectl rollout undo deployment/server` or a deploy of the previous SHA tag through the same workflow. It is a rolling deploy like any other: new matches start on the previous version as soon as its pods are ready; matches already running on the bad version finish under the drain.
- Rollback time is measured from the command to the moment every replica of the previous version is available. CI measures it on every server change (`image` job), the budget is 5 minutes; the staging drill GN-9 measures it with real load.

### Regional topology
- Phase 1 (soft launch and launch): one region, Germany. All player data, PostgreSQL and Redis live there (OQ-3).
- Phase 2: game-only clusters closer to players, added when soft launch data (#196) shows a market whose p99 move round-trip exceeds the 100 ms budget. Candidates on the same provider: Helsinki `hel1` for northern and eastern Europe, Singapore `sin` for Asia. MENA has no location on the same provider; a second provider for it is a later ADR with its own privacy review.
- A game-only cluster runs rooms and matchmaking for its region and writes match results, ledger and inventory changes to the primary database in Germany. Player data is processed there in memory, not stored (`docs/privacy.md` section 2 applies unchanged).

### Reconnect during a drain
- Rooms are in memory, so a player whose connection drops while their room's pod drains can resume only on that pod. A draining pod refuses public WebSocket connections today, so such a player is covered by the reconnect grace and then by bot takeover (`docs/penalties.md`).
- The fix is relay, not public per-pod addresses: any ready pod that receives a resume for a room it does not host looks up the room's pod in Redis (ADR 0005, room routing) and relays the session over the cluster network; a draining pod accepts relayed resumes for its own rooms. This belongs to cross-node routing in #51 and keeps TLS and the public endpoint on the load balancer.

## Alternatives considered and why rejected
- Plain Docker on VMs with deploy scripts: no orchestration to learn, but rolling with readiness, surge, grace periods, rollback history and secret injection would be written by hand and tested by us.
- Nomad: smaller than Kubernetes, but a smaller ecosystem on Hetzner, fewer people who know it, and a source-available licence.
- Managed Kubernetes or a platform outside Hetzner (managed clusters at large clouds, Fly.io): conflicts with the hosting decision OQ-3 or adds a second provider holding player traffic.
- A game server fleet manager on Kubernetes (Agones): built for one match per process allocated by an external matchmaker. Our process hosts thousands of rooms and runs matchmaking itself; its per-pod public addressing is the problem relay solves more simply. Reconsidered if rooms ever move to one process each.
- `StatefulSet`: stable pod names do not help without per-pod addressing, and its ordered rolling update deletes a pod before its replacement starts, which drops capacity for a whole drain per pod.

## Consequences
- Easier: rolling deploys, readiness, rollback history, secret injection and node replacement come from the platform; CI proves the drain order and the rollback time on a real Kubernetes API in every server PR.
- Harder: the team operates a k3s cluster (upgrades, certificates, etcd backups with #188). A deploy takes as long as the longest match on an old pod; up to `HD_DRAIN_TIMEOUT` old and new versions serve together, which the protocol already allows (ADR 0004 versioning).
- Must be verified on staging before launch: GN-9 drill with live matches during a deploy and a rollback under 5 minutes, node size and pods per node from the load test (#192), and the relay of #51 for reconnects during a drain.
