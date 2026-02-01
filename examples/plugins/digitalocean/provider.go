// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"fmt"

	"hypersdk/logger"
	"hypersdk/providers"
)

// DigitalOceanProvider implements the Provider interface for DigitalOcean
type DigitalOceanProvider struct {
	config   providers.ProviderConfig
	logger   logger.Logger
	token    string
	endpoint string
}

// Name returns the provider name
func (p *DigitalOceanProvider) Name() string {
	return "DigitalOcean"
}

// Type returns the provider type
func (p *DigitalOceanProvider) Type() providers.ProviderType {
	return providers.ProviderType("digitalocean")
}

// Connect establishes connection to DigitalOcean API
func (p *DigitalOceanProvider) Connect(ctx context.Context, config providers.ProviderConfig) error {
	p.logger.Info("connecting to DigitalOcean API")

	// Extract token from metadata
	if token, ok := config.Metadata["token"].(string); ok {
		p.token = token
	} else {
		return fmt.Errorf("missing required field: token")
	}

	// Set endpoint (default to official API)
	if endpoint, ok := config.Metadata["endpoint"].(string); ok {
		p.endpoint = endpoint
	} else {
		p.endpoint = "https://api.digitalocean.com/v2"
	}

	p.config = config

	// Validate credentials
	if err := p.ValidateCredentials(ctx); err != nil {
		return fmt.Errorf("credential validation failed: %w", err)
	}

	p.logger.Info("connected to DigitalOcean API successfully")

	return nil
}

// Disconnect closes the connection
func (p *DigitalOceanProvider) Disconnect() error {
	p.logger.Info("disconnecting from DigitalOcean API")
	p.token = ""
	return nil
}

// ValidateCredentials validates API credentials
func (p *DigitalOceanProvider) ValidateCredentials(ctx context.Context) error {
	// In a real implementation, make an API call to validate the token
	// For this example, just check if token is not empty
	if p.token == "" {
		return fmt.Errorf("token is empty")
	}

	p.logger.Debug("credentials validated")

	return nil
}

// ListVMs lists all droplets (VMs)
func (p *DigitalOceanProvider) ListVMs(ctx context.Context, filter providers.VMFilter) ([]*providers.VMInfo, error) {
	p.logger.Info("listing droplets", "filter", filter)

	// In a real implementation:
	// 1. Make API call to list droplets
	// 2. Filter results based on filter criteria
	// 3. Convert to VMInfo format

	// Example response
	vms := []*providers.VMInfo{
		{
			Provider:    p.Type(),
			ID:          "123456",
			Name:        "web-server-01",
			State:       "running",
			Location:    "nyc3",
			PowerState:  "running",
			GuestOS:     "Ubuntu 22.04 x64",
			MemoryMB:    2048,
			NumCPUs:     2,
			StorageGB:   50,
			IPAddresses: []string{"192.0.2.100"},
			Tags:        map[string]string{"env": "production"},
		},
	}

	p.logger.Info("listed droplets", "count", len(vms))

	return vms, nil
}

// GetVM retrieves information about a specific VM
func (p *DigitalOceanProvider) GetVM(ctx context.Context, identifier string) (*providers.VMInfo, error) {
	p.logger.Info("getting droplet", "id", identifier)

	// In a real implementation:
	// 1. Make API call to get droplet by ID
	// 2. Convert to VMInfo format

	vm := &providers.VMInfo{
		Provider:    p.Type(),
		ID:          identifier,
		Name:        "web-server-01",
		State:       "running",
		Location:    "nyc3",
		PowerState:  "running",
		GuestOS:     "Ubuntu 22.04 x64",
		MemoryMB:    2048,
		NumCPUs:     2,
		StorageGB:   50,
		IPAddresses: []string{"192.0.2.100"},
	}

	return vm, nil
}

// SearchVMs searches for VMs matching a query
func (p *DigitalOceanProvider) SearchVMs(ctx context.Context, query string) ([]*providers.VMInfo, error) {
	p.logger.Info("searching droplets", "query", query)

	// Use ListVMs with a filter
	filter := providers.VMFilter{
		NamePattern: query,
	}

	return p.ListVMs(ctx, filter)
}

// ExportVM exports a VM
func (p *DigitalOceanProvider) ExportVM(ctx context.Context, identifier string, opts providers.ExportOptions) (*providers.ExportResult, error) {
	p.logger.Info("exporting droplet", "id", identifier, "format", opts.Format)

	// In a real implementation:
	// 1. Create snapshot of droplet
	// 2. Download snapshot image
	// 3. Convert to requested format
	// 4. Save to output path

	result := &providers.ExportResult{
		Provider:   p.Type(),
		VMName:     "web-server-01",
		VMID:       identifier,
		Format:     opts.Format,
		OutputPath: opts.OutputPath,
		Files: []string{
			opts.OutputPath + "/disk.qcow2",
		},
		Size:     5368709120, // 5 GB
		Checksum: "abc123def456",
		Metadata: map[string]interface{}{
			"snapshot_id": "snapshot-12345",
		},
	}

	p.logger.Info("droplet exported successfully", "size_gb", result.Size/1024/1024/1024)

	return result, nil
}

// GetExportCapabilities returns export capabilities
func (p *DigitalOceanProvider) GetExportCapabilities() providers.ExportCapabilities {
	return PluginInfo.Capabilities
}
