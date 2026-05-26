package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Handler processes one Kafka record. Returning an error causes the record to
// be nacked (offset not committed); return nil to commit.
type Handler func(ctx context.Context, topic string, key, value []byte) error

// Consumer is a long-running Kafka consumer group worker.
type Consumer struct {
	client  *kgo.Client
	handler Handler
}

// NewConsumer creates a consumer group client subscribed to the given topics.
func NewConsumer(brokers []string, groupID string, topics []string, handler Handler) (*Consumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.ConsumeTopics(topics...),
		kgo.DisableAutoCommit(), // manual commit after successful handler
		kgo.FetchMaxBytes(10<<20),
	)
	if err != nil {
		return nil, fmt.Errorf("kafka consumer %s: %w", groupID, err)
	}
	return &Consumer{client: client, handler: handler}, nil
}

// Close gracefully stops the consumer.
func (c *Consumer) Close() { c.client.Close() }

// Run polls Kafka and dispatches records to the handler until ctx is cancelled.
// It commits offsets only after the handler returns nil for all records in a fetch.
func (c *Consumer) Run(ctx context.Context) error {
	for {
		fetches := c.client.PollFetches(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, e := range errs {
				log.Printf("[kafka] fetch error topic=%s partition=%d: %v",
					e.Topic, e.Partition, e.Err)
			}
			continue
		}

		var commitErr bool
		fetches.EachRecord(func(r *kgo.Record) {
			if err := c.handler(ctx, r.Topic, r.Key, r.Value); err != nil {
				log.Printf("[kafka] handler error topic=%s key=%s: %v", r.Topic, r.Key, err)
				commitErr = true
			}
		})

		if !commitErr {
			if err := c.client.CommitUncommittedOffsets(ctx); err != nil {
				log.Printf("[kafka] commit error: %v", err)
			}
		}
	}
}

// UnmarshalRecord is a convenience helper used by consumer handlers.
func UnmarshalRecord(value []byte, dst interface{}) error {
	return json.Unmarshal(value, dst)
}
