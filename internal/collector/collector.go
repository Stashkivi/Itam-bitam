// Package collector orchestrates concurrent system data collection.
package collector

import (
	"context"
	"sync"
	"time"

	"github.com/itam/agent/internal/siem"
	"github.com/itam/agent/pkg/model"
)

// Scanner owns the collection pipeline for one agent invocation.
type Scanner struct {
	hostUUID    string
	siemPlugins []siem.Plugin
}

func New(hostUUID string, plugins []siem.Plugin) *Scanner {
	return &Scanner{hostUUID: hostUUID, siemPlugins: plugins}
}

// Scan runs all sub-collectors concurrently and merges their results into a
// single ScanPayload. Partial failures are tolerated: a failed sub-collector
// leaves its section empty and does not abort the scan.
func (s *Scanner) Scan(ctx context.Context) (*model.ScanPayload, error) {
	start := time.Now()

	var (
		hostInfo   model.HostInfo
		hardware   model.Hardware
		interfaces []model.NetIface
		services   []model.Service
		processes  []model.Process
		openPorts  []model.Port
	)

	var wg sync.WaitGroup
	wg.Add(5)

	go func() {
		defer wg.Done()
		hostInfo, _ = collectHostInfo(ctx)
	}()
	go func() {
		defer wg.Done()
		hardware, _ = collectHardware(ctx)
	}()
	go func() {
		defer wg.Done()
		interfaces, _ = collectNetInterfaces(ctx)
	}()
	go func() {
		defer wg.Done()
		services, _ = collectServices(ctx)
	}()
	go func() {
		defer wg.Done()
		processes, openPorts, _ = collectProcessesAndPorts(ctx)
	}()

	wg.Wait()

	// Merge data from available SIEM plugins (non-blocking: skip on error).
	for _, p := range s.siemPlugins {
		if !p.IsAvailable() {
			continue
		}
		events, err := p.Collect(ctx)
		if err == nil {
			mergeAssetEvents(events, &processes)
		}
	}

	return &model.ScanPayload{
		SchemaVersion:  model.SchemaVersion,
		PayloadType:    "full_scan",
		HostUUID:       s.hostUUID,
		CapturedAt:     time.Now().UTC(),
		Host:           hostInfo,
		Hardware:       hardware,
		Interfaces:     interfaces,
		Services:       services,
		Processes:      processes,
		OpenPorts:      openPorts,
		ScanDurationMS: time.Since(start).Milliseconds(),
	}, nil
}

// mergeAssetEvents overlays SIEM-sourced process data onto the locally
// collected slice. SIEM data wins for overlapping fields because it carries
// enriched context (e.g. Wazuh's process reputation score).
func mergeAssetEvents(events []siem.AssetEvent, processes *[]model.Process) {
	for _, ev := range events {
		found := false
		for i := range *processes {
			if (*processes)[i].PID == ev.PID {
				if ev.Username != "" {
					(*processes)[i].Username = ev.Username
				}
				found = true
				break
			}
		}
		if !found && ev.PID > 0 {
			*processes = append(*processes, model.Process{
				PID:      ev.PID,
				Name:     ev.Name,
				Username: ev.Username,
				Status:   "siem_reported",
			})
		}
	}
}
