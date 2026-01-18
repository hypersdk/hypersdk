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
