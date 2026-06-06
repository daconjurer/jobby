package mongodb

import (
	"context"
	"fmt"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// MongoJobsWriter implements JobsWriter against MongoDB collections.
type MongoJobsWriter struct {
	metadataCollection *mongo.Collection
	logsCollection     *mongo.Collection
}

var _ metadata.JobsWriter = (*MongoJobsWriter)(nil)

// Create inserts a new job metadata record (no domain validation).
func (w *MongoJobsWriter) Create(ctx context.Context, job metadata.JobMetadata) error {
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
func (w *MongoJobsWriter) Update(ctx context.Context, jobID string, patch metadata.UpdateJob) error {
	setDoc, err := bsonPartialSet(&patch)
	if err != nil {
		return fmt.Errorf("build update job patch: %w", err)
	}
	if len(setDoc) == 0 {
		return metadata.ErrEmptyUpdateJob
	}

	filter := bson.M{"jobId": jobID}
	result, err := w.metadataCollection.UpdateOne(ctx, filter, bson.M{"$set": setDoc})
	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	if result.MatchedCount == 0 {
		return metadata.ErrJobNotFound
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
		return metadata.ErrJobNotFound
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
		return metadata.ErrJobNotFound
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
		return metadata.ErrJobNotFound
	}

	return nil
}

// InsertJobs bulk-inserts job metadata documents.
func (w *MongoJobsWriter) InsertJobs(ctx context.Context, jobs []*JobMetadataModel) (int, error) {
	if len(jobs) == 0 {
		return 0, nil
	}

	docs := make([]any, len(jobs))
	for i, job := range jobs {
		docs[i] = job
	}

	result, err := w.metadataCollection.InsertMany(ctx, docs)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk insert jobs: %w", err)
	}
	return len(result.InsertedIDs), nil
}

// InsertLogs bulk-inserts job log documents.
func (w *MongoJobsWriter) InsertLogs(ctx context.Context, logs []JobLog) (int, error) {
	if len(logs) == 0 {
		return 0, nil
	}

	docs := make([]any, len(logs))
	for i := range logs {
		docs[i] = logs[i]
	}

	result, err := w.logsCollection.InsertMany(ctx, docs)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk insert logs: %w", err)
	}
	return len(result.InsertedIDs), nil
}

// AddLog appends a log entry (no field validation; collection JSON schema enforces shape).
func (w *MongoJobsWriter) AddLog(ctx context.Context, log metadata.JobLog) error {
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

// MarkDispatchedIfPending transitions a job with matching jobId as pending_dispatch → dispatched and sets dispatchedAt.
// Returns false when no document matched (already dispatched or terminal).
func (w *MongoJobsWriter) MarkDispatchedIfPending(ctx context.Context, jobID string, dispatchedAt time.Time) (bool, error) {
	filter := bson.M{
		"jobId":  jobID,
		"status": metadata.JobStatusPendingDispatch,
	}
	update := bson.M{
		"$set": bson.M{
			"status":       metadata.JobStatusDispatched,
			"dispatchedAt": dispatchedAt,
		},
	}
	result, err := w.metadataCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return false, fmt.Errorf("mark job dispatched: %w", err)
	}
	return result.MatchedCount > 0, nil
}

// RecordDispatchAttemptIfPending increments dispatch attempt metadata while still pending_dispatch.
func (w *MongoJobsWriter) RecordDispatchAttemptIfPending(ctx context.Context, jobID string, attempts int, lastError string) (bool, error) {
	filter := bson.M{
		"jobId":  jobID,
		"status": metadata.JobStatusPendingDispatch,
	}
	update := bson.M{
		"$set": bson.M{
			"dispatchAttempts":  attempts,
			"dispatchLastError": lastError,
		},
	}
	result, err := w.metadataCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		return false, fmt.Errorf("record dispatch attempt: %w", err)
	}
	return result.MatchedCount > 0, nil
}
