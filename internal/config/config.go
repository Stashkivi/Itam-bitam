package config

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Config holds the full agent runtime configuration. It is loaded from a
// YAML file on disk and partially overwritten by the EnrollResponse (scan cron,
// server-pushed settings) after enrollment.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Agent    AgentConfig    `yaml:"agent"`
	Cache    CacheConfig    `yaml:"cache"`
	SIEM     SIEMConfig     `yaml:"siem"`
	Enrolled EnrolledState  `yaml:"enrolled"`
}

type ServerConfig struct {
	BaseURL         string `yaml:"base_url"`          // https://itam.corp.example.com
	EnrollmentToken string `yaml:"enrollment_token"`  // pre-shared, consumed on first run
	TLSSkipVerify   bool   `yaml:"tls_skip_verify"`   // dev-only
	CACertPath      string `yaml:"ca_cert_path"`      // path to bundled server CA
}

type AgentConfig struct {
	DataDir    string `yaml:"data_dir"`    // writable directory for certs and state
	ScanCron   string `yaml:"scan_cron"`   // default: "0 2 * * *", overridden by server
	LogLevel   string `yaml:"log_level"`   // debug | info | warn | error
	GOGC       int    `yaml:"gogc"`        // GC percentage; 50 is recommended
	MemLimitMB int    `yaml:"mem_limit_mb"` // maps to GOMEMLIMIT
}

type CacheConfig struct {
	MaxPendingRows int `yaml:"max_pending_rows"` // evict oldest when limit reached
	FlushBatchSize int `yaml:"flush_batch_size"` // rows per flush attempt
	MaxAttempts    int `yaml:"max_attempts"`     // before moving to dead_letter
}

type SIEMConfig struct {
	WazuhEnabled    bool   `yaml:"wazuh_enabled"`
	WazuhAlertsPath string `yaml:"wazuh_alerts_path"` // default: /var/ossec/logs/alerts/alerts.json
	ElasticEnabled  bool   `yaml:"elastic_enabled"`
	ElasticAPIAddr  string `yaml:"elastic_api_addr"` // default: https://localhost:8220
}

// EnrolledState is persisted after the enrollment handshake completes.
type EnrolledState struct {
	HostUUID     string `yaml:"host_uuid"`
	CertPath     string `yaml:"cert_path"`
	KeyPath      string `yaml:"key_path"`
	CACertPath   string `yaml:"ca_cert_path"`
	AccessToken  string `yaml:"access_token"`
	RefreshToken string `yaml:"refresh_token"`
	ScanCron     string `yaml:"scan_cron"`
}

// DataDir returns the platform-appropriate default if none is set.
func DataDir(cfg *Config) string {
	if cfg.Agent.DataDir != "" {
		return cfg.Agent.DataDir
	}
	if runtime.GOOS == "windows" {
		base := os.Getenv("PROGRAMDATA")
		if base == "" {
			base = `C:\ProgramData`
		}
		return filepath.Join(base, "ITAMAgent")
	}
	return "/etc/itam-agent"
}

// Load reads the YAML config file at path. Missing optional fields retain zero values.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	applyDefaults(&cfg)
	return &cfg, nil
}

// Save writes the current config back to path (used to persist EnrolledState).
func Save(cfg *Config, path string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func applyDefaults(cfg *Config) {
	if cfg.Agent.ScanCron == "" {
		cfg.Agent.ScanCron = "0 2 * * *"
	}
	if cfg.Agent.LogLevel == "" {
		cfg.Agent.LogLevel = "info"
	}
	if cfg.Agent.GOGC == 0 {
		cfg.Agent.GOGC = 50
	}
	if cfg.Agent.MemLimitMB == 0 {
		cfg.Agent.MemLimitMB = 15
	}
	if cfg.Cache.MaxPendingRows == 0 {
		cfg.Cache.MaxPendingRows = 10000
	}
	if cfg.Cache.FlushBatchSize == 0 {
		cfg.Cache.FlushBatchSize = 50
	}
	if cfg.Cache.MaxAttempts == 0 {
		cfg.Cache.MaxAttempts = 5
	}
	if cfg.SIEM.WazuhAlertsPath == "" {
		cfg.SIEM.WazuhAlertsPath = "/var/ossec/logs/alerts/alerts.json"
	}
	if cfg.SIEM.ElasticAPIAddr == "" {
		cfg.SIEM.ElasticAPIAddr = "https://localhost:8220"
	}
}

// DeriveHostUUID computes a stable UUID from machine-identity sources.
// The exact inputs are platform-specific (see hostid_*.go files).
func DeriveHostUUID(machineID, primaryMAC, hostname string) string {
	h := sha256.New()
	h.Write([]byte(machineID))
	h.Write([]byte("|"))
	h.Write([]byte(primaryMAC))
	h.Write([]byte("|"))
	h.Write([]byte(hostname))
	sum := h.Sum(nil)
	// Format as a UUID v4-like string (version bits are overwritten but
	// determinism matters more than strict RFC 4122 compliance here).
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}
