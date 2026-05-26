// Package postgres wraps pgxpool and owns schema migrations.
package postgres

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/itam/server/internal/config"
)

// DB is a thin wrapper around pgxpool.Pool exposing domain-level methods.
type DB struct {
	Pool *pgxpool.Pool
}

// Connect creates the connection pool and runs pending SQL migrations.
func Connect(ctx context.Context, cfg *config.PostgresConfig) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.MaxConns)
	poolCfg.MinConns = int32(cfg.MinConns)

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("pool connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	db := &DB{Pool: pool}
	if err := db.migrate(ctx, cfg.MigrationsDir); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// Close releases all pool connections.
func (db *DB) Close() { db.Pool.Close() }

// migrate applies all *.sql files in migrationsDir in lexicographic order.
// It uses an advisory lock so concurrent servers don't race during startup.
func (db *DB) migrate(ctx context.Context, dir string) error {
	// Create the migration tracking table if absent.
	_, err := db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir %s: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	// Advisory lock prevents concurrent migration races on multi-instance deploys.
	conn, err := db.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(987654321)`); err != nil {
		return err
	}
	defer conn.Exec(ctx, `SELECT pg_advisory_unlock(987654321)`) //nolint:errcheck

	for _, name := range files {
		var applied bool
		err := conn.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)`, name,
		).Scan(&applied)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		sql, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}

		if _, err := conn.Exec(ctx,
			`INSERT INTO schema_migrations (filename) VALUES ($1)`, name,
		); err != nil {
			return fmt.Errorf("record %s: %w", name, err)
		}
	}

	return nil
}
