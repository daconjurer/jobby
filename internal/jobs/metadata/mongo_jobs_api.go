package metadata

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrJobNotFound     = errors.New("job not found")
	ErrInvalidJobID    = errors.New("invalid job ID")
	ErrInvalidStatus   = errors.New("invalid job status")
	ErrInvalidLogLevel = errors.New("invalid log level")
)

// MongoJobsApi is the MongoDB implementation of the JobsApi interface.
type MongoJobsApi struct {
	db                 *mongo.Database
	metadataCollection *mongo.Collection
	logsCollection     *mongo.Collection
}

var _ JobsApi = (*MongoJobsApi)(nil)

// MongoConfig holds MongoDB connection configuration
type MongoConfig struct {
	URI                string
	Database           string
	CollectionMetadata string
	CollectionLogs     string
	Timeout            time.Duration
	MaxPoolSize        uint64
	MinPoolSize        uint64
}

// NewMongoJobsApi creates a new MongoDB-based JobsApi implementation
func NewMongoJobsApi(ctx context.Context, config MongoConfig) (*MongoJobsApi, error) {
	clientOpts := options.Client().
		ApplyURI(config.URI).
		SetTimeout(config.Timeout).
		SetMaxPoolSize(config.MaxPoolSize).
		SetMinPoolSize(config.MinPoolSize)

	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(config.Database)
	metadataCollection := db.Collection(config.CollectionMetadata)
	logsCollection := db.Collection(config.CollectionLogs)

	api := &MongoJobsApi{
		db:                 db,
		metadataCollection: metadataCollection,
		logsCollection:     logsCollection,
	}

	if err := api.ensureIndexes(ctx); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return api, nil
}

// ensureIndexes creates necessary indexes for optimal query performance
func (m *MongoJobsApi) ensureIndexes(ctx context.Context) error {
	metadataIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "jobId", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetName("idx_jobId_unique"),
		},
		{
			Keys: bson.D{{Key: "status", Value: 1}},
			Options: options.Index().
				SetName("idx_status"),
		},
		{
			Keys: bson.D{{Key: "name", Value: 1}},
			Options: options.Index().
				SetName("idx_name"),
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "priority", Value: -1},
				{Key: "createdAt", Value: -1},
			},
			Options: options.Index().
				SetName("idx_status_priority_created"),
		},
		{
			Keys: bson.D{{Key: "tags", Value: 1}},
			Options: options.Index().
				SetName("idx_tags"),
		},
		{
			Keys: bson.D{{Key: "createdAt", Value: -1}},
			Options: options.Index().
				SetName("idx_createdAt"),
		},
	}

	if _, err := m.metadataCollection.Indexes().CreateMany(ctx, metadataIndexes); err != nil {
		return fmt.Errorf("failed to create metadata indexes: %w", err)
	}

	logsIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "jobId", Value: 1},
				{Key: "timestamp", Value: -1},
			},
			Options: options.Index().
				SetName("idx_jobId_timestamp"),
		},
		{
			Keys: bson.D{{Key: "level", Value: 1}},
			Options: options.Index().
				SetName("idx_level"),
		},
	}

	if _, err := m.logsCollection.Indexes().CreateMany(ctx, logsIndexes); err != nil {
		return fmt.Errorf("failed to create logs indexes: %w", err)
	}

	return nil
}

// Create inserts a new job metadata record
func (m *MongoJobsApi) Create(ctx context.Context, job JobMetadata) error {
	if job == nil {
		return errors.New("job cannot be nil")
	}

	if err := job.Validate(); err != nil {
		return fmt.Errorf("job validation failed: %w", err)
	}

	model, ok := job.(*JobMetadataModel)
	if !ok {
		return errors.New("job must be of type *JobMetadataModel")
	}

	_, err := m.metadataCollection.InsertOne(ctx, model)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("job with ID %s already exists", job.GetJobID())
		}
		return fmt.Errorf("failed to insert job: %w", err)
	}

	return nil
}

// Get retrieves a job metadata by ID
func (m *MongoJobsApi) Get(ctx context.Context, jobID string) (JobMetadata, error) {
	if jobID == "" {
		return nil, ErrInvalidJobID
	}

	filter := bson.M{"jobId": jobID}

	var job JobMetadataModel
	err := m.metadataCollection.FindOne(ctx, filter).Decode(&job)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &job, nil
}

// Update updates an existing job metadata record
func (m *MongoJobsApi) Update(ctx context.Context, job JobMetadata) error {
	if job == nil {
		return errors.New("job cannot be nil")
	}

	if err := job.Validate(); err != nil {
		return fmt.Errorf("job validation failed: %w", err)
	}

	model, ok := job.(*JobMetadataModel)
	if !ok {
		return errors.New("job must be of type *JobMetadataModel")
	}

	filter := bson.M{"jobId": model.JobID}
	update := bson.M{"$set": model}

	result, err := m.metadataCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrJobNotFound
	}

	return nil
}

// Delete removes a job metadata record
func (m *MongoJobsApi) Delete(ctx context.Context, jobID string) error {
	if jobID == "" {
		return ErrInvalidJobID
	}

	filter := bson.M{"jobId": jobID}

	result, err := m.metadataCollection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	if result.DeletedCount == 0 {
		return ErrJobNotFound
	}

	return nil
}

// List retrieves job metadata with filtering and pagination
func (m *MongoJobsApi) List(ctx context.Context, filter ListFilter) ([]JobMetadata, error) {
	query := m.buildListQuery(filter)

	opts := options.Find()

	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}

	if filter.Skip > 0 {
		opts.SetSkip(int64(filter.Skip))
	}

	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "createdAt"
	}

	sortOrder := -1
	if !filter.SortDesc {
		sortOrder = 1
	}
	opts.SetSort(bson.D{{Key: sortBy, Value: sortOrder}})

	cursor, err := m.metadataCollection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer cursor.Close(ctx)

	var jobs []JobMetadata
	for cursor.Next(ctx) {
		var job JobMetadataModel
		if err := cursor.Decode(&job); err != nil {
			return nil, fmt.Errorf("failed to decode job: %w", err)
		}
		jobs = append(jobs, &job)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if jobs == nil {
		jobs = []JobMetadata{}
	}

	return jobs, nil
}

// buildListQuery constructs a MongoDB query from the ListFilter
func (m *MongoJobsApi) buildListQuery(filter ListFilter) bson.M {
	query := bson.M{}

	if len(filter.Names) > 0 {
		query["name"] = bson.M{"$in": filter.Names}
	}

	if len(filter.Statuses) > 0 {
		query["status"] = bson.M{"$in": filter.Statuses}
	}

	if len(filter.Tags) > 0 {
		query["tags"] = bson.M{"$in": filter.Tags}
	}

	if filter.MinPriority != nil || filter.MaxPriority != nil {
		priorityQuery := bson.M{}
		if filter.MinPriority != nil {
			priorityQuery["$gte"] = *filter.MinPriority
		}
		if filter.MaxPriority != nil {
			priorityQuery["$lte"] = *filter.MaxPriority
		}
		query["priority"] = priorityQuery
	}

	if filter.CreatedAfter != nil || filter.CreatedBefore != nil {
		createdQuery := bson.M{}
		if filter.CreatedAfter != nil {
			createdQuery["$gt"] = *filter.CreatedAfter
		}
		if filter.CreatedBefore != nil {
			createdQuery["$lt"] = *filter.CreatedBefore
		}
		query["createdAt"] = createdQuery
	}

	return query
}

// UpdateStatus updates the job status and related timestamps atomically
func (m *MongoJobsApi) UpdateStatus(ctx context.Context, jobID string, status JobStatus) error {
	if jobID == "" {
		return ErrInvalidJobID
	}

	if !status.IsValid() {
		return ErrInvalidStatus
	}

	job, err := m.Get(ctx, jobID)
	if err != nil {
		return err
	}

	model := job.(*JobMetadataModel)

	if !model.Status.CanTransitionTo(status) {
		return fmt.Errorf("cannot transition from %s to %s", model.Status, status)
	}

	update := bson.M{
		"$set": bson.M{
			"status": status,
		},
	}

	now := time.Now()

	switch status {
	case JobStatusRunning:
		if model.StartedAt == nil {
			update["$set"].(bson.M)["startedAt"] = now
		}
	case JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		if model.CompletedAt == nil {
			update["$set"].(bson.M)["completedAt"] = now
		}
	}

	filter := bson.M{"jobId": jobID}

	result, err := m.metadataCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrJobNotFound
	}

	return nil
}

// IncrementRetryCount increments the retry counter atomically
func (m *MongoJobsApi) IncrementRetryCount(ctx context.Context, jobID string) error {
	if jobID == "" {
		return ErrInvalidJobID
	}

	filter := bson.M{"jobId": jobID}
	update := bson.M{
		"$inc": bson.M{"retryCount": 1},
	}

	result, err := m.metadataCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrJobNotFound
	}

	return nil
}

// AddLog appends a log entry for the specified job
func (m *MongoJobsApi) AddLog(ctx context.Context, log JobLog) error {
	if log.JobID == "" {
		return ErrInvalidJobID
	}

	if !log.Level.IsValid() {
		return ErrInvalidLogLevel
	}

	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	_, err := m.logsCollection.InsertOne(ctx, log)
	if err != nil {
		return fmt.Errorf("failed to insert log: %w", err)
	}

	return nil
}

// GetLogs retrieves logs for a specific job with optional filtering
func (m *MongoJobsApi) GetLogs(ctx context.Context, jobID string, filter LogFilter) ([]JobLog, error) {
	if jobID == "" {
		return nil, ErrInvalidJobID
	}

	query := m.buildLogsQuery(jobID, filter)

	opts := options.Find()

	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}

	if filter.Skip > 0 {
		opts.SetSkip(int64(filter.Skip))
	}

	opts.SetSort(bson.D{{Key: "timestamp", Value: -1}})

	cursor, err := m.logsCollection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}
	defer cursor.Close(ctx)

	var logs []JobLog
	for cursor.Next(ctx) {
		var log JobLog
		if err := cursor.Decode(&log); err != nil {
			return nil, fmt.Errorf("failed to decode log: %w", err)
		}
		logs = append(logs, log)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if logs == nil {
		logs = []JobLog{}
	}

	return logs, nil
}

// buildLogsQuery constructs a MongoDB query from the LogFilter
func (m *MongoJobsApi) buildLogsQuery(jobID string, filter LogFilter) bson.M {
	query := bson.M{"jobId": jobID}

	if len(filter.Levels) > 0 {
		query["level"] = bson.M{"$in": filter.Levels}
	}

	if filter.Since != nil || filter.Until != nil {
		timestampQuery := bson.M{}
		if filter.Since != nil {
			timestampQuery["$gte"] = *filter.Since
		}
		if filter.Until != nil {
			timestampQuery["$lte"] = *filter.Until
		}
		query["timestamp"] = timestampQuery
	}

	return query
}

// Close closes the MongoDB connection
func (m *MongoJobsApi) Close(ctx context.Context) error {
	return m.db.Client().Disconnect(ctx)
}
