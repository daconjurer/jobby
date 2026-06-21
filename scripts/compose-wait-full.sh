#!/usr/bin/env bash
# Wait for the full E2E Compose stack to become ready.
# Polls container healthchecks (Mongo, Pulsar, jobs-server) and host HTTP /health.
#
# Usage:
#   ./scripts/compose-wait-full.sh [timeout_seconds] [interval_seconds]
#
# Exit codes:
#   0: Full stack ready for black-box E2E tests
#   1: Timeout or error

set -euo pipefail

TIMEOUT="${1:-180}"
INTERVAL="${2:-5}"
JOBS_API_BASE_URL="${JOBS_API_BASE_URL:-http://localhost:3001}"

echo "Waiting for E2E stack (timeout: ${TIMEOUT}s, interval: ${INTERVAL}s)"

./scripts/wait-for-compose-services.sh "mongodb pulsar jobs-server" "$TIMEOUT" "$INTERVAL"

elapsed=0
while [ "$elapsed" -lt "$TIMEOUT" ]; do
  if curl -sf "${JOBS_API_BASE_URL}/health" > /dev/null; then
    echo "jobs-server HTTP health OK at ${JOBS_API_BASE_URL}/health"
    # Brief buffer for jobs-dispatcher and jobs-executor to connect to Mongo/Pulsar.
    sleep 5
    echo "Full stack ready for E2E tests"
    exit 0
  fi

  echo "  [$elapsed/${TIMEOUT}s] waiting for ${JOBS_API_BASE_URL}/health"
  sleep "$INTERVAL"
  elapsed=$((elapsed + INTERVAL))
done

echo "Timeout waiting for jobs-server HTTP health at ${JOBS_API_BASE_URL}/health" >&2
exit 1
