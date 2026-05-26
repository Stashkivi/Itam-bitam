package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/itam/agent/internal/cache"
	"github.com/itam/agent/internal/collector"
	"github.com/itam/agent/internal/config"
	"github.com/itam/agent/internal/scheduler"
	"github.com/itam/agent/internal/siem"
	"github.com/itam/agent/internal/transport"
)

func main() {
	cfgPath := flag.String("config", defaultConfigPath(), "path to agent config YAML")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Apply GC tuning from config before any significant allocation.
	debug.SetGCPercent(cfg.Agent.GOGC)
	debug.SetMemoryLimit(int64(cfg.Agent.MemLimitMB) * 1024 * 1024)

	// Derive or confirm host UUID.
	if cfg.Enrolled.HostUUID == "" {
		machineID, mac, hostname := config.MachineIdentity()
		cfg.Enrolled.HostUUID = config.DeriveHostUUID(machineID, mac, hostname)
	}

	// Enrollment: only runs once; subsequent starts skip this block.
	if cfg.Server.EnrollmentToken != "" && cfg.Enrolled.CertPath == "" {
		log.Println("[main] starting enrollment...")
		if err := transport.Enroll(context.Background(), cfg); err != nil {
			log.Fatalf("enrollment failed: %v", err)
		}
		cfg.Server.EnrollmentToken = "" // consume token — do not re-enroll
		if err := config.Save(cfg, *cfgPath); err != nil {
			log.Fatalf("save config after enrollment: %v", err)
		}
		log.Printf("[main] enrolled as %s", cfg.Enrolled.HostUUID)
	}

	// Transport client (mTLS + JWT).
	client, err := transport.New(cfg, *cfgPath)
	if err != nil {
		log.Fatalf("transport: %v", err)
	}

	// Offline cache.
	cacheKey := cache.DeriveKey(cfg.Enrolled.RefreshToken)
	store, err := cache.Open(
		filepath.Join(config.DataDir(cfg), "cache.db"),
		cacheKey,
		cfg.Cache.MaxPendingRows,
		cfg.Cache.MaxAttempts,
		cfg.Cache.FlushBatchSize,
	)
	if err != nil {
		log.Fatalf("cache: %v", err)
	}
	defer store.Close()

	// SIEM plugins — only register those the config enables.
	var plugins []siem.Plugin
	if cfg.SIEM.WazuhEnabled {
		plugins = append(plugins, siem.NewWazuh(cfg.SIEM.WazuhAlertsPath))
	}

	scanner := collector.New(cfg.Enrolled.HostUUID, plugins)

	scanFn := func(ctx context.Context) error {
		payload, err := scanner.Scan(ctx)
		if err != nil {
			log.Printf("[scan] partial error: %v", err)
		}
		if client.IsReachable(ctx) {
			return deliverOrCache(ctx, client, store, payload)
		}
		return store.Push(ctx, "full_scan", payload)
	}

	flushFn := func(ctx context.Context) error {
		return flushCache(ctx, client, store)
	}

	scanCron := cfg.Enrolled.ScanCron
	if scanCron == "" {
		scanCron = cfg.Agent.ScanCron
	}

	sched, err := scheduler.New(scanCron, scanFn, flushFn)
	if err != nil {
		log.Fatalf("scheduler: %v", err)
	}

	sched.Start()
	log.Printf("[main] agent running | host=%s | next_scan=%s",
		cfg.Enrolled.HostUUID, sched.NextScanTime().Format(time.RFC3339))

	// Run an immediate scan at startup so the server has fresh data.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		if err := scanFn(ctx); err != nil {
			log.Printf("[startup scan] %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[main] shutting down...")
	sched.Stop()
}

// deliverOrCache tries to POST the payload directly. On failure it falls back
// to the local cache so no data is lost.
func deliverOrCache(ctx context.Context, client *transport.Client, store *cache.Store, payload interface{}) error {
	resp, err := client.Post(ctx, "/v1/ingest/scan", payload)
	if err != nil || resp.StatusCode >= 500 {
		if resp != nil {
			resp.Body.Close()
		}
		log.Printf("[deliver] server unavailable, caching payload")
		return store.Push(ctx, "full_scan", payload)
	}
	if resp.StatusCode == http.StatusUnprocessableEntity {
		// Schema rejection — do not retry, log and discard.
		resp.Body.Close()
		log.Printf("[deliver] payload rejected by server (schema mismatch)")
		return nil
	}
	resp.Body.Close()
	return nil
}

// flushCache delivers pending rows to the server in FIFO order with
// exponential backoff per row on transient failures.
func flushCache(ctx context.Context, client *transport.Client, store *cache.Store) error {
	if !client.IsReachable(ctx) {
		return nil
	}

	rows, err := store.Peek(ctx)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	log.Printf("[flush] delivering %d cached payload(s)", len(rows))

	for _, row := range rows {
		var payload json.RawMessage = row.Payload
		resp, err := client.Post(ctx, "/v1/ingest/scan", payload)
		if err != nil {
			backoff(row.ID)
			store.Nack(ctx, row.ID, err.Error()) //nolint:errcheck
			continue
		}
		if resp.StatusCode >= 500 {
			resp.Body.Close()
			store.Nack(ctx, row.ID, fmt.Sprintf("status %d", resp.StatusCode)) //nolint:errcheck
			continue
		}
		resp.Body.Close()
		store.Ack(ctx, row.ID) //nolint:errcheck
	}
	return nil
}

// backoff sleeps for (2^attempt * 100ms) + jitter, capped at 30 seconds.
// attempt is approximated from the row ID as a best-effort stand-in.
func backoff(attempt int64) {
	base := 100 * time.Millisecond
	exp := math.Min(float64(attempt), 8)
	delay := time.Duration(math.Pow(2, exp)) * base
	jitter := time.Duration(rand.Int63n(int64(base)))
	cap := 30 * time.Second
	if delay+jitter > cap {
		delay = cap
		jitter = 0
	}
	time.Sleep(delay + jitter)
}

func defaultConfigPath() string {
	if p := os.Getenv("ITAM_CONFIG"); p != "" {
		return p
	}
	return filepath.Join(config.DataDir(&config.Config{}), "config.yaml")
}
