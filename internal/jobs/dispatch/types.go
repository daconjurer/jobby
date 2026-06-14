package dispatch

import (
	"context"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

// PendingJobFetcher loads pending_dispatch jobs for poll fallback.
type PendingJobFetcher interface {
	FetchPending(ctx context.Context, maxAttempts int, limit int) ([]JobDispatchProjection, error)
}

// StreamRunner watches for new pending_dispatch jobs (e.g. MongoDB change stream).
type StreamRunner interface {
	Run(ctx context.Context)
}

// JobPublisher sends a pending_dispatch job to its target topic (saga phase 2).
// Implementations own wire format and broker details.
type JobDispatchPublisher interface {
	Publish(ctx context.Context, job JobDispatchProjection) error
}

// JobDispatchHandler runs saga phases 2–3 for a single job dispatch projection.
// Implementations must follow docs/architecture/job-saga.md (publish, then confirm status).
type JobDispatchHandler interface {
	HandleDispatch(ctx context.Context, jobProjection JobDispatchProjection) error
}

// JobUpdater persists dispatch saga outcomes on job_metadata.
type JobUpdater interface {
	MarkJobDispatched(ctx context.Context, jobID string) error
	MarkJobDispatchFailed(ctx context.Context, jobID string, dispatchErr error) error
	RecordDispatchAttempt(ctx context.Context, jobID string, attempts int, lastError string) error
}

// JobDispatchProjection holds the fields needed to publish a pending_dispatch row to Pulsar.
type JobDispatchProjection struct {
	JobID            string
	Name             string
	Topic            string
	Payload          map[string]any
	DispatchAttempts int
}

// JobDispatchFromMetadata maps a metadata model to a dispatch job projection.
func JobDispatchFromMetadata(m *metadata.JobMetadataModel) JobDispatchProjection {
	payload := m.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	return JobDispatchProjection{
		JobID:            m.JobID,
		Name:             m.Name,
		Topic:            m.Topic,
		Payload:          payload,
		DispatchAttempts: m.DispatchAttempts,
	}
}
