//go:build integration

package integrationtest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
	jobpulsar "github.com/daconjurer/jobby/internal/jobs/pulsar"
	"github.com/daconjurer/jobby/internal/jobs/service"
)

// Integration tests require MongoDB replica set and live Pulsar.
// Run: task mongo-up && docker compose up -d pulsar && task test-integration
//
// Required: MONGODB_URI, PULSAR_SERVICE_URL.
// Optional: JOB_TOPICS_CONFIG_PATH (relative to module root; default config/job-topics.yaml).

func TestIntegration_DispatchSaga_EnqueueToDispatched(t *testing.T) {
	h := newDispatchSagaHarness(t)

	const jobName = "account-lifecycle"
	const wantTopic = "persistent://public/default/accounts/jobs"

	subscription := fmt.Sprintf("integration-dispatch-%s", metadata.GenerateJobID())
	msgCh := make(chan jobpulsar.JobMessage, 1)
	errCh := make(chan error, 1)
	go func() {
		msg, err := consumeJobMessage(h.pulsarCfg.ServiceURL, wantTopic, subscription, 30*time.Second)
		if err != nil {
			errCh <- err
			return
		}
		msgCh <- msg
	}()

	time.Sleep(200 * time.Millisecond)

	ctx := context.Background()
	payload := map[string]any{"source": "dispatch-integration", "n": 1}
	job, err := h.enqueueSvc.Enqueue(ctx, jobName, payload, service.CreateJobOptions{})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if job.Status != metadata.JobStatusPendingDispatch {
		t.Fatalf("status=%s want pending_dispatch", job.Status)
	}
	if job.Topic != wantTopic {
		t.Fatalf("topic=%q want %q", job.Topic, wantTopic)
	}

	dispatched := waitForJobStatus(t, h.metadataSvc, job.JobID, metadata.JobStatusDispatched, 30*time.Second)
	if dispatched.DispatchedAt == nil {
		t.Fatal("expected dispatchedAt to be set")
	}

	select {
	case err := <-errCh:
		t.Fatalf("consume: %v", err)
	case msg := <-msgCh:
		if msg.JobID != job.JobID {
			t.Fatalf("message jobId=%q want %q", msg.JobID, job.JobID)
		}
		if msg.Name != jobName {
			t.Fatalf("message name=%q want %q", msg.Name, jobName)
		}
		decoded, err := jobpulsar.DecodePayload[map[string]any](msg)
		if err != nil {
			t.Fatal(err)
		}
		if decoded["source"] != "dispatch-integration" {
			t.Fatalf("payload=%v", decoded)
		}
	case <-time.After(35 * time.Second):
		t.Fatal("timeout waiting for Pulsar message")
	}
}
