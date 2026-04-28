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

	jobsApi, err := metadata.NewMongoJobsApi(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create MongoJobsApi: %v", err)
	}
	defer jobsApi.Close(ctx)

	fmt.Println("Jobs CLI initialized successfully")
}
