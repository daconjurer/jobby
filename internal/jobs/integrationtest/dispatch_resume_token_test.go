
package integrationtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/daconjurer/jobby/internal/jobs/dispatchruntime"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/mongodb"
	jobpulsar "github.com/daconjurer/jobby/internal/jobs/pulsar"
	"github.com/daconjurer/jobby/internal/jobs/service"
)

func TestIntegration_DispatchResumeToken_RestartProcessesMissedJob(t *testing.T) {
	cfg := integrationDispatchConfig(t)

	ctx := context.Background()

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

	tokenPath := filepath.Join(t.TempDir(), "resume-token.json")
	tokens := mongodb.NewFileResumeTokenStore(tokenPath)

	const jobName = "account-lifecycle"
	const wantTopic = "persistent://public/default/accounts/jobs"

	subscription := fmt.Sprintf("integration-resume-%s", metadata.GenerateJobID())
	msgCounts := make(map[string]int)
	var mu sync.Mutex
	consumerDone := make(chan struct{})
	consumerErr := make(chan error, 1)

	go func() {
		defer close(consumerDone)
		client, err := pulsar.NewClient(pulsar.ClientOptions{URL: cfg.Pulsar.ServiceURL})
		if err != nil {
			consumerErr <- fmt.Errorf("pulsar.NewClient: %w", err)
			return
		}
		defer client.Close()

		consumer, err := client.Subscribe(pulsar.ConsumerOptions{
			Topic:                       wantTopic,
			SubscriptionName:            subscription,
			Type:                        pulsar.Shared,
			SubscriptionInitialPosition: pulsar.SubscriptionPositionLatest,
		})
		if err != nil {
			consumerErr <- fmt.Errorf("Subscribe: %w", err)
			return
		}
		defer consumer.Close()

		deadline := time.Now().Add(60 * time.Second)
		for time.Now().Before(deadline) {
			recvCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			raw, err := consumer.Receive(recvCtx)
			cancel()
			if err != nil {
				if recvCtx.Err() != nil {
					continue
				}
				consumerErr <- fmt.Errorf("Receive: %w", err)
				return
			}
			_ = consumer.Ack(raw)
			msg, err := jobpulsar.Unmarshal(raw.Payload())
			if err != nil {
				consumerErr <- fmt.Errorf("Unmarshal: %w", err)
				return
			}
			mu.Lock()
			msgCounts[msg.JobID]++
			mu.Unlock()
			if len(msgCounts) >= 2 {
				return
			}
		}
		consumerErr <- fmt.Errorf("timeout collecting messages for two jobs")
	}()

	time.Sleep(200 * time.Millisecond)

	// Stream only — avoids poll racing the change stream on the same insert.
	_, runCancel, runtime := startDispatchRuntime(t, cfg, db, metadataSvc, dispatchruntime.Options{
		ResumeTokens:   tokens,
		PendingFetcher: emptyPendingFetcher{},
	})

	job1Model, err := metadataSvc.CreateJob(ctx, jobName, map[string]any{"n": 1}, service.CreateJobOptions{Topic: wantTopic})
	if err != nil {
		t.Fatalf("CreateJob job1: %v", err)
	}
	waitForJobStatus(t, metadataSvc, job1Model.JobID, metadata.JobStatusDispatched, 30*time.Second)

	runCancel()
	if err := runtime.Close(); err != nil {
		t.Fatalf("runtime.Close: %v", err)
	}

	if _, err := os.Stat(tokenPath); err != nil {
		t.Fatalf("resume token file missing: %v", err)
	}

	// Second job inserted while the watcher is stopped; restart must still dispatch it.
	job2Model, err := metadataSvc.CreateJob(ctx, jobName, map[string]any{"n": 2}, service.CreateJobOptions{Topic: wantTopic})
	if err != nil {
		t.Fatalf("CreateJob job2: %v", err)
	}

	_, runCancel2, runtime2 := startDispatchRuntime(t, cfg, db, metadataSvc, dispatchruntime.Options{
		ResumeTokens:   tokens,
		PendingFetcher: emptyPendingFetcher{},
	})
	t.Cleanup(func() {
		runCancel2()
		_ = runtime2.Close()
	})

	waitForJobStatus(t, metadataSvc, job2Model.JobID, metadata.JobStatusDispatched, 30*time.Second)

	select {
	case err := <-consumerErr:
		t.Fatalf("consumer: %v", err)
	case <-consumerDone:
	case <-time.After(65 * time.Second):
		t.Fatal("timeout waiting for consumer")
	}

	mu.Lock()
	defer mu.Unlock()
	if msgCounts[job1Model.JobID] != 1 {
		t.Fatalf("job1 publish count=%d want 1 (no duplicate after resume)", msgCounts[job1Model.JobID])
	}
	if msgCounts[job2Model.JobID] != 1 {
		t.Fatalf("job2 publish count=%d want 1", msgCounts[job2Model.JobID])
	}
}
