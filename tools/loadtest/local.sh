#!/usr/bin/env bash
set -euo pipefail

usage="usage: local.sh <run dir> <rooms>... ; env: NODE_CPUS NODE_MEMORY NODE_CPUSET BOT_CPUSET RAMP HOLD BOT_FLAGS PROFILE_AT PREBUILT"
run_root="${1:?$usage}"
shift
[ $# -gt 0 ] || { echo "$usage" >&2; exit 2; }

repo=$(cd "$(dirname "$0")/../.." && pwd)
node_cpus="${NODE_CPUS:-4}"
node_memory="${NODE_MEMORY:-4g}"
node_pinning=()
bot_pinning=()
[ -z "${NODE_CPUSET:-}" ] || node_pinning=(--cpuset-cpus "$NODE_CPUSET")
[ -z "${BOT_CPUSET:-}" ] || bot_pinning=(--cpuset-cpus "$BOT_CPUSET")
ramp="${RAMP:-1m}"
hold="${HOLD:-4m}"
read -ra bot_flags <<< "${BOT_FLAGS:-}"
profile_at="${PROFILE_AT:-150}"
image=gcr.io/distroless/static-debian12:nonroot
network=hd-load
prefix=hd-load
database="postgres://hypedomino@$prefix-postgres:5432/hypedomino?sslmode=disable"

docker() {
  MSYS_NO_PATHCONV=1 command docker "$@"
}

host_path() {
  if command -v cygpath >/dev/null; then cygpath -m "$1"; else echo "$1"; fi
}

build() {
  mkdir -p "$run_root/bin"
  if [ -n "${PREBUILT:-}" ]; then
    cp "$PREBUILT/loadbot" "$PREBUILT/migrate" "$run_root/bin/"
    cp "$PREBUILT/commit" "$run_root/commit" 2>/dev/null || echo "prebuilt $PREBUILT" > "$run_root/commit"
    return
  fi
  (cd "$repo/server" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o "$run_root/bin/" ./cmd/loadbot ./cmd/migrate)
  git -C "$repo" rev-parse HEAD > "$run_root/commit"
}

foreign_containers() {
  docker ps --format '{{.Names}}' | grep -v "^$prefix-" || true
}

vm_load() {
  docker exec "$prefix-redis" cat /proc/loadavg | cut -d' ' -f1
}

wait_quiet() {
  while true; do
    local others load
    others=$(foreign_containers)
    load=$(vm_load)
    if [ -z "$others" ] && awk -v l="$load" 'BEGIN { exit !(l < 1.5) }'; then
      echo "quiet: no other containers, VM load $load"
      return
    fi
    echo "$(date -u +%FT%TZ) waiting: other containers [${others//$'\n'/ }], VM load $load"
    sleep 60
  done
}

stores_up() {
  docker network inspect "$network" >/dev/null 2>&1 || docker network create "$network" >/dev/null
  docker rm -f "$prefix-postgres" "$prefix-redis" >/dev/null 2>&1 || true
  docker run -d --name "$prefix-postgres" --network "$network" \
    -e POSTGRES_USER=hypedomino -e POSTGRES_DB=hypedomino -e POSTGRES_HOST_AUTH_METHOD=trust \
    postgres:16-alpine -c shared_preload_libraries=pg_stat_statements -c max_connections=200 >/dev/null
  docker run -d --name "$prefix-redis" --network "$network" redis:7-alpine >/dev/null
  until docker exec "$prefix-postgres" pg_isready -U hypedomino -d hypedomino >/dev/null 2>&1; do sleep 1; done
  sleep 2
  docker exec "$prefix-postgres" psql -U hypedomino -d hypedomino -qc 'create extension if not exists pg_stat_statements' >/dev/null
  docker run --rm --network "$network" -v "$(host_path "$run_root/bin"):/work:ro" -e HD_DATABASE_URL="$database" \
    "$image" /work/migrate up > "$run_root/migrate.log" 2>&1
}

node_up() {
  docker rm -f "$prefix-node" >/dev/null 2>&1 || true
  docker run -d --name "$prefix-node" --network "$network" --cpus "$node_cpus" --memory "$node_memory" "${node_pinning[@]}" \
    --ulimit nofile=262144:262144 -p 127.0.0.1:16060:6060 -p 127.0.0.1:16061:6061 \
    -v "$(host_path "$run_root/bin"):/work:ro" \
    -e HD_LISTEN_ADDR=:8080 -e HD_DEBUG_ADDR=:6060 -e HD_PPROF_ADDR=:6061 -e HD_LOG_LEVEL=warn \
    -e HD_DATABASE_URL="$database" -e HD_REDIS_URL="redis://$prefix-redis:6379/0" -e HD_NODE_ID=load-1 \
    "$image" /work/loadbot serve >/dev/null
  until curl -fs -o /dev/null http://127.0.0.1:16060/metrics; do sleep 1; done
}

pprof() {
  curl -fsS "http://127.0.0.1:16061/debug/pprof/$1"
}

stores_snapshot() {
  docker exec "$prefix-postgres" psql -U hypedomino -d hypedomino -At -F ' ' -c \
    "select xact_commit, tup_inserted, tup_updated, blks_hit, blks_read from pg_stat_database where datname = 'hypedomino'" > "$1.pg"
  docker exec "$prefix-redis" redis-cli info commandstats > "$1.redis"
  docker exec "$prefix-redis" redis-cli info stats | grep -E '^total_commands_processed' >> "$1.redis"
}

stores_reset() {
  docker exec "$prefix-postgres" psql -U hypedomino -d hypedomino -qc 'select pg_stat_statements_reset(); select pg_stat_reset()' >/dev/null
  docker exec "$prefix-redis" redis-cli config resetstat >/dev/null
}

sample() {
  local dir="$1"
  while true; do
    local now
    now=$(date -u +%s)
    curl -fsS http://127.0.0.1:16060/metrics > "$dir/metrics/$now.prom" 2>/dev/null || true
    docker stats --no-stream --format "$now {{.Name}} {{.CPUPerc}} {{.MemUsage}}" \
      "$prefix-node" "$prefix-bot" "$prefix-postgres" "$prefix-redis" >> "$dir/stats.log" 2>/dev/null || true
    sleep 10
  done
}

profile() {
  local dir="$1"
  sleep "$profile_at"
  pprof "profile?seconds=30" > "$dir/cpu.pprof" || true
  pprof "goroutine?debug=1" > "$dir/goroutines.txt" || true
  pprof "heap?debug=1" | grep -E '^# (NumGC|PauseTotalNs|GCCPUFraction|HeapAlloc|Sys) ' > "$dir/memstats.mid" || true
}

step() {
  local rooms="$1"
  local dir="$run_root/rooms-$rooms"
  mkdir -p "$dir/metrics"
  wait_quiet | tee "$dir/quiet.log"
  node_up
  stores_reset
  stores_snapshot "$dir/before"
  date -u +%s > "$dir/started"
  sample "$dir" &
  local sampler=$!
  profile "$dir" &
  local profiler=$!
  docker rm -f "$prefix-bot" >/dev/null 2>&1 || true
  set +e
  docker run --name "$prefix-bot" --network "$network" --ulimit nofile=262144:262144 "${bot_pinning[@]}" \
    --sysctl net.ipv4.ip_local_port_range="1024 65000" -e HD_LOG_LEVEL=info \
    -v "$(host_path "$run_root/bin"):/work:ro" -v "$(host_path "$dir"):/out" \
    "$image" /work/loadbot run -target "ws://$prefix-node:8080/ws" -metrics "http://$prefix-node:6060/metrics" \
    -rooms "$rooms" -ramp "$ramp" -hold "$hold" -out /out/report.json "${bot_flags[@]}" > "$dir/report.txt" 2> "$dir/bot.log"
  local status=$?
  set -e
  kill "$sampler" "$profiler" 2>/dev/null || true
  wait "$sampler" "$profiler" 2>/dev/null || true
  curl -fsS http://127.0.0.1:16060/metrics > "$dir/final.prom" || true
  pprof heap > "$dir/heap.pprof" || true
  pprof "heap?debug=1" | grep -E '^# (NumGC|PauseTotalNs|GCCPUFraction|HeapAlloc|Sys) ' > "$dir/memstats.end" || true
  stores_snapshot "$dir/after"
  docker exec "$prefix-postgres" psql -U hypedomino -d hypedomino -At -F ' | ' -c \
    "select calls, round(total_exec_time::numeric, 1), round(mean_exec_time::numeric, 3), left(regexp_replace(query, '\s+', ' ', 'g'), 120) from pg_stat_statements order by total_exec_time desc limit 10" > "$dir/pg_top.txt" || true
  docker logs "$prefix-node" > "$dir/server.log" 2>&1 || true
  docker rm -f "$prefix-bot" >/dev/null 2>&1 || true
  echo "rooms $rooms exit $status"
  cat "$dir/report.txt"
}

mkdir -p "$run_root"
run_root=$(cd "$run_root" && pwd)
build
stores_up
{
  date -u +%FT%TZ
  docker info --format 'docker {{.ServerVersion}}, VM {{.NCPU}} CPUs, {{.MemTotal}} bytes'
  echo "node: --cpus $node_cpus --memory $node_memory ${node_pinning[*]}; bot: ${bot_pinning[*]}; ramp $ramp hold $hold; bot flags: ${bot_flags[*]}"
} > "$run_root/machine.txt"
for rooms in "$@"; do
  step "$rooms"
done
docker rm -f "$prefix-node" >/dev/null 2>&1 || true
