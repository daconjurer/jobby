package main

import (
	"fmt"
	"log"

	"github.com/daconjurer/jobby/internal/config"
	"github.com/daconjurer/jobby/internal/jobs/mongodb"
)

// jobsCLIConfig holds configuration for the CLI process.
type jobsCLIConfig struct {
	Mongo  mongodb.MongoConfig
	Topics config.JobTopicsConfig
}

func loadConfig() (jobsCLIConfig, error) {
	var cfg jobsCLIConfig
	var err error

	if cfg.Mongo, err = loadMongoMetadataConfig(); err != nil {
		return cfg, configLoadError("mongodb", err)
	}
	if cfg.Topics, err = loadJobTopicsConfig(); err != nil {
		return cfg, configLoadError("job topics", err)
	}
	return cfg, nil
}

func configLoadError(section string, err error) error {
	log.Printf("failed to load %s configuration: %v", section, err)
	return fmt.Errorf("%s: %w", section, err)
}

func loadMongoMetadataConfig() (mongodb.MongoConfig, error) {
	var mc config.MongoConfig
	if err := config.LoadInto(&mc); err != nil {
		return mongodb.MongoConfig{}, fmt.Errorf("parsing mongo config: %w", err)
	}
	if err := mc.Validate(); err != nil {
		return mongodb.MongoConfig{}, fmt.Errorf("validating mongo config: %w", err)
	}
	return mongodb.MongoConfig{
		URI:                mc.URI,
		Database:           mc.Database,
		CollectionMetadata: mc.CollectionMetadata,
		CollectionLogs:     mc.CollectionLogs,
		Timeout:            mc.Timeout,
		MaxPoolSize:        mc.MaxPoolSize,
		MinPoolSize:        mc.MinPoolSize,
	}, nil
}

func loadJobTopicsConfig() (config.JobTopicsConfig, error) {
	var tc config.JobTopicsConfig
	if err := config.LoadInto(&tc); err != nil {
		return config.JobTopicsConfig{}, fmt.Errorf("parsing job topics config: %w", err)
	}
	if err := tc.Validate(); err != nil {
		return config.JobTopicsConfig{}, fmt.Errorf("validating job topics config: %w", err)
	}
	return tc, nil
}
