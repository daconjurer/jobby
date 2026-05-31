package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
)

const defaultBatchSize = 500

var defaultMaxAge = 30 * 24 * time.Hour

// Run generates and inserts jobs (and optional logs) in batches.
func Run(ctx context.Context, writer *metadata.MongoJobsWriter, opts Options) (Result, error) {
	if writer == nil {
		return Result{}, fmt.Errorf("mongo writer is required")
	}
	if opts.Count < 1 {
		return Result{}, fmt.Errorf("count must be at least 1")
	}
	if opts.BatchSize < 1 {
		opts.BatchSize = defaultBatchSize
	}
	if opts.MaxAge <= 0 {
		opts.MaxAge = defaultMaxAge
	}
	if opts.WithLogs && opts.LogsPerJobMin < 0 {
		return Result{}, fmt.Errorf("logs-per-job-min cannot be negative")
	}
	if opts.WithLogs && opts.LogsPerJobMax < opts.LogsPerJobMin {
		return Result{}, fmt.Errorf("logs-per-job-max must be >= logs-per-job-min")
	}

	f := newFaker(opts.Seed)

	var result Result
	jobsBatch := make([]*metadata.JobMetadataModel, 0, opts.BatchSize)
	logsBatch := make([]metadata.JobLog, 0, opts.BatchSize)

	flushJobs := func() error {
		if len(jobsBatch) == 0 {
			return nil
		}
		n, err := writer.InsertJobs(ctx, jobsBatch)
		if err != nil {
			return err
		}
		result.JobsInserted += n
		jobsBatch = jobsBatch[:0]
		return nil
	}

	flushLogs := func() error {
		if len(logsBatch) == 0 {
			return nil
		}
		n, err := writer.InsertLogs(ctx, logsBatch)
		if err != nil {
			return err
		}
		result.LogsInserted += n
		logsBatch = logsBatch[:0]
		return nil
	}

	for i := 0; i < opts.Count; i++ {
		job, err := BuildJob(f, opts.MaxAge)
		if err != nil {
			return result, fmt.Errorf("build job %d: %w", i+1, err)
		}
		jobsBatch = append(jobsBatch, job)

		if opts.WithLogs {
			logs := BuildLogs(job, opts.LogsPerJobMin, opts.LogsPerJobMax, f)
			logsBatch = append(logsBatch, logs...)
		}

		if len(jobsBatch) >= opts.BatchSize {
			if err := flushJobs(); err != nil {
				return result, fmt.Errorf("insert jobs: %w", err)
			}
		}
		if len(logsBatch) >= opts.BatchSize {
			if err := flushLogs(); err != nil {
				return result, fmt.Errorf("insert logs: %w", err)
			}
		}
	}

	if err := flushJobs(); err != nil {
		return result, fmt.Errorf("insert jobs: %w", err)
	}
	if err := flushLogs(); err != nil {
		return result, fmt.Errorf("insert logs: %w", err)
	}

	return result, nil
}
