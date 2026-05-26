// Package workers contains the Kafka consumer workers that run as background
// goroutines in the server process.
package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/itam/server/internal/db/postgres"
	"github.com/itam/server/internal/kafka"
	"github.com/itam/server/internal/model"
)

// PGWriter consumes agent.scan.full and persists snapshots + diffs to PostgreSQL.
type PGWriter struct {
	db       *postgres.DB
	consumer *kafka.Consumer
}

func NewPGWriter(brokers []string, db *postgres.DB) (*PGWriter, error) {
	w := &PGWriter{db: db}
	consumer, err := kafka.NewConsumer(
		brokers,
		kafka.GroupPGWriter,
		[]string{kafka.TopicScanFull, kafka.TopicScanDiff},
		w.handle,
	)
	if err != nil {
		return nil, err
	}
	w.consumer = consumer
	return w, nil
}

func (w *PGWriter) Run(ctx context.Context) error {
	defer w.consumer.Close()
	return w.consumer.Run(ctx)
}

func (w *PGWriter) handle(ctx context.Context, _ string, key, value []byte) error {
	var scan model.ScanPayload
	if err := json.Unmarshal(value, &scan); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	// 1. Fetch the previous snapshot for diffing.
	prevRaw, _, err := w.db.LatestSnapshot(ctx, string(key))
	if err != nil {
		return fmt.Errorf("latest snapshot: %w", err)
	}

	// 2. Insert the new snapshot.
	snapshotID, err := w.db.InsertSnapshot(ctx, string(key), scan.CapturedAt, scan)
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}

	// 3. Compute and store diffs.
	if prevRaw != nil {
		var prev model.ScanPayload
		if err := json.Unmarshal(prevRaw, &prev); err == nil {
			if _, err := w.db.ComputeAndStoreDiffs(ctx, string(key), &prev, &scan); err != nil {
				log.Printf("[pg-writer] diff error host=%s: %v", key, err)
			}
		}
	}

	// 4. Auto-approve the baseline on first scan.
	if err := w.db.AutoApproveFirstScan(ctx, string(key), scan, snapshotID); err != nil {
		log.Printf("[pg-writer] auto baseline host=%s: %v", key, err)
	}

	// 5. Keep host.last_seen_at current.
	w.db.TouchLastSeen(ctx, string(key)) //nolint:errcheck

	return nil
}
