package config

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/settings"
)

func TestMongoEnv_BackwardsCompatibleWithSettingsParsing(t *testing.T) {
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

	want := legacyMongoMetadataUsingSettings(t)

	var mc MongoConfig
	if err := LoadInto(&mc); err != nil {
		t.Fatalf("LoadInto MongoConfig: %v", err)
	}
	got := mongoMetadataFromConfig(mc)

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("mongo env mismatch vs legacy settings helpers\nwant %+v\ngot %+v", want, got)
	}
}

func legacyMongoMetadataUsingSettings(t *testing.T) metadata.MongoConfig {
	t.Helper()
	mustEnv := func(key string) string {
		t.Helper()
		v := os.Getenv(key)
		if v == "" {
			t.Fatalf("missing env %q", key)
		}
		return v
	}
	return metadata.MongoConfig{
		URI:                mustEnv("MONGODB_URI"),
		Database:           mustEnv("MONGODB_DATABASE"),
		CollectionMetadata: mustEnv("MONGODB_COLLECTION_METADATA"),
		CollectionLogs:     mustEnv("MONGODB_COLLECTION_LOGS"),
		Timeout:            settings.ParseDuration(settings.GetEnv("MONGODB_TIMEOUT", "10s")),
		MaxPoolSize:        settings.ParseUint64(settings.GetEnv("MONGODB_MAX_POOL_SIZE", "100")),
		MinPoolSize:        settings.ParseUint64(settings.GetEnv("MONGODB_MIN_POOL_SIZE", "10")),
	}
}

func TestMongoEnv_BackwardsCompatible_DefaultOptionalViaSettingsHelpers(t *testing.T) {
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

	want := legacyMongoMetadataUsingSettings(t)

	var mc MongoConfig
	if err := LoadInto(&mc); err != nil {
		t.Fatalf("LoadInto MongoConfig: %v", err)
	}
	got := mongoMetadataFromConfig(mc)

	if !reflect.DeepEqual(want, got) {
		t.Fatalf("defaults mismatch\nwant %+v\ngot %+v", want, got)
	}
	if got.Timeout != 10*time.Second || got.MaxPoolSize != 100 || got.MinPoolSize != 10 {
		t.Fatalf("expected embedded defaults from tags / legacy helpers, got %+v", got)
	}
}

func TestServerEnv_BackwardsCompatibleWithExplicitPort(t *testing.T) {
	temporaryUnsetEnv(t, "PORT")
	const wantPort = "8085"
	t.Setenv("PORT", wantPort)

	var sc ServerConfig
	if err := LoadInto(&sc); err != nil {
		t.Fatalf("LoadInto ServerConfig: %v", err)
	}
	if sc.Port != wantPort {
		t.Fatalf("Port: got %q want %q", sc.Port, wantPort)
	}
	if os.Getenv("PORT") != wantPort {
		t.Fatalf("expected PORT in environment")
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
