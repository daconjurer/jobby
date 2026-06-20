package integrationtest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/dispatchruntime"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/mongodb"
	"github.com/daconjurer/jobby/internal/jobs/service"
)

func TestIntegration_DispatchPollFallback_CompletesWithoutChangeStream(t *testing.T) {
	cfg := integrationDispatchConfig(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reader, writer, mongoClient, err := mongodb.OpenMongoJobs(ctx, cfg.Mongo)
	if err != nil {
		t.Fatalf("OpenMongoJobs: %v", err)
	}
	db := mongoClient.Database(cfg.Mongo.Database)
	if err := clearJobCollections(ctx, db, cfg.Mongo); err != nil {
		t.Fatalf("clearJobCollections: %v", err)
	}
	t.Cleanup(func() {
		_ = mongoClient.Disconnect(context.Background())
		_ = clearJobCollections(context.Background(), db, cfg.Mongo)
	})

	metadataSvc := service.NewMetadataService(reader, writer)
	metadataColl := db.Collection(cfg.Mongo.CollectionMetadata)

	const jobName = "account-lifecycle"
	const wantTopic = "persistent://public/default/accounts/jobs"

	// Insert job before the worker starts so only poll can pick it up.
	payload := map[string]any{"source": "poll-fallback-integration"}
	model, err := metadataSvc.CreateJob(ctx, jobName, payload, service.CreateJobOptions{
		Topic: wantTopic,
	})
	if err != nil {
		t.Fatalf("CreateJob: %v", err)
	}
	if model.Status != metadata.JobStatusPendingDispatch {
		t.Fatalf("status=%s want pending_dispatch", model.Status)
	}

	subscription := fmt.Sprintf("integration-poll-%s", metadata.GenerateJobID())
	msgCh := make(chan struct{}, 1)
	errCh := make(chan error, 1)
	go func() {
		_, err := consumeJobMessage(cfg.Pulsar.ServiceURL, wantTopic, subscription, 30*time.Second)
		if err != nil {
			errCh <- err
			return
		}
		msgCh <- struct{}{}
	}()
	time.Sleep(200 * time.Millisecond)

	_, runCancel, runtime := startDispatchRuntime(t, cfg, db, metadataSvc, dispatchruntime.Options{
		StreamRunner:   noopStreamRunner{},
		PendingFetcher: mongodb.NewMongoPendingJobFetcher(metadataColl),
	})
	t.Cleanup(func() {
		runCancel()
		_ = runtime.Close()
	})

	dispatched := waitForJobStatus(t, metadataSvc, model.JobID, metadata.JobStatusDispatched, 30*time.Second)
	if dispatched.DispatchedAt == nil {
		t.Fatal("expected dispatchedAt to be set")
	}

	select {
	case err := <-errCh:
		t.Fatalf("consume: %v", err)
	case <-msgCh:
	case <-time.After(35 * time.Second):
		t.Fatal("timeout waiting for Pulsar message via poll fallback")
	}
}
