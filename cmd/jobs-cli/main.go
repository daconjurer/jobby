// Command jobs-cli operates on job metadata in MongoDB (parity with cmd/jobs-server HTTP API).
//
// Requires MONGODB_URI and related MONGODB_* variables (see .env.example).
//
// Commands: ping, create, get, list, stats, fail, cancel, retry, logs, seed.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/daconjurer/jobby/cmd/jobs-cli/cli"
	"github.com/daconjurer/jobby/internal/jobs/appruntime"
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

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	rt, cleanup, err := appruntime.Bootstrap(ctx, appruntime.Config{
		Mongo:            cfg.Mongo,
		TopicsConfigPath: cfg.Topics.ConfigPath,
	})
	if err != nil {
		return err
	}
	defer cleanup()

	log.Printf("Connected to MongoDB (%s database)", cfg.Mongo.Database)

	c := cli.New(rt.Metadata, rt.Enqueue, rt.Writer)
	return newRootCmd(c).Execute()
}
