package model

import "time"

const SchemaVersion = 1

// ScanPayload is the canonical envelope sent to the server on every scan cycle.
type ScanPayload struct {
	SchemaVersion int       `json:"schema_ver"`
	PayloadType   string    `json:"payload_type"` // "full_scan" | "diff" | "heartbeat"
	HostUUID      string    `json:"host_uuid"`
	CapturedAt    time.Time `json:"captured_at"`
	Host          HostInfo  `json:"host"`
	Hardware      Hardware  `json:"hardware"`
	Interfaces    []NetIface `json:"interfaces"`
	Services      []Service  `json:"services"`
	Processes     []Process  `json:"processes"`
	OpenPorts     []Port     `json:"open_ports"`
	ScanDurationMS int64     `json:"scan_duration_ms"`
}

type HostInfo struct {
	Hostname    string `json:"hostname"`
	OS          string `json:"os"`
	Platform    string `json:"platform"`
	Arch        string `json:"arch"`
	KernelVer   string `json:"kernel_version"`
	UptimeSecs  uint64 `json:"uptime_secs"`
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
	DriveType  string `json:"drive_type"` // HDD | SSD | NVMe | Unknown
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

// Process represents a running process with its resolved network bindings.
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

// Port is a top-level port entry enriched with its owning PID.
type Port struct {
	Port     uint16 `json:"port"`
	Protocol string `json:"protocol"` // tcp | udp
	PID      int32  `json:"pid"`
	BindAddr string `json:"bind_addr"`
	Process  string `json:"process_name,omitempty"`
}

// Service represents a system service (systemd unit on Linux, SCM entry on Windows).
type Service struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // running | stopped | failed | paused | unknown
	Version string `json:"version,omitempty"`
	PID     int32  `json:"pid,omitempty"`
}

// EnrollRequest is sent by the agent on first run.
type EnrollRequest struct {
	HostUUID        string `json:"host_uuid"`
	PublicKeyPEM    string `json:"public_key_pem"`
	EnrollmentToken string `json:"enrollment_token"`
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Arch            string `json:"arch"`
}

// EnrollResponse is returned by the server after successful enrollment.
type EnrollResponse struct {
	HostUUID     string `json:"host_uuid"`
	CertPEM      string `json:"cert_pem"`
	CACertPEM    string `json:"ca_cert_pem"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ScanCron     string `json:"scan_cron"` // e.g. "0 2 * * *"
}

// Anomaly is pushed from the server to the frontend via WebSocket.
type Anomaly struct {
	HostUUID   string      `json:"host_uuid"`
	RuleID     string      `json:"rule_id"`
	Severity   string      `json:"severity"` // CRITICAL | HIGH | MEDIUM | LOW
	Entity     interface{} `json:"entity"`
	DetectedAt time.Time   `json:"detected_at"`
}
