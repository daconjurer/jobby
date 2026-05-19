Dev setup
==========

Local automation uses **[Task](https://taskfile.dev/installation/)** (`go install`-able or distro packages — see upstream docs).
Tasks are declared in **[Taskfile.yml](../../Taskfile.yml)** at repo root (`task --list` to discover names).

Then get started with:

```sh
task build
```

The [compose.yml](../../compose.yml) file defines the **docker compose** stack used for development
and integration testing. The **`mongodb`** service maps container port **27017** to host port **27018**
(`ports: "27018:27017"`), wires in [scripts/mongo-init.js](../../scripts/mongo-init.js) on first start,
and uses **`MONGO_INITDB_DATABASE: jobby`** plus root credentials for bootstrap. That init script
creates the application user, the `jobby` database, `job_metadata` and `job_logs` collections (with
validation), and the named indexes that `OpenMongoJobs` / `NewMongoJobsReaderWriter` verify at
startup.

You can recreate the same schema without relying on **`mongo-init.js`** (for example before Phase 2
removes it from Compose) using **`cmd/migrate`** — see **[migrations/README.md](../../migrations/README.md)**.

```sh
task mongo-up
```

Copy [**.env.example**](../../.env.example) to **`.env`** and adjust for how you run the apps.
The Go toolchain does not load **`.env`** for you—export those variables into your shell (or use your
preferred loader) before running **`cmd/jobs-server`**, **`cmd/jobs-cli`**, or **`task test-integration`**.
Variables there mirror what **`cmd/jobs-server`** and **`cmd/jobs-cli`** expect: **`MONGODB_URI`**
uses **`localhost`** and the **composed host port** when binaries run on your machine; the
commented alternate **`MONGODB_URI`** in `.env.example` matches in-cluster access (**`mongodb`** as
hostname, **27017**) consistent with the service name and internal port in [compose.yml](../../compose.yml).
Database name and collection names align with what **`mongo-init.js`** creates for that stack.

# MongoDB and jobs metadata

- **`MONGODB_*`** (and **`APP_PORT`** for the HTTP server) — required by **`cmd/jobs-server`** and
  **`cmd/jobs-cli`** unless you rely on defaults for timeout and pool size; see [**.env.example**](../../.env.example).
- **`MONGODB_URI`** — also **required** by **`internal/jobs/metadata`** integration tests
  (`task test-integration`); use the same value as for local binaries (from `.env` / your shell).

# Tests

- **`task test`** — `go test ./...` (unit tests; no integration tag).
- **`task test-integration`** — runs tests with `-tags=integration` (see [Taskfile.yml](../../Taskfile.yml)); requires MongoDB (e.g. `task mongo-up`) **and** **`MONGODB_URI`** set in the environment (see [**.env.example**](../../.env.example)).

# CI

See **[ci.md](./ci.md)** for GitHub Actions prerequisites (Docker Hub secrets, **`base`** environment).

# Project structure

This project is a monorepo with multiple microservices.

For this version everything is in one Go module, but I expect this will change as the dependency
trees get more complex or need to be narrowed down.
