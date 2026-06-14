package dispatch

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakePendingFetcher struct {
	jobs        []JobDispatchProjection
	err         error
	maxAttempts int
	limit       int
}

func (f *fakePendingFetcher) FetchPending(_ context.Context, maxAttempts, limit int) ([]JobDispatchProjection, error) {
	f.maxAttempts = maxAttempts
	f.limit = limit
	if f.err != nil {
		return nil, f.err
	}
	return f.jobs, nil
}

type recordingStreamRunner struct {
	runCh chan struct{}
}

func (s *recordingStreamRunner) Run(ctx context.Context) {
	if s.runCh != nil {
		close(s.runCh)
	}
	<-ctx.Done()
}

func TestDispatchWorker_PollOnceDispatchesPendingJobs(t *testing.T) {
	ctx := context.Background()
	jobA := JobDispatchProjection{JobID: "job-a", Name: "load-products", Topic: "topic-a"}
	jobB := JobDispatchProjection{JobID: "job-b", Name: "load-products", Topic: "topic-b"}
	pending := &fakePendingFetcher{jobs: []JobDispatchProjection{jobA, jobB}}
	updater := &trackingUpdater{}
	handler := NewDispatchHandler(DispatchHandlerConfig{MaxAttempts: 3}, mockPublisher{}, updater)
	worker := NewDispatchWorker(
		WorkerConfig{MaxAttempts: 3, BatchSize: 10},
		handler,
		pending,
		&recordingStreamRunner{},
	)

	worker.pollOnce(ctx)

	if pending.maxAttempts != 3 || pending.limit != 10 {
		t.Fatalf("fetch args maxAttempts=%d limit=%d", pending.maxAttempts, pending.limit)
	}
	if len(updater.dispatched) != 2 {
		t.Fatalf("dispatched=%v", updater.dispatched)
	}
	if updater.dispatched[0] != jobA.JobID || updater.dispatched[1] != jobB.JobID {
		t.Fatalf("dispatched=%v", updater.dispatched)
	}
}

func TestDispatchWorker_PollOnceFetchErrorDoesNotDispatch(t *testing.T) {
	ctx := context.Background()
	updater := &trackingUpdater{}
	handler := NewDispatchHandler(DispatchHandlerConfig{MaxAttempts: 3}, mockPublisher{}, updater)
	worker := NewDispatchWorker(
		WorkerConfig{MaxAttempts: 3, BatchSize: 10},
		handler,
		&fakePendingFetcher{err: errors.New("mongo down")},
		&recordingStreamRunner{},
	)

	worker.pollOnce(ctx)

	if len(updater.dispatched) != 0 {
		t.Fatalf("dispatched=%v want none", updater.dispatched)
	}
}

func TestDispatchWorker_PollOnceHandlerErrorContinuesRemainingJobs(t *testing.T) {
	ctx := context.Background()
	jobA := JobDispatchProjection{JobID: "job-a", Name: "load-products", Topic: "topic-a"}
	jobB := JobDispatchProjection{JobID: "job-b", Name: "load-products", Topic: "topic-b"}
	updater := &trackingUpdater{}
	handler := NewDispatchHandler(
		DispatchHandlerConfig{MaxAttempts: 3},
		failingPublisher{failJobID: jobA.JobID},
		updater,
	)
	worker := NewDispatchWorker(
		WorkerConfig{MaxAttempts: 3, BatchSize: 10},
		handler,
		&fakePendingFetcher{jobs: []JobDispatchProjection{jobA, jobB}},
		&recordingStreamRunner{},
	)

	worker.pollOnce(ctx)

	if len(updater.dispatched) != 1 || updater.dispatched[0] != jobB.JobID {
		t.Fatalf("dispatched=%v want [job-b]", updater.dispatched)
	}
}

func TestDispatchWorker_RunPollsImmediatelyBeforeTicker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dispatched := make(chan struct{}, 1)
	updater := &callbackUpdater{
		onDispatched: func(jobID string) {
			if jobID == "job-immediate" {
				dispatched <- struct{}{}
			}
		},
	}
	job := JobDispatchProjection{JobID: "job-immediate", Name: "load-products", Topic: "topic-a"}
	handler := NewDispatchHandler(DispatchHandlerConfig{MaxAttempts: 3}, mockPublisher{}, updater)
	worker := NewDispatchWorker(
		WorkerConfig{PollInterval: time.Hour, MaxAttempts: 3, BatchSize: 10},
		handler,
		&fakePendingFetcher{jobs: []JobDispatchProjection{job}},
		&recordingStreamRunner{},
	)

	go worker.runPoll(ctx)

	select {
	case <-dispatched:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for immediate poll dispatch")
	}
}

func TestDispatchWorker_RunPollRepeatsOnTicker(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fetchCount := 0
	pending := &countingPendingFetcher{
		onFetch: func() {
			fetchCount++
			if fetchCount >= 2 {
				cancel()
			}
		},
	}
	handler := NewDispatchHandler(DispatchHandlerConfig{MaxAttempts: 3}, mockPublisher{}, &trackingUpdater{})
	worker := NewDispatchWorker(
		WorkerConfig{PollInterval: 20 * time.Millisecond, MaxAttempts: 3, BatchSize: 10},
		handler,
		pending,
		&recordingStreamRunner{},
	)

	worker.runPoll(ctx)

	if fetchCount < 2 {
		t.Fatalf("fetchCount=%d want at least 2", fetchCount)
	}
}

func TestDispatchWorker_RunStartsStreamRunner(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := &recordingStreamRunner{runCh: make(chan struct{})}
	handler := NewDispatchHandler(DispatchHandlerConfig{MaxAttempts: 3}, mockPublisher{}, &trackingUpdater{})
	worker := NewDispatchWorker(
		WorkerConfig{PollInterval: time.Hour, MaxAttempts: 3, BatchSize: 10},
		handler,
		&fakePendingFetcher{},
		stream,
	)

	go worker.Run(ctx)

	select {
	case <-stream.runCh:
	case <-time.After(time.Second):
		t.Fatal("stream runner was not started")
	}
}

type failingPublisher struct {
	failJobID string
}

func (p failingPublisher) Publish(_ context.Context, job JobDispatchProjection) error {
	if job.JobID == p.failJobID {
		return errors.New("publish failed")
	}
	return nil
}

type callbackUpdater struct {
	onDispatched func(jobID string)
}

func (u *callbackUpdater) MarkJobDispatched(_ context.Context, jobID string) error {
	if u.onDispatched != nil {
		u.onDispatched(jobID)
	}
	return nil
}

func (callbackUpdater) MarkJobDispatchFailed(context.Context, string, error) error { return nil }
func (callbackUpdater) RecordDispatchAttempt(context.Context, string, int, string) error {
	return nil
}

type countingPendingFetcher struct {
	onFetch func()
}

func (f *countingPendingFetcher) FetchPending(context.Context, int, int) ([]JobDispatchProjection, error) {
	if f.onFetch != nil {
		f.onFetch()
	}
	return nil, nil
}
