package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/daconjurer/jobby/internal/jobs/dispatch"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// insertPendingDispatchPipeline matches new pending_dispatch inserts only.
var insertPendingDispatchPipeline = mongo.Pipeline{
	bson.D{{Key: "$match", Value: bson.D{
		{Key: "operationType", Value: "insert"},
		{Key: "fullDocument.status", Value: metadata.JobStatusPendingDispatch},
	}}},
}

// StreamWatcher consumes MongoDB change stream insert events.
type StreamWatcher struct {
	collection *mongo.Collection
	handler    dispatch.JobDispatchHandler
	tokens     ResumeTokenStore
}

// NewStreamWatcher creates a change stream consumer for job_metadata inserts.
func NewStreamWatcher(collection *mongo.Collection, handler dispatch.JobDispatchHandler, tokens ResumeTokenStore) *StreamWatcher {
	if tokens == nil {
		tokens = NopResumeTokenStore{}
	}
	return &StreamWatcher{
		collection: collection,
		handler:    handler,
		tokens:     tokens,
	}
}

// Run watches inserts until ctx is cancelled; reconnects after errors.
func (w *StreamWatcher) Run(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		if err := w.runOnce(ctx); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("dispatch change stream error: %v", err)
		}
	}
}

func (w *StreamWatcher) runOnce(ctx context.Context) (err error) {
	opts := options.ChangeStream().SetFullDocument(options.UpdateLookup)
	token, err := w.tokens.Load()
	if err != nil {
		return err
	}
	if token != nil {
		opts.SetResumeAfter(token)
	}

	stream, err := w.collection.Watch(ctx, insertPendingDispatchPipeline, opts)
	if err != nil {
		return fmt.Errorf("open change stream: %w", err)
	}
	defer func() {
		if cerr := stream.Close(ctx); cerr != nil {
			err = errors.Join(err, fmt.Errorf("close change stream: %w", cerr))
		}
	}()

	for stream.Next(ctx) {
		if err := w.handleEvent(ctx, stream); err != nil {
			log.Printf("dispatch stream event job error: %v", err)
		}
		if token := stream.ResumeToken(); token != nil {
			if err := w.tokens.Save(token); err != nil {
				log.Printf("dispatch mongodb resume token save error: %v", err)
			}
		}
	}
	if err := stream.Err(); err != nil {
		return fmt.Errorf("change stream cursor: %w", err)
	}
	return nil
}

func (w *StreamWatcher) handleEvent(ctx context.Context, stream *mongo.ChangeStream) error {
	var evt struct {
		FullDocument metadata.JobMetadataModel `bson:"fullDocument"`
	}
	if err := stream.Decode(&evt); err != nil {
		return fmt.Errorf("decode change event: %w", err)
	}
	return handlePendingDispatchInsert(ctx, w.handler, &evt.FullDocument)
}

func handlePendingDispatchInsert(ctx context.Context, handler dispatch.JobDispatchHandler, doc *metadata.JobMetadataModel) error {
	if doc.Topic == "" {
		return fmt.Errorf("inserted job %s missing topic", doc.JobID)
	}
	return handler.HandleDispatch(ctx, dispatch.JobDispatchFromMetadata(doc))
}
