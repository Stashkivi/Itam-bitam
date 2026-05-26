package collector

import (
	"context"

	"github.com/itam/agent/pkg/model"
	"github.com/shirou/gopsutil/v3/process"
)

// collectProcessesAndPorts builds the process list and the top-level port list
// in one pass to avoid calling buildPortToPIDMap twice.
func collectProcessesAndPorts(ctx context.Context) ([]model.Process, []model.Port, error) {
	portToPID, err := buildPortToPIDMap()
	if err != nil {
		portToPID = make(map[uint16]int32)
	}

	// Invert: PID -> []port for enriching each Process entry.
	pidToPorts := make(map[int32][]uint16)
	for port, pid := range portToPID {
		pidToPorts[pid] = append(pidToPorts[pid], port)
	}

	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Pre-build a name lookup so Port entries can include process_name.
	pidToName := make(map[int32]string, len(procs))

	result := make([]model.Process, 0, len(procs))
	for _, p := range procs {
		name, _ := p.NameWithContext(ctx)
		pidToName[p.Pid] = name

		exe, _ := p.ExeWithContext(ctx)
		ppid, _ := p.PpidWithContext(ctx)
		user, _ := p.UsernameWithContext(ctx)
		memInfo, _ := p.MemoryInfoWithContext(ctx)
		cpuPct, _ := p.CPUPercentWithContext(ctx)
		statuses, _ := p.StatusWithContext(ctx)
		createTime, _ := p.CreateTimeWithContext(ctx)

		status := "unknown"
		if len(statuses) > 0 {
			status = statuses[0]
		}

		var rss float32
		if memInfo != nil {
			rss = float32(memInfo.RSS) / 1024 / 1024
		}

		result = append(result, model.Process{
			PID:        p.Pid,
			PPID:       ppid,
			Name:       name,
			Exe:        exe,
			Username:   user,
			CPUPCT:     cpuPct,
			MemRSSMB:   rss,
			OpenPorts:  pidToPorts[p.Pid],
			Status:     status,
			CreateTime: createTime,
		})
	}

	// Build the top-level Port list.
	ports := make([]model.Port, 0, len(portToPID))
	for port, pid := range portToPID {
		ports = append(ports, model.Port{
			Port:     port,
			Protocol: "tcp",
			PID:      pid,
			Process:  pidToName[pid],
		})
	}

	return result, ports, nil
}
