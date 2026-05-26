//go:build linux

package config

import (
	"net"
	"os"
	"strings"
)

// MachineIdentity returns (machineID, primaryMAC, hostname) used to derive the
// stable host UUID. On Linux, /etc/machine-id is the most reliable source.
func MachineIdentity() (machineID, primaryMAC, hostname string) {
	raw, err := os.ReadFile("/etc/machine-id")
	if err == nil {
		machineID = strings.TrimSpace(string(raw))
	}
	primaryMAC = firstHardwareMAC()
	hostname, _ = os.Hostname()
	return
}

func firstHardwareMAC() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if len(iface.HardwareAddr) > 0 {
			return iface.HardwareAddr.String()
		}
	}
	return ""
}
