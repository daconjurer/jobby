package pulsar

import "context"

// JobProducer publishes job envelopes to Pulsar topics.
// Implemented by PulsarJobProducer in a later phase.
type JobProducer interface {
	Publish(ctx context.Context, topic string, msg JobMessage) error
	Close() error
}

// JobMessageConsumer runs a Pulsar subscription and dispatches decoded JobMessage values.
// Implemented by PulsarJobMessageConsumer in a later phase.
type JobMessageConsumer interface {
	Run(ctx context.Context, handler func(ctx context.Context, msg JobMessage) error) error
	Close() error
}
