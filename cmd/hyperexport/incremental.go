// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"hypersdk/logger"
	"hypersdk/providers/vsphere"
)

// IncrementalExportManager handles incremental VM exports
type IncrementalExportManager struct {
	log      logger.Logger
	stateDir string // Directory to store export state
}

// ExportState tracks the state of previous exports for incremental support
type ExportState struct {
	VMPath       string            `json:"vm_path"`
	LastExportTime time.Time       `json:"last_export_time"`
	DiskChecksums map[string]string `json:"disk_checksums"` // disk path -> checksum
	DiskSizes     map[string]int64  `json:"disk_sizes"`     // disk path -> size
	TotalSize     int64             `json:"total_size"`
	ExportPath    string            `json:"export_path"`
	Format        string            `json:"format"`
	Version       int               `json:"version"` // State format version
}

// IncrementalResult contains results of incremental export analysis
type IncrementalResult struct {
	ChangedDisks   []string
	UnchangedDisks []string
	NewDisks       []string
	RemovedDisks   []string
	TotalSavings   int64
	NeedsFullExport bool
	Reason         string
}

// NewIncrementalExportManager creates a new incremental export manager
func NewIncrementalExportManager(stateDir string, log logger.Logger) (*IncrementalExportManager, error) {
	if stateDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home directory: %w", err)
		}
		stateDir = filepath.Join(homeDir, ".hyperexport", "state")
	}

	// Create state directory if it doesn't exist
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}

	return &IncrementalExportManager{
		log:      log,
		stateDir: stateDir,
	}, nil
}

// LoadExportState loads the previous export state for a VM
func (iem *IncrementalExportManager) LoadExportState(vmPath string) (*ExportState, error) {
	stateFile := iem.getStateFile(vmPath)

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			iem.log.Info("no previous export state found", "vm", vmPath)
			return nil, nil // No previous export
		}
		return nil, fmt.Errorf("read state file: %w", err)
	}

	var state ExportState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal state: %w", err)
	}

	iem.log.Info("loaded export state",
		"vm", vmPath,
		"lastExport", state.LastExportTime,
		"disks", len(state.DiskChecksums))

	return &state, nil
}

// SaveExportState saves the export state for a VM
func (iem *IncrementalExportManager) SaveExportState(state *ExportState) error {
	stateFile := iem.getStateFile(state.VMPath)

	state.Version = 1 // Current state format version
	state.LastExportTime = time.Now()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}

	iem.log.Info("saved export state",
		"vm", state.VMPath,
		"disks", len(state.DiskChecksums))

	return nil
}

// AnalyzeChanges analyzes which disks have changed since last export
func (iem *IncrementalExportManager) AnalyzeChanges(ctx context.Context, client *vsphere.VSphereClient, vmPath string) (*IncrementalResult, error) {
	result := &IncrementalResult{
		ChangedDisks:   []string{},
		UnchangedDisks: []string{},
		NewDisks:       []string{},
		RemovedDisks:   []string{},
	}

	// Load previous state
	prevState, err := iem.LoadExportState(vmPath)
	if err != nil {
		return nil, fmt.Errorf("load export state: %w", err)
	}

	if prevState == nil {
		// No previous export, need full export
		result.NeedsFullExport = true
		result.Reason = "No previous export found"
		iem.log.Info("full export required: no previous state")
		return result, nil
	}

	// Get current VM info
	_, err = client.GetVMInfo(ctx, vmPath)
	if err != nil {
		return nil, fmt.Errorf("get VM info: %w", err)
	}

	// TODO: Implement GetVMDisks in vsphere.VSphereClient
	// For now, return a result that forces full export
	result.NeedsFullExport = true
	result.Reason = "Disk change detection not yet implemented"
	iem.log.Info("full export required: disk detection not implemented")
	return result, nil

	// NOTE: Code below is commented out until GetVMDisks is implemented in vsphere.VSphereClient
	// The implementation would:
	// 1. Get current disk information
	// 2. Compare with previous state
	// 3. Identify changed, new, removed disks
	// 4. Calculate potential savings
	// 5. Determine if full export is needed
}

// CreateExportState creates export state from export results
func (iem *IncrementalExportManager) CreateExportState(vmPath string, exportResult *vsphere.ExportResult, vmInfo *vsphere.VMInfo) *ExportState {
	state := &ExportState{
		VMPath:        vmPath,
		DiskChecksums: make(map[string]string),
		DiskSizes:     make(map[string]int64),
		TotalSize:     exportResult.TotalSize,
		ExportPath:    exportResult.OutputDir,
		Format:        exportResult.Format,
	}

	// Calculate checksums for exported disks
	for _, file := range exportResult.Files {
		if filepath.Ext(file) == ".vmdk" {
			checksum, err := iem.calculateFileChecksum(file)
			if err != nil {
				iem.log.Warn("failed to calculate checksum", "file", file, "error", err)
				continue
			}

			// Get file size
			info, err := os.Stat(file)
			if err != nil {
				iem.log.Warn("failed to stat file", "file", file, "error", err)
				continue
			}

			diskName := filepath.Base(file)
			state.DiskChecksums[diskName] = checksum
			state.DiskSizes[diskName] = info.Size()
		}
	}

	return state
}

// calculateFileChecksum calculates SHA-256 checksum of a file
func (iem *IncrementalExportManager) calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// getStateFile returns the path to the state file for a VM
func (iem *IncrementalExportManager) getStateFile(vmPath string) string {
	// Create a safe filename from VM path
	safeFileName := sanitizeForPath(vmPath) + ".json"
	return filepath.Join(iem.stateDir, safeFileName)
}

// CleanupOldStates removes state files for VMs that no longer exist
func (iem *IncrementalExportManager) CleanupOldStates(ctx context.Context, client *vsphere.VSphereClient) error {
	// List all state files
	stateFiles, err := filepath.Glob(filepath.Join(iem.stateDir, "*.json"))
	if err != nil {
		return fmt.Errorf("glob state files: %w", err)
	}

	iem.log.Info("checking state files", "count", len(stateFiles))

	// Get list of all VMs
	vms, err := client.FindAllVMs(ctx)
	if err != nil {
		return fmt.Errorf("find VMs: %w", err)
	}

	vmMap := make(map[string]bool)
	for _, vm := range vms {
		stateFile := iem.getStateFile(vm)
		vmMap[stateFile] = true
	}

	// Remove state files for non-existent VMs
	removedCount := 0
	for _, stateFile := range stateFiles {
		if !vmMap[stateFile] {
			iem.log.Info("removing orphaned state file", "file", stateFile)
			if err := os.Remove(stateFile); err != nil {
				iem.log.Warn("failed to remove state file", "file", stateFile, "error", err)
			} else {
				removedCount++
			}
		}
	}

	iem.log.Info("state cleanup complete", "removed", removedCount)
	return nil
}

// GetIncrementalSavings calculates potential savings from incremental export
func (iem *IncrementalExportManager) GetIncrementalSavings(result *IncrementalResult) string {
	if result.TotalSavings == 0 {
		return "No savings (full export required)"
	}

	savingsMB := float64(result.TotalSavings) / 1024 / 1024
	unchangedCount := len(result.UnchangedDisks)

	return fmt.Sprintf("%.1f MB (%d unchanged disks)", savingsMB, unchangedCount)
}
