//go:build windows

package config

import (
	"net"
	"os"

	"golang.org/x/sys/windows/registry"
)

// MachineIdentity returns (machineID, primaryMAC, hostname) used to derive the
// stable host UUID. On Windows, the Cryptography MachineGuid registry value is
// stable across reboots and survives driver updates.
func MachineIdentity() (machineID, primaryMAC, hostname string) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE)
	if err == nil {
		defer k.Close()
		machineID, _, _ = k.GetStringValue("MachineGuid")
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
