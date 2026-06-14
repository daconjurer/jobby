package dispatch

import (
	"context"
	"log"
	"time"
)

// WorkerConfig controls poll fallback and shared handler settings.
type WorkerConfig struct {
	PollInterval time.Duration
	BatchSize    int
	MaxAttempts  int
}

// DispatchWorker orchestrates job dispatch (Pulsar publish) using two complementary triggers.
//
// Primary — change stream (StreamRunner): started in a background goroutine and reacts
// to new job_metadata inserts with status pending_dispatch. This path provides low
// latency without waiting for the poll interval.
//
// Secondary — poll fallback (PendingJobFetcher): runs on the main goroutine on a fixed
// PollInterval (and once immediately at startup). It queries jobs still in
// pending_dispatch with dispatchAttempts below MaxAttempts and re-invokes the same
// DispatchHandler. This covers gaps the stream can miss: cursor errors before resume tokens
// apply, invalidate events, worker restarts, and insert events lost while the
// watcher was down.
//
// Both triggers share one DispatchHandler; duplicate deliveries are safe because publish and
// status updates are idempotent per job ID.
type DispatchWorker struct {
	cfg     WorkerConfig
	handler *DispatchHandler
	pending PendingJobFetcher
	stream  StreamRunner
}

// NewDispatchWorker wires the dual-trigger dispatch worker.
func NewDispatchWorker(cfg WorkerConfig, handler *DispatchHandler, pending PendingJobFetcher, stream StreamRunner) *DispatchWorker {
	return &DispatchWorker{
		cfg:     cfg,
		handler: handler,
		pending: pending,
		stream:  stream,
	}
}

// Run starts the change stream goroutine and poll loop until ctx is cancelled.
func (w *DispatchWorker) Run(ctx context.Context) {
	go w.stream.Run(ctx)
	w.runPoll(ctx)
}

func (w *DispatchWorker) runPoll(ctx context.Context) {
	w.pollOnce(ctx)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.pollOnce(ctx)
		}
	}
}

func (w *DispatchWorker) pollOnce(ctx context.Context) {
	jobs, err := w.pending.FetchPending(ctx, w.cfg.MaxAttempts, w.cfg.BatchSize)
	if err != nil {
		log.Printf("dispatch poll error: %v", err)
		return
	}
	for _, job := range jobs {
		if err := w.handler.HandleDispatch(ctx, job); err != nil {
			log.Printf("dispatch poll job %s: %v", job.JobID, err)
		}
	}
}
