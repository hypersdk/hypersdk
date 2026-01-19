// SPDX-License-Identifier: LGPL-3.0-or-later

package common

import (
	"context"
	"fmt"
	"os"
	"time"

	"hypersdk/logger"
)

// MigrationOrchestrator coordinates the complete end-to-end migration workflow
// integrating all Phase 0-5 components
type MigrationOrchestrator struct {
	// Phase 3: Conversion
	conversionManager *ConversionManager

	// Phase 4: Advanced features
	parallelConverter *ParallelConverter
	cloudStorage      CloudStorageProvider
	batchOrchestrator *BatchOrchestrator

	// Phase 5: Monitoring & Reporting
	progressTracker  *ProgressTracker
	metricsCollector *MetricsCollector
	auditLogger      *AuditLogger
	webhookManager   *WebhookManager

	logger logger.Logger
}

// MigrationConfig holds configuration for the complete migration
type MigrationConfig struct {
	// Basic config
	VMName       string
	Provider     string
	OutputDir    string
	TargetFormat string // qcow2, raw, vdi

	// Export options
	ExportManifest bool
	VerifyExport   bool

	// Phase 3: Conversion options
	EnableConversion bool
	ConvertOptions   ConvertOptions

	// Phase 4: Advanced features
	ParallelDisks    bool
	MaxParallelDisks int
	UploadToCloud    bool
	CloudDestination string

	// Phase 5: Monitoring
	EnableProgress     bool
	EnableMetrics      bool
	EnableAuditLogging bool
	EnableWebhooks     bool
	WebhookConfigs     []*WebhookConfig
	ProgressAPIPort    int
	MetricsAPIPort     int

	// User context
	User      string
	IPAddress string
}

// MigrationResult holds the results of a complete migration
type MigrationResult struct {
	TaskID   string
	VMName   string
	Provider string
	Success  bool
	Error    string

	// Export results
	ExportDuration time.Duration
	ExportedFiles  []string
	ExportSize     int64
	ManifestPath   string

	// Conversion results
	ConversionDuration time.Duration
	ConvertedFiles     []string
	ConversionSize     int64
	ConversionReport   string

	// Upload results
	UploadDuration   time.Duration
	UploadedFiles    []string
	CloudDestination string

	// Overall results
	TotalDuration time.Duration
	TotalSize     int64
	StartTime     time.Time
	EndTime       time.Time
}

// NewMigrationOrchestrator creates a new migration orchestrator
func NewMigrationOrchestrator(config *OrchestratorConfig, log logger.Logger) (*MigrationOrchestrator, error) {
	mo := &MigrationOrchestrator{
		logger: log,
	}

	// Initialize Phase 3: Conversion Manager
	if config.EnableConversion {
		convMgr, err := NewConversionManager(&ConverterConfig{}, log)
		if err != nil {
			return nil, fmt.Errorf("create conversion manager: %w", err)
		}
		mo.conversionManager = convMgr
	}

	// Initialize Phase 4: Advanced features
	if config.EnableParallelConversion {
		// ParallelConverter requires a base converter, use nil for now
		mo.parallelConverter = NewParallelConverter(nil, 4, log)
	}

	if config.EnableBatchOrchestration {
		batchOrch, err := NewBatchOrchestrator(&BatchMigrationConfig{}, log)
		if err != nil {
			return nil, fmt.Errorf("create batch orchestrator: %w", err)
		}
		mo.batchOrchestrator = batchOrch
	}

	// Initialize Phase 5: Monitoring & Reporting
	if config.EnableProgress {
		mo.progressTracker = NewProgressTracker()
	}

	if config.EnableMetrics {
		mo.metricsCollector = NewMetricsCollector()
	}

	if config.EnableAuditLogging && config.AuditLogPath != "" {
		auditLogger, err := NewAuditLogger(config.AuditLogPath)
		if err != nil {
			return nil, fmt.Errorf("create audit logger: %w", err)
		}
		mo.auditLogger = auditLogger
	}

	if config.EnableWebhooks && len(config.WebhookConfigs) > 0 {
		mo.webhookManager = NewWebhookManager(config.WebhookConfigs, log)
	}

	return mo, nil
}

// OrchestratorConfig holds configuration for the orchestrator
type OrchestratorConfig struct {
	// Phase 3
	EnableConversion bool

	// Phase 4
	EnableParallelConversion bool
	EnableCloudStorage       bool
	EnableBatchOrchestration bool

	// Phase 5
	EnableProgress     bool
	EnableMetrics      bool
	EnableAuditLogging bool
	EnableWebhooks     bool
	AuditLogPath       string
	WebhookConfigs     []*WebhookConfig
}

// Migrate performs a complete end-to-end migration
func (mo *MigrationOrchestrator) Migrate(ctx context.Context, config *MigrationConfig) (*MigrationResult, error) {
	startTime := time.Now()
	taskID := generateTaskID()

	result := &MigrationResult{
		TaskID:    taskID,
		VMName:    config.VMName,
		Provider:  config.Provider,
		StartTime: startTime,
	}

	mo.logger.Info("starting migration orchestration",
		"task_id", taskID,
		"vm", config.VMName,
		"provider", config.Provider)

	// Phase 5: Start progress tracking
	if mo.progressTracker != nil {
		mo.progressTracker.StartTask(taskID, config.VMName, config.Provider)
	}

	// Phase 5: Record migration start
	if mo.metricsCollector != nil {
		mo.metricsCollector.RecordMigrationStart(config.Provider)
	}

	if mo.auditLogger != nil {
		mo.auditLogger.LogMigrationStart(taskID, config.VMName, config.Provider, config.User)
	}

	if mo.webhookManager != nil {
		mo.webhookManager.NotifyStart(taskID, config.VMName, config.Provider)
	}

	// Phase 1: Export with manifest (handled by caller)
	// This returns ExportedFiles, ManifestPath, etc.

	// Phase 3: Conversion (if enabled)
	if config.EnableConversion && mo.conversionManager != nil {
		mo.logger.Info("starting conversion phase", "task_id", taskID)

		if mo.progressTracker != nil {
			mo.progressTracker.SetStatus(taskID, StatusConverting)
		}

		if mo.auditLogger != nil {
			mo.auditLogger.LogConversionStart(taskID, config.VMName)
		}

		convStart := time.Now()

		// Load manifest
		manifestPath := config.OutputDir + "/manifest.json"

		// Phase 3: Sequential conversion using conversion manager
		convResult, err := mo.conversionManager.Convert(ctx, manifestPath, config.ConvertOptions)
		if err != nil {
			return mo.handleFailure(taskID, config, result, fmt.Errorf("conversion: %w", err))
		}

		convertedFiles := convResult.ConvertedFiles
		// Calculate size from converted files
		var conversionSize int64
		for _, file := range convertedFiles {
			if fi, err := os.Stat(file); err == nil {
				conversionSize += fi.Size()
			}
		}

		convDuration := time.Since(convStart)
		result.ConversionDuration = convDuration
		result.ConvertedFiles = convertedFiles
		result.ConversionSize = conversionSize

		if mo.auditLogger != nil {
			mo.auditLogger.LogConversionComplete(taskID, config.VMName, convDuration, convertedFiles)
		}

		mo.logger.Info("conversion completed",
			"task_id", taskID,
			"duration", convDuration,
			"files", len(convertedFiles))
	}

	// Phase 4: Cloud upload (if enabled)
	// TODO: Implement cloud upload when cloud storage provider is available
	if config.UploadToCloud && config.CloudDestination != "" {
		mo.logger.Info("cloud upload requested but not yet implemented",
			"task_id", taskID,
			"destination", config.CloudDestination)
		// Placeholder for future cloud upload implementation
	}

	// Complete migration
	endTime := time.Now()
	totalDuration := endTime.Sub(startTime)

	result.Success = true
	result.EndTime = endTime
	result.TotalDuration = totalDuration
	result.TotalSize = result.ExportSize + result.ConversionSize

	// Phase 5: Complete tracking
	if mo.progressTracker != nil {
		mo.progressTracker.CompleteTask(taskID)
	}

	if mo.metricsCollector != nil {
		mo.metricsCollector.RecordMigrationSuccess(
			config.Provider,
			result.ExportDuration,
			result.ConversionDuration,
			result.UploadDuration,
			result.ExportSize,
			result.ConversionSize,
			result.ConversionSize,
		)
	}

	if mo.auditLogger != nil {
		mo.auditLogger.LogMigrationComplete(
			taskID,
			config.VMName,
			config.Provider,
			config.User,
			totalDuration,
			map[string]interface{}{
				"exported_files":  len(result.ExportedFiles),
				"converted_files": len(result.ConvertedFiles),
				"uploaded_files":  len(result.UploadedFiles),
				"total_size":      result.TotalSize,
			},
		)
	}

	if mo.webhookManager != nil {
		mo.webhookManager.NotifyComplete(taskID, config.VMName, config.Provider, totalDuration)
	}

	mo.logger.Info("migration orchestration completed",
		"task_id", taskID,
		"total_duration", totalDuration,
		"success", true)

	return result, nil
}

// handleFailure handles migration failures
func (mo *MigrationOrchestrator) handleFailure(
	taskID string,
	config *MigrationConfig,
	result *MigrationResult,
	err error,
) (*MigrationResult, error) {
	result.Success = false
	result.Error = err.Error()
	result.EndTime = time.Now()
	result.TotalDuration = result.EndTime.Sub(result.StartTime)

	mo.logger.Error("migration orchestration failed",
		"task_id", taskID,
		"error", err)

	// Phase 5: Record failure
	if mo.progressTracker != nil {
		mo.progressTracker.FailTask(taskID, err)
	}

	if mo.metricsCollector != nil {
		mo.metricsCollector.RecordMigrationFailure(config.Provider)
	}

	if mo.auditLogger != nil {
		mo.auditLogger.LogMigrationFailed(taskID, config.VMName, config.Provider, config.User, err)
	}

	if mo.webhookManager != nil {
		mo.webhookManager.NotifyError(taskID, config.VMName, config.Provider, err)
	}

	return result, err
}

// MigrateBatch performs batch migration using Phase 4 BatchOrchestrator
func (mo *MigrationOrchestrator) MigrateBatch(
	ctx context.Context,
	configs []*MigrationConfig,
) ([]*MigrationResult, error) {
	if mo.batchOrchestrator == nil {
		return nil, fmt.Errorf("batch orchestration not enabled")
	}

	mo.logger.Info("starting batch migration", "count", len(configs))

	// This would coordinate with BatchOrchestrator from Phase 4
	// For now, we'll do sequential migrations
	results := make([]*MigrationResult, 0, len(configs))

	for i, config := range configs {
		mo.logger.Info("batch migration progress",
			"current", i+1,
			"total", len(configs),
			"vm", config.VMName)

		result, err := mo.Migrate(ctx, config)
		if err != nil {
			mo.logger.Error("batch migration item failed",
				"vm", config.VMName,
				"error", err)
		}

		results = append(results, result)
	}

	return results, nil
}

// GetProgressTracker returns the progress tracker
func (mo *MigrationOrchestrator) GetProgressTracker() *ProgressTracker {
	return mo.progressTracker
}

// GetMetricsCollector returns the metrics collector
func (mo *MigrationOrchestrator) GetMetricsCollector() *MetricsCollector {
	return mo.metricsCollector
}

// GetAuditLogger returns the audit logger
func (mo *MigrationOrchestrator) GetAuditLogger() *AuditLogger {
	return mo.auditLogger
}

// Close closes all resources
func (mo *MigrationOrchestrator) Close() error {
	if mo.auditLogger != nil {
		return mo.auditLogger.Close()
	}
	return nil
}

// generateTaskID generates a unique task ID
func generateTaskID() string {
	return fmt.Sprintf("task_%d", time.Now().UnixNano())
}
