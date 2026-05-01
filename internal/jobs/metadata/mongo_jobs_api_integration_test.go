//go:build integration

package metadata

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Integration tests require MongoDB (for example: docker compose up -d).
// Run: make test-integration
//
// MONGO_URI defaults to the same connection string as cmd/jobs-cli when unset.

func testMongoConfig(tb testing.TB) MongoConfig {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test (-short)")
	}
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://jobby_app:jobby_app_pass@localhost:27018/jobby?authSource=jobby"
	}
	return MongoConfig{
		URI:                uri,
		Database:           "jobby",
		CollectionMetadata: "job_metadata",
		CollectionLogs:     "job_logs",
		Timeout:            30 * time.Second,
		MaxPoolSize:        50,
		MinPoolSize:        0,
	}
}

func integrationConnect(tb testing.TB, ctx context.Context, cfg MongoConfig) *mongo.Client {
	tb.Helper()
	client, err := mongo.Connect(
		options.Client().
			ApplyURI(cfg.URI).
			SetTimeout(cfg.Timeout).
			SetMaxPoolSize(cfg.MaxPoolSize).
			SetMinPoolSize(cfg.MinPoolSize),
	)
	if err != nil {
		tb.Fatalf("connect to mongo: %v", err)
	}
	tb.Cleanup(func() {
		_ = client.Disconnect(context.Background())
	})
	if err := client.Ping(ctx, nil); err != nil {
		tb.Fatalf("ping mongo: %v (is the compose mongo service up?)", err)
	}
	return client
}

func clearJobCollections(ctx context.Context, db *mongo.Database, cfg MongoConfig) error {
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

// setupIntegrationCollections removes all documents from metadata and logs collections.
// Indexes and validators from scripts/mongo-init.js stay in place so later tests (e.g. EnsureIndexes) still pass.
func setupIntegrationCollections(ctx context.Context, db *mongo.Database, cfg MongoConfig) error {
	return clearJobCollections(ctx, db, cfg)
}

// teardownIntegrationCollections clears both collections again so writes do not leak across runs.
func teardownIntegrationCollections(ctx context.Context, db *mongo.Database, cfg MongoConfig) {
	_ = clearJobCollections(ctx, db, cfg)
}

// TestIntegration_MongoJobsApi groups subtests in order: EnsureIndexes runs against a DB
// provisioned by compose/mongo-init.js; Create uses setup/teardown fixtures.
func TestIntegration_MongoJobsApi(t *testing.T) {
	ctx := context.Background()
	cfg := testMongoConfig(t)

	t.Run("EnsureIndexes", func(t *testing.T) {
		api, err := NewMongoJobsApi(ctx, cfg)
		if err != nil {
			t.Fatalf("NewMongoJobsApi: %v", err)
		}
		if err := api.Close(ctx); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	t.Run("Create", func(t *testing.T) {
		client := integrationConnect(t, ctx, cfg)
		db := client.Database(cfg.Database)

		if err := setupIntegrationCollections(ctx, db, cfg); err != nil {
			t.Fatalf("setupIntegrationCollections: %v", err)
		}
		t.Cleanup(func() {
			teardownIntegrationCollections(ctx, db, cfg)
		})

		api, err := NewMongoJobsApi(ctx, cfg)
		if err != nil {
			t.Fatalf("NewMongoJobsApi: %v", err)
		}
		defer func() {
			if err := api.Close(ctx); err != nil {
				t.Errorf("api Close: %v", err)
			}
		}()

		job := NewJobMetadata(GenerateJobID(), "integration-create", map[string]any{"k": "v"})
		if err := api.Create(ctx, job); err != nil {
			t.Fatalf("Create: %v", err)
		}

		got, err := api.Get(ctx, job.JobID)
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got.GetJobID() != job.JobID {
			t.Errorf("JobID = %q, want %q", got.GetJobID(), job.JobID)
		}
		if got.GetName() != job.Name {
			t.Errorf("Name = %q, want %q", got.GetName(), job.Name)
		}
		if got.GetStatus() != JobStatusPending {
			t.Errorf("Status = %s, want %s", got.GetStatus(), JobStatusPending)
		}
	})
}
