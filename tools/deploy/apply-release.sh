#!/usr/bin/env bash
set -euo pipefail

overlay="${1:?usage: apply-release.sh <overlay dir> <namespace>}"
namespace="${2:?usage: apply-release.sh <overlay dir> <namespace>}"
rendered=$(mktemp)
trap 'rm -f "$rendered"' EXIT

k() {
  kubectl -n "$namespace" "$@"
}

job_conditions() {
  k get job/migrate -o jsonpath='{range .status.conditions[?(@.status=="True")]}{.type}{" "}{end}'
}

wait_for_migration() {
  local deadline=$(( $(date +%s) + $(k get job/migrate -o jsonpath='{.spec.activeDeadlineSeconds}') + 60 ))
  local conditions=""
  while [ "$(date +%s)" -lt "$deadline" ]; do
    conditions=" $(job_conditions || true)"
    case "$conditions" in
      *" Complete "*) return 0 ;;
      *" Failed "*) return 1 ;;
    esac
    sleep 2
  done
  return 1
}

kubectl kustomize "$overlay" >"$rendered"
k delete job/migrate --ignore-not-found --cascade=foreground --wait=true
kubectl apply -f "$rendered" -l 'app.kubernetes.io/name!=server'
if ! wait_for_migration; then
  k get job/migrate -o wide || true
  k logs -l job-name=migrate --prefix --tail=-1 || true
  echo "migration job did not complete; the server was not applied"
  exit 1
fi
k logs job/migrate
kubectl apply -f "$rendered" -l 'app.kubernetes.io/name!=migrate'
k rollout status deploy/server
