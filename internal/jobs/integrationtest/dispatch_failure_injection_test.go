package integrationtest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/dispatch"
	"github.com/daconjurer/jobby/internal/jobs/dispatchruntime"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/mongodb"
	jobpulsar "github.com/daconjurer/jobby/internal/jobs/pulsar"
	"github.com/daconjurer/jobby/internal/jobs/service"
)

const brokerDownErrMsg = "broker down"

type failingPublisher struct {
	err error
}

func (p failingPublisher) Publish(context.Context, dispatch.JobDispatchProjection) error {
	return p.err
}

// recoverablePublisher fails until recover() is called, then delegates to a real publisher.
type recoverablePublisher struct {
	mu        sync.Mutex
	recovered bool
	failErr   error
	delegate  dispatch.JobDispatchPublisher
}

func (p *recoverablePublisher) recover() {
	p.mu.Lock()
	p.recovered = true
	p.mu.Unlock()
}

func (p *recoverablePublisher) Publish(ctx context.Context, job dispatch.JobDispatchProjection) error {
	p.mu.Lock()
	recovered := p.recovered
	p.mu.Unlock()
	if !recovered {
		return p.failErr
	}
	return p.delegate.Publish(ctx, job)
}

func preparePollOnlyFailureRuntime(
	tb testing.TB,
	cfg dispatchruntime.Config,
	publisher dispatch.JobDispatchPublisher,
) *service.MetadataService {
	tb.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	tb.Cleanup(cancel)

	reader, writer, mongoClient, err := mongodb.OpenMongoJobs(ctx, cfg.Mongo)
	if err != nil {
		cancel()
		tb.Fatalf("OpenMongoJobs: %v", err)
	}
	db := mongoClient.Database(cfg.Mongo.Database)
	if err := clearJobCollections(ctx, db, cfg.Mongo); err != nil {
		cancel()
		tb.Fatalf("clearJobCollections: %v", err)
	}
	tb.Cleanup(func() {
		_ = mongoClient.Disconnect(context.Background())
		_ = clearJobCollections(context.Background(), db, cfg.Mongo)
	})

	metadataSvc := service.NewMetadataService(reader, writer)
	metadataColl := db.Collection(cfg.Mongo.CollectionMetadata)

	_, runCancel, runtime := startDispatchRuntime(tb, cfg, db, metadataSvc, dispatchruntime.Options{
		StreamRunner:   noopStreamRunner{},
		PendingFetcher: mongodb.NewMongoPendingJobFetcher(metadataColl),
		Publisher:      publisher,
	})
	tb.Cleanup(func() {
		runCancel()
		_ = runtime.Close()
	})

	return metadataSvc
}

func TestIntegration_DispatchFailure_BrokerDownKeepsPendingAndRecordsAttempt(t *testing.T) {
	cfg := integrationDispatchConfigWithPublisher(t)
	failPub := failingPublisher{err: errors.New(brokerDownErrMsg)}

	metadataSvc := preparePollOnlyFailureRuntime(t, cfg, failPub)

	const jobName = "account-lifecycle"
	const wantTopic = "persistent://public/default/accounts/jobs"

	model, err := metadataSvc.CreateJob(context.Background(), jobName, map[string]any{"phase": "broker-down"}, service.CreateJobOptions{
		Topic: wantTopic,
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	got := waitForMinDispatchAttempts(t, metadataSvc, model.JobID, 1, 15*time.Second)
	if got.Status != metadata.JobStatusPendingDispatch {
		t.Fatalf("status=%s want pending_dispatch", got.Status)
	}
	if got.DispatchLastError == "" {
		t.Fatal("expected dispatchLastError to be set")
	}
	if !strings.Contains(got.DispatchLastError, brokerDownErrMsg) {
		t.Fatalf("dispatchLastError=%q want substring %q", got.DispatchLastError, brokerDownErrMsg)
	}
}

func TestIntegration_DispatchFailure_RecoveryDispatchesAfterBrokerUp(t *testing.T) {
	cfg := integrationDispatchConfig(t)

	pulsarClient, err := jobpulsar.NewPulsarClient(cfg.Pulsar)
	if err != nil {
		t.Fatalf("NewPulsarClient: %v", err)
	}
	t.Cleanup(func() { _ = pulsarClient.Close() })
	producer := jobpulsar.NewPulsarJobProducer(pulsarClient)
	t.Cleanup(func() { _ = producer.Close() })
	realPub := jobpulsar.NewDispatchPublisher(producer)

	recoverable := &recoverablePublisher{
		failErr:  errors.New(brokerDownErrMsg),
		delegate: realPub,
	}

	metadataSvc := preparePollOnlyFailureRuntime(t, cfg, recoverable)

	const jobName = "account-lifecycle"
	const wantTopic = "persistent://public/default/accounts/jobs"

	subscription := fmt.Sprintf("integration-recovery-%s", metadata.GenerateJobID())
	msgCh := make(chan jobpulsar.JobMessage, 1)
	errCh := make(chan error, 1)
	go func() {
		msg, err := consumeJobMessage(cfg.Pulsar.ServiceURL, wantTopic, subscription, 30*time.Second)
		if err != nil {
			errCh <- err
			return
		}
		msgCh <- msg
	}()
	time.Sleep(200 * time.Millisecond)

	model, err := metadataSvc.CreateJob(context.Background(), jobName, map[string]any{"phase": "recovery"}, service.CreateJobOptions{
		Topic: wantTopic,
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	waitForMinDispatchAttempts(t, metadataSvc, model.JobID, 1, 15*time.Second)
	recoverable.recover()

	dispatched := waitForJobStatus(t, metadataSvc, model.JobID, metadata.JobStatusDispatched, 30*time.Second)
	if dispatched.DispatchedAt == nil {
		t.Fatal("expected dispatchedAt to be set after recovery")
	}

	select {
	case err := <-errCh:
		t.Fatalf("consume: %v", err)
	case msg := <-msgCh:
		if msg.JobID != model.JobID {
			t.Fatalf("message jobId=%q want %q", msg.JobID, model.JobID)
		}
	case <-time.After(35 * time.Second):
		t.Fatal("timeout waiting for Pulsar message after recovery")
	}
}

func TestIntegration_DispatchFailure_ExhaustionMarksDispatchFailed(t *testing.T) {
	cfg := integrationDispatchConfigWithPublisher(t)
	cfg.Worker.MaxAttempts = 3
	failPub := failingPublisher{err: errors.New(brokerDownErrMsg)}

	metadataSvc := preparePollOnlyFailureRuntime(t, cfg, failPub)

	const jobName = "account-lifecycle"
	const wantTopic = "persistent://public/default/accounts/jobs"

	model, err := metadataSvc.CreateJob(context.Background(), jobName, map[string]any{"phase": "exhaustion"}, service.CreateJobOptions{
		Topic: wantTopic,
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}

	failed := waitForJobStatus(t, metadataSvc, model.JobID, metadata.JobStatusDispatchFailed, 30*time.Second)
	if failed.DispatchAttempts != 3 {
		t.Fatalf("dispatchAttempts=%d want 3", failed.DispatchAttempts)
	}
	if failed.DispatchLastError == "" {
		t.Fatal("expected dispatchLastError to be set")
	}
	if !strings.Contains(failed.DispatchLastError, brokerDownErrMsg) {
		t.Fatalf("dispatchLastError=%q want substring %q", failed.DispatchLastError, brokerDownErrMsg)
	}
	if len(failed.Errors) == 0 {
		t.Fatal("expected errors array to be set on dispatch_failed")
	}
}
