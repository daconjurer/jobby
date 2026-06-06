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
| `PULSAR_SERVICE_URL` | string | — | yes* | Broker URL; must start with `pulsar://` or `pulsar+ssl://`. Used by **`cmd/jobs-dispatcher`**. |
| `PULSAR_SUBSCRIPTION_NAME` | string | `jobber` | no | Shared subscription name (reserved for future executor consumers). |
| `DISPATCH_POLL_INTERVAL` | duration | `5s` | no | Poll fallback ticker; must be **≥ 100ms**. |
| `DISPATCH_POLL_BATCH_SIZE` | int | `50` | no | Max jobs per poll batch; must be **≥ 1**. |
| `DISPATCH_MAX_ATTEMPTS` | int | `5` | no | Publish retries before `dispatch_failed`; must be **≥ 1**. |
| `DISPATCH_STREAM_MAX_POOL_SIZE` | uint64 | `2` | no | Watch-client pool size; must be **1–10**. |
| `DISPATCH_STREAM_MONGODB_RESUME_TOKEN_PATH` | string | *(empty)* | no | File path for change-stream resume token; empty disables persistence. |
| `JOB_TOPICS_CONFIG_PATH` | string | `config/job-topics.yaml` | no | Job name → topic YAML manifest; used by **`cmd/jobs-server`**. |

\* `PULSAR_SERVICE_URL` is required only when loading `PulsarConfig` (dispatcher).

## Validation rules

| Check | Rationale |
|-------|-----------|
| `MONGODB_MAX_POOL_SIZE` ≥ `MONGODB_MIN_POOL_SIZE` | Driver pool sizing must be consistent. |
| `MONGODB_TIMEOUT` ≥ 1 second | Very short timeouts tend to fail connects under load. |
| `MONGODB_MAX_POOL_SIZE` ≤ 1000 | Conservative cap aligned with typical driver/ops limits. |
| `APP_PORT` is numeric and in **1024–65535** | Avoids privileged ports and invalid listen addresses. |
| `PULSAR_SERVICE_URL` starts with `pulsar://` or `pulsar+ssl://` | Matches Pulsar client URI expectations. |
| `PULSAR_SUBSCRIPTION_NAME` non-empty | Subscription must be named for Shared consumers. |
| `DISPATCH_POLL_INTERVAL` ≥ 100ms | Avoids tight poll loops. |
| `DISPATCH_POLL_BATCH_SIZE` ≥ 1 | Batch must fetch at least one job. |
| `DISPATCH_MAX_ATTEMPTS` ≥ 1 | At least one publish attempt before failure. |
| `DISPATCH_STREAM_MAX_POOL_SIZE` in **1–10** | Bounds dedicated watch-client pool. |
| `JOB_TOPICS_CONFIG_PATH` non-empty | Enqueue needs a topic manifest path. |

[`MongoConfig.Validate`](./validation.go), [`PulsarConfig.Validate`](./validation.go), [`MongoDispatchWorkerConfig.Validate`](./validation.go), and [`JobTopicsConfig.Validate`](./validation.go) may return multiple errors joined with [`errors.Join`](https://pkg.go.dev/errors#Join).

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
