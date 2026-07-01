# Continuous integration (GitHub Actions)

Workflows live under **[`.github/workflows/`](../../.github/workflows)**.

## Pipeline

The **`ci`** workflow runs on **`pull_request`** and **`push`** to **`main`**:

1. **`pre-tests`** — format and lint checks (`task format-check`, `task lint-check` in **`dockerpaps/golang-for-ci:latest`**)
2. **`unit-tests`** — `task test` (runs only if pre-tests succeeds)
3. **Integration jobs** — run in parallel after unit tests succeed:
   - **`integration-mongodb`**, **`integration-cli`**, **`integration-http`**, **`integration-pulsar`**, **`integration-dispatch`**
4. **`e2e-tests`** — full Compose stack + **`task test-e2e`** (parallel with integration jobs after unit tests)

Integration jobs call the reusable **[`integration-tests.yaml`](../../.github/workflows/integration-tests.yaml)** workflow. E2E calls **[`e2e-tests.yaml`](../../.github/workflows/e2e-tests.yaml)**.

| Job | Category | Compose services | Task |
|-----|----------|------------------|------|
| `integration-mongodb` | mongodb | mongodb, mongo-init, migrate | `task test-integration-ci-mongodb` |
| `integration-cli` | cli | mongodb, mongo-init, migrate | `task test-integration-ci-cli` |
| `integration-http` | http | mongodb, mongo-init, migrate | `task test-integration-ci-http` |
| `integration-pulsar` | pulsar | pulsar | `task test-integration-ci-pulsar` |
| `integration-dispatch` | dispatch | mongodb, mongo-init, migrate, pulsar | `task test-integration-ci-dispatch` |
| `e2e-tests` | e2e | full stack | `task test-e2e-ci` |

Each integration job runs on the GitHub runner using the built-in Docker daemon (not Docker-in-Docker). Tests hit published ports on **`localhost`** (e.g. **`27018`** for MongoDB), matching **`.env.example`** defaults.

On failure, compose logs are written to **`ci-compose.log`** and uploaded as an Actions artifact.

**Manual re-run:** open the **`integration-tests`** or **`e2e-tests`** workflow in Actions and use **Run workflow** (**`workflow_dispatch`**).

## Repository configuration

Configure in GitHub (**Settings → Secrets and variables → Actions**):

| Secret | Purpose |
|--------|---------|
| **`DOCKER_USERNAME`** | Docker Hub username (pulls **`dockerpaps/golang-for-ci:latest`**) |
| **`DOCKER_PASSWORD`** | Docker Hub PAT or password |

Create a GitHub Actions **environment** named **`base`**: **Settings → Environments → New environment → `base`**.

## Running CI locally

CI orchestration lives in Task recipes as well as workflow YAML. Reproduce the exact CI flow on your machine:

| Task | Description |
|------|-------------|
| `task test-integration-ci-mongodb` | MongoDB integration tests (compose start, migrate, test, cleanup) |
| `task test-integration-ci-pulsar` | Pulsar integration tests |
| `task test-integration-ci-dispatch` | Dispatch saga tests (mongodb + pulsar) |
| `task test-integration-ci-http` | HTTP handler tests |
| `task test-integration-ci-cli` | jobs-cli tests |
| `task test-e2e-ci` | E2E tests with full stack |

```bash
# Same flow as GitHub Actions integration-mongodb job
task test-integration-ci-mongodb

# Same flow as GitHub Actions e2e-tests job
task test-e2e-ci
```

Each task: starts compose services via **`scripts/ci-start-compose-services.sh`**, runs migrate when needed, executes tests with **`INTEGRATION_TESTS=true`**, collects logs to **`ci-compose.log`**, then **`docker compose down -v`**.

Orchestration logic lives in **[`taskfiles/integration/Taskfile.yml`](../../taskfiles/integration/Taskfile.yml)** — update compose startup, healthchecks, or teardown there rather than duplicating in workflow YAML.

For workflow step details, see **[`.github/workflows/ci.yaml`](../../.github/workflows/ci.yaml)** and the reusable workflow files linked above.

## Go / Docker note

Container actions bind-mount the checkout; **`go list`** can fail with **`error obtaining VCS status: exit status 128`** when git refuses to touch that tree. **`pre-tests`** and **`unit-tests`** set **`GOFLAGS=-buildvcs=false`**, and their **`run.sh`** scripts mark **`$GITHUB_WORKSPACE`** as **`safe.directory`** for git.

## Related docs

- [Testing](./testing.md) — integration test overview and category tasks
- [Integration tests (reference)](./integration-tests.md) — per-package requirements and failure modes
