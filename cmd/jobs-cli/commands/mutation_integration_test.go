package commands

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/cli"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/spf13/cobra"
)

func TestIntegration_Enqueue_and_Get_roundTrip(t *testing.T) {
	c, cleanup := prepareIntegrationCLI(t)
	defer cleanup()

	createOut := runCLICommand(t, c, func(t *testing.T, c *cli.CLI) *cobra.Command {
		cmd := NewCreateCmd(c)
		cmd.SetArgs([]string{
			"--name=account-lifecycle",
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
	if created.JobID == "" || created.Name != "account-lifecycle" {
		t.Fatalf("unexpected created job: %+v", created)
	}
	if created.Topic != "persistent://public/default/accounts/jobs" {
		t.Fatalf("topic=%q", created.Topic)
	}
	if created.Status != metadata.JobStatusPendingDispatch {
		t.Fatalf("status=%s want pending_dispatch", created.Status)
	}
	if created.Priority != 7 {
		t.Fatalf("priority=%d want 7", created.Priority)
	}

	getOut := runCLICommand(t, c, func(t *testing.T, c *cli.CLI) *cobra.Command {
		cmd := NewGetCmd(c)
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
	c, cleanup := prepareIntegrationCLI(t)
	defer cleanup()

	err := runCLICommandExpectError(t, c, func(t *testing.T, c *cli.CLI) *cobra.Command {
		cmd := NewCreateCmd(c)
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
	c, cleanup := prepareIntegrationCLI(t)
	defer cleanup()

	err := runCLICommandExpectError(t, c, func(t *testing.T, c *cli.CLI) *cobra.Command {
		cmd := NewCreateCmd(c)
		cmd.SetArgs([]string{"--payload={}"})
		return cmd
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestIntegration_Enqueue_validation_unknown_job_name(t *testing.T) {
	c, cleanup := prepareIntegrationCLI(t)
	defer cleanup()

	err := runCLICommandExpectError(t, c, func(t *testing.T, c *cli.CLI) *cobra.Command {
		cmd := NewCreateCmd(c)
		cmd.SetArgs([]string{"--name=not-a-real-job-type"})
		return cmd
	})
	if err == nil {
		t.Fatal("expected error for unknown job name")
	}
	if !strings.Contains(err.Error(), "unknown job type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntegration_Fail_cancel_retry_flow(t *testing.T) {
	c, cleanup := prepareIntegrationCLI(t)
	defer cleanup()

	createOut := runCLICommand(t, c, func(t *testing.T, c *cli.CLI) *cobra.Command {
		cmd := NewCreateCmd(c)
		cmd.SetArgs([]string{"--name=account-lifecycle"})
		return cmd
	})
	var created metadata.JobMetadataModel
	if err := json.Unmarshal(createOut, &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}

	markJobRunningForTest(t, c, created.JobID)

	runCLICommand(t, c, func(t *testing.T, c *cli.CLI) *cobra.Command {
		cmd := NewFailCmd(c)
		cmd.SetArgs([]string{created.JobID, "--error=boom"})
		return cmd
	})

	err := runCLICommandExpectError(t, c, func(t *testing.T, c *cli.CLI) *cobra.Command {
		cmd := NewCancelCmd(c)
		cmd.SetArgs([]string{created.JobID})
		return cmd
	})
	if err == nil {
		t.Fatal("expected error cancelling terminal job")
	}
	if !strings.Contains(err.Error(), "cannot cancel") {
		t.Fatalf("unexpected cancel error: %v", err)
	}

	runCLICommand(t, c, func(t *testing.T, c *cli.CLI) *cobra.Command {
		cmd := NewRetryCmd(c)
		cmd.SetArgs([]string{created.JobID})
		return cmd
	})

	getOut := runCLICommand(t, c, func(t *testing.T, c *cli.CLI) *cobra.Command {
		cmd := NewGetCmd(c)
		cmd.SetArgs([]string{created.JobID})
		return cmd
	})
	var got metadata.JobMetadataModel
	if err := json.Unmarshal(getOut, &got); err != nil {
		t.Fatalf("decode get: %v", err)
	}
	if got.Status != metadata.JobStatusPendingDispatch {
		t.Fatalf("after retry status=%s want pending_dispatch", got.Status)
	}
	if got.RetryCount < 1 {
		t.Fatalf("retryCount=%d want at least 1", got.RetryCount)
	}
}
