// Command jobs-cli operates on job metadata in MongoDB (parity with cmd/jobs-server HTTP API).
//
// Requires MONGODB_URI and related MONGODB_* variables (see .env.example).
//
// Commands: ping (more subcommands in later phases).
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/daconjurer/jobby/cmd/jobs-cli/app"
)

func main() {
	log.SetPrefix("jobs-cli: ")
	log.SetFlags(0)

	if err := run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	mongoCfg, err := loadMongoMetadataConfig()
	if err != nil {
		return fmt.Errorf("load mongodb configuration: %w", err)
	}

	application, cleanup, err := app.Bootstrap(ctx, mongoCfg)
	if err != nil {
		return err
	}
	defer cleanup()

	log.Printf("Connected to MongoDB (%s database)", mongoCfg.Database)

	return newRootCmd(application).Execute()
}
