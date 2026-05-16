# Test targets (add more as CI grows: race, coverage, integration, etc.)

# Integration tests (go:build integration) need MongoDB — same flags as test-integration.
INTEGRATION_TEST_FLAGS := -tags=integration -p 1 -parallel 1 -count=1
# Atomic mode so unit + integration profiles can be merged for combined HTML/func reports.
COVERMODE := atomic

COVERAGE_OUT := coverage.out
COVERAGE_UNIT_OUT := coverage-unit.out
COVERAGE_INTEGRATION_OUT := coverage-integration.out
COVERAGE_ALL_OUT := coverage-all.out

.PHONY: test test-integration test-verbose test-coverage \
	coverage coverage-html \
	coverage-integration coverage-integration-html \
	coverage-all coverage-all-html \
	clean-coverage

test:
	go test ./...

# Requires MongoDB (see compose.yml). Runs one package and one test at a time — no parallel tests.
# -v lists each test/subtest; the summary line counts --- PASS completions (subtests count separately).
test-integration:
	@go test $(INTEGRATION_TEST_FLAGS) -v ./...

test-verbose:
	go test -v ./...

test-coverage:
	go test -cover ./...

# Unit tests only (default tag set).
coverage:
	@echo "Generating unit-test coverage report..."
	@go test -covermode=$(COVERMODE) -coverprofile=$(COVERAGE_OUT) ./...
	@go tool cover -func=$(COVERAGE_OUT)
	@echo ""
	@echo "Run 'make coverage-html' for unit coverage in the browser"

coverage-html:
	@echo "Generating unit-test coverage report..."
	@go test -covermode=$(COVERMODE) -coverprofile=$(COVERAGE_OUT) ./...
	@go tool cover -html=$(COVERAGE_OUT)

# Integration tests only (-tags=integration). Requires MongoDB (e.g. docker compose up -d).
coverage-integration:
	@echo "Generating integration-test coverage ($(COVERAGE_INTEGRATION_OUT))..."
	@go test $(INTEGRATION_TEST_FLAGS) -covermode=$(COVERMODE) -coverprofile=$(COVERAGE_INTEGRATION_OUT) ./...
	@go tool cover -func=$(COVERAGE_INTEGRATION_OUT)
	@echo ""
	@echo "Run 'make coverage-integration-html' to open this profile in the browser"

coverage-integration-html:
	@echo "Generating integration-test coverage ($(COVERAGE_INTEGRATION_OUT))..."
	@go test $(INTEGRATION_TEST_FLAGS) -covermode=$(COVERMODE) -coverprofile=$(COVERAGE_INTEGRATION_OUT) ./...
	@go tool cover -html=$(COVERAGE_INTEGRATION_OUT)

# Unit + integration merged profile (integration requires MongoDB).
coverage-all:
	@echo "Generating merged unit + integration coverage ($(COVERAGE_ALL_OUT))..."
	@go test -covermode=$(COVERMODE) -coverprofile=$(COVERAGE_UNIT_OUT) ./...
	@go test $(INTEGRATION_TEST_FLAGS) -covermode=$(COVERMODE) -coverprofile=$(COVERAGE_INTEGRATION_OUT) ./...
	@{ echo "mode: $(COVERMODE)"; grep -h -v "^mode:" $(COVERAGE_UNIT_OUT) $(COVERAGE_INTEGRATION_OUT); } > $(COVERAGE_ALL_OUT)
	@go tool cover -func=$(COVERAGE_ALL_OUT)
	@echo ""
	@echo "Run 'make coverage-all-html' to open the merged report in the browser"

coverage-all-html:
	@echo "Generating merged unit + integration coverage ($(COVERAGE_ALL_OUT))..."
	@go test -covermode=$(COVERMODE) -coverprofile=$(COVERAGE_UNIT_OUT) ./...
	@go test $(INTEGRATION_TEST_FLAGS) -covermode=$(COVERMODE) -coverprofile=$(COVERAGE_INTEGRATION_OUT) ./...
	@{ echo "mode: $(COVERMODE)"; grep -h -v "^mode:" $(COVERAGE_UNIT_OUT) $(COVERAGE_INTEGRATION_OUT); } > $(COVERAGE_ALL_OUT)
	@go tool cover -html=$(COVERAGE_ALL_OUT)

clean-coverage:
	rm -f $(COVERAGE_OUT) $(COVERAGE_UNIT_OUT) $(COVERAGE_INTEGRATION_OUT) $(COVERAGE_ALL_OUT)
