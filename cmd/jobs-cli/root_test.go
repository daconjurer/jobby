package main

import (
	"strings"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
)

func TestNewRootCmd(t *testing.T) {
	root := newRootCmd(app.New(nil, nil))

	want := map[string]bool{
		"ping":   false,
		"create": false,
		"fail":   false,
		"cancel": false,
		"retry":  false,
		"seed":   false,
	}
	for _, c := range root.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Fatalf("%q subcommand not registered", name)
		}
	}
	if !foundSeed {
		t.Fatal("seed subcommand not registered")
	}

	if !strings.Contains(root.Long, "MONGODB_URI") {
		t.Fatalf("root Long should mention MONGODB_URI, got: %q", root.Long)
	}
}
