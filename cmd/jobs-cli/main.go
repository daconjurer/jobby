package main

import (
	"context"
	"fmt"
	"log"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

func main() {
	ctx := context.Background()

	mongoCfg, err := loadMongoMetadataConfig()
	if err != nil {
		log.Fatalf("Failed to load MongoDB configuration: %v", err)
	}

	reader, client, err := metadata.OpenMongoJobsReader(ctx, mongoCfg)
	if err != nil {
		log.Fatalf("Failed to open MongoDB jobs persistence: %v", err)
	}
	defer func() {
		if err := client.Disconnect(ctx); err != nil {
			log.Printf("mongo disconnect: %v", err)
		}
	}()

	if !reader.IndexesPresent {
		log.Printf("warning: expected indexes missing on one or both collections (run `go run ./cmd/migrate up` — see migrations/README.md)")
	}

	fmt.Println("Jobs CLI initialized successfully")
}
