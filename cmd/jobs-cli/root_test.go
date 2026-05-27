package main

import (
	"strings"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
)

func TestNewRootCmd(t *testing.T) {
	root := newRootCmd(app.New(nil, nil))

	foundPing := false
	foundSeed := false
	for _, c := range root.Commands() {
		switch c.Name() {
		case "ping":
			foundPing = true
		case "seed":
			foundSeed = true
		}
	}
	if !foundPing {
		t.Fatal("ping subcommand not registered")
	}
	if !foundSeed {
		t.Fatal("seed subcommand not registered")
	}

	if !strings.Contains(root.Long, "MONGODB_URI") {
		t.Fatalf("root Long should mention MONGODB_URI, got: %q", root.Long)
	}
}
