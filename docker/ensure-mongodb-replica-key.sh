#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
KEY="$ROOT/config/mongodb-replica.key"

force=false
if [[ "${1:-}" == "--force" ]]; then
  force=true
fi

if [[ -f "$KEY" && "$force" != true ]]; then
  exit 0
fi

mkdir -p "$(dirname "$KEY")"
if [[ -f "$KEY" ]]; then
  rm -f "$KEY"
fi

openssl rand -base64 756 >"$KEY"
chmod 400 "$KEY"

if [[ "$force" == true ]]; then
  echo "Regenerated $KEY (run task mongo-reset or docker-clean if MongoDB already used a previous key)"
else
  echo "Created $KEY for local Compose replica set rs0"
fi
