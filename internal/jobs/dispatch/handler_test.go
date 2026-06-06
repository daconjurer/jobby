package dispatch

import (
	"context"
	"errors"
	"testing"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

type mockPublisher struct {
	err error
}

func (p mockPublisher) Publish(context.Context, JobDispatchProjection) error {
	return p.err
}

type trackingUpdater struct {
	dispatched     []string
	dispatchFailed []string
	attempts       map[string]int
}

func (u *trackingUpdater) MarkJobDispatched(_ context.Context, jobID string) error {
	u.dispatched = append(u.dispatched, jobID)
	return nil
}

func (u *trackingUpdater) MarkJobDispatchFailed(_ context.Context, jobID string, _ error) error {
	u.dispatchFailed = append(u.dispatchFailed, jobID)
	return nil
}

func (u *trackingUpdater) RecordDispatchAttempt(_ context.Context, jobID string, attempts int, _ string) error {
	if u.attempts == nil {
		u.attempts = map[string]int{}
	}
	u.attempts[jobID] = attempts
	return nil
}

func TestDispatchHandler_PublishSuccessMarksDispatched(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata(metadata.GenerateJobID(), "load-products", map[string]any{"x": 1})
	job.Topic = "topic-a"
	updater := &trackingUpdater{}
	handler := NewDispatchHandler(DispatchHandlerConfig{MaxAttempts: 3}, mockPublisher{}, updater)

	if err := handler.HandleDispatch(ctx, JobDispatchFromMetadata(job)); err != nil {
		t.Fatal(err)
	}
	if len(updater.dispatched) != 1 || updater.dispatched[0] != job.JobID {
		t.Fatalf("dispatched=%v", updater.dispatched)
	}
}

func TestDispatchHandler_MaxAttemptsMarksDispatchFailed(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata(metadata.GenerateJobID(), "load-products", nil)
	job.Topic = "topic-a"
	job.DispatchAttempts = 2
	updater := &trackingUpdater{}
	handler := NewDispatchHandler(DispatchHandlerConfig{MaxAttempts: 3}, mockPublisher{err: errors.New("broker down")}, updater)

	if err := handler.HandleDispatch(ctx, JobDispatchFromMetadata(job)); err != nil {
		t.Fatal(err)
	}
	if len(updater.dispatchFailed) != 1 {
		t.Fatalf("dispatchFailed=%v", updater.dispatchFailed)
	}
	if updater.attempts[job.JobID] != 3 {
		t.Fatalf("attempts=%d want 3", updater.attempts[job.JobID])
	}
}

func TestDispatchHandler_MissingTopic(t *testing.T) {
	ctx := context.Background()
	job := metadata.NewJobMetadata(metadata.GenerateJobID(), "load-products", nil)
	handler := NewDispatchHandler(DispatchHandlerConfig{MaxAttempts: 3}, mockPublisher{}, &trackingUpdater{})

	err := handler.HandleDispatch(ctx, JobDispatchFromMetadata(job))
	if err == nil {
		t.Fatal("expected error")
	}
}
