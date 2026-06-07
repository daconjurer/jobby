package output

import (
	"strings"
	"testing"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/service"
)

func TestOutputFormat_Table(t *testing.T) {
	t.Run("stats", func(t *testing.T) {
		var buf strings.Builder
		stats := service.JobStats{
			Total:           10,
			PendingDispatch: 2,
			Dispatched:      1,
			DispatchFailed:  1,
			Running:         1,
			Completed:       3,
			Failed:          1,
			Cancelled:       1,
		}
		if err := WriteStatsTable(&buf, stats); err != nil {
			t.Fatalf("WriteStatsTable: %v", err)
		}
		out := buf.String()
		for _, want := range []string{"pending_dispatch", "2", "dispatched", "running", "1", "total", "10"} {
			if !strings.Contains(out, want) {
				t.Fatalf("stats table missing %q:\n%s", want, out)
			}
		}
	})

	t.Run("jobs", func(t *testing.T) {
		var buf strings.Builder
		jobs := []metadata.JobMetadata{
			&metadata.JobMetadataModel{
				JobID:      "00000000-0000-0000-0000-000000000001",
				Name:       "demo",
				Status:     metadata.JobStatusPendingDispatch,
				Priority:   7,
				RetryCount: 0,
				CreatedAt:  time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC),
			},
		}
		if err := WriteJobsTable(&buf, jobs); err != nil {
			t.Fatalf("WriteJobsTable: %v", err)
		}
		out := buf.String()
		for _, want := range []string{"JOB ID", "demo", "pending_dispatch", "7"} {
			if !strings.Contains(out, want) {
				t.Fatalf("jobs table missing %q:\n%s", want, out)
			}
		}
	})

	t.Run("logs", func(t *testing.T) {
		var buf strings.Builder
		logs := []metadata.JobLog{
			{
				Timestamp: time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC),
				Level:     metadata.LogLevelInfo,
				Message:   "Job created: demo",
				Source:    "service",
			},
		}
		if err := WriteLogsTable(&buf, logs); err != nil {
			t.Fatalf("WriteLogsTable: %v", err)
		}
		out := buf.String()
		for _, want := range []string{"TIMESTAMP", "info", "Job created", "service"} {
			if !strings.Contains(out, want) {
				t.Fatalf("logs table missing %q:\n%s", want, out)
			}
		}
	})
}
