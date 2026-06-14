package pulsar

import (
	"context"
	"fmt"
	"sync"

	"github.com/apache/pulsar-client-go/pulsar"
)

// PulsarJobProducer publishes JobMessage envelopes using a shared PulsarClient.
type PulsarJobProducer struct {
	client    *PulsarClient
	mu        sync.Mutex
	producers map[string]pulsar.Producer
}

// NewPulsarJobProducer creates a producer adapter backed by client.
func NewPulsarJobProducer(client *PulsarClient) *PulsarJobProducer {
	return &PulsarJobProducer{
		client:    client,
		producers: make(map[string]pulsar.Producer),
	}
}

// Publish sends msg to topic with jobId as the Pulsar message key for idempotency.
func (p *PulsarJobProducer) Publish(ctx context.Context, topic string, msg JobMessage) error {
	producer, err := p.producerForTopic(topic)
	if err != nil {
		return err
	}
	payload, err := Marshal(msg)
	if err != nil {
		return err
	}
	_, err = producer.Send(ctx, &pulsar.ProducerMessage{
		Key:     msg.JobID,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("publish to %q: %w", topic, err)
	}
	return nil
}

// Close releases the producer cache; PulsarClient.Close shuts down underlying producers.
func (p *PulsarJobProducer) Close() error {
	p.mu.Lock()
	p.producers = make(map[string]pulsar.Producer)
	p.mu.Unlock()
	return nil
}

func (p *PulsarJobProducer) producerForTopic(topic string) (pulsar.Producer, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if producer, ok := p.producers[topic]; ok {
		return producer, nil
	}
	producer, err := p.client.CreateProducer(topic)
	if err != nil {
		return nil, err
	}
	p.producers[topic] = producer
	return producer, nil
}
