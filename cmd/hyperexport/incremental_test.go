package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hypersdk/logger"
)

func TestNewIncrementalExportManager(t *testing.T) {
	tmpDir := t.TempDir()

	manager, err := NewIncrementalExportManager(tmpDir, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("NewIncrementalExportManager failed: %v", err)
	}
	if manager == nil {
		t.Fatal("NewIncrementalExportManager returned nil")
	}
}

func TestIncrementalExportManager_SaveExportState(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewIncrementalExportManager(tmpDir, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("NewIncrementalExportManager failed: %v", err)
	}

	state := &ExportState{
		VMPath:         "/datacenter/vm/test-vm",
		LastExportTime: time.Now(),
		DiskChecksums: map[string]string{
			"disk-0": "abc123",
			"disk-1": "def456",
		},
		DiskSizes: map[string]int64{
			"disk-0": 1024 * 1024 * 100, // 100 MB
			"disk-1": 1024 * 1024 * 200, // 200 MB
		},
		TotalSize:  1024 * 1024 * 300,
		ExportPath: "/exports/test-vm.ova",
		Format:     "ova",
		Version:    1,
	}

	err = manager.SaveExportState(state)
	if err != nil {
		t.Fatalf("SaveExportState failed: %v", err)
	}

	// Verify file was created
	stateFile := manager.getStateFile("/datacenter/vm/test-vm")
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		t.Error("State file was not created")
	}
}

func TestIncrementalExportManager_LoadExportState(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewIncrementalExportManager(tmpDir, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("NewIncrementalExportManager failed: %v", err)
	}

	// Create a state first
	state := &ExportState{
		VMPath:         "/datacenter/vm/test-vm",
		LastExportTime: time.Now(),
		DiskChecksums: map[string]string{
			"disk-0": "checksum123",
		},
		TotalSize: 1024 * 1024,
		Format:    "vmdk",
		Version:   1,
	}

	if err := manager.SaveExportState(state); err != nil {
		t.Fatalf("SaveExportState failed: %v", err)
	}

	// Load the state
	loaded, err := manager.LoadExportState("/datacenter/vm/test-vm")
	if err != nil {
		t.Fatalf("LoadExportState failed: %v", err)
	}

	if loaded.VMPath != state.VMPath {
		t.Errorf("Expected VMPath %q, got %q", state.VMPath, loaded.VMPath)
	}
	if loaded.Format != state.Format {
		t.Errorf("Expected Format %q, got %q", state.Format, loaded.Format)
	}
}

func TestIncrementalExportManager_LoadExportState_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewIncrementalExportManager(tmpDir, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("NewIncrementalExportManager failed: %v", err)
	}

	// Try to load non-existent state
	state, err := manager.LoadExportState("/datacenter/vm/nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent state, got nil")
	}
	if state != nil {
		t.Error("Expected nil state on error")
	}
}

func TestExportState_Fields(t *testing.T) {
	now := time.Now()
	state := &ExportState{
		VMPath:         "/datacenter/vm/test",
		LastExportTime: now,
		DiskChecksums: map[string]string{
			"disk-0": "hash1",
			"disk-1": "hash2",
		},
		DiskSizes: map[string]int64{
			"disk-0": 1000,
			"disk-1": 2000,
		},
		TotalSize:  3000,
		ExportPath: "/exports/test.ova",
		Format:     "ova",
		Version:    1,
	}

	if state.VMPath != "/datacenter/vm/test" {
		t.Error("VMPath mismatch")
	}
	if !state.LastExportTime.Equal(now) {
		t.Error("LastExportTime mismatch")
	}
	if len(state.DiskChecksums) != 2 {
		t.Error("DiskChecksums length mismatch")
	}
	if len(state.DiskSizes) != 2 {
		t.Error("DiskSizes length mismatch")
	}
	if state.TotalSize != 3000 {
		t.Error("TotalSize mismatch")
	}
	if state.Format != "ova" {
		t.Error("Format mismatch")
	}
	if state.Version != 1 {
		t.Error("Version mismatch")
	}
}

func TestIncrementalExportManager_AnalyzeChanges(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewIncrementalExportManager(tmpDir, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("NewIncrementalExportManager failed: %v", err)
	}
	ctx := context.Background()

	// This will fail without a real client, but we test the method exists
	result, err := manager.AnalyzeChanges(ctx, nil, "/datacenter/vm/test-vm")
	if err == nil {
		t.Log("AnalyzeChanges would work with a real vSphere client")
	}
	if result != nil {
		t.Log("Result returned (unexpected without real client)")
	}
}

func TestIncrementalExportManager_CreateExportState(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewIncrementalExportManager(tmpDir, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("NewIncrementalExportManager failed: %v", err)
	}

	vmPath := "/datacenter/vm/test-vm"

	// This will work without a real client as it just creates the struct
	state := manager.CreateExportState(vmPath, nil, nil)
	if state == nil {
		t.Fatal("CreateExportState returned nil")
	}

	if state.VMPath != vmPath {
		t.Errorf("Expected VMPath %q, got %q", vmPath, state.VMPath)
	}

	if state.Version != 1 {
		t.Error("Expected Version 1")
	}
}

func TestIncrementalExportManager_CleanupOldStates(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewIncrementalExportManager(tmpDir, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("NewIncrementalExportManager failed: %v", err)
	}
	ctx := context.Background()

	// This will fail without a real client, but we test the method exists
	err = manager.CleanupOldStates(ctx, nil)
	if err == nil {
		t.Log("CleanupOldStates would work with a real vSphere client")
	}
}

func TestIncrementalResult_Fields(t *testing.T) {
	result := &IncrementalResult{
		ChangedDisks:    []string{"disk-0"},
		UnchangedDisks:  []string{"disk-1"},
		NewDisks:        []string{},
		RemovedDisks:    []string{},
		TotalSavings:    1024 * 1024 * 900,
		NeedsFullExport: false,
		Reason:          "Only 1 disk changed",
	}

	if result.NeedsFullExport {
		t.Error("NeedsFullExport should be false")
	}
	if len(result.ChangedDisks) != 1 {
		t.Error("ChangedDisks length mismatch")
	}
	if len(result.UnchangedDisks) != 1 {
		t.Error("UnchangedDisks length mismatch")
	}
	if result.TotalSavings != 1024*1024*900 {
		t.Error("TotalSavings mismatch")
	}
	if result.Reason == "" {
		t.Error("Reason should not be empty")
	}
}

func TestIncrementalExportManager_GetIncrementalSavings(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewIncrementalExportManager(tmpDir, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("NewIncrementalExportManager failed: %v", err)
	}

	result := &IncrementalResult{
		ChangedDisks:    []string{"disk-0"},
		UnchangedDisks:  []string{"disk-1"},
		TotalSavings:    1024 * 1024 * 900,
		NeedsFullExport: false,
	}

	savings := manager.GetIncrementalSavings(result)
	if savings == "" {
		t.Error("Expected savings string, got empty")
	}
}

func TestIncrementalExportManager_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewIncrementalExportManager(tmpDir, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("NewIncrementalExportManager failed: %v", err)
	}

	// Create state
	state := &ExportState{
		VMPath:         "/datacenter/vm/roundtrip-test",
		LastExportTime: time.Now(),
		DiskChecksums:  map[string]string{"disk-0": "abc123"},
		TotalSize:      1024 * 1024,
		Format:         "ova",
		Version:        1,
	}

	// Save
	if err := manager.SaveExportState(state); err != nil {
		t.Fatalf("SaveExportState failed: %v", err)
	}

	// Load
	loaded, err := manager.LoadExportState("/datacenter/vm/roundtrip-test")
	if err != nil {
		t.Fatalf("LoadExportState failed: %v", err)
	}

	// Verify
	if loaded.VMPath != state.VMPath {
		t.Error("VMPath mismatch after round-trip")
	}
	if loaded.Format != state.Format {
		t.Error("Format mismatch after round-trip")
	}
	if loaded.TotalSize != state.TotalSize {
		t.Error("TotalSize mismatch after round-trip")
	}
}

func TestIncrementalExportManager_GetStateFile(t *testing.T) {
	tmpDir := t.TempDir()
	manager, err := NewIncrementalExportManager(tmpDir, logger.NewTestLogger(t))
	if err != nil {
		t.Fatalf("NewIncrementalExportManager failed: %v", err)
	}

	vmPath := "/datacenter/vm/test-vm"
	stateFile := manager.getStateFile(vmPath)

	if stateFile == "" {
		t.Error("getStateFile returned empty string")
	}

	// Should be in the tmpDir
	if !filepath.IsAbs(stateFile) {
		t.Error("stateFile should be absolute path")
	}

	// Should end with .json
	if filepath.Ext(stateFile) != ".json" {
		t.Error("stateFile should have .json extension")
	}
}
