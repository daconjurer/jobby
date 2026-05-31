//go:build integration

package commands

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/spf13/cobra"
)

// CLI integration parity with internal/jobs/handler/jobs_handler_integration_test.go:
//
//   - ping / health                    → TestIntegration_Ping
//   - create + validation              → TestIntegration_Enqueue_*
//   - get + not found                  → TestIntegration_Get_not_found
//   - list + filters + stats           → TestIntegration_List_*
//   - fail + cancel + retry flow       → TestIntegration_Fail_cancel_retry_flow, TestIntegration_Cancel_pending_job, TestIntegration_Retry_non_failed_job, TestIntegration_Fail_not_found
//   - logs + invalid levels            → TestIntegration_Logs_levels
//   - seed (dev only)                  → TestIntegration_Seed_inserts_jobs_and_logs

func TestIntegration_Cancel_pending_job(t *testing.T) {
	application, cleanup := prepareIntegrationApp(t)
	defer cleanup()

	createOut := runCLICommand(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewCreateCmd(a)
		cmd.SetArgs([]string{"--name=cancel-me"})
		return cmd
	})
	var created metadata.JobMetadataModel
	if err := json.Unmarshal(createOut, &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}

	runCLICommand(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewCancelCmd(a)
		cmd.SetArgs([]string{created.JobID, "--reason=ops hold"})
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
	if got.Status != metadata.JobStatusCancelled {
		t.Fatalf("status=%s want cancelled", got.Status)
	}
}

func TestIntegration_Retry_non_failed_job(t *testing.T) {
	application, cleanup := prepareIntegrationApp(t)
	defer cleanup()

	createOut := runCLICommand(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewCreateCmd(a)
		cmd.SetArgs([]string{"--name=still-pending"})
		return cmd
	})
	var created metadata.JobMetadataModel
	if err := json.Unmarshal(createOut, &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}

	err := runCLICommandExpectError(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewRetryCmd(a)
		cmd.SetArgs([]string{created.JobID})
		return cmd
	})
	if err == nil {
		t.Fatal("expected error retrying non-failed job")
	}
	if !strings.Contains(err.Error(), "only failed jobs can be retried") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntegration_Fail_not_found(t *testing.T) {
	application, cleanup := prepareIntegrationApp(t)
	defer cleanup()

	err := runCLICommandExpectError(t, application, func(t *testing.T, a *app.App) *cobra.Command {
		cmd := NewFailCmd(a)
		cmd.SetArgs([]string{metadata.GenerateJobID(), "--error=x"})
		return cmd
	})
	if err == nil {
		t.Fatal("expected error for missing job")
	}
	if err.Error() != "job not found" {
		t.Fatalf("unexpected error: %v", err)
	}
}
