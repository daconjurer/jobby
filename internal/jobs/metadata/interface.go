package metadata

import (
	"context"
	"time"
)

// JobMetadata is the interface that ALL job types must implement.
type JobMetadata interface {
	// GetJobID returns the unique job identifier (UUID format)
	GetJobID() string

	// GetName returns the job type name (e.g., "process-kml")
	GetName() string

	// GetStatus returns the current job lifecycle status (dispatch + execution).
	GetStatus() JobStatus

	// GetPriority returns the job priority (0-10, where 10 is highest)
	GetPriority() int

	// GetCreatedAt returns the job creation timestamp
	GetCreatedAt() time.Time

	// GetStartedAt returns the job start timestamp (nil if not started)
	GetStartedAt() *time.Time

	// GetCompletedAt returns the job completion timestamp (nil if not completed)
	GetCompletedAt() *time.Time

	// GetPayload returns the job-specific data (dynamic schema per job type)
	GetPayload() any

	// GetMetadata returns additional metadata fields as key-value pairs
	GetMetadata() map[string]any

	// GetErrors returns the complete error history across all retry attempts
	GetErrors() []JobError

	// GetLatestError returns the most recent error message (convenience method)
	GetLatestError() string

	// GetRetryCount returns the number of retry attempts
	GetRetryCount() int

	// GetTags returns job tags for filtering and categorization
	GetTags() []string

	// Validate checks if the job metadata is valid according to business rules
	Validate() error
}

// JobStatus represents the full job lifecycle: dispatch (Mongo → Pulsar) then execution.
type JobStatus string

const (
	// Dispatch phase
	JobStatusPendingDispatch JobStatus = "pending_dispatch"
	JobStatusDispatched      JobStatus = "dispatched"
	JobStatusDispatchFailed  JobStatus = "dispatch_failed"

	// Execution phase
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// String returns the string representation of JobStatus
func (s JobStatus) String() string {
	return string(s)
}

// IsValid checks if the status is a valid JobStatus value
func (s JobStatus) IsValid() bool {
	switch s {
	case JobStatusPendingDispatch, JobStatusDispatched, JobStatusDispatchFailed,
		JobStatusRunning, JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if the job status is terminal (completed, failed, or cancelled).
// dispatch_failed is recoverable and not terminal.
func (s JobStatus) IsTerminal() bool {
	return s == JobStatusCompleted || s == JobStatusFailed || s == JobStatusCancelled
}

// IsDispatchPhase returns true while the job may not yet be consumed or dispatch can be retried.
func (s JobStatus) IsDispatchPhase() bool {
	return s == JobStatusPendingDispatch || s == JobStatusDispatched || s == JobStatusDispatchFailed
}

// IsRunning returns true if the job status is 'running' (mostly for consistency).
func (s JobStatus) IsRunning() bool {
	return s == JobStatusRunning
}

// CanTransitionTo checks if transitioning from current status to target status is valid
func (s JobStatus) CanTransitionTo(target JobStatus) bool {
	if !target.IsValid() {
		return false
	}
	if s == target {
		return false
	}
	if s == JobStatusFailed {
		return target == JobStatusPendingDispatch
	}
	if s.IsTerminal() {
		return false
	}

	switch s {
	case JobStatusPendingDispatch:
		return target == JobStatusDispatched ||
			target == JobStatusDispatchFailed ||
			target == JobStatusCancelled

	case JobStatusDispatchFailed:
		return target == JobStatusPendingDispatch

	case JobStatusDispatched:
		return target == JobStatusRunning ||
			target == JobStatusCancelled ||
			target == JobStatusFailed

	case JobStatusRunning:
		return target == JobStatusCompleted ||
			target == JobStatusFailed ||
			target == JobStatusCancelled

	default:
		return false
	}
}

// JobsReader defines read-only operations for job metadata and logs.
type JobsReader interface {
	Get(ctx context.Context, jobID string) (JobMetadata, error)
	List(ctx context.Context, filter ListFilter) ([]JobMetadata, error)
	GetLogs(ctx context.Context, jobID string, filter LogFilter) ([]JobLog, error)
	CountJobs(ctx context.Context, filter ListFilter) (int64, error)
	GetJobsByStatus(ctx context.Context, status JobStatus, limit int) ([]JobMetadata, error)
	GetPendingJobs(ctx context.Context, limit int) ([]JobMetadata, error)
	GetDispatchedJobs(ctx context.Context, limit int) ([]JobMetadata, error)
	GetRecentLogs(ctx context.Context, jobID string, limit int) ([]JobLog, error)
	GetErrorLogs(ctx context.Context, jobID string) ([]JobLog, error)
}

// JobsWriter defines write operations for job metadata and logs.
// Methods take context.Context.
type JobsWriter interface {
	Create(ctx context.Context, job JobMetadata) error
	Update(ctx context.Context, jobID string, patch UpdateJob) error
	Delete(ctx context.Context, jobID string) error
	IncrementRetryCount(ctx context.Context, jobID string) error
	// ClearJobExecutionTimestamps removes startedAt and completedAt from job_metadata (e.g. after retry).
	ClearJobExecutionTimestamps(ctx context.Context, jobID string) error
	// MarkDispatchedIfPending sets status to dispatched when still pending_dispatch.
	MarkDispatchedIfPending(ctx context.Context, jobID string, dispatchedAt time.Time) (bool, error)
	// RecordDispatchAttemptIfPending updates dispatch retry fields while pending_dispatch.
	RecordDispatchAttemptIfPending(ctx context.Context, jobID string, attempts int, lastError string) (bool, error)
	// MarkDispatchFailedIfPending transitions pending_dispatch → dispatch_failed.
	MarkDispatchFailedIfPending(ctx context.Context, jobID string, errorMsg string) (bool, error)
	// MarkRunningIfDispatched transitions dispatched → running atomically.
	// Returns (true, nil) on success, (false, nil) if job not dispatched (idempotent duplicate).
	MarkRunningIfDispatched(ctx context.Context, jobID string, startedAt time.Time) (bool, error)
	// CompleteIfRunning sets status=completed when jobId matches and status=running.
	// Returns (true, nil) on match, (false, nil) if not running.
	CompleteIfRunning(ctx context.Context, jobID string, completedAt time.Time, metadata *map[string]any) (bool, error)
	// FailIfNotTerminal appends execution error and sets status=failed when status is running or dispatched.
	// Returns (true, nil) on match, (false, nil) if already terminal or not fail-able.
	FailIfNotTerminal(ctx context.Context, jobID string, jobErr JobError, completedAt time.Time) (bool, error)
	AddLog(ctx context.Context, log JobLog) error
	DeleteOldLogs(ctx context.Context, olderThan time.Duration) (int64, error)
}

// ListFilter defines filtering options for listing jobs
type ListFilter struct {
	// Names filters jobs by job type names (OR condition)
	Names []string

	// Statuses filters jobs by execution statuses (OR condition)
	Statuses []JobStatus

	// Tags filters jobs that have ANY of these tags (OR condition)
	Tags []string

	// MinPriority filters jobs with priority >= this value
	MinPriority *int

	// MaxPriority filters jobs with priority <= this value
	MaxPriority *int

	// CreatedAfter filters jobs created after this timestamp
	CreatedAfter *time.Time

	// CreatedBefore filters jobs created before this timestamp
	CreatedBefore *time.Time

	// Limit sets maximum number of results (0 = no limit)
	Limit int

	// Skip sets number of results to skip (for pagination)
	Skip int

	// SortBy sets the field to sort by (default: "createdAt")
	SortBy string

	// SortDesc sets whether to sort descending (default: true)
	SortDesc bool
}

// UpdateJob selects fields to set on job_metadata by job ID. Nil pointers omit that field from the update.
// bson tags mirror JobMetadataModel (see bsonPartialSet). Use IncrementRetryCount for atomic retry bumps.
type UpdateJob struct {
	Status            *JobStatus      `bson:"status,omitempty"`
	Name              *string         `bson:"name,omitempty"`
	Priority          *int            `bson:"priority,omitempty"`
	StartedAt         *time.Time      `bson:"startedAt,omitempty"`
	CompletedAt       *time.Time      `bson:"completedAt,omitempty"`
	Payload           *map[string]any `bson:"payload,omitempty"`
	Metadata          *map[string]any `bson:"metadata,omitempty"`
	Errors            *[]JobError     `bson:"errors,omitempty"`
	Tags              *[]string       `bson:"tags,omitempty"`
	Topic             *string         `bson:"topic,omitempty"`
	DispatchAttempts  *int            `bson:"dispatchAttempts,omitempty"`
	DispatchLastError *string         `bson:"dispatchLastError,omitempty"`
	DispatchedAt      *time.Time      `bson:"dispatchedAt,omitempty"`
}

// JobLog represents a log entry for job execution
type JobLog struct {
	JobID      string         `bson:"jobId" json:"jobId"`
	Timestamp  time.Time      `bson:"timestamp" json:"timestamp"`
	Level      LogLevel       `bson:"level" json:"level"`
	Message    string         `bson:"message" json:"message"`
	Context    map[string]any `bson:"context,omitempty" json:"context,omitempty"`
	Source     string         `bson:"source,omitempty" json:"source,omitempty"`
	StackTrace string         `bson:"stackTrace,omitempty" json:"stackTrace,omitempty"`
}

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// IsValid checks if the log level is valid
func (l LogLevel) IsValid() bool {
	switch l {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError, LogLevelFatal:
		return true
	default:
		return false
	}
}

// LogFilter defines filtering options for retrieving logs
type LogFilter struct {
	// Levels filters logs by severity levels (OR condition)
	Levels []LogLevel

	// Since filters logs created after this timestamp
	Since *time.Time

	// Until filters logs created before this timestamp
	Until *time.Time

	// Limit sets maximum number of results (0 = no limit)
	Limit int

	// Skip sets number of results to skip (for pagination)
	Skip int
}
