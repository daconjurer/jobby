Continuous integration (GitHub Actions)
=======================================

Workflows live under **[`.github/workflows/`](../../.github/workflows)**.

## CI Jobs

The **`ci`** workflow runs on **`pull_request`** and **`push`** to **`main`**, triggering these parallel jobs:

- **`pre-tests`** — format and lint checks
- **`unit-tests`** — unit test execution
- **`integration-mongodb`** — MongoDB integration tests with Docker-in-Docker

### Pre-tests job

1. **`actions/checkout`** on **`ubuntu-latest`**
2. **`docker/login-action`** — logs into Docker Hub so pulls of **`dockerpaps/golang-for-ci:latest`** are less likely to hit anonymous rate limits
3. Composite action **[`.github/actions/pre-test/`](../../.github/actions/pre-test)** runs **`task format-check`** and **`task lint-check`** inside the CI container (Go, Task, and **`golangci-lint`** are provided by that image).

### Unit-tests job

1. **`actions/checkout`** on **`ubuntu-latest`**
2. **`docker/login-action`** — same Docker Hub auth as pre-tests
3. Composite action **[`.github/actions/unit-test/`](../../.github/actions/unit-test)** runs **`task test`** inside the CI container.

The unit-tests job runs **`task test`** (`go test ./...` without **`INTEGRATION_TESTS`**). Integration-tagged tests skip at runtime; only unit tests execute (fast, no infrastructure dependencies). Integration jobs (Phase 5) call category tasks such as **`task test-integration-mongodb`**, which set **`INTEGRATION_TESTS=true`** internally.

### Integration-mongodb job

1. **`actions/checkout`** on **`ubuntu-latest`** with submodules
2. **Docker-in-Docker service** — `docker:24-dind` with `--privileged` option provides Docker daemon
3. **`docker/login-action`** — same Docker Hub auth to avoid rate limits
4. **`actions/setup-go`** — installs Go version from `go.mod` with caching
5. **Install Task** — `go install github.com/go-task/task/v3/cmd/task@latest`
6. **Prepare environment** — copies `.env.example` to `.env` and runs `task mongo-replica-key`
7. **Start services** — `docker compose up -d mongodb mongo-init migrate`
8. **Wait for services** — runs `scripts/wait-for-compose-services.sh` to poll service health (timeout: 180s)
9. **Run tests** — `task test-integration-mongodb` with `INTEGRATION_TESTS=true`
10. **Cleanup** — `docker compose down -v` (always runs, even on failure)

**Docker-in-Docker pattern:** Tests run on the **host runner** (not in a container), hitting `localhost:27018` for MongoDB (published port from Compose). The DinD service provides the Docker daemon via `DOCKER_HOST=tcp://docker:2375`, allowing `docker compose` commands to work. This pattern is simpler than running tests inside a container and matches local development workflow.

**Service dependencies:** MongoDB category requires three services:
- `mongodb` — MongoDB 8.0 replica set with healthcheck
- `mongo-init` — initializes replica set (one-shot with `restart: on-failure`)
- `migrate` — runs schema migrations (one-shot with `restart: "no"`)

The wait script polls until `mongodb` is healthy and `migrate` exits successfully (exit code 0).

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
