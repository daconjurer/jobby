Dev setup
==========

Local automation uses **[Task](https://taskfile.dev/installation/)** (`go install`-able or distro packages — see upstream docs).
Tasks are declared in **[Taskfile.yml](../../Taskfile.yml)** at repo root (`task --list` to discover names).
Integration test tasks live in **[taskfiles/integration/Taskfile.yml](../../taskfiles/integration/Taskfile.yml)** (included with `flatten: true`, so names like `task test-integration` stay unchanged).

Then get started with:

```sh
# Copy environment template
cp .env.example .env

# Build binaries
task build
```

The [compose.yml](../../compose.yml) file defines the **docker compose** stack used for development
and integration testing. Services load configuration from **`.env`** (created from [**.env.example**](../../.env.example))
via `env_file`, with in-cluster overrides using **`COMPOSE_*`** variables.
Services share the explicit Docker network **`jobby`** (`networks.jobby.name`).
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
| Full stack in Docker | `task docker-up` | From `.env` with **`COMPOSE_APP_MONGODB_URI`** override (`mongodb:27017`, `pulsar:6650`) |
| API on host | `task mongo-up` then `task run-jobs-server` | **`MONGODB_URI`** from `.env` (`localhost:27018`, `replicaSet=rs0`, `directConnection=true`) |
| Dispatch on host | `task mongo-up` + Pulsar, then `task run-jobs-dispatcher` | Same Mongo URI; **`PULSAR_SERVICE_URL`** / **`DISPATCH_*`** from `.env` |
| Executor on host | `task mongo-up` + Pulsar, then `task run-jobs-executor` | Same Mongo URI; **`PULSAR_SERVICE_URL`** / **`JOB_TOPICS_CONFIG_PATH`** from `.env` |
| Integration tests | `task mongo-up` then `task test-integration` | Same **`MONGODB_URI`** (Task loads `.env`) |
| E2E tests | `task docker-up` then `task test-e2e` | Full stack required (executor completes jobs) |
| Single integration category | See [integration test categories](#integration-test-categories) below | Compose subset per category |

If **`migrate`** fails, **`jobs-server`** does not start (`depends_on: service_completed_successfully`). Fix migrate logs first (`docker compose logs migrate`). For a clean database reset: `task mongo-reset` then `task mongo-up` (or `docker compose down -v`).

If MongoDB fails with **`security.keyFile is required`**, you are on an older volume from before the replica-set setup — run **`task mongo-reset`** (wipes the volume and regenerates the key), then **`task mongo-up`**. To regenerate only the key file without wiping data, run **`task mongo-replica-key-regenerate`** — that requires a volume reset afterward or MongoDB will not start with the new key.

If migrate fails with **`network … not found`**, an old **migrate** container is still bound to a removed Compose network (common after `docker network prune` or recreating only **mongodb**). Remove it and retry: `docker compose rm -f migrate && task mongo-up`. **`task mongo-up`** recreates **migrate** each run to avoid this.

**Environment configuration:**

Copy [**.env.example**](../../.env.example) to **`.env`** before running **`docker compose`** or integration tests.
Compose services use `env_file: .env` to load base configuration, with in-cluster overrides for network-specific URIs:

- **`COMPOSE_MONGODB_URI`** — admin URI for **`migrate`** service (`mongodb:27017`, `authSource=admin`)
- **`COMPOSE_APP_MONGODB_URI`** — application URI for **`jobs-server`**, **`jobs-dispatcher`**, **`jobs-executor`** (`mongodb:27017`, `authSource=jobby`)
- **`COMPOSE_PULSAR_SERVICE_URL`** — Pulsar broker for dispatcher and executor (`pulsar:6650`)

Host binaries and integration tests use the non-`COMPOSE_*` variables from **`.env`**:

- **`MONGODB_URI`** — **`localhost:27018`**, **`replicaSet=rs0`**, **`directConnection=true`** (see below)
- **`PULSAR_SERVICE_URL`** — **`pulsar://localhost:6650`**
- **`JOBS_API_BASE_URL`** — **`http://localhost:3001`** (E2E tests)

The Go toolchain does not auto-load **`.env`** for **`go run`** — export variables into your shell (or use your preferred loader) before running **`cmd/jobs-server`**, **`cmd/jobs-dispatcher`**, or **`cmd/jobs-cli`**. **`task test-integration`** loads **`.env`** via Task `dotenv`.

### Host `MONGODB_URI` and the replica set

Local Mongo runs as replica set **`rs0`** so change streams work (dispatch worker). **`mongo-init`** registers the member as **`mongodb:27017`** (correct inside the Compose network).

From the **host**, connect through the published port **`localhost:27018`**. The URI must include:

- **`replicaSet=rs0`** — driver treats the deployment as a replica set (change streams, transactions semantics).
- **`directConnection=true`** — pin to the seed host only. Without it, the driver learns **`mongodb:27017`** from the server and tries to reach that hostname, which does not resolve on the host (`lookup mongodb: no such host`).

In-container services (**`jobs-server`**, **`jobs-dispatcher`**, **`jobs-executor`**, **`migrate`**) use **`COMPOSE_*`** variables with **`mongodb:27017`** (in-cluster hostname) and do **not** need **`directConnection=true`**.

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

Apply migrations through **`005_error_type`** (`task mongo-up` or `task docker-up`) before relying on enqueue + dispatch + execution.

**Job lifecycle:**
1. `pending_dispatch` — created by `jobs-server` enqueue
2. `dispatched` — confirmed by `jobs-dispatcher` after Pulsar publish
3. `running` — marked by `jobs-executor` when handler starts
4. `completed` or `failed` — terminal states after handler execution
5. `cancelled` — user-requested cancellation (only from `pending_dispatch` or `dispatched`)

# Tests

Integration tests no longer use the Go **`integration` build tag**. They are always compiled (so gopls analyzes them) and **skip at runtime** unless **`INTEGRATION_TESTS`** is truthy (`true`, `1`, or `yes`). Shared helpers call **`testutil.SkipUnlessIntegration`**.

- **`task test`** — `go test ./...` without **`INTEGRATION_TESTS`**; unit tests run, integration tests skip.
- **`task test-integration`** — sets **`INTEGRATION_TESTS=true`**, loads **`.env`**, runs the full tree (`./...`). Requires MongoDB (e.g. `task mongo-up`) and host **`MONGODB_URI`** with **`replicaSet=rs0`** and **`directConnection=true`** (see [**.env.example**](../../.env.example)). Saga tests in **`internal/jobs/integrationtest/`** also need Pulsar (`docker compose up -d pulsar`) and **`PULSAR_SERVICE_URL`**; they skip when the broker env is unset. **`task test-e2e`** needs the full stack (`task docker-up`).

## Integration test categories

Integration tests are grouped by infrastructure dependencies (see [planning/tests-in-ci/phase-2.md](../../planning/tests-in-ci/phase-2.md)). Tasks live in **[taskfiles/integration/Taskfile.yml](../../taskfiles/integration/Taskfile.yml)** (included from the root Taskfile). Each category task sets **`INTEGRATION_TESTS=true`**, loads **`.env`**, and runs a fixed package path.

| Category | Task | Compose services |
|----------|------|------------------|
| all | `task test-integration` | Full stack for E2E; Mongo + Pulsar for dispatch |
| mongodb | `task test-integration-mongodb` | `mongodb`, `mongo-init`, `migrate` (`task mongo-up`) |
| pulsar | `task test-integration-pulsar` | `pulsar` |
| dispatch | `task test-integration-dispatch` | `mongodb`, `mongo-init`, `migrate`, `pulsar` — **stop** `jobs-dispatcher` and `jobs-executor` (in-process workers conflict) |
| http | `task test-integration-http` | `mongodb`, `mongo-init`, `migrate` |
| cli | `task test-integration-cli` | `mongodb`, `mongo-init`, `migrate` |
| e2e | `task test-e2e` | Full stack (`task docker-up`) — requires `jobs-dispatcher` and `jobs-executor` |

**Container conflict:** In-process dispatch tests compete with running **`jobs-dispatcher`** / **`jobs-executor`** containers. Stop those services before **`task test-integration-dispatch`**; start them again for **`task test-e2e`**. CI runs one category per job, so this does not apply there. For local runs, prefer per-category tasks over **`task test-integration`** with **`docker-up`** (full stack breaks dispatch; without the app stack, E2E fails).

```sh
# Dispatch (Mongo + Pulsar only; no worker containers)
task mongo-up
docker compose up -d pulsar
docker compose stop jobs-dispatcher jobs-executor
task test-integration-dispatch

# E2E (full stack)
task docker-up
task test-e2e

# Switching after E2E → dispatch
docker compose stop jobs-dispatcher jobs-executor
task test-integration-dispatch

# Switching after dispatch → E2E
docker compose up -d jobs-server jobs-dispatcher jobs-executor
task test-e2e
```

Convenience tasks set **`INTEGRATION_TESTS=true`** and **`TEST_CATEGORY`** internally. CI (Phase 5) can call either a convenience task or the core runner with explicit env:

```sh
# Full suite (needs Mongo + Pulsar; E2E needs full stack)
task mongo-up
docker compose up -d pulsar
task test-integration

# Single category (convenience task)
task mongo-up
task test-integration-mongodb

# Explicit env (CI-style)
INTEGRATION_TESTS=true TEST_CATEGORY=cli task test-integration-category

# Skip integration (exits 0, logs skip)
INTEGRATION_TESTS=false task test-integration-category

# E2E (full stack)
task docker-up
task test-e2e

# Smoke test category → package mapping (no go test, no Compose)
task test-integration-category-resolve

# Manual go test (same semantics as Task)
INTEGRATION_TESTS=true go test -v ./internal/jobs/mongodb/...
```

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
