package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type RateLimitConfig struct {
	MaxBurst   int `yaml:"max_burst"`
	GeneralRPM int `yaml:"general_rpm"`
}

type DefaultsConfig struct {
	Timeout          int  `yaml:"timeout"`
	Snapshot         bool `yaml:"snapshot"`
	PollIntervalMs   int  `yaml:"poll_interval_ms"`
	PollMaxRetries   int  `yaml:"poll_max_retries"`
}

type DeployConfig struct {
	AutoGenerate bool     `yaml:"auto_generate"`
	AutoRun      bool     `yaml:"auto_run"`
	Environment  string   `yaml:"environment"`
	Workflows    []string `yaml:"workflows"`
}

type NotificationsConfig struct {
	SlackWebhook string `yaml:"slack_webhook,omitempty"`
}

type KfsConfig struct {
	Project       string              `yaml:"project"`
	APIKey        string              `yaml:"api_key"`
	BaseURL       string              `yaml:"base_url"`
	PhoneNumber   string              `yaml:"phone_number"`
	RateLimit     RateLimitConfig     `yaml:"rate_limit"`
	Defaults      DefaultsConfig      `yaml:"defaults"`
	SpecsDir      string              `yaml:"specs_dir"`
	SnapshotsDir  string              `yaml:"snapshots_dir"`
	ReportsDir    string              `yaml:"reports_dir"`
	Deploy        DeployConfig        `yaml:"deploy"`
	Notifications NotificationsConfig `yaml:"notifications,omitempty"`
}

func LoadConfig(path string) (*KfsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand env vars in the yaml data (e.g. ${KAPSO_API_KEY})
	expandedData := os.ExpandEnv(string(data))

	var cfg KfsConfig
	if err := yaml.Unmarshal([]byte(expandedData), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults if missing
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.kapso.ai/platform/v1"
	}
	if cfg.SpecsDir == "" {
		cfg.SpecsDir = "kfs-specs"
	}
	if cfg.SnapshotsDir == "" {
		cfg.SnapshotsDir = "kfs-snapshots"
	}
	if cfg.ReportsDir == "" {
		cfg.ReportsDir = "kfs-reports"
	}

	return &cfg, nil
}
