package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/daconjurer/jobby/internal/jobs/executor"
	"github.com/daconjurer/jobby/internal/jobs/executor/handlers"
	"github.com/daconjurer/jobby/internal/jobs/mongodb"
	"github.com/daconjurer/jobby/internal/jobs/pulsar"
	"github.com/daconjurer/jobby/internal/jobs/service"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("cannot start jobs-executor: %v", err)
	}

	reader, writer, mongoClient, err := mongodb.OpenMongoJobs(ctx, cfg.Mongo)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			log.Printf("mongo disconnect: %v", err)
		}
	}()

	if !reader.IndexesPresent {
		log.Println("warning: one or more expected indexes are missing (make sure the migrations are applied)")
	}

	log.Println("Connected to MongoDB (jobby database)")

	topicResolver, err := pulsar.NewFileTopicResolver(cfg.Topics.ConfigPath)
	if err != nil {
		log.Fatalf("Failed to load job topics config: %v", err)
	}

	pulsarClient, err := pulsar.NewPulsarClient(cfg.Pulsar)
	if err != nil {
		log.Fatalf("Failed to create Pulsar client: %v", err)
	}
	defer func() {
		if err := pulsarClient.Close(); err != nil {
			log.Printf("pulsar client close: %v", err)
		}
	}()

	log.Println("Connected to Pulsar")

	topics := topicResolver.UniqueTopics()
	log.Printf("Subscribing to topics: %v", topics)

	consumer, err := pulsar.NewPulsarJobConsumer(pulsarClient, topics)
	if err != nil {
		log.Fatalf("Failed to create Pulsar consumer: %v", err)
	}
	defer func() {
		if err := consumer.Close(); err != nil {
			log.Printf("consumer close: %v", err)
		}
	}()

	metadataSvc := service.NewMetadataService(reader, writer)

	registry := executor.NewRegistry()
	executor.Register(registry, "echo", &handlers.EchoHandler{})

	log.Println("Registered job handlers: echo")

	execSvc := executor.NewExecutorService(metadataSvc, registry)

	log.Println("Starting job executor consumer loop")

	if err := consumer.Run(ctx, execSvc.ExecuteJob); err != nil && err != context.Canceled {
		log.Fatalf("Consumer error: %v", err)
	}

	log.Println("Executor shutdown complete")
}
