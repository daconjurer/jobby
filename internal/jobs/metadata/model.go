package metadata

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// JobErrorType distinguishes execution failures from dispatch-phase failures.
type JobErrorType string

const (
	JobErrorTypeExecution JobErrorType = "execution"
	JobErrorTypeDispatch  JobErrorType = "dispatch"
)

func (t JobErrorType) IsValid() bool {
	switch t {
	case JobErrorTypeExecution, JobErrorTypeDispatch:
		return true
	default:
		return false
	}
}

// JobError represents a single error occurrence during job execution or dispatch.
// Multiple JobError entries track the error history across retry attempts.
type JobError struct {
	Type         JobErrorType `bson:"type" json:"type"`
	RetryAttempt int          `bson:"retryAttempt" json:"retryAttempt"`
	Error        string       `bson:"error" json:"error"`
	Timestamp    time.Time    `bson:"timestamp" json:"timestamp"`
}

// JobMetadataModel is the concrete implementation of the JobMetadata interface.
// This struct is used for MongoDB persistence with BSON tags for proper serialization.
//
// BSON Tags:
// - `bson:"fieldName"` - MongoDB field name
// - `omitempty` - Omit field if zero value
// - `json:"fieldName"` - JSON field name for API responses
type JobMetadataModel struct {
	ID              bson.ObjectID   `bson:"_id,omitempty" json:"id,omitempty"`
	JobID           string          `bson:"jobId" json:"jobId"`
	Name            string          `bson:"name" json:"name"`
	Status          JobStatus       `bson:"status,omitempty" json:"status,omitempty"`
	DispatchStatus  DispatchStatus  `bson:"dispatchStatus" json:"dispatchStatus"`
	ExecutionStatus ExecutionStatus `bson:"executionStatus" json:"executionStatus"`
	Priority        int             `bson:"priority" json:"priority"`
	CreatedAt       time.Time       `bson:"createdAt" json:"createdAt"`
	StartedAt       *time.Time      `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	CompletedAt     *time.Time      `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	Payload         map[string]any  `bson:"payload" json:"payload"`
	Metadata        map[string]any  `bson:"metadata" json:"metadata"`
	Errors          []JobError      `bson:"errors,omitempty" json:"errors,omitempty"`
	RetryCount      int             `bson:"retryCount" json:"retryCount"`
	Tags            []string        `bson:"tags" json:"tags"`

	// Dispatch phase (embedded on job_metadata; set at enqueue)
	Topic             string     `bson:"topic,omitempty" json:"topic,omitempty"`
	DispatchAttempts  int        `bson:"dispatchAttempts" json:"dispatchAttempts"`
	DispatchLastError string     `bson:"dispatchLastError,omitempty" json:"dispatchLastError,omitempty"`
	DispatchedAt      *time.Time `bson:"dispatchedAt,omitempty" json:"dispatchedAt,omitempty"`
}

var _ JobMetadata = (*JobMetadataModel)(nil)

// NewJobMetadata creates a new job metadata with sensible defaults
func NewJobMetadata(jobID, name string, payload map[string]any) *JobMetadataModel {
	now := time.Now()

	if payload == nil {
		payload = make(map[string]any)
	}

	return &JobMetadataModel{
		JobID:            jobID,
		Name:             name,
		Status:           JobStatusPendingDispatch,
		DispatchStatus:   DispatchStatusPending,
		ExecutionStatus:  ExecutionStatusNotStarted,
		Priority:         5,
		CreatedAt:        now,
		Payload:          payload,
		Metadata:         make(map[string]any),
		RetryCount:       0,
		Tags:             []string{},
		DispatchAttempts: 0,
	}
}

func (j *JobMetadataModel) GetJobID() string            { return j.JobID }
func (j *JobMetadataModel) GetName() string             { return j.Name }
func (j *JobMetadataModel) GetStatus() JobStatus        { return j.DisplayStatus() }
func (j *JobMetadataModel) GetPriority() int            { return j.Priority }
func (j *JobMetadataModel) GetCreatedAt() time.Time     { return j.CreatedAt }
func (j *JobMetadataModel) GetStartedAt() *time.Time    { return j.StartedAt }
func (j *JobMetadataModel) GetCompletedAt() *time.Time  { return j.CompletedAt }
func (j *JobMetadataModel) GetPayload() any             { return j.Payload }
func (j *JobMetadataModel) GetMetadata() map[string]any { return j.Metadata }
func (j *JobMetadataModel) GetErrors() []JobError       { return j.Errors }
func (j *JobMetadataModel) GetRetryCount() int          { return j.RetryCount }
func (j *JobMetadataModel) GetTags() []string           { return j.Tags }

// GetLatestError returns the most recent error message, or empty string if no errors.
// This is a convenience method for backward compatibility.
func (j *JobMetadataModel) GetLatestError() string {
	if len(j.Errors) == 0 {
		return ""
	}
	return j.Errors[len(j.Errors)-1].Error
}

// DisplayStatus computes the external seven-value status from the two sub-states.
func (j *JobMetadataModel) DisplayStatus() JobStatus {
	return CompositeStatus{Dispatch: j.DispatchStatus, Execution: j.ExecutionStatus}.DisplayStatus()
}

// Validate checks if the job metadata is valid according to business rules
func (j *JobMetadataModel) Validate() error {
	if j.JobID == "" {
		return errors.New("jobId is required")
	}

	if len(j.JobID) != 36 {
		return errors.New("jobId must be a valid UUID (36 characters)")
	}

	if j.Name == "" {
		return errors.New("name is required")
	}

	if len(j.Name) > 100 {
		return errors.New("name must not exceed 100 characters")
	}

	if !j.DispatchStatus.IsValid() {
		return fmt.Errorf("invalid dispatchStatus value: %s", j.DispatchStatus)
	}

	if !j.ExecutionStatus.IsValid() {
		return fmt.Errorf("invalid executionStatus value: %s", j.ExecutionStatus)
	}

	if !j.Status.IsValid() {
		return fmt.Errorf("invalid status value: %s", j.Status)
	}

	if j.Priority < 0 || j.Priority > 10 {
		return errors.New("priority must be between 0 and 10")
	}

	if j.CreatedAt.IsZero() {
		return errors.New("createdAt is required")
	}

	if j.RetryCount < 0 {
		return errors.New("retryCount cannot be negative")
	}

	if j.DisplayStatus().IsDispatchPhase() {
		if j.StartedAt != nil || j.CompletedAt != nil {
			return errors.New("dispatch phase job must not have startedAt or completedAt")
		}
	}

	if j.DispatchStatus == DispatchStatusPending && j.Topic == "" {
		return errors.New("pending_dispatch job must have topic")
	}

	if j.DispatchAttempts < 0 {
		return errors.New("dispatchAttempts cannot be negative")
	}

	if j.ExecutionStatus == ExecutionStatusRunning && j.StartedAt == nil {
		return errors.New("running job must have startedAt timestamp")
	}

	if j.ExecutionStatus == ExecutionStatusCompleted || j.ExecutionStatus == ExecutionStatusCancelled {
		if j.StartedAt == nil {
			return errors.New("completed or cancelled job must have startedAt timestamp")
		}
		if j.CompletedAt == nil {
			return errors.New("completed or cancelled job must have completedAt timestamp")
		}
	}

	if j.ExecutionStatus == ExecutionStatusFailed {
		if j.CompletedAt == nil {
			return errors.New("failed job must have completedAt timestamp")
		}
		if len(j.Errors) == 0 {
			return errors.New("failed job must have error message")
		}
	}

	return nil
}

// SetStatus updates the job status and related timestamps automatically.
// Routes a single JobStatus value to the correct sub-field (DispatchStatus or ExecutionStatus)
// and sets the appropriate timestamps.
func (j *JobMetadataModel) SetStatus(status JobStatus) error {
	current := j.DisplayStatus()
	if !current.CanTransitionTo(status) {
		return fmt.Errorf("cannot transition from %s to %s", current, status)
	}

	j.Status = status
	now := time.Now()

	switch status {
	case JobStatusPendingDispatch:
		j.DispatchStatus = DispatchStatusPending
		if j.ExecutionStatus.IsTerminal() {
			j.ExecutionStatus = ExecutionStatusNotStarted
		}
	case JobStatusDispatched:
		j.DispatchStatus = DispatchStatusDispatched
		if j.DispatchedAt == nil {
			j.DispatchedAt = &now
		}
	case JobStatusDispatchFailed:
		j.DispatchStatus = DispatchStatusFailed
	case JobStatusRunning:
		j.ExecutionStatus = ExecutionStatusRunning
		if j.StartedAt == nil {
			j.StartedAt = &now
		}
	case JobStatusCompleted:
		j.ExecutionStatus = ExecutionStatusCompleted
		if j.StartedAt == nil {
			j.StartedAt = &now
		}
		if j.CompletedAt == nil {
			j.CompletedAt = &now
		}
	case JobStatusFailed:
		j.ExecutionStatus = ExecutionStatusFailed
		if j.StartedAt == nil {
			j.StartedAt = &now
		}
		if j.CompletedAt == nil {
			j.CompletedAt = &now
		}
	case JobStatusCancelled:
		j.ExecutionStatus = ExecutionStatusCancelled
		if j.StartedAt == nil {
			j.StartedAt = &now
		}
		if j.CompletedAt == nil {
			j.CompletedAt = &now
		}
	}

	return nil
}

// AddError appends an error to the errors history and transitions to failed status.
// It records the retry attempt number and timestamp for audit trails.
func (j *JobMetadataModel) AddError(err error) error {
	if err != nil {
		jobErr := JobError{
			Type:         JobErrorTypeExecution,
			RetryAttempt: j.RetryCount,
			Error:        err.Error(),
			Timestamp:    time.Now().UTC(),
		}
		j.Errors = append(j.Errors, jobErr)
		return j.SetStatus(JobStatusFailed)
	}
	return nil
}

// AddTag adds a tag to the job if it doesn't already exist
func (j *JobMetadataModel) AddTag(tag string) {
	if tag == "" {
		return
	}

	for _, t := range j.Tags {
		if t == tag {
			return
		}
	}

	j.Tags = append(j.Tags, tag)
}

// RemoveTag removes a tag from the job
func (j *JobMetadataModel) RemoveTag(tag string) {
	for i, t := range j.Tags {
		if t == tag {
			j.Tags = append(j.Tags[:i], j.Tags[i+1:]...)
			return
		}
	}
}

// SetMetadataField sets a metadata field (overwrites if exists)
func (j *JobMetadataModel) SetMetadataField(key string, value any) {
	if j.Metadata == nil {
		j.Metadata = make(map[string]any)
	}
	j.Metadata[key] = value
}

// GetMetadataField retrieves a metadata field by key
func (j *JobMetadataModel) GetMetadataField(key string) (any, bool) {
	if j.Metadata == nil {
		return nil, false
	}
	value, exists := j.Metadata[key]
	return value, exists
}

// IncrementRetryCount increments the retry counter
func (j *JobMetadataModel) IncrementRetryCount() {
	j.RetryCount++
}

// Duration calculates the job execution duration (startedAt to completedAt)
// Returns zero duration if job hasn't started or completed
func (j *JobMetadataModel) Duration() time.Duration {
	if j.StartedAt == nil || j.CompletedAt == nil {
		return 0
	}
	return j.CompletedAt.Sub(*j.StartedAt)
}

// Age returns how long ago the job was created
func (j *JobMetadataModel) Age() time.Duration {
	return time.Since(j.CreatedAt)
}

// MarshalJSON includes the computed displayStatus alongside the persisted fields.
func (j *JobMetadataModel) MarshalJSON() ([]byte, error) {
	type alias JobMetadataModel
	return json.Marshal(&struct {
		DisplayStatus JobStatus `json:"displayStatus"`
		*alias
	}{
		DisplayStatus: j.DisplayStatus(),
		alias:         (*alias)(j),
	})
}

// AsJobModel returns the concrete model behind a JobMetadata value.
func AsJobModel(job JobMetadata) (*JobMetadataModel, error) {
	model, ok := job.(*JobMetadataModel)
	if !ok {
		return nil, fmt.Errorf("unexpected job metadata type %T", job)
	}
	return model, nil
}
