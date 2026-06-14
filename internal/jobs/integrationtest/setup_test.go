//go:build integration

package integrationtest

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/daconjurer/jobby/internal/config"
	"github.com/daconjurer/jobby/internal/jobs/dispatch"
	"github.com/daconjurer/jobby/internal/jobs/dispatchruntime"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/mongodb"
	jobpulsar "github.com/daconjurer/jobby/internal/jobs/pulsar"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"github.com/daconjurer/jobby/internal/testutil"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

const integrationPollInterval = 200 * time.Millisecond

func integrationMongoEnv(tb testing.TB) mongodb.MongoConfig {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test (-short)")
	}
	db := os.Getenv("MONGODB_DATABASE")
	if db == "" {
		db = "jobby"
	}
	metaColl := os.Getenv("MONGODB_COLLECTION_METADATA")
	if metaColl == "" {
		metaColl = "job_metadata"
	}
	logsColl := os.Getenv("MONGODB_COLLECTION_LOGS")
	if logsColl == "" {
		logsColl = "job_logs"
	}
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		tb.Fatalf("MONGODB_URI is not set (required for integration tests; see .env.example)")
	}
	return mongodb.MongoConfig{
		URI:                uri,
		Database:           db,
		CollectionMetadata: metaColl,
		CollectionLogs:     logsColl,
		Timeout:            30 * time.Second,
		MaxPoolSize:        50,
		MinPoolSize:        0,
	}
}

func integrationPulsarEnv(tb testing.TB) config.PulsarConfig {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test (-short)")
	}
	url := os.Getenv("PULSAR_SERVICE_URL")
	if url == "" {
		tb.Skip("PULSAR_SERVICE_URL is not set (required for dispatch integration tests)")
	}
	// Use unique subscription per test to avoid consuming old messages from previous runs
	uniqueSubscription := fmt.Sprintf("integration-dispatch-%s", metadata.GenerateJobID())
	return config.PulsarConfig{
		ServiceURL:       url,
		SubscriptionName: uniqueSubscription,
	}
}

func integrationDispatchWorkerConfig(tb testing.TB) dispatch.WorkerConfig {
	tb.Helper()
	return dispatch.WorkerConfig{
		PollInterval: integrationPollInterval,
		BatchSize:    50,
		MaxAttempts:  5,
	}
}

func integrationDispatchConfig(tb testing.TB) dispatchruntime.Config {
	tb.Helper()
	pulsarCfg := integrationPulsarEnv(tb)
	return dispatchruntime.Config{
		Mongo:  integrationMongoEnv(tb),
		Worker: integrationDispatchWorkerConfig(tb),
		Pulsar: pulsarCfg,
		Stream: dispatchruntime.StreamConfig{MaxPoolSize: 2},
	}
}

// integrationDispatchConfigWithPublisher uses Mongo only; caller supplies Publisher (no live Pulsar).
func integrationDispatchConfigWithPublisher(tb testing.TB) dispatchruntime.Config {
	tb.Helper()
	return dispatchruntime.Config{
		Mongo:  integrationMongoEnv(tb),
		Worker: integrationDispatchWorkerConfig(tb),
		Stream: dispatchruntime.StreamConfig{MaxPoolSize: 2},
	}
}

func clearJobCollections(ctx context.Context, db *mongo.Database, cfg mongodb.MongoConfig) error {
	meta := db.Collection(cfg.CollectionMetadata)
	logs := db.Collection(cfg.CollectionLogs)
	if _, err := meta.DeleteMany(ctx, bson.M{}); err != nil {
		return fmt.Errorf("clear %q: %w", cfg.CollectionMetadata, err)
	}
	if _, err := logs.DeleteMany(ctx, bson.M{}); err != nil {
		return fmt.Errorf("clear %q: %w", cfg.CollectionLogs, err)
	}
	return nil
}

// noopStreamRunner satisfies dispatch.StreamRunner without watching MongoDB.
type noopStreamRunner struct{}

func (noopStreamRunner) Run(ctx context.Context) { <-ctx.Done() }

// emptyPendingFetcher satisfies dispatch.PendingJobFetcher without querying MongoDB.
type emptyPendingFetcher struct{}

func (emptyPendingFetcher) FetchPending(context.Context, int, int) ([]dispatch.JobDispatchProjection, error) {
	return nil, nil
}

type dispatchHarness struct {
	cfg         dispatchruntime.Config
	mongoClient *mongo.Client
	db          *mongo.Database
	cancel      context.CancelFunc
	runtime     *dispatchruntime.Runtime
	metadataSvc *service.MetadataService
	enqueueSvc  *service.EnqueueService
	pulsarCfg   config.PulsarConfig
}

func newDispatchHarness(tb testing.TB, opts dispatchruntime.Options) *dispatchHarness {
	tb.Helper()
	cfg := integrationDispatchConfig(tb)

	ctx, cancel := context.WithCancel(context.Background())

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

	metadataSvc := service.NewMetadataService(reader, writer)
	metadataColl := db.Collection(cfg.Mongo.CollectionMetadata)

	runtime, err := dispatchruntime.New(ctx, cfg, metadataColl, metadataSvc, opts)
	if err != nil {
		cancel()
		tb.Fatalf("dispatchruntime.New: %v", err)
	}

	go runtime.Run(ctx)

	resolver, err := jobpulsar.NewFileTopicResolver(testutil.JobTopicsConfigPath(tb))
	if err != nil {
		cancel()
		_ = runtime.Close()
		tb.Fatalf("NewFileTopicResolver: %v", err)
	}
	enqueueSvc := service.NewEnqueueService(metadataSvc, resolver)

	h := &dispatchHarness{
		cfg:         cfg,
		mongoClient: mongoClient,
		db:          db,
		cancel:      cancel,
		runtime:     runtime,
		metadataSvc: metadataSvc,
		enqueueSvc:  enqueueSvc,
		pulsarCfg:   cfg.Pulsar,
	}

	tb.Cleanup(func() {
		h.cancel()
		_ = h.runtime.Close()
		_ = h.mongoClient.Disconnect(context.Background())
		_ = clearJobCollections(context.Background(), db, cfg.Mongo)
	})

	return h
}

func newDispatchSagaHarness(tb testing.TB) *dispatchHarness {
	tb.Helper()
	return newDispatchHarness(tb, dispatchruntime.Options{})
}

func (h *dispatchHarness) stopRuntime(tb testing.TB) {
	tb.Helper()
	h.cancel()
	if err := h.runtime.Close(); err != nil {
		tb.Fatalf("runtime.Close: %v", err)
	}
}

func startDispatchRuntime(
	tb testing.TB,
	cfg dispatchruntime.Config,
	db *mongo.Database,
	metadataSvc *service.MetadataService,
	opts dispatchruntime.Options,
) (context.Context, context.CancelFunc, *dispatchruntime.Runtime) {
	tb.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	metadataColl := db.Collection(cfg.Mongo.CollectionMetadata)
	runtime, err := dispatchruntime.New(ctx, cfg, metadataColl, metadataSvc, opts)
	if err != nil {
		cancel()
		tb.Fatalf("dispatchruntime.New: %v", err)
	}
	go runtime.Run(ctx)
	return ctx, cancel, runtime
}

func waitForMinDispatchAttempts(tb testing.TB, svc *service.MetadataService, jobID string, minAttempts int, timeout time.Duration) *metadata.JobMetadataModel {
	tb.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		job, err := svc.GetJob(context.Background(), jobID)
		if err == nil {
			model := job.(*metadata.JobMetadataModel)
			if model.DispatchAttempts >= minAttempts {
				return model
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	job, err := svc.GetJob(context.Background(), jobID)
	if err != nil {
		tb.Fatalf("GetJob(%s): %v", jobID, err)
	}
	model := job.(*metadata.JobMetadataModel)
	tb.Fatalf("job %s dispatchAttempts=%d want >= %d after %s", jobID, model.DispatchAttempts, minAttempts, timeout)
	return nil
}

func waitForJobStatus(tb testing.TB, svc *service.MetadataService, jobID string, want metadata.JobStatus, timeout time.Duration) *metadata.JobMetadataModel {
	tb.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		job, err := svc.GetJob(context.Background(), jobID)
		if err == nil {
			model := job.(*metadata.JobMetadataModel)
			if model.Status == want {
				return model
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	job, err := svc.GetJob(context.Background(), jobID)
	if err != nil {
		tb.Fatalf("GetJob(%s): %v", jobID, err)
	}
	model := job.(*metadata.JobMetadataModel)
	tb.Fatalf("job %s status=%s want %s after %s", jobID, model.Status, want, timeout)
	return nil
}

// waitForJobStatusOrBeyond waits for a job to reach at least the target status.
// For terminal states like "completed", this accepts if the job has already passed through earlier states.
func waitForJobStatusOrBeyond(tb testing.TB, svc *service.MetadataService, jobID string, minStatus metadata.JobStatus, timeout time.Duration) *metadata.JobMetadataModel {
	tb.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		job, err := svc.GetJob(context.Background(), jobID)
		if err == nil {
			model := job.(*metadata.JobMetadataModel)
			// Accept if at target status or any terminal status
			if model.Status == minStatus || model.Status.IsTerminal() {
				return model
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	job, err := svc.GetJob(context.Background(), jobID)
	if err != nil {
		tb.Fatalf("GetJob(%s): %v", jobID, err)
	}
	model := job.(*metadata.JobMetadataModel)
	tb.Fatalf("job %s status=%s want at least %s after %s", jobID, model.Status, minStatus, timeout)
	return nil
}

func consumeJobMessage(serviceURL, topic, subscription string, timeout time.Duration) (jobpulsar.JobMessage, error) {
	client, err := pulsar.NewClient(pulsar.ClientOptions{URL: serviceURL})
	if err != nil {
		return jobpulsar.JobMessage{}, fmt.Errorf("pulsar.NewClient: %w", err)
	}
	defer client.Close()

	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:                       topic,
		SubscriptionName:            subscription,
		Type:                        pulsar.Shared,
		SubscriptionInitialPosition: pulsar.SubscriptionPositionLatest,
	})
	if err != nil {
		return jobpulsar.JobMessage{}, fmt.Errorf("Subscribe: %w", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	raw, err := consumer.Receive(ctx)
	if err != nil {
		return jobpulsar.JobMessage{}, fmt.Errorf("Receive: %w", err)
	}
	if err := consumer.Ack(raw); err != nil {
		return jobpulsar.JobMessage{}, fmt.Errorf("Ack: %w", err)
	}
	if raw.Key() == "" {
		return jobpulsar.JobMessage{}, fmt.Errorf("expected non-empty Pulsar message key")
	}

	msg, err := jobpulsar.Unmarshal(raw.Payload())
	if err != nil {
		return jobpulsar.JobMessage{}, fmt.Errorf("Unmarshal: %w", err)
	}
	if raw.Key() != msg.JobID {
		return jobpulsar.JobMessage{}, fmt.Errorf("message key=%q want jobId=%q", raw.Key(), msg.JobID)
	}
	return msg, nil
}
