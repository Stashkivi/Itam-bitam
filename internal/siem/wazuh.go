package siem

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// WazuhPlugin tails the Wazuh alerts JSON file and extracts syscollector
// process events. It is read-only and does not interfere with the Wazuh agent.
type WazuhPlugin struct {
	alertsPath string
	lastOffset int64
}

func NewWazuh(alertsPath string) *WazuhPlugin {
	return &WazuhPlugin{alertsPath: alertsPath}
}

func (w *WazuhPlugin) Name() string { return "wazuh" }

// IsAvailable checks that the Wazuh alerts file exists and is readable.
func (w *WazuhPlugin) IsAvailable() bool {
	_, err := os.Stat(w.alertsPath)
	return err == nil
}

type wazuhAlert struct {
	Data struct {
		Syscollector struct {
			Process struct {
				PID  int    `json:"pid"`
				Name string `json:"name"`
				Cmd  string `json:"cmd"`
				Euser string `json:"euser"`
			} `json:"process"`
		} `json:"syscollector"`
	} `json:"data"`
	Rule struct {
		Groups []string `json:"groups"`
	} `json:"rule"`
	Timestamp string `json:"timestamp"`
}

// Collect reads new lines from the alerts file since the last call.
// On first call lastOffset is 0 so we only read lines written in the
// last 60 seconds to avoid replaying an entire day of alerts.
func (w *WazuhPlugin) Collect(_ context.Context) ([]AssetEvent, error) {
	f, err := os.Open(w.alertsPath)
	if err != nil {
		return nil, fmt.Errorf("wazuh open: %w", err)
	}
	defer f.Close()

	// On first call, seek to a position that covers only the last 60 s.
	if w.lastOffset == 0 {
		size, _ := f.Seek(0, io.SeekEnd)
		// Rough heuristic: seek back 512 KB to cover recent lines.
		seek := size - 512*1024
		if seek < 0 {
			seek = 0
		}
		w.lastOffset = seek
	}

	if _, err := f.Seek(w.lastOffset, io.SeekStart); err != nil {
		return nil, err
	}

	cutoff := time.Now().Add(-60 * time.Second)
	var events []AssetEvent

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		var alert wazuhAlert
		if err := json.Unmarshal(line, &alert); err != nil {
			continue
		}

		// Only process syscollector process-inventory events.
		if !containsGroup(alert.Rule.Groups, "syscollector") {
			continue
		}

		ts, err := time.Parse(time.RFC3339, alert.Timestamp)
		if err != nil || ts.Before(cutoff) {
			continue
		}

		p := alert.Data.Syscollector.Process
		if p.PID == 0 {
			continue
		}
		events = append(events, AssetEvent{
			PID:      int32(p.PID),
			Name:     p.Name,
			Exe:      p.Cmd,
			Username: p.Euser,
			Source:   "wazuh",
		})
	}

	w.lastOffset, _ = f.Seek(0, io.SeekCurrent)
	return events, scanner.Err()
}

func containsGroup(groups []string, target string) bool {
	for _, g := range groups {
		if g == target {
			return true
		}
	}
	return false
}
