// Package settings provides basic environment variable access.
//
// Deprecated: Use internal/config with github.com/caarlos0/env/v11 instead.
// This package will be removed in a future version.
package settings

import (
	"log"
	"os"
)

// GetEnv returns the value of the environment variable named key when it is non-empty, otherwise defaultValue.
//
// Deprecated: Use internal/config instead.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvOrPanic returns the non-empty environment variable value for key, or panics.
//
// Deprecated: Use internal/config.LoadInto instead.
func GetEnvOrPanic(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Panicf("settings: required environment variable %q is not set or empty", key)
	}
	return value
}
