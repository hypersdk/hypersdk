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

	"github.com/vmware/govmomi/vim25/types"

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

	// Get current disk information
	currentDisks, err := client.ListVMDisks(ctx, vmPath)
	if err != nil {
		iem.log.Warn("failed to get VM disks, forcing full export", "error", err)
		result.NeedsFullExport = true
		result.Reason = fmt.Sprintf("Failed to get disk info: %v", err)
		return result, nil
	}

	// Build current disk map (disk key -> disk info)
	currentDiskMap := make(map[string]types.VirtualDisk)
	var totalCurrentSize int64
	for _, disk := range currentDisks {
		// Get disk backing info to extract the file path
		backing, ok := disk.Backing.(types.BaseVirtualDeviceFileBackingInfo)
		if !ok {
			continue
		}
		backingInfo := backing.GetVirtualDeviceFileBackingInfo()
		diskKey := filepath.Base(backingInfo.FileName)
		currentDiskMap[diskKey] = disk

		// Calculate total size
		if disk.CapacityInBytes > 0 {
			totalCurrentSize += disk.CapacityInBytes
		}
	}

	// Compare with previous state
	var changedDisks, newDisks, removedDisks []string

	// Check for new or changed disks
	for diskKey := range currentDiskMap {
		if prevSize, exists := prevState.DiskSizes[diskKey]; exists {
			// Disk existed before - check if it changed
			// For now, we consider any disk as potentially changed
			// In production, you might compare modification times or checksums
			if prevChecksum, hasChecksum := prevState.DiskChecksums[diskKey]; hasChecksum {
				// Disk has a checksum - assume it might have changed
				changedDisks = append(changedDisks, diskKey)
				iem.log.Debug("disk may have changed", "disk", diskKey, "prev_checksum", prevChecksum[:8])
			} else {
				// No checksum available - assume changed if size different
				if currentDiskMap[diskKey].CapacityInBytes != prevSize {
					changedDisks = append(changedDisks, diskKey)
					iem.log.Debug("disk size changed", "disk", diskKey,
						"old_size", prevSize, "new_size", currentDiskMap[diskKey].CapacityInBytes)
				}
			}
		} else {
			// New disk
			newDisks = append(newDisks, diskKey)
			iem.log.Debug("new disk detected", "disk", diskKey)
		}
	}

	// Check for removed disks
	for prevDisk := range prevState.DiskSizes {
		if _, exists := currentDiskMap[prevDisk]; !exists {
			removedDisks = append(removedDisks, prevDisk)
			iem.log.Debug("disk removed", "disk", prevDisk)
		}
	}

	// Determine if full export is needed
	if len(newDisks) > 0 || len(removedDisks) > 0 {
		result.NeedsFullExport = true
		result.Reason = fmt.Sprintf("Disk topology changed: %d new, %d removed disks", len(newDisks), len(removedDisks))
		result.ChangedDisks = changedDisks
		result.NewDisks = newDisks
		result.RemovedDisks = removedDisks
		iem.log.Info("full export required: disk topology changed",
			"new_disks", len(newDisks),
			"removed_disks", len(removedDisks))
		return result, nil
	}

	// Calculate potential savings with incremental export
	if len(changedDisks) > 0 {
		// Some disks changed - incremental might be beneficial
		unchangedSize := int64(0)
		for diskKey, size := range prevState.DiskSizes {
			if _, changed := contains(changedDisks, diskKey); !changed {
				unchangedSize += size
			}
		}

		result.NeedsFullExport = false
		result.ChangedDisks = changedDisks
		result.TotalSavings = unchangedSize
		result.Reason = fmt.Sprintf("Incremental export can save ~%s (%d unchanged disks)",
			formatBytes(unchangedSize), len(prevState.DiskSizes)-len(changedDisks))

		iem.log.Info("incremental export recommended",
			"changed_disks", len(changedDisks),
			"unchanged_disks", len(prevState.DiskSizes)-len(changedDisks),
			"potential_savings", formatBytes(unchangedSize))
	} else {
		// No disks changed
		result.NeedsFullExport = false
		result.Reason = "No disk changes detected since last export"
		result.TotalSavings = prevState.TotalSize

		iem.log.Info("no changes detected - incremental not needed",
			"vm_size", formatBytes(totalCurrentSize))
	}

	return result, nil
}

// Helper function to check if a string slice contains a value
func contains(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
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
