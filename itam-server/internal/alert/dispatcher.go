package alert

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/itam/server/internal/db/postgres"
	"github.com/itam/server/internal/kafka"
	"github.com/itam/server/internal/model"
)

// ChannelConfig holds the decrypted configuration for one alert channel.
// The raw values are decrypted by the service at startup from alert_channels.config_enc.
type ChannelConfig struct {
	OrgID       string
	ChannelType string // slack | telegram
	WebhookURL  string // slack
	BotToken    string // telegram
	ChatID      int64  // telegram
}

// Dispatcher is the Kafka consumer that routes anomaly events to external channels.
type Dispatcher struct {
	db       *postgres.DB
	channels map[string][]ChannelConfig // keyed by org_id
	consumer *kafka.Consumer
}

// NewDispatcher builds the alert-forwarder consumer. channels is pre-loaded
// and decrypted at startup; hot-reload is a future enhancement.
func NewDispatcher(brokers []string, db *postgres.DB, channels []ChannelConfig) (*Dispatcher, error) {
	d := &Dispatcher{
		db:       db,
		channels: make(map[string][]ChannelConfig),
	}
	for _, ch := range channels {
		d.channels[ch.OrgID] = append(d.channels[ch.OrgID], ch)
	}

	consumer, err := kafka.NewConsumer(
		brokers,
		kafka.GroupAlerter,
		[]string{kafka.TopicAlert},
		d.handle,
	)
	if err != nil {
		return nil, err
	}
	d.consumer = consumer
	return d, nil
}

// Run blocks until ctx is cancelled.
func (d *Dispatcher) Run(ctx context.Context) error {
	defer d.consumer.Close()
	return d.consumer.Run(ctx)
}

func (d *Dispatcher) handle(ctx context.Context, _ string, key, value []byte) error {
	var anomaly model.Anomaly
	if err := json.Unmarshal(value, &anomaly); err != nil {
		return fmt.Errorf("unmarshal anomaly: %w", err)
	}

	host, err := d.db.GetHostByUUID(ctx, string(key))
	if err != nil || host == nil {
		return nil
	}

	channels := d.channels[host.OrgID]
	for _, ch := range channels {
		if err := d.send(ctx, ch, &anomaly); err != nil {
			log.Printf("[alerter] channel=%s org=%s err=%v", ch.ChannelType, ch.OrgID, err)
			d.recordFailure(ctx, anomaly.ID, ch.ChannelType, err)
		} else {
			d.recordSuccess(ctx, anomaly.ID, ch.ChannelType)
		}
	}
	return nil
}

func (d *Dispatcher) send(ctx context.Context, ch ChannelConfig, anomaly *model.Anomaly) error {
	switch ch.ChannelType {
	case "slack":
		return SendSlack(ctx, ch.WebhookURL, anomaly)
	case "telegram":
		return SendTelegram(ctx, ch.BotToken, ch.ChatID, anomaly)
	default:
		return fmt.Errorf("unknown channel type: %s", ch.ChannelType)
	}
}

func (d *Dispatcher) recordSuccess(ctx context.Context, anomalyID int64, channel string) {
	d.db.Pool.Exec(ctx, `
		INSERT INTO alert_deliveries (anomaly_id, channel, delivered_at, attempts)
		VALUES ($1, $2, $3, 1)`,
		anomalyID, channel, time.Now(),
	) //nolint:errcheck
}

func (d *Dispatcher) recordFailure(ctx context.Context, anomalyID int64, channel string, err error) {
	d.db.Pool.Exec(ctx, `
		INSERT INTO alert_deliveries (anomaly_id, channel, failed_at, attempts, last_error)
		VALUES ($1, $2, $3, 1, $4)`,
		anomalyID, channel, time.Now(), err.Error(),
	) //nolint:errcheck
}
