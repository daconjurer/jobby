package dispatch

import (
	"context"
	"fmt"
)

// DispatchHandlerConfig controls dispatch retry behavior.
type DispatchHandlerConfig struct {
	MaxAttempts int
}

// DispatchHandler runs the publish and confirm steps of the dispatch saga for a single job.
type DispatchHandler struct {
	cfg       DispatchHandlerConfig
	publisher JobDispatchPublisher
	jobs      JobUpdater
}

// NewDispatchHandler creates a dispatch saga handler.
func NewDispatchHandler(cfg DispatchHandlerConfig, publisher JobDispatchPublisher, jobs JobUpdater) *DispatchHandler {
	return &DispatchHandler{cfg: cfg, publisher: publisher, jobs: jobs}
}

// HandleDispatch runs the publish and confirm steps per docs/architecture/job-saga.md:
// publish via JobPublisher, then MarkJobDispatched, RecordDispatchAttempt, or MarkJobDispatchFailed.
// Idempotent when the job is already dispatched.
func (h *DispatchHandler) HandleDispatch(ctx context.Context, job JobDispatchProjection) error {
	if job.Topic == "" {
		return fmt.Errorf("job %s missing topic", job.JobID)
	}

	publishErr := h.publisher.Publish(ctx, job)
	if publishErr == nil {
		return h.jobs.MarkJobDispatched(ctx, job.JobID)
	}

	attempts := job.DispatchAttempts + 1
	if err := h.jobs.RecordDispatchAttempt(ctx, job.JobID, attempts, publishErr.Error()); err != nil {
		return err
	}
	if attempts >= h.cfg.MaxAttempts {
		return h.jobs.MarkJobDispatchFailed(ctx, job.JobID, publishErr)
	}
	return nil
}
