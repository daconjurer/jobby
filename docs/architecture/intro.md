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
| **`jobs-executor`** | Execution worker — Pulsar consumer → handler execution → status updates (`dispatched` → `running` → `completed`/`failed`). |
| **`jobs-cli`** | Cobra CLI with operational parity to the jobs HTTP API: same `MetadataService` for get, list, stats, fail, cancel, retry, logs, and seed. JSON stdout by default; `--output table` for interactive use. |
| **`migrate`** | Applies `migrations/` via golang-migrate (admin URI). |

**End-to-end flow (enqueue → dispatch → execute)**

```
┌─────────────┐   POST /api/jobs    ┌──────────────┐   change stream   ┌──────────────┐
│             │ ──────────────────> │              │ ────────────────> │              │
│   Client    │                     │ jobs-server  │    or poll        │   jobs-      │
│             │ <────────────────── │              │ <──────────────── │ dispatcher   │
└─────────────┘   201 Created       └──────────────┘                   └──────────────┘
                  {jobID, status:                                             │
                   pending_dispatch}                                          │ publish
                                                                              │ JobMessage
                        MongoDB: job_metadata                                 v
                        ┌────────────────────┐                          ┌─────────┐
                        │ pending_dispatch   │                          │ Pulsar  │
                        │ ──> dispatched     │                          └─────────┘
                        │ ──> running        │                                │
                        │ ──> completed      │                                │ consume
                        └────────────────────┘                                v
                                                                        ┌──────────────┐
                        ┌────────────────────┐                          │   jobs-      │
                        │ GET /api/jobs/:id  │ <──────────────────────  │  executor    │
                        │ status=completed   │    updates metadata      └──────────────┘
                        └────────────────────┘
```

1. **`POST /api/jobs`** on `jobs-server` resolves a topic from `config/job-topics.yaml` and inserts `pending_dispatch` into MongoDB (saga **persist** step).
2. **`jobs-dispatcher`** reacts via change stream (primary) and poll (fallback), publishes a `JobMessage` to Pulsar (**publish**), then updates status to `dispatched` or records retry / `dispatch_failed` (**confirm**).
3. **`jobs-executor`** consumes the message from Pulsar, transitions job to `running`, executes the registered handler, and marks job as `completed` or `failed`.

**Client polling:** After enqueue, poll `GET /api/jobs/:id` until `status` is `completed`, `failed`, or `cancelled`. Typical flow: `pending_dispatch` → `dispatched` → `running` → `completed`.

See [job-saga.md](./job-saga.md) for the saga protocol and [dispatch-worker.md](./dispatch-worker.md) for worker wiring.

See [Getting started](../dev/getting-started.md) for local run workflows and [Testing](../dev/testing.md) for test tasks.

For the `cmd/` and `internal/` package map, see [Project structure](../dev/project-structure.md).
