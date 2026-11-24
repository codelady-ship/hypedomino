#!/usr/bin/env bash
set -euo pipefail

usage="usage: hetzner.sh up | run <name> [loadbot run flags] | fetch <name> | down | all
env: HCLOUD_TOKEN, HD_LOAD_SSH_KEY (key name in the project), HD_LOAD_SSH_IDENTITY (private key file, default ~/.ssh/id_ed25519),
     HD_LOAD_WORK (~/hd-load, results and the store password), HD_LOAD_LOCATION (fsn1), HD_LOAD_TYPE (ccx23), HD_LOAD_NODE_TYPE and HD_LOAD_BOT_TYPE (HD_LOAD_TYPE),
     HD_LOAD_STORES_TYPE (ccx23), a cax* type gets arm64 binaries, RAMP_MIN (5), HOLD_MIN (30), PROFILE_AT_MIN (ramp + hold / 2)"
command="${1:?$usage}"
shift

here=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$here/../.." && pwd)
work="${HD_LOAD_WORK:-$HOME/hd-load}"
label=purpose=hd-load
location="${HD_LOAD_LOCATION:-fsn1}"
node_type="${HD_LOAD_NODE_TYPE:-${HD_LOAD_TYPE:-ccx23}}"
bot_type="${HD_LOAD_BOT_TYPE:-${HD_LOAD_TYPE:-ccx23}}"
stores_type="${HD_LOAD_STORES_TYPE:-ccx23}"
identity="${HD_LOAD_SSH_IDENTITY:-$HOME/.ssh/id_ed25519}"
ramp_min="${RAMP_MIN:-5}"
hold_min="${HOLD_MIN:-30}"
profile_at_min="${PROFILE_AT_MIN:-$(( ramp_min + hold_min / 2 ))}"
ssh_options=(-i "$identity" -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile="$work/known_hosts" -o ConnectTimeout=10 -o ServerAliveInterval=30)

public_ip() {
  hcloud server ip "hd-load-$1"
}

private_ip() {
  hcloud server describe "hd-load-$1" -o format='{{ (index .PrivateNet 0).IP }}'
}

on() {
  local host="$1"
  shift
  ssh "${ssh_options[@]}" "root@$(public_ip "$host")" "$@"
}

copy_to() {
  local host="$1"
  shift
  scp "${ssh_options[@]}" "$@" "root@$(public_ip "$host"):"
}

secret() {
  [ -f "$work/stores.secret" ] || { umask 077; openssl rand -hex 24 > "$work/stores.secret"; }
  cat "$work/stores.secret"
}

database_url() {
  echo "postgres://hypedomino:$(secret)@$(private_ip stores):5432/hypedomino?sslmode=disable"
}

redis_url() {
  echo "redis://:$(secret)@$(private_ip stores):6379/0"
}

create_server() {
  local name="$1" type="$2"
  hcloud server create --name "hd-load-$name" --type "$type" --image ubuntu-24.04 --location "$location" \
    --ssh-key "${HD_LOAD_SSH_KEY:?HD_LOAD_SSH_KEY names the SSH key in the Hetzner project}" \
    --network hd-load --firewall hd-load --label "$label" >/dev/null
}

wait_ssh() {
  until on "$1" true 2>/dev/null; do sleep 5; done
}

arch_of() {
  case "$1" in
    cax* | CAX*) echo arm64 ;;
    *) echo amd64 ;;
  esac
}

host_arch() {
  case "$1" in
    node) arch_of "$node_type" ;;
    bot) arch_of "$bot_type" ;;
  esac
}

build() {
  local arch
  for arch in $( (arch_of "$node_type"; arch_of "$bot_type") | sort -u); do
    mkdir -p "$work/bin/$arch"
    (cd "$repo/server" && CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -o "$work/bin/$arch/" ./cmd/loadbot ./cmd/migrate)
  done
  git -C "$repo" rev-parse HEAD > "$work/commit"
}

up() {
  mkdir -p "$work"
  build
  hcloud network create --name hd-load --ip-range 10.70.0.0/16 --label "$label" >/dev/null
  hcloud network add-subnet hd-load --network-zone eu-central --type cloud --ip-range 10.70.1.0/24 >/dev/null
  echo '[{"direction":"in","protocol":"tcp","port":"22","source_ips":["0.0.0.0/0","::/0"]}]' > "$work/firewall.json"
  hcloud firewall create --name hd-load --label "$label" --rules-file "$work/firewall.json" >/dev/null
  create_server node "$node_type"
  create_server bot "$bot_type"
  create_server stores "$stores_type"
  for host in node bot stores; do wait_ssh "$host"; done
  local password stores_ip
  password=$(secret)
  stores_ip=$(private_ip stores)
  on stores "cloud-init status --wait >/dev/null; apt-get update -qq && apt-get install -y -qq docker.io >/dev/null &&
    docker run -d --name postgres --network host --restart unless-stopped -e POSTGRES_USER=hypedomino -e POSTGRES_DB=hypedomino -e POSTGRES_PASSWORD=$password \
      postgres:16 -c listen_addresses=$stores_ip -c max_connections=200 -c shared_preload_libraries=pg_stat_statements &&
    docker run -d --name redis --network host --restart unless-stopped redis:7 redis-server --bind $stores_ip --requirepass $password &&
    until docker exec postgres pg_isready -h $stores_ip -U hypedomino >/dev/null 2>&1; do sleep 1; done &&
    docker exec postgres psql -U hypedomino -qc 'create extension if not exists pg_stat_statements'"
  for host in node bot; do
    copy_to "$host" "$work/bin/$(host_arch "$host")/loadbot" "$work/bin/$(host_arch "$host")/migrate" "$here/node.sh" "$here/bot.sh" "$here/check.sh"
    on "$host" "cloud-init status --wait >/dev/null; sysctl -qw net.ipv4.ip_local_port_range='1024 65000' net.core.somaxconn=65535"
  done
  on node "HD_DATABASE_URL='$(database_url)' ./migrate up"
  echo "up: node $(private_ip node) $node_type, bot $(private_ip bot) $bot_type, stores $(private_ip stores) $stores_type; commit $(cat "$work/commit")"
}

run() {
  local name="${1:?run needs a name}"
  shift
  local node_ip
  node_ip=$(private_ip node)
  on stores "docker exec postgres psql -U hypedomino -qc 'select pg_stat_statements_reset()' >/dev/null; docker exec redis redis-cli -a $(secret) --no-auth-warning config resetstat >/dev/null"
  on node "pkill -x loadbot || true; sleep 2; HD_DATABASE_URL='$(database_url)' HD_REDIS_URL='$(redis_url)' HD_NODE_ID=load-1 setsid nohup ./node.sh $node_ip node-$name >/dev/null 2>&1 < /dev/null &"
  until on bot "curl -fsS -o /dev/null http://$node_ip:6060/metrics" 2>/dev/null; do sleep 2; done
  on bot "sleep $(( profile_at_min * 60 )); mkdir -p bot-$name; curl -fsS http://$node_ip:6061/debug/pprof/profile?seconds=30 > bot-$name/cpu.pprof" &
  local profiler=$!
  set +e
  on bot "./bot.sh $node_ip bot-$name -ramp ${ramp_min}m -hold ${hold_min}m $*"
  local status=$?
  set -e
  wait "$profiler" || true
  on node "pkill -INT -x loadbot || true; sleep 5"
  on stores "docker exec postgres psql -U hypedomino -At -F ' | ' -c \"select calls, round(total_exec_time::numeric, 1), round(mean_exec_time::numeric, 3), left(regexp_replace(query, '\\s+', ' ', 'g'), 120) from pg_stat_statements order by total_exec_time desc limit 10\"; docker exec redis redis-cli -a $(secret) --no-auth-warning info commandstats" > "$work/stores-$name.txt" || true
  fetch "$name"
  set +e
  (cd "$work/$name" && bash "$here/check.sh" "bot-$name" "$ramp_min" "$hold_min")
  local checked=$?
  set -e
  echo "run $name: bot exit $status, check exit $checked; results in $work/$name"
  [ "$status" -eq 0 ] && [ "$checked" -eq 0 ]
}

fetch() {
  local name="${1:?fetch needs a name}"
  mkdir -p "$work/$name"
  scp "${ssh_options[@]}" -r "root@$(public_ip bot):bot-$name" "$work/$name/"
  scp "${ssh_options[@]}" -r "root@$(public_ip node):node-$name" "$work/$name/"
  cp "$work/$name/node-$name/vmstat.log" "$work/$name/bot-$name/"
  if [ -f "$work/stores-$name.txt" ]; then mv "$work/stores-$name.txt" "$work/$name/stores.txt"; fi
  cp "$work/commit" "$work/$name/"
}

down() {
  hcloud server list -l "$label" -o noheader -o columns=name | xargs -r -n 1 hcloud server delete
  hcloud firewall list -l "$label" -o noheader -o columns=name | xargs -r -n 1 hcloud firewall delete
  hcloud network list -l "$label" -o noheader -o columns=name | xargs -r -n 1 hcloud network delete
  rm -f "$work/stores.secret" "$work/known_hosts"
}

case "$command" in
  up) up ;;
  run) run "$@" ;;
  fetch) fetch "$@" ;;
  down) down ;;
  all)
    trap down EXIT
    up
    failed=0
    run load || failed=1
    run storm -storm $(( ramp_min + 10 ))m || failed=1
    exit "$failed"
    ;;
  *) echo "$usage" >&2; exit 2 ;;
esac
