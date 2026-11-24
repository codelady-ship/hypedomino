#!/usr/bin/env bash
set -euo pipefail

dockerfile="$(cd "$(dirname "$0")" && pwd)/Dockerfile"
postgres_major=$(sed -nE 's/^FROM postgres:([0-9]+)-.*/\1/p' "$dockerfile")
walg_version=$(sed -nE 's/^ARG WALG_VERSION=(v[0-9.]+)$/\1/p' "$dockerfile")
[ -n "$postgres_major" ] && [ -n "$walg_version" ] || { echo "cannot read the PostgreSQL major or the WAL-G version from $dockerfile" >&2; exit 1; }
echo "${postgres_major}-walg-${walg_version}"
