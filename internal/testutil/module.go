package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

// ModuleRoot returns the directory containing go.mod by walking up from the process
// working directory. Fails the test if no module root is found.
func ModuleRoot(tb testing.TB) string {
	tb.Helper()
	dir, err := os.Getwd()
	if err != nil {
		tb.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			tb.Fatal("module root not found (no go.mod in parent chain)")
		}
		dir = parent
	}
}

// JobTopicsConfigPath returns the path to config/job-topics.yaml for tests.
//
// Resolution order:
//  1. JOB_TOPICS_CONFIG_PATH if set — absolute paths are used as-is; relative paths
//     are resolved against the module root (same semantics as production defaults).
//  2. Otherwise filepath.Join(moduleRoot, "config", "job-topics.yaml").
func JobTopicsConfigPath(tb testing.TB) string {
	tb.Helper()
	if p := os.Getenv("JOB_TOPICS_CONFIG_PATH"); p != "" {
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(ModuleRoot(tb), p)
	}
	return filepath.Join(ModuleRoot(tb), "config", "job-topics.yaml")
}
