package metadata

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MongoJobsReader implements JobsReader against MongoDB collections.
type MongoJobsReader struct {
	metadataCollection *mongo.Collection
	logsCollection     *mongo.Collection

	// IndexesPresent is true when every expected index name existed on both collections at construction.
	IndexesPresent bool
}

var _ JobsReader = (*MongoJobsReader)(nil)

// Get retrieves a job metadata by ID (no ID-shape validation; empty string is a normal filter).
func (r *MongoJobsReader) Get(ctx context.Context, jobID string) (JobMetadata, error) {
	filter := bson.M{"jobId": jobID}

	var job JobMetadataModel
	err := r.metadataCollection.FindOne(ctx, filter).Decode(&job)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrJobNotFound
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	return &job, nil
}

// List retrieves job metadata with filtering and pagination.
func (r *MongoJobsReader) List(ctx context.Context, filter ListFilter) (jobs []JobMetadata, err error) {
	query := buildListQuery(filter)

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

	cursor, err := r.metadataCollection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list jobs: %w", err)
	}
	defer func() {
		if cerr := cursor.Close(ctx); cerr != nil {
			err = errors.Join(err, fmt.Errorf("close cursor: %w", cerr))
		}
	}()

	for cursor.Next(ctx) {
		var job JobMetadataModel
		if decodeErr := cursor.Decode(&job); decodeErr != nil {
			return nil, fmt.Errorf("failed to decode job: %w", decodeErr)
		}
		jobs = append(jobs, &job)
	}

	if err = cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if jobs == nil {
		jobs = []JobMetadata{}
	}

	return jobs, nil
}

// CountJobs counts jobs matching the list filter (Limit and Skip are ignored).
func (r *MongoJobsReader) CountJobs(ctx context.Context, filter ListFilter) (int64, error) {
	query := buildListQuery(filter)
	n, err := r.metadataCollection.CountDocuments(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count jobs: %w", err)
	}
	return n, nil
}

// GetJobsByStatus lists jobs in the given status ordered by createdAt descending.
func (r *MongoJobsReader) GetJobsByStatus(ctx context.Context, status JobStatus, limit int) ([]JobMetadata, error) {
	f := ListFilter{
		Statuses: []JobStatus{status},
		SortBy:   "createdAt",
		SortDesc: true,
		Limit:    limit,
	}
	return r.List(ctx, f)
}

// GetPendingJobs lists pending jobs ordered by createdAt descending.
func (r *MongoJobsReader) GetPendingJobs(ctx context.Context, limit int) ([]JobMetadata, error) {
	return r.GetJobsByStatus(ctx, JobStatusPending, limit)
}

// GetLogs retrieves logs for a specific job with optional filtering.
func (r *MongoJobsReader) GetLogs(ctx context.Context, jobID string, filter LogFilter) (logs []JobLog, err error) {
	query := buildLogsQuery(jobID, filter)

	opts := options.Find()

	if filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}

	if filter.Skip > 0 {
		opts.SetSkip(int64(filter.Skip))
	}

	opts.SetSort(bson.D{{Key: "timestamp", Value: -1}})

	cursor, err := r.logsCollection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}
	defer func() {
		if cerr := cursor.Close(ctx); cerr != nil {
			err = errors.Join(err, fmt.Errorf("close cursor: %w", cerr))
		}
	}()

	for cursor.Next(ctx) {
		var lg JobLog
		if decodeErr := cursor.Decode(&lg); decodeErr != nil {
			return nil, fmt.Errorf("failed to decode log: %w", decodeErr)
		}
		logs = append(logs, lg)
	}

	if err = cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %w", err)
	}

	if logs == nil {
		logs = []JobLog{}
	}

	return logs, nil
}

// GetRecentLogs returns logs for jobID newest-first, capped by limit (0 = no limit).
func (r *MongoJobsReader) GetRecentLogs(ctx context.Context, jobID string, limit int) ([]JobLog, error) {
	return r.GetLogs(ctx, jobID, LogFilter{Limit: limit})
}

// GetErrorLogs returns error- and fatal-level logs for jobID, newest-first.
func (r *MongoJobsReader) GetErrorLogs(ctx context.Context, jobID string) ([]JobLog, error) {
	return r.GetLogs(ctx, jobID, LogFilter{
		Levels: []LogLevel{LogLevelError, LogLevelFatal},
	})
}
