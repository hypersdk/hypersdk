// SPDX-License-Identifier: LGPL-3.0-or-later

package common

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"hypersdk/logger"
)

func TestProgressTracker(t *testing.T) {
	tracker := NewProgressTracker()

	// Start a task
	info := tracker.StartTask("task-001", "web-server", "vsphere")

	if info.TaskID != "task-001" {
		t.Errorf("TaskID = %s, want task-001", info.TaskID)
	}

	if info.Status != StatusPending {
		t.Errorf("Status = %s, want %s", info.Status, StatusPending)
	}

	// Update status
	err := tracker.SetStatus("task-001", StatusExporting)
	if err != nil {
		t.Errorf("SetStatus failed: %v", err)
	}

	// Get progress
	progress, err := tracker.GetProgress("task-001")
	if err != nil {
		t.Errorf("GetProgress failed: %v", err)
	}

	if progress.Status != StatusExporting {
		t.Errorf("Status = %s, want %s", progress.Status, StatusExporting)
	}

	// Complete task
	err = tracker.CompleteTask("task-001")
	if err != nil {
		t.Errorf("CompleteTask failed: %v", err)
	}

	progress, _ = tracker.GetProgress("task-001")
	if progress.Status != StatusCompleted {
		t.Errorf("Status = %s, want %s", progress.Status, StatusCompleted)
	}

	if progress.Percentage != 100.0 {
		t.Errorf("Percentage = %.2f, want 100.00", progress.Percentage)
	}

	t.Log("✅ Progress tracker test passed")
}

func TestProgressTrackerMultipleTasks(t *testing.T) {
	tracker := NewProgressTracker()

	// Start multiple tasks
	tracker.StartTask("task-001", "vm-001", "vsphere")
	tracker.StartTask("task-002", "vm-002", "aws")
	tracker.StartTask("task-003", "vm-003", "azure")

	// Get all progress
	allProgress := tracker.GetAllProgress()
	if len(allProgress) != 3 {
		t.Errorf("Got %d tasks, want 3", len(allProgress))
	}

	// Complete one task
	tracker.CompleteTask("task-001")

	// Get all progress again
	allProgress = tracker.GetAllProgress()
	completedCount := 0
	for _, p := range allProgress {
		if p.Status == StatusCompleted {
			completedCount++
		}
	}

	if completedCount != 1 {
		t.Errorf("Completed count = %d, want 1", completedCount)
	}

	t.Log("✅ Multiple tasks test passed")
	t.Logf("   Total tasks: %d", len(allProgress))
	t.Logf("   Completed: %d", completedCount)
}

func TestProgressSubscription(t *testing.T) {
	tracker := NewProgressTracker()

	// Start a task
	tracker.StartTask("task-001", "vm-001", "vsphere")

	// Subscribe to updates
	ch := tracker.Subscribe("task-001")
	defer tracker.Unsubscribe("task-001", ch)

	// Update task in goroutine
	go func() {
		time.Sleep(10 * time.Millisecond)
		tracker.SetStatus("task-001", StatusExporting)
	}()

	// Wait for update
	select {
	case update := <-ch:
		if update.Status != StatusExporting {
			t.Errorf("Status = %s, want %s", update.Status, StatusExporting)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for update")
	}

	t.Log("✅ Progress subscription test passed")
}

func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()

	// Record migration start
	collector.RecordMigrationStart("vsphere")

	// Record successful migration
	collector.RecordMigrationSuccess(
		"vsphere",
		10*time.Minute, // export
		5*time.Minute,  // conversion
		2*time.Minute,  // upload
		1024*1024*100,  // 100 MB exported
		1024*1024*80,   // 80 MB converted
		1024*1024*80,   // 80 MB uploaded
	)

	// Get stats
	stats := collector.GetStats()

	migrations := stats["migrations"].(map[string]interface{})
	if migrations["total"].(int64) != 1 {
		t.Errorf("Total migrations = %d, want 1", migrations["total"])
	}

	if migrations["succeeded"].(int64) != 1 {
		t.Errorf("Succeeded migrations = %d, want 1", migrations["succeeded"])
	}

	if migrations["active"].(int64) != 0 {
		t.Errorf("Active migrations = %d, want 0", migrations["active"])
	}

	// Get Prometheus metrics
	metrics := collector.GetMetrics()
	if metrics == "" {
		t.Error("Metrics string is empty")
	}

	t.Log("✅ Metrics collector test passed")
	t.Logf("   Total migrations: %d", migrations["total"])
	t.Logf("   Success rate: %.2f%%", stats["success_rate"])
}

func TestMetricsCollectorMultipleMigrations(t *testing.T) {
	collector := NewMetricsCollector()

	// Record multiple migrations
	for i := 0; i < 10; i++ {
		collector.RecordMigrationStart("vsphere")
		if i < 9 {
			collector.RecordMigrationSuccess("vsphere", 10*time.Minute, 5*time.Minute, 2*time.Minute, 1024*1024*100, 1024*1024*80, 1024*1024*80)
		} else {
			collector.RecordMigrationFailure("vsphere")
		}
	}

	stats := collector.GetStats()
	migrations := stats["migrations"].(map[string]interface{})

	if migrations["total"].(int64) != 10 {
		t.Errorf("Total migrations = %d, want 10", migrations["total"])
	}

	if migrations["succeeded"].(int64) != 9 {
		t.Errorf("Succeeded migrations = %d, want 9", migrations["succeeded"])
	}

	if migrations["failed"].(int64) != 1 {
		t.Errorf("Failed migrations = %d, want 1", migrations["failed"])
	}

	successRate := stats["success_rate"].(float64)
	if successRate < 89 || successRate > 91 {
		t.Errorf("Success rate = %.2f, want ~90", successRate)
	}

	t.Log("✅ Multiple migrations metrics test passed")
	t.Logf("   Total: %d", migrations["total"])
	t.Logf("   Succeeded: %d", migrations["succeeded"])
	t.Logf("   Failed: %d", migrations["failed"])
	t.Logf("   Success rate: %.2f%%", successRate)
}

func TestWebhookNotifier(t *testing.T) {
	log := logger.New("info")

	config := &WebhookConfig{
		Type:       WebhookSlack,
		URL:        "https://hooks.slack.com/test",
		Enabled:    false, // Disabled for testing
		OnStart:    true,
		OnComplete: true,
		OnError:    true,
	}

	notifier := NewWebhookNotifier(config, log)

	if notifier.config.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", notifier.config.Timeout)
	}

	if notifier.config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", notifier.config.MaxRetries)
	}

	t.Log("✅ Webhook notifier test passed")
}

func TestWebhookPayloadFormatting(t *testing.T) {
	log := logger.New("info")

	config := &WebhookConfig{
		Type:          WebhookSlack,
		URL:           "https://hooks.slack.com/test",
		Enabled:       false,
		SlackUsername: "HyperSDK",
		SlackChannel:  "#migrations",
	}

	notifier := NewWebhookNotifier(config, log)

	event := &WebhookEvent{
		EventType: "complete",
		TaskID:    "task-001",
		VMName:    "web-server",
		Provider:  "vsphere",
		Status:    "completed",
		Message:   "Migration completed successfully",
		Timestamp: time.Now(),
		Duration:  15 * time.Minute,
	}

	payload := notifier.formatSlackPayload(event)

	if payload["username"] != "HyperSDK" {
		t.Errorf("Username = %v, want HyperSDK", payload["username"])
	}

	if payload["channel"] != "#migrations" {
		t.Errorf("Channel = %v, want #migrations", payload["channel"])
	}

	t.Log("✅ Webhook payload formatting test passed")
}

func TestAuditLogger(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	// Log migration start
	err = logger.LogMigrationStart("task-001", "web-server", "vsphere", "admin")
	if err != nil {
		t.Errorf("LogMigrationStart failed: %v", err)
	}

	// Log migration complete
	err = logger.LogMigrationComplete("task-001", "web-server", "vsphere", "admin", 15*time.Minute, map[string]interface{}{
		"bytes_transferred": 1024 * 1024 * 100,
	})
	if err != nil {
		t.Errorf("LogMigrationComplete failed: %v", err)
	}

	// Close logger to flush
	logger.Close()

	// Verify log file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("Audit log file not created")
	}

	// Query logs
	events, err := QueryAuditLogs(logPath, QueryOptions{
		TaskID: "task-001",
	})
	if err != nil {
		t.Errorf("QueryAuditLogs failed: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Got %d events, want 2", len(events))
	}

	t.Log("✅ Audit logger test passed")
	t.Logf("   Events logged: %d", len(events))
}

func TestAuditLogQuery(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "audit-query-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "audit.log")
	logger, err := NewAuditLogger(logPath)
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}

	// Log multiple events
	logger.LogMigrationStart("task-001", "vm-001", "vsphere", "admin")
	logger.LogMigrationComplete("task-001", "vm-001", "vsphere", "admin", 10*time.Minute, nil)
	logger.LogMigrationStart("task-002", "vm-002", "aws", "admin")
	logger.LogMigrationFailed("task-002", "vm-002", "aws", "admin", fmt.Errorf("network error"))

	logger.Close()

	// Query successful migrations
	successTrue := true
	events, _ := QueryAuditLogs(logPath, QueryOptions{
		Success: &successTrue,
	})

	// Should find start + complete for task-001, and start for task-002 (3 total)
	if len(events) < 2 {
		t.Errorf("Got %d successful events, want at least 2", len(events))
	}

	// Query by provider
	events, _ = QueryAuditLogs(logPath, QueryOptions{
		Provider: "vsphere",
	})

	if len(events) != 2 {
		t.Errorf("Got %d vsphere events, want 2", len(events))
	}

	t.Log("✅ Audit log query test passed")
	t.Logf("   Query results: %d events", len(events))
}

func TestDashboardProvider(t *testing.T) {
	tracker := NewProgressTracker()
	collector := NewMetricsCollector()

	// Create audit logger
	tmpDir, _ := os.MkdirTemp("", "dashboard-test-*")
	defer os.RemoveAll(tmpDir)
	audit, _ := NewAuditLogger(filepath.Join(tmpDir, "audit.log"))
	defer audit.Close()

	provider := NewDashboardProvider(tracker, collector, audit)

	// Add some data
	tracker.StartTask("task-001", "vm-001", "vsphere")
	tracker.StartTask("task-002", "vm-002", "aws")
	tracker.CompleteTask("task-001")

	collector.RecordMigrationStart("vsphere")
	collector.RecordMigrationSuccess("vsphere", 10*time.Minute, 5*time.Minute, 2*time.Minute, 1024*1024*100, 1024*1024*80, 1024*1024*80)

	// Get dashboard data
	data := provider.GetDashboardData()

	if data.Status == "" {
		t.Error("Status is empty")
	}

	if data.Metrics == nil {
		t.Error("Metrics is nil")
	}

	if data.Metrics.TotalMigrations != 1 {
		t.Errorf("TotalMigrations = %d, want 1", data.Metrics.TotalMigrations)
	}

	if len(data.ActiveTasks) != 1 {
		t.Errorf("ActiveTasks count = %d, want 1", len(data.ActiveTasks))
	}

	if len(data.RecentCompletions) != 1 {
		t.Errorf("RecentCompletions count = %d, want 1", len(data.RecentCompletions))
	}

	// Get health status
	health := provider.GetHealthStatus()
	if !health.Healthy {
		t.Error("Health check failed")
	}

	t.Log("✅ Dashboard provider test passed")
	t.Logf("   Status: %s", data.Status)
	t.Logf("   Active tasks: %d", len(data.ActiveTasks))
	t.Logf("   Recent completions: %d", len(data.RecentCompletions))
	t.Logf("   Health: %v", health.Healthy)
}

func TestWebhookManager(t *testing.T) {
	log := logger.New("info")

	configs := []*WebhookConfig{
		{
			Type:       WebhookSlack,
			URL:        "https://hooks.slack.com/test1",
			Enabled:    false, // Disabled for testing
			OnComplete: true,
		},
		{
			Type:       WebhookDiscord,
			URL:        "https://discord.com/api/webhooks/test2",
			Enabled:    false, // Disabled for testing
			OnError:    true,
		},
	}

	manager := NewWebhookManager(configs, log)

	if len(manager.notifiers) != 0 { // Both disabled
		t.Errorf("Notifiers count = %d, want 0 (both disabled)", len(manager.notifiers))
	}

	// Enable one
	configs[0].Enabled = true
	manager = NewWebhookManager(configs, log)

	if len(manager.notifiers) != 1 {
		t.Errorf("Notifiers count = %d, want 1", len(manager.notifiers))
	}

	t.Log("✅ Webhook manager test passed")
}
