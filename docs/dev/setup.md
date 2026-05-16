Dev setup
==========

The whole setup uses [make](https://www.gnu.org/software/make/) so *make* (lol) sure you have
it installed. This is the main [Makefile](../../Makefile), so checkout the targets there.

Then get started with:

```sh
make build
```

The [compose.yml](../../compose.yml) file defines the **docker compose** stack used for development
and integration testing. The **`mongodb`** service maps container port **27017** to host port **27018**
(`ports: "27018:27017"`), wires in [scripts/mongo-init.js](../../scripts/mongo-init.js) on first start,
and uses **`MONGO_INITDB_DATABASE: jobby`** plus root credentials for bootstrap. That init script
creates the application user, the `jobby` database, `job_metadata` and `job_logs` collections (with
validation), and the named indexes that `OpenMongoJobs` / `NewMongoJobsReaderWriter` verify at
startup.

```sh
make mongo-up
```

Copy [**.env.example**](../../.env.example) to **`.env`** and adjust for how you run the apps.
The Go toolchain does not load **`.env`** for you—export those variables into your shell (or use your
preferred loader) before running **`cmd/jobs-server`**, **`cmd/jobs-cli`**, or **`make test-integration`**.
Variables there mirror what **`cmd/jobs-server`** and **`cmd/jobs-cli`** expect: **`MONGODB_URI`**
uses **`localhost`** and the **composed host port** when binaries run on your machine; the
commented alternate **`MONGODB_URI`** in `.env.example` matches in-cluster access (**`mongodb`** as
hostname, **27017**) consistent with the service name and internal port in [compose.yml](../../compose.yml).
Database name and collection names align with what **`mongo-init.js`** creates for that stack.

# MongoDB and jobs metadata

- **`MONGODB_*`** (and **`PORT`** for the HTTP server) — required by **`cmd/jobs-server`** and
  **`cmd/jobs-cli`** unless you rely on defaults for timeout and pool size; see [**.env.example**](../../.env.example).
- **`MONGODB_URI`** — also **required** by **`internal/jobs/metadata`** integration tests
  (`make test-integration`); use the same value as for local binaries (from `.env` / your shell).

# Tests

- **`make test`** — `go test ./...` (unit tests; no integration tag).
- **`make test-integration`** — runs tests with `-tags=integration` (see [scripts/make/tests.mk](../../scripts/make/tests.mk)); requires MongoDB (e.g. `make mongo-up`) **and** **`MONGODB_URI`** set in the environment (see [**.env.example**](../../.env.example)).

# Project structure

This project is a monorepo with multiple microservices.

For this version everything is in one Go module, but I expect this will change as the dependency
trees get more complex or need to be narrowed down.

Design notes for the jobs metadata database feature: `planning/jobs-metadata-database/`.
