package main

import (
	"context"

	"github.com/daconjurer/jobby/internal/jobs/dispatchruntime"
	"github.com/daconjurer/jobby/internal/jobs/service"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func newDispatchRuntime(
	ctx context.Context,
	cfg dispatcherConfig,
	metadataSvc *service.MetadataService,
	metadataColl *mongo.Collection,
) (*dispatchruntime.Runtime, error) {
	return dispatchruntime.New(
		ctx,
		dispatchruntime.ConfigFromEnv(cfg.Mongo, cfg.Pulsar, cfg.Dispatch),
		metadataColl,
		metadataSvc,
		dispatchruntime.Options{},
	)
}
