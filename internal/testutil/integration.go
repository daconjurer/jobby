package testutil

import (
	"os"
	"strings"
	"testing"
)

// IntegrationEnabled reports whether integration tests should run.
// INTEGRATION_TESTS must be true, 1, or yes (case-insensitive).
func IntegrationEnabled() bool {
	v := strings.TrimSpace(os.Getenv("INTEGRATION_TESTS"))
	switch strings.ToLower(v) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

// SkipUnlessIntegration skips the current test when INTEGRATION_TESTS is not truthy.
// Integration test packages call this from shared setup helpers so unit CI can run
// go test ./... without external services while integration tasks set INTEGRATION_TESTS=true.
func SkipUnlessIntegration(tb testing.TB) {
	tb.Helper()
	if !IntegrationEnabled() {
		tb.Skip("integration tests disabled (set INTEGRATION_TESTS=true to run)")
	}
}
