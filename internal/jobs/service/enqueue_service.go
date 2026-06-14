package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/jobs/pulsar"
)

// EnqueueService accepts jobs and persists metadata with embedded dispatch intent (no inline publish).
type EnqueueService struct {
	metadata *MetadataService
	topics   pulsar.TopicResolver
}

// NewEnqueueService wires create with topic resolution.
func NewEnqueueService(metadata *MetadataService, topics pulsar.TopicResolver) *EnqueueService {
	return &EnqueueService{
		metadata: metadata,
		topics:   topics,
	}
}

// Enqueue validates the job name and creates metadata with topic in a single write.
func (s *EnqueueService) Enqueue(ctx context.Context, name string, payload map[string]any, opts CreateJobOptions) (*metadata.JobMetadataModel, error) {
	topic, err := s.topics.Resolve(name)
	if err != nil {
		if errors.Is(err, pulsar.ErrUnknownJobType) {
			return nil, err
		}
		return nil, fmt.Errorf("resolve topic: %w", err)
	}

	opts.Topic = topic
	return s.metadata.CreateJob(ctx, name, payload, opts)
}
