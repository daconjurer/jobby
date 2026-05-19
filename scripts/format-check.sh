#!/usr/bin/env bash
# gofmt cleanliness check (formerly the Make format-check target).
set -euo pipefail

files="$(go list -f '{{$dir := .Dir}}{{range .GoFiles}}{{$dir}}/{{.}}{{"\n"}}{{end}}{{range .TestGoFiles}}{{$dir}}/{{.}}{{"\n"}}{{end}}{{range .XTestGoFiles}}{{$dir}}/{{.}}{{"\n"}}{{end}}' ./...)"
if [ -z "${files:-}" ]; then exit 0; fi
need_fmt="$(printf '%s\n' "${files}" | xargs gofmt -l 2>/dev/null || true)"
if [ -n "${need_fmt:-}" ]; then
	echo "gofmt would reformat:"
	printf '%s\n' "${need_fmt}"
	echo
	printf '%s\n' "${need_fmt}" | xargs gofmt -d
	echo >&2 "error: code is not gofmt-clean (diff above; fix with: task format)"
	exit 1
fi
