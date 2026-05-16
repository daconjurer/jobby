package settings

import (
	"log"
	"os"
)

func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GetEnvOrPanic(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Panicf("settings: required environment variable %q is not set or empty", key)
	}
	return value
}
