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

**Operational entrypoints**

| Binary | Role |
|--------|------|
| **`cmd/jobs-server`** | Gin HTTP API under `/api/jobs` via **`handler.JobsHandler`** |
| **`cmd/jobs-cli`** | Cobra CLI with the same job operations via **`service.MetadataService`** (no HTTP hop) |

Both binaries bootstrap MongoDB through **`OpenMongoJobs`** and share validation and state-transition rules. The CLI defaults to JSON stdout for scripting; **`--output table`** formats **`list`**, **`stats`**, and **`logs`** for interactive use.

See `docs/dev/setup.md` for local run and test workflows.
