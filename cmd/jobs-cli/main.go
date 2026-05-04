package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

func main() {
	ctx := context.Background()

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://jobby_app:jobby_app_pass@localhost:27018/jobby?authSource=jobby"
	}

	config := metadata.MongoConfig{
		URI:                mongoURI,
		Database:           "jobby",
		CollectionMetadata: "job_metadata",
		CollectionLogs:     "job_logs",
		Timeout:            10 * time.Second,
		MaxPoolSize:        100,
		MinPoolSize:        10,
	}

	reader, _, client, err := metadata.OpenMongoJobs(ctx, config)
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
