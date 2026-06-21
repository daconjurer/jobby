# MongoDB migrations (`golang-migrate`)

Migration files pair **numbered** prefixes with **`{up|down}.json`** suffixes, for example `001_initialize_database.up.json`.

Each `.json` file is a single **JSON array** of MongoDB server commands executed as **`db.runCommand`** against the **`jobby`** database (because the runner’s **`MONGO_URI`** must name **`jobby`** in the URI path segment). Commands use **canonical Extended JSON**: every key **must be quoted**. Do **not** add `"comment"` keys — golang-migrate passes each array element to `runCommand`, and MongoDB has no `comment` command.

Authoritative docs for this format:

- [`github.com/golang-migrate/migrate` — Mongo driver README](https://github.com/golang-migrate/migrate/blob/master/database/mongodb/README.md)

## Running the migrate binary

From the repository root:

```bash
export MONGO_URI='mongodb://jobby_admin:jobby_admin_pass@localhost:27018/jobby?authSource=admin'
export MIGRATIONS_PATH=./migrations   # optional; default is ./migrations

go run ./cmd/migrate up
go run ./cmd/migrate version
```

Other commands:

- **`down`** — roll back **one** applied migration (`Steps(-1)`).
- **`force <version>`** — set migration version manually (recovery / tooling only; see golang-migrate docs).

Tracking collections (created by golang-migrate) include **`schema_migrations`** (version state) and, by default, **`migrate_advisory_lock`** (concurrency lock).

## Docker Compose

The **`migrate`** service in [compose.yml](../compose.yml) runs after MongoDB is healthy, applies pending migrations once, and exits (`restart: "no"`). It reads **`COMPOSE_MONGODB_URI`** from your **`.env`** (see [`.env.example`](../.env.example)) as **`MONGO_URI`** — use an admin-privileged URI with hostname **`mongodb`** and port **27017**.

```bash
cp .env.example .env   # if you have not already
task mongo-reset       # fresh volume + replica key (or: docker compose down -v)
task docker-up         # mongodb → migrate → jobs-server + jobs-dispatcher (+ pulsar)
docker compose logs migrate
curl -s http://localhost:3001/health
```

Build the migration image locally:

```bash
docker build -f Dockerfile.migrate -t jobby-migrate .
docker run --rm -e MONGO_URI='mongodb://jobby_admin:jobby_admin_pass@host.docker.internal:27018/jobby?authSource=admin' jobby-migrate version
```

## Applied migrations

| Version | Files | Purpose |
|---------|--------|---------|
| `001` | `001_initialize_database` | `job_metadata`, `job_logs`, indexes, app user |
| `002` | `002_job_status_dispatch` | Dispatch-phase `JobStatus` values; backfill `pending` → `pending_dispatch` |
| `003` | `003_job_dispatch_embedded` | Embedded dispatch fields on `job_metadata`; poll index for dispatch worker |
| `004` | `004_error_history` | Replace legacy `error` string with `errors[]` history array; migrate existing data |
| `005` | `005_error_type` | Require `type` (`execution` \| `dispatch`) on each `errors[]` item; backfill existing entries |

## Adding a new migration

1. Allocate the **next sequential version**: `00N_short_description.up.json` and `00N_short_description.down.json`.
2. Each file stays a **single array**; each element is **one** `runCommand`-shape document (**`create`**, **`createIndexes`**, **`dropIndexes`**, **`drop`**, **`collMod`**, **`createUser`**, **`dropUser`**, etc.).
3. Run **`migrate up`** with an admin-privileged **`MONGO_URI`**.
