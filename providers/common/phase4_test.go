// SPDX-License-Identifier: LGPL-3.0-or-later

package common

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hypersdk/logger"
)

// Mock converter for testing
type mockConverter struct{}

func (mc *mockConverter) Convert(ctx context.Context, manifestPath string, opts ConvertOptions) (*ConversionResult, error) {
	// Simulate conversion
	time.Sleep(10 * time.Millisecond)

	return &ConversionResult{
		Success: true,
		ConvertedFiles: []string{
			"/tmp/disk-0.qcow2",
			"/tmp/disk-1.qcow2",
		},
		Duration: 10 * time.Millisecond,
	}, nil
}

func (mc *mockConverter) GetVersion() (string, error) {
	return "mock v1.0.0", nil
}

func (mc *mockConverter) Validate() error {
	return nil
}

func TestParallelConverter(t *testing.T) {
	log := logger.New("info")
	converter := &mockConverter{}
	pc := NewParallelConverter(converter, 2, log)

	if pc.maxParallel != 2 {
		t.Errorf("maxParallel = %d, want 2", pc.maxParallel)
	}

	t.Log("✅ ParallelConverter created successfully")
}

func TestParallelConversion(t *testing.T) {
	log := logger.New("info")
	converter := &mockConverter{}
	pc := NewParallelConverter(converter, 2, log)

	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "parallel-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create mock manifest files
	var tasks []*DiskConversionTask
	for i := 0; i < 4; i++ {
		manifestPath := filepath.Join(tmpDir, "manifest-"+string(rune('0'+i))+".json")
		os.WriteFile(manifestPath, []byte("{}"), 0644)

		tasks = append(tasks, &DiskConversionTask{
			ManifestPath: manifestPath,
			DiskIndex:    i,
			Options: ConvertOptions{
				StreamOutput: false,
			},
		})
	}

	// Run parallel conversion
	ctx := context.Background()
	results, err := pc.ConvertParallel(ctx, tasks)
	if err != nil {
		t.Fatalf("ParallelConversion failed: %v", err)
	}

	if len(results) != 4 {
		t.Errorf("Got %d results, want 4", len(results))
	}

	// Check all succeeded
	for _, result := range results {
		if result.Error != nil {
			t.Errorf("Task %d failed: %v", result.DiskIndex, result.Error)
		}
	}

	// Get stats
	stats := GetStats(results)
	if stats.TotalTasks != 4 {
		t.Errorf("TotalTasks = %d, want 4", stats.TotalTasks)
	}
	if stats.SuccessfulTasks != 4 {
		t.Errorf("SuccessfulTasks = %d, want 4", stats.SuccessfulTasks)
	}

	t.Log("✅ Parallel conversion test passed")
	t.Logf("   Total tasks: %d", stats.TotalTasks)
	t.Logf("   Successful: %d", stats.SuccessfulTasks)
	t.Logf("   Average duration: %v", stats.AverageDuration)
}

func TestPipelineConfig(t *testing.T) {
	// Test default config
	cfg := NewDefaultPipelineConfig()

	if cfg.Name != "default" {
		t.Errorf("Name = %s, want 'default'", cfg.Name)
	}

	if !cfg.IsStageEnabled(StageInspect) {
		t.Error("Inspect stage should be enabled")
	}

	if !cfg.IsStageEnabled(StageConvert) {
		t.Error("Convert stage should be enabled")
	}

	// Test stage options
	convertOpts := cfg.GetStageOptions(StageConvert)
	if convertOpts == nil {
		t.Error("Convert stage options should not be nil")
	}

	targetFormat, ok := cfg.GetStageOption(StageConvert, "target_format")
	if !ok {
		t.Error("target_format option not found")
	}
	if targetFormat != "qcow2" {
		t.Errorf("target_format = %v, want 'qcow2'", targetFormat)
	}

	// Test enable/disable stage
	cfg.DisableStage(StageInspect)
	if cfg.IsStageEnabled(StageInspect) {
		t.Error("Inspect stage should be disabled")
	}

	cfg.EnableStage(StageInspect)
	if !cfg.IsStageEnabled(StageInspect) {
		t.Error("Inspect stage should be enabled")
	}

	// Test validation
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	t.Log("✅ Pipeline config test passed")
}

func TestMinimalPipelineConfig(t *testing.T) {
	cfg := NewMinimalPipelineConfig()

	if cfg.IsStageEnabled(StageInspect) {
		t.Error("Inspect stage should be disabled in minimal config")
	}

	if cfg.IsStageEnabled(StageFix) {
		t.Error("Fix stage should be disabled in minimal config")
	}

	if !cfg.IsStageEnabled(StageConvert) {
		t.Error("Convert stage should be enabled in minimal config")
	}

	if cfg.IsStageEnabled(StageValidate) {
		t.Error("Validate stage should be disabled in minimal config")
	}

	t.Log("✅ Minimal pipeline config test passed")
}

func TestOptimizedPipelineConfig(t *testing.T) {
	cfg := NewOptimizedPipelineConfig()

	if !cfg.IsStageEnabled(StageOptimize) {
		t.Error("Optimize stage should be enabled in optimized config")
	}

	if !cfg.IsStageEnabled(StageCompress) {
		t.Error("Compress stage should be enabled in optimized config")
	}

	t.Log("✅ Optimized pipeline config test passed")
}

func TestGuestConfig(t *testing.T) {
	cfg := NewDefaultGuestConfig()

	if cfg.Hostname != "migrated-vm" {
		t.Errorf("Hostname = %s, want 'migrated-vm'", cfg.Hostname)
	}

	if cfg.Timezone != "UTC" {
		t.Errorf("Timezone = %s, want 'UTC'", cfg.Timezone)
	}

	// Add a user
	cfg.Users = append(cfg.Users, &UserConfig{
		Username: "admin",
		Password: "password123",
		Sudo:     true,
		Groups:   []string{"sudo", "wheel"},
	})

	// Add SSH key
	cfg.SSHKeys = append(cfg.SSHKeys, &SSHKeyConfig{
		User:      "admin",
		PublicKey: "ssh-rsa AAAAB3... admin@example.com",
	})

	// Validate
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	// Test cloud-init generation
	cloudInit, err := cfg.ToCloudInit()
	if err != nil {
		t.Errorf("Cloud-init generation failed: %v", err)
	}

	if cloudInit == "" {
		t.Error("Cloud-init output is empty")
	}

	t.Log("✅ Guest config test passed")
	t.Logf("   Hostname: %s", cfg.Hostname)
	t.Logf("   Users: %d", len(cfg.Users))
	t.Logf("   SSH keys: %d", len(cfg.SSHKeys))
}

func TestNetworkConfig(t *testing.T) {
	cfg := &GuestConfig{
		Network: &NetworkConfig{
			Interfaces: []*NetworkInterface{
				{
					Name:      "eth0",
					Method:    "static",
					IPAddress: "192.168.1.100",
					Netmask:   "255.255.255.0",
					Gateway:   "192.168.1.1",
				},
			},
			DNSServers: []string{"8.8.8.8", "8.8.4.4"},
		},
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	t.Log("✅ Network config test passed")
}

func TestCloudStorageConfig(t *testing.T) {
	// Test S3 config
	s3Config := &CloudStorageConfig{
		Provider: ProviderS3,
		S3Config: &S3Config{
			Bucket: "my-bucket",
			Region: "us-east-1",
		},
	}

	if err := s3Config.Validate(); err != nil {
		t.Errorf("S3 config validation failed: %v", err)
	}

	// Test Azure config
	azureConfig := &CloudStorageConfig{
		Provider: ProviderAzure,
		AzureConfig: &AzureStorageConfig{
			AccountName: "myaccount",
			Container:   "mycontainer",
		},
	}

	if err := azureConfig.Validate(); err != nil {
		t.Errorf("Azure config validation failed: %v", err)
	}

	// Test GCS config
	gcsConfig := &CloudStorageConfig{
		Provider: ProviderGCS,
		GCSConfig: &GCSConfig{
			Bucket:    "my-bucket",
			ProjectID: "my-project",
		},
	}

	if err := gcsConfig.Validate(); err != nil {
		t.Errorf("GCS config validation failed: %v", err)
	}

	t.Log("✅ Cloud storage config test passed")
}

func TestBatchMigrationConfig(t *testing.T) {
	cfg := &BatchMigrationConfig{
		VMs: []*VMMigrationTask{
			{
				ID:       "vm-001",
				Name:     "web-server",
				Provider: "vsphere",
				Priority: 10,
			},
			{
				ID:       "vm-002",
				Name:     "db-server",
				Provider: "vsphere",
				Priority: 20,
			},
		},
		OutputDir:   "/tmp/batch-migration",
		MaxParallel: 2,
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	if len(cfg.VMs) != 2 {
		t.Errorf("VMs count = %d, want 2", len(cfg.VMs))
	}

	t.Log("✅ Batch migration config test passed")
	t.Logf("   VMs: %d", len(cfg.VMs))
	t.Logf("   Max parallel: %d", cfg.MaxParallel)
}

func TestBatchOrchestrator(t *testing.T) {
	log := logger.New("info")
	cfg := &BatchMigrationConfig{
		VMs: []*VMMigrationTask{
			{
				ID:       "vm-001",
				Name:     "test-vm",
				Provider: "vsphere",
			},
		},
		OutputDir:   "/tmp/batch-test",
		MaxParallel: 1,
	}

	orchestrator, err := NewBatchOrchestrator(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create batch orchestrator: %v", err)
	}

	if orchestrator.config.MaxParallel != 1 {
		t.Errorf("MaxParallel = %d, want 1", orchestrator.config.MaxParallel)
	}

	t.Log("✅ Batch orchestrator test passed")
}

func TestSortVMsByPriority(t *testing.T) {
	log := logger.New("info")
	cfg := &BatchMigrationConfig{
		VMs: []*VMMigrationTask{
			{ID: "vm-001", Name: "vm-001", Provider: "vsphere", Priority: 10},
			{ID: "vm-002", Name: "vm-002", Provider: "vsphere", Priority: 30},
			{ID: "vm-003", Name: "vm-003", Provider: "vsphere", Priority: 20},
		},
		OutputDir:   "/tmp/test",
		MaxParallel: 1,
	}

	orchestrator, err := NewBatchOrchestrator(cfg, log)
	if err != nil {
		t.Fatalf("Failed to create batch orchestrator: %v", err)
	}
	sorted := orchestrator.sortVMsByPriority()

	if sorted[0].ID != "vm-002" {
		t.Errorf("First VM should be vm-002 (priority 30), got %s", sorted[0].ID)
	}

	if sorted[1].ID != "vm-003" {
		t.Errorf("Second VM should be vm-003 (priority 20), got %s", sorted[1].ID)
	}

	if sorted[2].ID != "vm-001" {
		t.Errorf("Third VM should be vm-001 (priority 10), got %s", sorted[2].ID)
	}

	t.Log("✅ VM priority sorting test passed")
	for i, vm := range sorted {
		t.Logf("   %d: %s (priority %d)", i+1, vm.ID, vm.Priority)
	}
}
