//go:build integration

package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"github.com/spf13/cobra"
)

func TestIntegration_Get_not_found(t *testing.T) {
	application, cleanup := prepareIntegrationApp(t)
	defer cleanup()

	cmd := NewGetCmd(application)
	cmd.SetArgs([]string{metadata.GenerateJobID()})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for missing job")
	}
	if err.Error() != "job not found" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntegration_List_invalid_status(t *testing.T) {
	application, cleanup := prepareIntegrationApp(t)
	defer cleanup()

	cmd := NewListCmd(application)
	cmd.SetArgs([]string{"--status=nope"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("invalid status")) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntegration_List_filter_status_and_stats(t *testing.T) {
	application, cleanup := prepareIntegrationApp(t)
	defer cleanup()
	ctx := context.Background()

	statsOut := runCLICommand(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		return NewStatsCmd(a)
	})
	var stats0 service.JobStats
	if err := json.Unmarshal(statsOut, &stats0); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if stats0.Total != 0 {
		t.Fatalf("want empty stats, got %+v", stats0)
	}

	created, err := application.Service.CreateJob(ctx, "listed-job", nil, service.CreateJobOptions{})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	listOut := runCLICommand(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewListCmd(a)
		cmd.SetArgs([]string{"--status=pending"})
		return cmd
	})
	var listWrap struct {
		Jobs []metadata.JobMetadataModel `json:"jobs"`
	}
	if err := json.Unmarshal(listOut, &listWrap); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	found := false
	for _, j := range listWrap.Jobs {
		if j.JobID == created.JobID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("pending list missing job %+v", listWrap.Jobs)
	}

	statsOut1 := runCLICommand(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		return NewStatsCmd(a)
	})
	var stats1 service.JobStats
	if err := json.Unmarshal(statsOut1, &stats1); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if stats1.Total < 1 || stats1.Pending < 1 {
		t.Fatalf("stats after create: %+v", stats1)
	}
}

func TestIntegration_Seed_inserts_jobs_and_logs(t *testing.T) {
	application, cleanup := prepareIntegrationApp(t)
	defer cleanup()

	seedOut := runCLICommand(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewSeedCmd(a)
		cmd.SetArgs([]string{"--count=5", "--seed=123", "--logs-per-job-min=1", "--logs-per-job-max=2"})
		return cmd
	})
	var seedResult struct {
		JobsInserted int `json:"jobsInserted"`
		LogsInserted int `json:"logsInserted"`
	}
	if err := json.Unmarshal(seedOut, &seedResult); err != nil {
		t.Fatalf("decode seed result: %v", err)
	}
	if seedResult.JobsInserted != 5 {
		t.Fatalf("jobsInserted = %d, want 5", seedResult.JobsInserted)
	}
	if seedResult.LogsInserted < 5 {
		t.Fatalf("logsInserted = %d, want at least 5", seedResult.LogsInserted)
	}

	statsOut := runCLICommand(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		return NewStatsCmd(a)
	})
	var stats service.JobStats
	if err := json.Unmarshal(statsOut, &stats); err != nil {
		t.Fatalf("decode stats: %v", err)
	}
	if stats.Total < 5 {
		t.Fatalf("stats total = %d, want at least 5", stats.Total)
	}
}

func TestIntegration_Logs_levels(t *testing.T) {
	application, cleanup := prepareIntegrationApp(t)
	defer cleanup()
	ctx := context.Background()

	created, err := application.Service.CreateJob(ctx, "log-job", nil, service.CreateJobOptions{})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	logsOut := runCLICommand(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewLogsCmd(a)
		cmd.SetArgs([]string{created.JobID})
		return cmd
	})
	var logsWrap struct {
		Logs []metadata.JobLog `json:"logs"`
	}
	if err := json.Unmarshal(logsOut, &logsWrap); err != nil {
		t.Fatalf("decode logs: %v", err)
	}
	if len(logsWrap.Logs) < 1 {
		t.Fatalf("expected creation log, got %+v", logsWrap.Logs)
	}

	badCmd := NewLogsCmd(application)
	badCmd.SetArgs([]string{created.JobID, "--levels=nope"})
	badCmd.SetOut(&bytes.Buffer{})
	badCmd.SetErr(&bytes.Buffer{})
	err = badCmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("invalid log level")) {
		t.Fatalf("unexpected error: %v", err)
	}
}
