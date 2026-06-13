package metadata

import (
	"errors"
	"testing"
	"time"
)

// TestNewJobMetadata tests the constructor function
func TestNewJobMetadata(t *testing.T) {
	t.Run("creates job with default values", func(t *testing.T) {
		jobID := "123e4567-e89b-12d3-a456-426614174000"
		name := "test-job"
		payload := map[string]any{"key": "value"}

		job := NewJobMetadata(jobID, name, payload)

		if job.JobID != jobID {
			t.Errorf("expected jobID %s, got %s", jobID, job.JobID)
		}
		if job.Name != name {
			t.Errorf("expected name %s, got %s", name, job.Name)
		}
		if job.Status != JobStatusPendingDispatch {
			t.Errorf("expected status %s, got %s", JobStatusPendingDispatch, job.Status)
		}
		if job.Priority != 5 {
			t.Errorf("expected priority 5, got %d", job.Priority)
		}
		if job.CreatedAt.IsZero() {
			t.Error("expected createdAt to be set")
		}
		if job.RetryCount != 0 {
			t.Errorf("expected retryCount 0, got %d", job.RetryCount)
		}
		if len(job.Tags) != 0 {
			t.Errorf("expected empty tags, got %v", job.Tags)
		}
		if job.Metadata == nil {
			t.Error("expected metadata to be initialized")
		}
	})

	t.Run("handles nil payload", func(t *testing.T) {
		job := NewJobMetadata("test-id", "test-job", nil)

		if job.Payload == nil {
			t.Error("expected payload to be initialized as empty map")
		}
		if len(job.Payload) != 0 {
			t.Errorf("expected empty payload, got %v", job.Payload)
		}
	})
}

// TestJobMetadataModel_Getters tests all getter methods
func TestJobMetadataModel_Getters(t *testing.T) {
	now := time.Now()
	startedAt := now.Add(1 * time.Hour)
	completedAt := now.Add(2 * time.Hour)

	job := &JobMetadataModel{
		JobID:       "test-id",
		Name:        "test-name",
		Status:      JobStatusRunning,
		Priority:    7,
		CreatedAt:   now,
		StartedAt:   &startedAt,
		CompletedAt: &completedAt,
		Payload:     map[string]any{"key": "value"},
		Metadata:    map[string]any{"meta": "data"},
		Errors:      []JobError{{RetryAttempt: 0, Error: "test error", Timestamp: now}},
		RetryCount:  3,
		Tags:        []string{"tag1", "tag2"},
	}

	if job.GetJobID() != "test-id" {
		t.Errorf("GetJobID() = %s, want test-id", job.GetJobID())
	}
	if job.GetName() != "test-name" {
		t.Errorf("GetName() = %s, want test-name", job.GetName())
	}
	if job.GetStatus() != JobStatusRunning {
		t.Errorf("GetStatus() = %s, want %s", job.GetStatus(), JobStatusRunning)
	}
	if job.GetPriority() != 7 {
		t.Errorf("GetPriority() = %d, want 7", job.GetPriority())
	}
	if !job.GetCreatedAt().Equal(now) {
		t.Errorf("GetCreatedAt() = %v, want %v", job.GetCreatedAt(), now)
	}
	if job.GetStartedAt() == nil || !job.GetStartedAt().Equal(startedAt) {
		t.Errorf("GetStartedAt() = %v, want %v", job.GetStartedAt(), &startedAt)
	}
	if job.GetCompletedAt() == nil || !job.GetCompletedAt().Equal(completedAt) {
		t.Errorf("GetCompletedAt() = %v, want %v", job.GetCompletedAt(), &completedAt)
	}
	if len(job.GetErrors()) != 1 {
		t.Errorf("GetErrors() length = %d, want 1", len(job.GetErrors()))
	}
	if job.GetLatestError() != "test error" {
		t.Errorf("GetLatestError() = %s, want test error", job.GetLatestError())
	}
	if job.GetRetryCount() != 3 {
		t.Errorf("GetRetryCount() = %d, want 3", job.GetRetryCount())
	}
	if len(job.GetTags()) != 2 {
		t.Errorf("GetTags() length = %d, want 2", len(job.GetTags()))
	}

	payload := job.GetPayload()
	if payload == nil {
		t.Error("GetPayload() should not be nil")
	}

	metadata := job.GetMetadata()
	if metadata == nil {
		t.Error("GetMetadata() should not be nil")
	}
	if len(metadata) != 1 {
		t.Errorf("GetMetadata() length = %d, want 1", len(metadata))
	}
}

// TestJobMetadataModel_Validate tests validation logic
func TestJobMetadataModel_Validate(t *testing.T) {
	tests := []struct {
		name    string
		job     *JobMetadataModel
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid pending_dispatch job",
			job: &JobMetadataModel{
				JobID:      "123e4567-e89b-12d3-a456-426614174000",
				Name:       "test-job",
				Status:     JobStatusPendingDispatch,
				Topic:      "persistent://public/default/jobs-test",
				Priority:   5,
				CreatedAt:  time.Now(),
				RetryCount: 0,
				Tags:       []string{},
				Payload:    map[string]any{},
				Metadata:   map[string]any{},
			},
			wantErr: false,
		},
		{
			name: "missing jobId",
			job: &JobMetadataModel{
				Name:      "test-job",
				Status:    JobStatusPendingDispatch,
				Priority:  5,
				CreatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "jobId is required",
		},
		{
			name: "invalid jobId format",
			job: &JobMetadataModel{
				JobID:     "invalid-id",
				Name:      "test-job",
				Status:    JobStatusPendingDispatch,
				Priority:  5,
				CreatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "jobId must be a valid UUID (36 characters)",
		},
		{
			name: "missing name",
			job: &JobMetadataModel{
				JobID:     "123e4567-e89b-12d3-a456-426614174000",
				Status:    JobStatusPendingDispatch,
				Priority:  5,
				CreatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "name too long",
			job: &JobMetadataModel{
				JobID:     "123e4567-e89b-12d3-a456-426614174000",
				Name:      string(make([]byte, 101)),
				Status:    JobStatusPendingDispatch,
				Priority:  5,
				CreatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "name must not exceed 100 characters",
		},
		{
			name: "invalid status",
			job: &JobMetadataModel{
				JobID:     "123e4567-e89b-12d3-a456-426614174000",
				Name:      "test-job",
				Status:    "invalid",
				Priority:  5,
				CreatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "invalid status value",
		},
		{
			name: "priority too low",
			job: &JobMetadataModel{
				JobID:     "123e4567-e89b-12d3-a456-426614174000",
				Name:      "test-job",
				Status:    JobStatusPendingDispatch,
				Priority:  -1,
				CreatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "priority must be between 0 and 10",
		},
		{
			name: "priority too high",
			job: &JobMetadataModel{
				JobID:     "123e4567-e89b-12d3-a456-426614174000",
				Name:      "test-job",
				Status:    JobStatusPendingDispatch,
				Priority:  11,
				CreatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "priority must be between 0 and 10",
		},
		{
			name: "missing createdAt",
			job: &JobMetadataModel{
				JobID:    "123e4567-e89b-12d3-a456-426614174000",
				Name:     "test-job",
				Status:   JobStatusPendingDispatch,
				Priority: 5,
			},
			wantErr: true,
			errMsg:  "createdAt is required",
		},
		{
			name: "negative retryCount",
			job: &JobMetadataModel{
				JobID:      "123e4567-e89b-12d3-a456-426614174000",
				Name:       "test-job",
				Status:     JobStatusPendingDispatch,
				Priority:   5,
				CreatedAt:  time.Now(),
				RetryCount: -1,
			},
			wantErr: true,
			errMsg:  "retryCount cannot be negative",
		},
		{
			name: "running job without startedAt",
			job: &JobMetadataModel{
				JobID:     "123e4567-e89b-12d3-a456-426614174000",
				Name:      "test-job",
				Status:    JobStatusRunning,
				Priority:  5,
				CreatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "running job must have startedAt timestamp",
		},
		{
			name: "completed job without completedAt",
			job: func() *JobMetadataModel {
				started := time.Now()
				return &JobMetadataModel{
					JobID:     "123e4567-e89b-12d3-a456-426614174000",
					Name:      "test-job",
					Status:    JobStatusCompleted,
					Priority:  5,
					CreatedAt: time.Now(),
					StartedAt: &started,
				}
			}(),
			wantErr: true,
			errMsg:  "completed or cancelled job must have completedAt timestamp",
		},
		{
			name: "failed job without error message",
			job: func() *JobMetadataModel {
				now := time.Now()
				return &JobMetadataModel{
					JobID:       "123e4567-e89b-12d3-a456-426614174000",
					Name:        "test-job",
					Status:      JobStatusFailed,
					Priority:    5,
					CreatedAt:   time.Now(),
					CompletedAt: &now,
				}
			}(),
			wantErr: true,
			errMsg:  "failed job must have error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.job.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					if len(err.Error()) < len(tt.errMsg) || err.Error()[:len(tt.errMsg)] != tt.errMsg {
						t.Errorf("error message = %q, want to contain %q", err.Error(), tt.errMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestJobMetadataModel_SetStatus tests status transitions
func TestJobMetadataModel_SetStatus(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus JobStatus
		targetStatus  JobStatus
		wantErr       bool
		checkStarted  bool
		checkComplete bool
	}{
		{
			name:          "dispatched to running",
			initialStatus: JobStatusDispatched,
			targetStatus:  JobStatusRunning,
			wantErr:       false,
			checkStarted:  true,
		},
		{
			name:          "pending_dispatch to cancelled",
			initialStatus: JobStatusPendingDispatch,
			targetStatus:  JobStatusCancelled,
			wantErr:       false,
			checkComplete: true,
		},
		{
			name:          "running to completed",
			initialStatus: JobStatusRunning,
			targetStatus:  JobStatusCompleted,
			wantErr:       false,
			checkComplete: true,
		},
		{
			name:          "running to failed",
			initialStatus: JobStatusRunning,
			targetStatus:  JobStatusFailed,
			wantErr:       false,
			checkComplete: true,
		},
		{
			name:          "running to cancelled",
			initialStatus: JobStatusRunning,
			targetStatus:  JobStatusCancelled,
			wantErr:       false,
			checkComplete: true,
		},
		{
			name:          "pending_dispatch to completed (invalid)",
			initialStatus: JobStatusPendingDispatch,
			targetStatus:  JobStatusCompleted,
			wantErr:       true,
		},
		{
			name:          "completed to running (invalid)",
			initialStatus: JobStatusCompleted,
			targetStatus:  JobStatusRunning,
			wantErr:       true,
		},
		{
			name:          "failed to running (invalid)",
			initialStatus: JobStatusFailed,
			targetStatus:  JobStatusRunning,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &JobMetadataModel{
				Status: tt.initialStatus,
			}

			err := job.SetStatus(tt.targetStatus)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if job.Status != tt.targetStatus {
					t.Errorf("status = %s, want %s", job.Status, tt.targetStatus)
				}
				if tt.checkStarted && job.StartedAt == nil {
					t.Error("expected startedAt to be set")
				}
				if tt.checkComplete && job.CompletedAt == nil {
					t.Error("expected completedAt to be set")
				}
			}
		})
	}
}

// TestJobMetadataModel_AddError tests error handling with history tracking
func TestJobMetadataModel_AddError(t *testing.T) {
	t.Run("adds error and transitions to failed", func(t *testing.T) {
		job := &JobMetadataModel{
			Status:     JobStatusRunning,
			RetryCount: 0,
		}

		testErr := errors.New("test error message")
		err := job.AddError(testErr)

		if err != nil {
			t.Fatalf("AddError() returned error: %v", err)
		}
		if job.Status != JobStatusFailed {
			t.Errorf("status = %s, want failed", job.Status)
		}
		if len(job.Errors) != 1 {
			t.Fatalf("len(Errors) = %d, want 1", len(job.Errors))
		}
		if job.Errors[0].Error != "test error message" {
			t.Errorf("Errors[0].Error = %s, want test error message", job.Errors[0].Error)
		}
		if job.Errors[0].RetryAttempt != 0 {
			t.Errorf("Errors[0].RetryAttempt = %d, want 0", job.Errors[0].RetryAttempt)
		}
		if job.Errors[0].Timestamp.IsZero() {
			t.Error("Errors[0].Timestamp should be set")
		}
		if job.CompletedAt == nil {
			t.Error("expected completedAt to be set")
		}
	})

	t.Run("handles nil error", func(t *testing.T) {
		job := &JobMetadataModel{
			Status:     JobStatusRunning,
			RetryCount: 0,
		}

		err := job.AddError(nil)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(job.Errors) != 0 {
			t.Errorf("errors array should be empty, got length %d", len(job.Errors))
		}
	})
}

// TestJobMetadataModel_AddError_MultipleRetries tests error history across retry attempts
func TestJobMetadataModel_AddError_MultipleRetries(t *testing.T) {
	job := &JobMetadataModel{
		Status:     JobStatusRunning,
		RetryCount: 0,
	}

	// First error
	if err := job.AddError(errors.New("first error")); err != nil {
		t.Fatalf("AddError() returned error: %v", err)
	}
	if len(job.Errors) != 1 {
		t.Fatalf("len(Errors) = %d, want 1", len(job.Errors))
	}
	if job.Errors[0].RetryAttempt != 0 {
		t.Errorf("Errors[0].RetryAttempt = %d, want 0", job.Errors[0].RetryAttempt)
	}
	if job.Errors[0].Error != "first error" {
		t.Errorf("Errors[0].Error = %s, want 'first error'", job.Errors[0].Error)
	}

	// Simulate retry
	job.RetryCount = 1
	job.Status = JobStatusRunning

	// Second error
	if err := job.AddError(errors.New("second error")); err != nil {
		t.Fatalf("AddError() returned error: %v", err)
	}
	if len(job.Errors) != 2 {
		t.Fatalf("len(Errors) = %d, want 2", len(job.Errors))
	}
	if job.Errors[1].RetryAttempt != 1 {
		t.Errorf("Errors[1].RetryAttempt = %d, want 1", job.Errors[1].RetryAttempt)
	}
	if job.Errors[1].Error != "second error" {
		t.Errorf("Errors[1].Error = %s, want 'second error'", job.Errors[1].Error)
	}

	// Verify timestamps are ordered
	if !job.Errors[1].Timestamp.After(job.Errors[0].Timestamp) && !job.Errors[1].Timestamp.Equal(job.Errors[0].Timestamp) {
		t.Error("second error timestamp should be after or equal to first error timestamp")
	}
}

// TestJobMetadataModel_GetLatestError tests the convenience method for getting the most recent error
func TestJobMetadataModel_GetLatestError(t *testing.T) {
	t.Run("returns empty string for no errors", func(t *testing.T) {
		job := &JobMetadataModel{Errors: []JobError{}}

		if got := job.GetLatestError(); got != "" {
			t.Errorf("GetLatestError() = %s, want empty string", got)
		}
	})

	t.Run("returns latest error message", func(t *testing.T) {
		job := &JobMetadataModel{
			Errors: []JobError{
				{RetryAttempt: 0, Error: "first error", Timestamp: time.Now()},
				{RetryAttempt: 1, Error: "second error", Timestamp: time.Now()},
				{RetryAttempt: 2, Error: "third error", Timestamp: time.Now()},
			},
		}

		if got := job.GetLatestError(); got != "third error" {
			t.Errorf("GetLatestError() = %s, want 'third error'", got)
		}
	})

	t.Run("returns single error", func(t *testing.T) {
		job := &JobMetadataModel{
			Errors: []JobError{
				{RetryAttempt: 0, Error: "only error", Timestamp: time.Now()},
			},
		}

		if got := job.GetLatestError(); got != "only error" {
			t.Errorf("GetLatestError() = %s, want 'only error'", got)
		}
	})
}

// TestJobMetadataModel_AddTag tests tag management
func TestJobMetadataModel_AddTag(t *testing.T) {
	t.Run("adds new tag", func(t *testing.T) {
		job := &JobMetadataModel{
			Tags: []string{},
		}

		job.AddTag("test-tag")

		if len(job.Tags) != 1 {
			t.Errorf("tags length = %d, want 1", len(job.Tags))
		}
		if job.Tags[0] != "test-tag" {
			t.Errorf("tag = %s, want test-tag", job.Tags[0])
		}
	})

	t.Run("ignores duplicate tag", func(t *testing.T) {
		job := &JobMetadataModel{
			Tags: []string{"existing-tag"},
		}

		job.AddTag("existing-tag")

		if len(job.Tags) != 1 {
			t.Errorf("tags length = %d, want 1", len(job.Tags))
		}
	})

	t.Run("ignores empty tag", func(t *testing.T) {
		job := &JobMetadataModel{
			Tags: []string{},
		}

		job.AddTag("")

		if len(job.Tags) != 0 {
			t.Errorf("tags length = %d, want 0", len(job.Tags))
		}
	})
}

// TestJobMetadataModel_RemoveTag tests tag removal
func TestJobMetadataModel_RemoveTag(t *testing.T) {
	t.Run("removes existing tag", func(t *testing.T) {
		job := &JobMetadataModel{
			Tags: []string{"tag1", "tag2", "tag3"},
		}

		job.RemoveTag("tag2")

		if len(job.Tags) != 2 {
			t.Errorf("tags length = %d, want 2", len(job.Tags))
		}
		for _, tag := range job.Tags {
			if tag == "tag2" {
				t.Error("tag2 should have been removed")
			}
		}
	})

	t.Run("handles non-existent tag", func(t *testing.T) {
		job := &JobMetadataModel{
			Tags: []string{"tag1"},
		}

		job.RemoveTag("non-existent")

		if len(job.Tags) != 1 {
			t.Errorf("tags length = %d, want 1", len(job.Tags))
		}
	})
}

// TestJobMetadataModel_MetadataFields tests metadata field management
func TestJobMetadataModel_MetadataFields(t *testing.T) {
	t.Run("sets and gets metadata field", func(t *testing.T) {
		job := &JobMetadataModel{
			Metadata: make(map[string]any),
		}

		job.SetMetadataField("key1", "value1")
		job.SetMetadataField("key2", 42)

		val1, exists1 := job.GetMetadataField("key1")
		if !exists1 {
			t.Error("expected key1 to exist")
		}
		if val1 != "value1" {
			t.Errorf("key1 value = %v, want value1", val1)
		}

		val2, exists2 := job.GetMetadataField("key2")
		if !exists2 {
			t.Error("expected key2 to exist")
		}
		if val2 != 42 {
			t.Errorf("key2 value = %v, want 42", val2)
		}
	})

	t.Run("initializes metadata map if nil", func(t *testing.T) {
		job := &JobMetadataModel{}

		job.SetMetadataField("key", "value")

		if job.Metadata == nil {
			t.Error("expected metadata to be initialized")
		}
	})

	t.Run("returns false for non-existent key", func(t *testing.T) {
		job := &JobMetadataModel{
			Metadata: make(map[string]any),
		}

		_, exists := job.GetMetadataField("non-existent")
		if exists {
			t.Error("expected key to not exist")
		}
	})

	t.Run("handles nil metadata map", func(t *testing.T) {
		job := &JobMetadataModel{}

		_, exists := job.GetMetadataField("key")
		if exists {
			t.Error("expected key to not exist when metadata is nil")
		}
	})
}

// TestJobMetadataModel_IncrementRetryCount tests retry counter
func TestJobMetadataModel_IncrementRetryCount(t *testing.T) {
	job := &JobMetadataModel{
		RetryCount: 0,
	}

	job.IncrementRetryCount()
	if job.RetryCount != 1 {
		t.Errorf("retryCount = %d, want 1", job.RetryCount)
	}

	job.IncrementRetryCount()
	if job.RetryCount != 2 {
		t.Errorf("retryCount = %d, want 2", job.RetryCount)
	}
}

// TestJobMetadataModel_Duration tests duration calculation
func TestJobMetadataModel_Duration(t *testing.T) {
	t.Run("calculates duration for completed job", func(t *testing.T) {
		start := time.Now()
		end := start.Add(5 * time.Second)

		job := &JobMetadataModel{
			StartedAt:   &start,
			CompletedAt: &end,
		}

		duration := job.Duration()
		if duration != 5*time.Second {
			t.Errorf("duration = %v, want 5s", duration)
		}
	})

	t.Run("returns zero for job without start time", func(t *testing.T) {
		job := &JobMetadataModel{}

		duration := job.Duration()
		if duration != 0 {
			t.Errorf("duration = %v, want 0", duration)
		}
	})

	t.Run("returns zero for running job", func(t *testing.T) {
		start := time.Now()
		job := &JobMetadataModel{
			StartedAt: &start,
		}

		duration := job.Duration()
		if duration != 0 {
			t.Errorf("duration = %v, want 0", duration)
		}
	})
}

// TestJobMetadataModel_Age tests age calculation
func TestJobMetadataModel_Age(t *testing.T) {
	createdAt := time.Now().Add(-10 * time.Second)
	job := &JobMetadataModel{
		CreatedAt: createdAt,
	}

	age := job.Age()
	if age < 9*time.Second || age > 11*time.Second {
		t.Errorf("age = %v, want approximately 10s", age)
	}
}

// TestJobMetadataModel_InterfaceCompliance verifies interface implementation
func TestJobMetadataModel_InterfaceCompliance(t *testing.T) {
	var _ JobMetadata = (*JobMetadataModel)(nil)
}

func TestAsJobModel_AcceptsModel(t *testing.T) {
	job := NewJobMetadata("123e4567-e89b-12d3-a456-426614174000", "test-job", nil)

	got, err := AsJobModel(job)
	if err != nil {
		t.Fatal(err)
	}
	if got != job {
		t.Fatal("expected same model pointer")
	}
}

func TestAsJobModel_RejectsUnexpectedType(t *testing.T) {
	_, err := AsJobModel(bogusJobMetadata{})
	if err == nil {
		t.Fatal("expected error")
	}
}

type bogusJobMetadata struct{}

func (bogusJobMetadata) GetJobID() string            { return "job-1" }
func (bogusJobMetadata) GetName() string             { return "x" }
func (bogusJobMetadata) GetStatus() JobStatus        { return JobStatusPendingDispatch }
func (bogusJobMetadata) GetPriority() int            { return 5 }
func (bogusJobMetadata) GetCreatedAt() time.Time     { return time.Now() }
func (bogusJobMetadata) GetStartedAt() *time.Time    { return nil }
func (bogusJobMetadata) GetCompletedAt() *time.Time  { return nil }
func (bogusJobMetadata) GetPayload() any             { return nil }
func (bogusJobMetadata) GetMetadata() map[string]any { return nil }
func (bogusJobMetadata) GetErrors() []JobError       { return nil }
func (bogusJobMetadata) GetLatestError() string      { return "" }
func (bogusJobMetadata) GetRetryCount() int          { return 0 }
func (bogusJobMetadata) GetTags() []string           { return nil }
func (bogusJobMetadata) Validate() error             { return nil }
