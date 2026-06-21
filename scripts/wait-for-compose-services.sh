#!/usr/bin/env bash
# Wait for Docker Compose services to become healthy or complete successfully.
# Used by CI integration test jobs (Phase 5).
#
# Usage:
#   ./scripts/wait-for-compose-services.sh "mongodb mongo-init migrate" [timeout_seconds] [interval_seconds]
#
# Arguments:
#   $1: Space-separated list of service names to wait for (required)
#   $2: Timeout in seconds (default: 180)
#   $3: Poll interval in seconds (default: 5)
#
# Exit codes:
#   0: All services healthy or completed successfully
#   1: Timeout or error

set -euo pipefail

SERVICES="${1:-}"
TIMEOUT="${2:-180}"
INTERVAL="${3:-5}"

if [ -z "$SERVICES" ]; then
  echo "Usage: $0 <services> [timeout] [interval]" >&2
  echo "Example: $0 'mongodb mongo-init migrate' 180 5" >&2
  exit 1
fi

service_state() {
  local service="$1"
  docker compose ps -a --format json \
    | jq -r --arg svc "$service" 'select(.Service == $svc) | .State' \
    | head -n 1
}

service_health() {
  local service="$1"
  docker compose ps -a --format json \
    | jq -r --arg svc "$service" 'select(.Service == $svc) | .Health // ""' \
    | head -n 1
}

service_exit_code() {
  local service="$1"
  docker compose ps -a --format json \
    | jq -r --arg svc "$service" 'select(.Service == $svc) | .ExitCode // 1' \
    | head -n 1
}

echo "Waiting for services: $SERVICES (timeout: ${TIMEOUT}s, interval: ${INTERVAL}s)"

elapsed=0
while [ $elapsed -lt "$TIMEOUT" ]; do
  all_ready=true

  for service in $SERVICES; do
    state="$(service_state "$service")"
    health="$(service_health "$service")"

    if [ -z "$state" ] || [ "$state" = "null" ]; then
      echo "  [$elapsed/${TIMEOUT}s] $service: not found"
      all_ready=false
      continue
    fi

    case "$state" in
      running)
        if [ "$health" = "healthy" ]; then
          echo "  [$elapsed/${TIMEOUT}s] $service: ready (running, healthy)"
        else
          echo "  [$elapsed/${TIMEOUT}s] $service: running but not healthy yet (health=${health:-none})"
          all_ready=false
        fi
        ;;
      exited)
        exit_code="$(service_exit_code "$service")"
        if [ "$exit_code" = "0" ]; then
          echo "  [$elapsed/${TIMEOUT}s] $service: completed successfully (exit 0)"
        else
          echo "  [$elapsed/${TIMEOUT}s] $service: exited with code $exit_code" >&2
          echo "Failed: $service exited with non-zero code" >&2
          exit 1
        fi
        ;;
      *)
        echo "  [$elapsed/${TIMEOUT}s] $service: state=$state, health=${health:-none} (waiting)"
        all_ready=false
        ;;
    esac
  done

  if $all_ready; then
    echo ""
    echo "All services ready"
    exit 0
  fi

  sleep "$INTERVAL"
  elapsed=$((elapsed + INTERVAL))
done

echo ""
echo "Timeout waiting for services after ${TIMEOUT}s" >&2
exit 1
