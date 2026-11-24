#!/usr/bin/env bash
set -euo pipefail
export MSYS_NO_PATHCONV=1

here=$(cd "$(dirname "$0")" && pwd)
run=$(date +%s)-$$
image="hd-backup-drill:local"
server_image="hd-backup-drill-server:local"
network="hd-drill-$run"
objectstore="hd-drill-objectstore-$run"
source_db="hd-drill-source-$run"
restored_db="hd-drill-restored-$run"
source_volume="hd-drill-source-data-$run"
restored_volume="hd-drill-restored-data-$run"
bucket="hd-backups"
deletion_bucket="hd-deletion-log"
access_key="drill$(openssl rand -hex 6)"
secret_key="$(openssl rand -hex 16)"
encryption_key="$(openssl rand -hex 32)"

cleanup() {
  docker rm -f "$source_db" "$restored_db" "$objectstore" >/dev/null 2>&1 || true
  docker volume rm "$source_volume" "$restored_volume" >/dev/null 2>&1 || true
  docker network rm "$network" >/dev/null 2>&1 || true
}
trap cleanup EXIT

walg_env=(
  -e "WALG_S3_PREFIX=s3://$bucket/drill"
  -e "AWS_ENDPOINT=http://$objectstore:9000"
  -e "AWS_S3_FORCE_PATH_STYLE=true"
  -e "AWS_REGION=us-east-1"
  -e "AWS_ACCESS_KEY_ID=$access_key"
  -e "AWS_SECRET_ACCESS_KEY=$secret_key"
  -e "WALG_LIBSODIUM_KEY=$encryption_key"
  -e "WALG_LIBSODIUM_KEY_TRANSFORM=hex"
  -e "WALG_COMPRESSION_METHOD=zstd"
  -e "PGUSER=postgres"
  -e "PGHOST=/var/run/postgresql"
)

sql() {
  docker exec -u postgres "$1" psql -qtAX -v ON_ERROR_STOP=1 -c "$2"
}

server_tool() {
  local db=$1 tool=$2
  shift 2
  docker run --rm --network "$network" \
    -e "HD_DATABASE_URL=postgres://postgres:drill@$db:5432/postgres?sslmode=disable" \
    -e "HD_DELETION_LOG_URL=http://$objectstore:9000/$deletion_bucket" \
    -e "HD_DELETION_LOG_REGION=us-east-1" \
    -e "HD_DELETION_LOG_ACCESS_KEY=$access_key" \
    -e "HD_DELETION_LOG_SECRET_KEY=$secret_key" \
    --entrypoint "/$tool" "$server_image" "$@"
}

wait_ready() {
  for _ in $(seq 1 120); do
    docker exec -u postgres "$1" pg_isready -q -h 127.0.0.1 && return 0
    sleep 0.5
  done
  echo "$1 did not become ready"
  docker logs "$1" 2>&1 | tail -30
  return 1
}

elapsed_since() {
  echo $(( $(date +%s) - $1 ))
}

create_bucket() {
  docker run --rm --network "$network" --entrypoint bash "$image" -c "
    for i in \$(seq 1 60); do
      curl -sSf -o /dev/null --aws-sigv4 aws:amz:us-east-1:s3 --user $access_key:$secret_key -X PUT http://$objectstore:9000/$1 && exit 0
      sleep 1
    done
    exit 1" >/dev/null
}

seed_account() {
  sql "$source_db" "with a as (insert into accounts default values returning id),
    i as (insert into auth_identities (provider, subject, account_id, email) select 'google', 'drill-$1', id, 'drill-$1@example.com' from a)
    insert into profiles (account_id, nickname, nickname_key) select id, 'Drill$1', 'drill$1' from a returning account_id"
}

request_deletion() {
  sql "$source_db" "update accounts set state = 'deleting', deletion_requested_at = now() - interval '31 days' where id = '$1'"
}

personal_rows() {
  sql "$1" "select (select count(*) from accounts where id = '$2')
    + (select count(*) from auth_identities where (provider = 'google' and subject = 'drill-$3') or email = 'drill-$3@example.com' or account_id = '$2')
    + (select count(*) from profiles where nickname_key = 'drill$3' or account_id = '$2')
    + (select count(*) from nickname_history h left join accounts a on a.id = h.account_id where h.nickname_key = 'drill$3' and (a.state is distinct from 'deleted' or a.id is distinct from '$4'))"
}

database_state() {
  sql "$1" "select md5(concat_ws('|',
    (select string_agg(id::text || state || coalesce(deleted_at::text, ''), ',' order by id) from accounts),
    (select string_agg(subject || account_id::text, ',' order by subject) from auth_identities),
    (select string_agg(nickname_key || account_id::text, ',' order by nickname_key) from profiles),
    (select string_agg(nickname_key || account_id::text, ',' order by nickname_key, account_id) from nickname_history)))"
}

(cd "$here" && docker build -q -t "$image" .) >/dev/null
(cd "$here/../../server" && docker build -q -t "$server_image" .) >/dev/null
docker network create "$network" >/dev/null

docker run -d --name "$objectstore" --network "$network"   -e "AWS_ACCESS_KEY_ID=$access_key" -e "AWS_SECRET_ACCESS_KEY=$secret_key"   chrislusf/seaweedfs:4.53 server -s3 -s3.port=9000 -dir=/data >/dev/null
create_bucket "$bucket"
create_bucket "$deletion_bucket"

docker volume create "$source_volume" >/dev/null
docker run -d --name "$source_db" --network "$network" "${walg_env[@]}" \
  -e POSTGRES_PASSWORD=drill -v "$source_volume:/var/lib/postgresql/data" "$image" \
  postgres -c wal_level=replica -c archive_mode=on -c archive_timeout=60 \
  -c "archive_command=wal-g wal-push %p" >/dev/null
wait_ready "$source_db"

server_tool "$source_db" migrate up >/dev/null
early_account=$(seed_account early)
late_account=$(seed_account late)
request_deletion "$early_account"

sql "$source_db" "create table ledger (id bigserial primary key, batch text not null, amount int not null, at timestamptz not null default clock_timestamp())"
sql "$source_db" "insert into ledger (batch, amount) select 'before-backup', g from generate_series(1, 1000) g"
docker exec -u postgres "${walg_env[@]}" "$source_db" wal-g backup-push /var/lib/postgresql/data >/dev/null

sql "$source_db" "insert into ledger (batch, amount) select 'after-backup', g from generate_series(1, 500) g"
server_tool "$source_db" account-deletion run
sql "$source_db" "select pg_switch_wal()" >/dev/null
sleep 2
target_time=$(sql "$source_db" "select to_char(clock_timestamp() at time zone 'UTC', 'YYYY-MM-DD HH24:MI:SS.US') || '+00'")
replay_since=$(sql "$source_db" "select to_char((clock_timestamp() - interval '10 minutes') at time zone 'UTC', 'YYYY-MM-DD\"T\"HH24:MI:SS.US\"Z\"')")
sleep 2
request_deletion "$late_account"
server_tool "$source_db" account-deletion run
late_tombstone=$(sql "$source_db" "select account_id from nickname_history where nickname_key = 'drilllate'")
sql "$source_db" "insert into ledger (batch, amount) select 'after-target', g from generate_series(1, 250) g"
sql "$source_db" "select pg_switch_wal()" >/dev/null

[ -n "$late_tombstone" ] || { echo "the late account has no tombstone in the source"; exit 1; }
[ "$(personal_rows "$source_db" "$late_account" late "$late_tombstone")" = "0" ] || { echo "the deletion job left personal rows of the late account in the source"; exit 1; }
[ "$(sql "$source_db" "select count(*) from deletion_log_pending")" = "0" ] || { echo "a deletion was erased but not logged"; exit 1; }

for _ in $(seq 1 60); do
  archived=$(sql "$source_db" "select coalesce(last_archived_wal >= pg_walfile_name(pg_current_wal_lsn() - 1048576), false) from pg_stat_archiver")
  [ "$archived" = "t" ] && break
  sleep 1
done
[ "$archived" = "t" ] || { echo "the last WAL segment was not archived in time"; exit 1; }
failed=$(sql "$source_db" "select failed_count from pg_stat_archiver")
[ "$failed" = "0" ] || { echo "WAL archiving failed $failed times"; exit 1; }
echo "source rows: $(sql "$source_db" "select count(*) from ledger") (recovery target $target_time)"
docker rm -f "$source_db" >/dev/null

docker volume create "$restored_volume" >/dev/null
restore_started=$(date +%s)
docker run --rm "${walg_env[@]}" --network "$network" --user root -v "$restored_volume:/var/lib/postgresql/data" \
  --entrypoint bash "$image" -c '
    set -e
    wal-g backup-list
    wal-g backup-fetch /var/lib/postgresql/data LATEST
    touch /var/lib/postgresql/data/recovery.signal
    chown -R postgres:postgres /var/lib/postgresql/data
    chmod 700 /var/lib/postgresql/data'

docker run -d --name "$restored_db" --network "$network" "${walg_env[@]}" \
  -v "$restored_volume:/var/lib/postgresql/data" "$image" \
  postgres -c "restore_command=wal-g wal-fetch %f %p" \
  -c "recovery_target_time=$target_time" -c recovery_target_action=promote >/dev/null
wait_ready "$restored_db"
for _ in $(seq 1 60); do
  [ "$(sql "$restored_db" "select pg_is_in_recovery()")" = "f" ] && break
  sleep 1
done
restore_seconds=$(elapsed_since "$restore_started")

total=$(sql "$restored_db" "select count(*) from ledger")
before=$(sql "$restored_db" "select count(*) from ledger where batch = 'before-backup'")
after=$(sql "$restored_db" "select count(*) from ledger where batch = 'after-backup'")
past_target=$(sql "$restored_db" "select count(*) from ledger where batch = 'after-target'")
echo "restored rows: total=$total before-backup=$before after-backup=$after after-target=$past_target in ${restore_seconds}s"

[ "$before" = "1000" ] || { echo "expected 1000 rows from before the base backup, got $before"; exit 1; }
[ "$after" = "500" ] || { echo "expected 500 rows replayed from archived WAL, got $after"; exit 1; }
[ "$past_target" = "0" ] || { echo "found $past_target rows written after the recovery target"; exit 1; }
sql "$restored_db" "insert into ledger (batch, amount) values ('post-restore', 1)"
echo "restore drill passed: point-in-time recovery stopped at $target_time"

[ "$(sql "$restored_db" "select count(*) from auth_identities where subject = 'drill-late'")" = "1" ] || { echo "the restore did not bring back the account deleted after the target, so the replay check proves nothing"; exit 1; }
[ "$(sql "$restored_db" "select count(*) from accounts where id = '$early_account'")" = "0" ] || { echo "the account deleted before the target came back with the restore"; exit 1; }
first_replay=$(server_tool "$restored_db" account-deletion replay -since "$replay_since")
echo "$first_replay"
after_first=$(database_state "$restored_db")
second_replay=$(server_tool "$restored_db" account-deletion replay -since "$replay_since")
echo "$second_replay"
after_second=$(database_state "$restored_db")

case "$first_replay" in *"1 of 2 logged deletion(s)"*) ;; *) echo "the first replay should re-apply exactly the late deletion of the two logged"; exit 1 ;; esac
case "$second_replay" in *"0 of 2 logged deletion(s)"*) ;; *) echo "the second replay re-applied a deletion again"; exit 1 ;; esac
[ "$after_first" = "$after_second" ] || { echo "the second replay changed the database"; exit 1; }
leftover=$(personal_rows "$restored_db" "$late_account" late "$late_tombstone")
[ "$leftover" = "0" ] || { echo "after the replay $leftover rows still reach the late account by id, sign-in identity, email or nickname"; exit 1; }
[ "$(sql "$restored_db" "select count(*) from accounts where id = '$late_tombstone' and state = 'deleted'")" = "1" ] || { echo "the replay did not restore the logged tombstone $late_tombstone"; exit 1; }
echo "deletion replay passed: the account deleted after the target is erased again under tombstone $late_tombstone, a second replay changed nothing"
