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
