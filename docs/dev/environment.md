# Environment

Copy [**.env.example**](../../.env.example) to **`.env`** before running **`docker compose`** or integration tests.

## Compose vs host variables

Compose services use `env_file: .env` with in-cluster overrides:

| Variable | Used by | Value (in-cluster) |
|----------|---------|----------------------|
| **`COMPOSE_MONGODB_URI`** | **`migrate`** | `mongodb:27017`, `authSource=admin` |
| **`COMPOSE_APP_MONGODB_URI`** | **`jobs-server`**, **`jobs-dispatcher`**, **`jobs-executor`** | `mongodb:27017`, `authSource=jobby` |
| **`COMPOSE_PULSAR_SERVICE_URL`** | dispatcher and executor | `pulsar:6650` |

Host binaries and integration tests use the non-`COMPOSE_*` variables from **`.env`**:

| Variable | Typical value | Notes |
|----------|---------------|-------|
| **`MONGODB_URI`** | `localhost:27018`, `replicaSet=rs0`, `directConnection=true` | See [Host MongoDB URI](#host-mongodb-uri-and-the-replica-set) below |
| **`PULSAR_SERVICE_URL`** | `pulsar://localhost:6650` | Required for dispatch and executor tests |
| **`JOBS_API_BASE_URL`** | `http://localhost:3001` | E2E tests |

The Go toolchain does not auto-load **`.env`** for **`go run`** — export variables into your shell before running **`cmd/jobs-server`**, **`cmd/jobs-dispatcher`**, or **`cmd/jobs-cli`**. **`task test-integration`** loads **`.env`** via Task `dotenv`.

Full variable reference and validation rules: [internal/config/README.md](../../internal/config/README.md).

## Host MongoDB URI and the replica set

Local Mongo runs as replica set **`rs0`** so change streams work (dispatch worker). **`mongo-init`** registers the member as **`mongodb:27017`** (correct inside the Compose network).

From the **host**, connect through the published port **`localhost:27018`**. The URI must include:

- **`replicaSet=rs0`** — driver treats the deployment as a replica set (change streams, transactions semantics).
- **`directConnection=true`** — pin to the seed host only. Without it, the driver learns **`mongodb:27017`** from the server and tries to reach that hostname, which does not resolve on the host (`lookup mongodb: no such host`).

In-container services use **`COMPOSE_*`** variables with **`mongodb:27017`** and do **not** need **`directConnection=true`**.

Database name and collection names align with what **`migrations/001_initialize_database`** creates for that stack.

## Variables by binary

- **`MONGODB_*`** — required by **`cmd/jobs-server`**, **`cmd/jobs-dispatcher`**, **`cmd/jobs-executor`**, and **`cmd/jobs-cli`** unless you rely on defaults; see [**.env.example**](../../.env.example).
- **`APP_PORT`**, **`JOB_TOPICS_CONFIG_PATH`** — **`cmd/jobs-server`** and **`cmd/jobs-executor`** (enqueue topic resolution and handler registration).
- **`PULSAR_*`**, **`DISPATCH_*`** — **`cmd/jobs-dispatcher`** only.
- **`PULSAR_*`**, **`JOB_TOPICS_CONFIG_PATH`** — **`cmd/jobs-executor`** only.
- **`MONGODB_URI`** — also required by **`internal/jobs/metadata`** integration tests (`task test-integration`).

## Apache Pulsar

**`github.com/apache/pulsar-client-go`** is in the Go module (CGO/native libs; CI uses **`dockerpaps/golang-for-ci`** in [pre-test](../../.github/actions/pre-test/action.yaml)).

**Dispatch and execution:** `POST /api/jobs` on **`jobs-server`** persists **`pending_dispatch`** with embedded **`topic`**. **`jobs-dispatcher`** publishes to Pulsar (change stream + poll). **`jobs-executor`** consumes from Pulsar, executes handlers, and updates job status. Configure:

- **`JOB_TOPICS_CONFIG_PATH`** on the API and executor — defaults to **`config/job-topics.yaml`**
- **`PULSAR_SERVICE_URL`**, **`DISPATCH_*`** on the dispatcher — see [**.env.example**](../../.env.example)
- **`PULSAR_SERVICE_URL`**, **`PULSAR_SUBSCRIPTION_NAME`** on the executor — subscription defaults to **`jobber`** (Shared subscription for load balancing)
- **`MONGODB_*`** on the executor — same as server/dispatcher for job lifecycle updates

Apply migrations through **`005_error_type`** (`task mongo-up` or `task docker-up`) before relying on enqueue + dispatch + execution.

## Job lifecycle

1. `pending_dispatch` — created by `jobs-server` enqueue
2. `dispatched` — confirmed by `jobs-dispatcher` after Pulsar publish
3. `running` — marked by `jobs-executor` when handler starts
4. `completed` or `failed` — terminal states after handler execution
5. `cancelled` — user-requested cancellation (only from `pending_dispatch` or `dispatched`)
