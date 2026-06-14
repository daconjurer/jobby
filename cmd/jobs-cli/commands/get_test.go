package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/cli"
)

func TestGetCommand_MissingArgs(t *testing.T) {
	c := cli.New(nil, nil, nil)
	cmd := NewGetCmd(c)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when jobId arg is missing")
	}
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Fatalf("unexpected error: %v", err)
	}
}
