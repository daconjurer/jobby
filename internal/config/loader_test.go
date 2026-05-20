package config

import (
	"errors"
	"testing"
	"time"
)

func TestLoadInto_Mongo_AllRequiredPresent(t *testing.T) {
	temporaryUnsetEnv(t, mongoEnvKeys...)
	const (
		wantURI             = "mongodb://localhost"
		wantDB              = "jobby"
		wantMetaColl        = "metadata"
		wantLogsColl        = "logs"
		wantDuration        = 5 * time.Second
		wantMax      uint64 = 50
		wantMin      uint64 = 12
	)
	t.Setenv("MONGODB_URI", wantURI)
	t.Setenv("MONGODB_DATABASE", wantDB)
	t.Setenv("MONGODB_COLLECTION_METADATA", wantMetaColl)
	t.Setenv("MONGODB_COLLECTION_LOGS", wantLogsColl)
	t.Setenv("MONGODB_TIMEOUT", "5s")
	t.Setenv("MONGODB_MAX_POOL_SIZE", "50")
	t.Setenv("MONGODB_MIN_POOL_SIZE", "12")

	var cfg MongoConfig
	if err := LoadInto(&cfg); err != nil {
		t.Fatalf("LoadInto: %v", err)
	}
	if cfg.URI != wantURI || cfg.Database != wantDB ||
		cfg.CollectionMetadata != wantMetaColl || cfg.CollectionLogs != wantLogsColl {
		t.Fatalf("unexpected mongodb strings: %+v", cfg)
	}
	if cfg.Timeout != wantDuration {
		t.Fatalf("Timeout: got %v want %v", cfg.Timeout, wantDuration)
	}
	if cfg.MaxPoolSize != wantMax || cfg.MinPoolSize != wantMin {
		t.Fatalf("pool sizes: got max=%d min=%d want max=%d min=%d",
			cfg.MaxPoolSize, cfg.MinPoolSize, wantMax, wantMin)
	}
}

func TestLoadInto_Mongo_MissingRequired(t *testing.T) {
	temporaryUnsetEnv(t, mongoEnvKeys...)
	t.Setenv("MONGODB_DATABASE", "db")
	t.Setenv("MONGODB_COLLECTION_METADATA", "m")
	t.Setenv("MONGODB_COLLECTION_LOGS", "l")
	// MONGODB_URI still unset after temporaryUnsetEnv

	var cfg MongoConfig
	err := LoadInto(&cfg)
	if err == nil {
		t.Fatal("expected error when MONGODB_URI is missing")
	}
	t.Log(err)
}

func TestLoadInto_Mongo_EmptyRequired(t *testing.T) {
	temporaryUnsetEnv(t, mongoEnvKeys...)
	t.Setenv("MONGODB_URI", "")
	t.Setenv("MONGODB_DATABASE", "db")
	t.Setenv("MONGODB_COLLECTION_METADATA", "m")
	t.Setenv("MONGODB_COLLECTION_LOGS", "l")

	var cfg MongoConfig
	err := LoadInto(&cfg)
	if err == nil {
		t.Fatal("expected error when MONGODB_URI is empty")
	}
	t.Log(err)
}

func TestLoadInto_Mongo_DefaultValues(t *testing.T) {
	temporaryUnsetEnv(t, mongoEnvKeys...)
	t.Setenv("MONGODB_URI", "mongodb://x")
	t.Setenv("MONGODB_DATABASE", "jobby")
	t.Setenv("MONGODB_COLLECTION_METADATA", "meta")
	t.Setenv("MONGODB_COLLECTION_LOGS", "logs")

	var cfg MongoConfig
	if err := LoadInto(&cfg); err != nil {
		t.Fatalf("LoadInto: %v", err)
	}
	if cfg.Timeout != 10*time.Second {
		t.Fatalf("Timeout default: got %v", cfg.Timeout)
	}
	const wantMax uint64 = 100
	const wantMin uint64 = 10
	if cfg.MaxPoolSize != wantMax || cfg.MinPoolSize != wantMin {
		t.Fatalf("pool defaults: got max=%d min=%d", cfg.MaxPoolSize, cfg.MinPoolSize)
	}
}

func TestLoadInto_Mongo_InvalidDuration(t *testing.T) {
	temporaryUnsetEnv(t, mongoEnvKeys...)
	t.Setenv("MONGODB_URI", "mongodb://x")
	t.Setenv("MONGODB_DATABASE", "db")
	t.Setenv("MONGODB_COLLECTION_METADATA", "m")
	t.Setenv("MONGODB_COLLECTION_LOGS", "l")
	t.Setenv("MONGODB_TIMEOUT", "not-a-duration")

	var cfg MongoConfig
	err := LoadInto(&cfg)
	if err == nil {
		t.Fatal("expected error for invalid MONGODB_TIMEOUT")
	}
	t.Log(err)
}

func TestLoadInto_Mongo_InvalidUint64(t *testing.T) {
	temporaryUnsetEnv(t, mongoEnvKeys...)
	t.Setenv("MONGODB_URI", "mongodb://x")
	t.Setenv("MONGODB_DATABASE", "db")
	t.Setenv("MONGODB_COLLECTION_METADATA", "m")
	t.Setenv("MONGODB_COLLECTION_LOGS", "l")
	t.Setenv("MONGODB_MAX_POOL_SIZE", "oops")

	var cfg MongoConfig
	err := LoadInto(&cfg)
	if err == nil {
		t.Fatal("expected error for invalid MONGODB_MAX_POOL_SIZE")
	}
	t.Log(err)
}

func TestLoadInto_Server_HappyPath(t *testing.T) {
	temporaryUnsetEnv(t, "APP_PORT")
	t.Setenv("APP_PORT", "8080")

	var cfg ServerConfig
	if err := LoadInto(&cfg); err != nil {
		t.Fatalf("LoadInto: %v", err)
	}
	if cfg.Port != "8080" {
		t.Fatalf("Port: got %q", cfg.Port)
	}
}

func TestLoadInto_Server_MissingRequired(t *testing.T) {
	temporaryUnsetEnv(t, "APP_PORT")

	var cfg ServerConfig
	err := LoadInto(&cfg)
	if err == nil {
		t.Fatal("expected error when APP_PORT missing")
	}
	t.Log(err)
}

func TestLoadInto_Server_EmptyPort(t *testing.T) {
	temporaryUnsetEnv(t, "APP_PORT")
	t.Setenv("APP_PORT", "")

	var cfg ServerConfig
	err := LoadInto(&cfg)
	if err == nil {
		t.Fatal("expected error when APP_PORT is empty")
	}
	t.Log(err)
}

func TestLoadInto_NilDestination(t *testing.T) {
	err := LoadInto((*MongoConfig)(nil))
	if !errors.Is(err, ErrNilDestination) {
		t.Fatalf("got %v want ErrNilDestination", err)
	}
}

var migrateEnvKeys = []string{"MONGO_URI", "MIGRATIONS_PATH"}

func TestLoadInto_Migrate_HappyPath(t *testing.T) {
	temporaryUnsetEnv(t, migrateEnvKeys...)
	const wantURI = "mongodb://admin@localhost/jobby"
	const wantPath = "./migrations"
	t.Setenv("MONGO_URI", wantURI)
	t.Setenv("MIGRATIONS_PATH", wantPath)

	var cfg MigrateConfig
	if err := LoadInto(&cfg); err != nil {
		t.Fatalf("LoadInto: %v", err)
	}
	if cfg.URI != wantURI || cfg.MigrationsPath != wantPath {
		t.Fatalf("unexpected migrate config: %+v", cfg)
	}
}

func TestLoadInto_Migrate_DefaultMigrationsPath(t *testing.T) {
	temporaryUnsetEnv(t, migrateEnvKeys...)
	t.Setenv("MONGO_URI", "mongodb://admin@localhost/jobby")

	var cfg MigrateConfig
	if err := LoadInto(&cfg); err != nil {
		t.Fatalf("LoadInto: %v", err)
	}
	if cfg.MigrationsPath != "./migrations" {
		t.Fatalf("MigrationsPath default: got %q want ./migrations", cfg.MigrationsPath)
	}
}

func TestLoadInto_Migrate_MissingURI(t *testing.T) {
	temporaryUnsetEnv(t, migrateEnvKeys...)

	var cfg MigrateConfig
	err := LoadInto(&cfg)
	if err == nil {
		t.Fatal("expected error when MONGO_URI is missing")
	}
	t.Log(err)
}

func TestLoadInto_Migrate_EmptyURI(t *testing.T) {
	temporaryUnsetEnv(t, migrateEnvKeys...)
	t.Setenv("MONGO_URI", "")

	var cfg MigrateConfig
	err := LoadInto(&cfg)
	if err == nil {
		t.Fatal("expected error when MONGO_URI is empty")
	}
	t.Log(err)
}
