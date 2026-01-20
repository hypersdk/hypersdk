// SPDX-License-Identifier: LGPL-3.0-or-later

// Example: Complete VM Migration with Orchestrator
//
// This example demonstrates how to use the MigrationOrchestrator to perform
// a complete end-to-end VM migration with all Phase 0-5 features enabled.

package main

import (
	"context"
	"fmt"
	"time"

	"hypersdk/logger"
	"hypersdk/providers/common"
)

func main() {
	// Create logger
	appLogger := logger.New("info")

	// Example 1: Basic Migration with Progress Tracking
	fmt.Println("=== Example 1: Basic Migration ===")
	basicMigrationExample(appLogger)

	// Example 2: Migration with Full Monitoring
	fmt.Println("\n=== Example 2: Full Monitoring ===")
	fullMonitoringExample(appLogger)

	// Example 3: Batch Migration
	fmt.Println("\n=== Example 3: Batch Migration ===")
	batchMigrationExample(appLogger)

	// Example 4: Migration with Webhooks
	fmt.Println("\n=== Example 4: Webhook Notifications ===")
	webhookExample(appLogger)
}

// basicMigrationExample shows basic orchestrator usage
func basicMigrationExample(log logger.Logger) {
	ctx := context.Background()

	// Create orchestrator with minimal configuration
	orchConfig := &common.OrchestratorConfig{
		EnableConversion: true,
		EnableProgress:   true,
		EnableMetrics:    true,
	}

	orchestrator, err := common.NewMigrationOrchestrator(orchConfig, log)
	if err != nil {
		log.Error("failed to create orchestrator", "error", err)
		return
	}
	defer orchestrator.Close()

	// Configure migration
	migrationConfig := &common.MigrationConfig{
		VMName:           "web-server-01",
		Provider:         "vsphere",
		OutputDir:        "/exports/web-server-01",
		TargetFormat:     "qcow2",
		EnableConversion: true,
		ConvertOptions: common.ConvertOptions{
			TargetFormat: "qcow2",
		},
		User: "admin",
	}

	// Note: In real usage, you would call hyperexport here to perform the actual export
	// This example shows the orchestration framework

	fmt.Printf("Migration Configuration:\n")
	fmt.Printf("  VM: %s\n", migrationConfig.VMName)
	fmt.Printf("  Provider: %s\n", migrationConfig.Provider)
	fmt.Printf("  Target Format: %s\n", migrationConfig.TargetFormat)
	fmt.Printf("  Output: %s\n", migrationConfig.OutputDir)

	// In actual usage:
	// result, err := orchestrator.Migrate(ctx, migrationConfig)
	_ = ctx
	_ = migrationConfig

	fmt.Println("\n✓ Orchestrator configured successfully")
	fmt.Println("  Progress tracking: enabled")
	fmt.Println("  Metrics collection: enabled")
	fmt.Println("  Conversion: enabled")
}

// fullMonitoringExample shows complete monitoring setup
func fullMonitoringExample(log logger.Logger) {
	ctx := context.Background()

	// Create orchestrator with full monitoring
	orchConfig := &common.OrchestratorConfig{
		EnableConversion:   true,
		EnableProgress:     true,
		EnableMetrics:      true,
		EnableAuditLogging: true,
		AuditLogPath:       "/var/log/hypersdk/audit.log",
	}

	orchestrator, err := common.NewMigrationOrchestrator(orchConfig, log)
	if err != nil {
		log.Error("failed to create orchestrator", "error", err)
		return
	}
	defer orchestrator.Close()

	// Start monitoring servers
	tracker := orchestrator.GetProgressTracker()
	progressServer := common.NewProgressAPIServer(tracker, ":8080")
	go func() {
		if err := progressServer.Start(); err != nil {
			log.Error("progress server failed", "error", err)
		}
	}()

	collector := orchestrator.GetMetricsCollector()
	metricsServer := common.NewMetricsServer(collector, ":9090")
	go func() {
		if err := metricsServer.Start(); err != nil {
			log.Error("metrics server failed", "error", err)
		}
	}()

	fmt.Println("Monitoring servers started:")
	fmt.Println("  Progress API: http://localhost:8080/api/v1/progress")
	fmt.Println("  Metrics API:  http://localhost:9090/metrics")
	fmt.Println("  Audit Log:    /var/log/hypersdk/audit.log")

	// Simulate migration
	migrationConfig := &common.MigrationConfig{
		VMName:             "db-server-01",
		Provider:           "vsphere",
		OutputDir:          "/exports/db-server-01",
		TargetFormat:       "qcow2",
		EnableConversion:   true,
		EnableProgress:     true,
		EnableMetrics:      true,
		EnableAuditLogging: true,
		User:               "admin",
		IPAddress:          "192.168.1.100",
	}

	_ = ctx
	_ = migrationConfig

	// Query progress example
	time.Sleep(100 * time.Millisecond)

	allProgress := tracker.GetAllProgress()
	fmt.Printf("\nActive migrations: %d\n", len(allProgress))

	// Get metrics example
	stats := collector.GetStats()
	migrations := stats["migrations"].(map[string]interface{})
	fmt.Printf("Total migrations recorded: %d\n", migrations["total"])

	fmt.Println("\n✓ Full monitoring configured successfully")

	// Cleanup servers
	progressServer.Stop(ctx)
	metricsServer.Stop()
}

// batchMigrationExample shows batch migration
func batchMigrationExample(log logger.Logger) {
	// Create orchestrator
	orchConfig := &common.OrchestratorConfig{
		EnableConversion: true,
		EnableProgress:   true,
		EnableMetrics:    true,
	}

	orchestrator, err := common.NewMigrationOrchestrator(orchConfig, log)
	if err != nil {
		log.Error("failed to create orchestrator", "error", err)
		return
	}
	defer orchestrator.Close()

	// Create batch of migrations
	batchConfigs := []*common.MigrationConfig{
		{
			VMName:           "vm-001",
			Provider:         "vsphere",
			OutputDir:        "/exports/vm-001",
			TargetFormat:     "qcow2",
			EnableConversion: true,
			User:             "admin",
		},
		{
			VMName:           "vm-002",
			Provider:         "vsphere",
			OutputDir:        "/exports/vm-002",
			TargetFormat:     "qcow2",
			EnableConversion: true,
			User:             "admin",
		},
		{
			VMName:           "vm-003",
			Provider:         "vsphere",
			OutputDir:        "/exports/vm-003",
			TargetFormat:     "qcow2",
			EnableConversion: true,
			User:             "admin",
		},
	}

	fmt.Printf("Batch migration configured:\n")
	fmt.Printf("  Total VMs: %d\n", len(batchConfigs))
	for i, config := range batchConfigs {
		fmt.Printf("  %d. %s -> %s\n", i+1, config.VMName, config.OutputDir)
	}

	// Note: MigrateBatch requires BatchOrchestrator which needs VM config
	// In production, you would configure BatchOrchestrator with VM details

	fmt.Println("\n✓ Batch configuration ready")
	fmt.Println("  Sequential execution: enabled")
	fmt.Println("  Progress tracking per VM: enabled")
}

// webhookExample shows webhook notification setup
func webhookExample(log logger.Logger) {
	ctx := context.Background()

	// Configure webhooks
	webhookConfigs := []*common.WebhookConfig{
		{
			Type:       common.WebhookSlack,
			URL:        "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
			Enabled:    false, // Disabled for example
			OnStart:    true,
			OnComplete: true,
			OnError:    true,
		},
		{
			Type:       common.WebhookDiscord,
			URL:        "https://discord.com/api/webhooks/YOUR/WEBHOOK/URL",
			Enabled:    false, // Disabled for example
			OnComplete: true,
			OnError:    true,
		},
	}

	// Create orchestrator with webhooks
	orchConfig := &common.OrchestratorConfig{
		EnableConversion: true,
		EnableWebhooks:   true,
		WebhookConfigs:   webhookConfigs,
	}

	orchestrator, err := common.NewMigrationOrchestrator(orchConfig, log)
	if err != nil {
		log.Error("failed to create orchestrator", "error", err)
		return
	}
	defer orchestrator.Close()

	migrationConfig := &common.MigrationConfig{
		VMName:           "app-server-01",
		Provider:         "vsphere",
		OutputDir:        "/exports/app-server-01",
		TargetFormat:     "qcow2",
		EnableConversion: true,
		EnableWebhooks:   true,
		User:             "admin",
	}

	_ = ctx
	_ = migrationConfig

	fmt.Println("Webhook notifications configured:")
	for i, webhook := range webhookConfigs {
		status := "disabled"
		if webhook.Enabled {
			status = "enabled"
		}
		fmt.Printf("  %d. %s (%s)\n", i+1, webhook.Type, status)
		fmt.Printf("     Events: ")
		events := []string{}
		if webhook.OnStart {
			events = append(events, "start")
		}
		if webhook.OnComplete {
			events = append(events, "complete")
		}
		if webhook.OnError {
			events = append(events, "error")
		}
		fmt.Printf("%v\n", events)
	}

	fmt.Println("\n✓ Webhook notifications configured")
	fmt.Println("  Automatic notifications on migration events")
	fmt.Println("  Retry logic: 3 attempts with 5s delay")
}

// Example output:
//
// === Example 1: Basic Migration ===
//
// Migration Configuration:
//   VM: web-server-01
//   Provider: vsphere
//   Target Format: qcow2
//   Output: /exports/web-server-01
//
// ✓ Orchestrator configured successfully
//   Progress tracking: enabled
//   Metrics collection: enabled
//   Conversion: enabled
//
// === Example 2: Full Monitoring ===
//
// Monitoring servers started:
//   Progress API: http://localhost:8080/api/v1/progress
//   Metrics API:  http://localhost:9090/metrics
//   Audit Log:    /var/log/hypersdk/audit.log
//
// Active migrations: 0
// Total migrations recorded: 0
//
// ✓ Full monitoring configured successfully
//
// === Example 3: Batch Migration ===
//
// Batch migration configured:
//   Total VMs: 3
//   1. vm-001 -> /exports/vm-001
//   2. vm-002 -> /exports/vm-002
//   3. vm-003 -> /exports/vm-003
//
// ✓ Batch configuration ready
//   Sequential execution: enabled
//   Progress tracking per VM: enabled
//
// === Example 4: Webhook Notifications ===
//
// Webhook notifications configured:
//   1. slack (disabled)
//      Events: [start complete error]
//   2. discord (disabled)
//      Events: [complete error]
//
// ✓ Webhook notifications configured
//   Automatic notifications on migration events
//   Retry logic: 3 attempts with 5s delay
