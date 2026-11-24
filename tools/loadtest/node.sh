#!/usr/bin/env bash
set -euo pipefail

private_addr="${1:?usage: node.sh <private address> [run dir]}"
run_dir="${2:-node-$(date -u +%Y%m%dT%H%M%SZ)}"
mkdir -p "$run_dir"
ulimit -n 65536

{
  date -u +%FT%TZ
  uname -a
  lscpu
  free -b
  ./loadbot -h 2>&1 | head -n 1 || true
} > "$run_dir/node.txt"

vmstat -n -t 10 > "$run_dir/vmstat.log" &
sampler=$!
trap 'kill "$sampler" 2>/dev/null || true' EXIT

HD_LISTEN_ADDR="$private_addr:8080" HD_DEBUG_ADDR="$private_addr:6060" HD_PPROF_ADDR="$private_addr:6061" HD_LOG_LEVEL="${HD_LOG_LEVEL:-warn}" \
  ./loadbot serve > "$run_dir/server.log" 2>&1
