package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Producer is a thin wrapper around the franz-go synchronous producer.
// Partitioning by host_uuid ensures all events for one host land on the
// same partition, preserving ordering within a host.
type Producer struct {
	client *kgo.Client
}

// NewProducer creates a Kafka producer connected to brokers.
func NewProducer(brokers []string) (*Producer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.RequiredAcks(kgo.LeaderAck()), // acks=1 for throughput
		kgo.ProducerBatchCompression(kgo.Lz4Compression()),
		kgo.ProducerBatchMaxBytes(1<<20), // 1 MB max batch
	)
	if err != nil {
		return nil, fmt.Errorf("kafka producer: %w", err)
	}
	return &Producer{client: client}, nil
}

// Close flushes pending records and closes the connection.
func (p *Producer) Close() { p.client.Close() }

// Publish serialises payload as JSON and sends it to topic, keyed by hostUUID.
func (p *Producer) Publish(ctx context.Context, topic, hostUUID string, payload interface{}) error {
	value, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(hostUUID), // partition key — keeps one host on one partition
		Value: value,
	}

	results := p.client.ProduceSync(ctx, record)
	return results.FirstErr()
}
