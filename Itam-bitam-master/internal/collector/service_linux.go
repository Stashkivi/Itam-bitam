//go:build linux

package collector

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/itam/agent/pkg/model"
)

type systemdUnit struct {
	Unit        string `json:"unit"`
	ActiveState string `json:"active"`
	SubState    string `json:"sub"`
}

func collectServices(ctx context.Context) ([]model.Service, error) {
	out, err := exec.CommandContext(ctx,
		"systemctl", "list-units",
		"--type=service", "--all",
		"--output=json", "--no-pager",
	).Output()
	if err != nil {
		return nil, err
	}

	var units []systemdUnit
	if err := json.Unmarshal(out, &units); err != nil {
		return nil, err
	}

	services := make([]model.Service, 0, len(units))
	for _, u := range units {
		services = append(services, model.Service{
			Name:   strings.TrimSuffix(u.Unit, ".service"),
			Status: mapSystemdState(u.ActiveState, u.SubState),
		})
	}
	return services, nil
}

func mapSystemdState(active, sub string) string {
	switch {
	case active == "active" && sub == "running":
		return "running"
	case active == "failed":
		return "failed"
	case active == "inactive":
		return "stopped"
	default:
		return active
	}
}
