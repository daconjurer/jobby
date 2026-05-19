#!/bin/bash
set -euo pipefail

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
