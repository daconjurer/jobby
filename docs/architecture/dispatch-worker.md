# Dispatch worker architecture

Internal reference for `internal/jobs/dispatch/` — how the MongoDB **watch client** (change stream) and the **dispatch worker** (dual triggers + saga handler) fit together via **`internal/jobs/dispatchruntime`** in **`cmd/jobs-dispatcher`**. The HTTP API in **`cmd/jobs-server`** only persists `pending_dispatch`; dispatch runs in a separate process.

**Code layout**

```text
internal/jobs/dispatch/       # orchestration + saga (transport-agnostic)
├── worker.go, handler.go     # DispatchWorker, DispatchHandler
├── types.go                  # JobDispatchPublisher, JobDispatchHandler, JobUpdater, …
└── (unit tests only)

internal/jobs/dispatchruntime/  # composition root (wire mongo + pulsar + dispatch)
├── config.go                 # Config, ConfigFromEnv
└── runtime.go                # New, Run, Close

internal/jobs/mongodb/        # Mongo implementations
├── stream.go                 # StreamWatcher (change stream consumer)
├── pending.go                # PendingFetcher (poll fallback queries)
├── resume_token.go           # FileResumeTokenStore | NopResumeTokenStore
└── mongo_connection.go       # OpenMongoWatchClient (dedicated watch client)

internal/jobs/pulsar/         # Pulsar adapter (DispatchPublisher, JobProducer)

internal/jobs/integrationtest/  # cross-package saga integration tests

cmd/jobs-dispatcher/          # env loading + dispatchruntime.New
```

---

## Watch client — change stream path

The watch path uses a **separate MongoDB client** (small pool, long-lived cursor) and `mongodb.StreamWatcher`, which implements `dispatch.StreamRunner`.

```mermaid
flowchart TB
    subgraph bootstrap["dispatchruntime.New"]
        CFG["dispatchruntime.Config.Stream"]
        MONGO["dispatchruntime.Config.Mongo"]
        TOK_CFG["Stream.ResumeTokenPath"]
    end

    subgraph watch_client["Dedicated watch Mongo client"]
        OMW["mongodb.OpenMongoWatchClient(ctx, mongoConfig, maxPoolSize)"]
        WC["*mongo.Client (maxPoolSize ≈ 2)"]
        COLL["*mongo.Collection job_metadata"]
        OMW --> WC
        OMW --> COLL
    end

    subgraph resume["Resume token (optional)"]
        TOK_CFG -->|non-empty| FILE["FileResumeTokenStore"]
        TOK_CFG -->|empty| NOP["NopResumeTokenStore"]
    end

    subgraph stream_watcher["mongodb.StreamWatcher"]
        RUN["Run(ctx) — reconnect loop"]
        ONCE["runOnce(ctx)"]
        LOAD["tokens.Load() → SetResumeAfter"]
        WATCH["collection.Watch(insertPendingDispatchPipeline)"]
        LOOP["for stream.Next → handleEvent"]
        SAVE["tokens.Save(stream.ResumeToken())"]
        RUN --> ONCE
        ONCE --> LOAD --> WATCH --> LOOP
        LOOP --> SAVE
    end

    subgraph pipeline["Change stream filter"]
        PIPE["$match: operationType=insert<br/>fullDocument.status=pending_dispatch"]
    end

    subgraph event["Per insert event"]
        DEC["Decode fullDocument → JobMetadataModel"]
        MAP["JobDispatchFromMetadata → JobDispatchProjection"]
        VAL["topic non-empty?"]
        HND["JobDispatchHandler.HandleDispatch(ctx, projection)"]
        DEC --> MAP --> VAL --> HND
    end

    MONGO --> OMW
    CFG --> OMW
    COLL --> stream_watcher
    FILE --> stream_watcher
    NOP --> stream_watcher
    PIPE --> WATCH
    LOOP --> event

    HND --> SAGA["dispatch.DispatchHandler<br/>(shared saga — see below)"]

    style watch_client fill:#e8f4fc
    style stream_watcher fill:#f0f8e8
```

### Behavior

| Step | What happens |
|------|----------------|
| **Client isolation** | API traffic uses `mongodb.OpenMongoJobs`; the watcher uses `mongodb.OpenMongoWatchClient` so the change stream does not compete with the main connection pool. |
| **Filter** | Only **inserts** whose `fullDocument.status` is `pending_dispatch` (jobs created by enqueue, not updates). |
| **Resilience** | `Run` loops: on cursor error it logs and reconnects; `runOnce` reopens the stream. |
| **Resume** | After each successful `Next`, the resume token is persisted (file) or dropped (nop). |
| **Handler contract** | `StreamWatcher` depends on `dispatch.JobDispatchHandler` — `main` passes `*dispatch.DispatchHandler` as that implementation. |

Replica set is required for change streams (see [Environment](../dev/environment.md)).

---

## Dispatch worker — dual triggers + saga

`dispatch.DispatchWorker` starts the watch goroutine and runs a **poll fallback** on the main goroutine. Both paths call the same `Handler` (Pulsar publish + metadata status updates). **`dispatchruntime.New`** assembles the worker, Pulsar publisher, and Mongo adapters.

```mermaid
flowchart TB
    subgraph worker["dispatch.DispatchWorker"]
        RUN["Run(ctx)"]
        GO["go stream.Run(ctx)"]
        POLL["runPoll(ctx) — blocking loop"]
        RUN --> GO
        RUN --> POLL
    end

    subgraph primary["Primary trigger — change stream"]
        SW["StreamRunner<br/>mongodb.StreamWatcher"]
        GO --> SW
        SW --> H1["DispatchHandler.HandleDispatch(projection)"]
    end

    subgraph secondary["Secondary trigger — poll fallback"]
        TICK["Ticker(PollInterval)"]
        ONCE["pollOnce — also runs immediately"]
        FETCH["PendingJobFetcher.FetchPending<br/>mongodb.PendingFetcher"]
        BATCH["status=pending_dispatch<br/>dispatchAttempts &lt; max<br/>topic set, sorted by createdAt"]
        LOOP["for each job → HandleDispatch"]
        POLL --> ONCE
        ONCE --> FETCH
        TICK --> ONCE
        FETCH --> BATCH --> LOOP
        LOOP --> H2["DispatchHandler.HandleDispatch(projection)"]
    end

    subgraph saga["dispatch.DispatchHandler — publish + confirm"]
        TOPIC{"topic empty?"}
        PUB["JobDispatchPublisher.Publish(projection)"]
        OK["jobs.MarkJobDispatched"]
        FAIL["RecordDispatchAttempt"]
        MAX{"attempts ≥ MaxAttempts?"}
        DFAIL["jobs.MarkJobDispatchFailed"]
        STAY["return nil — stay pending_dispatch"]
        TOPIC -->|yes| ERR["error"]
        TOPIC -->|no| PUB
        PUB -->|success| OK
        PUB -->|error| FAIL --> MAX
        MAX -->|yes| DFAIL
        MAX -->|no| STAY
    end

    subgraph deps["Dependencies wired in dispatchruntime.New"]
        PUB_ADP["pulsar.DispatchPublisher"]
        PROD["pulsar.JobProducer"]
        META["MetadataService implements JobUpdater"]
        JOBS_COLL["mongoClient → job_metadata<br/>(poll fetcher)"]
    end

    H1 --> saga
    H2 --> saga
    PUB --> PUB_ADP --> PROD
    OK --> META
    FAIL --> META
    DFAIL --> META
    FETCH --> JOBS_COLL

    subgraph upstream["Upstream (persist — not in Worker)"]
        HTTP["POST /api/jobs"]
        ENQ["EnqueueService → CreateJob"]
        INS["INSERT pending_dispatch + topic"]
        HTTP --> ENQ --> INS
    end

    INS -.->|insert event| SW
    INS -.->|missed / retry| FETCH

    style worker fill:#fff4e6
    style saga fill:#e8f4fc
    style upstream fill:#f5f5f5
```

### End-to-end flow

```text
Persist (HTTP)          Publish + confirm (Worker)
─────────────────       ─────────────────────────────────────
Enqueue → INSERT        ┌─ Change stream (primary, fast)
  pending_dispatch      │
                          └─ Poll (fallback, catches gaps / retries)
                                    │
                                    ▼
                              DispatchHandler.HandleDispatch
                                    │
                    ┌───────────────┴───────────────┐
                    ▼                               ▼
             Pulsar publish                  Mongo status update
             (jobId key)                     dispatched / attempts / dispatch_failed
```

### Why two triggers

| Trigger | Role |
|---------|------|
| **Change stream** | Low-latency reaction to new `pending_dispatch` rows after enqueue. |
| **Poll** | Safety net: stream gaps, restarts without resume token, publish retries left in `pending_dispatch`, broker outages. |

HTTP returns **201** after the persist step; publish and confirm run asynchronously in the dispatch worker.
