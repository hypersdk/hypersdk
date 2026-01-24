// SPDX-License-Identifier: LGPL-3.0-or-later

package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder()

	if builder == nil {
		t.Fatal("NewBuilder() returned nil")
	}

	if builder.manifest == nil {
		t.Fatal("Builder manifest is nil")
	}

	if builder.manifest.ManifestVersion != CurrentVersion {
		t.Errorf("Expected manifest version %q, got %q", CurrentVersion, builder.manifest.ManifestVersion)
	}
}

func TestBuilderWithSource(t *testing.T) {
	builder := NewBuilder()
	builder.WithSource("vsphere", "vm-1234", "test-vm", "DC1", "govc-export")

	if builder.manifest.Source == nil {
		t.Fatal("Source metadata is nil")
	}

	if builder.manifest.Source.Provider != "vsphere" {
		t.Errorf("Expected provider %q, got %q", "vsphere", builder.manifest.Source.Provider)
	}

	if builder.manifest.Source.VMID != "vm-1234" {
		t.Errorf("Expected VM ID %q, got %q", "vm-1234", builder.manifest.Source.VMID)
	}
}

func TestBuilderWithVM(t *testing.T) {
	builder := NewBuilder()
	builder.WithVM(4, 16, "uefi", "linux", "Ubuntu 22.04", false)

	if builder.manifest.VM == nil {
		t.Fatal("VM metadata is nil")
	}

	if builder.manifest.VM.CPU != 4 {
		t.Errorf("Expected CPU count 4, got %d", builder.manifest.VM.CPU)
	}

	if builder.manifest.VM.Firmware != "uefi" {
		t.Errorf("Expected firmware %q, got %q", "uefi", builder.manifest.VM.Firmware)
	}
}

func TestBuilderAddDisk(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	diskPath := filepath.Join(tmpDir, "test-disk.vmdk")
	if err := os.WriteFile(diskPath, []byte("test data"), 0644); err != nil {
		t.Fatalf("Failed to create test disk: %v", err)
	}

	builder := NewBuilder()
	builder.AddDisk("disk-0", "vmdk", diskPath, 1024, 0, "boot")

	if len(builder.manifest.Disks) != 1 {
		t.Fatalf("Expected 1 disk, got %d", len(builder.manifest.Disks))
	}

	disk := builder.manifest.Disks[0]
	if disk.ID != "disk-0" {
		t.Errorf("Expected disk ID %q, got %q", "disk-0", disk.ID)
	}

	if disk.SourceFormat != "vmdk" {
		t.Errorf("Expected source format %q, got %q", "vmdk", disk.SourceFormat)
	}

	if disk.Bytes != 1024 {
		t.Errorf("Expected bytes 1024, got %d", disk.Bytes)
	}

	if disk.BootOrderHint != 0 {
		t.Errorf("Expected boot order hint 0, got %d", disk.BootOrderHint)
	}

	if disk.DiskType != "boot" {
		t.Errorf("Expected disk type %q, got %q", "boot", disk.DiskType)
	}
}

func TestBuilderAddDiskInvalidID(t *testing.T) {
	tmpDir := t.TempDir()
	diskPath := filepath.Join(tmpDir, "test.vmdk")
	os.WriteFile(diskPath, []byte("test"), 0644)

	builder := NewBuilder()
	builder.AddDisk("disk with spaces", "vmdk", diskPath, 1024, 0, "boot")

	if len(builder.errors) == 0 {
		t.Error("Expected error for invalid disk ID, got none")
	}

	if !strings.Contains(builder.errors[0].Error(), "invalid disk ID") {
		t.Errorf("Expected 'invalid disk ID' error, got: %v", builder.errors[0])
	}
}

func TestBuilderAddDiskDuplicateID(t *testing.T) {
	tmpDir := t.TempDir()
	diskPath := filepath.Join(tmpDir, "test.vmdk")
	os.WriteFile(diskPath, []byte("test"), 0644)

	builder := NewBuilder()
	builder.AddDisk("disk-0", "vmdk", diskPath, 1024, 0, "boot")
	builder.AddDisk("disk-0", "vmdk", diskPath, 1024, 0, "boot")

	if len(builder.errors) == 0 {
		t.Error("Expected error for duplicate disk ID, got none")
	}

	if !strings.Contains(builder.errors[0].Error(), "duplicate disk ID") {
		t.Errorf("Expected 'duplicate disk ID' error, got: %v", builder.errors[0])
	}
}

func TestBuilderAddDiskInvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	diskPath := filepath.Join(tmpDir, "test.vmdk")
	os.WriteFile(diskPath, []byte("test"), 0644)

	builder := NewBuilder()
	builder.AddDisk("disk-0", "invalid", diskPath, 1024, 0, "boot")

	if len(builder.errors) == 0 {
		t.Error("Expected error for invalid source format, got none")
	}

	if !strings.Contains(builder.errors[0].Error(), "invalid source format") {
		t.Errorf("Expected 'invalid source format' error, got: %v", builder.errors[0])
	}
}

func TestBuilderAddDiskFileNotFound(t *testing.T) {
	builder := NewBuilder()
	builder.AddDisk("disk-0", "vmdk", "/nonexistent/path/disk.vmdk", 1024, 0, "boot")

	if len(builder.errors) == 0 {
		t.Error("Expected error for nonexistent file, got none")
	}

	if !strings.Contains(builder.errors[0].Error(), "disk file not found") {
		t.Errorf("Expected 'disk file not found' error, got: %v", builder.errors[0])
	}
}

func TestBuilderAddDiskWithChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	diskPath := filepath.Join(tmpDir, "test.vmdk")
	testData := []byte("test data for checksum")
	if err := os.WriteFile(diskPath, testData, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	builder := NewBuilder()
	builder.AddDiskWithChecksum("disk-0", "vmdk", diskPath, int64(len(testData)), 0, "boot", true)

	if len(builder.errors) > 0 {
		t.Fatalf("Unexpected error: %v", builder.errors[0])
	}

	if len(builder.manifest.Disks) != 1 {
		t.Fatalf("Expected 1 disk, got %d", len(builder.manifest.Disks))
	}

	disk := builder.manifest.Disks[0]
	if disk.Checksum == "" {
		t.Error("Expected checksum to be computed, got empty string")
	}

	if !strings.HasPrefix(disk.Checksum, "sha256:") {
		t.Errorf("Expected checksum to start with 'sha256:', got %q", disk.Checksum)
	}

	if len(disk.Checksum) != 71 { // "sha256:" + 64 hex characters
		t.Errorf("Expected checksum length 71, got %d", len(disk.Checksum))
	}
}

func TestBuilderAddNIC(t *testing.T) {
	builder := NewBuilder()
	builder.AddNIC("eth0", "00:50:56:ab:cd:ef", "VM Network")

	if len(builder.manifest.NICs) != 1 {
		t.Fatalf("Expected 1 NIC, got %d", len(builder.manifest.NICs))
	}

	nic := builder.manifest.NICs[0]
	if nic.ID != "eth0" {
		t.Errorf("Expected NIC ID %q, got %q", "eth0", nic.ID)
	}

	if nic.MAC != "00:50:56:ab:cd:ef" {
		t.Errorf("Expected MAC %q, got %q", "00:50:56:ab:cd:ef", nic.MAC)
	}
}

func TestBuilderAddNote(t *testing.T) {
	builder := NewBuilder()
	builder.AddNote("Test note 1")
	builder.AddNote("Test note 2")

	if len(builder.manifest.Notes) != 2 {
		t.Fatalf("Expected 2 notes, got %d", len(builder.manifest.Notes))
	}

	if builder.manifest.Notes[0] != "Test note 1" {
		t.Errorf("Expected note %q, got %q", "Test note 1", builder.manifest.Notes[0])
	}
}

func TestBuilderAddWarning(t *testing.T) {
	builder := NewBuilder()
	builder.AddWarning("export", "Test warning message")

	if len(builder.manifest.Warnings) != 1 {
		t.Fatalf("Expected 1 warning, got %d", len(builder.manifest.Warnings))
	}

	warning := builder.manifest.Warnings[0]
	if warning.Stage != "export" {
		t.Errorf("Expected stage %q, got %q", "export", warning.Stage)
	}

	if warning.Message != "Test warning message" {
		t.Errorf("Expected message %q, got %q", "Test warning message", warning.Message)
	}

	if warning.Timestamp == nil {
		t.Error("Expected warning timestamp to be set, got nil")
	}
}

func TestBuilderWithPipeline(t *testing.T) {
	builder := NewBuilder()
	builder.WithPipeline(true, true, true, true)

	if builder.manifest.Pipeline == nil {
		t.Fatal("Pipeline config is nil")
	}

	if !builder.manifest.Pipeline.Inspect.Enabled {
		t.Error("Expected inspect stage to be enabled")
	}

	if !builder.manifest.Pipeline.Fix.Enabled {
		t.Error("Expected fix stage to be enabled")
	}

	if !builder.manifest.Pipeline.Convert.Enabled {
		t.Error("Expected convert stage to be enabled")
	}

	if !builder.manifest.Pipeline.Validate.Enabled {
		t.Error("Expected validate stage to be enabled")
	}
}

func TestBuilderBuild(t *testing.T) {
	tmpDir := t.TempDir()
	diskPath := filepath.Join(tmpDir, "test.vmdk")
	os.WriteFile(diskPath, []byte("test"), 0644)

	builder := NewBuilder()
	builder.WithSource("vsphere", "vm-1234", "test-vm", "DC1", "govc-export")
	builder.AddDisk("disk-0", "vmdk", diskPath, 1024, 0, "boot")
	builder.WithPipeline(true, true, true, true)

	manifest, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	if manifest == nil {
		t.Fatal("Build() returned nil manifest")
	}

	if manifest.ManifestVersion != CurrentVersion {
		t.Errorf("Expected version %q, got %q", CurrentVersion, manifest.ManifestVersion)
	}

	if len(manifest.Disks) != 1 {
		t.Errorf("Expected 1 disk, got %d", len(manifest.Disks))
	}
}

func TestBuilderBuildNoDisks(t *testing.T) {
	builder := NewBuilder()

	_, err := builder.Build()
	if err == nil {
		t.Error("Expected error when building with no disks, got nil")
	}

	if !strings.Contains(err.Error(), "at least one disk") {
		t.Errorf("Expected 'at least one disk' error, got: %v", err)
	}
}

func TestBuilderBuildWithErrors(t *testing.T) {
	tmpDir := t.TempDir()
	diskPath := filepath.Join(tmpDir, "test.vmdk")
	os.WriteFile(diskPath, []byte("test"), 0644)

	builder := NewBuilder()
	builder.AddDisk("invalid id with spaces", "vmdk", diskPath, 1024, 0, "boot")

	_, err := builder.Build()
	if err == nil {
		t.Error("Expected error when building with invalid disk ID, got nil")
	}

	if !strings.Contains(err.Error(), "invalid disk ID") {
		t.Errorf("Expected 'invalid disk ID' error, got: %v", err)
	}
}

func TestValidate(t *testing.T) {
	tmpDir := t.TempDir()
	diskPath := filepath.Join(tmpDir, "test.vmdk")
	os.WriteFile(diskPath, []byte("test"), 0644)

	builder := NewBuilder()
	builder.AddDisk("disk-0", "vmdk", diskPath, 1024, 0, "boot")

	manifest, err := builder.Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	if err := Validate(manifest); err != nil {
		t.Errorf("Validate() failed: %v", err)
	}
}

func TestValidateInvalidVersion(t *testing.T) {
	tmpDir := t.TempDir()
	diskPath := filepath.Join(tmpDir, "test.vmdk")
	os.WriteFile(diskPath, []byte("test"), 0644)

	builder := NewBuilder()
	builder.AddDisk("disk-0", "vmdk", diskPath, 1024, 0, "boot")

	manifest, _ := builder.Build()
	manifest.ManifestVersion = "99.0"

	err := Validate(manifest)
	if err == nil {
		t.Error("Expected error for invalid version, got nil")
	}

	if !strings.Contains(err.Error(), "unsupported manifest version") {
		t.Errorf("Expected 'unsupported manifest version' error, got: %v", err)
	}
}

func TestSerializeJSON(t *testing.T) {
	tmpDir := t.TempDir()
	diskPath := filepath.Join(tmpDir, "test.vmdk")
	os.WriteFile(diskPath, []byte("test"), 0644)

	builder := NewBuilder()
	builder.WithSource("vsphere", "vm-1234", "test-vm", "DC1", "govc-export")
	builder.AddDisk("disk-0", "vmdk", diskPath, 1024, 0, "boot")

	manifest, _ := builder.Build()

	data, err := ToJSON(manifest)
	if err != nil {
		t.Fatalf("ToJSON() failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("ToJSON() returned empty data")
	}

	// Verify it's valid JSON
	if !strings.Contains(string(data), "manifest_version") {
		t.Error("JSON output doesn't contain manifest_version")
	}
}

func TestWriteAndReadFile(t *testing.T) {
	tmpDir := t.TempDir()
	diskPath := filepath.Join(tmpDir, "test.vmdk")
	os.WriteFile(diskPath, []byte("test"), 0644)

	manifestPath := filepath.Join(tmpDir, "manifest.json")

	builder := NewBuilder()
	builder.WithSource("vsphere", "vm-1234", "test-vm", "DC1", "govc-export")
	builder.AddDisk("disk-0", "vmdk", diskPath, 1024, 0, "boot")

	originalManifest, _ := builder.Build()

	// Write to file
	if err := WriteToFile(originalManifest, manifestPath); err != nil {
		t.Fatalf("WriteToFile() failed: %v", err)
	}

	// Read back
	loadedManifest, err := ReadFromFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFromFile() failed: %v", err)
	}

	if loadedManifest.ManifestVersion != originalManifest.ManifestVersion {
		t.Error("Loaded manifest version doesn't match original")
	}

	if len(loadedManifest.Disks) != len(originalManifest.Disks) {
		t.Errorf("Expected %d disks, got %d", len(originalManifest.Disks), len(loadedManifest.Disks))
	}
}

func TestComputeSHA256(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	testData := []byte("test data for checksum")
	os.WriteFile(filePath, testData, 0644)

	checksum, err := ComputeSHA256(filePath)
	if err != nil {
		t.Fatalf("ComputeSHA256() failed: %v", err)
	}

	if len(checksum) != 64 {
		t.Errorf("Expected checksum length 64, got %d", len(checksum))
	}

	// Verify it's hexadecimal
	for _, c := range checksum {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Checksum contains invalid character: %c", c)
		}
	}
}

func TestBuilderMetadata(t *testing.T) {
	builder := NewBuilder()
	tags := map[string]string{
		"env": "production",
		"app": "webserver",
	}
	builder.WithMetadata("0.1.0", "job-123", tags)

	if builder.manifest.Metadata.HyperSDKVersion != "0.1.0" {
		t.Errorf("Expected hypersdk version %q, got %q", "0.1.0", builder.manifest.Metadata.HyperSDKVersion)
	}

	if builder.manifest.Metadata.JobID != "job-123" {
		t.Errorf("Expected job ID %q, got %q", "job-123", builder.manifest.Metadata.JobID)
	}

	if builder.manifest.Metadata.Tags["env"] != "production" {
		t.Error("Tags not set correctly")
	}
}

func TestBuilderChaining(t *testing.T) {
	tmpDir := t.TempDir()
	diskPath := filepath.Join(tmpDir, "test.vmdk")
	os.WriteFile(diskPath, []byte("test"), 0644)

	// Test method chaining
	manifest, err := NewBuilder().
		WithSource("vsphere", "vm-1234", "test-vm", "DC1", "govc-export").
		WithVM(4, 16, "uefi", "linux", "Ubuntu 22.04", false).
		AddDisk("disk-0", "vmdk", diskPath, 1024, 0, "boot").
		AddNIC("eth0", "00:50:56:ab:cd:ef", "VM Network").
		AddNote("Exported from vSphere 7.0").
		WithPipeline(true, true, true, true).
		WithMetadata("0.1.0", "job-123", map[string]string{"env": "prod"}).
		Build()

	if err != nil {
		t.Fatalf("Chained build failed: %v", err)
	}

	if manifest.Source == nil || manifest.Source.Provider != "vsphere" {
		t.Error("Source not set correctly in chained build")
	}

	if manifest.VM == nil || manifest.VM.CPU != 4 {
		t.Error("VM not set correctly in chained build")
	}

	if len(manifest.Disks) != 1 {
		t.Error("Disk not added correctly in chained build")
	}

	if len(manifest.NICs) != 1 {
		t.Error("NIC not added correctly in chained build")
	}

	if len(manifest.Notes) != 1 {
		t.Error("Note not added correctly in chained build")
	}

	if manifest.Pipeline == nil {
		t.Error("Pipeline not set correctly in chained build")
	}

	if manifest.Metadata == nil || manifest.Metadata.HyperSDKVersion != "0.1.0" {
		t.Error("Metadata not set correctly in chained build")
	}
}

func TestTimestamps(t *testing.T) {
	builder := NewBuilder()

	// Check that created_at is set
	if builder.manifest.Metadata.CreatedAt == nil {
		t.Error("Expected metadata.created_at to be set, got nil")
	}

	// Check that timestamp is recent
	now := time.Now()
	diff := now.Sub(*builder.manifest.Metadata.CreatedAt)
	if diff > time.Second {
		t.Errorf("Timestamp is too old: %v", diff)
	}

	// Add a warning and check its timestamp
	builder.AddWarning("export", "test warning")
	if len(builder.manifest.Warnings) > 0 {
		if builder.manifest.Warnings[0].Timestamp == nil {
			t.Error("Expected warning timestamp to be set, got nil")
		}
	}
}

func TestToYAML(t *testing.T) {
	// Create a simple manifest to test YAML serialization
	manifest := &ArtifactManifest{
		ManifestVersion: "1.0.0",
		Source: &SourceMetadata{
			Provider: "vsphere",
			VMName:   "TestVM",
		},
		VM: &VMMetadata{
			CPU:   4,
			MemGB: 16,
		},
	}

	// Convert to YAML
	yamlBytes, err := ToYAML(manifest)
	if err != nil {
		t.Fatalf("Failed to convert to YAML: %v", err)
	}

	if len(yamlBytes) == 0 {
		t.Error("Expected non-empty YAML output")
	}

	// Verify YAML contains expected content
	yamlStr := string(yamlBytes)
	if !strings.Contains(yamlStr, "TestVM") {
		t.Error("YAML should contain VM name")
	}
	if !strings.Contains(yamlStr, "vsphere") {
		t.Error("YAML should contain source provider")
	}
}

func TestFromYAML(t *testing.T) {
	// Create a test YAML
	yamlContent := `manifest_version: "1.0"
metadata:
  created_at: 2024-01-01T00:00:00Z
source:
  provider: vsphere
  vm_id: vm-1234
  vm_name: TestVM
  datacenter: DC1
vm:
  cpu: 4
  mem_gb: 16
  firmware: uefi
output:
  directory: /output
  format: qcow2
  filename: test.qcow2
`

	manifest, err := FromYAML([]byte(yamlContent))
	if err != nil {
		t.Fatalf("Failed to parse YAML: %v", err)
	}

	if manifest.Source.VMName != "TestVM" {
		t.Errorf("Expected VM name 'TestVM', got '%s'", manifest.Source.VMName)
	}

	if manifest.Source.Provider != "vsphere" {
		t.Errorf("Expected provider 'vsphere', got '%s'", manifest.Source.Provider)
	}

	if manifest.Output.Format != "qcow2" {
		t.Errorf("Expected format 'qcow2', got '%s'", manifest.Output.Format)
	}
}

func TestFromYAML_InvalidYAML(t *testing.T) {
	invalidYAML := []byte("invalid: yaml: content: :\n")

	_, err := FromYAML(invalidYAML)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestWithOutput(t *testing.T) {
	builder := NewBuilder()
	builder.WithOutput("/output", "vmdk", "test.vmdk")

	// Check the output was set in the manifest (don't call Build which requires disks)
	if builder.manifest.Output == nil {
		t.Fatal("Expected Output to be set")
	}

	if builder.manifest.Output.Format != "vmdk" {
		t.Errorf("Expected output format 'vmdk', got '%s'", builder.manifest.Output.Format)
	}

	if builder.manifest.Output.Directory != "/output" {
		t.Errorf("Expected output directory '/output', got '%s'", builder.manifest.Output.Directory)
	}

	if builder.manifest.Output.Filename != "test.vmdk" {
		t.Errorf("Expected output filename 'test.vmdk', got '%s'", builder.manifest.Output.Filename)
	}
}

func TestWithOptions(t *testing.T) {
	builder := NewBuilder()
	builder.WithOptions(true, 2)

	// Check the options were set in the manifest (don't call Build which requires disks)
	if builder.manifest.Options == nil {
		t.Fatal("Expected options to be set")
	}

	// Verify options are set correctly
	if !builder.manifest.Options.DryRun {
		t.Error("Expected DryRun option to be true")
	}

	if builder.manifest.Options.Verbose != 2 {
		t.Errorf("Expected Verbose to be 2, got %d", builder.manifest.Options.Verbose)
	}

	// Verify report config is initialized
	if builder.manifest.Options.Report == nil {
		t.Error("Expected Report config to be initialized")
	}
}

func TestWithOptions_FalseValues(t *testing.T) {
	builder := NewBuilder()
	builder.WithOptions(false, 0)

	// Check the options were set
	if builder.manifest.Options == nil {
		t.Fatal("Expected options to be set")
	}

	if builder.manifest.Options.DryRun {
		t.Error("Expected DryRun to be false")
	}

	if builder.manifest.Options.Verbose != 0 {
		t.Errorf("Expected Verbose to be 0, got %d", builder.manifest.Options.Verbose)
	}
}
