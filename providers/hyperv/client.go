// SPDX-License-Identifier: LGPL-3.0-or-later

package hyperv

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"hypersdk/logger"
	"hypersdk/progress"
)

// Config holds Hyper-V provider configuration
type Config struct {
	Host         string        // Hyper-V host (empty for local)
	Username     string        // Windows username for WinRM
	Password     string        // Windows password for WinRM
	UseWinRM     bool          // Use WinRM for remote connections
	WinRMPort    int           // WinRM port (default 5985 for HTTP, 5986 for HTTPS)
	UseHTTPS     bool          // Use HTTPS for WinRM
	Timeout      time.Duration
}

// Client represents a Hyper-V client for VM operations
type Client struct {
	config *Config
	logger logger.Logger
}

// VMInfo represents Hyper-V VM information
type VMInfo struct {
	Name              string            `json:"Name"`
	ID                string            `json:"Id"`
	State             string            `json:"State"`
	CPUUsage          int               `json:"CPUUsage"`
	MemoryAssigned    int64             `json:"MemoryAssigned"`
	MemoryDemand      int64             `json:"MemoryDemand"`
	Uptime            string            `json:"Uptime"`
	Status            string            `json:"Status"`
	Generation        int               `json:"Generation"`
	Version           string            `json:"Version"`
	Path              string            `json:"Path"`
	ConfigurationPath string            `json:"ConfigurationLocation"`
	VHDPath           []string          `json:"-"` // Populated separately
	Notes             string            `json:"Notes"`
	CreationTime      string            `json:"CreationTime"`
}

// NewClient creates a new Hyper-V client
func NewClient(cfg *Config, log logger.Logger) (*Client, error) {
	if cfg.Timeout == 0 {
		cfg.Timeout = 1 * time.Hour
	}

	if cfg.WinRMPort == 0 {
		if cfg.UseHTTPS {
			cfg.WinRMPort = 5986
		} else {
			cfg.WinRMPort = 5985
		}
	}

	client := &Client{
		config: cfg,
		logger: log,
	}

	// Validate connection if remote
	if cfg.UseWinRM {
		if err := client.validateConnection(); err != nil {
			return nil, fmt.Errorf("failed to validate Hyper-V connection: %w", err)
		}
	}

	return client, nil
}

// ListVMs returns a list of all VMs on the Hyper-V host
func (c *Client) ListVMs(ctx context.Context) ([]*VMInfo, error) {
	c.logger.Info("listing Hyper-V VMs")

	// PowerShell script to get VM information
	script := `Get-VM | Select-Object Name, Id, State, CPUUsage, MemoryAssigned, MemoryDemand, Uptime, Status, Generation, Version, Path, ConfigurationLocation, Notes, CreationTime | ConvertTo-Json -Depth 3`

	output, err := c.executePowerShell(ctx, script)
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	// Parse JSON output
	var vms []*VMInfo

	// Handle both single VM (object) and multiple VMs (array)
	output = strings.TrimSpace(output)
	if strings.HasPrefix(output, "[") {
		// Multiple VMs
		if err := json.Unmarshal([]byte(output), &vms); err != nil {
			return nil, fmt.Errorf("failed to parse VM list: %w", err)
		}
	} else if strings.HasPrefix(output, "{") {
		// Single VM
		var vm VMInfo
		if err := json.Unmarshal([]byte(output), &vm); err != nil {
			return nil, fmt.Errorf("failed to parse VM: %w", err)
		}
		vms = []*VMInfo{&vm}
	} else {
		// No VMs or empty output
		return []*VMInfo{}, nil
	}

	// Get VHD paths for each VM
	for _, vm := range vms {
		vhdPaths, err := c.getVMVHDPaths(ctx, vm.Name)
		if err != nil {
			c.logger.Warn("failed to get VHD paths", "vm", vm.Name, "error", err)
		} else {
			vm.VHDPath = vhdPaths
		}
	}

	c.logger.Info("discovered Hyper-V VMs", "count", len(vms))
	return vms, nil
}

// GetVM retrieves information about a specific VM
func (c *Client) GetVM(ctx context.Context, vmName string) (*VMInfo, error) {
	c.logger.Info("getting Hyper-V VM", "vm", vmName)

	script := fmt.Sprintf(`Get-VM -Name '%s' | Select-Object Name, Id, State, CPUUsage, MemoryAssigned, MemoryDemand, Uptime, Status, Generation, Version, Path, ConfigurationLocation, Notes, CreationTime | ConvertTo-Json -Depth 3`, vmName)

	output, err := c.executePowerShell(ctx, script)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM: %w", err)
	}

	var vm VMInfo
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &vm); err != nil {
		return nil, fmt.Errorf("failed to parse VM info: %w", err)
	}

	// Get VHD paths
	vhdPaths, err := c.getVMVHDPaths(ctx, vmName)
	if err != nil {
		c.logger.Warn("failed to get VHD paths", "vm", vmName, "error", err)
	} else {
		vm.VHDPath = vhdPaths
	}

	return &vm, nil
}

// getVMVHDPaths retrieves all VHD paths for a VM
func (c *Client) getVMVHDPaths(ctx context.Context, vmName string) ([]string, error) {
	script := fmt.Sprintf(`Get-VM -Name '%s' | Get-VMHardDiskDrive | Select-Object -ExpandProperty Path`, vmName)

	output, err := c.executePowerShell(ctx, script)
	if err != nil {
		return nil, err
	}

	paths := strings.Split(strings.TrimSpace(output), "\n")
	result := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path != "" {
			result = append(result, path)
		}
	}

	return result, nil
}

// ExportVM exports a Hyper-V VM
func (c *Client) ExportVM(ctx context.Context, vmName, outputPath string, reporter progress.ProgressReporter) error {
	c.logger.Info("starting Hyper-V VM export", "vm", vmName, "output", outputPath)

	if reporter != nil {
		reporter.Describe("Exporting Hyper-V VM")
	}

	// Get VM info
	vmInfo, err := c.GetVM(ctx, vmName)
	if err != nil {
		return fmt.Errorf("get VM info: %w", err)
	}

	// Create output directory
	exportDir := filepath.Join(outputPath, vmName)

	// Export VM using PowerShell
	script := fmt.Sprintf(`Export-VM -Name '%s' -Path '%s' -ErrorAction Stop`, vmName, outputPath)

	c.logger.Info("executing VM export", "vm", vmName)

	output, err := c.executePowerShell(ctx, script)
	if err != nil {
		return fmt.Errorf("export VM failed: %w (output: %s)", err, output)
	}

	c.logger.Info("VM export completed", "vm", vmName, "output", exportDir)

	// Save metadata
	metadataPath := filepath.Join(outputPath, fmt.Sprintf("%s-metadata.json", vmName))
	if err := c.saveMetadata(vmInfo, metadataPath); err != nil {
		c.logger.Warn("failed to save metadata", "error", err)
	}

	if reporter != nil {
		reporter.Describe("Export complete")
		reporter.Update(100)
	}

	return nil
}

// ExportVHD exports VM VHD files to a specified directory
func (c *Client) ExportVHD(ctx context.Context, vmName, outputPath string, reporter progress.ProgressReporter) ([]string, error) {
	c.logger.Info("exporting Hyper-V VM VHDs", "vm", vmName)

	// Get VM VHD paths
	vhdPaths, err := c.getVMVHDPaths(ctx, vmName)
	if err != nil {
		return nil, fmt.Errorf("get VHD paths: %w", err)
	}

	if len(vhdPaths) == 0 {
		return nil, fmt.Errorf("VM has no VHD files")
	}

	exportedPaths := make([]string, 0, len(vhdPaths))

	for i, vhdPath := range vhdPaths {
		if reporter != nil {
			reporter.Describe(fmt.Sprintf("Copying VHD %d/%d", i+1, len(vhdPaths)))
		}

		vhdName := filepath.Base(vhdPath)
		destPath := filepath.Join(outputPath, vhdName)

		// Copy VHD file
		script := fmt.Sprintf(`Copy-Item -Path '%s' -Destination '%s' -Force`, vhdPath, destPath)

		c.logger.Info("copying VHD", "source", vhdPath, "dest", destPath)

		_, err := c.executePowerShell(ctx, script)
		if err != nil {
			c.logger.Error("failed to copy VHD", "vhd", vhdPath, "error", err)
			continue
		}

		exportedPaths = append(exportedPaths, destPath)
		c.logger.Info("VHD copied", "path", destPath)
	}

	if len(exportedPaths) == 0 {
		return nil, fmt.Errorf("failed to export any VHDs")
	}

	if reporter != nil {
		reporter.Update(100)
	}

	return exportedPaths, nil
}

// StartVM starts a Hyper-V VM
func (c *Client) StartVM(ctx context.Context, vmName string) error {
	c.logger.Info("starting VM", "vm", vmName)

	script := fmt.Sprintf(`Start-VM -Name '%s'`, vmName)

	_, err := c.executePowerShell(ctx, script)
	if err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}

	c.logger.Info("VM started", "vm", vmName)
	return nil
}

// StopVM stops a Hyper-V VM
func (c *Client) StopVM(ctx context.Context, vmName string) error {
	c.logger.Info("stopping VM", "vm", vmName)

	script := fmt.Sprintf(`Stop-VM -Name '%s' -Force`, vmName)

	_, err := c.executePowerShell(ctx, script)
	if err != nil {
		return fmt.Errorf("failed to stop VM: %w", err)
	}

	c.logger.Info("VM stopped", "vm", vmName)
	return nil
}

// DeleteVM deletes a Hyper-V VM
func (c *Client) DeleteVM(ctx context.Context, vmName string) error {
	c.logger.Info("deleting VM", "vm", vmName)

	script := fmt.Sprintf(`Remove-VM -Name '%s' -Force`, vmName)

	_, err := c.executePowerShell(ctx, script)
	if err != nil {
		return fmt.Errorf("failed to delete VM: %w", err)
	}

	c.logger.Info("VM deleted", "vm", vmName)
	return nil
}

// executePowerShell executes a PowerShell script
func (c *Client) executePowerShell(ctx context.Context, script string) (string, error) {
	if c.config.UseWinRM {
		return c.executePowerShellWinRM(ctx, script)
	}
	return c.executePowerShellLocal(ctx, script)
}

// executePowerShellLocal executes PowerShell locally
func (c *Client) executePowerShellLocal(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", script)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("PowerShell execution failed: %w", err)
	}

	return string(output), nil
}

// executePowerShellWinRM executes PowerShell via WinRM
func (c *Client) executePowerShellWinRM(ctx context.Context, script string) (string, error) {
	// For WinRM, we use the winrm command-line tool or a Go WinRM library
	// Here's a simple implementation using the winrm CLI tool

	// Escape script for WinRM
	escapedScript := strings.ReplaceAll(script, `"`, `\"`)
	escapedScript = strings.ReplaceAll(escapedScript, `'`, `''`)

	// Use winrm command-line tool
	cmd := exec.CommandContext(ctx, "winrm",
		"-hostname", c.config.Host,
		"-username", c.config.Username,
		"-password", c.config.Password,
		"-port", fmt.Sprintf("%d", c.config.WinRMPort),
		"-https=" + fmt.Sprintf("%t", c.config.UseHTTPS),
		"powershell", escapedScript,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("WinRM PowerShell execution failed: %w", err)
	}

	return string(output), nil
}

// validateConnection validates the connection to Hyper-V host
func (c *Client) validateConnection() error {
	// Try a simple command to verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	script := "Get-Command Get-VM"
	_, err := c.executePowerShell(ctx, script)
	if err != nil {
		return fmt.Errorf("connection validation failed: %w", err)
	}

	c.logger.Info("Hyper-V connection validated")
	return nil
}

// saveMetadata saves VM metadata to a JSON file
func (c *Client) saveMetadata(vmInfo *VMInfo, path string) error {
	metadata := fmt.Sprintf(`{
  "provider": "hyperv",
  "vm_name": "%s",
  "vm_id": "%s",
  "state": "%s",
  "generation": %d,
  "cpu_usage": %d,
  "memory_assigned": %d,
  "path": "%s",
  "vhd_paths": %s,
  "export_time": "%s"
}`,
		vmInfo.Name,
		vmInfo.ID,
		vmInfo.State,
		vmInfo.Generation,
		vmInfo.CPUUsage,
		vmInfo.MemoryAssigned,
		vmInfo.Path,
		toJSONArray(vmInfo.VHDPath),
		time.Now().Format(time.RFC3339),
	)

	return os.WriteFile(path, []byte(metadata), 0644)
}

// toJSONArray converts string slice to JSON array string
func toJSONArray(items []string) string {
	if len(items) == 0 {
		return "[]"
	}

	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf(`"%s"`, strings.ReplaceAll(item, `"`, `\"`))
	}

	return "[" + strings.Join(quoted, ", ") + "]"
}

// Close cleans up resources
func (c *Client) Close() error {
	c.logger.Info("Hyper-V client closed")
	return nil
}

// String returns a string representation of the client
func (c *Client) String() string {
	if c.config.UseWinRM {
		return fmt.Sprintf("Hyper-V Client (remote=%s, winrm=true)", c.config.Host)
	}
	return "Hyper-V Client (local)"
}

// SearchVMs searches for VMs matching a query
func (c *Client) SearchVMs(ctx context.Context, query string) ([]*VMInfo, error) {
	vms, err := c.ListVMs(ctx)
	if err != nil {
		return nil, err
	}

	query = strings.ToLower(query)
	var matches []*VMInfo

	for _, vm := range vms {
		if strings.Contains(strings.ToLower(vm.Name), query) ||
			strings.Contains(strings.ToLower(vm.State), query) ||
			strings.Contains(strings.ToLower(vm.Status), query) {
			matches = append(matches, vm)
		}
	}

	return matches, nil
}
