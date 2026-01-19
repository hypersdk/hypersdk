// SPDX-License-Identifier: LGPL-3.0-or-later

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// ISO storage directory (configurable via environment variable)
var ISOStorageDir = getISOStorageDir()

func getISOStorageDir() string {
	if dir := os.Getenv("HYPERSDK_ISO_DIR"); dir != "" {
		return dir
	}
	return "/var/lib/libvirt/images/isos"
}

// ISOFile represents an ISO file
type ISOFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// handleListISOs lists all available ISO files
func (s *Server) handleListISOs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create ISO directory if it doesn't exist
	if err := os.MkdirAll(ISOStorageDir, 0755); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to access ISO directory: %v", err)
		return
	}

	// Read directory
	entries, err := os.ReadDir(ISOStorageDir)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to read ISO directory: %v", err)
		return
	}

	isos := []ISOFile{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only include .iso files
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".iso") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		isos = append(isos, ISOFile{
			Name: entry.Name(),
			Path: filepath.Join(ISOStorageDir, entry.Name()),
			Size: info.Size(),
		})
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"isos":        isos,
		"total":       len(isos),
		"storage_dir": ISOStorageDir,
	})
}

// handleUploadISO handles ISO file upload
func (s *Server) handleUploadISO(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 10GB)
	if err := r.ParseMultipartForm(10 << 30); err != nil {
		http.Error(w, "failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("iso")
	if err != nil {
		http.Error(w, "no ISO file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate filename
	filename := filepath.Base(header.Filename)
	if !strings.HasSuffix(strings.ToLower(filename), ".iso") {
		http.Error(w, "file must have .iso extension", http.StatusBadRequest)
		return
	}

	// Create ISO directory if it doesn't exist
	if err := os.MkdirAll(ISOStorageDir, 0755); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to create ISO directory: %v", err)
		return
	}

	destPath := filepath.Join(ISOStorageDir, filename)

	// Check if file already exists
	if _, err := os.Stat(destPath); err == nil {
		s.errorResponse(w, http.StatusConflict, "ISO file already exists: %s", filename)
		return
	}

	// Check available disk space (max 10GB upload)
	var stat syscall.Statfs_t
	if err := syscall.Statfs(ISOStorageDir, &stat); err == nil {
		available := int64(stat.Bavail * uint64(stat.Bsize))
		required := int64(10 << 30) // 10GB
		if available < required {
			s.errorResponse(w, http.StatusInsufficientStorage,
				"insufficient disk space: need %d GB, have %d GB available",
				required/(1024*1024*1024), available/(1024*1024*1024))
			return
		}
	}

	// Create destination file
	dest, err := os.Create(destPath)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to create file: %v", err)
		return
	}
	defer dest.Close()

	// Copy uploaded file
	bytesWritten, err := io.Copy(dest, file)
	if err != nil {
		os.Remove(destPath) // Cleanup on error
		s.errorResponse(w, http.StatusInternalServerError, "failed to save file: %v", err)
		return
	}

	s.jsonResponse(w, http.StatusCreated, map[string]interface{}{
		"status":   "success",
		"message":  fmt.Sprintf("ISO %s uploaded successfully", filename),
		"filename": filename,
		"path":     destPath,
		"size":     bytesWritten,
	})
}

// handleDeleteISO deletes an ISO file
func (s *Server) handleDeleteISO(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate filename (prevent path traversal)
	filename := filepath.Base(req.Filename)
	if !strings.HasSuffix(strings.ToLower(filename), ".iso") {
		http.Error(w, "invalid ISO filename", http.StatusBadRequest)
		return
	}

	isoPath := filepath.Join(ISOStorageDir, filename)

	// Check if file exists
	if _, err := os.Stat(isoPath); os.IsNotExist(err) {
		s.errorResponse(w, http.StatusNotFound, "ISO file not found: %s", filename)
		return
	}

	// Delete file
	if err := os.Remove(isoPath); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to delete ISO: %v", err)
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":   "success",
		"message":  fmt.Sprintf("ISO %s deleted successfully", filename),
		"filename": filename,
	})
}

// handleAttachISO attaches an ISO to a VM
func (s *Server) handleAttachISO(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		VMName   string `json:"vm_name"`
		Filename string `json:"filename"`           // ISO filename
		ISOPath  string `json:"iso_path,omitempty"` // Or full path
		Device   string `json:"device,omitempty"`   // Optional: hdc, sda, etc.
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Determine ISO path
	var isoPath string
	if req.ISOPath != "" {
		// Validate that the provided path is within the ISO storage directory
		cleanPath := filepath.Clean(req.ISOPath)
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			s.errorResponse(w, http.StatusBadRequest, "invalid ISO path: %v", err)
			return
		}
		absStorageDir, err := filepath.Abs(ISOStorageDir)
		if err != nil {
			s.errorResponse(w, http.StatusInternalServerError, "failed to resolve storage directory: %v", err)
			return
		}

		// Ensure the path is within the storage directory
		if !strings.HasPrefix(absPath, absStorageDir) {
			s.errorResponse(w, http.StatusForbidden, "ISO path must be within storage directory: %s", ISOStorageDir)
			return
		}
		isoPath = absPath
	} else if req.Filename != "" {
		filename := filepath.Base(req.Filename)
		isoPath = filepath.Join(ISOStorageDir, filename)
	} else {
		http.Error(w, "either filename or iso_path required", http.StatusBadRequest)
		return
	}

	// Check if ISO exists
	if _, err := os.Stat(isoPath); os.IsNotExist(err) {
		s.errorResponse(w, http.StatusNotFound, "ISO file not found: %s", isoPath)
		return
	}

	// Default device if not specified
	if req.Device == "" {
		req.Device = "hdc"
	}

	// Attach ISO using virsh
	cmd := exec.Command("virsh", "attach-disk", req.VMName,
		isoPath,
		req.Device,
		"--type", "cdrom",
		"--mode", "readonly",
		"--config", "--live")
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to attach ISO: %s", string(output))
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":   "success",
		"message":  fmt.Sprintf("ISO attached to %s", req.VMName),
		"vm_name":  req.VMName,
		"iso_path": isoPath,
		"device":   req.Device,
	})
}

// handleDetachISO detaches an ISO from a VM
func (s *Server) handleDetachISO(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		VMName string `json:"vm_name"`
		Device string `json:"device"` // hdc, sda, etc.
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Default device if not specified
	if req.Device == "" {
		req.Device = "hdc"
	}

	// Eject using virsh change-media
	cmd := exec.Command("virsh", "change-media", req.VMName,
		req.Device,
		"--eject",
		"--config", "--live")
	output, err := cmd.CombinedOutput()
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to detach ISO: %s", string(output))
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": fmt.Sprintf("ISO detached from %s", req.VMName),
		"vm_name": req.VMName,
		"device":  req.Device,
	})
}
