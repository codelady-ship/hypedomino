#!/usr/bin/env bash
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
repo=$(cd "$here/../.." && pwd)
cluster=${1:?usage: kind-postgres.sh <kind cluster name>}
namespace=hypedomino
image="ghcr.io/alievs-corp/hypedomino-postgres-walg:$(bash "$here/image-tag.sh")"
bucket=hd-backups
access_key="smoke$(openssl rand -hex 6)"
secret_key="$(openssl rand -hex 16)"
work=$(mktemp -d)

k() {
  kubectl -n "$namespace" "$@"
}

diagnose() {
  k get pods,pvc,jobs -o wide || true
  k describe pod postgres-0 | tail -40 || true
  k logs postgres-0 --tail=60 || true
  k logs -l job-name --tail=60 --prefix || true
}
trap 'status=$?; [ "$status" = 0 ] || diagnose; rm -rf "$work"' EXIT

sql() {
  k exec postgres-0 -c postgres -- psql -qtAX -v ON_ERROR_STOP=1 -U postgres -d hypedomino -c "$1"
}

run_job_from() {
  k create job "$2" --from="cronjob/$1" >/dev/null
  k wait --for=condition=complete --timeout=240s "job/$2"
  k logs "job/$2"
}

docker build -q -t "$image" "$here" >/dev/null
kind load docker-image "$image" --name "$cluster"
kubectl create namespace "$namespace" >/dev/null

k apply -f - >/dev/null <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: objectstore
spec:
  selector:
    matchLabels:
      app: objectstore
  template:
    metadata:
      labels:
        app: objectstore
    spec:
      containers:
        - name: seaweedfs
          image: chrislusf/seaweedfs:4.53
          args: ["server", "-s3", "-s3.port=9000", "-dir=/data"]
          env:
            - name: AWS_ACCESS_KEY_ID
              value: $access_key
            - name: AWS_SECRET_ACCESS_KEY
              value: $secret_key
---
apiVersion: v1
kind: Service
metadata:
  name: objectstore
spec:
  selector:
    app: objectstore
  ports:
    - port: 9000
EOF
k rollout status deployment/objectstore --timeout=180s
k run make-bucket --rm -i --restart=Never --image="$image" --image-pull-policy=Never --command -- bash -c "
  for i in \$(seq 1 60); do
    curl -sS -o /dev/null --aws-sigv4 aws:amz:us-east-1:s3 --user $access_key:$secret_key -X PUT http://objectstore:9000/$bucket && exit 0
    sleep 1
  done
  exit 1" >/dev/null

k create secret generic postgres-superuser --from-literal=POSTGRES_PASSWORD="$(openssl rand -hex 16)" >/dev/null
k create secret generic walg-credentials \
  --from-literal=AWS_ACCESS_KEY_ID="$access_key" \
  --from-literal=AWS_SECRET_ACCESS_KEY="$secret_key" \
  --from-literal=WALG_LIBSODIUM_KEY="$(openssl rand -hex 32)" \
  --from-literal=PG_WALG_PASSWORD="$(openssl rand -hex 16)" >/dev/null
openssl req -x509 -newkey rsa:2048 -nodes -days 1 -subj "/CN=postgres.$namespace.svc" \
  -keyout "$work/tls.key" -out "$work/tls.crt" 2>/dev/null
k create secret tls postgres-tls --cert="$work/tls.crt" --key="$work/tls.key" >/dev/null

kubectl kustomize "$repo/deploy/k8s/backup" | k apply -f - >/dev/null
k patch configmap walg-config --type merge \
  -p "{\"data\":{\"WALG_S3_PREFIX\":\"s3://$bucket/postgres\",\"AWS_ENDPOINT\":\"http://objectstore:9000\",\"AWS_REGION\":\"us-east-1\"}}" >/dev/null
kubectl kustomize "$repo/deploy/k8s/postgres" | k apply -f - >/dev/null
k rollout status statefulset/postgres --timeout=300s

[ "$(sql "show archive_mode")" = "on" ] || { echo "archive_mode is not on: postgres-archive-config was not applied"; exit 1; }
[ "$(sql "show ssl")" = "on" ] || { echo "ssl is not on: postgres-config was not applied"; exit 1; }
sql "create table smoke (id int)"
sql "insert into smoke select generate_series(1, 1000)"

run_job_from walg-base-backup base-backup-smoke
sql "select pg_switch_wal()" >/dev/null
for _ in $(seq 1 60); do
  archived=$(sql "select archived_count > 0 and failed_count = 0 from pg_stat_archiver")
  [ "$archived" = "t" ] && break
  sleep 1
done
[ "$archived" = "t" ] || { echo "WAL was not archived by archive_command"; sql "select * from pg_stat_archiver"; exit 1; }

run_job_from walg-retention retention-smoke | tee "$work/retention.log"
grep -q '^base_' "$work/retention.log" || { echo "backup-list printed no base backup"; exit 1; }
echo "kind smoke passed: postgres StatefulSet ready, WAL archived, base backup pushed by the CronJob as role walg over TLS"
