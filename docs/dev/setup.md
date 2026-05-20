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

```sh
task mongo-up   # starts mongodb and runs migrate once
```

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

# CI

See **[ci.md](./ci.md)** for GitHub Actions prerequisites (Docker Hub secrets, **`base`** environment).

# Project structure

This project is a monorepo with multiple microservices.

For this version everything is in one Go module, but I expect this will change as the dependency
trees get more complex or need to be narrowed down.
