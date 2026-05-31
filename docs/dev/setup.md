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
(`ports: "27018:27017"`) and uses **`MONGO_INITDB_DATABASE: jobby`** plus root credentials for bootstrap.
The **`migrate`** service waits for MongoDB to be healthy, applies schema from
**[migrations/](../../migrations/)** via **`cmd/migrate`**, then exits. That creates the application user,
`job_metadata` and `job_logs` collections (with validation), and the named indexes that
`OpenMongoJobs` / `NewMongoJobsReaderWriter` verify at startup. See
**[migrations/README.md](../../migrations/README.md)** for manual runs and adding migrations.

The **`jobs-server`** service starts only after **`migrate`** exits successfully, connects as
**`jobby_app`**, and publishes the HTTP API on host port **3001** (`GET /health`, `/api/jobs/...`).

```sh
task docker-up   # mongodb → migrate → jobs-server (full stack)
task mongo-up    # mongodb + migrate only (for host go run / integration tests)
```

**When to use which workflow**

| Goal | Command | Mongo URI |
|------|---------|-----------|
| API in Docker, one command | `task docker-up` | Inline in [compose.yml](../../compose.yml) (`mongodb:27017`) |
| Hot reload / debugger on host | `task mongo-up` then `task run-jobs-server` | **`MONGODB_URI`** in `.env` with **`localhost:27018`** |
| Integration tests | `task mongo-up` then `task test-integration` | Same as host binary (`.env` / shell) |

If **`migrate`** fails, **`jobs-server`** does not start (`depends_on: service_completed_successfully`). Fix migrate logs first (`docker compose logs migrate`). For a clean database reset: `docker compose down -v`.

If migrate fails with **`network … not found`**, an old **migrate** container is still bound to a removed Compose network (common after `docker network prune` or recreating only **mongodb**). Remove it and retry: `docker compose rm -f migrate && task mongo-up`. **`task mongo-up`** recreates **migrate** each run to avoid this.

Copy [**.env.example**](../../.env.example) to **`.env`** before **`docker compose`** so **`COMPOSE_MONGODB_URI`** is available to the **`migrate`** service (admin URI, hostname **`mongodb`**, port **27017**). Compose loads **`.env`** automatically; the Go toolchain does not load it for **`go run`** — export variables into your shell (or use your preferred loader) before running **`cmd/jobs-server`**, **`cmd/jobs-cli`**, or **`task test-integration`**.

- **`COMPOSE_MONGODB_URI`** — used only by the **`migrate`** service in [compose.yml](../../compose.yml).
- **`MONGODB_URI`** — used by apps and integration tests on the host; use **`localhost`** and published port **27018** (see `.env.example`).

Database name and collection names align with what **`migrations/001_initialize_database`** creates for that stack.

# MongoDB and jobs metadata

- **`MONGODB_*`** (and **`APP_PORT`** for the HTTP server) — required by **`cmd/jobs-server`** and
  **`cmd/jobs-cli`** unless you rely on defaults for timeout and pool size; see [**.env.example**](../../.env.example).
- **`MONGODB_URI`** — also **required** by **`internal/jobs/metadata`** integration tests
  (`task test-integration`); use the same value as for local binaries (from `.env` / your shell).

# Tests

- **`task test`** — `go test ./...` (unit tests; no integration tag).
- **`task test-integration`** — runs tests with `-tags=integration` (see [Taskfile.yml](../../Taskfile.yml)); requires MongoDB (e.g. `task mongo-up`) **and** **`MONGODB_URI`** set in the environment (see [**.env.example**](../../.env.example)).

# jobs-cli examples

With MongoDB up and **`MONGODB_URI`** exported (same as for **`jobs-server`** on the host):

```sh
task mongo-up
source .env   # or export MONGODB_* manually

# Health check (Mongo ping)
go run ./cmd/jobs-cli ping

# Create and inspect a job (JSON default)
go run ./cmd/jobs-cli create --name demo --payload '{"k":"v"}' --priority 7
go run ./cmd/jobs-cli get <jobId>
go run ./cmd/jobs-cli list --status pending
go run ./cmd/jobs-cli stats

# Mutations
go run ./cmd/jobs-cli fail <jobId> --error "boom"
go run ./cmd/jobs-cli retry <jobId>
go run ./cmd/jobs-cli cancel <jobId> --reason "ops hold"

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

This project is a monorepo with multiple microservices.

For this version everything is in one Go module, but I expect this will change as the dependency
trees get more complex or need to be narrowed down.
