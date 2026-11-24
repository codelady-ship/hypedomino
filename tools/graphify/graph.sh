#!/usr/bin/env bash
set -euo pipefail

version="0.9.102"
cd "$(git rev-parse --show-toplevel)"

installed=$(graphify --version 2>/dev/null | awk '{ print $2 }' || true)
if [ "$installed" != "$version" ]; then
  echo "graphify $version is required, found '${installed:-none}': uv tool install graphifyy==$version" >&2
  exit 1
fi

export GRAPHIFY_BACKEND=none
if [ -f graphify-out/graph.json ] && [ "${1:-}" != "--rebuild" ]; then
  graphify update .
else
  graphify extract . --code-only
fi
