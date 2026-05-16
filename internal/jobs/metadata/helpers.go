package metadata

import (
	"time"

	"github.com/google/uuid"
)

// GenerateJobID generates a new UUID for a job
func GenerateJobID() string {
	return uuid.New().String()
}

// NewJobLog creates a new log entry for a job
func NewJobLog(jobID string, level LogLevel, message string) JobLog {
	return JobLog{
		JobID:     jobID,
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Context:   make(map[string]any),
	}
}

// NewJobLogWithContext creates a new log entry with additional context
func NewJobLogWithContext(jobID string, level LogLevel, message string, context map[string]any) JobLog {
	log := NewJobLog(jobID, level, message)
	log.Context = context
	return log
}

// NewJobLogWithSource creates a new log entry with source information
func NewJobLogWithSource(jobID string, level LogLevel, message, source string) JobLog {
	log := NewJobLog(jobID, level, message)
	log.Source = source
	return log
}
