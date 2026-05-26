package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type Host struct {
	ID          string
	OrgID       string
	Hostname    string
	OSFamily    string
	Arch        string
	IPPrimary   *string
	EnrolledAt  time.Time
	LastSeenAt  *time.Time
	Status      string
}

// UpsertHost inserts a new host or updates last_seen_at and ip_primary.
// Returns the canonical host UUID.
func (db *DB) UpsertHost(ctx context.Context, orgID, hostUUID, hostname, osFamily, arch string, ip *string) (string, error) {
	var id string
	err := db.Pool.QueryRow(ctx, `
		INSERT INTO hosts (id, org_id, hostname, os_family, arch, ip_primary, enrolled_at, last_seen_at)
		VALUES ($1, $2, $3, $4, $5, $6, now(), now())
		ON CONFLICT (hostname, org_id) DO UPDATE
		    SET last_seen_at = now(),
		        ip_primary   = COALESCE(EXCLUDED.ip_primary, hosts.ip_primary),
		        status       = 'active'
		RETURNING id`,
		hostUUID, orgID, hostname, osFamily, arch, ip,
	).Scan(&id)
	return id, err
}

// GetHostByUUID fetches a host record by its UUID.
func (db *DB) GetHostByUUID(ctx context.Context, uuid string) (*Host, error) {
	row := db.Pool.QueryRow(ctx, `
		SELECT id, org_id, hostname, os_family, arch, ip_primary,
		       enrolled_at, last_seen_at, status
		FROM hosts WHERE id = $1`, uuid)

	var h Host
	err := row.Scan(&h.ID, &h.OrgID, &h.Hostname, &h.OSFamily, &h.Arch,
		&h.IPPrimary, &h.EnrolledAt, &h.LastSeenAt, &h.Status)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &h, err
}

// ListHosts returns all active hosts for an organisation.
func (db *DB) ListHosts(ctx context.Context, orgID string) ([]Host, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT id, org_id, hostname, os_family, arch, ip_primary,
		       enrolled_at, last_seen_at, status
		FROM hosts
		WHERE org_id = $1 AND status = 'active'
		ORDER BY hostname`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []Host
	for rows.Next() {
		var h Host
		if err := rows.Scan(&h.ID, &h.OrgID, &h.Hostname, &h.OSFamily, &h.Arch,
			&h.IPPrimary, &h.EnrolledAt, &h.LastSeenAt, &h.Status); err != nil {
			return nil, err
		}
		hosts = append(hosts, h)
	}
	return hosts, rows.Err()
}

// TouchLastSeen updates last_seen_at without a full upsert (used by heartbeat handler).
func (db *DB) TouchLastSeen(ctx context.Context, hostUUID string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE hosts SET last_seen_at = now() WHERE id = $1`, hostUUID)
	return err
}
