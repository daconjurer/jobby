package integrationtest

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/dispatchruntime"
	"github.com/daconjurer/jobby/internal/jobs/executor"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	jobpulsar "github.com/daconjurer/jobby/internal/jobs/pulsar"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"github.com/daconjurer/jobby/internal/testutil"
)

// EchoJobHandler is a simple test handler that just completes the job.
type EchoJobHandler struct {
	executed chan string
}

type EchoJobPayload struct {
	Message string `json:"message"`
}

func (h *EchoJobHandler) Execute(ctx context.Context, jobID string, data EchoJobPayload) error {
	if h.executed != nil {
		select {
		case h.executed <- jobID:
		default:
		}
	}
	return nil
}

func TestIntegration_ExecutorSaga_DispatchedToCompleted(t *testing.T) {
	h := newDispatchHarness(t, dispatchruntime.Options{})

	const jobName = "account-lifecycle"

	// Create registry and register echo handler
	registry := executor.NewRegistry()
	echoHandler := &EchoJobHandler{
		executed: make(chan string, 1),
	}
	executor.Register(registry, jobName, echoHandler)

	// Create executor service
	executorSvc := executor.NewExecutorService(h.metadataSvc, registry)

	// Create Pulsar client and consumer
	pulsarClient, err := jobpulsar.NewPulsarClient(h.pulsarCfg)
	if err != nil {
		t.Fatalf("NewPulsarClient: %v", err)
	}
	defer func() { _ = pulsarClient.Close() }()

	// Get topics from resolver
	resolver, err := jobpulsar.NewFileTopicResolver(testutil.JobTopicsConfigPath(t))
	if err != nil {
		t.Fatalf("NewFileTopicResolver: %v", err)
	}
	topics := resolver.UniqueTopics()

	// Create job consumer with unique subscription
	subscription := fmt.Sprintf("integration-executor-%s", metadata.GenerateJobID())
	h.pulsarCfg.SubscriptionName = subscription
	consumer, err := jobpulsar.NewPulsarJobConsumer(pulsarClient, topics)
	if err != nil {
		t.Fatalf("NewPulsarJobConsumer: %v", err)
	}
	defer func() { _ = consumer.Close() }()

	// Start consumer in background
	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	defer cancelConsumer()

	go func() {
		err := consumer.Run(consumerCtx, func(ctx context.Context, jobID, name string, payload json.RawMessage) error {
			return executorSvc.ExecuteJob(ctx, jobID, name, payload)
		})
		if err != nil && err != context.Canceled {
			t.Logf("consumer.Run error: %v", err)
		}
	}()

	// Give consumer time to start
	time.Sleep(500 * time.Millisecond)

	// Enqueue a job
	ctx := context.Background()
	payload := map[string]any{"message": "test-executor-saga"}
	job, err := h.enqueueSvc.Enqueue(ctx, jobName, payload, service.CreateJobOptions{})
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if job.Status != metadata.JobStatusPendingDispatch {
		t.Fatalf("status=%s want pending_dispatch", job.Status)
	}

	// Wait for handler to be executed (job may complete very quickly)
	select {
	case executedJobID := <-echoHandler.executed:
		if executedJobID != job.JobID {
			t.Fatalf("handler executed for jobID=%s want %s", executedJobID, job.JobID)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for handler execution")
	}

	// Wait for job to complete (may already be completed)
	completed := waitForJobStatusOrBeyond(t, h.metadataSvc, job.JobID, metadata.JobStatusCompleted, 30*time.Second)

	// Verify the job went through the full lifecycle by checking timestamps
	if completed.DispatchedAt == nil {
		t.Fatal("expected dispatchedAt to be set (job was never dispatched)")
	}
	if completed.StartedAt == nil {
		t.Fatal("expected startedAt to be set (job never started execution)")
	}
	if completed.CompletedAt == nil {
		t.Fatal("expected completedAt to be set")
	}

	// Verify timestamps are in correct order
	if !completed.CreatedAt.Before(*completed.StartedAt) {
		t.Fatalf("createdAt=%s should be before startedAt=%s", completed.CreatedAt, *completed.StartedAt)
	}
	if !completed.StartedAt.Before(*completed.CompletedAt) {
		t.Fatalf("startedAt=%s should be before completedAt=%s", *completed.StartedAt, *completed.CompletedAt)
	}

	// Verify no error
	if len(completed.Errors) != 0 {
		t.Fatalf("expected no errors, got: %v", completed.Errors)
	}
}
