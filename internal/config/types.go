package config

import "time"

// ServerConfig holds HTTP server settings loaded from environment variables.
type ServerConfig struct {
	Port string `env:"APP_PORT,required,notEmpty"`
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
