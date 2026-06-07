Project structure
==================

This project follows the [Standard Go Project Layout convention](https://github.com/golang-standards/project-layout)
and the [Organizing a Go module](https://go.dev/doc/modules/layout) documentation.

The applications of the project are defined in `cmd/`. For example:

- **`cmd/jobs-server`** — HTTP API for jobs metadata (enqueue, list, cancel, …). Runs with **`task run-jobs-server`** or **`task run`**. One MongoDB jobs client; no Pulsar, no change stream.
- **`cmd/jobs-dispatcher`** — dispatch worker (change stream + poll fallback → Pulsar). Runs with **`task run-jobs-dispatcher`**. MongoDB jobs client + dedicated watch client + Pulsar producer. Bridges to the API via **`job_metadata`** in MongoDB.
- **`cmd/jobs-cli`** — Cobra CLI with operational parity to the jobs HTTP API: connects via **`mongodb.OpenMongoJobs`**, calls **`MetadataService`**, and writes JSON (default) or table output. Run with **`task run-jobs-cli`** or **`go run ./cmd/jobs-cli`**.
- **`cmd/migrate`** — golang-migrate runner for `migrations/`; used by the Compose **`migrate`** service and manual schema applies.

**`cmd/jobs-cli` layout**

- **`main.go`**, **`root.go`** — entrypoint and subcommand registration
- **`app/`** — shared runtime (`Bootstrap`, output format)
- **`commands/`** — subcommands (`ping`, `create`, `get`, `list`, `stats`, `fail`, `cancel`, `retry`, `logs`, `seed`)
- **`output/`** — JSON and table formatters

**`internal/`**

- **`internal/config`** — typed environment config (`MongoConfig`, `PulsarConfig`, `MongoDispatchWorkerConfig`, …) with validation. See [`internal/config/README.md`](../../internal/config/README.md).
- **`internal/testutil`** — small test helpers (e.g. `ModuleRoot`, `JobTopicsConfigPath` for resolving `config/job-topics.yaml` from the module root).
- **`internal/jobs/metadata/`** — jobs metadata domain: `JobMetadata` / `JobMetadataModel`, `JobLog`, `JobsReader` / `JobsWriter`, and related types. Unit tests run with `go test`.
- **`internal/jobs/mongodb/`** — MongoDB persistence (`MongoJobsReader` / `MongoJobsWriter`, `OpenMongoJobs`, change-stream and poll helpers). Integration tests use the `integration` build tag and expect a running MongoDB (see `docs/dev/setup.md`).
- **`internal/jobs/service/`** — **`MetadataService`** (business logic shared by **`jobs-server`** and **`jobs-cli`**); **`EnqueueService`** (topic resolution for HTTP enqueue on **`jobs-server`** only).
- **`internal/jobs/http/`** — Gin HTTP handlers for the jobs API (`JobsHandler`).
- **`internal/jobs/dispatch/`** — async dispatch worker (change stream + poll fallback, saga orchestration). Transport-agnostic interfaces in `types.go`; unit tests only. See [dispatch-worker.md](../architecture/dispatch-worker.md).
- **`internal/jobs/dispatchruntime/`** — composition root for the dispatch process: `New` wires Mongo watch + poll, Pulsar publish, and `DispatchHandler`/`DispatchWorker`. Used by **`cmd/jobs-dispatcher`** and saga integration tests.
- **`internal/jobs/integrationtest/`** — cross-package integration tests (`//go:build integration`) for dispatch saga and future end-to-end flows.
- **`internal/jobs/pulsar/`** — Pulsar client wrapper, topic resolver (`config/job-topics.yaml`), producer, and `DispatchPublisher`.

**`config/`**

- **`job-topics.yaml`** — job name → Pulsar topic manifest (loaded by `jobs-server` enqueue and validated at startup).

**`migrations/`**

- Numbered golang-migrate JSON files applied by **`cmd/migrate`**. See [migrations/README.md](../../migrations/README.md).

Other packages will appear here as services grow; the architecture summary is in `docs/architecture/intro.md`.
