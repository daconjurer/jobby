# MongoDB migrations (`golang-migrate`)

Migration files pair **numbered** prefixes with **`{up|down}.json`** suffixes, for example `001_initialize_database.up.json`.

Each `.json` file is a single **JSON array** of MongoDB server commands executed as **`db.runCommand`** against the **`jobby`** database (because the runner’s **`MONGO_URI`** must name **`jobby`** in the URI path segment). Commands use **canonical Extended JSON**: every key **must be quoted**.

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

Tracking collections (created by golang-migrate) include **`schema_migrations`** (version state) and, by default, **`migrate_advisory_lock`**.

## Compose + `mongo-init.js` caveat

Docker Compose currently mounts **`scripts/mongo-init.js`**, which creates the same schema on **first container start**. If **`mongo-init` has already run**, **`migrate up`** will fail creating collections that already exist.

To exercise migrations alone on a blank database:

1. Start MongoDB **without** the init script volume (temporary change to `compose.yml`), or  
2. Use a **fresh volume** (`docker compose down -v`) **and** an image/start path that skips `mongo-init`, or  
3. Use a disposable `docker run mongo:…` container with only **`MONGO_INITDB_ROOT_USERNAME` / PASSWORD`** (no `/docker-entrypoint-initdb.d` script).

Phase 2 will remove `mongo-init` from Compose and rely on this binary.

## Adding a new migration

1. Allocate the **next sequential version**: `002_short_description.up.json` and `002_short_description.down.json`.
2. Each file stays a **single array**; each element is **one** `runCommand`-shape document (**`create`**, **`createIndexes`**, **`dropIndexes`**, **`drop`**, **`collMod`**, **`createUser`**, **`dropUser`**, etc.).
3. Run **`migrate up`** with an admin-privileged **`MONGO_URI`**.
