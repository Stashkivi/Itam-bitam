package anomaly

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/itam/server/internal/db/postgres"
	"github.com/itam/server/internal/kafka"
	"github.com/itam/server/internal/model"
	"github.com/redis/go-redis/v9"
)

const dedupTTL = 15 * time.Minute

// Engine is the Kafka consumer that applies rules and publishes anomalies.
type Engine struct {
	db       *postgres.DB
	producer *kafka.Producer
	redis    *redis.Client
	rules    []Rule
	consumer *kafka.Consumer
}

// NewEngine wires up the anomaly consumer. It reads from agent.scan.diff and
// agent.scan.full because full scans are also compared to the active baseline.
func NewEngine(
	brokers []string,
	db *postgres.DB,
	producer *kafka.Producer,
	redisClient *redis.Client,
) (*Engine, error) {
	e := &Engine{
		db:      db,
		producer: producer,
		redis:   redisClient,
		rules:   DefaultRules,
	}

	consumer, err := kafka.NewConsumer(
		brokers,
		kafka.GroupAnomaly,
		[]string{kafka.TopicScanFull},
		e.handle,
	)
	if err != nil {
		return nil, err
	}
	e.consumer = consumer
	return e, nil
}

// Run starts the consumer loop. Blocks until ctx is cancelled.
func (e *Engine) Run(ctx context.Context) error {
	defer e.consumer.Close()
	return e.consumer.Run(ctx)
}

func (e *Engine) handle(ctx context.Context, topic string, key, value []byte) error {
	var scan model.ScanPayload
	if err := json.Unmarshal(value, &scan); err != nil {
		return fmt.Errorf("unmarshal scan: %w", err)
	}

	baseline, err := e.db.ActiveBaseline(ctx, string(key))
	if err != nil {
		return fmt.Errorf("fetch baseline: %w", err)
	}
	if baseline == nil {
		return nil // no baseline yet — nothing to compare against
	}

	host, err := e.db.GetHostByUUID(ctx, string(key))
	if err != nil || host == nil {
		return fmt.Errorf("host not found: %s", key)
	}

	for _, rule := range e.rules {
		for _, finding := range rule.Eval(baseline, &scan) {
			if err := e.dispatch(ctx, host, finding); err != nil {
				log.Printf("[anomaly] dispatch error rule=%s: %v", rule.ID, err)
			}
		}
	}
	return nil
}

func (e *Engine) dispatch(ctx context.Context, host *postgres.Host, f Finding) error {
	dedupKey := fmt.Sprintf("anomaly_dedup:%s:%s:%s", host.ID, f.RuleID, f.EntityKey)

	// Suppress duplicate alert dispatch within the dedup window.
	set, err := e.redis.SetNX(ctx, dedupKey, "1", dedupTTL).Result()
	if err != nil {
		return fmt.Errorf("redis setnx: %w", err)
	}

	anomaly := model.Anomaly{
		HostUUID:   host.ID,
		Hostname:   host.Hostname,
		RuleID:     f.RuleID,
		Severity:   f.Severity,
		EntityType: f.EntityType,
		EntityKey:  f.EntityKey,
		Details:    f.Details,
		DetectedAt: time.Now().UTC(),
	}

	// Always write to DB and publish WebSocket event.
	id, err := e.insertAnomaly(ctx, host.ID, f)
	if err != nil {
		return err
	}
	anomaly.ID = id

	// Publish to alert-forwarder only when not suppressed by dedup.
	if set {
		if err := e.producer.Publish(ctx, kafka.TopicAlert, host.ID, anomaly); err != nil {
			log.Printf("[anomaly] publish alert: %v", err)
		}
	}

	// Always broadcast to WebSocket hub via Redis Pub/Sub.
	payload, _ := json.Marshal(anomaly)
	e.redis.Publish(ctx, "anomaly:"+host.OrgID, payload) //nolint:errcheck

	return nil
}

func (e *Engine) insertAnomaly(ctx context.Context, hostID string, f Finding) (int64, error) {
	details, _ := json.Marshal(f.Details)
	dedupKey := fmt.Sprintf("%s:%s:%s", hostID, f.RuleID, f.EntityKey)

	var id int64
	err := e.db.Pool.QueryRow(ctx, `
		INSERT INTO anomalies
		    (host_id, rule_id, severity, entity_type, entity_key, details, dedup_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		hostID, f.RuleID, f.Severity, f.EntityType, f.EntityKey, details, dedupKey,
	).Scan(&id)
	return id, err
}
