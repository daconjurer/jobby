package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
)

func TestNewRootCmd(t *testing.T) {
	root := newRootCmd(app.New(nil, nil))

	want := map[string]bool{
		"ping":   false,
		"create": false,
		"get":    false,
		"list":   false,
		"stats":  false,
		"fail":   false,
		"cancel": false,
		"retry":  false,
		"logs":   false,
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

	if !strings.Contains(root.Long, "MONGODB_URI") {
		t.Fatalf("root Long should mention MONGODB_URI, got: %q", root.Long)
	}
	if !strings.Contains(root.Long, "--output") {
		t.Fatalf("root Long should mention --output, got: %q", root.Long)
	}
}

func TestRootCommand_Help(t *testing.T) {
	root := newRootCmd(app.New(nil, nil))
	root.SetArgs([]string{"--help"})

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	if err := root.Execute(); err != nil {
		t.Fatalf("help: %v", err)
	}

	help := buf.String()
	for _, cmd := range []string{
		"ping", "create", "get", "list", "stats",
		"fail", "cancel", "retry", "logs", "seed",
	} {
		if !strings.Contains(help, cmd) {
			t.Fatalf("help missing subcommand %q", cmd)
		}
	}
	if !strings.Contains(help, "--output") {
		t.Fatal("help missing --output flag")
	}
}
