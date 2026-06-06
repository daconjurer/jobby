//go:build integration

package commands

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/mongodb"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func integrationMongoConfig(tb testing.TB) mongodb.MongoConfig {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test (-short)")
	}
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		tb.Fatalf("MONGODB_URI is not set (required for integration tests; see .env and compose.yml files)")
	}
	db := os.Getenv("MONGODB_DATABASE")
	if db == "" {
		db = "jobby"
	}
	metaColl := os.Getenv("MONGODB_COLLECTION_METADATA")
	if metaColl == "" {
		metaColl = "job_metadata"
	}
	logsColl := os.Getenv("MONGODB_COLLECTION_LOGS")
	if logsColl == "" {
		logsColl = "job_logs"
	}
	return mongodb.MongoConfig{
		URI:                uri,
		Database:           db,
		CollectionMetadata: metaColl,
		CollectionLogs:     logsColl,
		Timeout:            30 * time.Second,
		MaxPoolSize:        50,
		MinPoolSize:        0,
	}
}

func prepareIntegrationApp(t *testing.T) (*app.App, func()) {
	t.Helper()
	cfg := integrationMongoConfig(t)
	ctx := context.Background()

	reader, writer, client, err := mongodb.OpenMongoJobs(ctx, cfg)
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

func clearJobCollections(ctx context.Context, db *mongo.Database, cfg mongodb.MongoConfig) error {
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

func markJobRunningForTest(t *testing.T, application *app.App, jobID string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	running := metadata.JobStatusRunning
	patch := metadata.UpdateJob{Status: &running, StartedAt: &now}
	if err := application.Writer.Update(ctx, jobID, patch); err != nil {
		t.Fatalf("mark job running: %v", err)
	}
}

func runCLICommandExpectError(t *testing.T, application *app.App, setup func(*testing.T, *app.App) *cobra.Command) error {
	t.Helper()
	var buf bytes.Buffer
	cmd := setup(t, application)
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	application.Out = &buf
	return cmd.Execute()
}
