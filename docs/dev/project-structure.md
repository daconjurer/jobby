Project structure
==================

This project follows the [Standard Go Project Layout convention](https://github.com/golang-standards/project-layout)
and the [Organizing a Go module](https://go.dev/doc/modules/layout) documentation.

The applications of the project are defined in `cmd/`. For example:

- **`cmd/jobs-cli`** — minimal CLI that connects to MongoDB using `internal/jobs/metadata` (`OpenMongoJobs`).

**`internal/`**

- **`internal/jobs/metadata/`** — jobs metadata domain: `JobMetadata` / `JobMetadataModel`, `JobLog`, `JobsReader` / `JobsWriter`, and `MongoJobsReader` / `MongoJobsWriter` (MongoDB persistence for `job_metadata` and `job_logs`). Unit tests run with `go test`; integration tests use the `integration` build tag and expect a running MongoDB (see `docs/dev/setup.md`).

Other packages will appear here as services grow; the architecture summary is in `docs/architecture/intro.md`.
