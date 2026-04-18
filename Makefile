# Root Makefile — grouped targets live in scripts/*.mk (for local use and CI)

include scripts/make/docker.mk
include scripts/make/tests.mk
include scripts/make/checks.mk
include scripts/make/dev.mk

.PHONY: run

run:
	go run ./cmd/jobs
