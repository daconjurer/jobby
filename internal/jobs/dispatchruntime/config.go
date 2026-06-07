package dispatchruntime

import (
	"github.com/daconjurer/jobby/internal/config"
	"github.com/daconjurer/jobby/internal/jobs/dispatch"
	"github.com/daconjurer/jobby/internal/jobs/mongodb"
)

// StreamConfig controls the dedicated MongoDB change-stream watch client.
type StreamConfig struct {
	MaxPoolSize     uint64
	ResumeTokenPath string // empty → NopResumeTokenStore
}

// Config is the production-shaped settings for assembling a dispatch stack.
type Config struct {
	Mongo  mongodb.MongoConfig
	Worker dispatch.WorkerConfig
	Pulsar config.PulsarConfig
	Stream StreamConfig
}

// ConfigFromEnv maps loaded environment config into dispatch runtime settings.
func ConfigFromEnv(
	mongo mongodb.MongoConfig,
	pulsar config.PulsarConfig,
	dispatchCfg config.MongoDispatchWorkerConfig,
) Config {
	return Config{
		Mongo: mongo,
		Worker: dispatch.WorkerConfig{
			PollInterval: dispatchCfg.PollInterval,
			BatchSize:    dispatchCfg.BatchSize,
			MaxAttempts:  dispatchCfg.MaxAttempts,
		},
		Pulsar: pulsar,
		Stream: StreamConfig{
			MaxPoolSize:     dispatchCfg.StreamMaxPoolSize,
			ResumeTokenPath: dispatchCfg.StreamMongoDBResumeTokenPath,
		},
	}
}
