//go:build windows

package collector

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// buildPortToPIDMap on Windows parses `netstat -ano` to correlate LISTENING
// ports with their owning PIDs. This avoids CGo while covering all Windows
// versions from Server 2012 onward.
func buildPortToPIDMap() (map[uint16]int32, error) {
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return nil, fmt.Errorf("netstat: %w", err)
	}

	portToPID := make(map[uint16]int32)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		proto := strings.ToUpper(fields[0])
		var (
			localAddr string
			pidStr    string
		)

		switch proto {
		case "TCP":
			// TCP line: Proto  LocalAddr  ForeignAddr  State  PID
			if len(fields) < 5 || fields[3] != "LISTENING" {
				continue
			}
			localAddr = fields[1]
			pidStr = fields[4]
		case "UDP":
			// UDP line: Proto  LocalAddr  *:*  PID  (no State column)
			if len(fields) < 4 {
				continue
			}
			localAddr = fields[1]
			pidStr = fields[3]
		default:
			continue
		}

		port, err := extractLastPort(localAddr)
		if err != nil {
			continue
		}
		pid, err := strconv.ParseInt(strings.TrimSpace(pidStr), 10, 32)
		if err != nil {
			continue
		}
		portToPID[port] = int32(pid)
	}
	return portToPID, nil
}

// extractLastPort handles both "1.2.3.4:8080" and "[::1]:8080" address formats.
func extractLastPort(addr string) (uint16, error) {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return 0, fmt.Errorf("no colon in %q", addr)
	}
	v, err := strconv.ParseUint(strings.TrimSpace(addr[idx+1:]), 10, 16)
	if err != nil {
		return 0, err
	}
	return uint16(v), nil
}
