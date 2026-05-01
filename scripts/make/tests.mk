# Test targets (add more as CI grows: race, coverage, integration, etc.)

.PHONY: test test-integration test-verbose test-coverage coverage coverage-html clean-coverage

test:
	go test ./...

# Requires MongoDB (see compose.yml). Runs one package and one test at a time — no parallel tests.
# -v lists each test/subtest; the summary line counts --- PASS completions (subtests count separately).
test-integration:
	@go test -tags=integration -p 1 -parallel 1 -count=1 -v ./...

test-verbose:
	go test -v ./...

test-coverage:
	go test -cover ./...

coverage:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out
	@echo ""
	@echo "Run 'make coverage-html' to view detailed coverage in browser"

coverage-html:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

clean-coverage:
	rm -f coverage.out
