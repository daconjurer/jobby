package main

import (
	"context"
	"fmt"

	"github.com/daconjurer/jobby/internal/config"
	"github.com/daconjurer/jobby/internal/jobs/dispatch"
	dispatchmongodb "github.com/daconjurer/jobby/internal/jobs/mongodb"
	jobpulsar "github.com/daconjurer/jobby/internal/jobs/pulsar"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// dispatchWorkerRuntime owns Pulsar, Mongo watch, and the running dispatch worker.
type dispatchWorkerRuntime struct {
	pulsarClient *jobpulsar.PulsarClient
	producer     *jobpulsar.PulsarJobProducer
	watchClient  *mongo.Client
	worker       *dispatch.Worker
}

func newPulsarDispatchWorker(
	ctx context.Context,
	pulsarCfg config.PulsarConfig,
	mongoCfg dispatchmongodb.MongoConfig,
	dispatchCfg config.MongoDispatchWorkerConfig,
	metadataSvc *service.MetadataService,
	metadataColl *mongo.Collection,
) (*dispatchWorkerRuntime, error) {
	pulsarClient, err := jobpulsar.NewPulsarClient(pulsarCfg)
	if err != nil {
		return nil, fmt.Errorf("connect pulsar: %w", err)
	}

	producer := jobpulsar.NewPulsarJobProducer(pulsarClient)
	publisher := jobpulsar.NewDispatchPublisher(producer)
	sagaHandler := dispatch.NewDispatchHandler(
		dispatch.DispatchHandlerConfig{MaxAttempts: dispatchCfg.MaxAttempts},
		publisher,
		metadataSvc,
	)

	runtime, err := buildDispatchWorker(ctx, mongoCfg, dispatchCfg, metadataColl, sagaHandler)
	if err != nil {
		_ = producer.Close()
		_ = pulsarClient.Close()
		return nil, err
	}
	runtime.pulsarClient = pulsarClient
	runtime.producer = producer
	return runtime, nil
}

func buildDispatchWorker(
	ctx context.Context,
	mongoCfg dispatchmongodb.MongoConfig,
	dispatchCfg config.MongoDispatchWorkerConfig,
	metadataColl *mongo.Collection,
	handler *dispatch.DispatchHandler,
) (*dispatchWorkerRuntime, error) {
	watchClient, watchColl, err := dispatchmongodb.OpenMongoWatchClient(ctx, mongoCfg, dispatchCfg.StreamMaxPoolSize)
	if err != nil {
		return nil, fmt.Errorf("open change stream watch client: %w", err)
	}

	var tokens dispatchmongodb.ResumeTokenStore = dispatchmongodb.NopResumeTokenStore{}
	if dispatchCfg.StreamMongoDBResumeTokenPath != "" {
		tokens = dispatchmongodb.NewFileResumeTokenStore(dispatchCfg.StreamMongoDBResumeTokenPath)
	}

	streamWatcher := dispatchmongodb.NewStreamWatcher(watchColl, handler, tokens)
	pendingFetcher := dispatchmongodb.NewMongoPendingJobFetcher(metadataColl)
	worker := dispatch.NewWorker(
		dispatch.WorkerConfig{
			PollInterval: dispatchCfg.PollInterval,
			BatchSize:    dispatchCfg.BatchSize,
			MaxAttempts:  dispatchCfg.MaxAttempts,
		},
		handler,
		pendingFetcher,
		streamWatcher,
	)

	return &dispatchWorkerRuntime{
		watchClient: watchClient,
		worker:      worker,
	}, nil
}

// Run blocks until ctx is cancelled (change stream + poll fallback).
func (r *dispatchWorkerRuntime) Run(ctx context.Context) {
	r.worker.Run(ctx)
}

func (r *dispatchWorkerRuntime) Close() {
	if r.producer != nil {
		_ = r.producer.Close()
	}
	if r.pulsarClient != nil {
		_ = r.pulsarClient.Close()
	}
	if r.watchClient != nil {
		_ = r.watchClient.Disconnect(context.Background())
	}
}
