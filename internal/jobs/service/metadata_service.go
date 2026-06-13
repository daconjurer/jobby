package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/dispatch"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

var _ dispatch.JobUpdater = (*MetadataService)(nil)

// MetadataService provides business logic for job metadata operations.
type MetadataService struct {
	reader metadata.JobsReader
	writer metadata.JobsWriter
}

// NewMetadataService creates a new metadata service.
func NewMetadataService(reader metadata.JobsReader, writer metadata.JobsWriter) *MetadataService {
	return &MetadataService{
		reader: reader,
		writer: writer,
	}
}

// CreateJob creates a new job with metadata.
func (s *MetadataService) CreateJob(ctx context.Context, name string, payload map[string]any, options CreateJobOptions) (*metadata.JobMetadataModel, error) {
	jobID := metadata.GenerateJobID()
	job := metadata.NewJobMetadata(jobID, name, payload)

	if options.Priority != nil {
		job.Priority = *options.Priority
	}
	if len(options.Tags) > 0 {
		job.Tags = options.Tags
	}
	if len(options.Metadata) > 0 {
		job.Metadata = options.Metadata
	}
	if options.Topic != "" {
		job.Topic = options.Topic
	}

	if err := job.Validate(); err != nil {
		return nil, fmt.Errorf("invalid job: %w", err)
	}

	if err := s.writer.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	logEntry := metadata.NewJobLogWithSource(
		jobID,
		metadata.LogLevelInfo,
		fmt.Sprintf("Job created: %s", name),
		"service",
	)
	if err := s.writer.AddLog(ctx, logEntry); err != nil {
		log.Printf("warning: failed to log job creation: %v", err)
	}

	return job, nil
}

// CreateJobOptions holds options for creating a job.
type CreateJobOptions struct {
	Priority *int
	Tags     []string
	Metadata map[string]any
	Topic    string
}

// GetJob retrieves a job by ID.
func (s *MetadataService) GetJob(ctx context.Context, jobID string) (metadata.JobMetadata, error) {
	return s.reader.Get(ctx, jobID)
}

// ListJobs retrieves jobs with filtering.
func (s *MetadataService) ListJobs(ctx context.Context, filter metadata.ListFilter) ([]metadata.JobMetadata, error) {
	return s.reader.List(ctx, filter)
}

// MarkJobDispatched transitions a job from pending_dispatch to dispatched after Pulsar publish.
func (s *MetadataService) MarkJobDispatched(ctx context.Context, jobID string) error {
	now := time.Now()
	matched, err := s.writer.MarkDispatchedIfPending(ctx, jobID, now)
	if err != nil {
		return fmt.Errorf("failed to mark job dispatched: %w", err)
	}
	if !matched {
		return nil
	}
	logEntry := metadata.NewJobLogWithSource(jobID, metadata.LogLevelInfo, "Job dispatched to queue", "service")
	if err := s.writer.AddLog(ctx, logEntry); err != nil {
		log.Printf("warning: failed to log job dispatch: %v", err)
	}
	return nil
}

// RecordDispatchAttempt stores publish retry metadata while the job remains pending_dispatch.
func (s *MetadataService) RecordDispatchAttempt(ctx context.Context, jobID string, attempts int, lastError string) error {
	matched, err := s.writer.RecordDispatchAttemptIfPending(ctx, jobID, attempts, lastError)
	if err != nil {
		return fmt.Errorf("failed to record dispatch attempt: %w", err)
	}
	if !matched {
		return nil
	}
	return nil
}

// MarkJobDispatchFailed transitions a job to dispatch_failed after relay exhausts retries.
func (s *MetadataService) MarkJobDispatchFailed(ctx context.Context, jobID string, dispatchErr error) error {
	errMsg := ""
	if dispatchErr != nil {
		errMsg = dispatchErr.Error()
	}
	matched, err := s.writer.MarkDispatchFailedIfPending(ctx, jobID, errMsg)
	if err != nil {
		return fmt.Errorf("failed to mark job dispatch failed: %w", err)
	}
	if !matched {
		return nil
	}
	return nil
}

// StartJob transitions a dispatched job to running status atomically.
// Returns error only on infrastructure failures (not duplicate delivery).
func (s *MetadataService) StartJob(ctx context.Context, jobID string) error {
	now := time.Now()
	matched, err := s.writer.MarkRunningIfDispatched(ctx, jobID, now)
	if err != nil {
		return fmt.Errorf("failed to mark job running: %w", err)
	}

	// If not matched, job is not in dispatched state (duplicate delivery or already running/terminal)
	if !matched {
		job, getErr := s.reader.Get(ctx, jobID)
		if getErr != nil {
			log.Printf("warning: duplicate delivery for job %s, failed to get current status: %v", jobID, getErr)
		} else {
			log.Printf("warning: duplicate delivery for job %s, current status: %s", jobID, job.GetStatus())
		}
		return nil
	}

	logEntry := metadata.NewJobLogWithSource(
		jobID,
		metadata.LogLevelInfo,
		"Job started",
		"service",
	)
	if err := s.writer.AddLog(ctx, logEntry); err != nil {
		log.Printf("warning: failed to log job start: %v", err)
	}

	return nil
}

// CompleteJob marks a job as completed successfully.
func (s *MetadataService) CompleteJob(ctx context.Context, jobID string, result map[string]any) error {
	job, err := s.reader.Get(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	model, err := metadata.AsJobModel(job)
	if err != nil {
		return err
	}

	if err := model.SetStatus(metadata.JobStatusCompleted); err != nil {
		return fmt.Errorf("invalid status transition: %w", err)
	}

	if result != nil {
		model.SetMetadataField("result", result)
	}

	st := model.Status
	meta := model.Metadata
	var completedAt *time.Time
	if model.CompletedAt != nil {
		t := *model.CompletedAt
		completedAt = &t
	}
	patch := metadata.UpdateJob{Status: &st, CompletedAt: completedAt, Metadata: &meta}
	if err := s.writer.Update(ctx, jobID, patch); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	logEntry := metadata.NewJobLogWithContext(
		jobID,
		metadata.LogLevelInfo,
		"Job completed successfully",
		map[string]any{
			"duration": model.Duration().String(),
		},
	)
	if err := s.writer.AddLog(ctx, logEntry); err != nil {
		log.Printf("warning: failed to log job completion: %v", err)
	}

	return nil
}

// FailJob marks a job as failed with an error message.
func (s *MetadataService) FailJob(ctx context.Context, jobID string, jobErr error) error {
	job, err := s.reader.Get(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	model, err := metadata.AsJobModel(job)
	if err != nil {
		return err
	}

	if err := model.AddError(jobErr); err != nil {
		return fmt.Errorf("failed to add error: %w", err)
	}

	st := model.Status
	errors := model.Errors
	var completedAt *time.Time
	if model.CompletedAt != nil {
		t := *model.CompletedAt
		completedAt = &t
	}
	patch := metadata.UpdateJob{Status: &st, Errors: &errors, CompletedAt: completedAt}
	if err := s.writer.Update(ctx, jobID, patch); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	logEntry := metadata.NewJobLogWithContext(
		jobID,
		metadata.LogLevelError,
		"Job failed",
		map[string]any{
			"error": jobErr.Error(),
		},
	)
	if err := s.writer.AddLog(ctx, logEntry); err != nil {
		log.Printf("warning: failed to log job failure: %v", err)
	}

	return nil
}

// CancelJob cancels a job if it's not already in a terminal state.
func (s *MetadataService) CancelJob(ctx context.Context, jobID string, reason string) error {
	job, err := s.reader.Get(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	if job.GetStatus().IsTerminal() {
		return fmt.Errorf("cannot cancel job in %s state", job.GetStatus())
	}

	model, err := metadata.AsJobModel(job)
	if err != nil {
		return err
	}

	if err := model.SetStatus(metadata.JobStatusCancelled); err != nil {
		return fmt.Errorf("invalid status transition: %w", err)
	}

	if reason != "" {
		model.SetMetadataField("cancellation_reason", reason)
	}

	st := model.Status
	meta := model.Metadata
	var completedAt *time.Time
	if model.CompletedAt != nil {
		t := *model.CompletedAt
		completedAt = &t
	}
	patch := metadata.UpdateJob{Status: &st, CompletedAt: completedAt, Metadata: &meta}
	if err := s.writer.Update(ctx, jobID, patch); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	logEntry := metadata.NewJobLogWithContext(
		jobID,
		metadata.LogLevelWarn,
		"Job cancelled",
		map[string]any{
			"reason": reason,
		},
	)
	if err := s.writer.AddLog(ctx, logEntry); err != nil {
		log.Printf("warning: failed to log job cancellation: %v", err)
	}

	return nil
}

// RetryJob increments retry count and resets job to pending.
func (s *MetadataService) RetryJob(ctx context.Context, jobID string) error {
	job, err := s.reader.Get(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}

	model, err := metadata.AsJobModel(job)
	if err != nil {
		return err
	}

	if model.GetStatus() != metadata.JobStatusFailed {
		return fmt.Errorf("only failed jobs can be retried")
	}

	if err := s.writer.IncrementRetryCount(ctx, jobID); err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}
	pendingDispatch := metadata.JobStatusPendingDispatch
	zeroAttempts := 0
	patch := metadata.UpdateJob{
		Status:           &pendingDispatch,
		DispatchAttempts: &zeroAttempts,
	}
	if err := s.writer.Update(ctx, jobID, patch); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}
	if err := s.writer.ClearJobExecutionTimestamps(ctx, jobID); err != nil {
		return fmt.Errorf("failed to clear job timestamps after retry: %w", err)
	}

	refreshed, err := s.reader.Get(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to reload job after retry: %w", err)
	}

	logEntry := metadata.NewJobLogWithContext(
		jobID,
		metadata.LogLevelInfo,
		"Job retry initiated",
		map[string]any{
			"retryCount": refreshed.GetRetryCount(),
		},
	)
	if err := s.writer.AddLog(ctx, logEntry); err != nil {
		log.Printf("warning: failed to log job retry: %v", err)
	}

	return nil
}

// AddJobLog adds a log entry for a job.
func (s *MetadataService) AddJobLog(ctx context.Context, jobID string, level metadata.LogLevel, message string, logCtx map[string]any) error {
	logEntry := metadata.NewJobLogWithContext(jobID, level, message, logCtx)
	logEntry.Source = "api"

	return s.writer.AddLog(ctx, logEntry)
}

// GetJobLogs retrieves logs for a job.
func (s *MetadataService) GetJobLogs(ctx context.Context, jobID string, filter metadata.LogFilter) ([]metadata.JobLog, error) {
	return s.reader.GetLogs(ctx, jobID, filter)
}

// GetJobStats returns statistics about jobs.
func (s *MetadataService) GetJobStats(ctx context.Context) (JobStats, error) {
	var stats JobStats

	for _, status := range []metadata.JobStatus{
		metadata.JobStatusPendingDispatch,
		metadata.JobStatusDispatched,
		metadata.JobStatusDispatchFailed,
		metadata.JobStatusRunning,
		metadata.JobStatusCompleted,
		metadata.JobStatusFailed,
		metadata.JobStatusCancelled,
	} {
		filter := metadata.ListFilter{
			Statuses: []metadata.JobStatus{status},
		}
		n, err := s.reader.CountJobs(ctx, filter)
		if err != nil {
			return stats, fmt.Errorf("failed to count %s jobs: %w", status, err)
		}
		count := int(n)

		switch status {
		case metadata.JobStatusPendingDispatch:
			stats.PendingDispatch = count
		case metadata.JobStatusDispatched:
			stats.Dispatched = count
		case metadata.JobStatusDispatchFailed:
			stats.DispatchFailed = count
		case metadata.JobStatusRunning:
			stats.Running = count
		case metadata.JobStatusCompleted:
			stats.Completed = count
		case metadata.JobStatusFailed:
			stats.Failed = count
		case metadata.JobStatusCancelled:
			stats.Cancelled = count
		}
	}

	stats.Total = stats.PendingDispatch + stats.Dispatched + stats.DispatchFailed +
		stats.Running + stats.Completed + stats.Failed + stats.Cancelled

	return stats, nil
}

// JobStats holds job statistics.
type JobStats struct {
	Total           int `json:"total"`
	PendingDispatch int `json:"pending_dispatch"`
	Dispatched      int `json:"dispatched"`
	DispatchFailed  int `json:"dispatch_failed"`
	Running         int `json:"running"`
	Completed       int `json:"completed"`
	Failed          int `json:"failed"`
	Cancelled       int `json:"cancelled"`
}
