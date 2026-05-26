package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

// InsertSnapshot persists a full scan payload as an immutable snapshot.
// Returns the new snapshot ID.
func (db *DB) InsertSnapshot(ctx context.Context, hostID string, capturedAt time.Time, payload interface{}) (int64, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	var id int64
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO host_snapshots (host_id, captured_at, payload)
		VALUES ($1, $2, $3)
		RETURNING id`,
		hostID, capturedAt, raw,
	).Scan(&id)
	return id, err
}

// LatestSnapshot returns the most recent snapshot payload for a host.
// Returns nil payload and nil error when no snapshot exists.
func (db *DB) LatestSnapshot(ctx context.Context, hostID string) (json.RawMessage, time.Time, error) {
	var raw json.RawMessage
	var capturedAt time.Time
	err := db.Pool.QueryRow(ctx, `
		SELECT payload, captured_at
		FROM host_snapshots
		WHERE host_id = $1
		ORDER BY captured_at DESC
		LIMIT 1`, hostID,
	).Scan(&raw, &capturedAt)
	if err == pgx.ErrNoRows {
		return nil, time.Time{}, nil
	}
	return raw, capturedAt, err
}

// SnapshotsBetween returns all snapshots for a host in a time range, ordered oldest-first.
func (db *DB) SnapshotsBetween(ctx context.Context, hostID string, from, to time.Time) ([]json.RawMessage, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT payload FROM host_snapshots
		WHERE host_id = $1 AND captured_at BETWEEN $2 AND $3
		ORDER BY captured_at ASC`,
		hostID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []json.RawMessage
	for rows.Next() {
		var raw json.RawMessage
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		result = append(result, raw)
	}
	return result, rows.Err()
}
