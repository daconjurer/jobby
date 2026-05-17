package config

import (
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

func TestMongoEnv_ParseMatchesReferenceFromOSEnv(t *testing.T) {
	temporaryUnsetEnv(t, mongoEnvKeys...)
	const (
		uri      = "mongodb://example"
		db       = "jobs_db"
		metaColl = "job_metadata"
		logsColl = "job_logs"
	)
	t.Setenv("MONGODB_URI", uri)
	t.Setenv("MONGODB_DATABASE", db)
	t.Setenv("MONGODB_COLLECTION_METADATA", metaColl)
	t.Setenv("MONGODB_COLLECTION_LOGS", logsColl)
	t.Setenv("MONGODB_TIMEOUT", "8s")
	t.Setenv("MONGODB_MAX_POOL_SIZE", "42")
	t.Setenv("MONGODB_MIN_POOL_SIZE", "7")

	want := referenceMongoMetadataFromOSEnv(t)

	var mc MongoConfig
	if err := LoadInto(&mc); err != nil {
		t.Fatalf("LoadInto MongoConfig: %v", err)
	}
	got := mongoMetadataFromConfig(mc)

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("mongo env mismatch vs reference from os.Getenv\nwant %+v\ngot %+v", want, got)
	}
}

func TestMongoEnv_DefaultOptionalMatchesTagDefaults(t *testing.T) {
	temporaryUnsetEnv(t, mongoEnvKeys...)
	const (
		uri      = "mongodb://example"
		db       = "jobs_db"
		metaColl = "job_metadata"
		logsColl = "job_logs"
	)
	t.Setenv("MONGODB_URI", uri)
	t.Setenv("MONGODB_DATABASE", db)
	t.Setenv("MONGODB_COLLECTION_METADATA", metaColl)
	t.Setenv("MONGODB_COLLECTION_LOGS", logsColl)

	want := referenceMongoMetadataFromOSEnv(t)

	var mc MongoConfig
	if err := LoadInto(&mc); err != nil {
		t.Fatalf("LoadInto MongoConfig: %v", err)
	}
	got := mongoMetadataFromConfig(mc)

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("defaults mismatch\nwant %+v\ngot %+v", want, got)
	}
	if got.Timeout != 10*time.Second || got.MaxPoolSize != 100 || got.MinPoolSize != 10 {
		t.Fatalf("expected embedded defaults from tags / reference helpers, got %+v", got)
	}
}

func TestServerEnv_WithExplicitAppPort(t *testing.T) {
	temporaryUnsetEnv(t, "APP_PORT")
	const wantPort = "8085"
	t.Setenv("APP_PORT", wantPort)

	var sc ServerConfig
	if err := LoadInto(&sc); err != nil {
		t.Fatalf("LoadInto ServerConfig: %v", err)
	}
	if sc.Port != wantPort {
		t.Fatalf("Port: got %q want %q", sc.Port, wantPort)
	}
	if err := sc.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if os.Getenv("APP_PORT") != wantPort {
		t.Fatalf("expected APP_PORT in environment")
	}
}

func referenceMongoMetadataFromOSEnv(t *testing.T) metadata.MongoConfig {
	t.Helper()
	mustEnv := func(key string) string {
		t.Helper()
		v := os.Getenv(key)
		if v == "" {
			t.Fatalf("missing env %q", key)
		}
		return v
	}
	timeoutStr := os.Getenv("MONGODB_TIMEOUT")
	if timeoutStr == "" {
		timeoutStr = "10s"
	}
	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		t.Fatalf("MONGODB_TIMEOUT: %v", err)
	}
	maxStr := os.Getenv("MONGODB_MAX_POOL_SIZE")
	if maxStr == "" {
		maxStr = "100"
	}
	maxPool, err := strconv.ParseUint(maxStr, 10, 64)
	if err != nil {
		t.Fatalf("MONGODB_MAX_POOL_SIZE: %v", err)
	}
	minStr := os.Getenv("MONGODB_MIN_POOL_SIZE")
	if minStr == "" {
		minStr = "10"
	}
	minPool, err := strconv.ParseUint(minStr, 10, 64)
	if err != nil {
		t.Fatalf("MONGODB_MIN_POOL_SIZE: %v", err)
	}
	return metadata.MongoConfig{
		URI:                mustEnv("MONGODB_URI"),
		Database:           mustEnv("MONGODB_DATABASE"),
		CollectionMetadata: mustEnv("MONGODB_COLLECTION_METADATA"),
		CollectionLogs:     mustEnv("MONGODB_COLLECTION_LOGS"),
		Timeout:            timeout,
		MaxPoolSize:        maxPool,
		MinPoolSize:        minPool,
	}
}

func mongoMetadataFromConfig(mc MongoConfig) metadata.MongoConfig {
	return metadata.MongoConfig{
		URI:                mc.URI,
		Database:           mc.Database,
		CollectionMetadata: mc.CollectionMetadata,
		CollectionLogs:     mc.CollectionLogs,
		Timeout:            mc.Timeout,
		MaxPoolSize:        mc.MaxPoolSize,
		MinPoolSize:        mc.MinPoolSize,
	}
}
