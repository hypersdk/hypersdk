// SPDX-License-Identifier: LGPL-3.0-or-later

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// BatchOperationResult represents the result of a batch operation
type BatchOperationResult struct {
	Operation  string           `json:"operation"`
	Total      int              `json:"total"`
	Successful int              `json:"successful"`
	Failed     int              `json:"failed"`
	Results    []DomainOpResult `json:"results"`
	StartTime  time.Time        `json:"start_time"`
	EndTime    time.Time        `json:"end_time"`
	Duration   string           `json:"duration"`
}

// DomainOpResult represents the result of an operation on a single domain
type DomainOpResult struct {
	Domain  string `json:"domain"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// handleBatchStart starts multiple VMs
func (s *Server) handleBatchStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Domains []string `json:"domains"`
		Paused  bool     `json:"paused"` // Start in paused state
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	startTime := time.Now()
	results := s.executeBatchOperation("start", req.Domains, func(domain string) error {
		args := []string{"start", domain}
		if req.Paused {
			args = append(args, "--paused")
		}
		cmd := exec.Command("virsh", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s", string(output))
		}
		return nil
	})
	endTime := time.Now()

	result := s.buildBatchResult("start", req.Domains, results, startTime, endTime)
	s.jsonResponse(w, http.StatusOK, result)
}

// handleBatchStop stops multiple VMs gracefully
func (s *Server) handleBatchStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Domains []string `json:"domains"`
		Force   bool     `json:"force"` // Use destroy instead of shutdown
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	operation := "shutdown"
	if req.Force {
		operation = "destroy"
	}

	startTime := time.Now()
	results := s.executeBatchOperation(operation, req.Domains, func(domain string) error {
		cmd := exec.Command("virsh", operation, domain)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s", string(output))
		}
		return nil
	})
	endTime := time.Now()

	result := s.buildBatchResult(operation, req.Domains, results, startTime, endTime)
	s.jsonResponse(w, http.StatusOK, result)
}

// handleBatchReboot reboots multiple VMs
func (s *Server) handleBatchReboot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Domains []string `json:"domains"`
		Force   bool     `json:"force"` // Hard reset instead of graceful reboot
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	operation := "reboot"
	if req.Force {
		operation = "reset"
	}

	startTime := time.Now()
	results := s.executeBatchOperation(operation, req.Domains, func(domain string) error {
		cmd := exec.Command("virsh", operation, domain)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s", string(output))
		}
		return nil
	})
	endTime := time.Now()

	result := s.buildBatchResult(operation, req.Domains, results, startTime, endTime)
	s.jsonResponse(w, http.StatusOK, result)
}

// handleBatchSnapshot creates snapshots for multiple VMs
func (s *Server) handleBatchSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Domains     []string `json:"domains"`
		NamePrefix  string   `json:"name_prefix"` // Snapshot name prefix (e.g., "backup")
		Description string   `json:"description"`
		Atomic      bool     `json:"atomic"`    // Create snapshot atomically
		DiskOnly    bool     `json:"disk_only"` // Disk-only snapshot (no memory)
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.NamePrefix == "" {
		req.NamePrefix = "snapshot"
	}

	timestamp := time.Now().Format("20060102-150405")

	startTime := time.Now()
	results := s.executeBatchOperation("snapshot-create-as", req.Domains, func(domain string) error {
		snapshotName := fmt.Sprintf("%s-%s-%s", req.NamePrefix, domain, timestamp)

		args := []string{"snapshot-create-as", domain, snapshotName}
		if req.Description != "" {
			args = append(args, "--description", req.Description)
		}
		if req.Atomic {
			args = append(args, "--atomic")
		}
		if req.DiskOnly {
			args = append(args, "--disk-only")
		}

		cmd := exec.Command("virsh", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s", string(output))
		}
		return nil
	})
	endTime := time.Now()

	result := s.buildBatchResult("snapshot", req.Domains, results, startTime, endTime)
	s.jsonResponse(w, http.StatusOK, result)
}

// handleBatchDelete deletes multiple VMs
func (s *Server) handleBatchDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Domains       []string `json:"domains"`
		RemoveStorage bool     `json:"remove_storage"` // Delete storage volumes
		SnapshotsOnly bool     `json:"snapshots_only"` // Only delete snapshots
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	operation := "undefine"

	startTime := time.Now()
	results := s.executeBatchOperation(operation, req.Domains, func(domain string) error {
		// First, destroy if running
		destroyCmd := exec.Command("virsh", "destroy", domain)
		destroyCmd.Run() // Ignore error if already stopped

		// Delete snapshots if requested or if removing storage
		if req.SnapshotsOnly || req.RemoveStorage {
			snapshotListCmd := exec.Command("virsh", "snapshot-list", domain, "--name")
			if snapshotOutput, err := snapshotListCmd.Output(); err == nil {
				snapshots := parseSnapshotList(string(snapshotOutput))
				for _, snapshot := range snapshots {
					deleteSnapshotCmd := exec.Command("virsh", "snapshot-delete", domain, snapshot)
					deleteSnapshotCmd.Run() // Ignore errors
				}
			}
		}

		// If only deleting snapshots, don't undefine the domain
		if req.SnapshotsOnly {
			return nil
		}

		// Undefine domain
		args := []string{"undefine", domain}
		if req.RemoveStorage {
			args = append(args, "--remove-all-storage")
		}

		cmd := exec.Command("virsh", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s", string(output))
		}
		return nil
	})
	endTime := time.Now()

	result := s.buildBatchResult(operation, req.Domains, results, startTime, endTime)
	s.jsonResponse(w, http.StatusOK, result)
}

// handleBatchPause pauses multiple VMs
func (s *Server) handleBatchPause(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Domains []string `json:"domains"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	startTime := time.Now()
	results := s.executeBatchOperation("suspend", req.Domains, func(domain string) error {
		cmd := exec.Command("virsh", "suspend", domain)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s", string(output))
		}
		return nil
	})
	endTime := time.Now()

	result := s.buildBatchResult("suspend", req.Domains, results, startTime, endTime)
	s.jsonResponse(w, http.StatusOK, result)
}

// handleBatchResume resumes multiple VMs
func (s *Server) handleBatchResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Domains []string `json:"domains"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	startTime := time.Now()
	results := s.executeBatchOperation("resume", req.Domains, func(domain string) error {
		cmd := exec.Command("virsh", "resume", domain)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("%s", string(output))
		}
		return nil
	})
	endTime := time.Now()

	result := s.buildBatchResult("resume", req.Domains, results, startTime, endTime)
	s.jsonResponse(w, http.StatusOK, result)
}

// executeBatchOperation executes an operation on multiple domains concurrently
func (s *Server) executeBatchOperation(operation string, domains []string, opFunc func(string) error) []DomainOpResult {
	var wg sync.WaitGroup
	results := make([]DomainOpResult, len(domains))

	for i, domain := range domains {
		wg.Add(1)
		go func(index int, dom string) {
			defer wg.Done()

			result := DomainOpResult{
				Domain:  dom,
				Success: true,
			}

			if err := opFunc(dom); err != nil {
				result.Success = false
				result.Error = err.Error()
			}

			results[index] = result
		}(i, domain)
	}

	wg.Wait()
	return results
}

// buildBatchResult builds a BatchOperationResult from individual results
func (s *Server) buildBatchResult(operation string, domains []string, results []DomainOpResult, startTime, endTime time.Time) BatchOperationResult {
	successful := 0
	failed := 0

	for _, result := range results {
		if result.Success {
			successful++
		} else {
			failed++
		}
	}

	return BatchOperationResult{
		Operation:  operation,
		Total:      len(domains),
		Successful: successful,
		Failed:     failed,
		Results:    results,
		StartTime:  startTime,
		EndTime:    endTime,
		Duration:   endTime.Sub(startTime).String(),
	}
}

// parseSnapshotList parses virsh snapshot-list output
func parseSnapshotList(output string) []string {
	lines := strings.Split(output, "\n")
	var snapshots []string

	for _, line := range lines {
		snapshot := strings.TrimSpace(line)
		if snapshot != "" {
			snapshots = append(snapshots, snapshot)
		}
	}

	return snapshots
}
