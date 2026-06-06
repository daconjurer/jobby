package pulsar

import (
	"context"
	"errors"
	"testing"

	"github.com/daconjurer/jobby/internal/jobs/dispatch"
)

type recordingProducer struct {
	topic string
	msg   JobMessage
	err   error
}

func (p *recordingProducer) Publish(_ context.Context, topic string, msg JobMessage) error {
	p.topic = topic
	p.msg = msg
	return p.err
}

func (recordingProducer) Close() error { return nil }

func TestDispatchPublisher_Publish(t *testing.T) {
	ctx := context.Background()
	producer := &recordingProducer{}
	publisher := NewDispatchPublisher(producer)

	job := dispatch.JobDispatchProjection{
		JobID:   "job-1",
		Name:    "load-products",
		Topic:   "persistent://public/default/jobs",
		Payload: map[string]any{"storeId": "s-1"},
	}
	if err := publisher.Publish(ctx, job); err != nil {
		t.Fatal(err)
	}
	if producer.topic != job.Topic {
		t.Fatalf("topic=%q want %q", producer.topic, job.Topic)
	}
	if producer.msg.JobID != job.JobID || producer.msg.Name != job.Name {
		t.Fatalf("msg=%+v", producer.msg)
	}
}

func TestDispatchPublisher_ProducerError(t *testing.T) {
	ctx := context.Background()
	producer := &recordingProducer{err: errors.New("broker down")}
	publisher := NewDispatchPublisher(producer)

	err := publisher.Publish(ctx, dispatch.JobDispatchProjection{
		JobID: "job-1",
		Name:  "load-products",
		Topic: "topic-a",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
