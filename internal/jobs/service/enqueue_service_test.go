package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/pulsar"
)

const (
	testJobAccountLifecycle = "account-lifecycle"
	testTopicAccountsJobs   = "persistent://public/default/accounts/jobs"
)

type fakeTopicResolver struct {
	topic string
	err   error
}

func (f fakeTopicResolver) Resolve(string) (string, error) {
	return f.topic, f.err
}

type fakeJobsWriter struct {
	metadata.JobsWriter
	created metadata.JobMetadata
}

func (w *fakeJobsWriter) Create(_ context.Context, job metadata.JobMetadata) error {
	w.created = job
	return nil
}

func (w *fakeJobsWriter) AddLog(context.Context, metadata.JobLog) error {
	return nil
}

func (w *fakeJobsWriter) MarkDispatchedIfPending(context.Context, string, time.Time) (bool, error) {
	return true, nil
}

func (w *fakeJobsWriter) RecordDispatchAttemptIfPending(context.Context, string, int, string) (bool, error) {
	return true, nil
}

func TestEnqueueService_EnqueueKnownJob(t *testing.T) {
	ctx := context.Background()
	writer := &fakeJobsWriter{}
	reader := &fakeJobsReader{}
	svc := NewMetadataService(reader, writer)
	enqueue := NewEnqueueService(svc, fakeTopicResolver{topic: testTopicAccountsJobs})

	job, err := enqueue.Enqueue(ctx, testJobAccountLifecycle, map[string]any{"k": "v"}, CreateJobOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != metadata.JobStatusPendingDispatch {
		t.Fatalf("status=%s", job.Status)
	}
	if job.Topic != testTopicAccountsJobs {
		t.Fatalf("topic=%q", job.Topic)
	}
	created := writer.created.(*metadata.JobMetadataModel)
	if created.Topic != job.Topic {
		t.Fatalf("persisted topic=%q", created.Topic)
	}
}

func TestEnqueueService_UnknownJobType(t *testing.T) {
	ctx := context.Background()
	svc := NewMetadataService(&fakeJobsReader{}, &fakeJobsWriter{})
	enqueue := NewEnqueueService(svc, fakeTopicResolver{err: pulsar.ErrUnknownJobType})

	_, err := enqueue.Enqueue(ctx, "missing", nil, CreateJobOptions{})
	if !errors.Is(err, pulsar.ErrUnknownJobType) {
		t.Fatalf("err=%v", err)
	}
}

type fakeJobsReader struct{}

func (fakeJobsReader) Get(context.Context, string) (metadata.JobMetadata, error) {
	return nil, metadata.ErrJobNotFound
}
func (fakeJobsReader) List(context.Context, metadata.ListFilter) ([]metadata.JobMetadata, error) {
	return nil, nil
}
func (fakeJobsReader) GetLogs(context.Context, string, metadata.LogFilter) ([]metadata.JobLog, error) {
	return nil, nil
}
func (fakeJobsReader) CountJobs(context.Context, metadata.ListFilter) (int64, error) {
	return 0, nil
}
func (fakeJobsReader) GetJobsByStatus(context.Context, metadata.JobStatus, int) ([]metadata.JobMetadata, error) {
	return nil, nil
}
func (fakeJobsReader) GetPendingJobs(context.Context, int) ([]metadata.JobMetadata, error) {
	return nil, nil
}
func (fakeJobsReader) GetDispatchedJobs(context.Context, int) ([]metadata.JobMetadata, error) {
	return nil, nil
}
func (fakeJobsReader) GetRecentLogs(context.Context, string, int) ([]metadata.JobLog, error) {
	return nil, nil
}
func (fakeJobsReader) GetErrorLogs(context.Context, string) ([]metadata.JobLog, error) {
	return nil, nil
}
