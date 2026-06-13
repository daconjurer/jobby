Dev setup
==========

Local automation uses **[Task](https://taskfile.dev/installation/)** (`go install`-able or distro packages — see upstream docs).
Tasks are declared in **[Taskfile.yml](../../Taskfile.yml)** at repo root (`task --list` to discover names).

Then get started with:

```sh
task build
```

The [compose.yml](../../compose.yml) file defines the **docker compose** stack used for development
and integration testing. Services share the explicit Docker network **`jobby`** (`networks.jobby.name`).
The **`mongodb`** service maps container port **27017** to host port **27018**
(`ports: "27018:27017"`), runs a single-node replica set **`rs0`** (required for change streams), and uses
**`config/mongodb-replica.key`** (gitignored, created by **`task mongo-replica-key`**) plus root credentials for bootstrap (replica sets with auth require a key file). **`task docker-up`** and **`task mongo-up`** run that step automatically.
The **`mongo-init`** one-shot service runs **[docker/mongo-init-replica-set.sh](../../docker/mongo-init-replica-set.sh)** to initiate **`rs0`** before **`migrate`** runs.
The **`migrate`** service waits for MongoDB to be healthy, applies schema from
**[migrations/](../../migrations/)** via **`cmd/migrate`**, then exits. That creates the application user,
`job_metadata` and `job_logs` collections (with validation), and the named indexes that
`mongodb.OpenMongoJobs` / `mongodb.NewMongoJobsReader` verify at startup. See
**[migrations/README.md](../../migrations/README.md)** for manual runs and adding migrations.

The **`pulsar`** service runs **standalone** Pulsar for local enqueue relay (broker **6650**, admin **8080** on the host when published).

The **`jobs-server`** service starts only after **`migrate`** exits successfully, connects as
**`jobby_app`**, and publishes the HTTP API on host port **3001** (`GET /health`, `/api/jobs/...`). It does **not** run the dispatch worker.

The **`jobs-dispatcher`** service runs the change-stream + poll worker (Mongo watch client + jobs client + Pulsar publish). It starts after **`migrate`**, **`mongodb`**, and **`pulsar`** are ready. Integration with the API is via **`job_metadata`** (`pending_dispatch` → `dispatched`).

The **`jobs-executor`** service consumes job messages from Pulsar and executes registered handlers. It starts after **`migrate`**, **`mongodb`**, and **`pulsar`** are ready. It transitions jobs through the execution lifecycle (`dispatched` → `running` → `completed`/`failed`).

```sh
task docker-up   # mongodb → migrate → jobs-server + jobs-dispatcher + jobs-executor (full stack)
task mongo-up    # mongodb + migrate only (for host go run / integration tests)
```

**When to use which workflow**

| Goal | Command | Mongo URI |
|------|---------|-----------|
| Full stack in Docker | `task docker-up` | Inline in [compose.yml](../../compose.yml) (`mongodb:27017`, `pulsar:6650`) |
| API on host | `task mongo-up` then `task run-jobs-server` | **`MONGODB_URI`** from `.env.example` (`localhost:27018`, `replicaSet=rs0`, `directConnection=true`) |
| Dispatch on host | `task mongo-up` + Pulsar, then `task run-jobs-dispatcher` | Same Mongo URI; **`PULSAR_SERVICE_URL`** / **`DISPATCH_*`** from `.env` |
| Executor on host | `task mongo-up` + Pulsar, then `task run-jobs-executor` | Same Mongo URI; **`PULSAR_SERVICE_URL`** / **`JOB_TOPICS_CONFIG_PATH`** from `.env` |
| Integration tests | `task mongo-up` then `task test-integration` | Same **`MONGODB_URI`** (Task loads `.env`) |
| E2E tests | `task docker-up` then `task test-integration` | Full stack required (executor completes jobs) |

If **`migrate`** fails, **`jobs-server`** does not start (`depends_on: service_completed_successfully`). Fix migrate logs first (`docker compose logs migrate`). For a clean database reset: `task mongo-reset` then `task mongo-up` (or `docker compose down -v`).

If MongoDB fails with **`security.keyFile is required`**, you are on an older volume from before the replica-set setup — run **`task mongo-reset`** (wipes the volume and regenerates the key), then **`task mongo-up`**. To regenerate only the key file without wiping data, run **`task mongo-replica-key-regenerate`** — that requires a volume reset afterward or MongoDB will not start with the new key.

If migrate fails with **`network … not found`**, an old **migrate** container is still bound to a removed Compose network (common after `docker network prune` or recreating only **mongodb**). Remove it and retry: `docker compose rm -f migrate && task mongo-up`. **`task mongo-up`** recreates **migrate** each run to avoid this.

Copy [**.env.example**](../../.env.example) to **`.env`** before **`docker compose`** so **`COMPOSE_MONGODB_URI`** is available to the **`migrate`** service (admin URI, hostname **`mongodb`**, port **27017**). Compose loads **`.env`** automatically; the Go toolchain does not load it for **`go run`** — export variables into your shell (or use your preferred loader) before running **`cmd/jobs-server`**, **`cmd/jobs-dispatcher`**, or **`cmd/jobs-cli`**. **`task test-integration`** loads **`.env`** via Task `dotenv`.

- **`COMPOSE_MONGODB_URI`** — used only by the **`migrate`** service in [compose.yml](../../compose.yml).
- **`MONGODB_URI`** — used by host binaries and integration tests. Copy the full value from **`.env.example`**: **`localhost:27018`**, **`replicaSet=rs0`**, and **`directConnection=true`** (see below).

### Host `MONGODB_URI` and the replica set

Local Mongo runs as replica set **`rs0`** so change streams work (dispatch worker). **`mongo-init`** registers the member as **`mongodb:27017`** (correct inside the Compose network).

From the **host**, connect through the published port **`localhost:27018`**. The URI must include:

- **`replicaSet=rs0`** — driver treats the deployment as a replica set (change streams, transactions semantics).
- **`directConnection=true`** — pin to the seed host only. Without it, the driver learns **`mongodb:27017`** from the server and tries to reach that hostname, which does not resolve on the host (`lookup mongodb: no such host`).

In-container services (**`jobs-server`**, **`jobs-dispatcher`**, **`migrate`**) use **`mongodb:27017`** in [compose.yml](../../compose.yml) and do **not** need **`directConnection=true`**.

Database name and collection names align with what **`migrations/001_initialize_database`** creates for that stack.

# MongoDB and jobs metadata

- **`MONGODB_*`** — required by **`cmd/jobs-server`**, **`cmd/jobs-dispatcher`**, **`cmd/jobs-executor`**, and **`cmd/jobs-cli`** unless you rely on defaults; see [**.env.example**](../../.env.example).
- **`APP_PORT`**, **`JOB_TOPICS_CONFIG_PATH`** — **`cmd/jobs-server`** and **`cmd/jobs-executor`** (enqueue topic resolution and handler registration).
- **`PULSAR_*`**, **`DISPATCH_*`** — **`cmd/jobs-dispatcher`** only.
- **`PULSAR_*`**, **`JOB_TOPICS_CONFIG_PATH`** — **`cmd/jobs-executor`** only.
- **`MONGODB_URI`** — also **required** by **`internal/jobs/metadata`** integration tests
  (`task test-integration`); use the same value as for local binaries (from `.env` / your shell).

# Apache Pulsar (job executor)

**`github.com/apache/pulsar-client-go`** is in the Go module (CGO/native libs; CI uses **`dockerpaps/golang-for-ci`** in [pre-test](../../.github/actions/pre-test/action.yaml)).

**Phase 2–4 (dispatch + execution):** `POST /api/jobs` on **`jobs-server`** persists **`pending_dispatch`** with embedded **`topic`**. **`jobs-dispatcher`** publishes to Pulsar (change stream + poll). **`jobs-executor`** consumes from Pulsar, executes handlers, and updates job status. Configure:

- **`JOB_TOPICS_CONFIG_PATH`** on the API and executor — defaults to **`config/job-topics.yaml`**
- **`PULSAR_SERVICE_URL`**, **`DISPATCH_*`** on the dispatcher — see [**.env.example**](../../.env.example)
- **`PULSAR_SERVICE_URL`**, **`PULSAR_SUBSCRIPTION_NAME`** on the executor — subscription defaults to **`jobber`** (Shared subscription for load balancing)
- **`MONGODB_*`** on the executor — same as server/dispatcher for job lifecycle updates

Apply migrations through **`004_error_history`** (`task mongo-up` or `task docker-up`) before relying on enqueue + dispatch + execution.

**Job lifecycle:**
1. `pending_dispatch` — created by `jobs-server` enqueue
2. `dispatched` — confirmed by `jobs-dispatcher` after Pulsar publish
3. `running` — marked by `jobs-executor` when handler starts
4. `completed` or `failed` — terminal states after handler execution
5. `cancelled` — user-requested cancellation (only from `pending_dispatch` or `dispatched`)

# Tests

- **`task test`** — `go test ./...` (unit tests; no integration tag).
- **`task test-integration`** — runs tests with `-tags=integration` (see [Taskfile.yml](../../Taskfile.yml)); loads **`.env`** automatically. Requires MongoDB (e.g. `task mongo-up`) and a host **`MONGODB_URI`** with **`replicaSet=rs0`** and **`directConnection=true`** (see [**.env.example**](../../.env.example)). Saga tests in **`internal/jobs/integrationtest/`** also need Pulsar (`docker compose up -d pulsar`) and **`PULSAR_SERVICE_URL`**; they skip when the broker env is unset.

# jobs-cli examples

With MongoDB up and **`MONGODB_URI`** exported (same as for **`jobs-server`** on the host):

```sh
task mongo-up
source .env   # or export MONGODB_* manually

# Health check (Mongo ping)
go run ./cmd/jobs-cli ping

# Create and inspect a job (JSON default)
go run ./cmd/jobs-cli create --name account-lifecycle --payload '{"k":"v"}' --priority 7
go run ./cmd/jobs-cli get <jobId>
go run ./cmd/jobs-cli list --status pending
go run ./cmd/jobs-cli stats

# Mutations
go run ./cmd/jobs-cli fail <jobId> --error "boom"
go run ./cmd/jobs-cli retry <jobId>
go run ./cmd/jobs-cli cancel <jobId> --reason "ops hold"

# HTTP fail uses the same errors shape as GET /api/jobs/:id
curl -X POST http://localhost:3001/api/jobs/<jobId>/fail \
  -H "Content-Type: application/json" \
  -d '{"errors":[{"error":"boom"}]}'

# Human-readable tables for read commands
go run ./cmd/jobs-cli --output table list --status pending
go run ./cmd/jobs-cli --output table stats
go run ./cmd/jobs-cli --output table logs <jobId>

# Dev/test data (not HTTP parity)
go run ./cmd/jobs-cli seed --count 20 --seed 42
```

Equivalent **`task`** shortcuts: **`task run-jobs-cli`**, **`task jobs-cli-help`**.

Compare CLI JSON output with **`curl`** against **`jobs-server`** (`task docker-up` or **`task run-jobs-server`**) for the same operation when verifying parity.

# CI

See **[ci.md](./ci.md)** for GitHub Actions prerequisites (Docker Hub secrets, **`base`** environment).

# Project structure

This project is a monorepo with multiple microservices. See **[project-structure.md](./project-structure.md)** for the `cmd/` and `internal/` layout and **[architecture/intro.md](../architecture/intro.md)** for the dispatch saga and worker design.
