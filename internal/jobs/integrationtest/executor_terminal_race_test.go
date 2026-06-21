//go:build integration

package integrationtest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/dispatchruntime"
	"github.com/daconjurer/jobby/internal/jobs/executor"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	jobpulsar "github.com/daconjurer/jobby/internal/jobs/pulsar"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"github.com/daconjurer/jobby/internal/testutil"
)

const integrationTestJobName = "account-lifecycle"

type blockingJobPayload struct {
	Message string `json:"message"`
}

// blockingJobHandler blocks until proceed is signaled so tests can race external updates.
type blockingJobHandler struct {
	started   chan struct{}
	proceed   chan struct{}
	execCount atomic.Int32
}

func newBlockingJobHandler() *blockingJobHandler {
	return &blockingJobHandler{
		started: make(chan struct{}, 1),
		proceed: make(chan struct{}),
	}
}

func (h *blockingJobHandler) Execute(ctx context.Context, jobID string, data blockingJobPayload) error {
	h.execCount.Add(1)
	select {
	case h.started <- struct{}{}:
	default:
	}
	select {
	case <-h.proceed:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type countingEchoJobHandler struct {
	echoHandler *EchoJobHandler
	execCount   atomic.Int32
}

func newCountingEchoJobHandler() *countingEchoJobHandler {
	return &countingEchoJobHandler{
		echoHandler: &EchoJobHandler{executed: make(chan string, 1)},
	}
}

func (h *countingEchoJobHandler) Execute(ctx context.Context, jobID string, data EchoJobPayload) error {
	h.execCount.Add(1)
	return h.echoHandler.Execute(ctx, jobID, data)
}

type executorIntegrationHarness struct {
	dispatch       *dispatchHarness
	registry       *executor.Registry
	executorSvc    *executor.ExecutorService
	pulsarClient   *jobpulsar.PulsarClient
	consumer       *jobpulsar.PulsarJobConsumer
	cancelConsumer context.CancelFunc
}

func newExecutorIntegrationHarness(tb testing.TB, register func(*executor.Registry)) *executorIntegrationHarness {
	tb.Helper()

	h := newDispatchHarness(tb, dispatchruntime.Options{})

	registry := executor.NewRegistry()
	if register != nil {
		register(registry)
	}
	executorSvc := executor.NewExecutorService(h.metadataSvc, registry)

	pulsarClient, err := jobpulsar.NewPulsarClient(h.pulsarCfg)
	if err != nil {
		tb.Fatalf("NewPulsarClient: %v", err)
	}

	resolver, err := jobpulsar.NewFileTopicResolver(testutil.JobTopicsConfigPath(tb))
	if err != nil {
		pulsarClient.Close()
		tb.Fatalf("NewFileTopicResolver: %v", err)
	}
	topics := resolver.UniqueTopics()

	subscription := fmt.Sprintf("integration-executor-%s", metadata.GenerateJobID())
	h.pulsarCfg.SubscriptionName = subscription
	consumer, err := jobpulsar.NewPulsarJobConsumer(pulsarClient, topics)
	if err != nil {
		pulsarClient.Close()
		tb.Fatalf("NewPulsarJobConsumer: %v", err)
	}

	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	go func() {
		err := consumer.Run(consumerCtx, func(ctx context.Context, jobID, name string, payload json.RawMessage) error {
			return executorSvc.ExecuteJob(ctx, jobID, name, payload)
		})
		if err != nil && err != context.Canceled {
			tb.Logf("consumer.Run error: %v", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)

	eh := &executorIntegrationHarness{
		dispatch:       h,
		registry:       registry,
		executorSvc:    executorSvc,
		pulsarClient:   pulsarClient,
		consumer:       consumer,
		cancelConsumer: cancelConsumer,
	}

	tb.Cleanup(func() {
		cancelConsumer()
		_ = consumer.Close()
		_ = pulsarClient.Close()
	})

	return eh
}

func enqueueIntegrationJob(tb testing.TB, h *executorIntegrationHarness, payload map[string]any) *metadata.JobMetadataModel {
	tb.Helper()
	ctx := context.Background()
	job, err := h.dispatch.enqueueSvc.Enqueue(ctx, integrationTestJobName, payload, service.CreateJobOptions{})
	if err != nil {
		tb.Fatalf("Enqueue: %v", err)
	}
	return job
}

func TestIntegration_Executor_ExternalFailDuringRun(t *testing.T) {
	blocking := newBlockingJobHandler()
	eh := newExecutorIntegrationHarness(t, func(r *executor.Registry) {
		executor.Register(r, integrationTestJobName, blocking)
	})

	job := enqueueIntegrationJob(t, eh, map[string]any{"message": "external-fail-during-run"})

	select {
	case <-blocking.started:
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for handler to start")
	}

	running := waitForJobStatus(t, eh.dispatch.metadataSvc, job.JobID, metadata.JobStatusRunning, 30*time.Second)
	if running.Status != metadata.JobStatusRunning {
		t.Fatalf("status=%s want running", running.Status)
	}

	const externalErr = "external fail during execution"
	if err := eh.dispatch.metadataSvc.FailJob(context.Background(), job.JobID, errors.New(externalErr)); err != nil {
		t.Fatalf("FailJob: %v", err)
	}

	close(blocking.proceed)

	failed := waitForJobStatus(t, eh.dispatch.metadataSvc, job.JobID, metadata.JobStatusFailed, 30*time.Second)
	if failed.Status != metadata.JobStatusFailed {
		t.Fatalf("status=%s want failed after external fail during run", failed.Status)
	}
	if len(failed.Errors) == 0 {
		t.Fatal("expected external error in errors[]")
	}
	if failed.GetLatestError() != externalErr {
		t.Fatalf("latest error=%q want %q", failed.GetLatestError(), externalErr)
	}

	time.Sleep(2 * time.Second)
	if got := blocking.execCount.Load(); got != 1 {
		t.Fatalf("handler execCount=%d want 1 (no redelivery loop)", got)
	}
}

func TestIntegration_Executor_CompleteBeforeExternalFail(t *testing.T) {
	echo := newCountingEchoJobHandler()
	eh := newExecutorIntegrationHarness(t, func(r *executor.Registry) {
		executor.Register(r, integrationTestJobName, echo)
	})

	job := enqueueIntegrationJob(t, eh, map[string]any{"message": "complete-before-fail"})

	select {
	case executedJobID := <-echo.echoHandler.executed:
		if executedJobID != job.JobID {
			t.Fatalf("handler executed for jobID=%s want %s", executedJobID, job.JobID)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for handler execution")
	}

	completed := waitForJobStatus(t, eh.dispatch.metadataSvc, job.JobID, metadata.JobStatusCompleted, 30*time.Second)
	if completed.Status != metadata.JobStatusCompleted {
		t.Fatalf("status=%s want completed", completed.Status)
	}

	err := eh.dispatch.metadataSvc.FailJob(context.Background(), job.JobID, errors.New("too late"))
	if !errors.Is(err, metadata.ErrJobAlreadyTerminal) {
		t.Fatalf("FailJob after complete: err=%v want ErrJobAlreadyTerminal", err)
	}

	final, err := eh.dispatch.metadataSvc.GetJob(context.Background(), job.JobID)
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	model := final.(*metadata.JobMetadataModel)
	if model.Status != metadata.JobStatusCompleted {
		t.Fatalf("status=%s want completed after rejected fail", model.Status)
	}
	if echo.execCount.Load() != 1 {
		t.Fatalf("handler execCount=%d want 1", echo.execCount.Load())
	}
}

func TestIntegration_Executor_TerminalRedeliverySkipsHandler(t *testing.T) {
	echo := newCountingEchoJobHandler()
	eh := newExecutorIntegrationHarness(t, func(r *executor.Registry) {
		executor.Register(r, integrationTestJobName, echo)
	})

	job := enqueueIntegrationJob(t, eh, map[string]any{"message": "terminal-redelivery"})

	select {
	case executedJobID := <-echo.echoHandler.executed:
		if executedJobID != job.JobID {
			t.Fatalf("handler executed for jobID=%s want %s", executedJobID, job.JobID)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for handler execution")
	}

	waitForJobStatus(t, eh.dispatch.metadataSvc, job.JobID, metadata.JobStatusCompleted, 30*time.Second)

	payload, err := json.Marshal(map[string]any{"message": "terminal-redelivery"})
	if err != nil {
		t.Fatalf("Marshal payload: %v", err)
	}
	if err := eh.executorSvc.ExecuteJob(context.Background(), job.JobID, integrationTestJobName, payload); err != nil {
		t.Fatalf("simulated redelivery ExecuteJob: %v", err)
	}

	if echo.execCount.Load() != 1 {
		t.Fatalf("handler execCount=%d want 1 after terminal redelivery", echo.execCount.Load())
	}

	final, err := eh.dispatch.metadataSvc.GetJob(context.Background(), job.JobID)
	if err != nil {
		t.Fatalf("GetJob: %v", err)
	}
	model := final.(*metadata.JobMetadataModel)
	if model.Status != metadata.JobStatusCompleted {
		t.Fatalf("status=%s want completed after redelivery skip", model.Status)
	}
}

func TestIntegration_Executor_CompleteVsFailRace(t *testing.T) {
	blocking := newBlockingJobHandler()
	eh := newExecutorIntegrationHarness(t, func(r *executor.Registry) {
		executor.Register(r, integrationTestJobName, blocking)
	})

	job := enqueueIntegrationJob(t, eh, map[string]any{"message": "complete-vs-fail-race"})

	select {
	case <-blocking.started:
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for handler to start")
	}
	waitForJobStatus(t, eh.dispatch.metadataSvc, job.JobID, metadata.JobStatusRunning, 30*time.Second)

	release := make(chan struct{})
	failErr := make(chan error, 1)
	go func() {
		<-release
		failErr <- eh.dispatch.metadataSvc.FailJob(context.Background(), job.JobID, errors.New("race fail"))
	}()
	go func() {
		<-release
		close(blocking.proceed)
	}()
	close(release)

	select {
	case err := <-failErr:
		if err != nil && !errors.Is(err, metadata.ErrJobAlreadyTerminal) {
			t.Fatalf("FailJob during race: %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for FailJob during race")
	}

	final := waitForJobStatusOrBeyond(t, eh.dispatch.metadataSvc, job.JobID, metadata.JobStatusCompleted, 30*time.Second)
	if !final.Status.IsTerminal() {
		t.Fatalf("status=%s want terminal after race", final.Status)
	}
	if final.Status != metadata.JobStatusCompleted && final.Status != metadata.JobStatusFailed {
		t.Fatalf("unexpected terminal status %s", final.Status)
	}
	if blocking.execCount.Load() != 1 {
		t.Fatalf("handler execCount=%d want 1 after race", blocking.execCount.Load())
	}
}
