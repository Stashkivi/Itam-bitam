// Package siem defines the plugin interface for consuming existing SIEM agents.
package siem

import "context"

// AssetEvent carries process/service data sourced from a SIEM agent.
// Fields that the SIEM did not populate remain zero-valued; the collector
// merge logic skips empty fields to avoid overwriting local data with blanks.
type AssetEvent struct {
	PID      int32
	Name     string
	Username string
	Exe      string
	Source   string // e.g. "wazuh", "elastic"
}

// Plugin is implemented by each SIEM integration.
type Plugin interface {
	// Name returns the human-readable plugin identifier.
	Name() string
	// IsAvailable reports whether the SIEM agent is installed and reachable
	// on this host. Called before Collect to short-circuit on disabled hosts.
	IsAvailable() bool
	// Collect fetches the latest asset events from the SIEM agent.
	Collect(ctx context.Context) ([]AssetEvent, error)
}
