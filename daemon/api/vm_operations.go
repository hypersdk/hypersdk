// SPDX-License-Identifier: LGPL-3.0-or-later

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// VMOperationRequest represents a request to perform an operation on a VM
type VMOperationRequest struct {
	VMPath  string `json:"vm_path"`
	Timeout int    `json:"timeout,omitempty"` // timeout in seconds
}

// VMOperationResponse represents the response from a VM operation
type VMOperationResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

// Handle VM shutdown
func (s *Server) handleVMShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VMOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: %v", err)
		return
	}

	if req.VMPath == "" {
		s.errorResponse(w, http.StatusBadRequest, "vm_path is required")
		return
	}

	// Default timeout
	timeout := 5 * time.Minute
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout) * time.Second
	}

	// Get vSphere client
	client, err := s.manager.GetVSphereClient()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to connect to vSphere: %v", err)
		return
	}
	defer client.Close()

	// Shutdown VM
	ctx := context.Background()
	if err := client.ShutdownVM(ctx, req.VMPath, timeout); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to shutdown VM: %v", err)
		return
	}

	resp := VMOperationResponse{
		Success:   true,
		Message:   "VM shutdown initiated successfully",
		Timestamp: time.Now(),
	}

	s.jsonResponse(w, http.StatusOK, resp)
}

// Handle VM power off
func (s *Server) handleVMPowerOff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VMOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: %v", err)
		return
	}

	if req.VMPath == "" {
		s.errorResponse(w, http.StatusBadRequest, "vm_path is required")
		return
	}

	// Get vSphere client
	client, err := s.manager.GetVSphereClient()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to connect to vSphere: %v", err)
		return
	}
	defer client.Close()

	// Power off VM
	ctx := context.Background()
	if err := client.PowerOffVM(ctx, req.VMPath); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to power off VM: %v", err)
		return
	}

	resp := VMOperationResponse{
		Success:   true,
		Message:   "VM powered off successfully",
		Timestamp: time.Now(),
	}

	s.jsonResponse(w, http.StatusOK, resp)
}

// Handle CD/DVD removal
func (s *Server) handleVMRemoveCDROM(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VMOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: %v", err)
		return
	}

	if req.VMPath == "" {
		s.errorResponse(w, http.StatusBadRequest, "vm_path is required")
		return
	}

	// Get vSphere client
	client, err := s.manager.GetVSphereClient()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to connect to vSphere: %v", err)
		return
	}
	defer client.Close()

	// Remove CD/DVD devices
	ctx := context.Background()
	if err := client.RemoveCDROMDevices(ctx, req.VMPath); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to remove CD/DVD devices: %v", err)
		return
	}

	resp := VMOperationResponse{
		Success:   true,
		Message:   "CD/DVD devices removed successfully",
		Timestamp: time.Now(),
	}

	s.jsonResponse(w, http.StatusOK, resp)
}

// Handle VM info retrieval
func (s *Server) handleVMInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req VMOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request: %v", err)
		return
	}

	if req.VMPath == "" {
		s.errorResponse(w, http.StatusBadRequest, "vm_path is required")
		return
	}

	// Get vSphere client
	client, err := s.manager.GetVSphereClient()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to connect to vSphere: %v", err)
		return
	}
	defer client.Close()

	// Get VM info
	ctx := context.Background()
	info, err := client.GetVMInfo(ctx, req.VMPath)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to get VM info: %v", err)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"vm_info":   info,
		"timestamp": time.Now(),
	})
}
