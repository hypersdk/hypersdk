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

func TestMigrationOrchestrator_BasicMigration(t *testing.T) {
	log := logger.New("info")

	// Create orchestrator with all features enabled
	orchConfig := &OrchestratorConfig{
		EnableConversion:         true,
		EnableParallelConversion: true,
		EnableProgress:           true,
		EnableMetrics:            true,
		EnableAuditLogging:       true,
		AuditLogPath:             filepath.Join(t.TempDir(), "audit.log"),
	}

	orchestrator, err := NewMigrationOrchestrator(orchConfig, log)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orchestrator.Close()

	// Verify all components initialized
	if orchestrator.conversionManager == nil {
		t.Error("Conversion manager not initialized")
	}

	if orchestrator.progressTracker == nil {
		t.Error("Progress tracker not initialized")
	}

	if orchestrator.metricsCollector == nil {
		t.Error("Metrics collector not initialized")
	}

	if orchestrator.auditLogger == nil {
		t.Error("Audit logger not initialized")
	}

	t.Log("✅ Migration orchestrator created successfully with all components")
}

func TestMigrationOrchestrator_EndToEndFlow(t *testing.T) {
	log := logger.New("info")
	ctx := context.Background()

	tmpDir := t.TempDir()

	// Create orchestrator
	orchConfig := &OrchestratorConfig{
		EnableProgress:     true,
		EnableMetrics:      true,
		EnableAuditLogging: true,
		AuditLogPath:       filepath.Join(tmpDir, "audit.log"),
	}

	orchestrator, err := NewMigrationOrchestrator(orchConfig, log)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orchestrator.Close()

	// Create migration config
	migrationConfig := &MigrationConfig{
		VMName:             "test-vm-001",
		Provider:           "vsphere",
		OutputDir:          tmpDir,
		TargetFormat:       "qcow2",
		ExportManifest:     true,
		EnableConversion:   false, // Skip actual conversion in test
		EnableProgress:     true,
		EnableMetrics:      true,
		EnableAuditLogging: true,
		User:               "test-user",
		IPAddress:          "192.168.1.100",
	}

	// Simulate migration (without actual export/conversion)
	result, err := orchestrator.Migrate(ctx, migrationConfig)
	if err == nil {
		// Expected to fail since we're not doing actual export
		// But we can verify the orchestration flow works
		t.Logf("Migration result: %+v", result)
	}

	// Verify progress tracking
	if orchestrator.progressTracker != nil {
		allProgress := orchestrator.progressTracker.GetAllProgress()
		if len(allProgress) == 0 {
			t.Error("No progress tracked")
		} else {
			t.Logf("✅ Progress tracked for %d tasks", len(allProgress))
		}
	}

	// Verify metrics collection
	if orchestrator.metricsCollector != nil {
		stats := orchestrator.metricsCollector.GetStats()
		migrations := stats["migrations"].(map[string]interface{})
		if migrations["total"].(int64) == 0 {
			t.Error("No migrations recorded in metrics")
		} else {
			t.Logf("✅ Metrics recorded: %d total migrations", migrations["total"])
		}
	}

	// Verify audit logging
	if orchestrator.auditLogger != nil {
		events, err := QueryAuditLogs(orchConfig.AuditLogPath, QueryOptions{
			VMName: "test-vm-001",
		})
		if err != nil {
			t.Errorf("Failed to query audit logs: %v", err)
		} else if len(events) > 0 {
			t.Logf("✅ Audit logging working: %d events logged", len(events))
		}
	}

	t.Log("✅ End-to-end orchestration flow test passed")
}

func TestMigrationOrchestrator_ProgressTracking(t *testing.T) {
	log := logger.New("info")

	orchConfig := &OrchestratorConfig{
		EnableProgress: true,
	}

	orchestrator, err := NewMigrationOrchestrator(orchConfig, log)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	tracker := orchestrator.GetProgressTracker()
	if tracker == nil {
		t.Fatal("Progress tracker not available")
	}

	// Start a task
	taskID := "test-task-001"
	tracker.StartTask(taskID, "test-vm", "vsphere")

	// Update progress through different stages
	tracker.SetStatus(taskID, StatusExporting)
	tracker.SetStatus(taskID, StatusConverting)
	tracker.SetStatus(taskID, StatusUploading)

	tracker.CompleteTask(taskID)

	// Verify final state
	progress, err := tracker.GetProgress(taskID)
	if err != nil {
		t.Fatalf("Failed to get progress: %v", err)
	}

	if progress.Status != StatusCompleted {
		t.Errorf("Status = %s, want %s", progress.Status, StatusCompleted)
	}

	if progress.Percentage != 100.0 {
		t.Errorf("Percentage = %.2f, want 100.00", progress.Percentage)
	}

	t.Log("✅ Progress tracking through all stages successful")
	t.Logf("   Final status: %s", progress.Status)
	t.Logf("   Percentage: %.2f%%", progress.Percentage)
}

func TestMigrationOrchestrator_MetricsCollection(t *testing.T) {
	log := logger.New("info")

	orchConfig := &OrchestratorConfig{
		EnableMetrics: true,
	}

	orchestrator, err := NewMigrationOrchestrator(orchConfig, log)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	collector := orchestrator.GetMetricsCollector()
	if collector == nil {
		t.Fatal("Metrics collector not available")
	}

	// Simulate multiple migrations
	for i := 0; i < 5; i++ {
		collector.RecordMigrationStart("vsphere")
		collector.RecordMigrationSuccess(
			"vsphere",
			10*time.Minute,
			5*time.Minute,
			2*time.Minute,
			1024*1024*1000,
			1024*1024*800,
			1024*1024*800,
		)
	}

	// One failure
	collector.RecordMigrationStart("vsphere")
	collector.RecordMigrationFailure("vsphere")

	// Get stats
	stats := collector.GetStats()
	migrations := stats["migrations"].(map[string]interface{})

	if migrations["total"].(int64) != 6 {
		t.Errorf("Total migrations = %d, want 6", migrations["total"])
	}

	if migrations["succeeded"].(int64) != 5 {
		t.Errorf("Succeeded migrations = %d, want 5", migrations["succeeded"])
	}

	if migrations["failed"].(int64) != 1 {
		t.Errorf("Failed migrations = %d, want 1", migrations["failed"])
	}

	successRate := stats["success_rate"].(float64)
	expectedRate := 5.0 / 6.0 * 100.0
	if successRate < expectedRate-1 || successRate > expectedRate+1 {
		t.Errorf("Success rate = %.2f, want ~%.2f", successRate, expectedRate)
	}

	t.Log("✅ Metrics collection test passed")
	t.Logf("   Total: %d", migrations["total"])
	t.Logf("   Succeeded: %d", migrations["succeeded"])
	t.Logf("   Failed: %d", migrations["failed"])
	t.Logf("   Success rate: %.2f%%", successRate)
}

func TestMigrationOrchestrator_AuditLogging(t *testing.T) {
	log := logger.New("info")
	tmpDir := t.TempDir()
	auditPath := filepath.Join(tmpDir, "audit.log")

	orchConfig := &OrchestratorConfig{
		EnableAuditLogging: true,
		AuditLogPath:       auditPath,
	}

	orchestrator, err := NewMigrationOrchestrator(orchConfig, log)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orchestrator.Close()

	auditLogger := orchestrator.GetAuditLogger()
	if auditLogger == nil {
		t.Fatal("Audit logger not available")
	}

	// Log complete migration lifecycle
	taskID := "audit-test-001"
	vmName := "test-vm"
	provider := "vsphere"
	user := "admin"

	auditLogger.LogMigrationStart(taskID, vmName, provider, user)
	auditLogger.LogExportStart(taskID, vmName, provider)
	auditLogger.LogExportComplete(taskID, vmName, provider, 5*time.Minute, 1024*1024*1000)
	auditLogger.LogConversionStart(taskID, vmName)
	auditLogger.LogConversionComplete(taskID, vmName, 3*time.Minute, []string{"disk1.qcow2"})
	auditLogger.LogMigrationComplete(taskID, vmName, provider, user, 8*time.Minute, map[string]interface{}{
		"success": true,
	})

	// Flush
	auditLogger.Close()

	// Query logs
	events, err := QueryAuditLogs(auditPath, QueryOptions{
		TaskID: taskID,
	})
	if err != nil {
		t.Fatalf("Failed to query audit logs: %v", err)
	}

	if len(events) != 6 {
		t.Errorf("Got %d events, want 6", len(events))
	}

	// Verify event types
	expectedTypes := []AuditEventType{
		EventMigrationStart,
		EventExportStart,
		EventExportComplete,
		EventConversionStart,
		EventConversionComplete,
		EventMigrationComplete,
	}

	for i, event := range events {
		if event.EventType != expectedTypes[i] {
			t.Errorf("Event %d type = %s, want %s", i, event.EventType, expectedTypes[i])
		}
	}

	t.Log("✅ Audit logging test passed")
	t.Logf("   Events logged: %d", len(events))
	t.Logf("   Complete lifecycle tracked")
}

func TestMigrationOrchestrator_WebhookNotifications(t *testing.T) {
	log := logger.New("info")

	webhookConfigs := []*WebhookConfig{
		{
			Type:       WebhookGeneric,
			URL:        "http://localhost:9999/webhook",
			Enabled:    false, // Disabled for testing
			OnStart:    true,
			OnComplete: true,
			OnError:    true,
		},
	}

	orchConfig := &OrchestratorConfig{
		EnableWebhooks: true,
		WebhookConfigs: webhookConfigs,
	}

	orchestrator, err := NewMigrationOrchestrator(orchConfig, log)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	if orchestrator.webhookManager == nil {
		t.Fatal("Webhook manager not initialized")
	}

	// Webhooks are disabled, so no actual HTTP calls will be made
	// But we can verify the manager was created

	t.Log("✅ Webhook notifications setup successful")
}

func TestMigrationOrchestrator_BatchMigration(t *testing.T) {
	log := logger.New("info")
	ctx := context.Background()
	tmpDir := t.TempDir()

	orchConfig := &OrchestratorConfig{
		EnableProgress:           true,
		EnableMetrics:            true,
		EnableBatchOrchestration: true,
	}

	orchestrator, err := NewMigrationOrchestrator(orchConfig, log)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	// Create batch of migration configs
	configs := []*MigrationConfig{
		{
			VMName:           "vm-001",
			Provider:         "vsphere",
			OutputDir:        filepath.Join(tmpDir, "vm-001"),
			TargetFormat:     "qcow2",
			EnableConversion: false,
			User:             "admin",
		},
		{
			VMName:           "vm-002",
			Provider:         "vsphere",
			OutputDir:        filepath.Join(tmpDir, "vm-002"),
			TargetFormat:     "qcow2",
			EnableConversion: false,
			User:             "admin",
		},
		{
			VMName:           "vm-003",
			Provider:         "vsphere",
			OutputDir:        filepath.Join(tmpDir, "vm-003"),
			TargetFormat:     "qcow2",
			EnableConversion: false,
			User:             "admin",
		},
	}

	// Run batch migration
	results, err := orchestrator.MigrateBatch(ctx, configs)
	if err != nil {
		t.Logf("Batch migration error (expected in test): %v", err)
	}

	if len(results) != len(configs) {
		t.Errorf("Got %d results, want %d", len(results), len(configs))
	}

	t.Log("✅ Batch migration orchestration test passed")
	t.Logf("   Migrations attempted: %d", len(results))
}

func TestMigrationOrchestrator_ComponentIntegration(t *testing.T) {
	log := logger.New("info")
	tmpDir := t.TempDir()

	// Test that all Phase 0-5 components integrate correctly
	orchConfig := &OrchestratorConfig{
		EnableConversion:         true,  // Phase 3
		EnableParallelConversion: true,  // Phase 4
		EnableCloudStorage:       false, // Phase 4 (disabled, would need actual cloud creds)
		EnableBatchOrchestration: true,  // Phase 4
		EnableProgress:           true,  // Phase 5
		EnableMetrics:            true,  // Phase 5
		EnableAuditLogging:       true,  // Phase 5
		EnableWebhooks:           false, // Phase 5 (disabled for test)
		AuditLogPath:             filepath.Join(tmpDir, "audit.log"),
	}

	orchestrator, err := NewMigrationOrchestrator(orchConfig, log)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orchestrator.Close()

	// Verify Phase 3 components
	if orchestrator.conversionManager == nil {
		t.Error("Phase 3: Conversion manager not initialized")
	} else {
		t.Log("✅ Phase 3: Conversion manager initialized")
	}

	// Verify Phase 4 components
	if orchestrator.parallelConverter == nil {
		t.Error("Phase 4: Parallel converter not initialized")
	} else {
		t.Log("✅ Phase 4: Parallel converter initialized")
	}

	if orchestrator.batchOrchestrator == nil {
		t.Error("Phase 4: Batch orchestrator not initialized")
	} else {
		t.Log("✅ Phase 4: Batch orchestrator initialized")
	}

	// Verify Phase 5 components
	if orchestrator.progressTracker == nil {
		t.Error("Phase 5: Progress tracker not initialized")
	} else {
		t.Log("✅ Phase 5: Progress tracker initialized")
	}

	if orchestrator.metricsCollector == nil {
		t.Error("Phase 5: Metrics collector not initialized")
	} else {
		t.Log("✅ Phase 5: Metrics collector initialized")
	}

	if orchestrator.auditLogger == nil {
		t.Error("Phase 5: Audit logger not initialized")
	} else {
		t.Log("✅ Phase 5: Audit logger initialized")
	}

	t.Log("\n✅ All Phase 0-5 components integrated successfully!")
}

func TestMigrationOrchestrator_FailureHandling(t *testing.T) {
	log := logger.New("info")
	ctx := context.Background()
	tmpDir := t.TempDir()

	orchConfig := &OrchestratorConfig{
		EnableProgress:     true,
		EnableMetrics:      true,
		EnableAuditLogging: true,
		AuditLogPath:       filepath.Join(tmpDir, "audit.log"),
		EnableWebhooks:     false,
	}

	orchestrator, err := NewMigrationOrchestrator(orchConfig, log)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}
	defer orchestrator.Close()

	// Create config that will fail
	migrationConfig := &MigrationConfig{
		VMName:             "nonexistent-vm",
		Provider:           "vsphere",
		OutputDir:          "/nonexistent/path",
		EnableConversion:   true,
		EnableProgress:     true,
		EnableMetrics:      true,
		EnableAuditLogging: true,
		User:               "test-user",
	}

	// Attempt migration (should fail)
	result, err := orchestrator.Migrate(ctx, migrationConfig)
	if err == nil {
		t.Error("Expected migration to fail, but it succeeded")
	}

	if result == nil {
		t.Fatal("Result should not be nil even on failure")
	}

	if result.Success {
		t.Error("Result.Success should be false")
	}

	if result.Error == "" {
		t.Error("Result.Error should be set")
	}

	// Verify failure was tracked
	if orchestrator.metricsCollector != nil {
		stats := orchestrator.metricsCollector.GetStats()
		migrations := stats["migrations"].(map[string]interface{})
		if migrations["failed"].(int64) == 0 {
			t.Error("Failure not recorded in metrics")
		}
	}

	// Verify failure was logged
	if orchestrator.auditLogger != nil {
		orchestrator.auditLogger.Close()
		events, err := QueryAuditLogs(orchConfig.AuditLogPath, QueryOptions{
			VMName: "nonexistent-vm",
		})
		if err != nil {
			t.Errorf("Failed to query audit logs: %v", err)
		}

		failureLogged := false
		for _, event := range events {
			if event.EventType == EventMigrationFailed {
				failureLogged = true
				break
			}
		}

		if !failureLogged {
			t.Error("Failure not logged in audit log")
		}
	}

	t.Log("✅ Failure handling test passed")
	t.Logf("   Error: %s", result.Error)
}

func TestMigrationOrchestrator_ConversionOptions(t *testing.T) {
	log := logger.New("info")

	orchConfig := &OrchestratorConfig{
		EnableConversion: true,
	}

	orchestrator, err := NewMigrationOrchestrator(orchConfig, log)
	if err != nil {
		t.Fatalf("Failed to create orchestrator: %v", err)
	}

	// Verify conversion manager was created
	if orchestrator.conversionManager == nil {
		t.Error("Conversion manager not initialized")
	}

	t.Log("✅ Conversion options test passed")
}

func TestMigrationResult_Structure(t *testing.T) {
	result := &MigrationResult{
		TaskID:    "task-001",
		VMName:    "test-vm",
		Provider:  "vsphere",
		Success:   true,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(10 * time.Minute),

		ExportDuration: 5 * time.Minute,
		ExportedFiles:  []string{"disk1.vmdk", "vm.ovf"},
		ExportSize:     1024 * 1024 * 1000,

		ConversionDuration: 3 * time.Minute,
		ConvertedFiles:     []string{"disk1.qcow2"},
		ConversionSize:     1024 * 1024 * 800,

		UploadDuration:   2 * time.Minute,
		UploadedFiles:    []string{"disk1.qcow2"},
		CloudDestination: "s3://bucket/vm",

		TotalDuration: 10 * time.Minute,
		TotalSize:     1024 * 1024 * 1800,
	}

	// Verify structure
	if result.TaskID == "" {
		t.Error("TaskID should be set")
	}

	if result.VMName == "" {
		t.Error("VMName should be set")
	}

	if !result.Success {
		t.Error("Success should be true")
	}

	if len(result.ExportedFiles) != 2 {
		t.Errorf("ExportedFiles count = %d, want 2", len(result.ExportedFiles))
	}

	if len(result.ConvertedFiles) != 1 {
		t.Errorf("ConvertedFiles count = %d, want 1", len(result.ConvertedFiles))
	}

	if result.TotalDuration != 10*time.Minute {
		t.Errorf("TotalDuration = %v, want 10m", result.TotalDuration)
	}

	t.Log("✅ Migration result structure test passed")
	t.Logf("   Task ID: %s", result.TaskID)
	t.Logf("   Total duration: %v", result.TotalDuration)
	t.Logf("   Total size: %d bytes", result.TotalSize)
}

// Helper function to verify file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
