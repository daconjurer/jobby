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

echo "Waiting for services: $SERVICES (timeout: ${TIMEOUT}s, interval: ${INTERVAL}s)"

elapsed=0
while [ $elapsed -lt "$TIMEOUT" ]; do
  all_ready=true
  
  # Get all compose services status as JSON array
  services_json=$(docker compose ps --format json 2>/dev/null || echo "[]")
  
  # Check each requested service
  for service in $SERVICES; do
    # Extract this service's state and health
    state=$(echo "$services_json" | jq -r --arg svc "$service" '.[] | select(.Service == $svc) | .State' 2>/dev/null || echo "")
    health=$(echo "$services_json" | jq -r --arg svc "$service" '.[] | select(.Service == $svc) | .Health' 2>/dev/null || echo "")
    
    if [ -z "$state" ]; then
      echo "  [$elapsed/${TIMEOUT}s] $service: not found"
      all_ready=false
      continue
    fi
    
    # Check if service is ready based on state and health
    case "$state" in
      running)
        if [ "$health" = "healthy" ] || [ "$health" = "" ]; then
          echo "  [$elapsed/${TIMEOUT}s] $service: ready (running, health=$health)"
        else
          echo "  [$elapsed/${TIMEOUT}s] $service: running but not healthy yet (health=$health)"
          all_ready=false
        fi
        ;;
      exited)
        # Services like 'migrate' with restart:"no" should exit successfully
        exit_code=$(echo "$services_json" | jq -r --arg svc "$service" '.[] | select(.Service == $svc) | .ExitCode' 2>/dev/null || echo "1")
        if [ "$exit_code" = "0" ]; then
          echo "  [$elapsed/${TIMEOUT}s] $service: completed successfully (exit 0)"
        else
          echo "  [$elapsed/${TIMEOUT}s] $service: exited with code $exit_code" >&2
          echo "Failed: $service exited with non-zero code" >&2
          exit 1
        fi
        ;;
      *)
        echo "  [$elapsed/${TIMEOUT}s] $service: state=$state, health=$health (waiting)"
        all_ready=false
        ;;
    esac
  done
  
  if $all_ready; then
    echo ""
    echo "✓ All services ready"
    exit 0
  fi
  
  sleep "$INTERVAL"
  elapsed=$((elapsed + INTERVAL))
done

echo ""
echo "✗ Timeout waiting for services after ${TIMEOUT}s" >&2
exit 1
