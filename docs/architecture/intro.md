Architecture overview
======================

This project is a microservices-based system for distributed workloads.

**Stack in this repository**

- **Go** (module `github.com/daconjurer/jobby`, Go 1.25)
- **Gin** for HTTP where services expose REST APIs
- **MongoDB 8** for **job execution metadata** and **job logs** (`job_metadata` and `job_logs` in the `jobby` database)
- **mongo-driver v2** (`go.mongodb.org/mongo-driver/v2`) for MongoDB access

The **jobs metadata** layer lives under `internal/jobs/metadata`. It defines:

- `JobMetadata` — interface implemented by `JobMetadataModel` for type-safe job records
- `JobsReader` / `JobsWriter` — CQRS-style persistence ports (queries vs commands); partial metadata updates use **`UpdateJob`** with **`JobsWriter.Update`** (no separate **`UpdateStatus`**—services assemble **`UpdateJob`** after domain rules)
- `MongoJobsReader` / `MongoJobsWriter` — MongoDB implementations; **`OpenMongoJobs`** connects once and returns reader, writer, and **`*mongo.Client`** while **verifying** required index names (indexes are created by database init / migrations, not by the application)
- Helpers such as `GenerateJobID` and `NewJobLog`

A small **`cmd/jobs-cli`** binary calls **`OpenMongoJobs`** from environment (see `docs/dev/setup.md`) for local smoke checks.
