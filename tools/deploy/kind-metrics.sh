#!/usr/bin/env bash
set -euo pipefail

namespace="${1:-hd-ci}"
probe_image=curlimages/curl:8.11.1

curl_from() {
  local from="$1" name="curl-$RANDOM" phase=""
  shift
  kubectl -n "$from" run "$name" --image="$probe_image" --restart=Never --quiet --command -- curl -sS --fail --max-time 5 "$@" >/dev/null
  for _ in $(seq 120); do
    phase=$(kubectl -n "$from" get pod "$name" -o jsonpath='{.status.phase}')
    [ "$phase" = Succeeded ] || [ "$phase" = Failed ] && break
    sleep 1
  done
  kubectl -n "$from" logs "$name" 2>&1 || true
  kubectl -n "$from" delete pod "$name" --wait=false >/dev/null
  [ "$phase" = Succeeded ]
}

kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
kubectl -n "$namespace" rollout status deploy/server --timeout=180s
pod=$(kubectl -n "$namespace" get pods -l app.kubernetes.io/name=server --field-selector=status.phase=Running -o go-template='{{range .items}}{{if not .metadata.deletionTimestamp}}{{println .metadata.name}}{{end}}{{end}}' | head -n 1)
[ -n "$pod" ] || { echo "no running server pod in $namespace"; exit 1; }
pod_ip=$(kubectl -n "$namespace" get pod "$pod" -o jsonpath='{.status.podIP}')
port=$(kubectl -n "$namespace" get deploy/server -o jsonpath='{.spec.template.metadata.annotations.prometheus\.io/port}')
[ "$port" = 6060 ] || { echo "the scrape annotation names port '$port', want 6060"; exit 1; }
if kubectl -n "$namespace" get service/server -o jsonpath='{.spec.ports[*].targetPort}' | grep -qw -e debug -e 6060; then
  echo "the public Service exposes the debug port"
  exit 1
fi

metrics=$(curl_from monitoring "http://$pod_ip:6060/metrics") || { echo "monitoring could not scrape http://$pod_ip:6060/metrics: $metrics"; exit 1; }
for series in move_roundtrip_seconds hd_http_responses_total hd_connections; do
  grep -q "^# TYPE $series " <<<"$metrics" || { echo "the monitoring scrape is missing $series"; exit 1; }
done
echo "monitoring scraped http://$pod_ip:6060/metrics"

status=$(curl_from monitoring -o /dev/null -w '%{http_code}
' "http://$pod_ip:6060/debug/pprof/heap") && { echo "the scrape port served a heap profile to monitoring"; exit 1; }
grep -qx 404 <<<"$status" || { echo "the scrape port answered pprof with '$status', want 404"; exit 1; }
echo "scrape port does not serve pprof"

if curl_from monitoring "http://$pod_ip:6061/debug/pprof/" >/dev/null; then
  echo "the pprof listener answered the pod network"
  exit 1
fi
echo "pprof listener refused from the pod network"

kubectl -n "$namespace" port-forward "pod/$pod" 16061:6061 >/dev/null &
forward=$!
trap 'kill "$forward" 2>/dev/null || true' EXIT
for _ in $(seq 30); do
  curl -sS --fail --max-time 5 -o /dev/null "http://127.0.0.1:16061/debug/pprof/heap" 2>/dev/null && profiled=yes && break
  sleep 1
done
[ "${profiled:-}" = yes ] || { echo "kubectl port-forward did not reach the pprof listener"; exit 1; }
echo "pprof reached through kubectl port-forward"

curl_from "$namespace" "http://$pod_ip:8080/healthz" >/dev/null || { echo "the game port is not reachable from $namespace"; exit 1; }
echo "game port reachable from $namespace"

if curl_from "$namespace" "http://$pod_ip:6060/metrics" >/dev/null; then
  echo "the debug port answered a pod outside the monitoring namespace"
  exit 1
fi
echo "debug port refused from $namespace"
