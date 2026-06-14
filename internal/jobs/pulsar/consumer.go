package pulsar

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/apache/pulsar-client-go/pulsar"
)

// PulsarJobConsumer consumes job messages from Pulsar with manual Receive() loops.
// It extracts fields from JobMessage and calls handler with flat args (decoupled from pulsar types).
type PulsarJobConsumer struct {
	client    *PulsarClient
	consumers []pulsar.Consumer
	topics    []string
}

// NewPulsarJobConsumer creates a consumer that subscribes to all topics upfront.
// Each topic gets a Shared subscription for load balancing across executor instances.
func NewPulsarJobConsumer(client *PulsarClient, topics []string) (*PulsarJobConsumer, error) {
	if len(topics) == 0 {
		return nil, fmt.Errorf("at least one topic required")
	}

	consumers := make([]pulsar.Consumer, 0, len(topics))
	for _, topic := range topics {
		consumer, err := client.client.Subscribe(pulsar.ConsumerOptions{
			Topic:            topic,
			SubscriptionName: client.cfg.SubscriptionName,
			Type:             pulsar.Shared,
		})
		if err != nil {
			// Close any consumers we've already created
			for _, c := range consumers {
				c.Close()
			}
			return nil, fmt.Errorf("subscribe to topic %q: %w", topic, err)
		}
		consumers = append(consumers, consumer)

		// Track in client for shutdown
		client.mu.Lock()
		client.consumers = append(client.consumers, consumer)
		client.mu.Unlock()
	}

	return &PulsarJobConsumer{
		client:    client,
		consumers: consumers,
		topics:    topics,
	}, nil
}

// Run starts goroutines that read from each consumer until ctx is cancelled.
// Handler is called with flat args (jobID, name, payload) extracted from JobMessage.
// Handler return value determines ack/nack: nil = ack, error = nack.
func (c *PulsarJobConsumer) Run(ctx context.Context, handler func(ctx context.Context, jobID, name string, payload json.RawMessage) error) error {
	if handler == nil {
		return fmt.Errorf("handler is required")
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(c.consumers))

	// Start a goroutine for each consumer
	for i, consumer := range c.consumers {
		topic := c.topics[i]
		wg.Add(1)

		go func(consumer pulsar.Consumer, topic string) {
			defer wg.Done()
			c.consumeLoop(ctx, consumer, topic, handler, errChan)
		}(consumer, topic)
	}

	// Wait for context cancellation
	<-ctx.Done()
	log.Printf("info: shutting down job consumer (context cancelled)")

	// Wait for all goroutines to finish
	wg.Wait()
	close(errChan)

	// Collect any errors (though we typically don't fail on individual message errors)
	for err := range errChan {
		if err != nil {
			log.Printf("error: consumer loop error: %v", err)
		}
	}

	return ctx.Err()
}

// consumeLoop continuously receives messages from a single consumer until ctx is cancelled.
func (c *PulsarJobConsumer) consumeLoop(
	ctx context.Context,
	consumer pulsar.Consumer,
	topic string,
	handler func(ctx context.Context, jobID, name string, payload json.RawMessage) error,
	errChan chan<- error,
) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Receive with context (blocks until message or ctx cancelled)
		msg, err := consumer.Receive(ctx)
		if err != nil {
			// Context cancelled is expected during shutdown
			if ctx.Err() != nil {
				return
			}
			log.Printf("error: failed to receive message from topic %s: %v", topic, err)
			continue
		}

		// Process the message
		c.handleMessage(ctx, consumer, msg, handler)
	}
}

// handleMessage unmarshals JobMessage, calls handler, and acks/nacks based on result.
func (c *PulsarJobConsumer) handleMessage(
	ctx context.Context,
	consumer pulsar.Consumer,
	msg pulsar.Message,
	handler func(ctx context.Context, jobID, name string, payload json.RawMessage) error,
) {
	// Unmarshal the envelope
	var jobMsg JobMessage
	if err := json.Unmarshal(msg.Payload(), &jobMsg); err != nil {
		log.Printf("error: failed to unmarshal JobMessage from topic %s (msgID=%s): %v - acking to skip",
			msg.Topic(), msg.ID(), err)
		if ackErr := consumer.Ack(msg); ackErr != nil {
			log.Printf("error: failed to ack message (msgID=%s): %v", msg.ID(), ackErr)
		}
		return
	}

	// Validate envelope fields
	if err := jobMsg.validate(); err != nil {
		log.Printf("error: invalid JobMessage from topic %s (msgID=%s): %v - acking to skip",
			msg.Topic(), msg.ID(), err)
		if ackErr := consumer.Ack(msg); ackErr != nil {
			log.Printf("error: failed to ack message (msgID=%s): %v", msg.ID(), ackErr)
		}
		return
	}

	// Call handler with flat args
	handlerErr := handler(ctx, jobMsg.JobID, jobMsg.Name, jobMsg.Payload)

	// Ack/nack based on handler return
	if handlerErr != nil {
		log.Printf("error: handler returned error for job %s: %v - nacking for retry", jobMsg.JobID, handlerErr)
		consumer.Nack(msg)
	} else {
		if err := consumer.Ack(msg); err != nil {
			log.Printf("error: failed to ack message for job %s (msgID=%s): %v", jobMsg.JobID, msg.ID(), err)
		}
	}
}

// Close closes all consumers.
func (c *PulsarJobConsumer) Close() error {
	for _, consumer := range c.consumers {
		consumer.Close()
	}
	return nil
}
