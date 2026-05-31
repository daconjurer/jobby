//go:build integration

package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

func integrationMongoConfig(tb testing.TB) metadata.MongoConfig {
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
	return metadata.MongoConfig{
		URI:                uri,
		Database:           db,
		CollectionMetadata: metaColl,
		CollectionLogs:     logsColl,
		Timeout:            30 * time.Second,
		MaxPoolSize:        50,
		MinPoolSize:        0,
	}
}

func TestIntegration_Ping(t *testing.T) {
	cfg := integrationMongoConfig(t)
	ctx := context.Background()

	application, cleanup, err := app.Bootstrap(ctx, cfg)
	if err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	defer cleanup()

	var buf bytes.Buffer
	application.Out = &buf

	cmd := NewPingCmd(application)
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("ping: %v", err)
	}

	var got healthResponse
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("decode health JSON: %v (body=%q)", err, buf.String())
	}
	if got.Status != "healthy" || got.Database != "connected" {
		t.Fatalf("health response: got %+v", got)
	}
}
