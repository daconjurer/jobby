package main

import (
	"context"
	"fmt"
	"log"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/settings"
)

func main() {
	ctx := context.Background()

	config := metadata.MongoConfig{
		URI:                settings.GetEnvOrPanic("MONGODB_URI"),
		Database:           settings.GetEnvOrPanic("MONGODB_DATABASE"),
		CollectionMetadata: settings.GetEnvOrPanic("MONGODB_COLLECTION_METADATA"),
		CollectionLogs:     settings.GetEnvOrPanic("MONGODB_COLLECTION_LOGS"),
		Timeout:            settings.ParseDuration(settings.GetEnv("MONGODB_TIMEOUT", "10s")),
		MaxPoolSize:        settings.ParseUint64(settings.GetEnv("MONGODB_MAX_POOL_SIZE", "100")),
		MinPoolSize:        settings.ParseUint64(settings.GetEnv("MONGODB_MIN_POOL_SIZE", "10")),
	}

	reader, client, err := metadata.OpenMongoJobsReader(ctx, config)
	if err != nil {
		log.Fatalf("Failed to open MongoDB jobs persistence: %v", err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("mongo disconnect: %v", err)
		}
	}()

	if !reader.IndexesPresent {
		log.Printf("warning: expected indexes missing on one or both collections (see mongo-init)")
	}

	fmt.Println("Jobs CLI initialized successfully")
}
