package collector

import (
	"context"

	"github.com/itam/agent/pkg/model"
	"github.com/shirou/gopsutil/v3/net"
)

func collectNetInterfaces(_ context.Context) ([]model.NetIface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	counters, _ := net.IOCounters(true) // per-interface counters; tolerate failure
	counterMap := make(map[string]net.IOCountersStat, len(counters))
	for _, c := range counters {
		counterMap[c.Name] = c
	}

	result := make([]model.NetIface, 0, len(ifaces))
	for _, iface := range ifaces {
		addrs := make([]string, 0, len(iface.Addrs))
		for _, a := range iface.Addrs {
			addrs = append(addrs, a.Addr)
		}

		ni := model.NetIface{
			Name:      iface.Name,
			MAC:       iface.HardwareAddr,
			Addresses: addrs,
			IsUp:      iface.Flags&net.FlagUp != 0,
		}
		if c, ok := counterMap[iface.Name]; ok {
			ni.BytesSent = c.BytesSent
			ni.BytesRecv = c.BytesRecv
		}
		result = append(result, ni)
	}
	return result, nil
}
