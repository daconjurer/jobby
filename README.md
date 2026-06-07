Jobby
======

I once worked on a system that had to interact heavily with a job queueing application.
It was terrible.

This project is inspired by the lessons from those days.

## Configuration

Application settings are read from the process environment with [`internal/config`](./internal/config/README.md). After parsing, configs are **validated** (pool bounds, timeouts, listen port).

## Quick Start

Install [Task](https://taskfile.dev/installation/) so you can run automation from [`Taskfile.yml`](./Taskfile.yml).

```bash
# Run unit tests
task test

# Full stack in Docker (MongoDB, Pulsar, migrate, HTTP API on :3001, dispatch worker)
task docker-up
curl http://localhost:3001/health

# Or: MongoDB + migrate only, then run binaries on the host
task mongo-up
task run-jobs-server          # HTTP API (shorthand: task run)
task run-jobs-dispatcher      # needs Pulsar (e.g. from compose) — see docs/dev/setup.md
```

## CI

Pull requests and pushes to **`main`** run GitHub Actions (format + lint via Task in **`dockerpaps/golang-for-ci:latest`**). Configure repository secrets **`DOCKER_USERNAME`** and **`DOCKER_PASSWORD`** and a GitHub Actions environment named **`base`** (see [docs/dev/ci.md](docs/dev/ci.md)).

## Documentation

All documentation lives in the [docs](./docs) folder.

- [Development setup](./docs/dev/setup.md) — Docker Compose, env vars, tests
- [Architecture overview](./docs/architecture/intro.md) — services, packages, dispatch flow
- [Job dispatch saga](./docs/architecture/job-saga.md) — enqueue → Pulsar → status confirmation
- [Dispatch worker](./docs/architecture/dispatch-worker.md) — change stream, poll fallback, wiring
