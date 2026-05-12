//go:build integration

package metadata

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// bogusJob satisfies JobMetadata for Create/Update but is never *JobMetadataModel.
type bogusJobMeta struct{}

func (bogusJobMeta) GetJobID() string                    { return "20000000-0000-0000-0000-000000000001" }
func (bogusJobMeta) GetName() string                     { return "bogus" }
func (bogusJobMeta) GetStatus() JobStatus                { return JobStatusPending }
func (bogusJobMeta) GetPriority() int                    { return 5 }
func (bogusJobMeta) GetCreatedAt() time.Time             { return time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC) }
func (bogusJobMeta) GetStartedAt() *time.Time            { return nil }
func (bogusJobMeta) GetCompletedAt() *time.Time          { return nil }
func (bogusJobMeta) GetPayload() interface{}             { return map[string]any{} }
func (bogusJobMeta) GetMetadata() map[string]interface{} { return map[string]interface{}{} }
func (bogusJobMeta) GetError() string                    { return "" }
func (bogusJobMeta) GetRetryCount() int                  { return 0 }
func (bogusJobMeta) GetTags() []string                   { return nil }
func (bogusJobMeta) Validate() error                     { return nil }

// Integration tests require MongoDB (for example: docker compose up -d).
// Run: make test-integration
//
// MONGO_URI defaults to the same connection string as cmd/jobs-cli when unset.

func testMongoConfig(tb testing.TB) MongoConfig {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test (-short)")
	}
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://jobby_app:jobby_app_pass@localhost:27018/jobby?authSource=jobby"
	}
	return MongoConfig{
		URI:                uri,
		Database:           "jobby",
		CollectionMetadata: "job_metadata",
		CollectionLogs:     "job_logs",
		Timeout:            30 * time.Second,
		MaxPoolSize:        50,
		MinPoolSize:        0,
	}
}

func integrationConnect(tb testing.TB, ctx context.Context, cfg MongoConfig) *mongo.Client {
	tb.Helper()
	client, err := mongo.Connect(
		options.Client().
			ApplyURI(cfg.URI).
			SetTimeout(cfg.Timeout).
			SetMaxPoolSize(cfg.MaxPoolSize).
			SetMinPoolSize(cfg.MinPoolSize),
	)
	if err != nil {
		tb.Fatalf("connect to mongo: %v", err)
	}
	tb.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})
	if err := client.Ping(ctx, nil); err != nil {
		tb.Fatalf("ping mongo: %v (is the compose mongo service up?)", err)
	}
	return client
}

func clearJobCollections(ctx context.Context, db *mongo.Database, cfg MongoConfig) error {
	meta := db.Collection(cfg.CollectionMetadata)
	logs := db.Collection(cfg.CollectionLogs)
	if _, err := meta.DeleteMany(ctx, bson.M{}); err != nil {
		return fmt.Errorf("clear %q: %w", cfg.CollectionMetadata, err)
	}
	if _, err := logs.DeleteMany(ctx, bson.M{}); err != nil {
		return fmt.Errorf("clear %q: %w", cfg.CollectionLogs, err)
	}
	return nil
}

// setupIntegrationCollections removes all documents from metadata and logs collections.
// Indexes and validators from scripts/mongo-init.js stay in place so later tests (e.g. EnsureIndexes) still pass.
func setupIntegrationCollections(ctx context.Context, db *mongo.Database, cfg MongoConfig) error {
	return clearJobCollections(ctx, db, cfg)
}

// teardownIntegrationCollections clears both collections again so writes do not leak across runs.
func teardownIntegrationCollections(ctx context.Context, db *mongo.Database, cfg MongoConfig) {
	_ = clearJobCollections(ctx, db, cfg)
}

// prepareIntegrationMongoPersistence opens one client shared by collection cleanup plus reader/writer handles.
func prepareIntegrationMongoPersistence(t *testing.T) (ctx context.Context, cfg MongoConfig, reader *MongoJobsReader, writer *MongoJobsWriter) {
	t.Helper()
	cfg = testMongoConfig(t)
	ctx = context.Background()
	client := integrationConnect(t, ctx, cfg)
	db := client.Database(cfg.Database)
	if err := setupIntegrationCollections(ctx, db, cfg); err != nil {
		t.Fatalf("setupIntegrationCollections: %v", err)
	}
	var err error
	reader, err = NewMongoJobsReader(ctx, db, cfg)
	if err != nil {
		t.Fatalf("NewMongoJobsReader: %v", err)
	}
	writer = &MongoJobsWriter{
		metadataCollection: reader.metadataCollection,
		logsCollection:     reader.logsCollection,
	}
	t.Cleanup(func() {
		teardownIntegrationCollections(ctx, db, cfg)
	})
	return ctx, cfg, reader, writer
}

// TestIntegration_MongoJobsPersistence groups subtests in order: EnsureIndexes runs against a DB
// provisioned by compose/mongo-init.js; other subtests exercise CRUD paths with fresh fixtures.
func TestIntegration_MongoJobsPersistence(t *testing.T) {
	ctxBase := context.Background()
	cfgBase := testMongoConfig(t)

	t.Run("EnsureIndexes", func(t *testing.T) {
		reader, writer, client, err := OpenMongoJobs(ctxBase, cfgBase)
		if err != nil {
			t.Fatalf("OpenMongoJobs: %v", err)
		}
		defer func() {
			if err := client.Disconnect(ctxBase); err != nil {
				t.Errorf("Disconnect: %v", err)
			}
		}()
		_ = writer
		if !reader.IndexesPresent {
			t.Error("expected IndexesPresent true with compose mongo-init indexes")
		}
	})

	t.Run("Create_and_Get_roundTrip", func(t *testing.T) {
		ctx, _, reader, writer := prepareIntegrationMongoPersistence(t)

		job := NewJobMetadata(GenerateJobID(), "integration-create", map[string]any{"k": "v"})
		if err := writer.Create(ctx, job); err != nil {
			t.Fatalf("Create: %v", err)
		}

		got, err := reader.Get(ctx, job.JobID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got.GetJobID() != job.JobID {
			t.Errorf("JobID = %q, want %q", got.GetJobID(), job.JobID)
		}
		if got.GetName() != job.Name {
			t.Errorf("Name = %q, want %q", got.GetName(), job.Name)
		}
		if got.GetStatus() != JobStatusPending {
			t.Errorf("Status = %s, want %s", got.GetStatus(), JobStatusPending)
		}
	})

	t.Run("Create_errors", func(t *testing.T) {
		ctx, _, _, writer := prepareIntegrationMongoPersistence(t)

		if err := writer.Create(ctx, nil); err == nil {
			t.Fatal("Create(nil): want error")
		}

		// Persistence no longer validates; collection JSON schema rejects empty name / wrong shape.
		bad := NewJobMetadata(GenerateJobID(), "", map[string]any{})
		if err := writer.Create(ctx, bad); err == nil {
			t.Fatal("Create invalid name per DB schema: want error")
		}

		if err := writer.Create(ctx, bogusJobMeta{}); err == nil {
			t.Fatal("Create non-model job (wrong BSON shape vs schema): want error")
		}

		dup := NewJobMetadata(GenerateJobID(), "dup", nil)
		if err := writer.Create(ctx, dup); err != nil {
			t.Fatalf("first Create: %v", err)
		}
		if err := writer.Create(ctx, dup); err == nil {
			t.Fatal("duplicate Create: want error")
		}
	})

	t.Run("Get_errors", func(t *testing.T) {
		ctx, _, reader, _ := prepareIntegrationMongoPersistence(t)

		if _, err := reader.Get(ctx, ""); !errors.Is(err, ErrJobNotFound) {
			t.Fatalf("Get empty id: got %v, want %v", err, ErrJobNotFound)
		}
		if _, err := reader.Get(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrJobNotFound) {
			t.Fatalf("Get missing job: got %v, want %v", err, ErrJobNotFound)
		}
	})

	t.Run("Update_Delete", func(t *testing.T) {
		ctx, _, reader, writer := prepareIntegrationMongoPersistence(t)

		job := NewJobMetadata(GenerateJobID(), "to-update", map[string]any{"v": 1})
		job.AddTag("t1")
		if err := writer.Create(ctx, job); err != nil {
			t.Fatalf("Create: %v", err)
		}

		job.Metadata["edited"] = true
		meta := job.Metadata
		if err := writer.Update(ctx, job.JobID, UpdateJob{Metadata: &meta}); err != nil {
			t.Fatalf("Update: %v", err)
		}
		got, err := reader.Get(ctx, job.JobID)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := got.(*JobMetadataModel).Metadata["edited"]; !ok {
			t.Fatalf("expected metadata key edited after Update")
		}

		stale := NewJobMetadata(GenerateJobID(), "missing", nil)
		staleName := stale.Name
		if err := writer.Update(ctx, stale.JobID, UpdateJob{Name: &staleName}); !errors.Is(err, ErrJobNotFound) {
			t.Fatalf("Update stale: got %v, want %v", err, ErrJobNotFound)
		}

		wantName := "patched-bogus"
		if err := writer.Update(ctx, bogusJobMeta{}.GetJobID(), UpdateJob{Name: &wantName}); !errors.Is(err, ErrJobNotFound) {
			t.Fatalf("Update unknown bogus job id: got %v want %v", err, ErrJobNotFound)
		}

		if err := writer.Update(ctx, job.JobID, UpdateJob{}); !errors.Is(err, ErrEmptyUpdateJob) {
			t.Fatalf("Update empty patch: got %v, want %v", err, ErrEmptyUpdateJob)
		}

		if err := writer.Delete(ctx, ""); !errors.Is(err, ErrJobNotFound) {
			t.Fatalf("Delete empty id: %v", err)
		}
		if err := writer.Delete(ctx, stale.JobID); !errors.Is(err, ErrJobNotFound) {
			t.Fatalf("Delete missing: %v", err)
		}
		if err := writer.Delete(ctx, job.JobID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := reader.Get(ctx, job.JobID); !errors.Is(err, ErrJobNotFound) {
			t.Fatalf("after delete Get: %v", err)
		}
	})

	t.Run("List_filters_and_pagination", func(t *testing.T) {
		ctx, _, reader, writer := prepareIntegrationMongoPersistence(t)

		base := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
		a := NewJobMetadata(GenerateJobID(), "worker-a", nil)
		a.Tags = append(a.Tags, "x", "shared")
		a.Priority = 2
		a.CreatedAt = base

		b := NewJobMetadata(GenerateJobID(), "worker-b", nil)
		b.Tags = append(b.Tags, "y", "shared")
		b.Priority = 7
		b.CreatedAt = base.Add(time.Hour)

		c := NewJobMetadata(GenerateJobID(), "worker-c", nil)
		c.Tags = append(c.Tags, "z")
		c.Priority = 9
		c.CreatedAt = base.Add(2 * time.Hour)

		for _, j := range []*JobMetadataModel{a, b, c} {
			if err := writer.Create(ctx, j); err != nil {
				t.Fatalf("seed Create: %v", err)
			}
			run := JobStatusRunning
			tStarted := time.Now().UTC()
			if err := writer.Update(ctx, j.JobID, UpdateJob{Status: &run, StartedAt: &tStarted}); err != nil {
				t.Fatalf("Update seed running: %v", err)
			}
			done := JobStatusCompleted
			tCompleted := time.Now().UTC()
			if err := writer.Update(ctx, j.JobID, UpdateJob{Status: &done, CompletedAt: &tCompleted}); err != nil {
				t.Fatalf("Update seed completed: %v", err)
			}
		}

		none, err := reader.List(ctx, ListFilter{
			Names: []string{"__no_match__"},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(none) != 0 {
			t.Fatalf("List mismatch names: len %d", len(none))
		}

		onlyA, err := reader.List(ctx, ListFilter{
			Names:    []string{"worker-a"},
			Statuses: []JobStatus{JobStatusCompleted},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(onlyA) != 1 || onlyA[0].GetJobID() != a.JobID {
			t.Fatalf("filter name+status: %+v", onlyA)
		}

		byTag, err := reader.List(ctx, ListFilter{Tags: []string{"y"}})
		if err != nil {
			t.Fatal(err)
		}
		if len(byTag) != 1 || byTag[0].GetJobID() != b.JobID {
			t.Fatalf("tags filter: %+v", byTag)
		}

		minP := 6
		highPri, err := reader.List(ctx, ListFilter{
			MinPriority: &minP,
			Statuses:    []JobStatus{JobStatusCompleted},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(highPri) != 2 {
			t.Fatalf("min priority expected 2 jobs, got %d", len(highPri))
		}

		maxP := 7
		lowPri, err := reader.List(ctx, ListFilter{
			MaxPriority: &maxP,
			SortBy:      "createdAt",
			SortDesc:    false,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(lowPri) != 2 {
			t.Fatalf("max priority: want 2 jobs, got %d", len(lowPri))
		}
		if lowPri[0].GetJobID() != a.JobID || lowPri[1].GetJobID() != b.JobID {
			t.Fatalf("asc sort wrong order: %+v", jobIDs(lowPri))
		}

		afterWindow := base.Add(30 * time.Minute)
		between, err := reader.List(ctx, ListFilter{
			CreatedAfter:  &afterWindow,
			CreatedBefore: func() *time.Time { z := base.Add(3 * time.Hour); return &z }(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(between) != 2 {
			t.Fatalf("created window: got %d jobs %+v", len(between), jobIDs(between))
		}

		page, err := reader.List(ctx, ListFilter{
			SortBy:   "createdAt",
			SortDesc: true,
			Skip:     1,
			Limit:    1,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(page) != 1 || page[0].GetJobID() != b.JobID {
			t.Fatalf("Skip/Limit pagination: %+v", jobIDs(page))
		}
	})

	t.Run("UpdateJob_patch", func(t *testing.T) {
		ctx, _, reader, writer := prepareIntegrationMongoPersistence(t)

		j := NewJobMetadata(GenerateJobID(), "flow", nil)
		if err := writer.Create(ctx, j); err != nil {
			t.Fatal(err)
		}

		run := JobStatusRunning
		if err := writer.Update(ctx, "", UpdateJob{Status: &run}); !errors.Is(err, ErrJobNotFound) {
			t.Fatalf("Update empty job id: %v", err)
		}
		bad := JobStatus("nope")
		if err := writer.Update(ctx, j.JobID, UpdateJob{Status: &bad}); err == nil {
			t.Fatal("invalid status value: want write error from collection validator")
		}

		// No transition rules in the repository: pending -> completed is persisted if callers supply timestamps / schema permits.
		done := JobStatusCompleted
		tCompleted := time.Now().UTC()
		if err := writer.Update(ctx, j.JobID, UpdateJob{Status: &done, CompletedAt: &tCompleted}); err != nil {
			t.Fatalf("pending->completed: %v", err)
		}
		jm, err := reader.Get(ctx, j.JobID)
		afterSkip := mustJobModel(t, jm, err)
		if afterSkip.Status != JobStatusCompleted || afterSkip.CompletedAt == nil {
			t.Fatalf("after pending->completed: %+v", afterSkip)
		}

		j2 := NewJobMetadata(GenerateJobID(), "flow2", nil)
		if err := writer.Create(ctx, j2); err != nil {
			t.Fatal(err)
		}
		running := JobStatusRunning
		tStarted := time.Now().UTC()
		if err := writer.Update(ctx, j2.JobID, UpdateJob{Status: &running, StartedAt: &tStarted}); err != nil {
			t.Fatalf("pending->running: %v", err)
		}
		jmRun, err := reader.Get(ctx, j2.JobID)
		afterRun := mustJobModel(t, jmRun, err)
		if afterRun.GetStartedAt() == nil {
			t.Fatal("expected startedAt after running")
		}

		completed := JobStatusCompleted
		tDone := time.Now().UTC()
		if err := writer.Update(ctx, j2.JobID, UpdateJob{Status: &completed, CompletedAt: &tDone}); err != nil {
			t.Fatalf("running->completed: %v", err)
		}
		jmDone, err := reader.Get(ctx, j2.JobID)
		afterDone := mustJobModel(t, jmDone, err)
		if afterDone.GetCompletedAt() == nil {
			t.Fatal("expected completedAt")
		}

		revived := JobStatusRunning
		if err := writer.Update(ctx, j2.JobID, UpdateJob{Status: &revived}); err != nil {
			t.Fatalf("terminal->running (allowed at persistence layer): %v", err)
		}
		jmRev, err := reader.Get(ctx, j2.JobID)
		got := mustJobModel(t, jmRev, err)
		if got.Status != JobStatusRunning {
			t.Fatalf("status = %s want running", got.Status)
		}
	})

	t.Run("UpdateJob_running_preserves_startedAt", func(t *testing.T) {
		ctx, _, reader, writer := prepareIntegrationMongoPersistence(t)

		j := NewJobMetadata(GenerateJobID(), "started-once", nil)
		if err := writer.Create(ctx, j); err != nil {
			t.Fatal(err)
		}
		running := JobStatusRunning
		tStarted := time.Now().UTC()
		if err := writer.Update(ctx, j.JobID, UpdateJob{Status: &running, StartedAt: &tStarted}); err != nil {
			t.Fatal(err)
		}
		firstJM, err := reader.Get(ctx, j.JobID)
		if err != nil {
			t.Fatal(err)
		}
		first := firstJM.(*JobMetadataModel)
		saved := first.StartedAt
		if saved == nil {
			t.Fatal("nil startedAt")
		}

		failed := JobStatusFailed
		tCompleted := time.Now().UTC()
		if err := writer.Update(ctx, j.JobID, UpdateJob{Status: &failed, CompletedAt: &tCompleted}); err != nil {
			t.Fatal(err)
		}

		gotJM, err := reader.Get(ctx, j.JobID)
		if err != nil {
			t.Fatal(err)
		}
		got := gotJM.(*JobMetadataModel)
		if got.StartedAt == nil || !got.StartedAt.Equal(*saved) {
			t.Fatalf("startedAt changed: %+v vs %+v", got.StartedAt, saved)
		}
	})

	t.Run("IncrementRetryCount", func(t *testing.T) {
		ctx, _, reader, writer := prepareIntegrationMongoPersistence(t)

		if err := writer.IncrementRetryCount(ctx, ""); !errors.Is(err, ErrJobNotFound) {
			t.Fatalf("IncrementRetryCount empty id: %v", err)
		}
		if err := writer.IncrementRetryCount(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrJobNotFound) {
			t.Fatal(err)
		}

		j := NewJobMetadata(GenerateJobID(), "retryMe", nil)
		if err := writer.Create(ctx, j); err != nil {
			t.Fatal(err)
		}
		if err := writer.IncrementRetryCount(ctx, j.JobID); err != nil {
			t.Fatal(err)
		}
		if err := writer.IncrementRetryCount(ctx, j.JobID); err != nil {
			t.Fatal(err)
		}
		jm, err := reader.Get(ctx, j.JobID)
		got := mustJobModel(t, jm, err)
		if got.RetryCount != 2 {
			t.Fatalf("retryCount=%d want 2", got.RetryCount)
		}
	})

	t.Run("ClearJobExecutionTimestamps", func(t *testing.T) {
		ctx, _, reader, writer := prepareIntegrationMongoPersistence(t)

		if err := writer.ClearJobExecutionTimestamps(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, ErrJobNotFound) {
			t.Fatalf("ClearJobExecutionTimestamps missing job: %v", err)
		}

		j := NewJobMetadata(GenerateJobID(), "clear-ts", nil)
		t0 := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
		t1 := time.Date(2026, 2, 1, 13, 0, 0, 0, time.UTC)
		j.StartedAt = &t0
		j.CompletedAt = &t1
		j.Status = JobStatusFailed
		j.Error = "boom"
		if err := writer.Create(ctx, j); err != nil {
			t.Fatal(err)
		}

		if err := writer.ClearJobExecutionTimestamps(ctx, j.JobID); err != nil {
			t.Fatal(err)
		}

		jm, err := reader.Get(ctx, j.JobID)
		got := mustJobModel(t, jm, err)
		if got.StartedAt != nil || got.CompletedAt != nil {
			t.Fatalf("expected nil timestamps, got startedAt=%v completedAt=%v", got.StartedAt, got.CompletedAt)
		}
	})

	t.Run("Logs", func(t *testing.T) {
		ctx, _, reader, writer := prepareIntegrationMongoPersistence(t)

		jobID := GenerateJobID()
		job := NewJobMetadata(jobID, "log-test", nil)
		if err := writer.Create(ctx, job); err != nil {
			t.Fatal(err)
		}

		if err := writer.AddLog(ctx, JobLog{JobID: "", Level: LogLevelInfo, Message: "x", Timestamp: time.Now().UTC()}); err == nil {
			t.Fatal("AddLog empty job id: want error from schema / write")
		}
		if err := writer.AddLog(ctx, JobLog{JobID: jobID, Level: LogLevel("nope"), Message: "x", Timestamp: time.Now().UTC()}); err == nil {
			t.Fatal("invalid log level: want error from schema / write")
		}

		t0 := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
		tLogs := []JobLog{
			{JobID: jobID, Timestamp: t0.Add(-time.Minute), Level: LogLevelDebug, Message: "d"},
			{JobID: jobID, Timestamp: t0, Level: LogLevelInfo, Message: "i"},
			{JobID: jobID, Timestamp: t0.Add(time.Minute), Level: LogLevelWarn, Message: "w"},
			{JobID: jobID, Timestamp: t0.Add(2 * time.Minute), Level: LogLevelError, Message: "e"},
		}
		for _, lg := range tLogs {
			if err := writer.AddLog(ctx, lg); err != nil {
				t.Fatalf("AddLog %+v: %v", lg, err)
			}
		}
		fatalLog := JobLog{JobID: jobID, Timestamp: t0.Add(10 * time.Minute), Level: LogLevelFatal, Message: "f"}
		if err := writer.AddLog(ctx, fatalLog); err != nil {
			t.Fatal(err)
		}

		all, err := reader.GetLogs(ctx, jobID, LogFilter{})
		if err != nil {
			t.Fatal(err)
		}
		if len(all) != 5 {
			t.Fatalf("want 5 logs, got %d", len(all))
		}

		emptyJobLogs, err := reader.GetLogs(ctx, "", LogFilter{})
		if err != nil {
			t.Fatal(err)
		}
		if len(emptyJobLogs) != 0 {
			t.Fatalf("GetLogs empty job id: want no rows, got %d", len(emptyJobLogs))
		}

		levels, err := reader.GetLogs(ctx, jobID, LogFilter{Levels: []LogLevel{LogLevelDebug, LogLevelError}})
		if err != nil {
			t.Fatal(err)
		}
		if len(levels) != 2 {
			t.Fatalf("level filter: want 2, got %d", len(levels))
		}

		since := t0.Add(-30 * time.Second)
		until := t0.Add(90 * time.Second)
		window, err := reader.GetLogs(ctx, jobID, LogFilter{Since: &since, Until: &until})
		if err != nil {
			t.Fatal(err)
		}
		if len(window) != 2 {
			t.Fatalf("time window: want 2, got %d", len(window))
		}

		page, err := reader.GetLogs(ctx, jobID, LogFilter{Limit: 2, Skip: 1})
		if err != nil {
			t.Fatal(err)
		}
		if len(page) != 2 {
			t.Fatalf("log pagination len=%d want 2", len(page))
		}
		if page[0].Message != "e" || page[1].Message != "w" {
			t.Fatalf("pagination order: got %q %q want e w", page[0].Message, page[1].Message)
		}
	})
}

func mustJobModel(t *testing.T, jm JobMetadata, err error) *JobMetadataModel {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	model, ok := jm.(*JobMetadataModel)
	if !ok {
		t.Fatalf("not *JobMetadataModel: %T", jm)
	}
	return model
}

func jobIDs(jobs []JobMetadata) []string {
	out := make([]string, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, j.GetJobID())
	}
	return out
}
