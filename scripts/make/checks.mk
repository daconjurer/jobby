# Build and static checks (format, lint, etc.)

.PHONY: build format

build:
	go build ./...

build-jobs-server:
	go build -o bin/jobs-server ./cmd/jobs-server

build-jobs-cli:
	go build -o bin/jobs-cli ./cmd/jobs-cli

format:
	go fmt ./...
