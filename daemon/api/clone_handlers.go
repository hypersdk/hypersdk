// SPDX-License-Identifier: LGPL-3.0-or-later

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
)

// handleCloneDomain clones a VM
func (s *Server) handleCloneDomain(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Source    string   `json:"source"`          // Source domain name
		Target    string   `json:"target"`          // Target domain name
		Files     []string `json:"files,omitempty"` // New disk paths (optional)
		NewMAC    bool     `json:"new_mac"`         // Generate new MAC addresses
		AutoClone bool     `json:"auto_clone"`      // Auto-generate clone names
		Preserve  bool     `json:"preserve"`        // Preserve original domain
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Build virt-clone command
	args := []string{
		"--original", req.Source,
		"--name", req.Target,
	}

	// Add custom disk paths if specified
	if len(req.Files) > 0 {
		for _, file := range req.Files {
			args = append(args, "--file", file)
		}
	} else {
		args = append(args, "--auto-clone")
	}

	// MAC address handling
	if req.NewMAC {
		// virt-clone auto-generates new MACs by default
		// No flag needed
	}

	// Preserve original domain (don't undefine after cloning)
	if req.Preserve {
		args = append(args, "--preserve-data")
	}

	cmd := exec.Command("virt-clone", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to clone domain: %s", string(output))
		return
	}

	s.jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("Domain %s cloned to %s", req.Source, req.Target),
		"source":  req.Source,
		"target":  req.Target,
		"output":  string(output),
	})
}

// handleCloneMultipleDomains clones a VM multiple times
func (s *Server) handleCloneMultipleDomains(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Source     string `json:"source"`      // Source domain name
		NamePrefix string `json:"name_prefix"` // Prefix for cloned VMs
		Count      int    `json:"count"`       // Number of clones to create
		StartIndex int    `json:"start_index"` // Starting index (default 1)
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Count <= 0 || req.Count > 100 {
		http.Error(w, "count must be between 1 and 100", http.StatusBadRequest)
		return
	}

	if req.StartIndex == 0 {
		req.StartIndex = 1
	}

	type CloneResult struct {
		Name    string `json:"name"`
		Success bool   `json:"success"`
		Error   string `json:"error,omitempty"`
	}

	results := make([]CloneResult, 0, req.Count)

	for i := 0; i < req.Count; i++ {
		index := req.StartIndex + i
		targetName := fmt.Sprintf("%s-%d", req.NamePrefix, index)

		result := CloneResult{Name: targetName, Success: true}

		args := []string{
			"--original", req.Source,
			"--name", targetName,
			"--auto-clone",
		}

		cmd := exec.Command("virt-clone", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			result.Success = false
			result.Error = string(output)
		}

		results = append(results, result)
	}

	successful := 0
	failed := 0
	for _, r := range results {
		if r.Success {
			successful++
		} else {
			failed++
		}
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":     "completed",
		"source":     req.Source,
		"total":      req.Count,
		"successful": successful,
		"failed":     failed,
		"results":    results,
	})
}

// handleCreateTemplate converts a VM into a template
func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Domain      string            `json:"domain"`      // Source domain
		Name        string            `json:"name"`        // Template name
		Description string            `json:"description"` // Template description
		Metadata    map[string]string `json:"metadata"`    // Additional metadata
		Seal        bool              `json:"seal"`        // Seal the template (virt-sysprep)
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Step 1: Ensure domain is shut down
	shutdownCmd := exec.Command("virsh", "shutdown", req.Domain)
	shutdownCmd.Run() // Ignore error if already shut down

	// Wait a few seconds for shutdown
	exec.Command("sleep", "3").Run()

	// Step 2: Clone the domain to create template
	templateName := req.Name
	if templateName == "" {
		templateName = req.Domain + "-template"
	}

	cloneArgs := []string{
		"--original", req.Domain,
		"--name", templateName,
		"--auto-clone",
	}

	cloneCmd := exec.Command("virt-clone", cloneArgs...)
	cloneOutput, err := cloneCmd.CombinedOutput()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to clone domain: %s", string(cloneOutput))
		return
	}

	// Step 3: Seal the template if requested (remove machine-specific config)
	if req.Seal {
		sealCmd := exec.Command("virt-sysprep", "-d", templateName)
		sealOutput, err := sealCmd.CombinedOutput()
		if err != nil {
			// If virt-sysprep fails, warn but continue
			s.logger.Warn("virt-sysprep failed", "error", string(sealOutput))
		}
	}

	// Step 4: Add metadata to template
	if req.Description != "" {
		descCmd := exec.Command("virsh", "desc", templateName, "--title", req.Description)
		descCmd.Run() // Ignore errors
	}

	s.jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"status":      "success",
		"message":     fmt.Sprintf("Template %s created from %s", templateName, req.Domain),
		"template":    templateName,
		"source":      req.Domain,
		"sealed":      req.Seal,
		"description": req.Description,
	})
}

// handleDeployFromTemplate deploys a new VM from a template
func (s *Server) handleDeployFromTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Template  string            `json:"template"`  // Template name
		Name      string            `json:"name"`      // New VM name
		Memory    int               `json:"memory"`    // Memory in MB (optional)
		VCPUs     int               `json:"vcpus"`     // Number of CPUs (optional)
		Network   string            `json:"network"`   // Network name (optional)
		AutoStart bool              `json:"autostart"` // Start after creation
		Customize map[string]string `json:"customize"` // Customization options
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Clone template to create new VM
	cloneArgs := []string{
		"--original", req.Template,
		"--name", req.Name,
		"--auto-clone",
	}

	cloneCmd := exec.Command("virt-clone", cloneArgs...)
	cloneOutput, err := cloneCmd.CombinedOutput()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to deploy from template: %s", string(cloneOutput))
		return
	}

	// Customize VM if requested
	if req.Memory > 0 {
		memCmd := exec.Command("virsh", "setmaxmem", req.Name, fmt.Sprintf("%dM", req.Memory), "--config")
		memCmd.Run()
		memCmd = exec.Command("virsh", "setmem", req.Name, fmt.Sprintf("%dM", req.Memory), "--config")
		memCmd.Run()
	}

	if req.VCPUs > 0 {
		vcpuCmd := exec.Command("virsh", "setvcpus", req.Name, fmt.Sprintf("%d", req.VCPUs), "--config", "--maximum")
		vcpuCmd.Run()
		vcpuCmd = exec.Command("virsh", "setvcpus", req.Name, fmt.Sprintf("%d", req.VCPUs), "--config")
		vcpuCmd.Run()
	}

	// Start VM if requested
	if req.AutoStart {
		startCmd := exec.Command("virsh", "start", req.Name)
		startCmd.Run()
	}

	// Enable autostart if requested
	autostartCmd := exec.Command("virsh", "autostart", req.Name)
	autostartCmd.Run()

	s.jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"status":    "success",
		"message":   fmt.Sprintf("VM %s deployed from template %s", req.Name, req.Template),
		"vm":        req.Name,
		"template":  req.Template,
		"started":   req.AutoStart,
		"memory_mb": req.Memory,
		"vcpus":     req.VCPUs,
	})
}

// handleListTemplates lists all VM templates
func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get all domains
	cmd := exec.Command("virsh", "list", "--all", "--name")
	output, err := cmd.Output()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to list domains: %v", err)
		return
	}

	lines := strings.Split(string(output), "\n")
	var templates []string

	// Filter domains that are likely templates (contain "template" in name or are shut off)
	for _, line := range lines {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}

		// Check if domain name contains "template"
		if strings.Contains(strings.ToLower(name), "template") {
			templates = append(templates, name)
		}
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"templates": templates,
		"total":     len(templates),
	})
}

// handleExportTemplate exports a template for backup/sharing
func (s *Server) handleExportTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Template   string `json:"template"`    // Template name
		ExportPath string `json:"export_path"` // Export directory
		Compress   bool   `json:"compress"`    // Compress export
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Export domain XML
	xmlPath := fmt.Sprintf("%s/%s.xml", req.ExportPath, req.Template)
	dumpCmd := exec.Command("virsh", "dumpxml", req.Template)
	xmlOutput, err := dumpCmd.Output()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to dump XML: %v", err)
		return
	}

	// Write XML to file
	writeCmd := exec.Command("bash", "-c", fmt.Sprintf("cat > %s", xmlPath))
	writeCmd.Stdin = strings.NewReader(string(xmlOutput))
	if err := writeCmd.Run(); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to write XML: %v", err)
		return
	}

	// Get disk paths
	domblklistCmd := exec.Command("virsh", "domblklist", req.Template)
	domblklistOutput, err := domblklistCmd.Output()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to list disks: %v", err)
		return
	}

	var diskPaths []string
	lines := strings.Split(string(domblklistOutput), "\n")
	for i, line := range lines {
		if i < 2 || strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			diskPaths = append(diskPaths, fields[1])
		}
	}

	// Copy disk files
	for _, diskPath := range diskPaths {
		cpCmd := exec.Command("cp", diskPath, req.ExportPath)
		cpCmd.Run() // Ignore errors
	}

	// Compress if requested
	if req.Compress {
		tarCmd := exec.Command("tar", "-czf",
			fmt.Sprintf("%s/%s.tar.gz", req.ExportPath, req.Template),
			"-C", req.ExportPath,
			fmt.Sprintf("%s.xml", req.Template))
		tarCmd.Run() // Ignore errors
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":      "success",
		"message":     fmt.Sprintf("Template %s exported to %s", req.Template, req.ExportPath),
		"template":    req.Template,
		"export_path": req.ExportPath,
		"xml_file":    xmlPath,
		"disk_files":  diskPaths,
		"compressed":  req.Compress,
	})
}
