Continuous integration (GitHub Actions)
=======================================

Workflows live under **[`.github/workflows/`](../../.github/workflows)**.

## CI Jobs

The **`ci`** workflow runs on **`pull_request`** and **`push`** to **`main`**, triggering these jobs in order:

1. **`pre-tests`** — format and lint checks
2. **`unit-tests`** — unit test execution (runs only if **`pre-tests`** succeeds)
3. **Integration jobs** — run in parallel with each other, but only if **`unit-tests`** succeeds:
   - **`integration-mongodb`**
   - **`integration-cli`**
   - **`integration-http`**
   - **`integration-pulsar`**
   - **`integration-dispatch`**
4. **`e2e-tests`** — runs in parallel with integration jobs after **`unit-tests`** succeeds; starts the full Compose stack and runs **`task test-e2e`**

Integration jobs call the reusable **[`integration-tests.yaml`](../../.github/workflows/integration-tests.yaml)** workflow (runner Docker + Compose). The E2E job calls **[`e2e-tests.yaml`](../../.github/workflows/e2e-tests.yaml)**.

Integration jobs are **validated in CI** (2026-06-21): all five categories pass; full pipeline soak ≥3 consecutive green runs; slowest job (`integration-dispatch`) completes in under 4 minutes.

## Running CI Locally (Phase 7)

**Phase 7** moves CI orchestration from GitHub Actions YAML into **Task recipes**, enabling local reproduction of the exact CI flow.

### Task-based CI tasks

| Task | Description |
|------|-------------|
| `task test-integration-ci-mongodb` | MongoDB integration tests with full CI flow (compose start, migrate, test, cleanup) |
| `task test-integration-ci-pulsar` | Pulsar integration tests with full CI flow |
| `task test-integration-ci-dispatch` | Dispatch saga tests with full CI flow |
| `task test-integration-ci-http` | HTTP handler tests with full CI flow |
| `task test-integration-ci-cli` | jobs-cli tests with full CI flow |
| `task test-e2e-ci` | E2E tests with full stack CI flow |

### Example: Run mongodb CI flow locally

```bash
# Exact same flow as GitHub Actions integration-mongodb job
task test-integration-ci-mongodb
```

This will:
1. Start compose services (`mongodb`, `mongo-init`, `migrate`) via **`ci-start-compose-services.sh`**
2. Build and run migrate (conditionally, if "migrate" in service list)
3. Run tests with **`INTEGRATION_TESTS=true TEST_CATEGORY=mongodb`**
4. Collect logs to **`ci-compose.log`** (on success or failure)
5. Clean up with **`docker compose down -v`**

### Logs

On both success and failure, compose logs are written to **`ci-compose.log`** in the workspace root before cleanup. In CI, this file is uploaded as an artifact on failure.

### Benefits

- **Local/CI parity**: Run the exact CI flow on your machine for debugging
- **Fast iteration**: No need to push to GitHub to test CI changes
- **Single source of truth**: All orchestration logic lives in **`taskfiles/integration/Taskfile.yml`**, not scattered across workflow YAML
- **Easy to change**: Update compose startup, healthchecks, or teardown in one place (Taskfile)

### Pre-tests job

1. **`actions/checkout`** on **`ubuntu-latest`**
2. **`docker/login-action`** — logs into Docker Hub so pulls of **`dockerpaps/golang-for-ci:latest`** are less likely to hit anonymous rate limits
3. Composite action **[`.github/actions/pre-test/`](../../.github/actions/pre-test)** runs **`task format-check`** and **`task lint-check`** inside the CI container (Go, Task, and **`golangci-lint`** are provided by that image).

### Unit-tests job

1. **`actions/checkout`** on **`ubuntu-latest`**
2. **`docker/login-action`** — same Docker Hub auth as pre-tests
3. Composite action **[`.github/actions/unit-test/`](../../.github/actions/unit-test)** runs **`task test`** inside the CI container.

The unit-tests job runs **`task test`** (`go test ./...` without **`INTEGRATION_TESTS`**). Integration-tagged tests skip at runtime; only unit tests execute (fast, no infrastructure dependencies). Integration jobs (Phase 5) call category tasks such as **`task test-integration-mongodb`**, which set **`INTEGRATION_TESTS=true`** internally.

### Integration test jobs

Category integration jobs call **[`.github/workflows/integration-tests.yaml`](../../.github/workflows/integration-tests.yaml)** with inputs:

| Input | Purpose |
|-------|---------|
| **`category`** | Test category (`mongodb`, `pulsar`, …) — sets **`TEST_CATEGORY`** and runs **`task test-integration-<category>`** |
| **`compose_services`** | Space-separated Compose services to start for that category |

Each job runs on the **GitHub runner** using the runner's built-in Docker daemon (not Docker-in-Docker). Tests hit published ports on **`localhost`** (e.g. **`27018`** for MongoDB), matching **`.env.example`** defaults and local development. Toolchain setup is handled by **[`.github/actions/integration-setup/`](../../.github/actions/integration-setup)** (pinned Task **`v3.49.1`**, cached).

**Startup sequence** (in the reusable workflow):

1. **`actions/checkout`**
2. Resolve **`compose_services`** (from caller or category defaults when run via **`workflow_dispatch`**)
3. **[`.github/actions/integration-setup/`](../../.github/actions/integration-setup)** — Docker login, Go, pinned Task install, **`.env`** prep
4. **`scripts/ci-start-compose-services.sh`** — **`docker compose pull`**, **`up -d`**, and wait for background services
5. If **`migrate`** is listed: **[`.github/actions/build-migrate/`](../../.github/actions/build-migrate)** (BuildKit layer cache), then run migrate in the foreground
6. **`task test-integration-<category>`** with **`INTEGRATION_TESTS=true`**
7. **`docker compose down -v`** (always, even on failure)

On failure, **`docker compose logs`** is printed to the job log and uploaded as an Actions artifact (**`integration-<category>-compose-logs`**).

**Manual re-run:** open the **`integration-tests`** workflow in Actions and use **Run workflow** (**`workflow_dispatch`**) with a single **`category`** input (Compose services are derived automatically).

#### integration-mongodb

Wired from **`ci.yaml`** as:

```yaml
integration-mongodb:
  uses: ./.github/workflows/integration-tests.yaml
  with:
    category: mongodb
    compose_services: mongodb mongo-init migrate
```

**Service dependencies:**

- **`mongodb`** — MongoDB 8.0 replica set with healthcheck
- **`mongo-init`** — initializes replica set (one-shot with **`restart: on-failure`**)
- **`migrate`** — runs schema migrations (one-shot with **`restart: "no"`**)

The wait script polls until **`mongodb`** is healthy and **`mongo-init`** exits successfully; migrate exit code is checked by Compose directly when run in the foreground.

#### integration-cli

Wired from **`ci.yaml`** as:

```yaml
integration-cli:
  uses: ./.github/workflows/integration-tests.yaml
  with:
    category: cli
    compose_services: mongodb mongo-init migrate
```

Runs **`task test-integration-cli`** (`./cmd/jobs-cli/commands/...`). Requires the same Mongo bootstrap as **`integration-mongodb`**; Pulsar and app services are not started.

#### integration-http

Wired from **`ci.yaml`** as:

```yaml
integration-http:
  uses: ./.github/workflows/integration-tests.yaml
  with:
    category: http
    compose_services: mongodb mongo-init migrate
```

Runs **`task test-integration-http`** (`./internal/jobs/http/...`). Requires the same Mongo bootstrap as **`integration-mongodb`**; Pulsar and app services are not started.

#### integration-pulsar

Wired from **`ci.yaml`** as:

```yaml
integration-pulsar:
  uses: ./.github/workflows/integration-tests.yaml
  with:
    category: pulsar
    compose_services: pulsar
```

Runs **`task test-integration-pulsar`** (`./internal/jobs/pulsar/...`). Starts only the **`pulsar`** broker; Mongo and app services are not required.

#### integration-dispatch

Wired from **`ci.yaml`** as:

```yaml
integration-dispatch:
  uses: ./.github/workflows/integration-tests.yaml
  with:
    category: dispatch
    compose_services: mongodb mongo-init migrate pulsar
```

Runs **`task test-integration-dispatch`** (`./internal/jobs/integrationtest/...`). Requires Mongo bootstrap and Pulsar. App services (`jobs-server`, `jobs-dispatcher`, `jobs-executor`) are not started — tests use in-process dispatch/executor harnesses.

### E2E test job

Wired from **`ci.yaml`** as:

```yaml
e2e-tests:
  needs: unit-tests
  uses: ./.github/workflows/e2e-tests.yaml
```

Runs **`task test-e2e`** (`TestE2EIntegration_*` in `./internal/jobs/executor/...`) against a live stack. Tests are black-box: only **`JOBS_API_BASE_URL`** (default **`http://localhost:3001`**) and infra health from outside the containers.

**Startup sequence** (in **[`e2e-tests.yaml`](../../.github/workflows/e2e-tests.yaml)**):

1. **`actions/checkout`**
2. **[`.github/actions/integration-setup/`](../../.github/actions/integration-setup)** — Docker login, Go, pinned Task install, **`.env`** prep
3. **`scripts/ci-start-compose-services.sh`** — start **`mongodb`**, **`mongo-init`**, **`pulsar`** (background + wait)
4. **[`.github/actions/build-migrate/`](../../.github/actions/build-migrate)** — BuildKit layer cache, then run **`migrate`** in the foreground
5. **`docker compose up -d --build`** — build and start **`jobs-server`**, **`jobs-dispatcher`**, **`jobs-executor`**
6. **`scripts/compose-wait-full.sh`** — poll Mongo/Pulsar/jobs-server healthchecks and **`http://localhost:3001/health`**
7. **`task test-e2e`** with **`INTEGRATION_TESTS=true`** and **`TEST_CATEGORY=e2e`**
8. **`docker compose down -v`** (always, even on failure)

On failure, **`docker compose logs`** for all E2E services is printed to the job log and uploaded as an Actions artifact (**`e2e-compose-logs`**).

**Manual re-run:** open the **`e2e-tests`** workflow in Actions and use **Run workflow** (**`workflow_dispatch`**).

### Go / Docker note

Container actions bind-mount the checkout; **`go list`** (used indirectly by **`gofmt`/`task format-check`**) can fail with **`error obtaining VCS status: exit status 128`** when git refuses to touch that tree (ownership) or stamping is brittle in CI.

Both **`pre-tests`** and **`unit-tests`** jobs set **`GOFLAGS=-buildvcs=false`**, and their respective **`run.sh`** scripts mark **`$GITHUB_WORKSPACE`** as **`safe.directory`** for git — those are toolchain mitigations, not missing module dependencies (the **`go: downloading`** lines are normal proxy/module fetches).

## Repository configuration

Configure the following in GitHub (**Settings → Secrets and variables → Actions**):

| Secret              | Purpose                         |
|---------------------|---------------------------------|
| **`DOCKER_USERNAME`**| Docker Hub username             |
| **`DOCKER_PASSWORD`**| Docker Hub PAT or password      |

The reusable **`pre-tests`** and **`unit-tests`** workflows declare these secrets **`required: true`** and pass them from the top-level **`ci`** workflow.

Create a GitHub Actions **environment** named **`base`** (same name Terrance-style workflows use): **Settings → Environments → New environment → `base`**.

If you use deployment protection rules, ensure they allow GitHub-hosted runners used by **`ubuntu-latest`** to run this workflow without unnecessary manual approval friction for CI.
