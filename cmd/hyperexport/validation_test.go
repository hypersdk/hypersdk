package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"hypersdk/providers/vsphere"
)

func TestNewPreExportValidator(t *testing.T) {
	validator := NewPreExportValidator(nil)
	if validator == nil {
		t.Fatal("NewPreExportValidator returned nil")
	}
}

func TestNewPostExportValidator(t *testing.T) {
	validator := NewPostExportValidator(nil)
	if validator == nil {
		t.Fatal("NewPostExportValidator returned nil")
	}
}

func TestPreExportValidator_ValidateExport(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPreExportValidator(nil)
	ctx := context.Background()

	vm := vsphere.VMInfo{
		Name:       "test-vm",
		PowerState: "poweredOff",
	}

	// Small estimated size (1 MB) should pass
	report := validator.ValidateExport(ctx, vm, tmpDir, 1024*1024)
	if report == nil {
		t.Fatal("ValidateExport returned nil")
	}

	if len(report.Checks) == 0 {
		t.Error("Expected validation checks, got none")
	}

	if report.Timestamp.IsZero() {
		t.Error("Report timestamp should be set")
	}
}

func TestPreExportValidator_ValidateDiskSpace_Sufficient(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPreExportValidator(nil)

	// Request small amount of space (1 MB)
	result := validator.validateDiskSpace(tmpDir, 1024*1024)

	if !result.Passed {
		t.Errorf("Expected disk space check to pass, got: %s", result.Message)
	}
	if result.Name != "Disk Space Check" {
		t.Error("Check name mismatch")
	}
}

func TestPreExportValidator_ValidateDiskSpace_Warning(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPreExportValidator(nil)

	// This test depends on actual disk space
	// We can't easily simulate low disk warning without mocking syscall.Statfs
	// Just verify the function runs without errors
	result := validator.validateDiskSpace(tmpDir, 100)

	if result.Name != "Disk Space Check" {
		t.Error("Check name mismatch")
	}
}

func TestPreExportValidator_ValidateDiskSpace_NonexistentPath(t *testing.T) {
	validator := NewPreExportValidator(nil)

	result := validator.validateDiskSpace("/nonexistent/path/that/does/not/exist", 1024)

	if result.Passed {
		t.Error("Expected disk space check to fail for nonexistent path")
	}
	if result.Name != "Disk Space Check" {
		t.Error("Check name mismatch")
	}
}

func TestPreExportValidator_ValidatePermissions_Writable(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPreExportValidator(nil)

	result := validator.validatePermissions(tmpDir)

	if !result.Passed {
		t.Errorf("Expected permissions check to pass for writable dir: %s", result.Message)
	}
	if result.Warning {
		t.Error("Should not have warning for writable directory")
	}
	if result.Name != "Permissions Check" {
		t.Error("Check name mismatch")
	}

	// Verify test file was cleaned up
	testFile := filepath.Join(tmpDir, ".hyperexport_test")
	if _, err := os.Stat(testFile); !os.IsNotExist(err) {
		t.Error("Test file should be cleaned up after permissions check")
	}
}

func TestPreExportValidator_ValidatePermissions_NotWritable(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPreExportValidator(nil)

	// Create a read-only directory
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0555); err != nil {
		t.Fatalf("Failed to create readonly dir: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup

	result := validator.validatePermissions(readOnlyDir)

	if result.Passed {
		t.Error("Expected permissions check to fail for readonly directory")
	}
	if result.Name != "Permissions Check" {
		t.Error("Check name mismatch")
	}
}

func TestPreExportValidator_ValidateVMState_PoweredOff(t *testing.T) {
	validator := NewPreExportValidator(nil)

	vm := vsphere.VMInfo{
		Name:       "test-vm",
		PowerState: "poweredOff",
	}

	result := validator.validateVMState(vm)

	if !result.Passed {
		t.Error("Expected VM state check to pass for powered off VM")
	}
	if result.Warning {
		t.Error("Should not have warning for powered off VM")
	}
	if result.Name != "VM State Check" {
		t.Error("Check name mismatch")
	}
}

func TestPreExportValidator_ValidateVMState_PoweredOn(t *testing.T) {
	validator := NewPreExportValidator(nil)

	vm := vsphere.VMInfo{
		Name:       "test-vm",
		PowerState: "poweredOn",
	}

	result := validator.validateVMState(vm)

	if !result.Passed {
		t.Error("Expected VM state check to pass (with warning) for powered on VM")
	}
	if !result.Warning {
		t.Error("Should have warning for powered on VM")
	}
	if !strings.Contains(result.Message, "crash-consistent") {
		t.Error("Message should mention crash-consistent for powered on VM")
	}
}

func TestPreExportValidator_CheckExistingExport_NotExists(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPreExportValidator(nil)

	result := validator.checkExistingExport(tmpDir, "new-vm")

	if !result.Passed {
		t.Error("Expected check to pass when no existing export")
	}
	if result.Warning {
		t.Error("Should not have warning when no existing export")
	}
}

func TestPreExportValidator_CheckExistingExport_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPreExportValidator(nil)

	// Create existing export directory
	vmName := "existing-vm"
	existingDir := filepath.Join(tmpDir, sanitizeVMName(vmName))
	if err := os.Mkdir(existingDir, 0755); err != nil {
		t.Fatalf("Failed to create existing export dir: %v", err)
	}

	result := validator.checkExistingExport(tmpDir, vmName)

	if !result.Passed {
		t.Error("Expected check to pass (with warning) when existing export found")
	}
	if !result.Warning {
		t.Error("Should have warning when existing export found")
	}
	if !strings.Contains(result.Message, "overwritten") {
		t.Error("Message should mention overwrite")
	}
}

func TestSanitizeVMName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal name", "my-vm", "my-vm"},
		{"with spaces", "my vm", "my vm"},
		{"with slash", "folder/vm", "folder_vm"},
		{"with backslash", "folder\\vm", "folder_vm"},
		{"with colon", "vm:1", "vm_1"},
		{"with asterisk", "vm*test", "vm_test"},
		{"with question", "vm?test", "vm_test"},
		{"with pipe", "vm|test", "vm_test"},
		{"with quotes", "\"vm\"", "_vm_"},
		{"with less/greater", "<vm>", "_vm_"},
		{"multiple invalid", "vm:/\\*?", "vm_____"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeVMName(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPostExportValidator_ValidateOVFFile_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPostExportValidator(nil)

	// Create a valid OVF file
	ovfFile := filepath.Join(tmpDir, "test.ovf")
	ovfContent := []byte("<?xml version=\"1.0\"?>\n<ovf:Envelope></ovf:Envelope>")
	if err := os.WriteFile(ovfFile, ovfContent, 0644); err != nil {
		t.Fatalf("Failed to create OVF file: %v", err)
	}

	result := validator.validateOVFFile(tmpDir)

	if !result.Passed {
		t.Errorf("Expected OVF check to pass: %s", result.Message)
	}
	if result.Name != "OVF File Check" {
		t.Error("Check name mismatch")
	}
}

func TestPostExportValidator_ValidateOVFFile_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPostExportValidator(nil)

	result := validator.validateOVFFile(tmpDir)

	if result.Passed {
		t.Error("Expected OVF check to fail when no OVF file exists")
	}
}

func TestPostExportValidator_ValidateOVFFile_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPostExportValidator(nil)

	// Create empty OVF file
	ovfFile := filepath.Join(tmpDir, "test.ovf")
	if err := os.WriteFile(ovfFile, []byte{}, 0644); err != nil {
		t.Fatalf("Failed to create empty OVF file: %v", err)
	}

	result := validator.validateOVFFile(tmpDir)

	if result.Passed {
		t.Error("Expected OVF check to fail for empty file")
	}
	if !strings.Contains(result.Message, "empty") {
		t.Error("Message should mention file is empty")
	}
}

func TestPostExportValidator_ValidateReferencedFiles_Present(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPostExportValidator(nil)

	// Create VMDK files
	vmdk1 := filepath.Join(tmpDir, "disk-0.vmdk")
	vmdk2 := filepath.Join(tmpDir, "disk-1.vmdk")
	os.WriteFile(vmdk1, []byte("vmdk data 1"), 0644)
	os.WriteFile(vmdk2, []byte("vmdk data 2"), 0644)

	result := validator.validateReferencedFiles(tmpDir)

	if !result.Passed {
		t.Errorf("Expected referenced files check to pass: %s", result.Message)
	}
	if !strings.Contains(result.Message, "2 files") {
		t.Error("Message should mention number of files found")
	}
}

func TestPostExportValidator_ValidateReferencedFiles_Missing(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPostExportValidator(nil)

	// No VMDK files created
	result := validator.validateReferencedFiles(tmpDir)

	if result.Passed {
		t.Error("Expected referenced files check to fail when no VMDKs exist")
	}
}

func TestPostExportValidator_ValidateFileSizes(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPostExportValidator(nil)

	// Create files with known sizes
	file1 := filepath.Join(tmpDir, "file1.vmdk")
	file2 := filepath.Join(tmpDir, "file2.ovf")
	os.WriteFile(file1, make([]byte, 1024), 0644)     // 1 KB
	os.WriteFile(file2, make([]byte, 2048), 0644)     // 2 KB

	result := validator.validateFileSizes(tmpDir)

	if !result.Passed {
		t.Errorf("Expected file size check to pass: %s", result.Message)
	}
}

func TestPostExportValidator_ValidateFileSizes_Zero(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPostExportValidator(nil)

	// Empty directory (no files)
	result := validator.validateFileSizes(tmpDir)

	if result.Passed {
		t.Error("Expected file size check to fail when total size is 0")
	}
}

func TestPostExportValidator_ValidateChecksums_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPostExportValidator(nil)

	result := validator.validateChecksums(tmpDir)

	if !result.Passed {
		t.Error("Expected checksum check to pass (with warning) when no checksum file")
	}
	if !result.Warning {
		t.Error("Should have warning when no checksum file")
	}
}

func TestPostExportValidator_ValidateChecksums_FileExists(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPostExportValidator(nil)

	// Create checksum file
	checksumFile := filepath.Join(tmpDir, "checksums.txt")
	content := "abc123  file1.vmdk\ndef456  file2.vmdk\n"
	os.WriteFile(checksumFile, []byte(content), 0644)

	result := validator.validateChecksums(tmpDir)

	if !result.Passed {
		t.Errorf("Expected checksum check to pass: %s", result.Message)
	}
}

func TestPostExportValidator_ValidateExportedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	validator := NewPostExportValidator(nil)

	// Create valid export structure
	os.WriteFile(filepath.Join(tmpDir, "test.ovf"), []byte("ovf content"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "disk.vmdk"), []byte("vmdk data"), 0644)

	report := validator.ValidateExportedFiles(tmpDir)

	if report == nil {
		t.Fatal("ValidateExportedFiles returned nil")
	}

	if len(report.Checks) == 0 {
		t.Error("Expected validation checks in report")
	}

	if report.Timestamp.IsZero() {
		t.Error("Report timestamp should be set")
	}
}

func TestCalculateFileChecksum(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file with known content
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("Hello, checksum!")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	checksum1, err := CalculateFileChecksum(testFile)
	if err != nil {
		t.Fatalf("CalculateFileChecksum failed: %v", err)
	}

	if checksum1 == "" {
		t.Error("Checksum should not be empty")
	}

	// Verify checksum is consistent
	checksum2, err := CalculateFileChecksum(testFile)
	if err != nil {
		t.Fatalf("CalculateFileChecksum failed on second call: %v", err)
	}

	if checksum1 != checksum2 {
		t.Error("Checksums should be identical for same file")
	}

	// Verify checksum is 64 characters (SHA256 hex)
	if len(checksum1) != 64 {
		t.Errorf("Expected checksum length 64, got %d", len(checksum1))
	}
}

func TestCalculateFileChecksum_NonexistentFile(t *testing.T) {
	_, err := CalculateFileChecksum("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestCalculateFileChecksum_DifferentContent(t *testing.T) {
	tmpDir := t.TempDir()

	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	os.WriteFile(file1, []byte("content 1"), 0644)
	os.WriteFile(file2, []byte("content 2"), 0644)

	checksum1, _ := CalculateFileChecksum(file1)
	checksum2, _ := CalculateFileChecksum(file2)

	if checksum1 == checksum2 {
		t.Error("Different files should have different checksums")
	}
}

func TestSaveChecksumManifest(t *testing.T) {
	tmpDir := t.TempDir()

	checksums := map[string]string{
		"file1.vmdk": "abc123def456",
		"file2.vmdk": "789012345678",
		"vm.ovf":     "fedcba987654",
	}

	err := SaveChecksumManifest(tmpDir, checksums)
	if err != nil {
		t.Fatalf("SaveChecksumManifest failed: %v", err)
	}

	// Verify manifest file was created
	manifestPath := filepath.Join(tmpDir, "checksums.txt")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Error("Checksum manifest file was not created")
	}

	// Read and verify content
	content, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read manifest: %v", err)
	}

	contentStr := string(content)
	for filename, checksum := range checksums {
		expectedLine := checksum + "  " + filename
		if !strings.Contains(contentStr, expectedLine) {
			t.Errorf("Manifest missing entry for %s", filename)
		}
	}
}

func TestComputeExportChecksums(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	files := map[string][]byte{
		"test.ovf":     []byte("ovf content"),
		"disk-0.vmdk":  []byte("disk 0 data"),
		"disk-1.vmdk":  []byte("disk 1 data"),
	}

	for filename, content := range files {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	checksums, err := ComputeExportChecksums(tmpDir, nil)
	if err != nil {
		t.Fatalf("ComputeExportChecksums failed: %v", err)
	}

	if len(checksums) != len(files) {
		t.Errorf("Expected %d checksums, got %d", len(files), len(checksums))
	}

	for filename := range files {
		if _, exists := checksums[filename]; !exists {
			t.Errorf("Missing checksum for %s", filename)
		}
	}
}

func TestComputeExportChecksums_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	checksums, err := ComputeExportChecksums(tmpDir, nil)
	if err != nil {
		t.Fatalf("ComputeExportChecksums failed: %v", err)
	}

	if len(checksums) != 0 {
		t.Errorf("Expected 0 checksums for empty directory, got %d", len(checksums))
	}
}

func TestValidationResult_Fields(t *testing.T) {
	result := ValidationResult{
		Name:    "Test Check",
		Passed:  true,
		Message: "Test passed successfully",
		Warning: false,
	}

	if result.Name != "Test Check" {
		t.Error("Name field mismatch")
	}
	if !result.Passed {
		t.Error("Passed field mismatch")
	}
	if result.Message != "Test passed successfully" {
		t.Error("Message field mismatch")
	}
	if result.Warning {
		t.Error("Warning field mismatch")
	}
}

func TestValidationReport_Fields(t *testing.T) {
	result1 := ValidationResult{Name: "Check 1", Passed: true, Message: "OK", Warning: false}
	result2 := ValidationResult{Name: "Check 2", Passed: true, Message: "Warning", Warning: true}

	report := ValidationReport{
		Checks:      []ValidationResult{result1, result2},
		AllPassed:   true,
		HasWarnings: true,
	}

	if len(report.Checks) != 2 {
		t.Error("Checks length mismatch")
	}
	if !report.AllPassed {
		t.Error("AllPassed should be true")
	}
	if !report.HasWarnings {
		t.Error("HasWarnings should be true")
	}
}

func TestValidationReport_AllPassed_False(t *testing.T) {
	result1 := ValidationResult{Name: "Check 1", Passed: false, Message: "Failed"}
	result2 := ValidationResult{Name: "Check 2", Passed: true, Message: "OK"}

	report := ValidationReport{
		Checks:    []ValidationResult{result1, result2},
		AllPassed: false,
	}

	if report.AllPassed {
		t.Error("AllPassed should be false when any check fails")
	}
}
