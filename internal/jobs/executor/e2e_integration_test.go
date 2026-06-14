//go:build integration

package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

// E2E integration test that exercises the full job execution pipeline:
// 1. Enqueue a job via HTTP API (jobs-server)
// 2. Job is dispatched to Pulsar (jobs-dispatcher)
// 3. Job is consumed and executed (jobs-executor)
// 4. Poll until job completes
//
// Prerequisites:
// - Full stack running: docker compose up (mongodb, pulsar, jobs-server, jobs-dispatcher, jobs-executor)
// - Or: JOBS_API_BASE_URL env var pointing to jobs-server (default: http://localhost:3001)
//
// Run: task test-integration
// Or: go test -tags=integration -v ./internal/jobs/executor -run TestE2EIntegration

func TestE2EIntegration_EchoJobExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e integration test (-short)")
	}

	baseURL := os.Getenv("JOBS_API_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3001"
	}

	client := &http.Client{Timeout: 5 * time.Second}
	ctx := context.Background()

	t.Run("echo_job_completes_successfully", func(t *testing.T) {
		payload := map[string]any{
			"message": "Hello from E2E test",
		}
		body := map[string]any{
			"name":    "echo",
			"payload": payload,
			"tags":    []string{"e2e", "integration", "echo"},
		}

		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}

		resp, err := client.Post(baseURL+"/api/jobs", "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("enqueue job: %v (is the stack running? docker compose up)", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("enqueue failed: status=%d body=%s", resp.StatusCode, respBody)
		}

		var created metadata.JobMetadataModel
		if err := json.Unmarshal(respBody, &created); err != nil {
			t.Fatalf("decode created job: %v", err)
		}

		if created.JobID == "" {
			t.Fatalf("created job has empty ID: %+v", created)
		}
		if created.Name != "echo" {
			t.Fatalf("created job name=%s want echo", created.Name)
		}

		t.Logf("Enqueued job %s, polling for completion...", created.JobID)

		jobID := created.JobID
		completed := waitForJobCompletion(ctx, t, client, baseURL, jobID, 30*time.Second)
		if !completed {
			t.Fatalf("job %s did not complete within timeout", jobID)
		}

		finalJob := getJob(t, client, baseURL, jobID)
		if finalJob.Status != metadata.JobStatusCompleted {
			t.Fatalf("final status=%s want completed, job=%+v", finalJob.Status, finalJob)
		}

		t.Logf("Job %s completed successfully: %+v", jobID, finalJob)
	})

	t.Run("echo_job_with_empty_message_fails", func(t *testing.T) {
		payload := map[string]any{
			"message": "",
		}
		body := map[string]any{
			"name":    "echo",
			"payload": payload,
			"tags":    []string{"e2e", "integration", "echo-fail"},
		}

		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}

		resp, err := client.Post(baseURL+"/api/jobs", "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("enqueue job: %v", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("enqueue failed: status=%d body=%s", resp.StatusCode, respBody)
		}

		var created metadata.JobMetadataModel
		if err := json.Unmarshal(respBody, &created); err != nil {
			t.Fatalf("decode created job: %v", err)
		}

		t.Logf("Enqueued job %s with empty message, expecting failure...", created.JobID)

		jobID := created.JobID
		failed := waitForJobStatus(ctx, t, client, baseURL, jobID, metadata.JobStatusFailed, 30*time.Second)
		if !failed {
			t.Fatalf("job %s did not fail within timeout", jobID)
		}

		finalJob := getJob(t, client, baseURL, jobID)
		if finalJob.Status != metadata.JobStatusFailed {
			t.Fatalf("final status=%s want failed, job=%+v", finalJob.Status, finalJob)
		}
		if len(finalJob.Errors) == 0 {
			t.Fatalf("expected error message in failed job, got empty errors array")
		}

		t.Logf("Job %s failed as expected: error=%s", jobID, finalJob.GetLatestError())
	})

	t.Run("error_history_preserved_across_retry", func(t *testing.T) {
		payload := map[string]any{
			"message": "",
		}
		body := map[string]any{
			"name":    "echo",
			"payload": payload,
			"tags":    []string{"e2e", "integration", "error-history"},
		}

		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}

		resp, err := client.Post(baseURL+"/api/jobs", "application/json", bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("enqueue job: %v", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("enqueue failed: status=%d body=%s", resp.StatusCode, respBody)
		}

		var created metadata.JobMetadataModel
		if err := json.Unmarshal(respBody, &created); err != nil {
			t.Fatalf("decode created job: %v", err)
		}

		jobID := created.JobID
		if !waitForJobStatus(ctx, t, client, baseURL, jobID, metadata.JobStatusFailed, 30*time.Second) {
			t.Fatalf("job %s did not fail on first attempt", jobID)
		}

		firstFailed := getJob(t, client, baseURL, jobID)
		if len(firstFailed.Errors) != 1 {
			t.Fatalf("after first failure len(errors)=%d want 1", len(firstFailed.Errors))
		}
		if firstFailed.Errors[0].RetryAttempt != 0 {
			t.Fatalf("first error retryAttempt=%d want 0", firstFailed.Errors[0].RetryAttempt)
		}
		if firstFailed.Errors[0].Type != metadata.JobErrorTypeExecution {
			t.Fatalf("first error type=%q want execution", firstFailed.Errors[0].Type)
		}

		retryResp, err := client.Post(fmt.Sprintf("%s/api/jobs/%s/retry", baseURL, jobID), "application/json", nil)
		if err != nil {
			t.Fatalf("retry job: %v", err)
		}
		defer retryResp.Body.Close()
		if retryResp.StatusCode != http.StatusOK {
			retryBody, _ := io.ReadAll(retryResp.Body)
			t.Fatalf("retry failed: status=%d body=%s", retryResp.StatusCode, retryBody)
		}

		if !waitForJobStatus(ctx, t, client, baseURL, jobID, metadata.JobStatusFailed, 60*time.Second) {
			t.Fatalf("job %s did not fail on second attempt", jobID)
		}

		secondFailed := getJob(t, client, baseURL, jobID)
		if len(secondFailed.Errors) != 2 {
			t.Fatalf("after retry cycle len(errors)=%d want 2, errors=%+v", len(secondFailed.Errors), secondFailed.Errors)
		}
		if secondFailed.Errors[1].RetryAttempt != 1 {
			t.Fatalf("second error retryAttempt=%d want 1", secondFailed.Errors[1].RetryAttempt)
		}
		if secondFailed.RetryCount != 1 {
			t.Fatalf("retryCount=%d want 1", secondFailed.RetryCount)
		}

		t.Logf("Job %s error history: %+v", jobID, secondFailed.Errors)
	})
}

func waitForJobCompletion(ctx context.Context, t *testing.T, client *http.Client, baseURL, jobID string, timeout time.Duration) bool {
	return waitForJobStatus(ctx, t, client, baseURL, jobID, metadata.JobStatusCompleted, timeout)
}

func waitForJobStatus(ctx context.Context, t *testing.T, client *http.Client, baseURL, jobID string, targetStatus metadata.JobStatus, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			if time.Now().After(deadline) {
				t.Logf("Timeout waiting for job %s to reach status %s", jobID, targetStatus)
				return false
			}

			job := getJob(t, client, baseURL, jobID)
			t.Logf("Job %s status: %s", jobID, job.Status)

			if job.Status == targetStatus {
				return true
			}

			if job.Status == metadata.JobStatusFailed && targetStatus != metadata.JobStatusFailed {
				t.Fatalf("job %s failed unexpectedly: error=%s", jobID, job.GetLatestError())
			}

			if job.Status == metadata.JobStatusCancelled {
				t.Fatalf("job %s was cancelled", jobID)
			}
		}
	}
}

func getJob(t *testing.T, client *http.Client, baseURL, jobID string) metadata.JobMetadataModel {
	t.Helper()
	url := fmt.Sprintf("%s/api/jobs/%s", baseURL, jobID)
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get job failed: status=%d body=%s", resp.StatusCode, body)
	}

	var job metadata.JobMetadataModel
	if err := json.Unmarshal(body, &job); err != nil {
		t.Fatalf("decode job: %v", err)
	}

	return job
}
