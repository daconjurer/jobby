#!/usr/bin/env bash
# Start Compose background services for integration CI and wait until ready.
# Does not build or run migrate — callers handle migrate separately.
#
# Usage:
#   ./scripts/ci-start-compose-services.sh "mongodb mongo-init migrate"
#
# Exit codes:
#   0: Background services ready
#   1: Error

set -euo pipefail

services="${1:-}"
if [ -z "$services" ]; then
  echo "Usage: $0 <space-separated compose services>" >&2
  exit 1
fi

background=""
wait_services=""

for service in $services; do
  if [ "$service" = "migrate" ]; then
    continue
  fi
  background="${background:+$background }$service"
  wait_services="${wait_services:+$wait_services }$service"
done

if [ -z "$background" ]; then
  echo "compose services must include at least one non-migrate service" >&2
  exit 1
fi

docker compose pull $background
docker compose up -d $background
./scripts/wait-for-compose-services.sh "$wait_services"
