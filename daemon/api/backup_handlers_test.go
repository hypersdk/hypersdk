// SPDX-License-Identifier: LGPL-3.0-or-later

package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name   string
		path   string
		exists bool
	}{
		{
			name:   "existing file",
			path:   testFile,
			exists: true,
		},
		{
			name:   "non-existent file",
			path:   filepath.Join(tmpDir, "nonexistent.txt"),
			exists: false,
		},
		{
			name:   "existing directory",
			path:   tmpDir,
			exists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fileExists(tt.path)
			if result != tt.exists {
				t.Errorf("fileExists(%s) = %v, want %v", tt.path, result, tt.exists)
			}
		})
	}
}

func TestGetAvailableDiskSpace(t *testing.T) {
	tmpDir := t.TempDir()

	space, err := getAvailableDiskSpace(tmpDir)
	if err != nil {
		t.Fatalf("getAvailableDiskSpace() error = %v", err)
	}

	if space <= 0 {
		t.Errorf("Expected positive disk space, got %d", space)
	}
}

func TestGetAvailableDiskSpace_InvalidPath(t *testing.T) {
	_, err := getAvailableDiskSpace("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("Expected error for non-existent path, got nil")
	}
}

func TestCheckDiskSpace(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		requiredBytes int64
		expectError   bool
	}{
		{
			name:          "sufficient space (1KB)",
			requiredBytes: 1024,
			expectError:   false,
		},
		{
			name:          "insufficient space (1PB)",
			requiredBytes: 1024 * 1024 * 1024 * 1024 * 1024, // 1 PB
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkDiskSpace(tmpDir, tt.requiredBytes)
			if (err != nil) != tt.expectError {
				t.Errorf("checkDiskSpace() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestCalculateDirectorySize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	if err := os.WriteFile(file1, []byte("12345"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	if err := os.WriteFile(file2, []byte("1234567890"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Calculate size
	size, err := calculateDirectorySize(tmpDir)
	if err != nil {
		t.Fatalf("calculateDirectorySize() error = %v", err)
	}

	expectedSize := int64(15) // 5 + 10 bytes
	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}
}

func TestCalculateDirectorySize_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	size, err := calculateDirectorySize(tmpDir)
	if err != nil {
		t.Fatalf("calculateDirectorySize() error = %v", err)
	}

	if size != 0 {
		t.Errorf("Expected size 0 for empty directory, got %d", size)
	}
}

func TestCalculateDirectorySize_WithSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// Create subdirectory with files
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create files in both directories
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(subDir, "file2.txt")

	if err := os.WriteFile(file1, []byte("12345"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	if err := os.WriteFile(file2, []byte("1234567890"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	// Calculate size - should include both files
	size, err := calculateDirectorySize(tmpDir)
	if err != nil {
		t.Fatalf("calculateDirectorySize() error = %v", err)
	}

	expectedSize := int64(15) // 5 + 10 bytes
	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}
}

func TestIsValidVMName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{
			name:  "valid simple name",
			input: "my-vm",
			valid: true,
		},
		{
			name:  "valid with underscores",
			input: "my_vm_123",
			valid: true,
		},
		{
			name:  "valid with dots",
			input: "vm.test.local",
			valid: true,
		},
		{
			name:  "valid mixed",
			input: "VM-123_test.v2",
			valid: true,
		},
		{
			name:  "empty name",
			input: "",
			valid: false,
		},
		{
			name:  "name with spaces",
			input: "my vm",
			valid: false,
		},
		{
			name:  "name with special chars",
			input: "vm@test",
			valid: false,
		},
		{
			name:  "name with slash",
			input: "vm/test",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidVMName(tt.input)
			if result != tt.valid {
				t.Errorf("isValidVMName(%q) = %v, want %v", tt.input, result, tt.valid)
			}
		})
	}
}
