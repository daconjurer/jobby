package metadata

import (
	"testing"
	"time"
)

// TestGenerateJobID tests UUID generation
func TestGenerateJobID(t *testing.T) {
	t.Run("generates valid UUID", func(t *testing.T) {
		id := GenerateJobID()

		if len(id) != 36 {
			t.Errorf("generated ID length = %d, want 36", len(id))
		}

		if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
			t.Errorf("generated ID = %s, does not match UUID format", id)
		}
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		id1 := GenerateJobID()
		id2 := GenerateJobID()

		if id1 == id2 {
			t.Error("generated IDs should be unique")
		}
	})
}

// TestNewJobLog tests basic log creation
func TestNewJobLog(t *testing.T) {
	jobID := "123e4567-e89b-12d3-a456-426614174000"
	level := LogLevelInfo
	message := "test message"

	log := NewJobLog(jobID, level, message)

	if log.JobID != jobID {
		t.Errorf("jobID = %s, want %s", log.JobID, jobID)
	}
	if log.Level != level {
		t.Errorf("level = %s, want %s", log.Level, level)
	}
	if log.Message != message {
		t.Errorf("message = %s, want %s", log.Message, message)
	}
	if log.Timestamp.IsZero() {
		t.Error("timestamp should be set")
	}
	if log.Context == nil {
		t.Error("context should be initialized")
	}
	if len(log.Context) != 0 {
		t.Errorf("context should be empty, got %v", log.Context)
	}
}

// TestNewJobLogWithContext tests log creation with context
func TestNewJobLogWithContext(t *testing.T) {
	jobID := "123e4567-e89b-12d3-a456-426614174000"
	level := LogLevelDebug
	message := "test message with context"
	context := map[string]any{
		"key1": "value1",
		"key2": 42,
	}

	log := NewJobLogWithContext(jobID, level, message, context)

	if log.JobID != jobID {
		t.Errorf("jobID = %s, want %s", log.JobID, jobID)
	}
	if log.Level != level {
		t.Errorf("level = %s, want %s", log.Level, level)
	}
	if log.Message != message {
		t.Errorf("message = %s, want %s", log.Message, message)
	}
	if log.Context == nil {
		t.Error("context should not be nil")
	}
	if len(log.Context) != 2 {
		t.Errorf("context length = %d, want 2", len(log.Context))
	}
	if log.Context["key1"] != "value1" {
		t.Errorf("context[key1] = %v, want value1", log.Context["key1"])
	}
	if log.Context["key2"] != 42 {
		t.Errorf("context[key2] = %v, want 42", log.Context["key2"])
	}
}

// TestNewJobLogWithSource tests log creation with source
func TestNewJobLogWithSource(t *testing.T) {
	jobID := "123e4567-e89b-12d3-a456-426614174000"
	level := LogLevelError
	message := "test message with source"
	source := "executor"

	log := NewJobLogWithSource(jobID, level, message, source)

	if log.JobID != jobID {
		t.Errorf("jobID = %s, want %s", log.JobID, jobID)
	}
	if log.Level != level {
		t.Errorf("level = %s, want %s", log.Level, level)
	}
	if log.Message != message {
		t.Errorf("message = %s, want %s", log.Message, message)
	}
	if log.Source != source {
		t.Errorf("source = %s, want %s", log.Source, source)
	}
}

// TestJobLog_AllFields tests that all log fields can be set
func TestJobLog_AllFields(t *testing.T) {
	now := time.Now()
	log := JobLog{
		JobID:      "test-job-id",
		Timestamp:  now,
		Level:      LogLevelWarn,
		Message:    "warning message",
		Context:    map[string]any{"ctx": "value"},
		Source:     "test-source",
		StackTrace: "line 1\nline 2\nline 3",
	}

	if log.JobID != "test-job-id" {
		t.Errorf("jobID = %s, want test-job-id", log.JobID)
	}
	if !log.Timestamp.Equal(now) {
		t.Errorf("timestamp = %v, want %v", log.Timestamp, now)
	}
	if log.Level != LogLevelWarn {
		t.Errorf("level = %s, want %s", log.Level, LogLevelWarn)
	}
	if log.Message != "warning message" {
		t.Errorf("message = %s, want warning message", log.Message)
	}
	if log.Source != "test-source" {
		t.Errorf("source = %s, want test-source", log.Source)
	}
	if log.StackTrace != "line 1\nline 2\nline 3" {
		t.Errorf("stackTrace = %s, want multi-line trace", log.StackTrace)
	}
}
