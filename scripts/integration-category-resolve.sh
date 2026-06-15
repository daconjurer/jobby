#!/usr/bin/env sh
# Smoke test: TEST_CATEGORY → Go package path mapping (Phase 4).
# Mirrors Taskfile INTEGRATION_CATEGORY_PACKAGES resolution.
set -eu

resolve_category() {
  case "${1:-all}" in
    mongodb) echo './internal/jobs/mongodb/...' ;;
    pulsar) echo './internal/jobs/pulsar/...' ;;
    dispatch) echo './internal/jobs/integrationtest/...' ;;
    http) echo './internal/jobs/http/...' ;;
    cli) echo './cmd/jobs-cli/commands/...' ;;
    e2e) echo './internal/jobs/executor/...' ;;
    all) echo './...' ;;
    *)
      echo "unknown TEST_CATEGORY: ${1}" >&2
      echo "Valid values: mongodb, pulsar, dispatch, http, cli, e2e, all" >&2
      return 1
    ;;
  esac
}

expected_mongodb='./internal/jobs/mongodb/...'
expected_pulsar='./internal/jobs/pulsar/...'
expected_dispatch='./internal/jobs/integrationtest/...'
expected_http='./internal/jobs/http/...'
expected_cli='./cmd/jobs-cli/commands/...'
expected_e2e='./internal/jobs/executor/...'
expected_all='./...'

[ "$(resolve_category mongodb)" = "$expected_mongodb" ] || exit 1
[ "$(resolve_category pulsar)" = "$expected_pulsar" ] || exit 1
[ "$(resolve_category dispatch)" = "$expected_dispatch" ] || exit 1
[ "$(resolve_category http)" = "$expected_http" ] || exit 1
[ "$(resolve_category cli)" = "$expected_cli" ] || exit 1
[ "$(resolve_category e2e)" = "$expected_e2e" ] || exit 1
[ "$(resolve_category all)" = "$expected_all" ] || exit 1
[ "$(resolve_category)" = "$expected_all" ] || exit 1

if resolve_category bogus 2>/dev/null; then
  echo "expected bogus category to fail" >&2
  exit 1
fi

echo "integration category resolution: ok"
