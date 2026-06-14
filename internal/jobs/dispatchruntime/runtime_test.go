package dispatchruntime

import (
	"context"
	"testing"
	"time"

	"github.com/daconjurer/jobby/internal/config"
	"github.com/daconjurer/jobby/internal/jobs/dispatch"
	"github.com/daconjurer/jobby/internal/jobs/mongodb"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type noopPublisher struct{}

func (noopPublisher) Publish(context.Context, dispatch.JobDispatchProjection) error { return nil }

type recordingStream struct {
	runCh chan struct{}
}

func (s *recordingStream) Run(ctx context.Context) {
	if s.runCh != nil {
		close(s.runCh)
	}
	<-ctx.Done()
}

type recordingPending struct{}

func (recordingPending) FetchPending(context.Context, int, int) ([]dispatch.JobDispatchProjection, error) {
	return nil, nil
}

func TestConfigFromEnv_MapsDispatchSettings(t *testing.T) {
	got := ConfigFromEnv(
		mongodb.MongoConfig{URI: "mongodb://x", Database: "jobby"},
		config.PulsarConfig{ServiceURL: "pulsar://localhost:6650"},
		config.MongoDispatchWorkerConfig{
			PollInterval:                 2 * time.Second,
			BatchSize:                    25,
			MaxAttempts:                  7,
			StreamMaxPoolSize:            3,
			StreamMongoDBResumeTokenPath: "/tmp/token.json",
		},
	)
	if got.Worker.PollInterval != 2*time.Second || got.Worker.BatchSize != 25 || got.Worker.MaxAttempts != 7 {
		t.Fatalf("worker config=%+v", got.Worker)
	}
	if got.Stream.MaxPoolSize != 3 || got.Stream.ResumeTokenPath != "/tmp/token.json" {
		t.Fatalf("stream config=%+v", got.Stream)
	}
	if got.Pulsar.ServiceURL != "pulsar://localhost:6650" {
		t.Fatalf("pulsar=%+v", got.Pulsar)
	}
}

func TestNew_WithInjectedAdaptersSkipsMongoAndPulsar(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := &recordingStream{runCh: make(chan struct{})}
	coll := &mongo.Collection{}
	svc := service.NewMetadataService(nil, nil)

	runtime, err := New(
		ctx,
		Config{Worker: dispatch.WorkerConfig{PollInterval: time.Hour, MaxAttempts: 3}},
		coll,
		svc,
		Options{
			Publisher:      noopPublisher{},
			StreamRunner:   stream,
			PendingFetcher: recordingPending{},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	go runtime.Run(ctx)

	select {
	case <-stream.runCh:
	case <-time.After(time.Second):
		t.Fatal("stream runner was not started")
	}
}

func TestNew_RequiresMetadataColl(t *testing.T) {
	_, err := New(context.Background(), Config{}, nil, service.NewMetadataService(nil, nil), Options{
		Publisher:      noopPublisher{},
		StreamRunner:   &recordingStream{},
		PendingFetcher: recordingPending{},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNew_RequiresMetadataSvc(t *testing.T) {
	_, err := New(context.Background(), Config{}, &mongo.Collection{}, nil, Options{
		Publisher:      noopPublisher{},
		StreamRunner:   &recordingStream{},
		PendingFetcher: recordingPending{},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNew_PendingFetcherOverrideKeepsDefaultStream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := &recordingStream{runCh: make(chan struct{})}
	coll := &mongo.Collection{}
	svc := service.NewMetadataService(nil, nil)

	runtime, err := New(
		ctx,
		Config{Worker: dispatch.WorkerConfig{PollInterval: time.Hour, MaxAttempts: 3}},
		coll,
		svc,
		Options{
			Publisher:      noopPublisher{},
			PendingFetcher: recordingPending{},
			StreamRunner:   stream,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	go runtime.Run(ctx)

	select {
	case <-stream.runCh:
	case <-time.After(time.Second):
		t.Fatal("stream runner was not started")
	}
}
