package dispatchruntime

import (
	"context"
	"errors"
	"fmt"

	"github.com/daconjurer/jobby/internal/jobs/dispatch"
	"github.com/daconjurer/jobby/internal/jobs/mongodb"
	jobpulsar "github.com/daconjurer/jobby/internal/jobs/pulsar"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Options override default adapter construction (for tests or experiments).
type Options struct {
	Publisher      dispatch.JobDispatchPublisher
	ResumeTokens   mongodb.ResumeTokenStore
	StreamRunner   dispatch.StreamRunner
	PendingFetcher dispatch.PendingJobFetcher
}

// Runtime owns the assembled dispatch worker and resources that require explicit shutdown.
type Runtime struct {
	Worker       *dispatch.DispatchWorker
	pulsarClient *jobpulsar.PulsarClient
	producer     *jobpulsar.PulsarJobProducer
	watchClient  *mongo.Client
}

// New wires MongoDB change stream + poll fallback, Pulsar publish, and the dispatch saga handler.
func New(
	ctx context.Context,
	cfg Config,
	metadataColl *mongo.Collection,
	metadataSvc *service.MetadataService,
	opts Options,
) (*Runtime, error) {
	if metadataColl == nil {
		return nil, fmt.Errorf("metadata collection is required")
	}
	if metadataSvc == nil {
		return nil, fmt.Errorf("metadata service is required")
	}

	runtime := &Runtime{}

	publisher := opts.Publisher
	if publisher == nil {
		pulsarClient, err := jobpulsar.NewPulsarClient(cfg.Pulsar)
		if err != nil {
			return nil, fmt.Errorf("connect pulsar: %w", err)
		}
		runtime.pulsarClient = pulsarClient
		runtime.producer = jobpulsar.NewPulsarJobProducer(pulsarClient)
		publisher = jobpulsar.NewDispatchPublisher(runtime.producer)
	}

	handler := dispatch.NewDispatchHandler(
		dispatch.DispatchHandlerConfig{MaxAttempts: cfg.Worker.MaxAttempts},
		publisher,
		metadataSvc,
	)

	streamRunner := opts.StreamRunner
	pendingFetcher := opts.PendingFetcher
	if streamRunner != nil || pendingFetcher != nil {
		if streamRunner == nil || pendingFetcher == nil {
			return nil, errors.Join(
				fmt.Errorf("StreamRunner and PendingFetcher overrides must both be set"),
				runtime.closePartial(),
			)
		}
	} else {
		watchClient, watchColl, err := mongodb.OpenMongoWatchClient(ctx, cfg.Mongo, cfg.Stream.MaxPoolSize)
		if err != nil {
			return nil, errors.Join(
				fmt.Errorf("open change stream watch client: %w", err),
				runtime.closePartial(),
			)
		}
		runtime.watchClient = watchClient

		tokens := opts.ResumeTokens
		if tokens == nil {
			if cfg.Stream.ResumeTokenPath != "" {
				tokens = mongodb.NewFileResumeTokenStore(cfg.Stream.ResumeTokenPath)
			} else {
				tokens = mongodb.NopResumeTokenStore{}
			}
		}

		streamRunner = mongodb.NewStreamWatcher(watchColl, handler, tokens)
		pendingFetcher = mongodb.NewMongoPendingJobFetcher(metadataColl)
	}

	runtime.Worker = dispatch.NewDispatchWorker(
		cfg.Worker,
		handler,
		pendingFetcher,
		streamRunner,
	)

	return runtime, nil
}

// Run blocks until ctx is cancelled (change stream + poll fallback).
func (r *Runtime) Run(ctx context.Context) {
	if r.Worker != nil {
		r.Worker.Run(ctx)
	}
}

// Close releases Pulsar producers/client and disconnects the watch MongoDB client.
func (r *Runtime) Close() error {
	return r.closePartial()
}

func (r *Runtime) closePartial() error {
	if r.producer != nil {
		_ = r.producer.Close()
		r.producer = nil
	}
	if r.pulsarClient != nil {
		_ = r.pulsarClient.Close()
		r.pulsarClient = nil
	}
	if r.watchClient != nil {
		err := r.watchClient.Disconnect(context.Background())
		r.watchClient = nil
		return err
	}
	return nil
}
