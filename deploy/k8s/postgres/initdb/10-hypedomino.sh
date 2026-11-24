#!/usr/bin/env bash
set -euo pipefail

[ -n "${PG_WALG_PASSWORD:-}" ] || { echo "PG_WALG_PASSWORD from Secret walg-credentials is required to create the walg role"; exit 1; }

echo "include_dir = '/etc/postgresql/conf.d'" >> "$PGDATA/postgresql.conf"

psql -v ON_ERROR_STOP=1 --no-psqlrc --host /var/run/postgresql --username "$POSTGRES_USER" --dbname postgres \
  -v walg_password="$PG_WALG_PASSWORD" <<'SQL'
create role walg login password :'walg_password';
grant pg_read_all_settings to walg;
grant execute on function pg_backup_start(text, boolean) to walg;
grant execute on function pg_backup_stop(boolean) to walg;
SQL
