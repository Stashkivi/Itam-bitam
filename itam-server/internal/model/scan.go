// Package model mirrors the agent's payload schema.
// Both services share this JSON contract; changes must be backward-compatible.
package model

import "time"

type ScanPayload struct {
	SchemaVersion  int       `json:"schema_ver"`
	PayloadType    string    `json:"payload_type"`
	HostUUID       string    `json:"host_uuid"`
	CapturedAt     time.Time `json:"captured_at"`
	Host           HostInfo  `json:"host"`
	Hardware       Hardware  `json:"hardware"`
	Interfaces     []NetIface `json:"interfaces"`
	Services       []Service  `json:"services"`
	Processes      []Process  `json:"processes"`
	OpenPorts      []Port     `json:"open_ports"`
	ScanDurationMS int64     `json:"scan_duration_ms"`
}

type HostInfo struct {
	Hostname   string `json:"hostname"`
	OS         string `json:"os"`
	Platform   string `json:"platform"`
	Arch       string `json:"arch"`
	KernelVer  string `json:"kernel_version"`
	UptimeSecs uint64 `json:"uptime_secs"`
}

type Hardware struct {
	CPUModel   string `json:"cpu_model"`
	CPUCores   int    `json:"cpu_cores"`
	CPUThreads int    `json:"cpu_threads"`
	TotalMemMB uint64 `json:"total_mem_mb"`
	Disks      []Disk `json:"disks"`
}

type Disk struct {
	Name       string `json:"name"`
	SizeGB     uint64 `json:"size_gb"`
	DriveType  string `json:"drive_type"`
	Filesystem string `json:"filesystem,omitempty"`
	MountPoint string `json:"mount_point,omitempty"`
}

type NetIface struct {
	Name      string   `json:"name"`
	MAC       string   `json:"mac"`
	Addresses []string `json:"addresses"`
	BytesSent uint64   `json:"bytes_sent"`
	BytesRecv uint64   `json:"bytes_recv"`
	IsUp      bool     `json:"is_up"`
}

type Process struct {
	PID        int32    `json:"pid"`
	PPID       int32    `json:"ppid"`
	Name       string   `json:"name"`
	Exe        string   `json:"exe"`
	Username   string   `json:"username"`
	CPUPCT     float64  `json:"cpu_pct"`
	MemRSSMB   float32  `json:"mem_rss_mb"`
	OpenPorts  []uint16 `json:"open_ports,omitempty"`
	Status     string   `json:"status"`
	CreateTime int64    `json:"create_time_unix"`
}

type Port struct {
	Port     uint16 `json:"port"`
	Protocol string `json:"protocol"`
	PID      int32  `json:"pid"`
	BindAddr string `json:"bind_addr"`
	Process  string `json:"process_name,omitempty"`
}

type Service struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
	PID     int32  `json:"pid,omitempty"`
}

// Anomaly is the event type broadcast to frontend clients via WebSocket
// and forwarded to Slack/Telegram.
type Anomaly struct {
	ID         int64       `json:"id"`
	HostUUID   string      `json:"host_uuid"`
	Hostname   string      `json:"hostname"`
	RuleID     string      `json:"rule_id"`
	Severity   string      `json:"severity"`
	EntityType string      `json:"entity_type,omitempty"`
	EntityKey  string      `json:"entity_key,omitempty"`
	Details    interface{} `json:"details,omitempty"`
	DetectedAt time.Time   `json:"detected_at"`
}
