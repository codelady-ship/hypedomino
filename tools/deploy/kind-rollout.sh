#!/usr/bin/env bash
set -euo pipefail

cluster="${1:?usage: kind-rollout.sh <kind cluster> [image]}"
image="${2:-hypedomino-server:ci}"
namespace=hd-ci
rollback_budget_seconds=300
work=$(mktemp -d)
watcher=""
next_release=""
trap '[ -n "$watcher" ] && kill "$watcher" 2>/dev/null; [ -n "$next_release" ] && rm -rf "$next_release"; true' EXIT

applied_migrations() {
  kubectl -n "$namespace" exec deploy/postgres -- psql -qtAX -U postgres -d hypedomino -c 'select count(*), max(version_id) from goose_db_version'
}

kind load docker-image "$image" --name "$cluster"
kubectl apply -f deploy/k8s/overlays/ci/namespace.yaml
kubectl -n "$namespace" apply -f tools/deploy/ci-postgres.yaml
kubectl -n "$namespace" rollout status deploy/postgres --timeout=180s

bash tools/deploy/apply-release.sh deploy/k8s/overlays/ci "$namespace"
first=$(applied_migrations)
newest=$(ls server/internal/db/migrations | sed -nE 's/^0*([0-9]+)_.*\.sql$/\1/p' | sort -n | tail -1)
echo "after the first release: rows|version $first, newest migration file $newest"
[ "${first#*|}" = "$newest" ] || { echo "the migration job did not reach version $newest"; exit 1; }

bash tools/deploy/apply-release.sh deploy/k8s/overlays/ci "$namespace"
second=$(applied_migrations)
echo "after the second release: rows|version $second"
[ "$second" = "$first" ] || { echo "the second migration run changed goose_db_version"; exit 1; }

kubectl -n "$namespace" scale deploy/postgres --replicas=0
kubectl -n "$namespace" wait --for=delete pod -l app.kubernetes.io/name=postgres --timeout=120s
generation=$(kubectl -n "$namespace" get deploy/server -o jsonpath='{.metadata.generation}')
next_release=$(mktemp -d deploy/k8s/overlays/next-release.XXXXXX)
cat >"$next_release/kustomization.yaml" <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../ci
patches:
  - target:
      kind: Deployment
      name: server
    patch: |
      - op: add
        path: /spec/template/metadata/annotations
        value:
          hypedomino/next-release: "1"
EOF
if bash tools/deploy/apply-release.sh "$next_release" "$namespace"; then
  echo "a release went on while its migration job could not reach the database"
  exit 1
fi
kubectl -n "$namespace" get job/migrate -o jsonpath='{.status.conditions[?(@.type=="Failed")].status}' | grep -qx True || { echo "the migration job is not marked Failed"; exit 1; }
[ "$(kubectl -n "$namespace" get deploy/server -o jsonpath='{.metadata.generation}')" = "$generation" ] || { echo "the server changed after a failed migration"; exit 1; }

replicas=$(kubectl -n "$namespace" get deploy/server -o jsonpath='{.spec.replicas}')

old_pods=$(kubectl -n "$namespace" get pods -l app.kubernetes.io/name=server -o jsonpath='{.items[*].metadata.name}')
followers=()
for pod in $old_pods; do
  kubectl -n "$namespace" logs -f "$pod" >"$work/$pod.log" 2>&1 &
  followers+=($!)
done

watch_available() {
  while true; do
    kubectl -n "$namespace" get deploy/server -o jsonpath='{.status.availableReplicas}{"\n"}' >>"$work/available.txt" 2>/dev/null || true
    sleep 1
  done
}
watch_available &
watcher=$!

kubectl -n "$namespace" rollout restart deploy/server
kubectl -n "$namespace" rollout status deploy/server --timeout=300s
for pid in "${followers[@]}"; do
  timeout 120 tail --pid="$pid" -f /dev/null || { echo "an old pod did not stop within 120s"; exit 1; }
done
kill "$watcher"
watcher=""

min_available=$(grep -E '^[0-9]+$' "$work/available.txt" | sort -n | head -1)
echo "available replicas during the rollout: min ${min_available:-none} of $replicas"
[ "${min_available:-0}" -ge "$replicas" ] || { echo "capacity dropped below $replicas during the rollout"; exit 1; }

for pod in $old_pods; do
  echo "--- $pod"
  cat "$work/$pod.log"
  grep -q '"msg":"draining"' "$work/$pod.log" || { echo "$pod did not drain"; exit 1; }
  grep -q '"msg":"stopped"' "$work/$pod.log" || { echo "$pod did not stop cleanly"; exit 1; }
done

started=$(date +%s)
kubectl -n "$namespace" rollout undo deploy/server
kubectl -n "$namespace" rollout status deploy/server --timeout="${rollback_budget_seconds}s"
elapsed=$(( $(date +%s) - started ))
echo "rollback serving on the previous revision after ${elapsed}s (budget ${rollback_budget_seconds}s)"
[ "$elapsed" -lt "$rollback_budget_seconds" ]
