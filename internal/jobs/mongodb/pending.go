package mongodb

import (
	"context"
	"errors"
	"fmt"

	"github.com/daconjurer/jobby/internal/jobs/dispatch"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// PendingFetcher queries job_metadata for the poll fallback path.
type PendingFetcher struct {
	collection *mongo.Collection
}

var _ dispatch.PendingJobFetcher = (*PendingFetcher)(nil)

// NewMongoPendingJobFetcher returns a fetcher for the given collection.
func NewMongoPendingJobFetcher(coll *mongo.Collection) *PendingFetcher {
	return &PendingFetcher{collection: coll}
}

// FetchPending returns projections on jobs that are still awaiting dispatch with attempts below maxAttempts.
func (f *PendingFetcher) FetchPending(ctx context.Context, maxAttempts int, limit int) (jobs []dispatch.JobDispatchProjection, err error) {
	filter := bson.M{
		"status":           metadata.JobStatusPendingDispatch,
		"dispatchAttempts": bson.M{"$lt": maxAttempts},
		"topic":            bson.M{"$exists": true, "$ne": ""},
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}})
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := f.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("fetch pending dispatch jobs: %w", err)
	}
	defer func() {
		if cerr := cursor.Close(ctx); cerr != nil {
			err = errors.Join(err, fmt.Errorf("close cursor: %w", cerr))
		}
	}()

	for cursor.Next(ctx) {
		var doc metadata.JobMetadataModel
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("decode pending dispatch job: %w", err)
		}
		jobs = append(jobs, dispatch.JobDispatchFromMetadata(&doc))
	}
	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("pending dispatch cursor: %w", err)
	}
	if jobs == nil {
		jobs = []dispatch.JobDispatchProjection{}
	}
	return jobs, nil
}
