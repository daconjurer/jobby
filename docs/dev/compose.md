# Compose services

The [compose.yml](../../compose.yml) file defines the Docker Compose stack used for development and integration testing. Services load configuration from **`.env`** (created from [**.env.example**](../../.env.example)) via `env_file`, with in-cluster overrides using **`COMPOSE_*`** variables. All services share the explicit Docker network **`jobby`** (`networks.jobby.name`).

Integration test tasks live in [taskfiles/integration/Taskfile.yml](../../taskfiles/integration/Taskfile.yml) (included with `flatten: true`, so names like `task test-integration` stay unchanged).

## Services

| Service | Role |
|---------|------|
| **`mongodb`** | Single-node replica set **`rs0`** (required for change streams). Host port **27018** (`ports: "27018:27017"`). Uses **`config/mongodb-replica.key`** (gitignored, created by **`task mongo-replica-key`**) plus root credentials for bootstrap. **`task docker-up`** and **`task mongo-up`** run that step automatically. |
| **`mongo-init`** | One-shot: runs [docker/mongo-init-replica-set.sh](../../docker/mongo-init-replica-set.sh) to initiate **`rs0`** before **`migrate`** runs. |
| **`migrate`** | One-shot: waits for MongoDB to be healthy, applies schema from [migrations/](../../migrations/) via **`cmd/migrate`**, then exits. Creates the application user, `job_metadata` and `job_logs` collections (with validation), and named indexes verified at startup. See [migrations/README.md](../../migrations/README.md). |
| **`pulsar`** | Standalone Pulsar broker for local enqueue relay (broker **6650**, admin **8080** on the host when published). |
| **`jobs-server`** | HTTP API on host port **3001** (`GET /health`, `/api/jobs/...`). Starts after **`migrate`** succeeds. Does **not** run the dispatch worker. |
| **`jobs-dispatcher`** | Change-stream + poll worker (Mongo watch client + jobs client + Pulsar publish). Starts after **`migrate`**, **`mongodb`**, and **`pulsar`** are ready. |
| **`jobs-executor`** | Consumes job messages from Pulsar and executes registered handlers. Transitions jobs through the execution lifecycle (`dispatched` → `running` → `completed`/`failed`). |

```sh
task docker-up   # mongodb → migrate → jobs-server + jobs-dispatcher + jobs-executor (full stack)
task mongo-up    # mongodb + migrate only (for host go run / integration tests)
```

## Troubleshooting

If **`migrate`** fails, **`jobs-server`** does not start (`depends_on: service_completed_successfully`). Fix migrate logs first (`docker compose logs migrate`). For a clean database reset: `task mongo-reset` then `task mongo-up` (or `docker compose down -v`).

If MongoDB fails with **`security.keyFile is required`**, you are on an older volume from before the replica-set setup — run **`task mongo-reset`** (wipes the volume and regenerates the key), then **`task mongo-up`**. To regenerate only the key file without wiping data, run **`task mongo-replica-key-regenerate`** — that requires a volume reset afterward or MongoDB will not start with the new key.

If migrate fails with **`network … not found`**, an old **migrate** container is still bound to a removed Compose network (common after `docker network prune` or recreating only **mongodb**). Remove it and retry: `docker compose rm -f migrate && task mongo-up`. **`task mongo-up`** recreates **migrate** each run to avoid this.

## Related docs

- [Environment](./environment.md) — `COMPOSE_*` vs host variables
- [Local development](./local-development.md) — when to use `docker-up` vs `mongo-up`
- [Testing](./testing.md) — which services each test category needs
