package app

import (
	"context"
	"fmt"
	"log"

	"github.com/daconjurer/jobby/internal/jobs/mongodb"
	"github.com/daconjurer/jobby/internal/jobs/service"
)

// Bootstrap connects to MongoDB, constructs MetadataService, and returns a cleanup function.
func Bootstrap(ctx context.Context, cfg mongodb.MongoConfig) (*App, func(), error) {
	reader, writer, client, err := mongodb.OpenMongoJobs(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("connect mongodb jobs persistence: %w", err)
	}

	if !reader.IndexesPresent {
		log.Println("warning: one or more expected indexes are missing (make sure the migrations are applied)")
	}

	cleanup := func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Printf("mongo disconnect: %v", err)
		}
	}

	return New(service.NewMetadataService(reader, writer), writer), cleanup, nil
}
