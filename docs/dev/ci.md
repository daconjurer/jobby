Continuous integration (GitHub Actions)
=======================================

Workflows live under **[`.github/workflows/`](../../.github/workflows)**.

## Pre-tests job

Runs on **`pull_request`** and **`push`** to **`main`**:

1. **`actions/checkout`** on **`ubuntu-latest`**
2. **`docker/login-action`** — logs into Docker Hub so pulls of **`dockerpaps/golang-for-ci:latest`** are less likely to hit anonymous rate limits
3. Composite action **[`.github/actions/pre-test/`](../../.github/actions/pre-test)** runs **`task format-check`** and **`task lint-check`** inside the CI container (Go, Task, and **`golangci-lint`** are provided by that image).

## Repository configuration

Configure the following in GitHub (**Settings → Secrets and variables → Actions**):

| Secret              | Purpose                         |
|---------------------|---------------------------------|
| **`DOCKER_USERNAME`**| Docker Hub username             |
| **`DOCKER_PASSWORD`**| Docker Hub PAT or password      |

The reusable **`pre-tests`** workflow declares these secrets **`required: true`** and passes them from the top-level **`ci`** workflow.

Create a GitHub Actions **environment** named **`base`** (same name Terrance-style workflows use): **Settings → Environments → New environment → `base`**.

If you use deployment protection rules, ensure they allow GitHub-hosted runners used by **`ubuntu-latest`** to run this workflow without unnecessary manual approval friction for CI.
