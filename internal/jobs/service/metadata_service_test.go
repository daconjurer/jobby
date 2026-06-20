package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

type recordingJobsWriter struct {
	markDispatchedMatched     bool
	markDispatchedErr         error
	recordAttemptMatched      bool
	recordAttemptErr          error
	recordAttemptCalls        []recordAttemptCall
	markDispatchFailedMatched bool
	markDispatchFailedErr     error
	markDispatchFailedCalls   []markDispatchFailedCall
	markRunningMatched        bool
	markRunningErr            error
	completeIfRunningMatched  bool
	completeIfRunningErr      error
	completeIfRunningCalls    []completeIfRunningCall
	failIfNotTerminalMatched  bool
	failIfNotTerminalErr      error
	failIfNotTerminalCalls    []failIfNotTerminalCall
	logs                      []metadata.JobLog
	createErr                 error
}

type completeIfRunningCall struct {
	jobID       string
	completedAt time.Time
	metadata    *map[string]any
}

type failIfNotTerminalCall struct {
	jobID       string
	jobErr      metadata.JobError
	completedAt time.Time
}

type recordAttemptCall struct {
	jobID     string
	attempts  int
	lastError string
}

type markDispatchFailedCall struct {
	jobID    string
	errorMsg string
}

func (w *recordingJobsWriter) Create(context.Context, metadata.JobMetadata) error {
	return w.createErr
}

func (w *recordingJobsWriter) Update(context.Context, string, metadata.UpdateJob) error {
	return nil
}

func (w *recordingJobsWriter) Delete(context.Context, string) error { return nil }

func (w *recordingJobsWriter) IncrementRetryCount(context.Context, string) error { return nil }

func (w *recordingJobsWriter) ClearJobExecutionTimestamps(context.Context, string) error {
	return nil
}

func (w *recordingJobsWriter) MarkDispatchedIfPending(_ context.Context, _ string, _ time.Time) (bool, error) {
	return w.markDispatchedMatched, w.markDispatchedErr
}

func (w *recordingJobsWriter) RecordDispatchAttemptIfPending(_ context.Context, jobID string, attempts int, lastError string) (bool, error) {
	w.recordAttemptCalls = append(w.recordAttemptCalls, recordAttemptCall{
		jobID:     jobID,
		attempts:  attempts,
		lastError: lastError,
	})
	return w.recordAttemptMatched, w.recordAttemptErr
}

func (w *recordingJobsWriter) MarkDispatchFailedIfPending(_ context.Context, jobID, errorMsg string) (bool, error) {
	w.markDispatchFailedCalls = append(w.markDispatchFailedCalls, markDispatchFailedCall{
		jobID:    jobID,
		errorMsg: errorMsg,
	})
	return w.markDispatchFailedMatched, w.markDispatchFailedErr
}

func (w *recordingJobsWriter) MarkRunningIfDispatched(_ context.Context, _ string, _ time.Time) (bool, error) {
	return w.markRunningMatched, w.markRunningErr
}

func (w *recordingJobsWriter) CompleteIfRunning(_ context.Context, jobID string, completedAt time.Time, meta *map[string]any) (bool, error) {
	w.completeIfRunningCalls = append(w.completeIfRunningCalls, completeIfRunningCall{
		jobID:       jobID,
		completedAt: completedAt,
		metadata:    meta,
	})
	return w.completeIfRunningMatched, w.completeIfRunningErr
}

func (w *recordingJobsWriter) FailIfNotTerminal(_ context.Context, jobID string, jobErr metadata.JobError, completedAt time.Time) (bool, error) {
	w.failIfNotTerminalCalls = append(w.failIfNotTerminalCalls, failIfNotTerminalCall{
		jobID:       jobID,
		jobErr:      jobErr,
		completedAt: completedAt,
	})
	return w.failIfNotTerminalMatched, w.failIfNotTerminalErr
}

func (w *recordingJobsWriter) AddLog(_ context.Context, log metadata.JobLog) error {
	w.logs = append(w.logs, log)
	return nil
}

func (w *recordingJobsWriter) DeleteOldLogs(context.Context, time.Duration) (int64, error) {
	return 0, nil
}

type stubJobsReader struct {
	job metadata.JobMetadata
	err error
}

func (r stubJobsReader) Get(context.Context, string) (metadata.JobMetadata, error) {
	return r.job, r.err
}

func (stubJobsReader) List(context.Context, metadata.ListFilter) ([]metadata.JobMetadata, error) {
	return nil, nil
}

func (stubJobsReader) GetLogs(context.Context, string, metadata.LogFilter) ([]metadata.JobLog, error) {
	return nil, nil
}

func (stubJobsReader) CountJobs(context.Context, metadata.ListFilter) (int64, error) {
	return 0, nil
}

func (stubJobsReader) GetJobsByStatus(context.Context, metadata.JobStatus, int) ([]metadata.JobMetadata, error) {
	return nil, nil
}

func (stubJobsReader) GetPendingJobs(context.Context, int) ([]metadata.JobMetadata, error) {
	return nil, nil
}

func (stubJobsReader) GetDispatchedJobs(context.Context, int) ([]metadata.JobMetadata, error) {
	return nil, nil
}

func (stubJobsReader) GetRecentLogs(context.Context, string, int) ([]metadata.JobLog, error) {
	return nil, nil
}

func (stubJobsReader) GetErrorLogs(context.Context, string) ([]metadata.JobLog, error) {
	return nil, nil
}

func TestMetadataService_MarkJobDispatched_WritesLogWhenMatched(t *testing.T) {
	ctx := context.Background()
	writer := &recordingJobsWriter{markDispatchedMatched: true}
	svc := NewMetadataService(stubJobsReader{}, writer)

	if err := svc.MarkJobDispatched(ctx, "job-1"); err != nil {
		t.Fatal(err)
	}
	if len(writer.logs) != 1 {
		t.Fatalf("logs=%d want 1", len(writer.logs))
	}
	if writer.logs[0].Message != "Job dispatched to queue" {
		t.Fatalf("log message=%q", writer.logs[0].Message)
	}
}

func TestMetadataService_MarkJobDispatched_SkipsLogWhenNotMatched(t *testing.T) {
	ctx := context.Background()
	writer := &recordingJobsWriter{markDispatchedMatched: false}
	svc := NewMetadataService(stubJobsReader{}, writer)

	if err := svc.MarkJobDispatched(ctx, "job-1"); err != nil {
		t.Fatal(err)
	}
	if len(writer.logs) != 0 {
		t.Fatalf("logs=%v want none", writer.logs)
	}
}

func TestMetadataService_RecordDispatchAttempt_RecordsWhenMatched(t *testing.T) {
	ctx := context.Background()
	writer := &recordingJobsWriter{recordAttemptMatched: true}
	svc := NewMetadataService(stubJobsReader{}, writer)

	if err := svc.RecordDispatchAttempt(ctx, "job-1", 2, "broker down"); err != nil {
		t.Fatal(err)
	}
	if len(writer.recordAttemptCalls) != 1 {
		t.Fatalf("calls=%v", writer.recordAttemptCalls)
	}
	got := writer.recordAttemptCalls[0]
	if got.jobID != "job-1" || got.attempts != 2 || got.lastError != "broker down" {
		t.Fatalf("call=%+v", got)
	}
}

func TestMetadataService_RecordDispatchAttempt_NoOpWhenNotMatched(t *testing.T) {
	ctx := context.Background()
	writer := &recordingJobsWriter{recordAttemptMatched: false}
	svc := NewMetadataService(stubJobsReader{}, writer)

	if err := svc.RecordDispatchAttempt(ctx, "job-1", 2, "broker down"); err != nil {
		t.Fatal(err)
	}
	if len(writer.recordAttemptCalls) != 1 {
		t.Fatalf("writer should still be called once, calls=%v", writer.recordAttemptCalls)
	}
}

func TestMetadataService_MarkJobDispatchFailed_WritesConditionalUpdate(t *testing.T) {
	ctx := context.Background()
	writer := &recordingJobsWriter{markDispatchFailedMatched: true}
	svc := NewMetadataService(stubJobsReader{}, writer)

	err := svc.MarkJobDispatchFailed(ctx, "job-1", errors.New("publish exhausted"))
	if err != nil {
		t.Fatal(err)
	}
	if len(writer.markDispatchFailedCalls) != 1 {
		t.Fatalf("calls=%v", writer.markDispatchFailedCalls)
	}
	got := writer.markDispatchFailedCalls[0]
	if got.jobID != "job-1" || got.errorMsg != "publish exhausted" {
		t.Fatalf("call=%+v", got)
	}
}

func TestMetadataService_MarkJobDispatchFailed_NoOpWhenNotMatched(t *testing.T) {
	ctx := context.Background()
	writer := &recordingJobsWriter{markDispatchFailedMatched: false}
	svc := NewMetadataService(stubJobsReader{}, writer)

	if err := svc.MarkJobDispatchFailed(ctx, "job-1", errors.New("late failure")); err != nil {
		t.Fatal(err)
	}
	if len(writer.markDispatchFailedCalls) != 1 {
		t.Fatalf("calls=%v", writer.markDispatchFailedCalls)
	}
}

func TestMetadataService_MarkJobDispatchFailed_NilErrorUsesEmptyMessage(t *testing.T) {
	ctx := context.Background()
	writer := &recordingJobsWriter{markDispatchFailedMatched: true}
	svc := NewMetadataService(stubJobsReader{}, writer)

	if err := svc.MarkJobDispatchFailed(ctx, "job-1", nil); err != nil {
		t.Fatal(err)
	}
	if len(writer.markDispatchFailedCalls) != 1 || writer.markDispatchFailedCalls[0].errorMsg != "" {
		t.Fatalf("calls=%v want empty error message", writer.markDispatchFailedCalls)
	}
}

func TestMetadataService_CreateJob_AppliesOptionsAndLogs(t *testing.T) {
	ctx := context.Background()
	writer := &recordingJobsWriter{}
	svc := NewMetadataService(stubJobsReader{}, writer)
	priority := 3

	job, err := svc.CreateJob(ctx, "load-products", map[string]any{"k": "v"}, CreateJobOptions{
		Priority: &priority,
		Tags:     []string{"a"},
		Metadata: map[string]any{"region": "eu"},
		Topic:    "topic-a",
	})
	if err != nil {
		t.Fatal(err)
	}
	if job.Priority != 3 || job.Topic != "topic-a" {
		t.Fatalf("job=%+v", job)
	}
	if len(job.Tags) != 1 || job.Tags[0] != "a" {
		t.Fatalf("tags=%v", job.Tags)
	}
	if job.Metadata["region"] != "eu" {
		t.Fatalf("metadata=%v", job.Metadata)
	}
	if len(writer.logs) != 1 || writer.logs[0].Message != "Job created: load-products" {
		t.Fatalf("logs=%v", writer.logs)
	}
}

func TestMetadataService_CreateJob_ValidationFailure(t *testing.T) {
	ctx := context.Background()
	writer := &recordingJobsWriter{}
	svc := NewMetadataService(stubJobsReader{}, writer)

	_, err := svc.CreateJob(ctx, "", nil, CreateJobOptions{})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if len(writer.logs) != 0 {
		t.Fatalf("logs=%v want none", writer.logs)
	}
}

func TestMetadataService_RetryJob_RejectsNonFailedStatus(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata(metadata.GenerateJobID(), "load-products", nil)
	job.Status = metadata.JobStatusPendingDispatch
	svc := NewMetadataService(stubJobsReader{job: job}, &recordingJobsWriter{})

	err := svc.RetryJob(ctx, job.JobID)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMetadataService_CancelJob_RejectsTerminalStatus(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata(metadata.GenerateJobID(), "load-products", nil)
	job.Status = metadata.JobStatusFailed
	svc := NewMetadataService(stubJobsReader{job: job}, &recordingJobsWriter{})

	err := svc.CancelJob(ctx, job.JobID, "too late")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMetadataService_StartJob_WritesLogWhenMatched(t *testing.T) {
	ctx := context.Background()
	writer := &recordingJobsWriter{markRunningMatched: true}
	svc := NewMetadataService(stubJobsReader{}, writer)

	matched, err := svc.StartJob(ctx, "job-1")
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Fatal("expected matched=true")
	}
	if len(writer.logs) != 1 {
		t.Fatalf("logs=%d want 1", len(writer.logs))
	}
	if writer.logs[0].Message != "Job started" {
		t.Fatalf("log message=%q", writer.logs[0].Message)
	}
}

func TestMetadataService_StartJob_SkipsLogWhenNotMatched(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata(metadata.GenerateJobID(), "load-products", nil)
	job.Status = metadata.JobStatusRunning
	writer := &recordingJobsWriter{markRunningMatched: false}
	svc := NewMetadataService(stubJobsReader{job: job}, writer)

	matched, err := svc.StartJob(ctx, job.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if matched {
		t.Fatal("expected matched=false")
	}
	if len(writer.logs) != 0 {
		t.Fatalf("logs=%v want none", writer.logs)
	}
}

func TestMetadataService_StartJob_HandlesGetErrorOnDuplicateDelivery(t *testing.T) {
	ctx := context.Background()
	writer := &recordingJobsWriter{markRunningMatched: false}
	readerErr := errors.New("db down")
	svc := NewMetadataService(stubJobsReader{err: readerErr}, writer)

	matched, err := svc.StartJob(ctx, "job-1")
	if err != nil {
		t.Fatal("expected no error on duplicate delivery even if Get fails")
	}
	if matched {
		t.Fatal("expected matched=false")
	}
	if len(writer.logs) != 0 {
		t.Fatalf("logs=%v want none", writer.logs)
	}
}

func TestMetadataService_CompleteJob_ReturnsErrJobAlreadyTerminalFromFailed(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata(metadata.GenerateJobID(), "test-job", nil)
	job.Status = metadata.JobStatusFailed
	now := time.Now()
	job.StartedAt = &now
	job.CompletedAt = &now
	job.Errors = []metadata.JobError{{Type: metadata.JobErrorTypeExecution, RetryAttempt: 0, Error: "test error", Timestamp: now}}

	svc := NewMetadataService(stubJobsReader{job: job}, &recordingJobsWriter{})

	err := svc.CompleteJob(ctx, job.JobID, nil)
	if !errors.Is(err, metadata.ErrJobAlreadyTerminal) {
		t.Fatalf("expected ErrJobAlreadyTerminal, got %v", err)
	}
}

func TestMetadataService_CompleteJob_ReturnsErrJobAlreadyTerminalFromCompleted(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata(metadata.GenerateJobID(), "test-job", nil)
	job.Status = metadata.JobStatusCompleted
	now := time.Now()
	job.StartedAt = &now
	job.CompletedAt = &now

	svc := NewMetadataService(stubJobsReader{job: job}, &recordingJobsWriter{})

	err := svc.CompleteJob(ctx, job.JobID, nil)
	if !errors.Is(err, metadata.ErrJobAlreadyTerminal) {
		t.Fatalf("expected ErrJobAlreadyTerminal, got %v", err)
	}
}

func TestMetadataService_CompleteJob_ReturnsErrJobAlreadyTerminalFromCancelled(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata(metadata.GenerateJobID(), "test-job", nil)
	job.Status = metadata.JobStatusCancelled
	now := time.Now()
	job.StartedAt = &now
	job.CompletedAt = &now

	svc := NewMetadataService(stubJobsReader{job: job}, &recordingJobsWriter{})

	err := svc.CompleteJob(ctx, job.JobID, nil)
	if !errors.Is(err, metadata.ErrJobAlreadyTerminal) {
		t.Fatalf("expected ErrJobAlreadyTerminal, got %v", err)
	}
}

func TestMetadataService_CompleteJob_SucceedsFromRunning(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata(metadata.GenerateJobID(), "test-job", nil)
	job.Status = metadata.JobStatusRunning
	now := time.Now()
	job.StartedAt = &now
	writer := &recordingJobsWriter{completeIfRunningMatched: true}

	svc := NewMetadataService(stubJobsReader{job: job}, writer)

	err := svc.CompleteJob(ctx, job.JobID, nil)
	if err != nil {
		t.Fatalf("expected no error from running status, got %v", err)
	}
	if len(writer.completeIfRunningCalls) != 1 {
		t.Fatalf("completeIfRunningCalls=%v", writer.completeIfRunningCalls)
	}
	if len(writer.logs) != 1 || writer.logs[0].Message != "Job completed successfully" {
		t.Fatalf("logs=%v", writer.logs)
	}
}

func TestMetadataService_FailJob_ReturnsErrJobAlreadyTerminalFromFailed(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata(metadata.GenerateJobID(), "test-job", nil)
	job.Status = metadata.JobStatusFailed
	now := time.Now()
	job.StartedAt = &now
	job.CompletedAt = &now
	job.Errors = []metadata.JobError{{Type: metadata.JobErrorTypeExecution, RetryAttempt: 0, Error: "test error", Timestamp: now}}

	svc := NewMetadataService(stubJobsReader{job: job}, &recordingJobsWriter{})

	err := svc.FailJob(ctx, job.JobID, errors.New("new error"))
	if !errors.Is(err, metadata.ErrJobAlreadyTerminal) {
		t.Fatalf("expected ErrJobAlreadyTerminal, got %v", err)
	}
}

func TestMetadataService_FailJob_ReturnsErrJobAlreadyTerminalFromCompleted(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata(metadata.GenerateJobID(), "test-job", nil)
	job.Status = metadata.JobStatusCompleted
	now := time.Now()
	job.StartedAt = &now
	job.CompletedAt = &now

	svc := NewMetadataService(stubJobsReader{job: job}, &recordingJobsWriter{})

	err := svc.FailJob(ctx, job.JobID, errors.New("new error"))
	if !errors.Is(err, metadata.ErrJobAlreadyTerminal) {
		t.Fatalf("expected ErrJobAlreadyTerminal, got %v", err)
	}
}

func TestMetadataService_FailJob_ReturnsErrJobAlreadyTerminalFromCancelled(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata(metadata.GenerateJobID(), "test-job", nil)
	job.Status = metadata.JobStatusCancelled
	now := time.Now()
	job.StartedAt = &now
	job.CompletedAt = &now

	svc := NewMetadataService(stubJobsReader{job: job}, &recordingJobsWriter{})

	err := svc.FailJob(ctx, job.JobID, errors.New("new error"))
	if !errors.Is(err, metadata.ErrJobAlreadyTerminal) {
		t.Fatalf("expected ErrJobAlreadyTerminal, got %v", err)
	}
}

func TestMetadataService_FailJob_SucceedsFromRunning(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata(metadata.GenerateJobID(), "test-job", nil)
	job.Status = metadata.JobStatusRunning
	now := time.Now()
	job.StartedAt = &now
	writer := &recordingJobsWriter{failIfNotTerminalMatched: true}

	svc := NewMetadataService(stubJobsReader{job: job}, writer)

	err := svc.FailJob(ctx, job.JobID, errors.New("handler error"))
	if err != nil {
		t.Fatalf("expected no error from running status, got %v", err)
	}
	if len(writer.failIfNotTerminalCalls) != 1 {
		t.Fatalf("failIfNotTerminalCalls=%v", writer.failIfNotTerminalCalls)
	}
	got := writer.failIfNotTerminalCalls[0]
	if got.jobErr.Error != "handler error" || got.jobErr.Type != metadata.JobErrorTypeExecution {
		t.Fatalf("jobErr=%+v", got.jobErr)
	}
	if len(writer.logs) != 1 || writer.logs[0].Message != "Job failed" {
		t.Fatalf("logs=%v", writer.logs)
	}
}

func TestMetadataService_FailJob_SucceedsFromDispatched(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata(metadata.GenerateJobID(), "test-job", nil)
	job.Status = metadata.JobStatusDispatched
	writer := &recordingJobsWriter{failIfNotTerminalMatched: true}

	svc := NewMetadataService(stubJobsReader{job: job}, writer)

	err := svc.FailJob(ctx, job.JobID, errors.New("api fail"))
	if err != nil {
		t.Fatalf("expected no error from dispatched status, got %v", err)
	}
	if len(writer.failIfNotTerminalCalls) != 1 {
		t.Fatalf("failIfNotTerminalCalls=%v", writer.failIfNotTerminalCalls)
	}
}
