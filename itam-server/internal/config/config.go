package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Postgres   PostgresConfig   `yaml:"postgres"`
	Graph      GraphConfig      `yaml:"graph"`
	Kafka      KafkaConfig      `yaml:"kafka"`
	Redis      RedisConfig      `yaml:"redis"`
	Enrollment EnrollmentConfig `yaml:"enrollment"`
	JWT        JWTConfig        `yaml:"jwt"`
	Admin      AdminConfig      `yaml:"admin"`
	DataDir    string           `yaml:"data_dir"`
}

type ServerConfig struct {
	Addr            string `yaml:"addr"`            // :8443
	TLSCertPath     string `yaml:"tls_cert_path"`
	TLSKeyPath      string `yaml:"tls_key_path"`
	ReadTimeoutSec  int    `yaml:"read_timeout_sec"`
	WriteTimeoutSec int    `yaml:"write_timeout_sec"`
}

type PostgresConfig struct {
	DSN          string `yaml:"dsn"`           // postgres://user:pass@host:5432/itam
	MaxConns     int    `yaml:"max_conns"`     // 50
	MinConns     int    `yaml:"min_conns"`     // 5
	MigrationsDir string `yaml:"migrations_dir"`
}

type GraphConfig struct {
	URI      string `yaml:"uri"`      // bolt://memgraph:7687 or neo4j://neo4j:7687
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type KafkaConfig struct {
	Brokers        []string `yaml:"brokers"`         // ["kafka:9092"]
	TopicScanFull  string   `yaml:"topic_scan_full"`
	TopicScanDiff  string   `yaml:"topic_scan_diff"`
	TopicHeartbeat string   `yaml:"topic_heartbeat"`
	TopicAlert     string   `yaml:"topic_alert"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`     // redis:6379
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type EnrollmentConfig struct {
	HMACSecret  string `yaml:"hmac_secret"`  // 32+ random bytes, base64-encoded
	CACertPath  string `yaml:"ca_cert_path"` // path to CA cert PEM
	CAKeyPath   string `yaml:"ca_key_path"`  // path to CA key PEM
	CertTTLDays int    `yaml:"cert_ttl_days"` // default: 365
}

type JWTConfig struct {
	Secret         string `yaml:"secret"`           // 32+ random bytes
	AccessTTLMin   int    `yaml:"access_ttl_min"`   // 15
	RefreshTTLHour int    `yaml:"refresh_ttl_hour"` // 24
}

type AdminConfig struct {
	Token string `yaml:"token"` // static bearer for /v1/admin/* endpoints
}

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

func applyDefaults(cfg *Config) {
	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8443"
	}
	if cfg.Server.ReadTimeoutSec == 0 {
		cfg.Server.ReadTimeoutSec = 30
	}
	if cfg.Server.WriteTimeoutSec == 0 {
		cfg.Server.WriteTimeoutSec = 30
	}
	if cfg.Postgres.MaxConns == 0 {
		cfg.Postgres.MaxConns = 50
	}
	if cfg.Postgres.MinConns == 0 {
		cfg.Postgres.MinConns = 5
	}
	if cfg.Postgres.MigrationsDir == "" {
		cfg.Postgres.MigrationsDir = "migrations"
	}
	if cfg.Kafka.TopicScanFull == "" {
		cfg.Kafka.TopicScanFull = "agent.scan.full"
	}
	if cfg.Kafka.TopicScanDiff == "" {
		cfg.Kafka.TopicScanDiff = "agent.scan.diff"
	}
	if cfg.Kafka.TopicHeartbeat == "" {
		cfg.Kafka.TopicHeartbeat = "agent.heartbeat"
	}
	if cfg.Kafka.TopicAlert == "" {
		cfg.Kafka.TopicAlert = "agent.alert"
	}
	if cfg.Enrollment.CertTTLDays == 0 {
		cfg.Enrollment.CertTTLDays = 365
	}
	if cfg.JWT.AccessTTLMin == 0 {
		cfg.JWT.AccessTTLMin = 15
	}
	if cfg.JWT.RefreshTTLHour == 0 {
		cfg.JWT.RefreshTTLHour = 24
	}
	if cfg.DataDir == "" {
		cfg.DataDir = "/var/lib/itam-server"
	}
}
