package config

import (
	"strings"
	"testing"
	"time"

	"github.com/caarlos0/env/v11"
)

func TestEndToEnd_ValidationErrors(t *testing.T) {
	temporaryUnsetEnv(t, mongoEnvKeys...)
	t.Setenv("MONGODB_URI", "mongodb://x")
	t.Setenv("MONGODB_DATABASE", "db")
	t.Setenv("MONGODB_COLLECTION_METADATA", "meta")
	t.Setenv("MONGODB_COLLECTION_LOGS", "logs")
	t.Setenv("MONGODB_TIMEOUT", "10s")
	t.Setenv("MONGODB_MAX_POOL_SIZE", "5000")
	t.Setenv("MONGODB_MIN_POOL_SIZE", "5")

	var mc MongoConfig
	if err := LoadInto(&mc); err != nil {
		t.Fatalf("LoadInto: %v", err)
	}
	err := mc.Validate()
	if err == nil {
		t.Fatal("expected validation error for huge max pool")
	}
	if !strings.Contains(err.Error(), "5000") || !strings.Contains(err.Error(), "1000") {
		t.Fatalf("expected actionable limit message: %v", err)
	}
}

func TestEndToEnd_ValidConfig(t *testing.T) {
	temporaryUnsetEnv(t, mongoEnvKeys...)
	t.Setenv("MONGODB_URI", "mongodb://localhost")
	t.Setenv("MONGODB_DATABASE", "jobby")
	t.Setenv("MONGODB_COLLECTION_METADATA", "md")
	t.Setenv("MONGODB_COLLECTION_LOGS", "lg")
	t.Setenv("MONGODB_TIMEOUT", "12s")
	t.Setenv("MONGODB_MAX_POOL_SIZE", "88")
	t.Setenv("MONGODB_MIN_POOL_SIZE", "11")

	var mc MongoConfig
	if err := LoadInto(&mc); err != nil {
		t.Fatalf("LoadInto: %v", err)
	}
	if err := mc.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if mc.Timeout != 12*time.Second || mc.MaxPoolSize != 88 || mc.MinPoolSize != 11 {
		t.Fatalf("unexpected values: %+v", mc)
	}
}

func TestEndToEnd_PrefixValidConfig(t *testing.T) {
	temporaryUnsetEnv(t, mongoEnvKeys...)
	t.Setenv(EnvPrefixEnvKey, "JOBBY_")
	t.Setenv("JOBBY_MONGODB_URI", "mongodb://p")
	t.Setenv("JOBBY_MONGODB_DATABASE", "j")
	t.Setenv("JOBBY_MONGODB_COLLECTION_METADATA", "a")
	t.Setenv("JOBBY_MONGODB_COLLECTION_LOGS", "b")
	t.Setenv("JOBBY_MONGODB_TIMEOUT", "2s")
	t.Setenv("JOBBY_MONGODB_MAX_POOL_SIZE", "20")
	t.Setenv("JOBBY_MONGODB_MIN_POOL_SIZE", "10")

	var mc MongoConfig
	opts := LoadOptionsFromEnv()
	if err := LoadIntoWithOptions(&mc, opts); err != nil {
		t.Fatalf("LoadIntoWithOptions: %v", err)
	}
	if err := mc.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestEndToEnd_CustomOptionsWithoutImplicitMerge(t *testing.T) {
	temporaryUnsetEnv(t, mongoEnvKeys...)
	custom := map[string]string{
		"MONGODB_URI":                "mongodb://map",
		"MONGODB_DATABASE":           "d",
		"MONGODB_COLLECTION_METADATA": "m",
		"MONGODB_COLLECTION_LOGS":    "l",
	}
	var mc MongoConfig
	err := LoadIntoWithOptions(&mc, env.Options{Environment: custom})
	if err != nil {
		t.Fatalf("LoadIntoWithOptions: %v", err)
	}
	if mc.URI != "mongodb://map" {
		t.Fatalf("URI: %q", mc.URI)
	}
}
