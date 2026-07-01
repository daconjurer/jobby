# Local development

Choose a workflow based on whether you run services in Docker or as host binaries.

## Workflows

| Goal | Command | Mongo URI |
|------|---------|-----------|
| Full stack in Docker | `task docker-up` | From `.env` with **`COMPOSE_APP_MONGODB_URI`** override (`mongodb:27017`, `pulsar:6650`) |
| API on host | `task mongo-up` then `task run-jobs-server` | **`MONGODB_URI`** from `.env` (`localhost:27018`, `replicaSet=rs0`, `directConnection=true`) |
| Dispatch on host | `task mongo-up` + Pulsar, then `task run-jobs-dispatcher` | Same Mongo URI; **`PULSAR_SERVICE_URL`** / **`DISPATCH_*`** from `.env` |
| Executor on host | `task mongo-up` + Pulsar, then `task run-jobs-executor` | Same Mongo URI; **`PULSAR_SERVICE_URL`** / **`JOB_TOPICS_CONFIG_PATH`** from `.env` |
| Integration tests | `task mongo-up` then category tasks (see [Testing](./testing.md)) | Same **`MONGODB_URI`** (Task loads `.env`) |
| E2E tests | `task docker-up` then `task test-e2e` | Full stack required (executor completes jobs) |

## Common commands

```sh
# MongoDB + migrations only (for host binaries or integration tests)
task mongo-up

# HTTP API on host (shorthand: task run)
task run-jobs-server

# Dispatch worker (needs Pulsar — start with compose or task docker-up)
task run-jobs-dispatcher

# Executor worker (needs Pulsar)
task run-jobs-executor
```

Host binaries do not auto-load **`.env`** — export variables into your shell before `go run`, or use Task recipes which load `.env` where noted. See [Environment](./environment.md) for URI details.

## Related docs

- [Compose services](./compose.md) — service dependencies and troubleshooting
- [CLI](./cli.md) — operational commands without the HTTP API
- [Testing](./testing.md) — unit and integration test tasks
