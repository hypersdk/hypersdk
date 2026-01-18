// SPDX-License-Identifier: LGPL-3.0-or-later

package models

import (
	"testing"
	"time"
)

func TestJobStatusConstants(t *testing.T) {
	tests := []struct {
		name   string
		status JobStatus
		want   string
	}{
		{"Pending", JobStatusPending, "pending"},
		{"Running", JobStatusRunning, "running"},
		{"Completed", JobStatusCompleted, "completed"},
		{"Failed", JobStatusFailed, "failed"},
		{"Cancelled", JobStatusCancelled, "cancelled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("JobStatus = %v, want %v", tt.status, tt.want)
			}
		})
	}
}

func TestExportOptionsToVSphereOptions(t *testing.T) {
	opts := &ExportOptions{
		ParallelDownloads:      8,
		RemoveCDROM:            true,
		ShowIndividualProgress: true,
	}

	vsOpts := opts.ToVSphereOptions("/tmp/export")

	if vsOpts.OutputPath != "/tmp/export" {
		t.Errorf("Expected OutputPath '/tmp/export', got '%s'", vsOpts.OutputPath)
	}
	if vsOpts.ParallelDownloads != 8 {
		t.Errorf("Expected ParallelDownloads 8, got %d", vsOpts.ParallelDownloads)
	}
	if !vsOpts.RemoveCDROM {
		t.Error("Expected RemoveCDROM to be true")
	}
	if !vsOpts.ShowIndividualProgress {
		t.Error("Expected ShowIndividualProgress to be true")
	}
}

func TestExportOptionsDefaults(t *testing.T) {
	// Nil options should use defaults
	var opts *ExportOptions
	vsOpts := opts.ToVSphereOptions("/tmp/test")

	if vsOpts.OutputPath != "/tmp/test" {
		t.Errorf("Expected OutputPath '/tmp/test', got '%s'", vsOpts.OutputPath)
	}
	// Check default values are set by DefaultExportOptions
	if vsOpts.ParallelDownloads == 0 {
		t.Error("Expected default ParallelDownloads to be set")
	}
}

func TestJobCreation(t *testing.T) {
	def := JobDefinition{
		ID:         "test-job-1",
		Name:       "Test Job",
		VMPath:     "/datacenter/vm/test",
		OutputPath: "/tmp/export",
		CreatedAt:  time.Now(),
	}

	job := &Job{
		Definition: def,
		Status:     JobStatusPending,
		UpdatedAt:  time.Now(),
	}

	if job.Status != JobStatusPending {
		t.Errorf("Expected status pending, got %s", job.Status)
	}
	if job.Definition.ID != "test-job-1" {
		t.Errorf("Expected job ID 'test-job-1', got '%s'", job.Definition.ID)
	}
}

func TestQueryRequest(t *testing.T) {
	req := &QueryRequest{
		JobIDs: []string{"job1", "job2"},
		Status: []JobStatus{JobStatusRunning, JobStatusCompleted},
		All:    false,
		Limit:  10,
	}

	if len(req.JobIDs) != 2 {
		t.Errorf("Expected 2 job IDs, got %d", len(req.JobIDs))
	}
	if len(req.Status) != 2 {
		t.Errorf("Expected 2 statuses, got %d", len(req.Status))
	}
	if req.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", req.Limit)
	}
}
