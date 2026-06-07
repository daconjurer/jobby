package appruntime

import (
	"context"
	"fmt"
	"log"

	"github.com/daconjurer/jobby/internal/jobs/mongodb"
	jobpulsar "github.com/daconjurer/jobby/internal/jobs/pulsar"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Config holds shared wiring inputs for jobs-server and jobs-cli.
type Config struct {
	Mongo            mongodb.MongoConfig
	TopicsConfigPath string
}

// Runtime holds MongoDB persistence and application services shared by HTTP and CLI entrypoints.
type Runtime struct {
	Reader      *mongodb.MongoJobsReader
	Writer      *mongodb.MongoJobsWriter
	MongoClient *mongo.Client
	Metadata    *service.MetadataService
	Enqueue     *service.EnqueueService
}

// Bootstrap connects to MongoDB, loads job topics, and wires MetadataService and EnqueueService.
func Bootstrap(ctx context.Context, cfg Config) (*Runtime, func(), error) {
	reader, writer, mongoClient, err := mongodb.OpenMongoJobs(ctx, cfg.Mongo)
	if err != nil {
		return nil, nil, fmt.Errorf("connect mongodb jobs persistence: %w", err)
	}

	if !reader.IndexesPresent {
		log.Println("warning: one or more expected indexes are missing (make sure the migrations are applied)")
	}

	topicResolver, err := jobpulsar.NewFileTopicResolver(cfg.TopicsConfigPath)
	if err != nil {
		_ = mongoClient.Disconnect(context.Background())
		return nil, nil, fmt.Errorf("load job topics config: %w", err)
	}

	metadataSvc := service.NewMetadataService(reader, writer)
	enqueueSvc := service.NewEnqueueService(metadataSvc, topicResolver)

	cleanup := func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			log.Printf("mongo disconnect: %v", err)
		}
	}

	return &Runtime{
		Reader:      reader,
		Writer:      writer,
		MongoClient: mongoClient,
		Metadata:    metadataSvc,
		Enqueue:     enqueueSvc,
	}, cleanup, nil
}
