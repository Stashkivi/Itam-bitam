// Package scheduler drives the agent's scan and flush cycles using a cron
// expression managed by the server. The cron expression can be updated at
// runtime without restarting the agent.
package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler wraps robfig/cron and allows swapping the scan job when the
// server pushes a new cron expression.
type Scheduler struct {
	c        *cron.Cron
	scanJobID cron.EntryID
	flushJobID cron.EntryID
	mu       sync.Mutex
}

// ScanFunc is the callback invoked on each scan tick.
type ScanFunc func(ctx context.Context) error

// FlushFunc is the callback invoked on each flush tick.
type FlushFunc func(ctx context.Context) error

// New creates a Scheduler with the given cron expression for scans and a
// fixed 5-minute flush interval for the offline cache.
func New(scanCron string, scan ScanFunc, flush FlushFunc) (*Scheduler, error) {
	c := cron.New(cron.WithSeconds())

	s := &Scheduler{c: c}
	if err := s.setScanJob(scanCron, scan); err != nil {
		return nil, err
	}

	// Flush cycle: every 5 minutes regardless of scan schedule.
	id, err := c.AddFunc("0 */5 * * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := flush(ctx); err != nil {
			log.Printf("[scheduler] flush error: %v", err)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("add flush job: %w", err)
	}
	s.flushJobID = id

	return s, nil
}

// Start begins the cron loop in the background.
func (s *Scheduler) Start() { s.c.Start() }

// Stop gracefully halts the cron loop, waiting for any running job to finish.
func (s *Scheduler) Stop() { s.c.Stop() }

// UpdateScanCron replaces the scan cron expression at runtime.
// Called when the server pushes a new schedule via the config endpoint.
func (s *Scheduler) UpdateScanCron(expr string, scan ScanFunc) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.c.Remove(s.scanJobID)
	return s.setScanJob(expr, scan)
}

// NextScanTime returns the next scheduled scan time for logging / health checks.
func (s *Scheduler) NextScanTime() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry := s.c.Entry(s.scanJobID)
	return entry.Next
}

func (s *Scheduler) setScanJob(expr string, scan ScanFunc) error {
	id, err := s.c.AddFunc(expr, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := scan(ctx); err != nil {
			log.Printf("[scheduler] scan error: %v", err)
		}
	})
	if err != nil {
		return fmt.Errorf("invalid cron %q: %w", expr, err)
	}
	s.scanJobID = id
	return nil
}
