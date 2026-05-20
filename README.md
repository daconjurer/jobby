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

# Full stack in Docker (MongoDB, migrate, HTTP API on :3001)
task docker-up
curl http://localhost:3001/health

# Or: MongoDB + migrate only, then run the server on the host
task mongo-up
task run-jobs-server
# or shorthand
task run
```

## CI

Pull requests and pushes to **`main`** run GitHub Actions (format + lint via Task in **`dockerpaps/golang-for-ci:latest`**). Configure repository secrets **`DOCKER_USERNAME`** and **`DOCKER_PASSWORD`** and a GitHub Actions environment named **`base`** (see [docs/dev/ci.md](docs/dev/ci.md)).

## Documentation

All documentation lives in the [docs](./docs) folder.
The [development setup](./docs/dev/setup.md) is a good starting point and the
[architecture overview](./docs/architecture/intro.md) provides technical details.
