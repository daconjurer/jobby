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

## Configuration reference

| Variable | Type | Default | Required | Notes |
|----------|------|---------|----------|--------|
| `APP_PORT` | string | — | yes | Must parse as integer **1024–65535** after load ([`ServerConfig.Validate`](./validation.go)). |
| `MONGO_URI` | string | — | yes | Admin-privileged MongoDB URI for `cmd/migrate`; database path segment should be **`jobby`**. |
| `MIGRATIONS_PATH` | string | `./migrations` | no | Directory of golang-migrate JSON migration files. |
| `MONGODB_URI` | string | — | yes | Non-empty connection URI. |
| `MONGODB_DATABASE` | string | — | yes | Database name. |
| `MONGODB_COLLECTION_METADATA` | string | — | yes | Metadata collection. |
| `MONGODB_COLLECTION_LOGS` | string | — | yes | Logs collection. |
| `MONGODB_TIMEOUT` | duration | `10s` | no | Must be **≥ 1s** after parse. |
| `MONGODB_MAX_POOL_SIZE` | uint64 | `100` | no | Must be **≥ `MONGODB_MIN_POOL_SIZE`** and **≤ 1000**. |
| `MONGODB_MIN_POOL_SIZE` | uint64 | `10` | no | See max pool rule. |

## Validation rules

| Check | Rationale |
|-------|-----------|
| `MONGODB_MAX_POOL_SIZE` ≥ `MONGODB_MIN_POOL_SIZE` | Driver pool sizing must be consistent. |
| `MONGODB_TIMEOUT` ≥ 1 second | Very short timeouts tend to fail connects under load. |
| `MONGODB_MAX_POOL_SIZE` ≤ 1000 | Conservative cap aligned with typical driver/ops limits. |
| `APP_PORT` is numeric and in **1024–65535** | Avoids privileged ports and invalid listen addresses. |

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

## Migration from `internal/settings`

The removed `internal/settings` package used ad hoc `GetEnv`/`ParseUint64` helpers. Migrate by:

1. Defining or reusing a struct in this package (`MongoConfig`, `ServerConfig`).
2. Calling `LoadInto`, or `LoadIntoWithOptions` with [`env.Options`](https://github.com/caarlos0/env) when you need a custom environment (e.g. tests).
3. Calling `Validate()` before connecting to MongoDB or binding the HTTP listener.

## Troubleshooting

| Symptom | What to check |
|---------|----------------|
| `required tag` / missing key | Set the variable from the configuration reference table above. |
| `must be >= MONGODB_MIN_POOL_SIZE` | Raise max or lower min pool size. |
| `must be at least 1s` | Increase `MONGODB_TIMEOUT`. |
| `exceeds the allowed maximum (1000)` | Lower `MONGODB_MAX_POOL_SIZE`. |
| `APP_PORT` out of range | Use a port from 1024 through 65535. |
