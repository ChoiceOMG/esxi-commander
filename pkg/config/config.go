package config

import (
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ESXi     ESXiConfig     `yaml:"esxi"`
	Defaults DefaultsConfig `yaml:"defaults"`
	Security SecurityConfig `yaml:"security"`
	Backup   BackupConfig   `yaml:"backup"`
	Metrics  MetricsConfig  `yaml:"metrics"`
}

type ESXiConfig struct {
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Insecure bool   `yaml:"insecure"`
	SSHKey   string `yaml:"ssh_key"`
}

type DefaultsConfig struct {
	Template  string `yaml:"template"`
	Datastore string `yaml:"datastore"`
	Network   string `yaml:"network"`
	CPU       int    `yaml:"cpu"`
	RAM       int    `yaml:"ram"`
	Disk      int    `yaml:"disk"`
}

type SecurityConfig struct {
	Mode       string   `yaml:"mode"` // restricted, standard, unrestricted
	AuditLog   string   `yaml:"audit_log"`
	IPAllowlist []string `yaml:"ip_allowlist"`
}

type BackupConfig struct {
	CatalogPath   string           `yaml:"catalog_path"`
	DefaultTarget string           `yaml:"default_target"`
	Compression   string           `yaml:"compression"`
	Retention     RetentionConfig  `yaml:"retention"`
}

type RetentionConfig struct {
	KeepLast    int `yaml:"keep_last"`
	KeepDaily   int `yaml:"keep_daily"`
	KeepWeekly  int `yaml:"keep_weekly"`
	KeepMonthly int `yaml:"keep_monthly"`
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Path    string `yaml:"path"`
}

// Load loads configuration from file
func Load(path string) (*Config, error) {
	if path == "" {
		// Try default locations
		locations := []string{
			"config.yaml",
			"/etc/ceso/config.yaml",
			os.ExpandEnv("$HOME/.ceso/config.yaml"),
		}
		
		for _, loc := range locations {
			if _, err := os.Stat(loc); err == nil {
				path = loc
				break
			}
		}
		
		if path == "" {
			return nil, fmt.Errorf("no configuration file found")
		}
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults
	if config.Backup.CatalogPath == "" {
		config.Backup.CatalogPath = "/var/lib/ceso/backup.db"
	}
	if config.Backup.DefaultTarget == "" {
		config.Backup.DefaultTarget = "datastore"
	}
	if config.Backup.Compression == "" {
		config.Backup.Compression = "gzip"
	}
	if config.Metrics.Port == 0 {
		config.Metrics.Port = 9090
	}
	if config.Metrics.Path == "" {
		config.Metrics.Path = "/metrics"
	}

	return &config, nil
}
