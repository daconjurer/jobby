Architecture overview
======================

This project is a microservices-based system for distributed workloads.

**Stack in this repository**

- **Go** (module `github.com/daconjurer/jobby`, Go 1.25)
- **Gin** for HTTP where services expose REST APIs
- **MongoDB 8** for **job execution metadata** and **job logs** (`job_metadata` and `job_logs` in the `jobby` database)
- **mongo-driver v2** (`go.mongodb.org/mongo-driver/v2`) for MongoDB access
- **Apache Pulsar** for async job dispatch (MongoDB → Pulsar relay)

**Binaries (`cmd/`)**

| Binary | Role |
|--------|------|
| **`jobs-server`** | HTTP API — enqueue, list, cancel, retry jobs. Persists `pending_dispatch` via `EnqueueService`; no Pulsar, no change stream. |
| **`jobs-dispatcher`** | Dispatch worker — change stream + poll fallback → Pulsar publish + status confirmation. |
| **`jobs-cli`** | Cobra CLI with operational parity to the jobs HTTP API: same `MetadataService` for get, list, stats, fail, cancel, retry, logs, and seed. JSON stdout by default; `--output table` for interactive use. |
| **`migrate`** | Applies `migrations/` via golang-migrate (admin URI). |

**End-to-end dispatch flow**

1. **`POST /api/jobs`** on `jobs-server` resolves a topic from `config/job-topics.yaml` and inserts `pending_dispatch` into MongoDB (saga phase 1).
2. **`jobs-dispatcher`** reacts via change stream (primary) and poll (fallback), publishes a `JobMessage` to Pulsar (phase 2), then updates status to `dispatched` or records retry / `dispatch_failed` (phase 3).

See [job-saga.md](./job-saga.md) for the saga protocol and [dispatch-worker.md](./dispatch-worker.md) for worker wiring.

**Operational entrypoints**

| Binary | Role |
|--------|------|
| **`cmd/jobs-server`** | Gin HTTP API under `/api/jobs` via **`http.JobsHandler`** |
| **`cmd/jobs-cli`** | Cobra CLI with the same job operations via **`service.MetadataService`** (no HTTP hop) |

Both binaries bootstrap MongoDB through **`mongodb.OpenMongoJobs`** (via **`internal/jobs/appruntime`**) and share validation and state-transition rules. **`EnqueueService`** (topic resolution from `config/job-topics.yaml`) backs HTTP enqueue and CLI **`create`**. The CLI defaults to JSON stdout for scripting; **`--output table`** formats **`list`**, **`stats`**, and **`logs`** for interactive use.

See `docs/dev/setup.md` for local run and test workflows.

**Jobs metadata layer**

**`internal/jobs/metadata`** — domain types and persistence ports:

- `JobMetadata` — interface implemented by `JobMetadataModel` for type-safe job records
- `JobStatus` — lifecycle including dispatch phase (`pending_dispatch`, `dispatched`, `dispatch_failed`) and execution phase (`running`, `completed`, `failed`, `cancelled`)
- `JobsReader` / `JobsWriter` — CQRS-style persistence ports (queries vs commands); partial metadata updates use **`UpdateJob`** with **`JobsWriter.Update`** (no separate **`UpdateStatus`**—services assemble **`UpdateJob`** after domain rules)
- Helpers such as `GenerateJobID` and `NewJobLog`
- **`metadata/seed/`** — test-data generator used by the CLI **`seed`** command

**`internal/jobs/mongodb`** — MongoDB implementations:

- `MongoJobsReader` / `MongoJobsWriter` implement the metadata ports
- **`OpenMongoJobs`** connects once and returns reader, writer, and **`*mongo.Client`** while **verifying** required index names (indexes are created by database init / migrations, not by the application)
- **`OpenMongoWatchClient`** — dedicated small-pool client for change streams
- Change-stream (`StreamWatcher`), poll (`PendingFetcher`), and resume-token helpers for the dispatch worker

**`internal/jobs/service`** — application services:

- `MetadataService` — CRUD and status transitions on job metadata (shared by **`jobs-server`** and **`jobs-cli`**)
- `EnqueueService` — topic resolution + `CreateJob` for enqueue on **`jobs-server`** and **`jobs-cli create`**

**`internal/jobs/http`** — Gin handlers (`JobsHandler`) for `/api/jobs/...`

**`internal/jobs/dispatch`** — transport-agnostic dispatch worker and saga handler (see [dispatch-worker.md](./dispatch-worker.md))

**`internal/jobs/pulsar`** — Pulsar client wrapper, topic resolver, producer, and `DispatchPublisher` adapter
