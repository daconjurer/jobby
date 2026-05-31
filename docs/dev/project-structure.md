Project structure
==================

This project follows the [Standard Go Project Layout convention](https://github.com/golang-standards/project-layout)
and the [Organizing a Go module](https://go.dev/doc/modules/layout) documentation.

The applications of the project are defined in `cmd/`. For example:

- **`cmd/jobs-server`** — HTTP server for jobs metadata (runs with **`task run-jobs-server`** or **`task run`**).
- **`cmd/jobs-cli`** — Cobra CLI with full parity to the jobs HTTP API: connects via **`OpenMongoJobs`**, calls **`MetadataService`**, and writes JSON (default) or table output. Run with **`task run-jobs-cli`** or **`go run ./cmd/jobs-cli`**.

**`cmd/jobs-cli` layout**

- **`main.go`**, **`root.go`** — entrypoint and subcommand registration
- **`app/`** — shared runtime (`Bootstrap`, output format)
- **`commands/`** — subcommands (`ping`, `create`, `get`, `list`, `stats`, `fail`, `cancel`, `retry`, `logs`, `seed`)
- **`output/`** — JSON and table formatters

**`internal/`**

- **`internal/jobs/metadata/`** — jobs metadata domain: `JobMetadata` / `JobMetadataModel`, `JobLog`, `JobsReader` / `JobsWriter`, and `MongoJobsReader` / `MongoJobsWriter` (MongoDB persistence for `job_metadata` and `job_logs`). Unit tests run with `go test`; integration tests use the `integration` build tag and expect a running MongoDB (see `docs/dev/setup.md`).
- **`internal/jobs/service/`** — **`MetadataService`** (business logic shared by **`jobs-server`** and **`jobs-cli`**)
- **`internal/jobs/handler/`** — HTTP handlers for **`jobs-server`**

Other packages will appear here as services grow; the architecture summary is in `docs/architecture/intro.md`.
