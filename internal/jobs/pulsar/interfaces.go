package pulsar

import "context"

// JobProducer publishes job envelopes to Pulsar topics.
type JobProducer interface {
	Publish(ctx context.Context, topic string, msg JobMessage) error
	Close() error
}

// JobConsumer runs a Pulsar subscription and dispatches decoded JobMessage values.
type JobConsumer interface {
	Run(ctx context.Context, handler func(ctx context.Context, msg JobMessage) error) error
	Close() error
}
