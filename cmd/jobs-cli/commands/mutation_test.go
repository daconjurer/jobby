package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/cli"
)

func TestCreateCommand_ValidationPriority(t *testing.T) {
	c := cli.New(nil, nil, nil)
	cmd := NewCreateCmd(c)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--name", "x", "--priority", "11"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
	if err.Error() != "priority must be between 0 and 10" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateCommand_MissingName(t *testing.T) {
	c := cli.New(nil, nil, nil)
	cmd := NewCreateCmd(c)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --name is missing")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateCommand_PayloadJSON(t *testing.T) {
	c := cli.New(nil, nil, nil)
	cmd := NewCreateCmd(c)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--name", "x", "--payload", "not-json"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid payload JSON")
	}
	if !strings.Contains(err.Error(), "invalid payload JSON") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFailCommand_MissingError(t *testing.T) {
	c := cli.New(nil, nil, nil)
	cmd := NewFailCmd(c)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"00000000-0000-0000-0000-000000000001"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --error is missing")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCancelCommand_OptionalReason(t *testing.T) {
	cmd := NewCancelCmd(cli.New(nil, nil, nil))
	reasonFlag := cmd.Flags().Lookup("reason")
	if reasonFlag == nil {
		t.Fatal("reason flag not defined")
	}
	if reasonFlag.DefValue != "" {
		t.Fatalf("reason default = %q, want empty", reasonFlag.DefValue)
	}
}
