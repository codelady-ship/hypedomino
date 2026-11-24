#!/usr/bin/env bash
set -euo pipefail

node="${1:?usage: bot.sh <node private address> [run dir] [loadbot run flags]}"
run_dir="${2:-bot-$(date -u +%Y%m%dT%H%M%SZ)}"
shift $(( $# < 2 ? $# : 2 ))
mkdir -p "$run_dir/metrics"
ulimit -n 65536

sample() {
  while true; do
    curl -fsS "http://$node:6060/metrics" > "$run_dir/metrics/$(date -u +%s).prom" || true
    sleep 60
  done
}
sample &
sampler=$!
trap 'kill "$sampler" 2>/dev/null || true' EXIT

date -u +%s > "$run_dir/started"
set +e
./loadbot run -target "ws://$node:8080/ws" -metrics "http://$node:6060/metrics" -rooms 10000 -2v2 50 -ramp 5m -hold 30m -out "$run_dir/report.json" "$@" > "$run_dir/report.txt" 2> "$run_dir/bot.log"
status=$?
set -e
curl -fsS "http://$node:6060/metrics" > "$run_dir/final.prom" || true
curl -fsS "http://$node:6061/debug/pprof/heap" > "$run_dir/heap.pprof" || true
cat "$run_dir/report.txt"
exit "$status"
