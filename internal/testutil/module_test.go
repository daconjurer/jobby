package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestModuleRoot_FindsGoMod(t *testing.T) {
	root := ModuleRoot(t)
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("go.mod missing under %q: %v", root, err)
	}
	if _, err := os.Stat(filepath.Join(root, "config", "job-topics.yaml")); err != nil {
		t.Fatalf("config/job-topics.yaml missing under %q: %v", root, err)
	}
}

func TestJobTopicsConfigPath_Default(t *testing.T) {
	got := JobTopicsConfigPath(t)
	want := filepath.Join(ModuleRoot(t), "config", "job-topics.yaml")
	if got != want {
		t.Fatalf("path=%q want %q", got, want)
	}
}

func TestJobTopicsConfigPath_RelativeEnv(t *testing.T) {
	t.Setenv("JOB_TOPICS_CONFIG_PATH", "config/job-topics.yaml")
	got := JobTopicsConfigPath(t)
	want := filepath.Join(ModuleRoot(t), "config", "job-topics.yaml")
	if got != want {
		t.Fatalf("path=%q want %q", got, want)
	}
}

func TestJobTopicsConfigPath_AbsoluteEnv(t *testing.T) {
	abs := filepath.Join(ModuleRoot(t), "config", "job-topics.yaml")
	t.Setenv("JOB_TOPICS_CONFIG_PATH", abs)
	if got := JobTopicsConfigPath(t); got != abs {
		t.Fatalf("path=%q want %q", got, abs)
	}
}
