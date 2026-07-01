# CLI

**`cmd/jobs-cli`** provides operational parity with the jobs HTTP API via **`MetadataService`** (no HTTP hop). JSON stdout by default; **`--output table`** for interactive use.

## Prerequisites

MongoDB up and **`MONGODB_URI`** exported (same as for **`jobs-server`** on the host):

```sh
task mongo-up
source .env   # or export MONGODB_* manually
```

## Examples

```sh
# Health check (Mongo ping)
go run ./cmd/jobs-cli ping

# Create and inspect a job (JSON default)
go run ./cmd/jobs-cli create --name account-lifecycle --payload '{"k":"v"}' --priority 7
go run ./cmd/jobs-cli get <jobId>
go run ./cmd/jobs-cli list --status pending
go run ./cmd/jobs-cli stats

# Mutations
go run ./cmd/jobs-cli fail <jobId> --error "boom"
go run ./cmd/jobs-cli retry <jobId>
go run ./cmd/jobs-cli cancel <jobId> --reason "ops hold"

# HTTP fail uses the same errors shape as GET /api/jobs/:id
curl -X POST http://localhost:3001/api/jobs/<jobId>/fail \
  -H "Content-Type: application/json" \
  -d '{"errors":[{"error":"boom"}]}'

# Human-readable tables for read commands
go run ./cmd/jobs-cli --output table list --status pending
go run ./cmd/jobs-cli --output table stats
go run ./cmd/jobs-cli --output table logs <jobId>

# Dev/test data (not HTTP parity)
go run ./cmd/jobs-cli seed --count 20 --seed 42
```

Equivalent **`task`** shortcuts: **`task run-jobs-cli`**, **`task jobs-cli-help`**.

Compare CLI JSON output with **`curl`** against **`jobs-server`** (`task docker-up` or **`task run-jobs-server`**) for the same operation when verifying parity.

## Related docs

- [Local development](./local-development.md) — running the API alongside the CLI
- [Project structure](./project-structure.md) — `cmd/jobs-cli` layout
