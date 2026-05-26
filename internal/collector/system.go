package collector

import (
	"context"
	"runtime"

	"github.com/itam/agent/pkg/model"
	"github.com/shirou/gopsutil/v3/host"
)

func collectHostInfo(_ context.Context) (model.HostInfo, error) {
	info, err := host.Info()
	if err != nil {
		return model.HostInfo{Arch: runtime.GOARCH}, err
	}
	return model.HostInfo{
		Hostname:   info.Hostname,
		OS:         info.OS,
		Platform:   info.Platform,
		Arch:       info.KernelArch,
		KernelVer:  info.KernelVersion,
		UptimeSecs: info.Uptime,
	}, nil
}
