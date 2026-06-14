package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/daconjurer/jobby/internal/jobs/mongodb"
	"github.com/daconjurer/jobby/internal/jobs/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("cannot start jobs-dispatcher: %v", err)
	}

	reader, writer, mongoClient, err := mongodb.OpenMongoJobs(ctx, cfg.Mongo)
	if err != nil {
		log.Fatalf("failed to connect MongoDB jobs persistence: %v", err)
	}
	defer func() { _ = mongoClient.Disconnect(context.Background()) }()

	log.Println("connected to MongoDB (jobby database)")
	if !reader.IndexesPresent {
		log.Println("warning: one or more expected indexes are missing (make sure the migrations are applied)")
	}

	db := mongoClient.Database(cfg.Mongo.Database)
	metadataSvc := service.NewMetadataService(reader, writer)

	runtime, err := newDispatchRuntime(
		ctx,
		cfg,
		metadataSvc,
		db.Collection(cfg.Mongo.CollectionMetadata),
	)
	if err != nil {
		log.Fatalf("failed to start dispatch worker: %v", err)
	}
	defer func() { _ = runtime.Close() }()

	log.Println("dispatch worker running (change stream + poll fallback)")
	runtime.Run(ctx)
	log.Println("dispatch worker stopped")
}
