package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
)

func TestGetCommand_MissingArgs(t *testing.T) {
	a := app.New(nil, nil)
	cmd := NewGetCmd(a)
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
