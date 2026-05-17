package config

import (
	"errors"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
)

// ErrNilDestination is returned when [LoadInto] is called with a nil destination pointer.
var ErrNilDestination = errors.New("config: nil destination")

// standardConfigEnvKeys are canonical env names used by [MongoConfig] and [ServerConfig].
// When [LoadIntoWithOptions] is used with a non-empty [env.Options.Prefix], values are read
// from PREFIX+key when the canonical key is unset or empty.
var standardConfigEnvKeys = []string{
	"PORT",
	"MONGODB_URI",
	"MONGODB_DATABASE",
	"MONGODB_COLLECTION_METADATA",
	"MONGODB_COLLECTION_LOGS",
	"MONGODB_TIMEOUT",
	"MONGODB_MAX_POOL_SIZE",
	"MONGODB_MIN_POOL_SIZE",
}

// LoadInto parses the current process environment into destination using env tags on T.
func LoadInto[T any](destination *T) error {
	if destination == nil {
		return ErrNilDestination
	}
	return env.Parse(destination)
}

// LoadIntoWithOptions parses the environment into destination using opts.
// If opts.Prefix is non-empty and opts.Environment is nil, a snapshot of the process
// environment is built in which each canonical key from standardConfigEnvKeys falls back
// to os.Getenv(prefix+key) when that key is missing or empty. opts.Prefix is cleared before
// calling the underlying parser so struct tags stay canonical (PORT, MONGODB_URI, …).
func LoadIntoWithOptions[T any](destination *T, opts env.Options) error {
	if destination == nil {
		return ErrNilDestination
	}
	if opts.Prefix != "" && opts.Environment == nil {
		opts.Environment = mergePrefixedIntoCanonical(opts.Prefix)
		opts.Prefix = ""
	}
	return env.ParseWithOptions(destination, opts)
}

// EnvPrefixEnvKey is the optional process env var that enables prefixed configuration.
// When set (e.g. to "JOBBY_"), [LoadOptionsFromEnv] supplies [env.Options] so names like
// JOBBY_PORT are used when the canonical variable (PORT) is unset or empty.
const EnvPrefixEnvKey = "JOBBY_ENV_PREFIX"

// LoadOptionsFromEnv returns [env.Options] for use with [LoadIntoWithOptions]. If
// [EnvPrefixEnvKey] is non-empty, its value is used as [env.Options.Prefix] for the
// merged-environment behaviour described on [LoadIntoWithOptions].
func LoadOptionsFromEnv() env.Options {
	p := strings.TrimSpace(os.Getenv(EnvPrefixEnvKey))
	if p != "" {
		return env.Options{Prefix: p}
	}
	return env.Options{}
}

func mergePrefixedIntoCanonical(prefix string) map[string]string {
	m := make(map[string]string)
	for _, pair := range os.Environ() {
		k, v, ok := strings.Cut(pair, "=")
		if !ok {
			k, v = pair, ""
		}
		m[k] = v
	}
	for _, key := range standardConfigEnvKeys {
		if cur := m[key]; cur == "" {
			prefixed := prefix + key
			if pv, ok := m[prefixed]; ok && pv != "" {
				m[key] = pv
			}
		}
	}
	return m
}
