//go:build linux

package collector

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type socketEntry struct {
	LocalPort uint16
	Inode     uint64
}

// buildPortToPIDMap returns a map of listening port → owning PID.
// It reads /proc/net/tcp[6] and /proc/net/udp[6] for socket inodes,
// then resolves inodes to PIDs by scanning /proc/<pid>/fd symlinks.
func buildPortToPIDMap() (map[uint16]int32, error) {
	inodeToPID, err := buildInodeToPIDMap()
	if err != nil {
		return nil, err
	}

	portToPID := make(map[uint16]int32)
	for _, procFile := range []string{
		"/proc/net/tcp", "/proc/net/tcp6",
		"/proc/net/udp", "/proc/net/udp6",
	} {
		entries, err := parseProcNetFile(procFile)
		if err != nil {
			continue // file absent means protocol not supported
		}
		for _, e := range entries {
			if pid, ok := inodeToPID[e.Inode]; ok {
				portToPID[e.LocalPort] = pid
			}
		}
	}
	return portToPID, nil
}

// buildInodeToPIDMap scans /proc/<pid>/fd/* for socket symlinks.
func buildInodeToPIDMap() (map[uint64]int32, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	m := make(map[uint64]int32)
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		fds, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", pid))
		if err != nil {
			continue // process may have exited
		}
		for _, fd := range fds {
			target, err := os.Readlink(filepath.Join(fmt.Sprintf("/proc/%d/fd", pid), fd.Name()))
			if err != nil {
				continue
			}
			// Symlink target format: "socket:[inode]"
			var inode uint64
			if _, err := fmt.Sscanf(target, "socket:[%d]", &inode); err == nil {
				m[inode] = int32(pid)
			}
		}
	}
	return m, nil
}

// parseProcNetFile parses a /proc/net/tcp-family file.
// Only LISTEN (0x0A) state entries are returned for TCP;
// all UDP entries are included since UDP has no listen state.
func parseProcNetFile(path string) ([]socketEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	isUDP := strings.Contains(path, "udp")
	lines := strings.Split(string(data), "\n")
	var out []socketEntry

	for _, line := range lines[1:] { // skip header
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		// fields[3] is the hex connection state; 0x0A = TCP_LISTEN
		state, err := strconv.ParseUint(fields[3], 16, 8)
		if err != nil {
			continue
		}
		if !isUDP && state != 0x0A {
			continue
		}

		port, err := parseHexPort(fields[1])
		if err != nil {
			continue
		}

		inode, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			continue
		}

		out = append(out, socketEntry{LocalPort: port, Inode: inode})
	}
	return out, nil
}

// parseHexPort extracts the port from a "XXXXXXXX:PPPP" /proc/net address field.
// The address portion is little-endian, but the port is always big-endian uint16.
func parseHexPort(hexAddr string) (uint16, error) {
	parts := strings.SplitN(hexAddr, ":", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("bad addr %q", hexAddr)
	}
	portBytes, err := hex.DecodeString(parts[1])
	if err != nil || len(portBytes) != 2 {
		return 0, fmt.Errorf("bad port %q", parts[1])
	}
	return binary.BigEndian.Uint16(portBytes), nil
}

// parseHexIP converts a little-endian hex IPv4 field to net.IP.
// Exported for use in tests.
func parseHexIP(hexAddr string) (net.IP, error) {
	parts := strings.SplitN(hexAddr, ":", 2)
	b, err := hex.DecodeString(parts[0])
	if err != nil || len(b) != 4 {
		return nil, fmt.Errorf("bad ip %q", parts[0])
	}
	// Reverse from little-endian storage order.
	return net.IP{b[3], b[2], b[1], b[0]}, nil
}
