// SPDX-License-Identifier: LGPL-3.0-or-later

package scheduler

import (
	"testing"
	"time"

	"hypersdk/daemon/models"
	"hypersdk/logger"
	"hypersdk/providers/carbon"
)

// carbonMockExecutor implements JobExecutor for testing carbon-aware scheduler
type carbonMockExecutor struct {
	submittedJobs []models.JobDefinition
}

func (m *carbonMockExecutor) SubmitJob(def models.JobDefinition) (string, error) {
	m.submittedJobs = append(m.submittedJobs, def)
	return "job-123", nil
}

func setupCarbonAwareScheduler(t *testing.T, config CarbonAwareConfig) (*CarbonAwareScheduler, *carbonMockExecutor) {
	log := logger.NewTestLogger(t)
	executor := &carbonMockExecutor{
		submittedJobs: make([]models.JobDefinition, 0),
	}

	baseScheduler := NewScheduler(executor, log)
	carbonProvider := carbon.NewMockProvider()

	carbonScheduler := NewCarbonAwareScheduler(
		baseScheduler,
		carbonProvider,
		config,
		log,
	)

	return carbonScheduler, executor
}

func TestCarbonAwareScheduler_DisabledGlobally(t *testing.T) {
	config := DefaultCarbonAwareConfig()
	config.Enabled = false // Disabled globally

	scheduler, executor := setupCarbonAwareScheduler(t, config)

	// Submit a job with carbon_aware = true
	job := models.JobDefinition{
		Name:   "test-job",
		VMPath: "/datacenter/vm/test",
		Metadata: map[string]interface{}{
			"carbon_aware": true,
		},
	}

	_, err := scheduler.SubmitJob(job)
	if err != nil {
		t.Fatalf("SubmitJob failed: %v", err)
	}

	// Should submit immediately without carbon check
	if len(executor.submittedJobs) != 1 {
		t.Errorf("Expected 1 job submitted, got %d", len(executor.submittedJobs))
	}
}

func TestCarbonAwareScheduler_EnabledCleanGrid(t *testing.T) {
	config := DefaultCarbonAwareConfig()
	config.Enabled = true
	config.DefaultMaxIntensity = 300.0 // High threshold (mock will be under this)

	scheduler, executor := setupCarbonAwareScheduler(t, config)

	// Submit a job with carbon awareness
	job := models.JobDefinition{
		Name:   "test-job",
		VMPath: "/datacenter/vm/test",
		Metadata: map[string]interface{}{
			"carbon_aware": true,
			"carbon_zone":  carbon.ZoneUSEast,
		},
	}

	_, err := scheduler.SubmitJob(job)
	if err != nil {
		t.Fatalf("SubmitJob failed: %v", err)
	}

	// Should submit immediately (grid is clean)
	if len(executor.submittedJobs) != 1 {
		t.Errorf("Expected 1 job submitted, got %d", len(executor.submittedJobs))
	}

	// Check that carbon metadata was added
	submittedJob := executor.submittedJobs[0]
	if submittedJob.Metadata["carbon_intensity_at_submit"] == nil {
		t.Error("Expected carbon_intensity_at_submit in metadata")
	}
	if submittedJob.Metadata["carbon_renewable_percent"] == nil {
		t.Error("Expected carbon_renewable_percent in metadata")
	}
}

func TestCarbonAwareScheduler_DirtyGridWithDelay(t *testing.T) {
	config := DefaultCarbonAwareConfig()
	config.Enabled = true
	config.DefaultMaxIntensity = 50.0 // Very low threshold (mock will be above)
	config.DefaultMaxDelay = 4 * time.Hour

	scheduler, executor := setupCarbonAwareScheduler(t, config)

	// Submit a job with carbon awareness
	job := models.JobDefinition{
		Name:   "test-job",
		VMPath: "/datacenter/vm/test",
		Metadata: map[string]interface{}{
			"carbon_aware": true,
			"carbon_zone":  carbon.ZoneUSEast,
		},
	}

	scheduleID, err := scheduler.SubmitJob(job)
	if err != nil {
		t.Fatalf("SubmitJob failed: %v", err)
	}

	// Should delay the job (not submitted immediately)
	if scheduleID == "" {
		t.Error("Expected schedule ID for delayed job")
	}

	// Check that job was NOT submitted immediately
	// (it will be submitted after the delay by goroutine)
	if len(executor.submittedJobs) > 0 {
		t.Errorf("Expected job to be delayed, but %d jobs were submitted", len(executor.submittedJobs))
	}
}

func TestCarbonAwareScheduler_DirtyGridNoGoodTime(t *testing.T) {
	config := DefaultCarbonAwareConfig()
	config.Enabled = true
	config.DefaultMaxIntensity = 10.0 // Impossibly low (mock will never meet it)
	config.DefaultMaxDelay = 1 * time.Hour

	scheduler, executor := setupCarbonAwareScheduler(t, config)

	// Submit a job with carbon awareness
	job := models.JobDefinition{
		Name:   "test-job",
		VMPath: "/datacenter/vm/test",
		Metadata: map[string]interface{}{
			"carbon_aware": true,
			"carbon_zone":  carbon.ZoneUSEast,
		},
	}

	_, err := scheduler.SubmitJob(job)
	if err != nil {
		t.Fatalf("SubmitJob failed: %v", err)
	}

	// Should submit anyway (no optimal time found)
	if len(executor.submittedJobs) != 1 {
		t.Errorf("Expected 1 job submitted, got %d", len(executor.submittedJobs))
	}
}

func TestCarbonAwareScheduler_JobSpecificSettings(t *testing.T) {
	config := DefaultCarbonAwareConfig()
	config.Enabled = true
	config.DefaultMaxIntensity = 200.0

	scheduler, _ := setupCarbonAwareScheduler(t, config)

	// Submit job with custom carbon settings
	job := models.JobDefinition{
		Name:   "test-job",
		VMPath: "/datacenter/vm/test",
		Metadata: map[string]interface{}{
			"carbon_aware":          true,
			"carbon_max_intensity":  150.0, // Override default
			"carbon_zone":           carbon.ZoneEUNorth,
			"carbon_max_delay":      2 * time.Hour,
		},
	}

	settings := scheduler.getCarbonSettings(job)

	if settings.MaxIntensity != 150.0 {
		t.Errorf("Expected max intensity 150.0, got %f", settings.MaxIntensity)
	}
	if settings.Zone != carbon.ZoneEUNorth {
		t.Errorf("Expected zone %s, got %s", carbon.ZoneEUNorth, settings.Zone)
	}
	if settings.MaxDelay != 2*time.Hour {
		t.Errorf("Expected max delay 2h, got %v", settings.MaxDelay)
	}
}

func TestCarbonAwareScheduler_GetCarbonStatus(t *testing.T) {
	config := DefaultCarbonAwareConfig()
	config.Enabled = true

	scheduler, _ := setupCarbonAwareScheduler(t, config)

	status, err := scheduler.GetCarbonStatus(carbon.ZoneUSEast, 200.0)
	if err != nil {
		t.Fatalf("GetCarbonStatus failed: %v", err)
	}

	if status.Current.Zone != carbon.ZoneUSEast {
		t.Errorf("Expected zone %s, got %s", carbon.ZoneUSEast, status.Current.Zone)
	}

	if status.Current.CarbonIntensity <= 0 {
		t.Error("Expected positive carbon intensity")
	}

	if status.Reasoning == "" {
		t.Error("Expected non-empty reasoning")
	}
}

func TestCarbonAwareScheduler_GenerateCarbonReport(t *testing.T) {
	config := DefaultCarbonAwareConfig()
	config.Enabled = true

	scheduler, _ := setupCarbonAwareScheduler(t, config)

	startTime := time.Now()
	endTime := startTime.Add(2 * time.Hour)

	report, err := scheduler.GenerateCarbonReport(
		"job-123",
		startTime,
		endTime,
		500.0, // 500 GB
		carbon.ZoneUSEast,
	)

	if err != nil {
		t.Fatalf("GenerateCarbonReport failed: %v", err)
	}

	if report.OperationID != "job-123" {
		t.Errorf("Expected operation ID 'job-123', got '%s'", report.OperationID)
	}

	if report.Duration <= 0 {
		t.Errorf("Expected positive duration, got %f", report.Duration)
	}

	if report.EnergyUsed <= 0 {
		t.Errorf("Expected positive energy, got %f", report.EnergyUsed)
	}

	if report.CarbonEmissions <= 0 {
		t.Errorf("Expected positive emissions, got %f", report.CarbonEmissions)
	}
}

func TestCarbonAwareScheduler_EstimateCarbonSavings(t *testing.T) {
	config := DefaultCarbonAwareConfig()
	config.Enabled = true

	scheduler, _ := setupCarbonAwareScheduler(t, config)

	estimate, err := scheduler.EstimateCarbonSavings(
		carbon.ZoneUSEast,
		500.0, // 500 GB
		2.0,   // 2 hours
	)

	if err != nil {
		t.Fatalf("EstimateCarbonSavings failed: %v", err)
	}

	// Check required fields
	requiredFields := []string{
		"current_intensity",
		"current_emissions",
		"best_intensity",
		"best_emissions",
		"savings_kg_co2",
		"savings_percent",
		"recommendation",
	}

	for _, field := range requiredFields {
		if estimate[field] == nil {
			t.Errorf("Expected field '%s' in estimate", field)
		}
	}

	// Recommendation should be non-empty
	if recommendation, ok := estimate["recommendation"].(string); !ok || recommendation == "" {
		t.Error("Expected non-empty recommendation")
	}
}

func TestCarbonAwareScheduler_FallbackOnError(t *testing.T) {
	config := DefaultCarbonAwareConfig()
	config.Enabled = true
	config.FallbackOnError = true
	config.DefaultZone = "INVALID-ZONE" // Will cause error

	scheduler, executor := setupCarbonAwareScheduler(t, config)

	job := models.JobDefinition{
		Name:   "test-job",
		VMPath: "/datacenter/vm/test",
		Metadata: map[string]interface{}{
			"carbon_aware": true,
		},
	}

	_, err := scheduler.SubmitJob(job)
	if err != nil {
		t.Fatalf("SubmitJob failed: %v", err)
	}

	// Should fallback and submit job despite error
	if len(executor.submittedJobs) != 1 {
		t.Errorf("Expected job to be submitted via fallback, got %d jobs", len(executor.submittedJobs))
	}
}

func TestCarbonAwareScheduler_NoFallbackOnError(t *testing.T) {
	config := DefaultCarbonAwareConfig()
	config.Enabled = true
	config.FallbackOnError = false
	config.DefaultZone = "INVALID-ZONE" // Will cause error

	scheduler, executor := setupCarbonAwareScheduler(t, config)

	job := models.JobDefinition{
		Name:   "test-job",
		VMPath: "/datacenter/vm/test",
		Metadata: map[string]interface{}{
			"carbon_aware": true,
		},
	}

	_, err := scheduler.SubmitJob(job)
	if err == nil {
		t.Error("Expected error when fallback is disabled")
	}

	// Should NOT submit job
	if len(executor.submittedJobs) > 0 {
		t.Errorf("Expected no jobs submitted, got %d", len(executor.submittedJobs))
	}
}

func TestDefaultCarbonAwareConfig(t *testing.T) {
	config := DefaultCarbonAwareConfig()

	// Check defaults
	if config.Enabled {
		t.Error("Expected Enabled to be false by default (opt-in)")
	}

	if config.DefaultMaxIntensity != 200.0 {
		t.Errorf("Expected max intensity 200.0, got %f", config.DefaultMaxIntensity)
	}

	if config.DefaultMaxDelay != 4*time.Hour {
		t.Errorf("Expected max delay 4h, got %v", config.DefaultMaxDelay)
	}

	if config.DefaultZone != carbon.ZoneUSEast {
		t.Errorf("Expected default zone %s, got %s", carbon.ZoneUSEast, config.DefaultZone)
	}

	if !config.FallbackOnError {
		t.Error("Expected FallbackOnError to be true")
	}
}

func TestEnrichWithCarbonMetadata(t *testing.T) {
	config := DefaultCarbonAwareConfig()
	scheduler, _ := setupCarbonAwareScheduler(t, config)

	intensity := carbon.CarbonIntensity{
		Zone:              carbon.ZoneUSEast,
		CarbonIntensity:   150.5,
		FossilFreePercent: 65.3,
		Timestamp:         time.Now(),
		Source:            "test",
	}

	metadata := scheduler.enrichWithCarbonMetadata(nil, intensity)

	// Check all fields were added
	if metadata["carbon_intensity_at_submit"] != 150.5 {
		t.Error("carbon_intensity_at_submit not set correctly")
	}

	if metadata["carbon_renewable_percent"] != 65.3 {
		t.Error("carbon_renewable_percent not set correctly")
	}

	if metadata["carbon_zone"] != carbon.ZoneUSEast {
		t.Error("carbon_zone not set correctly")
	}

	if metadata["carbon_timestamp"] == nil {
		t.Error("carbon_timestamp not set")
	}
}
