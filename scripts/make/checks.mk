# Build and static checks (format, lint, etc.)

.PHONY: build format

build:
	go build -o bin/jobs ./cmd/jobs

format:
	go fmt ./...
