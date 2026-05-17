package main

import (
	"github.com/daconjurer/jobby/internal/config"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

func loadMongoMetadataConfig() (metadata.MongoConfig, error) {
	var mc config.MongoConfig
	if err := config.LoadInto(&mc); err != nil {
		return metadata.MongoConfig{}, err
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
	if err := config.LoadInto(&sc); err != nil {
		return config.ServerConfig{}, err
	}
	return sc, nil
}
