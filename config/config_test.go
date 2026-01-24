// SPDX-License-Identifier: LGPL-3.0-or-later

package config

import (
	"os"
	"testing"
)

func TestFromEnvironment(t *testing.T) {
	// Set test environment variables
	os.Setenv("GOVC_URL", "https://test.vcenter.com/sdk")
	os.Setenv("GOVC_USERNAME", "testuser")
	os.Setenv("GOVC_PASSWORD", "testpass")
	os.Setenv("GOVC_INSECURE", "1")
	os.Setenv("DAEMON_ADDR", "localhost:9090")
	os.Setenv("LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("GOVC_URL")
		os.Unsetenv("GOVC_USERNAME")
		os.Unsetenv("GOVC_PASSWORD")
		os.Unsetenv("GOVC_INSECURE")
		os.Unsetenv("DAEMON_ADDR")
		os.Unsetenv("LOG_LEVEL")
	}()

	cfg := FromEnvironment()

	if cfg.VCenterURL != "https://test.vcenter.com/sdk" {
		t.Errorf("Expected VCenterURL 'https://test.vcenter.com/sdk', got '%s'", cfg.VCenterURL)
	}
	if cfg.Username != "testuser" {
		t.Errorf("Expected Username 'testuser', got '%s'", cfg.Username)
	}
	if cfg.Password != "testpass" {
		t.Errorf("Expected Password 'testpass', got '%s'", cfg.Password)
	}
	if !cfg.Insecure {
		t.Error("Expected Insecure to be true")
	}
	if cfg.DaemonAddr != "localhost:9090" {
		t.Errorf("Expected DaemonAddr 'localhost:9090', got '%s'", cfg.DaemonAddr)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("Expected LogLevel 'debug', got '%s'", cfg.LogLevel)
	}
}

func TestFromEnvironmentDefaults(t *testing.T) {
	// Clear all env vars
	os.Clearenv()

	cfg := FromEnvironment()

	if cfg.DownloadWorkers != 3 {
		t.Errorf("Expected default DownloadWorkers 3, got %d", cfg.DownloadWorkers)
	}
	if cfg.RetryAttempts != 3 {
		t.Errorf("Expected default RetryAttempts 3, got %d", cfg.RetryAttempts)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("Expected default LogLevel 'info', got '%s'", cfg.LogLevel)
	}
	if cfg.DaemonAddr != "localhost:8080" {
		t.Errorf("Expected default DaemonAddr 'localhost:8080', got '%s'", cfg.DaemonAddr)
	}
}

func TestFromFile(t *testing.T) {
	// Create temporary config file
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `vcenterurl: "https://file.vcenter.com/sdk"
username: "fileuser"
password: "filepass"
insecure: true
daemonaddr: "0.0.0.0:8888"
loglevel: "warn"
downloadworkers: 8
`
	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	tmpFile.Close()

	cfg, err := FromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("FromFile failed: %v", err)
	}

	if cfg.VCenterURL != "https://file.vcenter.com/sdk" {
		t.Errorf("Expected VCenterURL from file, got '%s'", cfg.VCenterURL)
	}
	if cfg.Username != "fileuser" {
		t.Errorf("Expected Username 'fileuser', got '%s'", cfg.Username)
	}
	if cfg.DaemonAddr != "0.0.0.0:8888" {
		t.Errorf("Expected DaemonAddr '0.0.0.0:8888', got '%s'", cfg.DaemonAddr)
	}
	if cfg.DownloadWorkers != 8 {
		t.Errorf("Expected DownloadWorkers 8, got %d", cfg.DownloadWorkers)
	}
}

func TestMergeWithEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("GOVC_URL", "https://env.vcenter.com/sdk")
	os.Setenv("LOG_LEVEL", "error")
	defer func() {
		os.Unsetenv("GOVC_URL")
		os.Unsetenv("LOG_LEVEL")
	}()

	// Create base config
	cfg := &Config{
		VCenterURL: "https://file.vcenter.com/sdk",
		LogLevel:   "info",
		DaemonAddr: "localhost:8080",
	}

	// Merge with env (env should take precedence)
	merged := cfg.MergeWithEnv()

	if merged.VCenterURL != "https://env.vcenter.com/sdk" {
		t.Errorf("Expected env to override VCenterURL, got '%s'", merged.VCenterURL)
	}
	if merged.LogLevel != "error" {
		t.Errorf("Expected env to override LogLevel, got '%s'", merged.LogLevel)
	}
	if merged.DaemonAddr != "localhost:8080" {
		t.Errorf("Expected DaemonAddr to remain from file, got '%s'", merged.DaemonAddr)
	}
}

func TestFromFile_NonexistentFile(t *testing.T) {
	_, err := FromFile("/nonexistent/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestFromFile_InvalidYAML(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "invalid-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write invalid YAML
	tmpFile.WriteString("invalid: yaml: content: :\n")
	tmpFile.Close()

	_, err = FromFile(tmpFile.Name())
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestFromFile_WithDefaults(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "minimal-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Minimal config - should get defaults
	configContent := `vcenterurl: "https://minimal.vcenter.com/sdk"
`
	tmpFile.WriteString(configContent)
	tmpFile.Close()

	cfg, err := FromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("FromFile failed: %v", err)
	}

	// Verify defaults are applied
	if cfg.DownloadWorkers != 3 {
		t.Errorf("Expected default DownloadWorkers 3, got %d", cfg.DownloadWorkers)
	}
	if cfg.RetryAttempts != 3 {
		t.Errorf("Expected default RetryAttempts 3, got %d", cfg.RetryAttempts)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("Expected default LogLevel 'info', got '%s'", cfg.LogLevel)
	}
	if cfg.DaemonAddr != "localhost:8080" {
		t.Errorf("Expected default DaemonAddr 'localhost:8080', got '%s'", cfg.DaemonAddr)
	}
	if cfg.ProgressStyle != "bar" {
		t.Errorf("Expected default ProgressStyle 'bar', got '%s'", cfg.ProgressStyle)
	}
}

func TestFromFile_WithConnectionPool(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "pool-*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	configContent := `vcenterurl: "https://pool.vcenter.com/sdk"
connectionpool:
  enabled: true
  max_connections: 10
`
	tmpFile.WriteString(configContent)
	tmpFile.Close()

	cfg, err := FromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("FromFile failed: %v", err)
	}

	// Verify connection pool config
	if cfg.ConnectionPool == nil {
		t.Fatal("Expected ConnectionPool to be set")
	}
	if !cfg.ConnectionPool.Enabled {
		t.Error("Expected ConnectionPool.Enabled to be true")
	}
	if cfg.ConnectionPool.MaxConnections != 10 {
		t.Errorf("Expected MaxConnections 10, got %d", cfg.ConnectionPool.MaxConnections)
	}
	// Verify defaults are applied for missing fields
	if cfg.ConnectionPool.IdleTimeout == 0 {
		t.Error("Expected default IdleTimeout to be set")
	}
	if cfg.ConnectionPool.HealthCheckInterval == 0 {
		t.Error("Expected default HealthCheckInterval to be set")
	}
}

func TestMergeWithEnv_AllFields(t *testing.T) {
	// Set comprehensive environment variables
	os.Setenv("GOVC_URL", "https://env.vcenter.com/sdk")
	os.Setenv("GOVC_USERNAME", "envuser")
	os.Setenv("GOVC_PASSWORD", "envpass")
	os.Setenv("GOVC_INSECURE", "1")
	os.Setenv("DOWNLOAD_WORKERS", "10")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("DAEMON_ADDR", "0.0.0.0:9999")
	defer func() {
		os.Unsetenv("GOVC_URL")
		os.Unsetenv("GOVC_USERNAME")
		os.Unsetenv("GOVC_PASSWORD")
		os.Unsetenv("GOVC_INSECURE")
		os.Unsetenv("DOWNLOAD_WORKERS")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("DAEMON_ADDR")
	}()

	// Create base config with different values
	cfg := &Config{
		VCenterURL:      "https://file.vcenter.com/sdk",
		Username:        "fileuser",
		Password:        "filepass",
		Insecure:        false,
		DownloadWorkers: 3,
		RetryAttempts:   3,
		LogLevel:        "info",
		DaemonAddr:      "localhost:8080",
	}

	merged := cfg.MergeWithEnv()

	// Verify env values override config values where supported
	if merged.VCenterURL != "https://env.vcenter.com/sdk" {
		t.Errorf("Expected env VCenterURL, got '%s'", merged.VCenterURL)
	}
	if merged.Username != "envuser" {
		t.Errorf("Expected env Username, got '%s'", merged.Username)
	}
	if merged.Password != "envpass" {
		t.Errorf("Expected env Password, got '%s'", merged.Password)
	}
	if !merged.Insecure {
		t.Error("Expected env Insecure to be true")
	}
	if merged.DownloadWorkers != 10 {
		t.Errorf("Expected env DownloadWorkers 10, got %d", merged.DownloadWorkers)
	}
	// RetryAttempts is not merged from env, should remain from config
	if merged.RetryAttempts != 3 {
		t.Errorf("Expected config RetryAttempts 3, got %d", merged.RetryAttempts)
	}
	if merged.LogLevel != "debug" {
		t.Errorf("Expected env LogLevel, got '%s'", merged.LogLevel)
	}
	if merged.DaemonAddr != "0.0.0.0:9999" {
		t.Errorf("Expected env DaemonAddr, got '%s'", merged.DaemonAddr)
	}
}
