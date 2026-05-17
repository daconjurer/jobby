package config

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/caarlos0/env/v11"
)

func TestMongoConfig_Validate_MaxLessThanMin(t *testing.T) {
	c := MongoConfig{
		MaxPoolSize: 5,
		MinPoolSize: 20,
		Timeout:     2 * time.Second,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "MONGODB_MAX_POOL_SIZE (5)") || !strings.Contains(err.Error(), "MONGODB_MIN_POOL_SIZE (20)") {
		t.Fatalf("error should mention both values: %v", err)
	}
}

func TestMongoConfig_Validate_TimeoutTooSmall(t *testing.T) {
	c := MongoConfig{
		MaxPoolSize: 100,
		MinPoolSize: 10,
		Timeout:     500 * time.Millisecond,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "MONGODB_TIMEOUT") || !strings.Contains(err.Error(), "1s") {
		t.Fatalf("expected minimum timeout hint: %v", err)
	}
}

func TestMongoConfig_Validate_MaxPoolSizeTooLarge(t *testing.T) {
	c := MongoConfig{
		MaxPoolSize: 2000,
		MinPoolSize: 10,
		Timeout:     5 * time.Second,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "2000") || !strings.Contains(err.Error(), "1000") {
		t.Fatalf("error should mention configured and max limit: %v", err)
	}
}

func TestServerConfig_Validate_InvalidPort(t *testing.T) {
	c := ServerConfig{Port: "not-a-number"}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "PORT") || !strings.Contains(err.Error(), "not-a-number") {
		t.Fatalf("expected invalid port detail: %v", err)
	}
}

func TestServerConfig_Validate_PortOutOfRange(t *testing.T) {
	low := ServerConfig{Port: "80"}
	if err := low.Validate(); err == nil || !strings.Contains(err.Error(), "1024") {
		t.Fatalf("expected privileged port error, got %v", err)
	}
	high := ServerConfig{Port: "65536"}
	if err := high.Validate(); err == nil || !strings.Contains(err.Error(), "65535") {
		t.Fatalf("expected high port error, got %v", err)
	}
}

func TestMongoConfig_Validate_MultipleViolations(t *testing.T) {
	c := MongoConfig{
		MaxPoolSize: 2000,
		MinPoolSize: 5000,
		Timeout:     time.Millisecond,
	}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected joined errors")
	}
	// Ordering follows Validate(): pool ordering, then timeout, then max cap.
	if !strings.Contains(err.Error(), "MONGODB_MAX_POOL_SIZE") || !strings.Contains(err.Error(), "MONGODB_TIMEOUT") {
		t.Fatalf("expected multiple constraint messages: %v", err)
	}
	var joined interface{ Unwrap() []error }
	if !errors.As(err, &joined) {
		t.Fatalf("expected errors.Join-style wrapper, got %T", err)
	}
	unw := joined.Unwrap()
	if len(unw) < 2 {
		t.Fatalf("expected at least 2 wrapped errors, got %d (%v)", len(unw), err)
	}
}

func TestLoadIntoThenValidate_ReportsValidationNotParse(t *testing.T) {
	temporaryUnsetEnv(t, mongoEnvKeys...)
	t.Setenv("MONGODB_URI", "mongodb://x")
	t.Setenv("MONGODB_DATABASE", "db")
	t.Setenv("MONGODB_COLLECTION_METADATA", "m")
	t.Setenv("MONGODB_COLLECTION_LOGS", "l")
	t.Setenv("MONGODB_MAX_POOL_SIZE", "5")
	t.Setenv("MONGODB_MIN_POOL_SIZE", "50")
	t.Setenv("MONGODB_TIMEOUT", "5s")

	var mc MongoConfig
	if err := LoadInto(&mc); err != nil {
		t.Fatalf("LoadInto: %v", err)
	}
	err := mc.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "MONGODB_MAX_POOL_SIZE") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadIntoWithOptions_NilDestination(t *testing.T) {
	err := LoadIntoWithOptions((*MongoConfig)(nil), env.Options{})
	if !errors.Is(err, ErrNilDestination) {
		t.Fatalf("got %v want ErrNilDestination", err)
	}
}

func TestLoadIntoWithOptions_PrefixedFallback(t *testing.T) {
	temporaryUnsetEnv(t, mongoEnvKeys...)
	t.Setenv("JOBBY_MONGODB_URI", "mongodb://pref")
	t.Setenv("JOBBY_MONGODB_DATABASE", "d")
	t.Setenv("JOBBY_MONGODB_COLLECTION_METADATA", "meta")
	t.Setenv("JOBBY_MONGODB_COLLECTION_LOGS", "logs")
	t.Setenv("JOBBY_MONGODB_TIMEOUT", "4s")
	t.Setenv("JOBBY_MONGODB_MAX_POOL_SIZE", "30")
	t.Setenv("JOBBY_MONGODB_MIN_POOL_SIZE", "5")

	var mc MongoConfig
	if err := LoadIntoWithOptions(&mc, env.Options{Prefix: "JOBBY_"}); err != nil {
		t.Fatalf("LoadIntoWithOptions: %v", err)
	}
	if mc.URI != "mongodb://pref" || mc.Timeout != 4*time.Second || mc.MaxPoolSize != 30 {
		t.Fatalf("unexpected config: %+v", mc)
	}
	if err := mc.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestLoadIntoWithOptions_CanonicalWinsOverPrefix(t *testing.T) {
	temporaryUnsetEnv(t, mongoEnvKeys...)
	t.Setenv("MONGODB_URI", "mongodb://canonical")
	t.Setenv("JOBBY_MONGODB_URI", "mongodb://shadow")
	t.Setenv("MONGODB_DATABASE", "d")
	t.Setenv("MONGODB_COLLECTION_METADATA", "meta")
	t.Setenv("MONGODB_COLLECTION_LOGS", "logs")

	var mc MongoConfig
	if err := LoadIntoWithOptions(&mc, env.Options{Prefix: "JOBBY_"}); err != nil {
		t.Fatalf("LoadIntoWithOptions: %v", err)
	}
	if mc.URI != "mongodb://canonical" {
		t.Fatalf("canonical URI should win, got %q", mc.URI)
	}
}

func TestLoadOptionsFromEnv_AppliesPrefix(t *testing.T) {
	temporaryUnsetEnv(t, mongoEnvKeys...)
	t.Setenv(EnvPrefixEnvKey, "JOBBY_")
	t.Setenv("JOBBY_PORT", "8080")

	var sc ServerConfig
	opts := LoadOptionsFromEnv()
	if err := LoadIntoWithOptions(&sc, opts); err != nil {
		t.Fatalf("LoadIntoWithOptions: %v", err)
	}
	if sc.Port != "8080" {
		t.Fatalf("Port: got %q", sc.Port)
	}
}
