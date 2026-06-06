package config

import "time"

// ServerConfig holds HTTP server settings loaded from environment variables.
type ServerConfig struct {
	Port string `env:"APP_PORT,required,notEmpty"`
}

// MigrateConfig holds settings for the cmd/migrate binary (golang-migrate runner).
type MigrateConfig struct {
	URI            string `env:"MONGO_URI,required,notEmpty"`
	MigrationsPath string `env:"MIGRATIONS_PATH" envDefault:"./migrations"`
}

// PulsarConfig holds Apache Pulsar client settings loaded from environment variables.
type PulsarConfig struct {
	ServiceURL       string `env:"PULSAR_SERVICE_URL,required,notEmpty"`
	SubscriptionName string `env:"PULSAR_SUBSCRIPTION_NAME" envDefault:"jobber"`
}

// MongoDispatchWorkerConfig holds change-stream dispatch worker settings.
type MongoDispatchWorkerConfig struct {
	PollInterval                 time.Duration `env:"DISPATCH_POLL_INTERVAL" envDefault:"5s"`
	BatchSize                    int           `env:"DISPATCH_POLL_BATCH_SIZE" envDefault:"50"`
	MaxAttempts                  int           `env:"DISPATCH_MAX_ATTEMPTS" envDefault:"5"`
	StreamMaxPoolSize            uint64        `env:"DISPATCH_STREAM_MAX_POOL_SIZE" envDefault:"2"`
	StreamMongoDBResumeTokenPath string        `env:"DISPATCH_STREAM_MONGODB_RESUME_TOKEN_PATH" envDefault:""`
}

// JobTopicsConfig holds the path to the job name → topic YAML manifest.
type JobTopicsConfig struct {
	ConfigPath string `env:"JOB_TOPICS_CONFIG_PATH" envDefault:"config/job-topics.yaml"`
}

// MongoConfig holds MongoDB connection and pool settings loaded from environment variables.
type MongoConfig struct {
	URI                string        `env:"MONGODB_URI,required,notEmpty"`
	Database           string        `env:"MONGODB_DATABASE,required,notEmpty"`
	CollectionMetadata string        `env:"MONGODB_COLLECTION_METADATA,required,notEmpty"`
	CollectionLogs     string        `env:"MONGODB_COLLECTION_LOGS,required,notEmpty"`
	Timeout            time.Duration `env:"MONGODB_TIMEOUT" envDefault:"10s"`
	MaxPoolSize        uint64        `env:"MONGODB_MAX_POOL_SIZE" envDefault:"100"`
	MinPoolSize        uint64        `env:"MONGODB_MIN_POOL_SIZE" envDefault:"10"`
}
