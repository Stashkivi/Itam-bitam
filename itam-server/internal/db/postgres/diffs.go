package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/itam/server/internal/model"
	"github.com/jackc/pgx/v5"
)

// DiffEntry represents one atomic entity change detected between snapshots.
type DiffEntry struct {
	EntityType string          // service | process | port | interface
	EntityKey  string          // stable identity string
	Operation  string          // added | removed | modified
	OldValue   json.RawMessage
	NewValue   json.RawMessage
}

// ComputeAndStoreDiffs compares prev and next scans, writes all changes to
// config_diffs, and returns the generated entries for downstream use.
func (db *DB) ComputeAndStoreDiffs(
	ctx context.Context,
	hostID string,
	prev, next *model.ScanPayload,
) ([]DiffEntry, error) {
	if prev == nil {
		return nil, nil // no baseline yet — first scan, nothing to diff
	}

	diffs := computeDiffs(prev, next)
	if len(diffs) == 0 {
		return nil, nil
	}

	changedAt := next.CapturedAt

	// Batch insert all diffs in one round-trip.
	batch := &pgx.Batch{}
	for _, d := range diffs {
		batch.Queue(`
			INSERT INTO config_diffs
			    (host_id, changed_at, entity_type, entity_key, operation, old_value, new_value, diff_hash)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			hostID, changedAt, d.EntityType, d.EntityKey, d.Operation,
			d.OldValue, d.NewValue, diffHash(d),
		)
	}

	br := db.Pool.SendBatch(ctx, batch)
	defer br.Close()
	for range diffs {
		if _, err := br.Exec(); err != nil {
			return diffs, fmt.Errorf("insert diff: %w", err)
		}
	}

	return diffs, nil
}

// RecentDiffs returns the last N diffs for a host ordered newest-first.
func (db *DB) RecentDiffs(ctx context.Context, hostID string, limit int) ([]DiffEntry, error) {
	rows, err := db.Pool.Query(ctx, `
		SELECT entity_type, entity_key, operation, old_value, new_value
		FROM config_diffs
		WHERE host_id = $1
		ORDER BY changed_at DESC
		LIMIT $2`, hostID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DiffEntry
	for rows.Next() {
		var d DiffEntry
		if err := rows.Scan(&d.EntityType, &d.EntityKey, &d.Operation, &d.OldValue, &d.NewValue); err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

// ---- diff computation logic ----

func computeDiffs(prev, next *model.ScanPayload) []DiffEntry {
	var diffs []DiffEntry
	diffs = append(diffs, diffServices(prev.Services, next.Services)...)
	diffs = append(diffs, diffPorts(prev.OpenPorts, next.OpenPorts)...)
	diffs = append(diffs, diffProcesses(prev.Processes, next.Processes)...)
	diffs = append(diffs, diffInterfaces(prev.Interfaces, next.Interfaces)...)
	return diffs
}

func diffServices(prev, next []model.Service) []DiffEntry {
	old := make(map[string]model.Service, len(prev))
	for _, s := range prev {
		old[s.Name] = s
	}
	nw := make(map[string]model.Service, len(next))
	for _, s := range next {
		nw[s.Name] = s
	}
	return genericDiff("service", old, nw, func(a, b interface{}) bool {
		sa, sb := a.(model.Service), b.(model.Service)
		return sa.Status == sb.Status && sa.Version == sb.Version
	})
}

func diffPorts(prev, next []model.Port) []DiffEntry {
	key := func(p model.Port) string {
		return fmt.Sprintf("%d/%s", p.Port, p.Protocol)
	}
	old := make(map[string]model.Port, len(prev))
	for _, p := range prev {
		old[key(p)] = p
	}
	nw := make(map[string]model.Port, len(next))
	for _, p := range next {
		nw[key(p)] = p
	}
	return genericDiff("port", old, nw, func(a, b interface{}) bool {
		pa, pb := a.(model.Port), b.(model.Port)
		return pa.BindAddr == pb.BindAddr && pa.Process == pb.Process
	})
}

// Processes are keyed by name+exe — PIDs are ephemeral and not stable.
func diffProcesses(prev, next []model.Process) []DiffEntry {
	key := func(p model.Process) string {
		return fmt.Sprintf("%s|%s", p.Name, p.Exe)
	}
	old := make(map[string]model.Process, len(prev))
	for _, p := range prev {
		old[key(p)] = p
	}
	nw := make(map[string]model.Process, len(next))
	for _, p := range next {
		nw[key(p)] = p
	}
	return genericDiff("process", old, nw, func(a, b interface{}) bool {
		pa, pb := a.(model.Process), b.(model.Process)
		return pa.Status == pb.Status && pa.Username == pb.Username
	})
}

func diffInterfaces(prev, next []model.NetIface) []DiffEntry {
	old := make(map[string]model.NetIface, len(prev))
	for _, i := range prev {
		old[i.Name] = i
	}
	nw := make(map[string]model.NetIface, len(next))
	for _, i := range next {
		nw[i.Name] = i
	}
	return genericDiff("interface", old, nw, func(a, b interface{}) bool {
		ia, ib := a.(model.NetIface), b.(model.NetIface)
		return ia.MAC == ib.MAC && ia.IsUp == ib.IsUp
	})
}

// genericDiff performs a symmetric difference between two typed maps.
// equal is called only for keys present in both maps; if it returns false,
// a "modified" entry is generated.
func genericDiff[V any](entityType string, old, nw map[string]V, equal func(a, b interface{}) bool) []DiffEntry {
	var diffs []DiffEntry
	for k, ov := range old {
		if nv, ok := nw[k]; !ok {
			diffs = append(diffs, entry(entityType, k, "removed", ov, nil))
		} else if !equal(ov, nv) {
			diffs = append(diffs, entry(entityType, k, "modified", ov, nv))
		}
	}
	for k, nv := range nw {
		if _, ok := old[k]; !ok {
			diffs = append(diffs, entry(entityType, k, "added", nil, nv))
		}
	}
	return diffs
}

func entry(entityType, key, op string, old, new interface{}) DiffEntry {
	oldJSON, _ := json.Marshal(old)
	newJSON, _ := json.Marshal(new)
	if string(oldJSON) == "null" {
		oldJSON = nil
	}
	if string(newJSON) == "null" {
		newJSON = nil
	}
	return DiffEntry{
		EntityType: entityType,
		EntityKey:  key,
		Operation:  op,
		OldValue:   oldJSON,
		NewValue:   newJSON,
	}
}

func diffHash(d DiffEntry) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%s|%s", d.EntityType, d.EntityKey, d.Operation, d.OldValue, d.NewValue)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// Unused import guard for time package.
var _ = time.Now
