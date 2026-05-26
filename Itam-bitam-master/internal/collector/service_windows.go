//go:build windows

package collector

import (
	"context"

	"github.com/itam/agent/pkg/model"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

func collectServices(_ context.Context) ([]model.Service, error) {
	m, err := mgr.Connect()
	if err != nil {
		return nil, err
	}
	defer m.Disconnect()

	names, err := m.ListServices()
	if err != nil {
		return nil, err
	}

	services := make([]model.Service, 0, len(names))
	for _, name := range names {
		s, err := m.OpenService(name)
		if err != nil {
			continue
		}
		status, err := s.Query()
		s.Close()
		if err != nil {
			continue
		}
		services = append(services, model.Service{
			Name:   name,
			Status: svcStateString(status.State),
		})
	}
	return services, nil
}

func svcStateString(state svc.State) string {
	switch state {
	case svc.Running:
		return "running"
	case svc.Stopped:
		return "stopped"
	case svc.Paused:
		return "paused"
	case svc.StartPending, svc.StopPending, svc.ContinuePending, svc.PausePending:
		return "pending"
	default:
		return "unknown"
	}
}
