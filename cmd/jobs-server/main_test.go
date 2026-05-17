package main

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

var jobsEnvMongoKeys = []string{
	"MONGODB_URI",
	"MONGODB_DATABASE",
	"MONGODB_COLLECTION_METADATA",
	"MONGODB_COLLECTION_LOGS",
	"MONGODB_TIMEOUT",
	"MONGODB_MAX_POOL_SIZE",
	"MONGODB_MIN_POOL_SIZE",
}

func temporaryUnsetEnv(t *testing.T, keys ...string) {
	t.Helper()
	prev := make(map[string]string)
	present := make(map[string]bool)
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			present[k] = true
			prev[k] = v
		}
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("Unsetenv(%q): %v", k, err)
		}
	}
	t.Cleanup(func() {
		for _, k := range keys {
			if present[k] {
				_ = os.Setenv(k, prev[k])
			} else {
				_ = os.Unsetenv(k)
			}
		}
	})
}

func TestLoadMongoMetadataConfig(t *testing.T) {
	temporaryUnsetEnv(t, jobsEnvMongoKeys...)
	const (
		uri      = "mongodb://localhost"
		db       = "jobby"
		metaColl = "job_metadata"
		logsColl = "job_logs"
	)
	t.Setenv("MONGODB_URI", uri)
	t.Setenv("MONGODB_DATABASE", db)
	t.Setenv("MONGODB_COLLECTION_METADATA", metaColl)
	t.Setenv("MONGODB_COLLECTION_LOGS", logsColl)
	t.Setenv("MONGODB_TIMEOUT", "15s")
	t.Setenv("MONGODB_MAX_POOL_SIZE", "77")
	t.Setenv("MONGODB_MIN_POOL_SIZE", "3")

	got, err := loadMongoMetadataConfig()
	if err != nil {
		t.Fatalf("loadMongoMetadataConfig: %v", err)
	}
	want := metadata.MongoConfig{
		URI:                uri,
		Database:           db,
		CollectionMetadata: metaColl,
		CollectionLogs:     logsColl,
		Timeout:            15 * time.Second,
		MaxPoolSize:        77,
		MinPoolSize:        3,
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("mongo config mismatch\nwant %+v\ngot %+v", want, got)
	}
}

func TestLoadServerListenConfig(t *testing.T) {
	temporaryUnsetEnv(t, "PORT")
	t.Setenv("PORT", "9090")

	got, err := loadServerListenConfig()
	if err != nil {
		t.Fatalf("loadServerListenConfig: %v", err)
	}
	if got.Port != "9090" {
		t.Fatalf("Port: got %q", got.Port)
	}
}

func TestLoadMongoMetadataConfig_MissingRequired(t *testing.T) {
	temporaryUnsetEnv(t, jobsEnvMongoKeys...)
	t.Setenv("MONGODB_DATABASE", "db")
	t.Setenv("MONGODB_COLLECTION_METADATA", "m")
	t.Setenv("MONGODB_COLLECTION_LOGS", "l")

	_, err := loadMongoMetadataConfig()
	if err == nil {
		t.Fatal("expected error when MONGODB_URI is unset")
	}
}

func TestLoadServerListenConfig_MissingPort(t *testing.T) {
	temporaryUnsetEnv(t, "PORT")

	_, err := loadServerListenConfig()
	if err == nil {
		t.Fatal("expected error when PORT is unset")
	}
}
