package pulsar

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/daconjurer/jobby/internal/testutil"
)

const (
	testJobAccountLifecycle = "account-lifecycle"
	testTopicAccountsJobs   = "persistent://public/default/accounts/jobs"
	testJobOptimisation     = "optimisation"
	testTopicRndJobs        = "persistent://public/default/rnd/jobs"
)

func TestFileTopicResolver_KnownAndUnknown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "topics.yaml")
	if err := os.WriteFile(path, []byte(`
topics:
  account-lifecycle:
    topic: persistent://public/default/accounts/jobs
    domain: customer-management
`), 0o600); err != nil {
		t.Fatal(err)
	}

	r, err := NewFileTopicResolver(path)
	if err != nil {
		t.Fatal(err)
	}

	topic, err := r.Resolve(testJobAccountLifecycle)
	if err != nil {
		t.Fatal(err)
	}
	if topic != testTopicAccountsJobs {
		t.Fatalf("topic=%q", topic)
	}

	_, err = r.Resolve("missing-job")
	if err == nil {
		t.Fatal("expected error for unknown job")
	}
	if !errors.Is(err, ErrUnknownJobType) {
		t.Fatalf("err=%v want ErrUnknownJobType", err)
	}
}

func TestFileTopicResolver_MissingDomain(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "topics.yaml")
	if err := os.WriteFile(path, []byte(`
topics:
  account-lifecycle:
    topic: persistent://public/default/accounts/jobs
`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := NewFileTopicResolver(path)
	if err == nil {
		t.Fatal("expected domain validation error")
	}
}

func TestFileTopicResolver_BundledConfig(t *testing.T) {
	r, err := NewFileTopicResolver(testutil.JobTopicsConfigPath(t))
	if err != nil {
		t.Fatal(err)
	}

	topic, err := r.Resolve(testJobAccountLifecycle)
	if err != nil {
		t.Fatal(err)
	}
	if topic != testTopicAccountsJobs {
		t.Fatalf("account-lifecycle topic=%q want %q", topic, testTopicAccountsJobs)
	}

	topic, err = r.Resolve(testJobOptimisation)
	if err != nil {
		t.Fatal(err)
	}
	if topic != testTopicRndJobs {
		t.Fatalf("optimisation topic=%q want %q", topic, testTopicRndJobs)
	}
}

func TestFileTopicResolver_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(":::not yaml"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := NewFileTopicResolver(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
}
