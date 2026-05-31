//go:build integration

package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func prepareIntegrationApp(t *testing.T) (*app.App, func()) {
	t.Helper()
	cfg := integrationMongoConfig(t)
	ctx := context.Background()

	reader, writer, client, err := metadata.OpenMongoJobs(ctx, cfg)
	if err != nil {
		t.Fatalf("OpenMongoJobs: %v", err)
	}
	db := client.Database(cfg.Database)
	if err := clearJobCollections(ctx, db, cfg); err != nil {
		t.Fatalf("clear collections: %v", err)
	}

	svc := service.NewMetadataService(reader, writer)
	application := app.New(svc, writer)

	cleanup := func() {
		_ = clearJobCollections(context.Background(), db, cfg)
		_ = client.Disconnect(context.Background())
	}

	return application, cleanup
}

func clearJobCollections(ctx context.Context, db *mongo.Database, cfg metadata.MongoConfig) error {
	meta := db.Collection(cfg.CollectionMetadata)
	logs := db.Collection(cfg.CollectionLogs)
	if _, err := meta.DeleteMany(ctx, bson.M{}); err != nil {
		return fmt.Errorf("clear %q: %w", cfg.CollectionMetadata, err)
	}
	if _, err := logs.DeleteMany(ctx, bson.M{}); err != nil {
		return fmt.Errorf("clear %q: %w", cfg.CollectionLogs, err)
	}
	return nil
}

func runCLICommand(t *testing.T, application *app.App, setup func(*testing.T, *app.App) *cobra.Command) []byte {
	t.Helper()
	var buf bytes.Buffer
	cmd := setup(t, application)
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	application.Out = &buf
	if err := cmd.Execute(); err != nil {
		t.Fatalf("command failed: %v (output=%q)", err, buf.String())
	}
	return buf.Bytes()
}

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
