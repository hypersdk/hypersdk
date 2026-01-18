// SPDX-License-Identifier: LGPL-3.0-or-later

package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test Progress Tracking endpoints

func TestHandleGetJobProgress(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/jobs/progress/test-job-id", nil)
	w := httptest.NewRecorder()

	server.handleGetJobProgress(w, req)

	// Should return 404 for non-existent job
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandleGetJobLogs(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/jobs/logs/test-job-id", nil)
	w := httptest.NewRecorder()

	server.handleGetJobLogs(w, req)

	// Should return 404 for non-existent job
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandleGetJobETA(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/jobs/eta/test-job-id", nil)
	w := httptest.NewRecorder()

	server.handleGetJobETA(w, req)

	// Should return 404 for non-existent job
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandleGetJobProgressMethodNotAllowed(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/jobs/progress/test-id", nil)
	w := httptest.NewRecorder()

	server.handleGetJobProgress(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

// Test ISO Management endpoints

func TestHandleListISOs(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/libvirt/isos/list", nil)
	w := httptest.NewRecorder()

	server.handleListISOs(w, req)

	// Should return 200 with empty list or 500 if directory can't be created
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 200 or 500, got %d", w.Code)
	}

	// Only check for isos field if status is 200
	if w.Code == http.StatusOK {
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Should have isos field
		if _, ok := response["isos"]; !ok {
			t.Error("response missing 'isos' field")
		}
	}
}

func TestHandleListISOsMethodNotAllowed(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/libvirt/isos/list", nil)
	w := httptest.NewRecorder()

	server.handleListISOs(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandleDeleteISO(t *testing.T) {
	server := newTestServer()

	reqBody := map[string]string{
		"filename": "nonexistent.iso",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/libvirt/isos/delete", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleDeleteISO(w, req)

	// Should return 404 for non-existent ISO
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandleDeleteISOInvalidRequest(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/libvirt/isos/delete", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	server.handleDeleteISO(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleAttachISO(t *testing.T) {
	server := newTestServer()

	reqBody := map[string]string{
		"vm_name":  "test-vm",
		"filename": "test.iso",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/libvirt/domain/attach-iso", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleAttachISO(w, req)

	// Will fail because ISO doesn't exist or VM doesn't exist
	if w.Code != http.StatusNotFound && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 404 or 500, got %d", w.Code)
	}
}

func TestHandleDetachISO(t *testing.T) {
	server := newTestServer()

	reqBody := map[string]string{
		"vm_name": "test-vm",
		"device":  "hdc",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/libvirt/domain/detach-iso", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleDetachISO(w, req)

	// Will fail because VM doesn't exist
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

// Test Backup & Restore endpoints

func TestHandleListBackups(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/libvirt/backup/list", nil)
	w := httptest.NewRecorder()

	server.handleListBackups(w, req)

	// Should return 200 with empty list or 500 if directory can't be created
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 200 or 500, got %d", w.Code)
	}

	// Only check for backups field if status is 200
	if w.Code == http.StatusOK {
		var response map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Should have backups field
		if _, ok := response["backups"]; !ok {
			t.Error("response missing 'backups' field")
		}
	}
}

func TestHandleListBackupsMethodNotAllowed(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/libvirt/backup/list", nil)
	w := httptest.NewRecorder()

	server.handleListBackups(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandleCreateBackup(t *testing.T) {
	server := newTestServer()

	reqBody := map[string]interface{}{
		"vm_name":  "test-vm",
		"compress": true,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/libvirt/backup/create", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleCreateBackup(w, req)

	// Will fail because VM doesn't exist
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestHandleCreateBackupInvalidRequest(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/libvirt/backup/create", bytes.NewReader([]byte("invalid")))
	w := httptest.NewRecorder()

	server.handleCreateBackup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleRestoreBackup(t *testing.T) {
	server := newTestServer()

	reqBody := map[string]interface{}{
		"backup_name": "nonexistent-backup",
		"start":       false,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/libvirt/backup/restore", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleRestoreBackup(w, req)

	// Should return 404 for non-existent backup
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandleVerifyBackup(t *testing.T) {
	server := newTestServer()

	reqBody := map[string]string{
		"backup_name": "nonexistent-backup",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/libvirt/backup/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleVerifyBackup(w, req)

	// Should return 404 for non-existent backup
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestHandleDeleteBackup(t *testing.T) {
	server := newTestServer()

	reqBody := map[string]string{
		"backup_name": "nonexistent-backup",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/libvirt/backup/delete", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleDeleteBackup(w, req)

	// Should return 404 for non-existent backup
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// Test Validation & Testing endpoints

func TestHandleValidateMigration(t *testing.T) {
	server := newTestServer()

	reqBody := map[string]string{
		"path": "/nonexistent/disk.vmdk",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/migration/validate", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleValidateMigration(w, req)

	// Should return 200 with validation result
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have validation field
	if _, ok := response["validation"]; !ok {
		t.Error("response missing 'validation' field")
	}
}

func TestHandleValidateMigrationMethodNotAllowed(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/migration/validate", nil)
	w := httptest.NewRecorder()

	server.handleValidateMigration(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}

func TestHandleValidateMigrationInvalidRequest(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodPost, "/migration/validate", bytes.NewReader([]byte("invalid")))
	w := httptest.NewRecorder()

	server.handleValidateMigration(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleVerifyMigration(t *testing.T) {
	server := newTestServer()

	reqBody := map[string]interface{}{
		"vm_name":   "nonexistent-vm",
		"boot_test": false,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/migration/verify", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleVerifyMigration(w, req)

	// Should return 200 with verification result
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have verification field
	if _, ok := response["verification"]; !ok {
		t.Error("response missing 'verification' field")
	}
}

func TestHandleCheckCompatibility(t *testing.T) {
	server := newTestServer()

	reqBody := map[string]interface{}{
		"os_type":  "linux",
		"firmware": "bios",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/migration/check-compatibility", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleCheckCompatibility(w, req)

	// Should return 200 with compatibility result
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have compatibility field
	if _, ok := response["compatibility"]; !ok {
		t.Error("response missing 'compatibility' field")
	}
}

func TestHandleTestMigration(t *testing.T) {
	server := newTestServer()

	reqBody := map[string]interface{}{
		"vm_name": "nonexistent-vm",
		"tests":   []string{"disk"},
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/migration/test", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server.handleTestMigration(w, req)

	// Should return 200 with test result
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have test field
	if _, ok := response["test"]; !ok {
		t.Error("response missing 'test' field")
	}
}

func TestHandleTestMigrationMethodNotAllowed(t *testing.T) {
	server := newTestServer()

	req := httptest.NewRequest(http.MethodGet, "/migration/test", nil)
	w := httptest.NewRecorder()

	server.handleTestMigration(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", w.Code)
	}
}
