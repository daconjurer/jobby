package pulsar

import (
	"fmt"
	"sync"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/daconjurer/jobby/internal/config"
)

// PulsarClient wraps the official Pulsar client with tracked producers and consumers for graceful shutdown.
type PulsarClient struct {
	cfg       config.PulsarConfig
	client    pulsar.Client
	mu        sync.Mutex
	producers []pulsar.Producer
	consumers []pulsar.Consumer
}

// NewPulsarClient connects to the broker using cfg.ServiceURL.
func NewPulsarClient(cfg config.PulsarConfig) (*PulsarClient, error) {
	client, err := pulsar.NewClient(pulsar.ClientOptions{
		URL: cfg.ServiceURL,
	})
	if err != nil {
		return nil, fmt.Errorf("create pulsar client: %w", err)
	}
	return &PulsarClient{
		cfg:    cfg,
		client: client,
	}, nil
}

// CreateProducer creates a producer for topic with backpressure when the send queue is full.
func (c *PulsarClient) CreateProducer(topic string) (pulsar.Producer, error) {
	producer, err := c.client.CreateProducer(pulsar.ProducerOptions{
		Topic:                   topic,
		DisableBlockIfQueueFull: false,
	})
	if err != nil {
		return nil, fmt.Errorf("create producer for topic %q: %w", topic, err)
	}
	c.mu.Lock()
	c.producers = append(c.producers, producer)
	c.mu.Unlock()
	return producer, nil
}

// CreateConsumer subscribes to topic with a Shared subscription and delivers messages to messageHandler.
// Phase 3 will use the same listener style for the executor receive loop.
func (c *PulsarClient) CreateConsumer(
	topic string,
	messageHandler func(pulsar.Consumer, pulsar.Message),
) (pulsar.Consumer, error) {
	ch := make(chan pulsar.ConsumerMessage)
	consumer, err := c.client.Subscribe(pulsar.ConsumerOptions{
		Topic:            topic,
		SubscriptionName: c.cfg.SubscriptionName,
		Type:             pulsar.Shared,
		MessageChannel:   ch,
	})
	if err != nil {
		close(ch)
		return nil, fmt.Errorf("create consumer for topic %q: %w", topic, err)
	}

	go func() {
		for cm := range ch {
			messageHandler(cm.Consumer, cm.Message)
		}
	}()

	c.mu.Lock()
	c.consumers = append(c.consumers, consumer)
	c.mu.Unlock()
	return consumer, nil
}

// Close closes all consumers, producers, then the underlying client.
func (c *PulsarClient) Close() error {
	c.mu.Lock()
	consumers := append([]pulsar.Consumer(nil), c.consumers...)
	producers := append([]pulsar.Producer(nil), c.producers...)
	c.consumers = nil
	c.producers = nil
	client := c.client
	c.client = nil
	c.mu.Unlock()

	for _, consumer := range consumers {
		consumer.Close()
	}
	for _, producer := range producers {
		producer.Close()
	}
	if client != nil {
		client.Close()
	}
	return nil
}
