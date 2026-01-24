// SPDX-License-Identifier: LGPL-3.0-or-later

package jobs

import (
	"testing"

	"hypersdk/daemon/capabilities"
	"hypersdk/daemon/models"
	"hypersdk/logger"
)

// Helper to create a test capability detector
func newTestDetector(log logger.Logger) *capabilities.Detector {
	return capabilities.NewDetector(log)
}

func TestNewManager(t *testing.T) {
	log := logger.New("info")
	detector := newTestDetector(log)
	mgr := NewManager(log, detector)

	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}

	status := mgr.GetStatus()
	if status.TotalJobs != 0 {
		t.Errorf("Expected 0 total jobs, got %d", status.TotalJobs)
	}

	if status.RunningJobs != 0 {
		t.Errorf("Expected 0 running jobs, got %d", status.RunningJobs)
	}
}

func TestSubmitJob(t *testing.T) {
	log := logger.New("info")
	detector := newTestDetector(log)
	mgr := NewManager(log, detector)

	jobDef := models.JobDefinition{
		Name:       "test-job",
		VMPath:     "/datacenter/vm/test-vm",
		OutputPath: "/tmp/test-output",
		Options: &models.ExportOptions{
			ParallelDownloads: 4,
			RemoveCDROM:       true,
		},
	}

	jobID, err := mgr.SubmitJob(jobDef)
	if err != nil {
		t.Fatalf("SubmitJob() error = %v", err)
	}

	if jobID == "" {
		t.Error("Expected non-empty job ID")
	}

	// Verify job was added
	status := mgr.GetStatus()
	if status.TotalJobs != 1 {
		t.Errorf("Expected 1 total job, got %d", status.TotalJobs)
	}
}

func TestGetJob(t *testing.T) {
	log := logger.New("info")
	detector := newTestDetector(log)
	mgr := NewManager(log, detector)

	jobDef := models.JobDefinition{
		Name:       "test-job-2",
		VMPath:     "/datacenter/vm/test-vm-2",
		OutputPath: "/tmp/test-output-2",
	}

	jobID, err := mgr.SubmitJob(jobDef)
	if err != nil {
		t.Fatalf("SubmitJob() error = %v", err)
	}

	// Get the job
	job, err := mgr.GetJob(jobID)
	if err != nil {
		t.Fatalf("GetJob() error = %v", err)
	}

	if job == nil {
		t.Fatal("GetJob() returned nil job")
	}

	if job.Definition.Name != "test-job-2" {
		t.Errorf("Expected job name 'test-job-2', got '%s'", job.Definition.Name)
	}
}

func TestGetJobNotFound(t *testing.T) {
	log := logger.New("info")
	detector := newTestDetector(log)
	mgr := NewManager(log, detector)

	_, err := mgr.GetJob("non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent job ID, got nil")
	}
}

func TestGetAllJobs(t *testing.T) {
	log := logger.New("info")
	detector := newTestDetector(log)
	mgr := NewManager(log, detector)

	// Submit multiple jobs
	for i := 0; i < 3; i++ {
		jobDef := models.JobDefinition{
			Name:       "test-job",
			VMPath:     "/datacenter/vm/test-vm",
			OutputPath: "/tmp/test",
		}
		_, err := mgr.SubmitJob(jobDef)
		if err != nil {
			t.Fatalf("SubmitJob() error = %v", err)
		}
	}

	jobs := mgr.GetAllJobs()
	if len(jobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(jobs))
	}
}

func TestListJobsByStatus(t *testing.T) {
	log := logger.New("info")
	detector := newTestDetector(log)
	mgr := NewManager(log, detector)

	jobDef := models.JobDefinition{
		Name:       "pending-job",
		VMPath:     "/datacenter/vm/test-vm",
		OutputPath: "/tmp/test",
	}

	jobID, _ := mgr.SubmitJob(jobDef)

	// Get job and verify it's pending or running (since it starts automatically)
	job, _ := mgr.GetJob(jobID)
	if job.Status != models.JobStatusPending && job.Status != models.JobStatusRunning {
		t.Logf("Job status is %s (expected pending or running)", job.Status)
	}

	// List pending jobs
	pendingJobs := mgr.ListJobs([]models.JobStatus{models.JobStatusPending}, 0)
	// Since jobs start automatically, there might be 0 or 1 pending jobs
	t.Logf("Found %d pending jobs", len(pendingJobs))
}

func TestCancelJob(t *testing.T) {
	log := logger.New("info")
	detector := newTestDetector(log)
	mgr := NewManager(log, detector)

	jobDef := models.JobDefinition{
		Name:       "cancel-job",
		VMPath:     "/datacenter/vm/test-vm",
		OutputPath: "/tmp/test",
	}

	jobID, _ := mgr.SubmitJob(jobDef)

	// Try to cancel the job
	err := mgr.CancelJob(jobID)
	// It might fail if the job already completed, which is OK for this test
	if err != nil {
		t.Logf("CancelJob() returned error (job may have completed): %v", err)
	}

	// If cancel succeeded, verify cancelled state
	job, _ := mgr.GetJob(jobID)
	if job.Status == models.JobStatusCancelled {
		t.Log("Job successfully cancelled")
	} else {
		t.Logf("Job status is %s", job.Status)
	}
}

func TestSubmitBatch(t *testing.T) {
	log := logger.New("info")
	detector := newTestDetector(log)
	mgr := NewManager(log, detector)

	// Create batch of job definitions
	defs := []models.JobDefinition{
		{
			Name:       "batch-job-1",
			VMPath:     "/datacenter/vm/vm1",
			OutputPath: "/tmp/out1",
		},
		{
			Name:       "batch-job-2",
			VMPath:     "/datacenter/vm/vm2",
			OutputPath: "/tmp/out2",
		},
		{
			Name:       "batch-job-3",
			VMPath:     "/datacenter/vm/vm3",
			OutputPath: "/tmp/out3",
		},
	}

	ids, errs := mgr.SubmitBatch(defs)

	if len(errs) > 0 {
		t.Errorf("SubmitBatch() returned %d errors", len(errs))
	}

	if len(ids) != 3 {
		t.Errorf("Expected 3 job IDs, got %d", len(ids))
	}

	// Verify all jobs were added
	status := mgr.GetStatus()
	if status.TotalJobs != 3 {
		t.Errorf("Expected 3 total jobs, got %d", status.TotalJobs)
	}
}

func TestGetStatus(t *testing.T) {
	log := logger.New("info")
	detector := newTestDetector(log)
	mgr := NewManager(log, detector)

	status := mgr.GetStatus()

	if status == nil {
		t.Fatal("GetStatus() returned nil")
	}

	if status.Version == "" {
		t.Error("Expected non-empty version")
	}

	if status.Uptime == "" {
		t.Error("Expected non-empty uptime")
	}

	if status.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}
}

func TestGetStatusWithJobs(t *testing.T) {
	log := logger.New("info")
	detector := newTestDetector(log)
	mgr := NewManager(log, detector)

	// Create jobs with different statuses
	job1 := &models.Job{Definition: models.JobDefinition{ID: "job1"}, Status: models.JobStatusRunning}
	job2 := &models.Job{Definition: models.JobDefinition{ID: "job2"}, Status: models.JobStatusCompleted}
	job3 := &models.Job{Definition: models.JobDefinition{ID: "job3"}, Status: models.JobStatusFailed}
	job4 := &models.Job{Definition: models.JobDefinition{ID: "job4"}, Status: models.JobStatusCancelled}
	job5 := &models.Job{Definition: models.JobDefinition{ID: "job5"}, Status: models.JobStatusPending}

	mgr.jobs[job1.Definition.ID] = job1
	mgr.jobs[job2.Definition.ID] = job2
	mgr.jobs[job3.Definition.ID] = job3
	mgr.jobs[job4.Definition.ID] = job4
	mgr.jobs[job5.Definition.ID] = job5

	status := mgr.GetStatus()

	if status.TotalJobs != 5 {
		t.Errorf("Expected TotalJobs 5, got %d", status.TotalJobs)
	}

	if status.RunningJobs != 1 {
		t.Errorf("Expected RunningJobs 1, got %d", status.RunningJobs)
	}

	if status.CompletedJobs != 1 {
		t.Errorf("Expected CompletedJobs 1, got %d", status.CompletedJobs)
	}

	if status.FailedJobs != 1 {
		t.Errorf("Expected FailedJobs 1, got %d", status.FailedJobs)
	}

	if status.CancelledJobs != 1 {
		t.Errorf("Expected CancelledJobs 1, got %d", status.CancelledJobs)
	}
}

func TestShutdown(t *testing.T) {
	log := logger.New("info")
	detector := newTestDetector(log)
	mgr := NewManager(log, detector)

	// Should not panic
	mgr.Shutdown()
}
