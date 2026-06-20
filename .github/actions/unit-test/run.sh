#!/bin/bash
set -euo pipefail

# Avoid Go trying to stamp builds with git metadata (`error obtaining VCS status: exit status 128`)
# when the Actions workspace is bind-mounted into the CI container as a non-default owner.
export GOFLAGS=-buildvcs=false

if [[ -n "${GITHUB_WORKSPACE:-}" && -e "${GITHUB_WORKSPACE}/.git" ]]; then
	git config --global --add safe.directory "${GITHUB_WORKSPACE}"
fi

echo "Running unit tests..."
if ! task test; then
	echo "Error: unit tests failed." >&2
	exit 1
fi
echo "✓ Unit tests passed"
