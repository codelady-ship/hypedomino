#!/usr/bin/env bash
set -euo pipefail

run_root="${1:?usage: curve.sh <local.sh run dir> [ramp seconds]}"
ramp="${2:-60}"

field() {
  awk -v key="$2" '$1 == "#" && $2 == key { print $4 }' "$1" 2>/dev/null
}

peak_gauge() {
  cat "$1"/metrics/*.prom | awk -v name="$2" '$1 == name && $2 > peak { peak = $2 } END { printf "%.0f", peak }'
}

bucket_p99() {
  awk -F'[{}" ]+' -v name="$2" '
    $1 == name "_bucket" { bound[n + 0] = $3; cum[n + 0] = $NF; n++ }
    $1 == name "_count" { count = $2 }
    END { if (!count) { print "-"; exit } for (i = 0; i < n; i++) if (cum[i] >= count * 0.99) { printf "%g", bound[i] * 1000; exit } }' "$1"
}

container() {
  awk -v name="$2" -v from="$3" '
    $2 == name && $1 >= from { cpu = $3; sub(/%/, "", cpu); sum += cpu; n++; mem = $4; if (mem ~ /GiB$/) { sub(/GiB/, "", mem); mem *= 1024 } else sub(/MiB/, "", mem); if (mem > peak) peak = mem }
    END { if (n) printf "%.0f %.0f", sum / n, peak; else print "- -" }' "$1"
}

pg_delta() {
  paste -d' ' "$1/before.pg" "$1/after.pg" | awk -v s="$2" '{ printf "%.0f %.0f", ($6 - $1) / s, ($7 - $2) / s }'
}

redis_rate() {
  local before after
  before=$(awk -F: '/^total_commands_processed/ { print $2 + 0 }' "$1/before.redis")
  after=$(awk -F: '/^total_commands_processed/ { print $2 + 0 }' "$1/after.redis")
  awk -v a="$before" -v b="$after" -v s="$2" 'BEGIN { printf "%.0f", (b - a) / s }'
}

redis_eval_us() {
  tr -d '\r' < "$1/after.redis" | awk -F'[=,]' '/^cmdstat_evalsha/ { print $6 }'
}

echo "| Rooms | Peak conns | Moves/s | p50 / p95 / p99 ms | Steady p99 / worst window ms | Errors % | Server p99 ms | Node CPU % (of 1 core) / peak MiB | Bot CPU % | Peak live heap MiB | Peak goroutines | GC cycles / GC CPU % | PG commits/s, rows/s | Redis cmds/s, script us | Begin / claim p99 ms | Result |"
echo "|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|"
for rooms_dir in $(ls -d "$run_root"/rooms-* | awk -F"rooms-" '{ print $NF }' | sort -n); do
  dir="$run_root/rooms-$rooms_dir"
  [ -f "$dir/final.prom" ] || continue
  report="$dir/report.txt"
  started=$(cat "$dir/started")
  rooms=$(awk '/^duration/ { gsub(/,/, ""); print $4 }' "$report")
  peak=$(awk '/^duration/ { gsub(/,/, ""); print $7 }' "$report")
  seconds=$(awk '/^duration/ { d = $2; sub(/,/, "", d); s = 0; if (d ~ /h/) { split(d, h, "h"); s += h[1] * 3600; d = h[2] } if (d ~ /m/) { split(d, m, "m"); s += m[1] * 60; d = m[2] } sub(/s/, "", d); s += d; print s }' "$report")
  moves=$(awk '/^moves/ { sub(/:/, "", $2); print $2 }' "$report")
  latency=$(awk '/^moves/ { for (i = 1; i <= NF; i++) { if ($i == "p50") a = $(i + 1); if ($i == "p95") b = $(i + 1); if ($i == "p99") c = $(i + 1) } print a " / " b " / " c }' "$report")
  steady=$(awk '/^steady state/ { gsub(/,/, ""); print $6 " / " $10 }' "$report")
  errors=$(awk '/error rate/ { r = $NF; sub(/%/, "", r); print r }' "$report")
  result=$(awk '/^PASS$/ { print "pass" } /^FAIL/ { print "fail" }' "$report")
  final="$dir/final.prom"
  heap=$(peak_gauge "$dir" go_heap_live_bytes | awk '{ printf "%.0f", $1 / 1048576 }')
  goroutines=$(peak_gauge "$dir" go_goroutines)
  node=$(container "$dir/stats.log" hd-load-node $((started + ramp)))
  bot=$(container "$dir/stats.log" hd-load-bot $((started + ramp)) | cut -d' ' -f1)
  gc_cycles=$(( $(field "$dir/memstats.end" NumGC || echo 0) ))
  gc_cpu=$(awk -v f="$(field "$dir/memstats.end" GCCPUFraction)" 'BEGIN { printf "%.2f", f * 100 }')
  read -r commits rows <<< "$(pg_delta "$dir" "$seconds")"
  echo "| $rooms | $peak | $(awk -v m="$moves" -v s="$seconds" 'BEGIN { printf "%.0f", m / s }') | ${latency//ms/} | ${steady//ms/} | $errors | $(bucket_p99 "$final" move_roundtrip_seconds) | ${node% *} / ${node#* } | $bot | $heap | $goroutines | $gc_cycles / $gc_cpu | $commits, $rows | $(redis_rate "$dir" "$seconds"), $(redis_eval_us "$dir") | $(bucket_p99 "$final" match_begin_seconds) / $(bucket_p99 "$final" route_claim_seconds) | $result |"
done
