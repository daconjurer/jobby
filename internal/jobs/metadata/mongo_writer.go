package metadata

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// MongoJobsWriter implements JobsWriter against MongoDB collections.
type MongoJobsWriter struct {
	metadataCollection *mongo.Collection
	logsCollection     *mongo.Collection
}

var _ JobsWriter = (*MongoJobsWriter)(nil)

// Create inserts a new job metadata record (no domain validation).
func (w *MongoJobsWriter) Create(ctx context.Context, job JobMetadata) error {
	_, err := w.metadataCollection.InsertOne(ctx, job)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return fmt.Errorf("job with ID %s already exists", job.GetJobID())
		}
		return fmt.Errorf("failed to insert job: %w", err)
	}

	return nil
}

// Update applies patch fields with a single $set (no domain validation).
func (w *MongoJobsWriter) Update(ctx context.Context, jobID string, patch UpdateJob) error {
	setDoc, err := bsonPartialSet(&patch)
	if err != nil {
		return fmt.Errorf("build update job patch: %w", err)
	}
	if len(setDoc) == 0 {
		return ErrEmptyUpdateJob
	}

	filter := bson.M{"jobId": jobID}
	result, err := w.metadataCollection.UpdateOne(ctx, filter, bson.M{"$set": setDoc})
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrJobNotFound
	}

	return nil
}

// Delete removes a job metadata record.
func (w *MongoJobsWriter) Delete(ctx context.Context, jobID string) error {
	filter := bson.M{"jobId": jobID}

	result, err := w.metadataCollection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	if result.DeletedCount == 0 {
		return ErrJobNotFound
	}

	return nil
}

// IncrementRetryCount increments the retry counter atomically.
func (w *MongoJobsWriter) IncrementRetryCount(ctx context.Context, jobID string) error {
	filter := bson.M{"jobId": jobID}
	update := bson.M{
		"$inc": bson.M{"retryCount": 1},
	}

	result, err := w.metadataCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrJobNotFound
	}

	return nil
}

// ClearJobExecutionTimestamps removes startedAt and completedAt via $unset.
func (w *MongoJobsWriter) ClearJobExecutionTimestamps(ctx context.Context, jobID string) error {
	filter := bson.M{"jobId": jobID}
	update := bson.M{
		"$unset": bson.M{
			"startedAt":   "",
			"completedAt": "",
		},
	}

	result, err := w.metadataCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to clear job timestamps: %w", err)
	}

	if result.MatchedCount == 0 {
		return ErrJobNotFound
	}

	return nil
}

// AddLog appends a log entry (no field validation; collection JSON schema enforces shape).
func (w *MongoJobsWriter) AddLog(ctx context.Context, log JobLog) error {
	_, err := w.logsCollection.InsertOne(ctx, log)
	if err != nil {
		return fmt.Errorf("failed to insert log: %w", err)
	}

	return nil
}

// DeleteOldLogs removes log documents with timestamp older than now minus olderThan.
func (w *MongoJobsWriter) DeleteOldLogs(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().UTC().Add(-olderThan)
	res, err := w.logsCollection.DeleteMany(ctx, bson.M{
		"timestamp": bson.M{"$lt": cutoff},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to delete old logs: %w", err)
	}
	return res.DeletedCount, nil
}
