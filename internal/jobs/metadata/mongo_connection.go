package metadata

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoConfig holds MongoDB connection configuration.
type MongoConfig struct {
	URI                string
	Database           string
	CollectionMetadata string
	CollectionLogs     string
	Timeout            time.Duration
	MaxPoolSize        uint64
	MinPoolSize        uint64
}

// NewMongoJobsReaderWriter verifies indexes and returns reader and writer sharing one database handle.
// The caller owns mongo.Client lifecycle (Disconnect once at shutdown).
func NewMongoJobsReaderWriter(ctx context.Context, db *mongo.Database, cfg MongoConfig) (*MongoJobsReader, *MongoJobsWriter, error) {
	metaColl := db.Collection(cfg.CollectionMetadata)
	logsColl := db.Collection(cfg.CollectionLogs)

	allPresent, err := ensureJobsIndexes(ctx, metaColl, logsColl)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify indexes: %w", err)
	}

	reader := &MongoJobsReader{
		metadataCollection: metaColl,
		logsCollection:     logsColl,
		IndexesPresent:     allPresent,
	}
	writer := &MongoJobsWriter{
		metadataCollection: metaColl,
		logsCollection:     logsColl,
	}
	return reader, writer, nil
}

// OpenMongoJobs connects to MongoDB, verifies indexes, and returns reader, writer, and client.
// Disconnect the client once when tearing down the application.
func OpenMongoJobs(ctx context.Context, cfg MongoConfig) (*MongoJobsReader, *MongoJobsWriter, *mongo.Client, error) {
	clientOpts := options.Client().
		ApplyURI(cfg.URI).
		SetTimeout(cfg.Timeout).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize)

	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, nil, nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(cfg.Database)
	reader, writer, err := NewMongoJobsReaderWriter(ctx, db, cfg)
	if err != nil {
		_ = client.Disconnect(ctx)
		return nil, nil, nil, err
	}

	return reader, writer, client, nil
}

// ensureJobsIndexes checks that expected index names exist. Missing indexes are logged but do not fail startup.
func ensureJobsIndexes(ctx context.Context, metadataColl, logsColl *mongo.Collection) (allPresent bool, err error) {
	metadataRequired := []string{
		"idx_jobId_unique",
		"idx_name",
		"idx_status",
		"idx_createdAt_desc",
		"idx_tags",
		"idx_name_status",
		"idx_status_priority_created",
	}
	metaOK, err := verifyRequiredIndexesPresent(ctx, metadataColl, metadataRequired)
	if err != nil {
		return false, err
	}
	if !metaOK {
		log.Printf("jobby: collection %q is missing one or more expected indexes (see mongo-init); performance may be degraded",
			metadataColl.Name())
	}

	logsRequired := []string{
		"idx_jobId_timestamp_desc",
		"idx_timestamp_desc",
		"idx_level",
		"idx_jobId_level_timestamp",
	}
	logsOK, err := verifyRequiredIndexesPresent(ctx, logsColl, logsRequired)
	if err != nil {
		return false, err
	}
	if !logsOK {
		log.Printf("jobby: collection %q is missing one or more expected indexes (see mongo-init); performance may be degraded",
			logsColl.Name())
	}

	return metaOK && logsOK, nil
}
