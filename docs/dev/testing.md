# Testing

## Unit tests

```sh
task test   # go test ./... without INTEGRATION_TESTS
```

Integration tests are always compiled (so gopls analyzes them) and **skip at runtime** unless **`INTEGRATION_TESTS`** is truthy (`true`, `1`, or `yes`). Shared helpers call **`testutil.SkipUnlessIntegration`**.

## Integration tests

```sh
task mongo-up
docker compose up -d pulsar   # when dispatch/saga tests need Pulsar
task test-integration-mongodb   # example: single category
```

Category tasks set **`INTEGRATION_TESTS=true`**, load **`.env`**, and run a fixed package path. Tasks live in [taskfiles/integration/Taskfile.yml](../../taskfiles/integration/Taskfile.yml).

| Category | Task | Compose services |
|----------|------|------------------|
| all | `task test-integration` | Full stack for E2E; Mongo + Pulsar for dispatch |
| mongodb | `task test-integration-mongodb` | `mongodb`, `mongo-init`, `migrate` (`task mongo-up`) |
| pulsar | `task test-integration-pulsar` | `pulsar` |
| dispatch | `task test-integration-dispatch` | Mongo + Pulsar — **stop** `jobs-dispatcher` and `jobs-executor` |
| http | `task test-integration-http` | `mongodb`, `mongo-init`, `migrate` |
| cli | `task test-integration-cli` | `mongodb`, `mongo-init`, `migrate` |
| e2e | `task test-e2e` | Full stack (`task docker-up`) |

**Container conflict:** In-process dispatch tests compete with running **`jobs-dispatcher`** / **`jobs-executor`** containers. Stop those services before **`task test-integration-dispatch`**; start them again for **`task test-e2e`**.

```sh
# Dispatch (Mongo + Pulsar only; no worker containers)
task mongo-up
docker compose up -d pulsar
docker compose stop jobs-dispatcher jobs-executor
task test-integration-dispatch

# E2E (full stack)
task docker-up
task test-e2e
```

For per-package requirements, contention details, failure modes, and CI parity tasks, see [Integration tests (reference)](./integration-tests.md).

## CI

GitHub Actions runs the same categories in parallel after unit tests. To reproduce CI locally: [ci.md](./ci.md).
