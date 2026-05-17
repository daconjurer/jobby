package main

import (
	"fmt"

	"github.com/daconjurer/jobby/internal/config"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

func loadMongoMetadataConfig() (metadata.MongoConfig, error) {
	var mc config.MongoConfig
	if err := config.LoadIntoWithOptions(&mc, config.LoadOptionsFromEnv()); err != nil {
		return metadata.MongoConfig{}, fmt.Errorf("parsing mongo config: %w", err)
	}
	if err := mc.Validate(); err != nil {
		return metadata.MongoConfig{}, fmt.Errorf("validating mongo config: %w", err)
	}
	return metadata.MongoConfig{
		URI:                mc.URI,
		Database:           mc.Database,
		CollectionMetadata: mc.CollectionMetadata,
		CollectionLogs:     mc.CollectionLogs,
		Timeout:            mc.Timeout,
		MaxPoolSize:        mc.MaxPoolSize,
		MinPoolSize:        mc.MinPoolSize,
	}, nil
}

func loadServerListenConfig() (config.ServerConfig, error) {
	var sc config.ServerConfig
	if err := config.LoadIntoWithOptions(&sc, config.LoadOptionsFromEnv()); err != nil {
		return config.ServerConfig{}, fmt.Errorf("parsing server config: %w", err)
	}
	if err := sc.Validate(); err != nil {
		return config.ServerConfig{}, fmt.Errorf("validating server config: %w", err)
	}
	return sc, nil
}
