package postgres

import (
	"context"
	"encoding/json"

	"github.com/itam/server/internal/model"
	"github.com/jackc/pgx/v5"
)

// ActiveBaseline fetches the approved baseline payload for a host.
// Returns nil if no active baseline exists (first scan, or manually cleared).
func (db *DB) ActiveBaseline(ctx context.Context, hostID string) (*model.ScanPayload, error) {
	var raw json.RawMessage
	err := db.Pool.QueryRow(ctx, `
		SELECT payload FROM baselines
		WHERE host_id = $1 AND is_active = true
		ORDER BY approved_at DESC
		LIMIT 1`, hostID,
	).Scan(&raw)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var payload model.ScanPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// SetBaseline deactivates all previous baselines for the host and inserts a
// new active one. snapshotID may be zero when no snapshot ID is known.
func (db *DB) SetBaseline(ctx context.Context, hostID, approvedBy string, snapshotID int64, payload interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`UPDATE baselines SET is_active = false WHERE host_id = $1 AND is_active = true`,
		hostID,
	); err != nil {
		return err
	}

	var sid *int64
	if snapshotID > 0 {
		sid = &snapshotID
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO baselines (host_id, approved_by, snapshot_id, is_active, payload)
		VALUES ($1, $2, $3, true, $4)`,
		hostID, approvedBy, sid, raw,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// AutoApproveFirstScan sets the baseline automatically when a host has
// no existing baseline. Called by the pg-writer consumer after first scan.
func (db *DB) AutoApproveFirstScan(ctx context.Context, hostID string, payload interface{}, snapshotID int64) error {
	existing, err := db.ActiveBaseline(ctx, hostID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil // baseline already exists
	}
	return db.SetBaseline(ctx, hostID, "system:auto", snapshotID, payload)
}
