# internal/config

Structured environment configuration for Jobby using [caarlos0/env/v11](https://github.com/caarlos0/env).

## Quick start

```go
var mc MongoConfig
if err := LoadInto(&mc); err != nil {
    return fmt.Errorf("parsing mongo config: %w", err)
}
if err := mc.Validate(); err != nil {
    return fmt.Errorf("validating mongo config: %w", err)
}
```

Binaries use [`LoadIntoWithOptions`](./loader.go) with [`LoadOptionsFromEnv`](./loader.go) so optional [`JOBBY_ENV_PREFIX`](./loader.go) is honored (see below).

## Configuration reference

| Variable | Type | Default | Required | Notes |
|----------|------|---------|----------|--------|
| `PORT` | string | — | yes | Must parse as integer **1024–65535** after load ([`ServerConfig.Validate`](./validation.go)). |
| `MONGODB_URI` | string | — | yes | Non-empty connection URI. |
| `MONGODB_DATABASE` | string | — | yes | Database name. |
| `MONGODB_COLLECTION_METADATA` | string | — | yes | Metadata collection. |
| `MONGODB_COLLECTION_LOGS` | string | — | yes | Logs collection. |
| `MONGODB_TIMEOUT` | duration | `10s` | no | Must be **≥ 1s** after parse. |
| `MONGODB_MAX_POOL_SIZE` | uint64 | `100` | no | Must be **≥ `MONGODB_MIN_POOL_SIZE`** and **≤ 1000**. |
| `MONGODB_MIN_POOL_SIZE` | uint64 | `10` | no | See max pool rule. |

## Prefixed variables (`JOBBY_*`)

Set `JOBBY_ENV_PREFIX` to a prefix such as `JOBBY_`. For each canonical variable above, if it is **missing or empty** in the environment, the loader will use `PREFIX` + canonical name (e.g. `JOBBY_PORT` when `PORT` is unset). If both are set, the **canonical** name wins.

This avoids collisions in shared shells or orchestration without breaking existing unprefixed deployments.

## Validation rules

| Check | Rationale |
|-------|-----------|
| `MONGODB_MAX_POOL_SIZE` ≥ `MONGODB_MIN_POOL_SIZE` | Driver pool sizing must be consistent. |
| `MONGODB_TIMEOUT` ≥ 1 second | Very short timeouts tend to fail connects under load. |
| `MONGODB_MAX_POOL_SIZE` ≤ 1000 | Conservative cap aligned with typical driver/ops limits. |
| `PORT` is numeric and in **1024–65535** | Avoids privileged ports and invalid listen addresses. |

[`MongoConfig.Validate`](./validation.go) may return multiple errors joined with [`errors.Join`](https://pkg.go.dev/errors#Join).

## Error handling

- **Parse errors** (missing `required` vars, bad duration, bad integer): returned by `LoadInto` / `LoadIntoWithOptions` from the env library; wrap with context (`parsing mongo config`, etc.).
- **Validation errors**: returned by `Validate()`; wrap with `validating …` so operators can tell parse vs semantic failure.

Messages include the offending values and a short hint (e.g. acceptable ranges).

## Examples

**Minimal Mongo + defaults**

```bash
export MONGODB_URI='mongodb://localhost:27017'
export MONGODB_DATABASE=jobby
export MONGODB_COLLECTION_METADATA=job_metadata
export MONGODB_COLLECTION_LOGS=job_logs
```

**Prefixed-only (with feature flag)**

```bash
export JOBBY_ENV_PREFIX='JOBBY_'
export JOBBY_MONGODB_URI='mongodb://localhost:27017'
export JOBBY_MONGODB_DATABASE=jobby
export JOBBY_MONGODB_COLLECTION_METADATA=job_metadata
export JOBBY_MONGODB_COLLECTION_LOGS=job_logs
export JOBBY_PORT=8080
```

## Migration from `internal/settings`

The removed `internal/settings` package used ad hoc `GetEnv`/`ParseUint64` helpers. Migrate by:

1. Defining or reusing a struct in this package (`MongoConfig`, `ServerConfig`).
2. Calling `LoadInto` or `LoadIntoWithOptions` + `LoadOptionsFromEnv`.
3. Calling `Validate()` before connecting to MongoDB or binding the HTTP listener.

## Troubleshooting

| Symptom | What to check |
|---------|----------------|
| `required tag` / missing key | Set the canonical variable or the prefixed fallback with `JOBBY_ENV_PREFIX`. |
| `must be >= MONGODB_MIN_POOL_SIZE` | Raise max or lower min pool size. |
| `must be at least 1s` | Increase `MONGODB_TIMEOUT`. |
| `exceeds the allowed maximum (1000)` | Lower `MONGODB_MAX_POOL_SIZE`. |
| `PORT` out of range | Use a port from 1024 through 65535. |
