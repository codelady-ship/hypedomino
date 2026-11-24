# 0013. PostgreSQL backups: WAL-G archiving to S3-compatible storage

Status: Accepted
Date: 2026-11-12
Issue: #188

## Context
PostgreSQL holds currency, purchases, inventory and ranks (ADR 0005). Losing a committed ledger entry or double-granting after a restore is the worst data incident the game can have. The targets are in `docs/backup-restore.md` section 2: RPO 5 minutes, RTO 1 hour.

Constraints:
- Hetzner, Germany (ADR 0010); the object store is S3-compatible (`server/internal/objectstore`).
- A small team: one tool, one way to restore, a restore that can be rehearsed on a laptop with Docker.
- Backups hold personal data (`docs/privacy.md` D-30): encrypted, retained for `BackupRetentionDays`, in the primary region.

## Decision
- Continuous WAL archiving plus a daily base backup, using WAL-G (`archive_command = 'wal-g wal-push %p'`, `restore_command = 'wal-g wal-fetch %f %p'`), stored in an S3-compatible bucket reserved for backups.
- WAL-G encrypts client-side with libsodium; the key lives outside the backup bucket and outside the cluster the database runs in.
- Point-in-time recovery uses PostgreSQL's native `recovery_target_time`; WAL-G only moves files.
- The release binary is pinned by version and SHA-256 in `tools/backup-drill/Dockerfile`; the same image runs PostgreSQL itself (`deploy/k8s/postgres`), the CronJobs in `deploy/k8s/backup` and the CI restore drill, so what is rehearsed is what runs. It is published as `ghcr.io/alievs-corp/hypedomino-postgres-walg` (`docs/backup-restore.md` 3.6).

## Alternatives considered and why rejected
- pgBackRest: equally mature and has parallel restore and a repo host, but its S3 support is repository-centric and its operation model (stanzas, `pgbackrest.conf` on every host, optional SSH repo host) is heavier than a small team needs. Reconsidered if the database moves to multiple replicas with a dedicated backup host.
- Barman: designed around a separate backup server pulling over SSH or streaming; one more server to run.
- pg_dump on a schedule: logical, no point-in-time recovery, RPO equals the schedule interval (hours), long restores on a large ledger.
- Managed PostgreSQL with built-in PITR: conflicts with the hosting decision (ADR 0010, OQ-3) or adds a second provider holding player data.
- Volume snapshots only: crash-consistent, no PITR between snapshots, RPO equals the snapshot interval.

## Consequences
- Easier: RPO is bounded by `archive_timeout` and the archive latency, not by a schedule; restore is two commands plus a recovery target; the drill runs in CI in about a minute.
- Harder: the database host needs the WAL-G binary beside PostgreSQL, a backup role and a bucket with its own credentials; a full WAL archive must be monitored (a failing `archive_command` fills the data volume).
- Must be verified on staging before launch: restore from the real bucket into a fresh node and the measured RTO (`docs/backup-restore.md` section 8).
