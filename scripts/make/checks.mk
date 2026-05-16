# Build and static checks (format, lint, etc.)

.PHONY: build format format-check lint lint-check

build:
	go build ./...

build-jobs-server:
	go build -o bin/jobs-server ./cmd/jobs-server

build-jobs-cli:
	go build -o bin/jobs-cli ./cmd/jobs-cli

format-check:
	@files=$$(go list -f '{{$$dir := .Dir}}{{range .GoFiles}}{{$$dir}}/{{.}}{{"\n"}}{{end}}{{range .TestGoFiles}}{{$$dir}}/{{.}}{{"\n"}}{{end}}{{range .XTestGoFiles}}{{$$dir}}/{{.}}{{"\n"}}{{end}}' ./...); \
	if [ -z "$$files" ]; then exit 0; fi; \
	need_fmt=$$(printf '%s\n' "$$files" | xargs gofmt -l 2>/dev/null); \
	if [ -n "$$need_fmt" ]; then \
		echo "gofmt would reformat:"; \
		printf '%s\n' "$$need_fmt"; \
		echo; \
		printf '%s\n' "$$need_fmt" | xargs gofmt -d; \
		echo >&2 "error: code is not gofmt-clean (diff above; fix with: make format)"; \
		exit 1; \
	fi

format:
	go fmt ./...

lint-check:
	golangci-lint run --fix=false ./...

lint:
	golangci-lint run ./...
