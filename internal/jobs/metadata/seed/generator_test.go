package seed

import (
	"testing"
	"time"
)

func TestBuildJob_Validate(t *testing.T) {
	f := newFaker(42)
	for i := 0; i < 200; i++ {
		job, err := BuildJob(f, 7*24*time.Hour)
		if err != nil {
			t.Fatalf("iteration %d: BuildJob: %v", i, err)
		}
		if err := job.Validate(); err != nil {
			t.Fatalf("iteration %d: Validate: %v", i, err)
		}
	}
}

func TestBuildLogs_WithinJobWindow(t *testing.T) {
	f := newFaker(99)
	job, err := BuildJob(f, 24*time.Hour)
	if err != nil {
		t.Fatalf("BuildJob: %v", err)
	}

	logs := BuildLogs(job, 3, 5, f)
	if len(logs) < 3 {
		t.Fatalf("expected at least 3 logs, got %d", len(logs))
	}

	end := time.Now().UTC()
	if job.CompletedAt != nil {
		end = *job.CompletedAt
	} else if job.StartedAt != nil {
		end = *job.StartedAt
	}

	for i, log := range logs {
		if log.JobID != job.JobID {
			t.Fatalf("log %d jobId = %q, want %q", i, log.JobID, job.JobID)
		}
		if log.Timestamp.Before(job.CreatedAt) || log.Timestamp.After(end) {
			t.Fatalf("log %d timestamp %v outside [%v, %v]", i, log.Timestamp, job.CreatedAt, end)
		}
		if !log.Level.IsValid() {
			t.Fatalf("log %d invalid level %q", i, log.Level)
		}
	}
}

func TestRun_InvalidOptions(t *testing.T) {
	t.Run("nil writer", func(t *testing.T) {
		_, err := Run(t.Context(), nil, Options{Count: 10})
		if err == nil {
			t.Fatal("expected error for nil writer")
		}
	})

	t.Run("count too small", func(t *testing.T) {
		_, err := Run(t.Context(), nil, Options{Count: 0})
		if err == nil {
			t.Fatal("expected error for count < 1")
		}
	})
}
