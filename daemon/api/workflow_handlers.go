// SPDX-License-Identifier: LGPL-3.0-or-later

package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// ConversionWorkflow represents a complete conversion workflow
type ConversionWorkflow struct {
	ID            string                 `json:"id"`
	Status        string                 `json:"status"` // pending, config_generated, converting, converted, importing, completed, failed
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	Request       ConfigGeneratorRequest `json:"request"`
	ConfigYAML    string                 `json:"config_yaml,omitempty"`
	ConfigPath    string                 `json:"config_path,omitempty"`
	ServiceFile   string                 `json:"service_file,omitempty"`
	ServicePath   string                 `json:"service_path,omitempty"`
	ConversionLog string                 `json:"conversion_log,omitempty"`
	LibvirtDomain string                 `json:"libvirt_domain,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Steps         []WorkflowStep         `json:"steps"`
}

// WorkflowStep represents a step in the workflow
type WorkflowStep struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"` // pending, running, completed, failed
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Error       string    `json:"error,omitempty"`
	Output      string    `json:"output,omitempty"`
}

// WorkflowManager manages conversion workflows
type WorkflowManager struct {
	workflows map[string]*ConversionWorkflow
	mu        sync.RWMutex
}

var globalWorkflowManager = &WorkflowManager{
	workflows: make(map[string]*ConversionWorkflow),
}

// handleConversionWorkflow starts a complete conversion workflow
func (s *Server) handleConversionWorkflow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ConfigGeneratorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.OSType == "" || req.OSFlavor == "" || req.VMDKPath == "" || req.OutputDir == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	// Create workflow
	workflow := &ConversionWorkflow{
		ID:        fmt.Sprintf("workflow-%d", time.Now().Unix()),
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Request:   req,
		Steps: []WorkflowStep{
			{Name: "Generate Configuration", Status: "pending"},
			{Name: "Write Config Files", Status: "pending"},
			{Name: "Run hyper2kvm Conversion", Status: "pending"},
			{Name: "Import to Libvirt", Status: "pending"},
			{Name: "Start VM (if requested)", Status: "pending"},
		},
	}

	// Store workflow
	globalWorkflowManager.mu.Lock()
	globalWorkflowManager.workflows[workflow.ID] = workflow
	globalWorkflowManager.mu.Unlock()

	// Start workflow in background
	go s.executeWorkflow(workflow)

	s.jsonResponse(w, http.StatusAccepted, map[string]interface{}{
		"workflow_id": workflow.ID,
		"status":      workflow.Status,
		"message":     "Conversion workflow started",
	})
}

// handleWorkflowStatus gets workflow status
func (s *Server) handleWorkflowStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workflowID := r.URL.Query().Get("id")
	if workflowID == "" {
		// List all workflows
		globalWorkflowManager.mu.RLock()
		workflows := make([]*ConversionWorkflow, 0, len(globalWorkflowManager.workflows))
		for _, wf := range globalWorkflowManager.workflows {
			workflows = append(workflows, wf)
		}
		globalWorkflowManager.mu.RUnlock()

		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"workflows": workflows,
			"total":     len(workflows),
		})
		return
	}

	// Get specific workflow
	globalWorkflowManager.mu.RLock()
	workflow, exists := globalWorkflowManager.workflows[workflowID]
	globalWorkflowManager.mu.RUnlock()

	if !exists {
		http.Error(w, "workflow not found", http.StatusNotFound)
		return
	}

	s.jsonResponse(w, http.StatusOK, workflow)
}

// executeWorkflow executes the complete conversion workflow
func (s *Server) executeWorkflow(workflow *ConversionWorkflow) {
	defer func() {
		workflow.UpdatedAt = time.Now()
		if r := recover(); r != nil {
			workflow.Status = "failed"
			workflow.Error = fmt.Sprintf("panic: %v", r)
		}
	}()

	// Step 1: Generate Configuration
	if err := s.workflowStep1GenerateConfig(workflow); err != nil {
		workflow.Status = "failed"
		workflow.Error = err.Error()
		return
	}

	// Step 2: Write Config Files
	if err := s.workflowStep2WriteFiles(workflow); err != nil {
		workflow.Status = "failed"
		workflow.Error = err.Error()
		return
	}

	// Step 3: Run Conversion
	if err := s.workflowStep3RunConversion(workflow); err != nil {
		workflow.Status = "failed"
		workflow.Error = err.Error()
		return
	}

	// Step 4: Import to Libvirt
	if err := s.workflowStep4ImportLibvirt(workflow); err != nil {
		workflow.Status = "failed"
		workflow.Error = err.Error()
		return
	}

	// Step 5: Start VM (if requested)
	if workflow.Request.LibvirtTest {
		if err := s.workflowStep5StartVM(workflow); err != nil {
			workflow.Status = "failed"
			workflow.Error = err.Error()
			return
		}
	} else {
		workflow.Steps[4].Status = "skipped"
	}

	workflow.Status = "completed"
	workflow.UpdatedAt = time.Now()
}

// workflowStep1GenerateConfig generates the config
func (s *Server) workflowStep1GenerateConfig(workflow *ConversionWorkflow) error {
	step := &workflow.Steps[0]
	step.Status = "running"
	step.StartedAt = time.Now()
	workflow.Status = "config_generated"

	// Build config
	config := s.buildHyper2KVMConfig(workflow.Request)

	// Convert to YAML
	yamlBytes, err := yaml.Marshal(config)
	if err != nil {
		step.Status = "failed"
		step.Error = err.Error()
		return fmt.Errorf("failed to generate YAML: %v", err)
	}

	// Add header
	yamlContent := s.generateConfigHeader(workflow.Request) + string(yamlBytes)

	workflow.ConfigYAML = yamlContent
	workflow.ConfigPath = filepath.Join(workflow.Request.OutputDir, fmt.Sprintf("hyper2kvm-%s.yaml", strings.ToLower(workflow.Request.VMName)))

	// Generate systemd service if requested
	if workflow.Request.GenerateService {
		workflow.ServiceFile = s.generateSystemdService(workflow.Request, workflow.ConfigPath)
		workflow.ServicePath = fmt.Sprintf("/etc/systemd/system/hyper2kvm-%s.service", strings.ToLower(workflow.Request.VMName))
	}

	step.Status = "completed"
	step.CompletedAt = time.Now()
	step.Output = fmt.Sprintf("Config generated: %s", workflow.ConfigPath)
	return nil
}

// workflowStep2WriteFiles writes config files to disk
func (s *Server) workflowStep2WriteFiles(workflow *ConversionWorkflow) error {
	step := &workflow.Steps[1]
	step.Status = "running"
	step.StartedAt = time.Now()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(workflow.Request.OutputDir, 0755); err != nil {
		step.Status = "failed"
		step.Error = err.Error()
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Write config file
	if err := ioutil.WriteFile(workflow.ConfigPath, []byte(workflow.ConfigYAML), 0644); err != nil {
		step.Status = "failed"
		step.Error = err.Error()
		return fmt.Errorf("failed to write config file: %v", err)
	}

	step.Status = "completed"
	step.CompletedAt = time.Now()
	step.Output = fmt.Sprintf("Config written to %s", workflow.ConfigPath)
	return nil
}

// workflowStep3RunConversion runs the hyper2kvm conversion
func (s *Server) workflowStep3RunConversion(workflow *ConversionWorkflow) error {
	step := &workflow.Steps[2]
	step.Status = "running"
	step.StartedAt = time.Now()
	workflow.Status = "converting"

	// Run hyper2kvm
	cmd := exec.Command("python3", "-m", "hyper2kvm", "--config", workflow.ConfigPath, "local")
	cmd.Dir = workflow.Request.OutputDir

	output, err := cmd.CombinedOutput()
	workflow.ConversionLog = string(output)

	if err != nil {
		step.Status = "failed"
		step.Error = string(output)
		return fmt.Errorf("conversion failed: %v - %s", err, string(output))
	}

	step.Status = "completed"
	step.CompletedAt = time.Now()
	step.Output = "Conversion completed successfully"
	workflow.Status = "converted"
	return nil
}

// workflowStep4ImportLibvirt imports the converted image to libvirt
func (s *Server) workflowStep4ImportLibvirt(workflow *ConversionWorkflow) error {
	step := &workflow.Steps[3]
	step.Status = "running"
	step.StartedAt = time.Now()
	workflow.Status = "importing"

	// Find the converted image
	convertedImage := filepath.Join(workflow.Request.OutputDir, fmt.Sprintf("%s-fixed.qcow2", strings.ToLower(workflow.Request.VMName)))
	if workflow.Request.VMName == "" {
		convertedImage = filepath.Join(workflow.Request.OutputDir, fmt.Sprintf("%s-%s-fixed.qcow2", workflow.Request.OSType, workflow.Request.OSFlavor))
	}

	// Check if image exists
	if _, err := os.Stat(convertedImage); os.IsNotExist(err) {
		step.Status = "failed"
		step.Error = fmt.Sprintf("converted image not found: %s", convertedImage)
		return fmt.Errorf("converted image not found: %s", convertedImage)
	}

	// Use virt-install to create domain
	vmName := workflow.Request.VMName
	if vmName == "" {
		vmName = fmt.Sprintf("%s-%s", workflow.Request.OSType, workflow.Request.OSFlavor)
	}

	args := []string{
		"--name", vmName,
		"--memory", fmt.Sprintf("%d", workflow.Request.Memory),
		"--vcpus", fmt.Sprintf("%d", workflow.Request.VCPUs),
		"--disk", fmt.Sprintf("path=%s,format=qcow2", convertedImage),
		"--import",
		"--os-variant", "generic",
		"--network", "default",
		"--graphics", "vnc,listen=0.0.0.0",
		"--noautoconsole",
	}

	if workflow.Request.UEFI {
		args = append(args, "--boot", "uefi")
	}

	cmd := exec.Command("virt-install", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		step.Status = "failed"
		step.Error = string(output)
		return fmt.Errorf("libvirt import failed: %v - %s", err, string(output))
	}

	workflow.LibvirtDomain = vmName
	step.Status = "completed"
	step.CompletedAt = time.Now()
	step.Output = fmt.Sprintf("VM imported as %s", vmName)
	return nil
}

// workflowStep5StartVM starts the VM
func (s *Server) workflowStep5StartVM(workflow *ConversionWorkflow) error {
	step := &workflow.Steps[4]
	step.Status = "running"
	step.StartedAt = time.Now()

	// VM should already be running after virt-install --import
	// But check status
	cmd := exec.Command("virsh", "domstate", workflow.LibvirtDomain)
	output, err := cmd.Output()

	if err != nil {
		step.Status = "failed"
		step.Error = fmt.Sprintf("failed to check VM state: %v", err)
		return fmt.Errorf("failed to check VM state: %v", err)
	}

	state := strings.TrimSpace(string(output))
	step.Status = "completed"
	step.CompletedAt = time.Now()
	step.Output = fmt.Sprintf("VM is %s", state)

	return nil
}
