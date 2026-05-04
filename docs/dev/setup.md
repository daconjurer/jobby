Dev setup
==========

The whole setup uses [make](https://www.gnu.org/software/make/) so *make* (lol) sure you have
it installed. This is the main [Makefile](../Makefile), so checkout the targets there.

Then get started with:

```sh
go build ./...
```

The [compose.yml](../compose.yml) file defines the **docker compose** stack used for development
and integration testing. It currently exposes **MongoDB 8** on host port **27018** (mapped to 27017 in the container). First-time init runs [scripts/mongo-init.js](../scripts/mongo-init.js), which creates the `jobby` database, `job_metadata` and `job_logs` collections (with validation), and the named indexes that `OpenMongoJobs` / `NewMongoJobsReaderWriter` verify at startup.

```sh
docker compose up -d mongodb
```

# MongoDB and jobs metadata

- **Connection** — application and tests default to  
  `mongodb://jobby_app:jobby_app_pass@localhost:27018/jobby?authSource=jobby`  
  (user `jobby_app` is created in `mongo-init.js`; adjust if you change the init script).
- **`MONGO_URI`** — optional override used by **`cmd/jobs-cli`** and by integration tests in `internal/jobs/metadata` when set; if unset, they use the default string above.

# Tests

- **`make test`** — `go test ./...` (unit tests; no integration tag).
- **`make test-integration`** — runs tests with `-tags=integration` (see [scripts/make/tests.mk](../scripts/make/tests.mk)); requires MongoDB (e.g. `docker compose up -d mongodb`).

# Project structure

This project is a monorepo with multiple microservices.

For this version everything is in one Go module, but I expect this will change as the dependency
trees get more complex or need to be narrowed down.

Design notes for the jobs metadata database feature: `planning/jobs-metadata-database/`.
