#!/usr/bin/env bash
set -u

lint_status=0
format_status=0

echo "task lint"
if ! task lint; then
	lint_status=1
	echo "Error: lint failed" >&2
fi

echo "task format"
if ! task format; then
	format_status=1
	echo "Error: format failed" >&2
fi

if [ "$lint_status" -ne 0 ] || [ "$format_status" -ne 0 ]; then
	echo >&2
	echo "One or more checks failed:" >&2
	[ "$lint_status" -ne 0 ] && echo "  - lint" >&2
	[ "$format_status" -ne 0 ] && echo "  - format" >&2
	exit 1
fi

echo "DONE! All checks passed."
