// Package anomaly implements the rule-based deviation detection engine.
package anomaly

import (
	"fmt"

	"github.com/itam/server/internal/model"
)

// Rule evaluates one condition against (baseline, current) scan pair.
type Rule struct {
	ID       string
	Severity string // CRITICAL | HIGH | MEDIUM | LOW
	Eval     func(baseline, current *model.ScanPayload) []Finding
}

// Finding is a single rule violation instance.
type Finding struct {
	RuleID     string
	Severity   string
	EntityType string
	EntityKey  string
	Details    map[string]interface{}
}

// DefaultRules is the ordered set of detection rules applied on every scan.
var DefaultRules = []Rule{
	{
		ID:       "PORT_BIND_CHANGE",
		Severity: "CRITICAL",
		Eval: func(baseline, current *model.ScanPayload) []Finding {
			oldPorts := portMap(baseline.OpenPorts)
			var findings []Finding
			for _, p := range current.OpenPorts {
				key := portKey(p)
				if old, ok := oldPorts[key]; ok {
					if old.BindAddr != p.BindAddr {
						findings = append(findings, Finding{
							RuleID:     "PORT_BIND_CHANGE",
							Severity:   "CRITICAL",
							EntityType: "port",
							EntityKey:  key,
							Details: map[string]interface{}{
								"old_bind": old.BindAddr,
								"new_bind": p.BindAddr,
								"process":  p.Process,
							},
						})
					}
				}
			}
			return findings
		},
	},
	{
		ID:       "UNKNOWN_PORT",
		Severity: "HIGH",
		Eval: func(baseline, current *model.ScanPayload) []Finding {
			approved := portMap(baseline.OpenPorts)
			var findings []Finding
			for _, p := range current.OpenPorts {
				if _, ok := approved[portKey(p)]; !ok {
					findings = append(findings, Finding{
						RuleID:     "UNKNOWN_PORT",
						Severity:   "HIGH",
						EntityType: "port",
						EntityKey:  portKey(p),
						Details: map[string]interface{}{
							"port":     p.Port,
							"protocol": p.Protocol,
							"bind":     p.BindAddr,
							"process":  p.Process,
							"pid":      p.PID,
						},
					})
				}
			}
			return findings
		},
	},
	{
		ID:       "SERVICE_CRASHED",
		Severity: "HIGH",
		Eval: func(baseline, current *model.ScanPayload) []Finding {
			baselineRunning := make(map[string]bool)
			for _, s := range baseline.Services {
				if s.Status == "running" {
					baselineRunning[s.Name] = true
				}
			}
			currentStatus := make(map[string]string)
			for _, s := range current.Services {
				currentStatus[s.Name] = s.Status
			}
			var findings []Finding
			for name := range baselineRunning {
				if status, ok := currentStatus[name]; ok && status == "failed" {
					findings = append(findings, Finding{
						RuleID:     "SERVICE_CRASHED",
						Severity:   "HIGH",
						EntityType: "service",
						EntityKey:  name,
						Details:    map[string]interface{}{"service": name, "status": status},
					})
				}
			}
			return findings
		},
	},
	{
		ID:       "NEW_SERVICE",
		Severity: "MEDIUM",
		Eval: func(baseline, current *model.ScanPayload) []Finding {
			known := make(map[string]bool)
			for _, s := range baseline.Services {
				known[s.Name] = true
			}
			var findings []Finding
			for _, s := range current.Services {
				if !known[s.Name] {
					findings = append(findings, Finding{
						RuleID:     "NEW_SERVICE",
						Severity:   "MEDIUM",
						EntityType: "service",
						EntityKey:  s.Name,
						Details:    map[string]interface{}{"service": s.Name, "status": s.Status},
					})
				}
			}
			return findings
		},
	},
	{
		ID:       "UNKNOWN_PROCESS",
		Severity: "MEDIUM",
		Eval: func(baseline, current *model.ScanPayload) []Finding {
			approvedExes := make(map[string]bool)
			for _, p := range baseline.Processes {
				approvedExes[processKey(p)] = true
			}
			var findings []Finding
			for _, p := range current.Processes {
				if !approvedExes[processKey(p)] && p.Username == "root" {
					// Only flag unknown root processes to reduce noise.
					findings = append(findings, Finding{
						RuleID:     "UNKNOWN_PROCESS",
						Severity:   "MEDIUM",
						EntityType: "process",
						EntityKey:  processKey(p),
						Details: map[string]interface{}{
							"name": p.Name, "exe": p.Exe,
							"pid": p.PID, "user": p.Username,
						},
					})
				}
			}
			return findings
		},
	},
}

func portMap(ports []model.Port) map[string]model.Port {
	m := make(map[string]model.Port, len(ports))
	for _, p := range ports {
		m[portKey(p)] = p
	}
	return m
}

func portKey(p model.Port) string {
	return fmt.Sprintf("%d/%s", p.Port, p.Protocol)
}

func processKey(p model.Process) string {
	return fmt.Sprintf("%s|%s", p.Name, p.Exe)
}
