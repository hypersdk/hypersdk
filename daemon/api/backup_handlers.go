// SPDX-License-Identifier: LGPL-3.0-or-later

package api

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// Backup storage directory (configurable via environment variable)
var BackupStorageDir = getBackupStorageDir()

func getBackupStorageDir() string {
	if dir := os.Getenv("HYPERSDK_BACKUP_DIR"); dir != "" {
		return dir
	}
	return "/var/lib/libvirt/backups"
}

// BackupInfo represents a backup
type BackupInfo struct {
	Name       string    `json:"name"`
	VMName     string    `json:"vm_name"`
	Path       string    `json:"path"`
	Type       string    `json:"type"` // full, incremental
	Size       int64     `json:"size"`
	CreatedAt  time.Time `json:"created_at"`
	Compressed bool      `json:"compressed"`
	Verified   bool      `json:"verified"`
}

// handleCreateBackup creates a full VM backup
func (s *Server) handleCreateBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		VMName      string `json:"vm_name"`
		BackupName  string `json:"backup_name,omitempty"` // Optional custom name
		Compress    bool   `json:"compress"`
		Description string `json:"description,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Generate backup name if not provided
	if req.BackupName == "" {
		req.BackupName = fmt.Sprintf("%s-%s", req.VMName, time.Now().Format("20060102-150405"))
	}

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(BackupStorageDir, 0755); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to create backup directory: %v", err)
		return
	}

	backupDir := filepath.Join(BackupStorageDir, req.BackupName)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to create backup subdirectory: %v", err)
		return
	}

	// Dump VM XML definition
	xmlPath := filepath.Join(backupDir, "domain.xml")
	dumpCmd := exec.Command("virsh", "dumpxml", req.VMName)
	xmlOutput, err := dumpCmd.Output()
	if err != nil {
		os.RemoveAll(backupDir) // Cleanup
		s.errorResponse(w, http.StatusInternalServerError, "failed to dump VM XML: %v", err)
		return
	}

	if err := os.WriteFile(xmlPath, xmlOutput, 0644); err != nil {
		os.RemoveAll(backupDir)
		s.errorResponse(w, http.StatusInternalServerError, "failed to write XML: %v", err)
		return
	}

	// Get disk paths
	domblklistCmd := exec.Command("virsh", "domblklist", req.VMName, "--details")
	domblklistOutput, err := domblklistCmd.Output()
	if err != nil {
		os.RemoveAll(backupDir)
		s.errorResponse(w, http.StatusInternalServerError, "failed to list disks: %v", err)
		return
	}

	// Parse disk list and copy disks
	disks := parseDiskList(string(domblklistOutput))
	backedUpDisks := []string{}

	for _, disk := range disks {
		if disk == "" || !fileExists(disk) {
			continue
		}

		diskName := filepath.Base(disk)
		backupDiskPath := filepath.Join(backupDir, diskName)

		// Copy disk using qemu-img convert
		var convertCmd *exec.Cmd
		if req.Compress {
			convertCmd = exec.Command("qemu-img", "convert", "-O", "qcow2", "-c", disk, backupDiskPath)
		} else {
			convertCmd = exec.Command("qemu-img", "convert", "-O", "qcow2", disk, backupDiskPath)
		}

		if output, err := convertCmd.CombinedOutput(); err != nil {
			s.logger.Warn("failed to backup disk", "disk", disk, "error", string(output))
			continue
		}

		backedUpDisks = append(backedUpDisks, backupDiskPath)
	}

	// Write metadata
	metadata := map[string]interface{}{
		"vm_name":     req.VMName,
		"backup_name": req.BackupName,
		"created_at":  time.Now(),
		"type":        "full",
		"compressed":  req.Compress,
		"description": req.Description,
		"disks":       backedUpDisks,
	}

	metadataPath := filepath.Join(backupDir, "metadata.json")
	metadataBytes, _ := json.MarshalIndent(metadata, "", "  ")
	os.WriteFile(metadataPath, metadataBytes, 0644)

	// Calculate total backup size
	totalSize, _ := calculateDirectorySize(backupDir)

	s.jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"status":          "success",
		"message":         fmt.Sprintf("Backup created for %s", req.VMName),
		"vm_name":         req.VMName,
		"backup_name":     req.BackupName,
		"backup_path":     backupDir,
		"disks_backed_up": len(backedUpDisks),
		"total_size":      totalSize,
		"compressed":      req.Compress,
	})
}

// handleListBackups lists all available backups
func (s *Server) handleListBackups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	vmName := r.URL.Query().Get("vm_name") // Optional filter by VM

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(BackupStorageDir, 0755); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to access backup directory: %v", err)
		return
	}

	entries, err := os.ReadDir(BackupStorageDir)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to read backup directory: %v", err)
		return
	}

	backups := []BackupInfo{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		backupDir := filepath.Join(BackupStorageDir, entry.Name())
		metadataPath := filepath.Join(backupDir, "metadata.json")

		// Read metadata
		var metadata map[string]interface{}
		if data, err := os.ReadFile(metadataPath); err == nil {
			json.Unmarshal(data, &metadata)
		}

		// Get backup info
		info, _ := entry.Info()
		backupVMName := ""
		backupType := "full"
		compressed := false
		verified := false

		if metadata != nil {
			if vm, ok := metadata["vm_name"].(string); ok {
				backupVMName = vm
			}
			if t, ok := metadata["type"].(string); ok {
				backupType = t
			}
			if c, ok := metadata["compressed"].(bool); ok {
				compressed = c
			}
			if v, ok := metadata["verified"].(bool); ok {
				verified = v
			}
		}

		// Filter by VM name if specified
		if vmName != "" && backupVMName != vmName {
			continue
		}

		// Calculate total size
		totalSize, _ := calculateDirectorySize(backupDir)

		backups = append(backups, BackupInfo{
			Name:       entry.Name(),
			VMName:     backupVMName,
			Path:       backupDir,
			Type:       backupType,
			Size:       totalSize,
			CreatedAt:  info.ModTime(),
			Compressed: compressed,
			Verified:   verified,
		})
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"backups":     backups,
		"total":       len(backups),
		"storage_dir": BackupStorageDir,
	})
}

// handleRestoreBackup restores a VM from backup
func (s *Server) handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		BackupName string `json:"backup_name"`
		NewVMName  string `json:"new_vm_name,omitempty"` // Optional: restore with different name
		Start      bool   `json:"start"`                 // Start VM after restore
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	backupDir := filepath.Join(BackupStorageDir, req.BackupName)
	xmlPath := filepath.Join(backupDir, "domain.xml")

	// Check if backup exists
	if !fileExists(xmlPath) {
		s.errorResponse(w, http.StatusNotFound, "backup not found: %s", req.BackupName)
		return
	}

	// Read XML
	xmlBytes, err := os.ReadFile(xmlPath)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to read backup XML: %v", err)
		return
	}

	xml := string(xmlBytes)

	// Replace VM name if requested
	if req.NewVMName != "" {
		var err error
		xml, err = replaceVMNameInXML(xml, req.NewVMName)
		if err != nil {
			s.errorResponse(w, http.StatusBadRequest, "failed to replace VM name: %v", err)
			return
		}
	}

	// Define the VM
	defineCmd := exec.Command("virsh", "define", "/dev/stdin")
	defineCmd.Stdin = strings.NewReader(xml)
	defineOutput, err := defineCmd.CombinedOutput()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to define VM: %s", string(defineOutput))
		return
	}

	// Read metadata to get VM name
	var metadata map[string]interface{}
	metadataPath := filepath.Join(backupDir, "metadata.json")
	if data, err := os.ReadFile(metadataPath); err == nil {
		json.Unmarshal(data, &metadata)
	}

	vmName := req.NewVMName
	if vmName == "" && metadata != nil {
		if vm, ok := metadata["vm_name"].(string); ok {
			vmName = vm
		}
	}

	// Start VM if requested
	if req.Start && vmName != "" {
		startCmd := exec.Command("virsh", "start", vmName)
		startCmd.Run() // Ignore error if already running
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":      "success",
		"message":     fmt.Sprintf("VM restored from backup %s", req.BackupName),
		"backup_name": req.BackupName,
		"vm_name":     vmName,
		"started":     req.Start,
	})
}

// handleVerifyBackup verifies backup integrity
func (s *Server) handleVerifyBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		BackupName string `json:"backup_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	backupDir := filepath.Join(BackupStorageDir, req.BackupName)
	xmlPath := filepath.Join(backupDir, "domain.xml")

	if !fileExists(xmlPath) {
		s.errorResponse(w, http.StatusNotFound, "backup not found: %s", req.BackupName)
		return
	}

	// Verify XML is valid
	validateCmd := exec.Command("virsh", "define", "--validate", xmlPath)
	if output, err := validateCmd.CombinedOutput(); err != nil {
		s.jsonResponse(w, http.StatusOK, map[string]interface{}{
			"status":      "failed",
			"backup_name": req.BackupName,
			"valid":       false,
			"error":       string(output),
		})
		return
	}

	// Check disk files with qemu-img
	diskErrors := []string{}
	disksChecked := 0

	filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		// Check qcow2 files
		if strings.HasSuffix(strings.ToLower(path), ".qcow2") {
			disksChecked++
			checkCmd := exec.Command("qemu-img", "check", path)
			if output, err := checkCmd.CombinedOutput(); err != nil {
				diskErrors = append(diskErrors, fmt.Sprintf("%s: %s", filepath.Base(path), string(output)))
			}
		}

		return nil
	})

	// Update metadata with verification result
	metadataPath := filepath.Join(backupDir, "metadata.json")
	var metadata map[string]interface{}
	if data, err := os.ReadFile(metadataPath); err == nil {
		json.Unmarshal(data, &metadata)
		metadata["verified"] = len(diskErrors) == 0
		metadata["verified_at"] = time.Now()
		if updatedBytes, err := json.MarshalIndent(metadata, "", "  "); err == nil {
			os.WriteFile(metadataPath, updatedBytes, 0644)
		}
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":        "success",
		"backup_name":   req.BackupName,
		"valid":         len(diskErrors) == 0,
		"disks_checked": disksChecked,
		"errors":        diskErrors,
	})
}

// handleDeleteBackup deletes a backup
func (s *Server) handleDeleteBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		BackupName string `json:"backup_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Prevent path traversal
	backupName := filepath.Base(req.BackupName)
	backupDir := filepath.Join(BackupStorageDir, backupName)

	if !fileExists(backupDir) {
		s.errorResponse(w, http.StatusNotFound, "backup not found: %s", backupName)
		return
	}

	// Delete backup directory
	if err := os.RemoveAll(backupDir); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to delete backup: %v", err)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":      "success",
		"message":     fmt.Sprintf("Backup %s deleted", backupName),
		"backup_name": backupName,
	})
}

// Helper functions
func parseDiskList(output string) []string {
	lines := strings.Split(output, "\n")
	disks := []string{}

	for i, line := range lines {
		if i < 2 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 4 {
			// Fourth field is the source path
			disks = append(disks, fields[3])
		}
	}

	return disks
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// getAvailableDiskSpace returns available disk space in bytes for the given path
func getAvailableDiskSpace(path string) (int64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	// Available blocks * block size
	available := stat.Bavail * uint64(stat.Bsize)
	return int64(available), nil
}

// checkDiskSpace verifies sufficient disk space is available
func checkDiskSpace(path string, requiredBytes int64) error {
	available, err := getAvailableDiskSpace(path)
	if err != nil {
		return fmt.Errorf("failed to check disk space: %w", err)
	}

	// Require at least 10% more than needed for safety
	requiredWithBuffer := int64(float64(requiredBytes) * 1.1)

	if available < requiredWithBuffer {
		return fmt.Errorf("insufficient disk space: need %d GB, have %d GB available",
			requiredWithBuffer/(1024*1024*1024),
			available/(1024*1024*1024))
	}
	return nil
}

// calculateDirectorySize calculates the total size of a directory
func calculateDirectorySize(dirPath string) (int64, error) {
	var totalSize int64
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	return totalSize, err
}

// isValidVMName validates that a VM name contains only safe characters
func isValidVMName(name string) bool {
	if len(name) == 0 || len(name) > 255 {
		return false
	}
	// Only allow alphanumeric, hyphens, underscores, and dots
	match, _ := regexp.MatchString(`^[a-zA-Z0-9._-]+$`, name)
	return match
}

// replaceVMNameInXML safely replaces the VM name in libvirt XML using proper XML parsing
func replaceVMNameInXML(xmlStr, newName string) (string, error) {
	// Validate the new name first
	if !isValidVMName(newName) {
		return "", fmt.Errorf("invalid VM name: must contain only alphanumeric characters, hyphens, underscores, and dots")
	}

	// Parse XML into generic structure
	type Domain struct {
		XMLName xml.Name `xml:"domain"`
		Name    string   `xml:"name"`
		Content []byte   `xml:",innerxml"`
	}

	var domain Domain
	if err := xml.Unmarshal([]byte(xmlStr), &domain); err != nil {
		return "", fmt.Errorf("failed to parse XML: %w", err)
	}

	// Replace the name
	domain.Name = newName

	// Marshal back to XML with proper formatting
	output, err := xml.MarshalIndent(domain, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to generate XML: %w", err)
	}

	// Add XML declaration
	result := xml.Header + string(output)
	return result, nil
}
