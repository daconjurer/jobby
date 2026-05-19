#!/bin/bash
set -euo pipefail

# Avoid Go trying to stamp builds with git metadata (`error obtaining VCS status: exit status 128`)
# when the Actions workspace is bind-mounted into the CI container as a non-default owner.
export GOFLAGS=-buildvcs=false

if [[ -n "${GITHUB_WORKSPACE:-}" && -e "${GITHUB_WORKSPACE}/.git" ]]; then
	git config --global --add safe.directory "${GITHUB_WORKSPACE}"
fi

echo "Checking formatting..."
if ! task format-check; then
	echo "Error: formatting check failed. Run 'task format' to fix." >&2
	exit 1
fi
echo "✓ Format check passed"

echo "Running linter..."
if ! task lint-check; then
	echo "Error: golangci-lint reported issues." >&2
	exit 1
fi
echo "✓ Lint check passed"

echo "All pre-tests passed! ✓"
