# Dispatch/executor race (historical)

> **Deprecated.** This document records the dispatch/executor race that existed under the single-field `status` design and the options that were considered before the split.
> The current implementation uses two independent fields, `dispatchStatus` and `executionStatus`, described in [dispatch-execution-status-split overview](../planning/dispatch-execution-status-split/overview.md).

## The race

`job_metadata.status` was shared between two services with different concerns:

1. The dispatcher used it for retry bookkeeping: `pending_dispatch → dispatched`.
2. The executor used it as a readiness precondition: `MarkRunningIfDispatched` gated on `status == dispatched` before starting a job.

Because the dispatcher publishes to Pulsar and then marks `dispatched`, an executor could receive the Pulsar message and try to start the job before the dispatcher's confirm write landed in MongoDB. `MarkRunningIfDispatched` would see `status == pending_dispatch` and treat the legitimate delivery as a duplicate, failing or skipping the execution.

## Options considered

The following options were evaluated before the status split was chosen.
They are all superseded by the two-field design.

### Option A — Enforce dispatcher ordering

Mark `status = dispatched` in MongoDB before publishing to Pulsar.
This makes the executor's precondition true before the message is sent, but it is crash-unsafe: a crash between the MongoDB write and the publish would silently lose the job.

### Option B — Make either ordering safe

Keep the publish-then-mark order, but allow the executor to start the job even when `status` is still `pending_dispatch`.
This requires approximating `dispatchedAt` after the fact and complicates the dispatcher's retry logic, because `pending_dispatch` no longer cleanly means "not yet published."

### Option C — Add a separate "claimed" status

Introduce a `claimed` value in the same `status` field that the executor writes before running.
Both the dispatcher and executor still write to one shared field, so the race is only narrowed, not eliminated; interleavings between `dispatched` and `claimed` remain possible.

### Option D — Optimistic concurrency on a version field

Add a monotonic version or timestamp and retry conflicting updates.
This prevents lost updates but does not remove the shared precondition; it also adds complexity to every status write path.

### Option E — Treat Pulsar receipt as proof of dispatch

Remove the `status == dispatched` gate entirely and let the executor run as soon as it receives the message.
This is the conceptual core of the chosen design, but with a single `status` field there is nowhere to record the executor-owned state independently.

## Superseded by the status split

The [dispatch-execution-status-split design](../planning/dispatch-execution-status-split/dispatch-execution-status-split.md) gives each actor its own field:

- `dispatchStatus` is owned by the dispatcher.
- `executionStatus` is owned by the executor and cancel/retry endpoints.

`MarkRunningIfDispatched` becomes `UpdateOne({jobId, executionStatus: not_started}, {$set: {executionStatus: running}})` and never reads `dispatchStatus`. The dispatcher can confirm before, during, or after the executor's claim; no interleaving makes a legitimate delivery look like a duplicate, because there is no longer a shared precondition to race over.
