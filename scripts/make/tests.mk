# Test targets (add more as CI grows: race, coverage, integration, etc.)

.PHONY: test

test:
	go test ./...
