//go:build integration

package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/spf13/cobra"
)

func runCLICommandExpectError(t *testing.T, application *app.App, setup func(*testing.T, *app.App) *cobra.Command) error {
	t.Helper()
	var buf bytes.Buffer
	cmd := setup(t, application)
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	application.Out = &buf
	return cmd.Execute()
}

func TestIntegration_Enqueue_and_Get_roundTrip(t *testing.T) {
	application, cleanup := prepareIntegrationApp(t)
	defer cleanup()

	createOut := runCLICommand(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewCreateCmd(a)
		cmd.SetArgs([]string{
			"--name=integration-cli-job",
			"--priority=7",
			"--tags=http",
			"--tags=integration",
			"--payload={\"k\":\"v\"}",
			"--metadata={\"region\":\"eu\"}",
		})
		return cmd
	})

	var created metadata.JobMetadataModel
	if err := json.Unmarshal(createOut, &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if created.JobID == "" || created.Name != "integration-cli-job" {
		t.Fatalf("unexpected created job: %+v", created)
	}
	if created.Status != metadata.JobStatusPending {
		t.Fatalf("status=%s want pending", created.Status)
	}
	if created.Priority != 7 {
		t.Fatalf("priority=%d want 7", created.Priority)
	}

	getOut := runCLICommand(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewGetCmd(a)
		cmd.SetArgs([]string{created.JobID})
		return cmd
	})
	var got metadata.JobMetadataModel
	if err := json.Unmarshal(getOut, &got); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if got.JobID != created.JobID || got.Name != created.Name {
		t.Fatalf("get mismatch: %+v vs %+v", got, created)
	}
}

func TestIntegration_Enqueue_validation_priority(t *testing.T) {
	application, cleanup := prepareIntegrationApp(t)
	defer cleanup()

	err := runCLICommandExpectError(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewCreateCmd(a)
		cmd.SetArgs([]string{"--name=x", "--priority=11"})
		return cmd
	})
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
	if !strings.Contains(err.Error(), "priority") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntegration_Enqueue_validation_missing_name(t *testing.T) {
	application, cleanup := prepareIntegrationApp(t)
	defer cleanup()

	err := runCLICommandExpectError(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewCreateCmd(a)
		cmd.SetArgs([]string{"--payload={}"})
		return cmd
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestIntegration_Fail_cancel_retry_flow(t *testing.T) {
	application, cleanup := prepareIntegrationApp(t)
	defer cleanup()

	createOut := runCLICommand(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewCreateCmd(a)
		cmd.SetArgs([]string{"--name=flow-job"})
		return cmd
	})
	var created metadata.JobMetadataModel
	if err := json.Unmarshal(createOut, &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}

	runCLICommand(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewFailCmd(a)
		cmd.SetArgs([]string{created.JobID, "--error=boom"})
		return cmd
	})

	err := runCLICommandExpectError(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewCancelCmd(a)
		cmd.SetArgs([]string{created.JobID})
		return cmd
	})
	if err == nil {
		t.Fatal("expected error cancelling terminal job")
	}
	if !strings.Contains(err.Error(), "cannot cancel") {
		t.Fatalf("unexpected cancel error: %v", err)
	}

	runCLICommand(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewRetryCmd(a)
		cmd.SetArgs([]string{created.JobID})
		return cmd
	})

	getOut := runCLICommand(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewGetCmd(a)
		cmd.SetArgs([]string{created.JobID})
		return cmd
	})
	var got metadata.JobMetadataModel
	if err := json.Unmarshal(getOut, &got); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if got.Status != metadata.JobStatusPending {
		t.Fatalf("after retry status=%s want pending", got.Status)
	}
	if got.RetryCount < 1 {
		t.Fatalf("retryCount=%d want at least 1", got.RetryCount)
	}
}
