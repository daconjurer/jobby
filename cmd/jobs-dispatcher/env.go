package main

import (
	"fmt"
	"log"

	"github.com/daconjurer/jobby/internal/config"
	"github.com/daconjurer/jobby/internal/jobs/mongodb"
)

// dispatcherConfig holds configuration for the dispatch worker process.
type dispatcherConfig struct {
	Mongo    mongodb.MongoConfig
	Pulsar   config.PulsarConfig
	Dispatch config.MongoDispatchWorkerConfig
}

func loadConfig() (dispatcherConfig, error) {
	var cfg dispatcherConfig
	var err error

	if cfg.Mongo, err = loadMongoMetadataConfig(); err != nil {
		return cfg, configLoadError("mongodb", err)
	}
	if cfg.Pulsar, err = loadPulsarConfig(); err != nil {
		return cfg, configLoadError("pulsar", err)
	}
	if cfg.Dispatch, err = loadDispatchWorkerConfig(); err != nil {
		return cfg, configLoadError("dispatch worker", err)
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

func loadPulsarConfig() (config.PulsarConfig, error) {
	var pc config.PulsarConfig
	if err := config.LoadInto(&pc); err != nil {
		return config.PulsarConfig{}, fmt.Errorf("parsing pulsar config: %w", err)
	}
	if err := pc.Validate(); err != nil {
		return config.PulsarConfig{}, fmt.Errorf("validating pulsar config: %w", err)
	}
	return pc, nil
}

func loadDispatchWorkerConfig() (config.MongoDispatchWorkerConfig, error) {
	var dc config.MongoDispatchWorkerConfig
	if err := config.LoadInto(&dc); err != nil {
		return config.MongoDispatchWorkerConfig{}, fmt.Errorf("parsing dispatch worker config: %w", err)
	}
	if err := dc.Validate(); err != nil {
		return config.MongoDispatchWorkerConfig{}, fmt.Errorf("validating dispatch worker config: %w", err)
	}
	return dc, nil
}
