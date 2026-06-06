package main

import (
	"fmt"

	"github.com/daconjurer/jobby/internal/config"
	"github.com/daconjurer/jobby/internal/jobs/mongodb"
)

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
