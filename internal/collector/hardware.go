package collector

import (
	"context"
	"strings"

	"github.com/itam/agent/pkg/model"
	"github.com/jaypipes/ghw"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

func collectHardware(_ context.Context) (model.Hardware, error) {
	hw := model.Hardware{}

	if cpuInfos, err := cpu.Info(); err == nil && len(cpuInfos) > 0 {
		hw.CPUModel = cpuInfos[0].ModelName
	}
	if counts, err := cpu.Counts(false); err == nil {
		hw.CPUCores = counts
	}
	if counts, err := cpu.Counts(true); err == nil {
		hw.CPUThreads = counts
	}

	if vm, err := mem.VirtualMemory(); err == nil {
		hw.TotalMemMB = vm.Total / 1024 / 1024
	}

	hw.Disks = collectDisks()

	return hw, nil
}

func collectDisks() []model.Disk {
	parts, err := disk.Partitions(false)
	if err != nil {
		return nil
	}

	// ghw gives us physical drive information (NVMe/SSD/HDD).
	driveTypes := physicalDriveTypes()

	seen := make(map[string]bool)
	var disks []model.Disk
	for _, p := range parts {
		if seen[p.Device] {
			continue
		}
		seen[p.Device] = true

		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		d := model.Disk{
			Name:       p.Device,
			SizeGB:     usage.Total / 1024 / 1024 / 1024,
			Filesystem: p.Fstype,
			MountPoint: p.Mountpoint,
			DriveType:  driveTypes[p.Device],
		}
		if d.DriveType == "" {
			d.DriveType = "Unknown"
		}
		disks = append(disks, d)
	}
	return disks
}

// physicalDriveTypes builds a device-path → drive-type map using ghw.
// ghw may require elevated privileges on some systems; failures are tolerated.
func physicalDriveTypes() map[string]string {
	result := make(map[string]string)
	blockInfo, err := ghw.Block()
	if err != nil {
		return result
	}
	for _, disk := range blockInfo.Disks {
		driveType := "Unknown"
		name := strings.ToLower(disk.Name)
		switch {
		case disk.DriveType.String() == "SSD":
			driveType = "SSD"
		case strings.HasPrefix(name, "nvme"):
			driveType = "NVMe"
		case disk.DriveType.String() == "HDD":
			driveType = "HDD"
		}
		result["/dev/"+disk.Name] = driveType
	}
	return result
}
