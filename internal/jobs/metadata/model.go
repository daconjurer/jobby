package metadata

import (
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// JobMetadataModel is the concrete implementation of the JobMetadata interface.
// This struct is used for MongoDB persistence with BSON tags for proper serialization.
//
// BSON Tags:
// - `bson:"fieldName"` - MongoDB field name
// - `omitempty` - Omit field if zero value
// - `json:"fieldName"` - JSON field name for API responses
type JobMetadataModel struct {
	ID          bson.ObjectID  `bson:"_id,omitempty" json:"id,omitempty"`
	JobID       string         `bson:"jobId" json:"jobId"`
	Name        string         `bson:"name" json:"name"`
	Status      JobStatus      `bson:"status" json:"status"`
	Priority    int            `bson:"priority" json:"priority"`
	CreatedAt   time.Time      `bson:"createdAt" json:"createdAt"`
	StartedAt   *time.Time     `bson:"startedAt,omitempty" json:"startedAt,omitempty"`
	CompletedAt *time.Time     `bson:"completedAt,omitempty" json:"completedAt,omitempty"`
	Payload     map[string]any `bson:"payload" json:"payload"`
	Metadata    map[string]any `bson:"metadata" json:"metadata"`
	Error       string         `bson:"error,omitempty" json:"error,omitempty"`
	RetryCount  int            `bson:"retryCount" json:"retryCount"`
	Tags        []string       `bson:"tags" json:"tags"`

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
func (j *JobMetadataModel) GetStatus() JobStatus        { return j.Status }
func (j *JobMetadataModel) GetPriority() int            { return j.Priority }
func (j *JobMetadataModel) GetCreatedAt() time.Time     { return j.CreatedAt }
func (j *JobMetadataModel) GetStartedAt() *time.Time    { return j.StartedAt }
func (j *JobMetadataModel) GetCompletedAt() *time.Time  { return j.CompletedAt }
func (j *JobMetadataModel) GetPayload() any             { return j.Payload }
func (j *JobMetadataModel) GetMetadata() map[string]any { return j.Metadata }
func (j *JobMetadataModel) GetError() string            { return j.Error }
func (j *JobMetadataModel) GetRetryCount() int          { return j.RetryCount }
func (j *JobMetadataModel) GetTags() []string           { return j.Tags }

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

	if j.Status.IsDispatchPhase() {
		if j.StartedAt != nil || j.CompletedAt != nil {
			return errors.New("dispatch phase job must not have startedAt or completedAt")
		}
	}

	if j.Status == JobStatusPendingDispatch && j.Topic == "" {
		return errors.New("pending_dispatch job must have topic")
	}

	if j.DispatchAttempts < 0 {
		return errors.New("dispatchAttempts cannot be negative")
	}

	if j.Status == JobStatusRunning && j.StartedAt == nil {
		return errors.New("running job must have startedAt timestamp")
	}

	if j.Status == JobStatusCompleted || j.Status == JobStatusCancelled {
		if j.StartedAt == nil {
			return errors.New("completed or cancelled job must have startedAt timestamp")
		}
		if j.CompletedAt == nil {
			return errors.New("completed or cancelled job must have completedAt timestamp")
		}
	}

	if j.Status == JobStatusFailed {
		if j.CompletedAt == nil {
			return errors.New("failed job must have completedAt timestamp")
		}
		if j.Error == "" {
			return errors.New("failed job must have error message")
		}
	}

	return nil
}

// SetStatus updates the job status and related timestamps automatically
func (j *JobMetadataModel) SetStatus(status JobStatus) error {
	if !j.Status.CanTransitionTo(status) {
		return fmt.Errorf("cannot transition from %s to %s", j.Status, status)
	}

	j.Status = status
	now := time.Now()

	switch status {
	case JobStatusRunning:
		if j.StartedAt == nil {
			j.StartedAt = &now
		}
	case JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		if j.StartedAt == nil {
			j.StartedAt = &now
		}
		if j.CompletedAt == nil {
			j.CompletedAt = &now
		}
	case JobStatusPendingDispatch, JobStatusDispatched, JobStatusDispatchFailed:
		// dispatch phase: no execution timestamps
	}

	return nil
}

// SetError sets the error message and transitions to failed status
func (j *JobMetadataModel) SetError(err error) error {
	if err != nil {
		j.Error = err.Error()
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
