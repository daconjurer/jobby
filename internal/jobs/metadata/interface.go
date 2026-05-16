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

	// GetStatus returns the current job execution status
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
	GetPayload() interface{}

	// GetMetadata returns additional metadata fields as key-value pairs
	GetMetadata() map[string]interface{}

	// GetError returns the error message if job failed (empty string if no error)
	GetError() string

	// GetRetryCount returns the number of retry attempts
	GetRetryCount() int

	// GetTags returns job tags for filtering and categorization
	GetTags() []string

	// Validate checks if the job metadata is valid according to business rules
	Validate() error
}

// JobStatus represents the execution state of a job
type JobStatus string

const (
	// JobStatusPending indicates job is queued but not yet started
	JobStatusPending JobStatus = "pending"

	// JobStatusRunning indicates job is currently executing
	JobStatusRunning JobStatus = "running"

	// JobStatusCompleted indicates job finished successfully
	JobStatusCompleted JobStatus = "completed"

	// JobStatusFailed indicates job encountered an error and failed
	JobStatusFailed JobStatus = "failed"

	// JobStatusCancelled indicates job was cancelled by user or system
	JobStatusCancelled JobStatus = "cancelled"
)

// String returns the string representation of JobStatus
func (s JobStatus) String() string {
	return string(s)
}

// IsValid checks if the status is a valid JobStatus value
func (s JobStatus) IsValid() bool {
	switch s {
	case JobStatusPending, JobStatusRunning, JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if the job status is terminal (completed, failed, or cancelled)
// Terminal states indicate the job will not transition to any other state.
func (s JobStatus) IsTerminal() bool {
	return s == JobStatusCompleted || s == JobStatusFailed || s == JobStatusCancelled
}

// CanTransitionTo checks if transitioning from current status to target status is valid
func (s JobStatus) CanTransitionTo(target JobStatus) bool {
	if s.IsTerminal() {
		return false
	}

	switch s {
	case JobStatusPending:
		return target == JobStatusRunning || target == JobStatusCancelled || target == JobStatusFailed
	case JobStatusRunning:
		return target == JobStatusCompleted || target == JobStatusFailed || target == JobStatusCancelled
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
	Status      *JobStatus      `bson:"status,omitempty"`
	Name        *string         `bson:"name,omitempty"`
	Priority    *int            `bson:"priority,omitempty"`
	StartedAt   *time.Time      `bson:"startedAt,omitempty"`
	CompletedAt *time.Time      `bson:"completedAt,omitempty"`
	Payload     *map[string]any `bson:"payload,omitempty"`
	Metadata    *map[string]any `bson:"metadata,omitempty"`
	Error       *string         `bson:"error,omitempty"`
	Tags        *[]string       `bson:"tags,omitempty"`
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
