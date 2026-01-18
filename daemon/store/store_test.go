// SPDX-License-Identifier: LGPL-3.0-or-later

package store

import (
	"fmt"
	"os"
	"testing"
	"time"

	"hypersdk/daemon/models"
)

func TestSQLiteStore_SaveAndGetJob(t *testing.T) {
	// Create temporary database
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create test job
	now := time.Now()
	job := &models.Job{
		Definition: models.JobDefinition{
			ID:         "test-job-1",
			Name:       "Test Job",
			VMPath:     "/data/vm/test",
			OutputPath: "/tmp/test",
			CreatedAt:  now,
		},
		Status: models.JobStatusPending,
	}

	// Save job
	if err := store.SaveJob(job); err != nil {
		t.Fatalf("Failed to save job: %v", err)
	}

	// Retrieve job
	retrieved, err := store.GetJob("test-job-1")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	// Verify
	if retrieved.Definition.ID != job.Definition.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.Definition.ID, job.Definition.ID)
	}
	if retrieved.Definition.Name != job.Definition.Name {
		t.Errorf("Name mismatch: got %s, want %s", retrieved.Definition.Name, job.Definition.Name)
	}
	if retrieved.Status != job.Status {
		t.Errorf("Status mismatch: got %s, want %s", retrieved.Status, job.Status)
	}
}

func TestSQLiteStore_UpdateJob(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create and save job
	now := time.Now()
	job := &models.Job{
		Definition: models.JobDefinition{
			ID:         "test-job-2",
			Name:       "Test Job 2",
			VMPath:     "/data/vm/test2",
			OutputPath: "/tmp/test2",
			CreatedAt:  now,
		},
		Status: models.JobStatusPending,
	}

	if err := store.SaveJob(job); err != nil {
		t.Fatalf("Failed to save job: %v", err)
	}

	// Update job
	startTime := time.Now()
	job.Status = models.JobStatusRunning
	job.StartedAt = &startTime
	job.Progress = &models.JobProgress{
		Phase:           "exporting",
		PercentComplete: 50.0,
	}

	if err := store.UpdateJob(job); err != nil {
		t.Fatalf("Failed to update job: %v", err)
	}

	// Retrieve updated job
	retrieved, err := store.GetJob("test-job-2")
	if err != nil {
		t.Fatalf("Failed to get job: %v", err)
	}

	// Verify updates
	if retrieved.Status != models.JobStatusRunning {
		t.Errorf("Status not updated: got %s, want %s", retrieved.Status, models.JobStatusRunning)
	}
	if retrieved.Progress == nil {
		t.Error("Progress not saved")
	} else if retrieved.Progress.PercentComplete != 50.0 {
		t.Errorf("Progress not updated: got %.1f, want 50.0", retrieved.Progress.PercentComplete)
	}
}

func TestSQLiteStore_ListJobs(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create multiple jobs
	jobs := []*models.Job{
		{
			Definition: models.JobDefinition{
				ID:         "job-1",
				Name:       "Job 1",
				VMPath:     "/vm1",
				OutputPath: "/out1",
				CreatedAt:  time.Now().Add(-2 * time.Hour),
			},
			Status: models.JobStatusCompleted,
		},
		{
			Definition: models.JobDefinition{
				ID:         "job-2",
				Name:       "Job 2",
				VMPath:     "/vm2",
				OutputPath: "/out2",
				CreatedAt:  time.Now().Add(-1 * time.Hour),
			},
			Status: models.JobStatusRunning,
		},
		{
			Definition: models.JobDefinition{
				ID:         "job-3",
				Name:       "Job 3",
				VMPath:     "/vm3",
				OutputPath: "/out3",
				CreatedAt:  time.Now(),
			},
			Status: models.JobStatusFailed,
		},
	}

	for _, job := range jobs {
		if err := store.SaveJob(job); err != nil {
			t.Fatalf("Failed to save job: %v", err)
		}
	}

	// Test: List all jobs
	allJobs, err := store.ListJobs(JobFilter{})
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}
	if len(allJobs) != 3 {
		t.Errorf("Expected 3 jobs, got %d", len(allJobs))
	}

	// Test: Filter by status
	runningJobs, err := store.ListJobs(JobFilter{
		Status: []models.JobStatus{models.JobStatusRunning},
	})
	if err != nil {
		t.Fatalf("Failed to list running jobs: %v", err)
	}
	if len(runningJobs) != 1 {
		t.Errorf("Expected 1 running job, got %d", len(runningJobs))
	}
	if runningJobs[0].Definition.ID != "job-2" {
		t.Errorf("Expected job-2, got %s", runningJobs[0].Definition.ID)
	}

	// Test: Limit
	limitedJobs, err := store.ListJobs(JobFilter{Limit: 2})
	if err != nil {
		t.Fatalf("Failed to list with limit: %v", err)
	}
	if len(limitedJobs) != 2 {
		t.Errorf("Expected 2 jobs, got %d", len(limitedJobs))
	}
}

func TestSQLiteStore_DeleteJob(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create and save job
	job := &models.Job{
		Definition: models.JobDefinition{
			ID:         "delete-test",
			Name:       "Delete Test",
			VMPath:     "/vm",
			OutputPath: "/out",
			CreatedAt:  time.Now(),
		},
		Status: models.JobStatusCompleted,
	}

	if err := store.SaveJob(job); err != nil {
		t.Fatalf("Failed to save job: %v", err)
	}

	// Delete job
	if err := store.DeleteJob("delete-test"); err != nil {
		t.Fatalf("Failed to delete job: %v", err)
	}

	// Verify deletion
	_, err = store.GetJob("delete-test")
	if err == nil {
		t.Error("Expected error when getting deleted job, got nil")
	}
}

func TestSQLiteStore_GetStatistics(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	store, err := NewSQLiteStore(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	// Create jobs with different statuses
	statuses := []models.JobStatus{
		models.JobStatusCompleted,
		models.JobStatusCompleted,
		models.JobStatusFailed,
		models.JobStatusRunning,
		models.JobStatusPending,
	}

	for i, status := range statuses {
		job := &models.Job{
			Definition: models.JobDefinition{
				ID:         fmt.Sprintf("stats-job-%d", i),
				Name:       fmt.Sprintf("Stats Job %d", i),
				VMPath:     "/vm",
				OutputPath: "/out",
				CreatedAt:  time.Now(),
			},
			Status: status,
		}
		if err := store.SaveJob(job); err != nil {
			t.Fatalf("Failed to save job: %v", err)
		}
	}

	// Get statistics
	stats, err := store.GetStatistics()
	if err != nil {
		t.Fatalf("Failed to get statistics: %v", err)
	}

	// Verify
	if stats.Total != 5 {
		t.Errorf("Expected 5 total jobs, got %d", stats.Total)
	}
	if stats.Completed != 2 {
		t.Errorf("Expected 2 completed jobs, got %d", stats.Completed)
	}
	if stats.Failed != 1 {
		t.Errorf("Expected 1 failed job, got %d", stats.Failed)
	}
	if stats.Running != 1 {
		t.Errorf("Expected 1 running job, got %d", stats.Running)
	}
	if stats.Pending != 1 {
		t.Errorf("Expected 1 pending job, got %d", stats.Pending)
	}
}
