//go:build integration

package pulsar

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/apache/pulsar-client-go/pulsar"
	"github.com/daconjurer/jobby/internal/config"
	"github.com/daconjurer/jobby/internal/jobs/metadata"
	"github.com/daconjurer/jobby/internal/testutil"
)

// Integration tests require a live Pulsar broker (for example: docker compose up -d pulsar).
// Run: task test-integration
//
// Required: PULSAR_SERVICE_URL.

func integrationPulsarConfig(tb testing.TB) config.PulsarConfig {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test (-short)")
	}
	url := os.Getenv("PULSAR_SERVICE_URL")
	if url == "" {
		tb.Skip("PULSAR_SERVICE_URL is not set (required for Pulsar integration tests)")
	}
	return config.PulsarConfig{
		ServiceURL:       url,
		SubscriptionName: "integration-test",
	}
}

func integrationTestTopic(tb testing.TB) string {
	tb.Helper()
	resolver, err := NewFileTopicResolver(testutil.JobTopicsConfigPath(tb))
	if err != nil {
		tb.Fatalf("NewFileTopicResolver: %v", err)
	}
	topic, err := resolver.Resolve("account-lifecycle")
	if err != nil {
		tb.Fatalf("Resolve(account-lifecycle): %v", err)
	}
	return topic
}

func consumePublishedMessage(serviceURL, topic, subscription string, timeout time.Duration) (JobMessage, error) {
	client, err := pulsar.NewClient(pulsar.ClientOptions{URL: serviceURL})
	if err != nil {
		return JobMessage{}, fmt.Errorf("pulsar.NewClient: %w", err)
	}
	defer client.Close()

	consumer, err := client.Subscribe(pulsar.ConsumerOptions{
		Topic:                       topic,
		SubscriptionName:            subscription,
		Type:                        pulsar.Shared,
		SubscriptionInitialPosition: pulsar.SubscriptionPositionLatest,
	})
	if err != nil {
		return JobMessage{}, fmt.Errorf("Subscribe: %w", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	raw, err := consumer.Receive(ctx)
	if err != nil {
		return JobMessage{}, fmt.Errorf("Receive: %w", err)
	}
	if err := consumer.Ack(raw); err != nil {
		return JobMessage{}, fmt.Errorf("Ack: %w", err)
	}

	decoded, err := Unmarshal(raw.Payload())
	if err != nil {
		return JobMessage{}, fmt.Errorf("Unmarshal: %w", err)
	}
	if raw.Key() != decoded.JobID {
		return JobMessage{}, fmt.Errorf("message key=%q want jobId=%q", raw.Key(), decoded.JobID)
	}
	return decoded, nil
}

func TestIntegration_PulsarJobProducer_PublishAndConsume(t *testing.T) {
	cfg := integrationPulsarConfig(t)
	topic := integrationTestTopic(t)

	jobID := metadata.GenerateJobID()
	subscription := fmt.Sprintf("integration-producer-%s", jobID)

	// Subscribe before publish so SubscriptionPositionLatest does not miss the message.
	recvDone := make(chan JobMessage, 1)
	recvErr := make(chan error, 1)
	go func() {
		msg, err := consumePublishedMessage(cfg.ServiceURL, topic, subscription, 30*time.Second)
		if err != nil {
			recvErr <- err
			return
		}
		recvDone <- msg
	}()

	time.Sleep(200 * time.Millisecond)

	pulsarClient, err := NewPulsarClient(cfg)
	if err != nil {
		t.Fatalf("NewPulsarClient: %v", err)
	}
	t.Cleanup(func() { _ = pulsarClient.Close() })

	producer := NewPulsarJobProducer(pulsarClient)
	t.Cleanup(func() { _ = producer.Close() })

	payload := map[string]any{"integration": true, "n": 42}
	msg, err := NewJobMessage(jobID, "account-lifecycle", payload)
	if err != nil {
		t.Fatal(err)
	}
	if err := producer.Publish(context.Background(), topic, msg); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case err := <-recvErr:
		t.Fatalf("consume: %v", err)
	case got := <-recvDone:
		if got.JobID != jobID || got.Name != "account-lifecycle" {
			t.Fatalf("message=%+v", got)
		}
		decoded, err := DecodePayload[map[string]any](got)
		if err != nil {
			t.Fatal(err)
		}
		if decoded["integration"] != true || decoded["n"] != float64(42) {
			t.Fatalf("payload=%v", decoded)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("timeout waiting for Pulsar message")
	}
}
