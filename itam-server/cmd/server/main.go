package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/itam/server/internal/alert"
	"github.com/itam/server/internal/anomaly"
	"github.com/itam/server/internal/api"
	"github.com/itam/server/internal/config"
	"github.com/itam/server/internal/db/graph"
	"github.com/itam/server/internal/db/postgres"
	"github.com/itam/server/internal/enrollment"
	"github.com/itam/server/internal/hub"
	"github.com/itam/server/internal/kafka"
	"github.com/itam/server/internal/workers"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfgPath := flag.String("config", "config.yaml", "path to server config YAML")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ----------------------------------------------------------------
	// Storage layer
	// ----------------------------------------------------------------
	db, err := postgres.Connect(ctx, &cfg.Postgres)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer db.Close()
	log.Println("[main] postgres connected")

	graphClient, err := graph.Connect(ctx, &cfg.Graph)
	if err != nil {
		log.Fatalf("graph: %v", err)
	}
	defer graphClient.Close(ctx)
	log.Println("[main] graph connected")

	// ----------------------------------------------------------------
	// Redis
	// ----------------------------------------------------------------
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis: %v", err)
	}
	log.Println("[main] redis connected")

	// ----------------------------------------------------------------
	// Kafka producer (shared by API and anomaly engine)
	// ----------------------------------------------------------------
	producer, err := kafka.NewProducer(cfg.Kafka.Brokers)
	if err != nil {
		log.Fatalf("kafka producer: %v", err)
	}
	defer producer.Close()

	// ----------------------------------------------------------------
	// Security: internal CA + JWT secret
	// ----------------------------------------------------------------
	if err := os.MkdirAll(cfg.DataDir, 0700); err != nil {
		log.Fatalf("data dir: %v", err)
	}
	ca, err := enrollment.LoadOrCreateCA(
		cfg.DataDir+"/ca.crt",
		cfg.DataDir+"/ca.key",
		cfg.Enrollment.CertTTLDays,
	)
	if err != nil {
		log.Fatalf("CA: %v", err)
	}
	log.Println("[main] internal CA ready")

	// ----------------------------------------------------------------
	// WebSocket hub
	// ----------------------------------------------------------------
	wsHub := hub.New(rdb)

	// ----------------------------------------------------------------
	// API server
	// ----------------------------------------------------------------
	adminToken := cfg.Admin.Token
	if t := os.Getenv("ITAM_ADMIN_TOKEN"); t != "" {
		adminToken = t
	}

	deps := &api.Deps{
		DB:         db,
		Producer:   producer,
		Hub:        wsHub,
		CA:         ca,
		JWTSecret:  []byte(cfg.JWT.Secret),
		HMACSecret: []byte(cfg.Enrollment.HMACSecret),
		AdminToken: adminToken,
	}
	deps.JWTCfg.AccessTTLMin = cfg.JWT.AccessTTLMin
	deps.JWTCfg.RefreshTTLHour = cfg.JWT.RefreshTTLHour

	router := api.NewRouter(deps)

	httpServer := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutSec) * time.Second,
		TLSConfig:    &tls.Config{MinVersion: tls.VersionTLS13},
	}

	// ----------------------------------------------------------------
	// Background workers (each runs in its own goroutine)
	// ----------------------------------------------------------------
	errCh := make(chan error, 5)

	pgWriter, err := workers.NewPGWriter(cfg.Kafka.Brokers, db)
	if err != nil {
		log.Fatalf("pg-writer: %v", err)
	}
	go func() { errCh <- fmt.Errorf("pg-writer: %w", pgWriter.Run(ctx)) }()

	gWriter, err := workers.NewGraphWriter(cfg.Kafka.Brokers, graphClient)
	if err != nil {
		log.Fatalf("graph-writer: %v", err)
	}
	go func() { errCh <- fmt.Errorf("graph-writer: %w", gWriter.Run(ctx)) }()

	anomalyEngine, err := anomaly.NewEngine(cfg.Kafka.Brokers, db, producer, rdb)
	if err != nil {
		log.Fatalf("anomaly engine: %v", err)
	}
	go func() { errCh <- fmt.Errorf("anomaly: %w", anomalyEngine.Run(ctx)) }()

	// Alert dispatcher — channels are loaded from DB at startup.
	// For brevity, an empty channel list is passed; production code should
	// query alert_channels, decrypt config_enc, and populate the slice.
	dispatcher, err := alert.NewDispatcher(cfg.Kafka.Brokers, db, nil)
	if err != nil {
		log.Fatalf("alerter: %v", err)
	}
	go func() { errCh <- fmt.Errorf("alerter: %w", dispatcher.Run(ctx)) }()

	// ----------------------------------------------------------------
	// HTTP server
	// ----------------------------------------------------------------
	go func() {
		log.Printf("[main] listening on %s (TLS)", cfg.Server.Addr)
		var err error
		if cfg.Server.TLSCertPath != "" {
			err = httpServer.ListenAndServeTLS(cfg.Server.TLSCertPath, cfg.Server.TLSKeyPath)
		} else {
			// Dev mode: plain HTTP.
			log.Println("[main] WARNING: TLS disabled — use only for local development")
			err = httpServer.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("http: %w", err)
		}
	}()

	// ----------------------------------------------------------------
	// Graceful shutdown
	// ----------------------------------------------------------------
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Printf("[main] signal %s — shutting down", sig)
	case err := <-errCh:
		log.Printf("[main] worker error: %v — shutting down", err)
	}

	cancel() // stop all workers

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("[main] HTTP shutdown error: %v", err)
	}

	log.Println("[main] shutdown complete")
}
