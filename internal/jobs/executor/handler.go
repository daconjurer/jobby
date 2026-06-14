package executor

import (
	"context"
)

// JobExecutor runs business logic for a typed payload.
// Handlers implement this interface for their specific job type.
type JobExecutor[T any] interface {
	Execute(ctx context.Context, jobID string, data T) error
}

// JobLifecycle models the valid operations on a Job once it has been dispatched
type JobLifecycle interface {
	// StartJob transitions a job from dispatched to running atomically.
	// Returns error only on infrastructure failures (not duplicate delivery).
	StartJob(ctx context.Context, jobID string) error

	// CompleteJob marks a job as completed with optional result data.
	CompleteJob(ctx context.Context, jobID string, result map[string]any) error

	// FailJob marks a job as failed with an error message.
	FailJob(ctx context.Context, jobID string, err error) error
}
