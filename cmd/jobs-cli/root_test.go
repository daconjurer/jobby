package main

import (
	"strings"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
)

func TestNewRootCmd(t *testing.T) {
	root := newRootCmd(app.New(nil))

	foundPing := false
	for _, c := range root.Commands() {
		if c.Name() == "ping" {
			foundPing = true
			break
		}
	}
	if !foundPing {
		t.Fatal("ping subcommand not registered")
	}

	if !strings.Contains(root.Long, "MONGODB_URI") {
		t.Fatalf("root Long should mention MONGODB_URI, got: %q", root.Long)
	}
}
