// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	VCenterURL      string
	Username        string
	Password        string
	Insecure        bool
	Timeout         time.Duration
	DownloadWorkers int
	ChunkSize       int64
	RetryAttempts   int
	RetryDelay      time.Duration
	LogLevel        string
	ProgressStyle   string // "bar", "spinner", "quiet"
	ShowETA         bool
	RefreshRate     time.Duration
	DaemonAddr      string // Daemon API server address

	// Connection pool configuration (Phase 1.1)
	ConnectionPool *ConnectionPoolConfig

	// Webhook configuration (Phase 1.2)
	Webhooks []WebhookConfig `yaml:"webhooks"`

	// Database path for persistence (Phase 2.3)
	DatabasePath string `yaml:"database_path"`

	// AWS configuration (Phase 4.1)
	AWS *AWSConfig `yaml:"aws"`

	// Azure configuration (Phase 4.2)
	Azure *AzureConfig `yaml:"azure"`

	// GCP configuration (Phase 4.3)
	GCP *GCPConfig `yaml:"gcp"`

	// Hyper-V configuration (Phase 4.4)
	HyperV *HyperVConfig `yaml:"hyperv"`

	// OCI configuration
	OCI *OCIConfig `yaml:"oci"`
}

// AWSConfig holds AWS-specific settings
type AWSConfig struct {
	Region       string `yaml:"region"`
	AccessKey    string `yaml:"access_key"`
	SecretKey    string `yaml:"secret_key"`
	S3Bucket     string `yaml:"s3_bucket"`
	ExportFormat string `yaml:"export_format"` // vmdk, vhd, raw
	Enabled      bool   `yaml:"enabled"`
}

// AzureConfig holds Azure-specific settings
type AzureConfig struct {
	SubscriptionID string `yaml:"subscription_id"`
	TenantID       string `yaml:"tenant_id"`
	ClientID       string `yaml:"client_id"`
	ClientSecret   string `yaml:"client_secret"`
	ResourceGroup  string `yaml:"resource_group"`
	Location       string `yaml:"location"`
	StorageAccount string `yaml:"storage_account"`
	Container      string `yaml:"container"`
	ContainerURL   string `yaml:"container_url"`
	ExportFormat   string `yaml:"export_format"` // vhd, image
	Enabled        bool   `yaml:"enabled"`
}

// GCPConfig holds GCP-specific settings
type GCPConfig struct {
	ProjectID       string `yaml:"project_id"`
	Zone            string `yaml:"zone"`
	Region          string `yaml:"region"`
	CredentialsJSON string `yaml:"credentials_json"` // Path to service account JSON
	GCSBucket       string `yaml:"gcs_bucket"`
	ExportFormat    string `yaml:"export_format"` // vmdk, image
	Enabled         bool   `yaml:"enabled"`
}

// HyperVConfig holds Hyper-V-specific settings
type HyperVConfig struct {
	Host         string `yaml:"host"`          // Hyper-V host (empty for local)
	Username     string `yaml:"username"`      // Windows username
	Password     string `yaml:"password"`      // Windows password
	UseWinRM     bool   `yaml:"use_winrm"`     // Use WinRM for remote
	WinRMPort    int    `yaml:"winrm_port"`    // WinRM port (5985/5986)
	UseHTTPS     bool   `yaml:"use_https"`     // Use HTTPS for WinRM
	ExportFormat string `yaml:"export_format"` // vhdx, vhd, hyperv
	Enabled      bool   `yaml:"enabled"`
}

// OCIConfig holds Oracle Cloud Infrastructure settings
type OCIConfig struct {
	TenancyOCID     string `yaml:"tenancy_ocid"`      // OCI tenancy OCID
	UserOCID        string `yaml:"user_ocid"`         // OCI user OCID
	Fingerprint     string `yaml:"fingerprint"`       // API key fingerprint
	PrivateKeyPath  string `yaml:"private_key_path"`  // Path to private key file
	Region          string `yaml:"region"`            // OCI region (e.g., us-phoenix-1)
	CompartmentOCID string `yaml:"compartment_ocid"`  // Compartment OCID for resources
	Bucket          string `yaml:"bucket"`            // Object Storage bucket name
	Namespace       string `yaml:"namespace"`         // Object Storage namespace
	ExportFormat    string `yaml:"export_format"`     // qcow2, vmdk
	Enabled         bool   `yaml:"enabled"`
}

// ConnectionPoolConfig holds connection pool settings
type ConnectionPoolConfig struct {
	Enabled             bool          `yaml:"enabled"`
	MaxConnections      int           `yaml:"max_connections"`
	IdleTimeout         time.Duration `yaml:"idle_timeout"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
}

// WebhookConfig holds webhook endpoint configuration
type WebhookConfig struct {
	URL     string            `yaml:"url" json:"url"`
	Events  []string          `yaml:"events" json:"events"`
	Headers map[string]string `yaml:"headers" json:"headers"`
	Timeout time.Duration     `yaml:"timeout" json:"timeout"`
	Retry   int               `yaml:"retry" json:"retry"`
	Enabled bool              `yaml:"enabled" json:"enabled"`
}

func FromEnvironment() *Config {
	timeout, _ := strconv.Atoi(getEnv("GOVC_TIMEOUT", "3600"))
	workers, _ := strconv.Atoi(getEnv("DOWNLOAD_WORKERS", "3"))
	retry, _ := strconv.Atoi(getEnv("RETRY_ATTEMPTS", "3"))
	chunkSize, _ := strconv.ParseInt(getEnv("CHUNK_SIZE", "33554432"), 10, 64) // 32MB
	refreshRate, _ := strconv.Atoi(getEnv("PROGRESS_REFRESH_RATE", "100"))

	return &Config{
		VCenterURL:      os.Getenv("GOVC_URL"),
		Username:        os.Getenv("GOVC_USERNAME"),
		Password:        os.Getenv("GOVC_PASSWORD"),
		Insecure:        getEnv("GOVC_INSECURE", "0") == "1",
		Timeout:         time.Duration(timeout) * time.Second,
		DownloadWorkers: workers,
		ChunkSize:       chunkSize,
		RetryAttempts:   retry,
		RetryDelay:      5 * time.Second,
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		ProgressStyle:   getEnv("PROGRESS_STYLE", "bar"),
		ShowETA:         getEnv("SHOW_ETA", "1") == "1",
		RefreshRate:     time.Duration(refreshRate) * time.Millisecond,
		DaemonAddr:      getEnv("DAEMON_ADDR", "localhost:8080"),
	}
}

// FromFile loads configuration from a YAML file
func FromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	// Apply defaults
	if cfg.Timeout == 0 {
		cfg.Timeout = 3600 * time.Second
	}
	if cfg.DownloadWorkers == 0 {
		cfg.DownloadWorkers = 3
	}
	if cfg.ChunkSize == 0 {
		cfg.ChunkSize = 33554432 // 32MB
	}
	if cfg.RetryAttempts == 0 {
		cfg.RetryAttempts = 3
	}
	if cfg.RetryDelay == 0 {
		cfg.RetryDelay = 5 * time.Second
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.ProgressStyle == "" {
		cfg.ProgressStyle = "bar"
	}
	if cfg.RefreshRate == 0 {
		cfg.RefreshRate = 100 * time.Millisecond
	}
	if cfg.DaemonAddr == "" {
		cfg.DaemonAddr = "localhost:8080"
	}

	// Connection pool defaults
	if cfg.ConnectionPool == nil {
		cfg.ConnectionPool = &ConnectionPoolConfig{
			Enabled:             false, // Disabled by default
			MaxConnections:      5,
			IdleTimeout:         5 * time.Minute,
			HealthCheckInterval: 30 * time.Second,
		}
	} else {
		if cfg.ConnectionPool.MaxConnections == 0 {
			cfg.ConnectionPool.MaxConnections = 5
		}
		if cfg.ConnectionPool.IdleTimeout == 0 {
			cfg.ConnectionPool.IdleTimeout = 5 * time.Minute
		}
		if cfg.ConnectionPool.HealthCheckInterval == 0 {
			cfg.ConnectionPool.HealthCheckInterval = 30 * time.Second
		}
	}

	// Database path default
	if cfg.DatabasePath == "" {
		cfg.DatabasePath = "./hypersdk.db"
	}

	// Webhook defaults
	for i := range cfg.Webhooks {
		if cfg.Webhooks[i].Timeout == 0 {
			cfg.Webhooks[i].Timeout = 10 * time.Second
		}
		if cfg.Webhooks[i].Retry == 0 {
			cfg.Webhooks[i].Retry = 3
		}
	}

	// AWS defaults
	if cfg.AWS == nil {
		cfg.AWS = &AWSConfig{
			Region:       "us-east-1",
			ExportFormat: "vmdk",
			Enabled:      false, // Disabled by default
		}
	} else {
		if cfg.AWS.Region == "" {
			cfg.AWS.Region = "us-east-1"
		}
		if cfg.AWS.ExportFormat == "" {
			cfg.AWS.ExportFormat = "vmdk"
		}
	}

	// Azure defaults
	if cfg.Azure == nil {
		cfg.Azure = &AzureConfig{
			Location:     "eastus",
			ExportFormat: "image",
			Enabled:      false, // Disabled by default
		}
	} else {
		if cfg.Azure.Location == "" {
			cfg.Azure.Location = "eastus"
		}
		if cfg.Azure.ExportFormat == "" {
			cfg.Azure.ExportFormat = "image"
		}
	}

	// GCP defaults
	if cfg.GCP == nil {
		cfg.GCP = &GCPConfig{
			Zone:         "us-central1-a",
			Region:       "us-central1",
			ExportFormat: "vmdk",
			Enabled:      false, // Disabled by default
		}
	} else {
		if cfg.GCP.Zone == "" {
			cfg.GCP.Zone = "us-central1-a"
		}
		if cfg.GCP.Region == "" {
			cfg.GCP.Region = "us-central1"
		}
		if cfg.GCP.ExportFormat == "" {
			cfg.GCP.ExportFormat = "vmdk"
		}
	}

	// Hyper-V defaults
	if cfg.HyperV == nil {
		cfg.HyperV = &HyperVConfig{
			UseWinRM:     false, // Local by default
			WinRMPort:    5985,
			UseHTTPS:     false,
			ExportFormat: "vhdx",
			Enabled:      false, // Disabled by default
		}
	} else {
		if cfg.HyperV.WinRMPort == 0 {
			if cfg.HyperV.UseHTTPS {
				cfg.HyperV.WinRMPort = 5986
			} else {
				cfg.HyperV.WinRMPort = 5985
			}
		}
		if cfg.HyperV.ExportFormat == "" {
			cfg.HyperV.ExportFormat = "vhdx"
		}
	}

	// OCI defaults
	if cfg.OCI == nil {
		cfg.OCI = &OCIConfig{
			Region:       "us-phoenix-1",
			ExportFormat: "qcow2",
			Enabled:      false, // Disabled by default
		}
	} else {
		if cfg.OCI.Region == "" {
			cfg.OCI.Region = "us-phoenix-1"
		}
		if cfg.OCI.ExportFormat == "" {
			cfg.OCI.ExportFormat = "qcow2"
		}
	}

	return cfg, nil
}

// MergeWithEnv merges file config with environment variables (env takes precedence)
func (c *Config) MergeWithEnv() *Config {
	envCfg := FromEnvironment()

	// Override with environment variables if set
	if envCfg.VCenterURL != "" {
		c.VCenterURL = envCfg.VCenterURL
	}
	if envCfg.Username != "" {
		c.Username = envCfg.Username
	}
	if envCfg.Password != "" {
		c.Password = envCfg.Password
	}
	if os.Getenv("GOVC_INSECURE") != "" {
		c.Insecure = envCfg.Insecure
	}
	if os.Getenv("DOWNLOAD_WORKERS") != "" {
		c.DownloadWorkers = envCfg.DownloadWorkers
	}
	if os.Getenv("LOG_LEVEL") != "" {
		c.LogLevel = envCfg.LogLevel
	}
	if os.Getenv("DAEMON_ADDR") != "" {
		c.DaemonAddr = envCfg.DaemonAddr
	}

	return c
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
