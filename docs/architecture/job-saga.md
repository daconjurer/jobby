# Job dispatch saga

How a job moves from HTTP enqueue to Pulsar publish and a confirmed MongoDB status. This is the **saga protocol** (phases 1–3), not the pulsar-job-executor project milestones.

For worker wiring (change stream, poll fallback, Compose), see [dispatch-worker.md](./dispatch-worker.md).

---

## Phases

| Phase | Action | System of record |
|-------|--------|------------------|
| **1** | Persist job with dispatch intent (`status: pending_dispatch`, resolved `topic`, payload) | MongoDB `job_metadata` |
| **2** | Publish `JobMessage` to Pulsar (message key = `jobId`) | Pulsar |
| **3** | Confirm outcome: `dispatched`, or record retry / `dispatch_failed` | MongoDB `job_metadata` |

**HTTP contract:** phase 1 completes on **201**. Phases 2–3 run asynchronously in `cmd/jobs-dispatcher`.

**Recovery model:** forward retry while `pending_dispatch`; no rollback of phase 1. Publish (phase 2) is safe to repeat; status updates (phase 3) use conditional writes on `status`.

---

## Code map

```text
Phase 1 — persist
  POST /api/jobs | jobs-cli create
    → service.EnqueueService.Enqueue
    → service.MetadataService.CreateJob

Phases 2–3 — publish + confirm
  dispatch.Worker (change stream + poll)
    → dispatch.JobDispatchHandler.HandleDispatch
       phase 2: dispatch.JobDispatchPublisher.Publish
                 (Pulsar: pulsar.DispatchPublisher → pulsar.JobProducer)
       phase 3: dispatch.JobUpdater (MetadataService)
                 MarkJobDispatched | RecordDispatchAttempt | MarkJobDispatchFailed
```

`JobDispatchHandler` implementors (notably `*dispatch.DispatchHandler`) orchestrate **phases 2–3** for one `JobDispatchProjection` per call. Phase 2 is delegated to `JobDispatchPublisher`; Pulsar wiring lives in `internal/jobs/pulsar/` and is composed in `cmd/jobs-dispatcher`. Triggers pass projections built from `pending_dispatch` rows via `JobDispatchFromMetadata`.

---

## `HandleDispatch` outcomes (phase 3)

| Publish result | Phase 3 action |
|----------------|----------------|
| Success | `MarkJobDispatched` → `dispatched` |
| Error, attempts remaining | `RecordDispatchAttempt` → stay `pending_dispatch` (poll/stream retries) |
| Error, max attempts reached | `MarkJobDispatchFailed` → `dispatch_failed` |

Duplicate `HandleDispatch` calls for the same job are expected; publish and status transitions must remain idempotent per `jobId`.

---

## Error history (`errors[]`)

Failed jobs store an **`errors`** array on `job_metadata` instead of a single `error` string (migration **004**). Each entry is a **`JobError`**:

| Field | Description |
|-------|-------------|
| `type` | **Required.** `execution` (handler / `FailJob`) or `dispatch` (`dispatch_failed`) |
| `retryAttempt` | Value of `retryCount` when the error occurred (`0` = first attempt) |
| `error` | Error message |
| `timestamp` | When the error was recorded (UTC) |

**Retry behaviour:** `POST /api/jobs/:id/retry` increments `retryCount` and resets dispatch fields but **does not clear** `errors`. A subsequent failure appends a new entry, so operators can see failure patterns across retries.

**HTTP API:**

- `GET /api/jobs/:id` — response includes `errors` (may be empty for non-failed jobs).
- `POST /api/jobs/:id/fail` — request body uses the same shape: `{"errors":[{"error":"message"}]}`. Response is the updated job document (same as GET).
- `POST /api/jobs/:id/retry` — no body; errors preserved.

**CLI:** `jobs-cli fail --error "msg"` calls `MetadataService.FailJob` directly (not the HTTP body shape). Use `--output json` on `get` to inspect `errors`.
