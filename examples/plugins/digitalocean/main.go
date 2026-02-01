// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"time"

	"hypersdk/logger"
	"hypersdk/providers"
	"hypersdk/providers/plugin"
)

// Plugin metadata - MUST be exported
var PluginInfo = plugin.Metadata{
	Name:         "digitalocean",
	Version:      "1.0.0",
	Description:  "DigitalOcean Droplet provider for HyperSDK",
	Author:       "HyperSDK Community",
	License:      "LGPL-3.0-or-later",
	ProviderType: providers.ProviderType("digitalocean"),
	Capabilities: providers.ExportCapabilities{
		SupportedFormats:    []string{"raw", "qcow2"},
		SupportsCompression: true,
		SupportsStreaming:   true,
		SupportsSnapshots:   true,
		MaxVMSizeGB:         0, // unlimited
		SupportedTargets:    []string{"local", "spaces"},
	},
	MinSDKVersion: "1.0.0",
	Dependencies:  []string{},
	BuildTime:     time.Now(),
	GoVersion:     "1.24",
}

// NewProvider creates a new DigitalOcean provider instance - MUST be exported
func NewProvider(config providers.ProviderConfig, log logger.Logger) (providers.Provider, error) {
	return &DigitalOceanProvider{
		config: config,
		logger: log,
	}, nil
}
