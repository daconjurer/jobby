package config

import (
	"os"
	"testing"
)

// mongoEnvKeys are environment variables exercised by MongoConfig tests.
var mongoEnvKeys = []string{
	EnvPrefixEnvKey,
	"MONGODB_URI",
	"MONGODB_DATABASE",
	"MONGODB_COLLECTION_METADATA",
	"MONGODB_COLLECTION_LOGS",
	"MONGODB_TIMEOUT",
	"MONGODB_MAX_POOL_SIZE",
	"MONGODB_MIN_POOL_SIZE",
	"JOBBY_MONGODB_URI",
	"JOBBY_MONGODB_DATABASE",
	"JOBBY_MONGODB_COLLECTION_METADATA",
	"JOBBY_MONGODB_COLLECTION_LOGS",
	"JOBBY_MONGODB_TIMEOUT",
	"JOBBY_MONGODB_MAX_POOL_SIZE",
	"JOBBY_MONGODB_MIN_POOL_SIZE",
	"JOBBY_PORT",
}

// temporaryUnsetEnv clears the given keys (using [os.Unsetenv]) and registers a cleanup
// that restores their previous values. Required because [testing.T.Setenv] with an empty
// string still counts as “set” for github.com/caarlos0/env’s `required` tag.
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
