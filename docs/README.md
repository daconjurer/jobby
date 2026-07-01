# Documentation

## Architecture

- [Overview](./architecture/intro.md) — system diagram, job lifecycle, stack
- [Job dispatch saga](./architecture/job-saga.md) — persist → publish → confirm
- [Dispatch worker](./architecture/dispatch-worker.md) — change stream, poll fallback, wiring

## Development

- [Getting started](./dev/getting-started.md) — fastest path to a running stack
- [Local development](./dev/local-development.md) — Docker vs host workflows
- [Compose services](./dev/compose.md) — service inventory and troubleshooting
- [Environment](./dev/environment.md) — env vars, MongoDB replica-set URIs
- [Testing](./dev/testing.md) — unit and integration test overview
- [Integration tests (reference)](./dev/integration-tests.md) — categories, contention, failure modes
- [CLI](./dev/cli.md) — `jobs-cli` examples
- [CI](./dev/ci.md) — GitHub Actions and local CI parity
- [Project structure](./dev/project-structure.md) — `cmd/` and `internal/` layout

## Reference

- [Configuration](../internal/config/README.md) — environment variables and validation
- [Migrations](../migrations/README.md) — MongoDB schema migrations
