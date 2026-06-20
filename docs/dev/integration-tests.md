Integration tests: services, tasks, and stack tiers
===================================================

This document summarizes which Docker Compose services and Task commands are required for integration testing in this repo. It complements **[setup.md](./setup.md)** (local dev) and the more detailed internal runbook at **`docs/internal/integration-test-isolation.md`** (local-only; that path is gitignored).

---

## Compose services

All services are defined in **[compose.yml](../../compose.yml)**.

| Service | Long-running? | Role |
|---------|---------------|------|
| **`mongodb`** | Yes | Single-node replica set **`rs0`** (change streams); host port **27018** |
| **`mongo-init`** | One-shot | Initiates **`rs0`** before migrate runs |
| **`migrate`** | One-shot | Applies schema migrations; creates app user and collections |
| **`pulsar`** | Yes | Standalone broker; host ports **6650** (client), **8080** (admin) |
| **`jobs-server`** | Yes | HTTP API on host **3001** |
| **`jobs-dispatcher`** | Yes | Change stream + poll → Pulsar publish |
| **`jobs-executor`** | Yes | Pulsar consumer → handler execution → Mongo updates |

### Stack tiers

| Tier | What runs | Typical command |
|------|-----------|-----------------|
| **Mongo only** | `mongodb`, `mongo-init`, `migrate` | `task mongo-up` |
| **Mongo + Pulsar** | Above + `pulsar` | `task mongo-up` then `docker compose up -d pulsar` |
| **Full stack** | Above + `jobs-server`, `jobs-dispatcher`, `jobs-executor` | `task docker-up` |

**Full stack** means the three **job app services** on top of Mongo bootstrap and Pulsar — the production-like path:

```
HTTP client → jobs-server → Mongo
                ↓
         jobs-dispatcher → Pulsar
                ↓
         jobs-executor → handlers → Mongo
```

Fresh database (wipes Mongo volume):

```sh
docker compose down -v
task mongo-up          # or task docker-up for full stack
```

---

## Task commands

Defined in **[Taskfile.yml](../../Taskfile.yml)**. Integration tasks load **`.env`** via Task `dotenv` (copy from **[.env.example](../../.env.example)** first).

| Task | What it runs | Isolation step |
|------|--------------|----------------|
| **`task test`** | Unit tests only (`go test ./...`) | None — no Docker required |
| **`task test-integration`** | All `-tags=integration` tests under `./...` | None built-in |
| **`task test-integration-mongodb`** | `./internal/jobs/mongodb/...` | None built-in |
| **`task test-integration-cli`** | `./cmd/jobs-cli/commands/...` | None built-in |
| **`task test-integration-http`** | `./internal/jobs/http/...` | None built-in |
| **`task test-integration-executor`** | `./internal/jobs/integrationtest/... -run TestIntegration_Executor` | Stops `jobs-dispatcher`, `jobs-executor`, `jobs-server` before run |

Shared flags for integration tasks (`INTEGRATION_TEST_FLAGS`):

```text
-tags=integration -p 1 -parallel 1 -count=1
```

Serial execution avoids races when tests share Mongo/Pulsar.

### Required environment variables

| Variable | Required by | Notes |
|----------|-------------|-------|
| **`MONGODB_URI`** | Mongo integration tests | Host: `localhost:27018`, `replicaSet=rs0`, `directConnection=true` |
| **`MONGODB_DATABASE`** | Optional | Default `jobby` |
| **`PULSAR_SERVICE_URL`** | Dispatch/executor saga, Pulsar tests | `pulsar://localhost:6650` |
| **`JOBS_API_BASE_URL`** | E2E only | Default `http://localhost:3001` |

Running `go test -tags=integration ...` without loading `.env` fails with **`MONGODB_URI is not set`**. Use a Task command or export vars explicitly.

---

## What each task needs (services)

### `task test-integration-executor`

| Service | Required? |
|---------|-----------|
| Mongo (`task mongo-up`) | **Yes** |
| Pulsar | **Yes** |
| `jobs-server` | **No** — must be **stopped** |
| `jobs-dispatcher` | **No** — must be **stopped** |
| `jobs-executor` | **No** — must be **stopped** |

The test harness runs an **in-process** dispatch runtime and Pulsar consumer; compose app workers would steal messages and Mongo change-stream capacity.

```sh
task mongo-up && docker compose up -d pulsar
task test-integration-executor
```

Runs **5 tests** (~5.5s on a fresh stack):

- `TestIntegration_ExecutorSaga_DispatchedToCompleted` (regression)
- `TestIntegration_Executor_ExternalFailDuringRun`
- `TestIntegration_Executor_CompleteBeforeExternalFail`
- `TestIntegration_Executor_TerminalRedeliverySkipsHandler`
- `TestIntegration_Executor_CompleteVsFailRace`

### `task test-integration` (full suite)

Runs **all** integration-tagged tests across the repo. Requirements **vary by package** — neither Mongo-only nor a single stack tier covers everything.

| Package / tests | Mongo | Pulsar | Compose app services |
|-----------------|-------|--------|----------------------|
| `internal/jobs/mongodb` | Yes | No | No |
| `internal/jobs/http` | Yes | No | No |
| `internal/jobs/pulsar` | No | Yes | No |
| `internal/jobs/integrationtest` (11 tests) | Yes | Yes | **Must be stopped** |
| `internal/jobs/executor` (`TestE2EIntegration_*`) | Yes | Yes | **Must be running** |
| `cmd/jobs-cli/...` | Yes | No | No |

**Conflict:** `integrationtest` needs app services **off**; E2E needs them **on**. A single full-suite run can therefore pass some packages and fail others even on a healthy machine.

**Practical minimum** (skip E2E, cover writer + dispatch + executor harness):

```sh
task mongo-up && docker compose up -d pulsar
docker compose stop jobs-dispatcher jobs-executor jobs-server
task test-integration
```

**Full E2E green** (including `TestE2EIntegration_*`):

```sh
task docker-up
go test -tags=integration -p 1 -count=1 -v \
  ./internal/jobs/executor/... -run TestE2EIntegration
```

Do **not** stop app services for E2E.

---

## The contention problem

When **`jobs-dispatcher`** and **`jobs-executor`** containers run **while** `internal/jobs/integrationtest` runs:

- Compose dispatcher watches the same Mongo change stream and publishes to Pulsar.
- Compose executor consumes topics before the test harness consumer.
- Multiple Mongo clients exhaust the dispatch stream pool (`client is disconnected`, 30s timeouts).

**Fix:** stop compose app services before in-process integration tests:

```sh
docker compose stop jobs-dispatcher jobs-executor jobs-server
```

`task test-integration-executor` does this automatically (`|| true` if containers are not running).

Restart app services when you need E2E again:

```sh
docker compose start jobs-server jobs-dispatcher jobs-executor
```

---

## Recommended workflows

### Mongo writer / CAS / metadata service

```sh
docker compose down -v && task mongo-up
go test -tags=integration -p 1 -count=1 -v \
  ./internal/jobs/mongodb/... -run TestIntegration_MongoTerminalWriter
```

Pulsar and app services not required.

### Executor saga + terminal race (safe-job-failure Phase 3)

```sh
docker compose down -v && task mongo-up && docker compose up -d pulsar
task test-integration-executor
```

### Dispatch / change stream / full integrationtest package

```sh
task mongo-up && docker compose up -d pulsar
docker compose stop jobs-dispatcher jobs-executor jobs-server
go test -tags=integration -p 1 -parallel 1 -count=1 -v \
  ./internal/jobs/integrationtest/...
```

### HTTP handler integration

Mongo only (`task mongo-up`). Tests exercise the jobs HTTP handler against live Mongo (enqueue, get, list, fail/cancel/retry, logs).

```sh
task mongo-up
task test-integration-http
```

CI runs this as the **`integration-http`** job (see **[ci.md](./ci.md)**).

### jobs-cli integration

Mongo only (`task mongo-up`). Tests exercise CLI commands against live Mongo (enqueue, get, list, seed, logs, ping, fail/cancel/retry flows).

```sh
task mongo-up
task test-integration-cli
```

CI runs this as the **`integration-cli`** job (see **[ci.md](./ci.md)**).

### Full pipeline E2E (HTTP → dispatch → executor)

```sh
task docker-up
go test -tags=integration -p 1 -count=1 -v \
  ./internal/jobs/executor/... -run TestE2EIntegration
```

---

## Failure modes (quick reference)

| Symptom | Likely cause | Fix |
|---------|--------------|-----|
| `MONGODB_URI is not set` | Ran `go test` without `.env` | Use `task test-integration*` or export vars |
| Connection refused on `:27018` | Mongo not up | `task mongo-up` |
| 30s timeout (dispatch/Pulsar tests) | Pulsar not running | `docker compose up -d pulsar` |
| `client is disconnected` in `integrationtest` | Compose app services running | Stop them or use `task test-integration-executor` |
| E2E: `connection refused` on `:3001` | `jobs-server` not running | `task docker-up` or `docker compose start jobs-server` |
| Unrelated `job not found` mid-suite | Stale Mongo from prior run | `docker compose down -v && task mongo-up` |

---

## Summary table

| Goal | Mongo | Pulsar | App workers | Task / command |
|------|-------|--------|-------------|----------------|
| Unit tests | No | No | No | `task test` |
| Mongo CAS / writer | Yes | No | No | `go test ./internal/jobs/mongodb/...` |
| Executor saga + terminal race | Yes | Yes | **Stopped** | `task test-integration-executor` |
| All `integrationtest` (11 tests) | Yes | Yes | **Stopped** | stop services + `go test ./internal/jobs/integrationtest/...` |
| HTTP handler integration | Yes | No | No | `task test-integration-http` |
| jobs-cli integration | Yes | No | No | `task test-integration-cli` |
| E2E HTTP pipeline | Yes | Yes | **Running** | `task docker-up` + `TestE2EIntegration` |
| Full integration suite | Mixed | Mixed | **Conflicting** | `task test-integration` — interpret failures by package |

**Key takeaway:** Match the stack tier to the package under test. Failures in one tier do not imply regressions in another. For executor work, prefer **`task test-integration-executor`** over the full suite.

---

## `test-integration-executor` implementation notes

The Task recipe encodes validated isolation in two steps:

1. **`docker compose stop jobs-dispatcher jobs-executor jobs-server || true`** — removes competing consumers and dispatch workers.
2. **`go test ... -run TestIntegration_Executor`** — fast, focused executor coverage with shared `INTEGRATION_TEST_FLAGS`.

Prerequisites (Mongo + Pulsar) are left to the caller so infra startup stays explicit. Optional future extensions: a `test-integration-executor-fresh` task that runs `docker compose down -v && task mongo-up` first, or a preflight check that `:27018` and `:6650` are reachable.

---

*Last updated: 2026-06-20*
