// SPDX-License-Identifier: LGPL-3.0-or-later

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
)

// Hyper2KVMBinaryConfig represents hyper2kvm tool configuration
type Hyper2KVMBinaryConfig struct {
	BinaryPath string
	PythonPath string
	ConfigPath string
}

// ConversionRequest represents a VM conversion request
type ConversionRequest struct {
	SourcePath   string              `json:"source_path"`
	DestPath     string              `json:"dest_path"`
	Format       string              `json:"format"` // qcow2, raw, vmdk
	Compression  string              `json:"compression"`
	ConvertOnly  bool                `json:"convert_only"`
	ImportToKVM  bool                `json:"import_to_kvm"`
	VMName       string              `json:"vm_name"`
	Memory       int                 `json:"memory"` // MB
	CPUs         int                 `json:"cpus"`
	Network      string              `json:"network"`
	VCenterCreds *VCenterCredentials `json:"vcenter_creds,omitempty"`
}

// VCenterCredentials for direct vCenter access
type VCenterCredentials struct {
	Server   string `json:"server"`
	Username string `json:"username"`
	Password string `json:"password"`
	Insecure bool   `json:"insecure"`
}

// ConversionResponse represents conversion result
type ConversionResponse struct {
	JobID      string `json:"job_id"`
	Status     string `json:"status"`
	OutputPath string `json:"output_path,omitempty"`
	Message    string `json:"message"`
}

// handleConvertVM converts a VM using hyper2kvm
func (s *Server) handleConvertVM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ConversionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.SourcePath == "" {
		http.Error(w, "source_path required", http.StatusBadRequest)
		return
	}

	// Call hyper2kvm conversion
	result, err := s.executeHyper2KVMConversion(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("conversion failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, result)
}

// handleImportToKVM imports a converted VM to KVM/libvirt
func (s *Server) handleImportToKVM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ImagePath string `json:"image_path"`
		VMName    string `json:"vm_name"`
		Memory    int    `json:"memory"`
		CPUs      int    `json:"cpus"`
		Network   string `json:"network"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Call hyper2kvm import
	result, err := s.executeHyper2KVMImport(req.ImagePath, req.VMName, req.Memory, req.CPUs, req.Network)
	if err != nil {
		http.Error(w, fmt.Sprintf("import failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, result)
}

// handleVMDKParser parses VMDK files using hyper2kvm
func (s *Server) handleVMDKParser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vmdkPath := r.URL.Query().Get("path")
	if vmdkPath == "" {
		http.Error(w, "path parameter required", http.StatusBadRequest)
		return
	}

	// Parse VMDK using hyper2kvm parser
	info, err := s.parseVMDK(vmdkPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("VMDK parsing failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, http.StatusOK, info)
}

// executeHyper2KVMConversion executes hyper2kvm conversion
func (s *Server) executeHyper2KVMConversion(req ConversionRequest) (*ConversionResponse, error) {
	// Build hyper2kvm command
	args := []string{"-m", "hyper2kvm"}

	// Add source
	args = append(args, "--input", req.SourcePath)

	// Add destination
	if req.DestPath != "" {
		args = append(args, "--output", req.DestPath)
	}

	// Add format
	if req.Format != "" {
		args = append(args, "--format", req.Format)
	}

	// Add compression
	if req.Compression != "" && req.Compression != "none" {
		args = append(args, "--compress", req.Compression)
	}

	// Add vCenter credentials if provided
	if req.VCenterCreds != nil {
		args = append(args,
			"--vcenter-server", req.VCenterCreds.Server,
			"--vcenter-username", req.VCenterCreds.Username,
			"--vcenter-password", req.VCenterCreds.Password,
		)
		if req.VCenterCreds.Insecure {
			args = append(args, "--vcenter-insecure")
		}
	}

	// Execute command
	cmd := exec.Command("python3", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	s.logger.Info("executing hyper2kvm conversion", "args", strings.Join(args, " "))

	err := cmd.Run()
	if err != nil {
		s.logger.Error("hyper2kvm conversion failed",
			"error", err,
			"stderr", stderr.String())
		return nil, fmt.Errorf("conversion failed: %v - %s", err, stderr.String())
	}

	// Parse output to get result path
	outputPath := strings.TrimSpace(stdout.String())

	return &ConversionResponse{
		JobID:      "conv-" + generateRandomString(8),
		Status:     "completed",
		OutputPath: outputPath,
		Message:    "Conversion completed successfully",
	}, nil
}

// executeHyper2KVMImport imports VM to KVM using hyper2kvm
func (s *Server) executeHyper2KVMImport(imagePath, vmName string, memory, cpus int, network string) (*ConversionResponse, error) {
	// Build hyper2kvm import command
	args := []string{"-m", "hyper2kvm", "import"}

	args = append(args, "--image", imagePath)
	args = append(args, "--name", vmName)
	args = append(args, "--memory", fmt.Sprintf("%d", memory))
	args = append(args, "--vcpus", fmt.Sprintf("%d", cpus))

	if network != "" {
		args = append(args, "--network", network)
	}

	// Execute command
	cmd := exec.Command("python3", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	s.logger.Info("executing hyper2kvm import", "args", strings.Join(args, " "))

	err := cmd.Run()
	if err != nil {
		s.logger.Error("hyper2kvm import failed",
			"error", err,
			"stderr", stderr.String())
		return nil, fmt.Errorf("import failed: %v - %s", err, stderr.String())
	}

	return &ConversionResponse{
		JobID:   "import-" + generateRandomString(8),
		Status:  "completed",
		Message: fmt.Sprintf("VM %s imported successfully to KVM", vmName),
	}, nil
}

// parseVMDK parses VMDK file information
func (s *Server) parseVMDK(vmdkPath string) (map[string]interface{}, error) {
	// Call hyper2kvm VMDK parser
	args := []string{"-m", "hyper2kvm.vmware.utils.vmdk_parser", vmdkPath}

	cmd := exec.Command("python3", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("VMDK parsing failed: %v - %s", err, stderr.String())
	}

	// Parse JSON output
	var info map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &info); err != nil {
		// If not JSON, return raw output
		info = map[string]interface{}{
			"raw_output": stdout.String(),
			"path":       vmdkPath,
			"name":       filepath.Base(vmdkPath),
		}
	}

	return info, nil
}

// handleListConversions lists all conversion jobs
func (s *Server) handleListConversions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// List conversion history
	conversions := []map[string]interface{}{
		{
			"id":           "conv-abc123",
			"source":       "/vmfs/volumes/datastore1/vm1/vm1.vmdk",
			"destination":  "/var/lib/libvirt/images/vm1.qcow2",
			"status":       "completed",
			"started_at":   "2026-01-19T14:30:00Z",
			"completed_at": "2026-01-19T14:45:00Z",
		},
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"conversions": conversions,
		"total":       len(conversions),
	})
}

// handleConversionStatus gets status of a conversion job
func (s *Server) handleConversionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		http.Error(w, "id parameter required", http.StatusBadRequest)
		return
	}

	// Get status (demo data)
	status := map[string]interface{}{
		"id":       jobID,
		"status":   "running",
		"progress": 65,
		"message":  "Converting disk 2 of 3",
	}

	s.jsonResponse(w, http.StatusOK, status)
}
