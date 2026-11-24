#!/usr/bin/env bash
set -euo pipefail

run_dir="${1:?usage: check.sh <run dir with vmstat.log copied in> [ramp minutes] [hold minutes]}"
ramp="${2:-5}"
hold="${3:-30}"
started=$(cat "$run_dir/started")
failed=0

heap_at() {
  local want=$(( started + ($1) * 60 ))
  local file
  file=$(ls "$run_dir"/metrics/*.prom | awk -F/ -v want="$want" '{ n = $NF; sub(/\.prom$/, "", n); d = n - want; if (d < 0) d = -d; if (best == "" || d < bestd) { best = $0; bestd = d } } END { print best }')
  awk '$1 == "go_heap_live_bytes" { print $2 }' "$file"
}

early_minute=$(( hold < 30 ? hold / 3 : 10 ))
early=$(heap_at "$ramp + $early_minute")
late=$(heap_at "$ramp + $hold")
drift=$(awk -v a="$early" -v b="$late" 'BEGIN { printf "%.2f", (b - a) * 100 / a }')
echo "heap hold minute $early_minute: $early bytes, hold minute $hold: $late bytes, drift $drift%"
awk -v d="$drift" 'BEGIN { exit !(d > 5 || d < -5) }' && { echo "FAIL heap drift over 5%"; failed=1; }

cpu=$(awk -v skip="$(( ramp * 6 + 2 ))" -v take="$(( hold * 6 ))" 'NR > skip && NR <= skip + take { busy += 100 - $15; n++ } END { if (n) printf "%.1f", busy / n }' "$run_dir/vmstat.log")
echo "average CPU during the hold: $cpu%"
awk -v c="$cpu" 'BEGIN { exit !(c == "" || c >= 80) }' && { echo "FAIL average CPU not under 80%"; failed=1; }
steal=$(awk -v skip="$(( ramp * 6 + 2 ))" -v take="$(( hold * 6 ))" 'NR > skip && NR <= skip + take { stolen += $17; n++ } END { if (n) printf "%.1f", stolen / n }' "$run_dir/vmstat.log")
echo "average CPU steal during the hold: $steal%"

awk -F'[{}" ]+' '
  $1 ~ /_seconds_bucket$/ { name = $1; sub(/_bucket$/, "", name); if (!(name in rows)) order[k++] = name; row = rows[name]++ + 0; bound[name, row] = $3; cum[name, row] = $NF }
  $1 ~ /_seconds_count$/ { name = $1; sub(/_count$/, "", name); count[name] = $2 }
  $1 == "hd_store_failures" { failures = $2 }
  END {
    for (j = 0; j < k; j++) {
      name = order[j]
      if (count[name] == 0) continue
      for (i = 0; i < rows[name]; i++) if (cum[name, i] >= count[name] * 0.99) { print "server " name " p99 <= " bound[name, i] " s over " count[name]; break }
    }
    if (failures != "") print "store failures " failures
  }' "$run_dir/final.prom"

exit "$failed"
