package workers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/itam/server/internal/db/graph"
	"github.com/itam/server/internal/kafka"
	"github.com/itam/server/internal/model"
)

// GraphWriter consumes full scans and keeps the live dependency graph current.
type GraphWriter struct {
	graph    *graph.Client
	consumer *kafka.Consumer
}

func NewGraphWriter(brokers []string, graphClient *graph.Client) (*GraphWriter, error) {
	w := &GraphWriter{graph: graphClient}
	consumer, err := kafka.NewConsumer(
		brokers,
		kafka.GroupGraphWriter,
		[]string{kafka.TopicScanFull},
		w.handle,
	)
	if err != nil {
		return nil, err
	}
	w.consumer = consumer
	return w, nil
}

func (w *GraphWriter) Run(ctx context.Context) error {
	defer w.consumer.Close()
	return w.consumer.Run(ctx)
}

func (w *GraphWriter) handle(ctx context.Context, _ string, _, value []byte) error {
	var scan model.ScanPayload
	if err := json.Unmarshal(value, &scan); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return w.graph.UpsertScan(ctx, &scan)
}
