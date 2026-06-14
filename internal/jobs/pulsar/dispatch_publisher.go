package pulsar

import (
	"context"
	"fmt"

	"github.com/daconjurer/jobby/internal/jobs/dispatch"
)

// PulsarJobPublisher adapts JobProducer to dispatch.JobDispatchPublisher.
type PulsarJobPublisher struct {
	producer JobProducer
}

var _ dispatch.JobDispatchPublisher = (*PulsarJobPublisher)(nil)

// NewDispatchPublisher returns a saga-phase-2 publisher backed by Pulsar.
func NewDispatchPublisher(producer JobProducer) *PulsarJobPublisher {
	return &PulsarJobPublisher{producer: producer}
}

// Publish builds a JobMessage envelope and sends it to the job's topic.
func (p *PulsarJobPublisher) Publish(ctx context.Context, job dispatch.JobDispatchProjection) error {
	msg, err := NewJobMessage(job.JobID, job.Name, job.Payload)
	if err != nil {
		return fmt.Errorf("build job message: %w", err)
	}
	return p.producer.Publish(ctx, job.Topic, msg)
}
