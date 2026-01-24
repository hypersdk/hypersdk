// SPDX-License-Identifier: LGPL-3.0-or-later

package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.BackupDir != "./backups" {
		t.Errorf("expected backup dir ./backups, got %s", config.BackupDir)
	}

	if !config.EnableCompression {
		t.Error("expected compression to be enabled")
	}

	if config.MaxBackups != 10 {
		t.Errorf("expected max backups 10, got %d", config.MaxBackups)
	}

	if config.RetentionDays != 30 {
		t.Errorf("expected retention days 30, got %d", config.RetentionDays)
	}
}

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.BackupDir = tmpDir

	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	if manager == nil {
		t.Fatal("expected manager to be created")
	}

	// Verify backup directory was created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("backup directory was not created")
	}
}

func TestNewManagerNilConfig(t *testing.T) {
	// Override default config backup dir for testing
	defaultBackupDir := "./backups"
	defer os.RemoveAll(defaultBackupDir)

	manager, err := NewManager(nil, nil)
	if err != nil {
		t.Fatalf("failed to create manager with nil config: %v", err)
	}

	if manager == nil {
		t.Fatal("expected manager to be created")
	}

	if manager.config == nil {
		t.Error("expected default config to be set")
	}
}

func TestCreateBackup(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	sourceDir := filepath.Join(tmpDir, "source")

	// Create source directory with test files
	os.MkdirAll(sourceDir, 0755)
	os.WriteFile(filepath.Join(sourceDir, "file1.txt"), []byte("test content 1"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "file2.txt"), []byte("test content 2"), 0644)

	config := DefaultConfig()
	config.BackupDir = backupDir

	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	metadata, err := manager.CreateBackup(ctx, sourceDir, BackupTypeFull)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	if metadata.Status != BackupStatusCompleted {
		t.Errorf("expected status completed, got %s", metadata.Status)
	}

	if metadata.Type != BackupTypeFull {
		t.Errorf("expected type full, got %s", metadata.Type)
	}

	if metadata.Size == 0 {
		t.Error("expected size to be greater than 0")
	}

	if metadata.Checksum == "" {
		t.Error("expected checksum to be set")
	}

	if len(metadata.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(metadata.Files))
	}

	// Verify backup file exists
	if _, err := os.Stat(metadata.BackupPath); os.IsNotExist(err) {
		t.Error("backup file was not created")
	}
}

func TestRestoreBackup(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	sourceDir := filepath.Join(tmpDir, "source")
	restoreDir := filepath.Join(tmpDir, "restore")

	// Create source directory with test files
	os.MkdirAll(sourceDir, 0755)
	testContent := "test content for restore"
	os.WriteFile(filepath.Join(sourceDir, "file1.txt"), []byte(testContent), 0644)

	config := DefaultConfig()
	config.BackupDir = backupDir

	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Create backup
	metadata, err := manager.CreateBackup(ctx, sourceDir, BackupTypeFull)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// Restore backup
	err = manager.RestoreBackup(ctx, metadata.ID, restoreDir)
	if err != nil {
		t.Fatalf("failed to restore backup: %v", err)
	}

	// Verify restored file
	restoredFile := filepath.Join(restoreDir, "file1.txt")
	content, err := os.ReadFile(restoredFile)
	if err != nil {
		t.Fatalf("failed to read restored file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("expected content %s, got %s", testContent, string(content))
	}
}

func TestListBackups(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	sourceDir := filepath.Join(tmpDir, "source")

	os.MkdirAll(sourceDir, 0755)
	os.WriteFile(filepath.Join(sourceDir, "file.txt"), []byte("test"), 0644)

	config := DefaultConfig()
	config.BackupDir = backupDir

	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Create one backup
	metadata, err := manager.CreateBackup(ctx, sourceDir, BackupTypeFull)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	backups := manager.ListBackups()
	if len(backups) != 1 {
		t.Errorf("expected 1 backup, got %d", len(backups))
	}

	// Verify the backup is in the list
	found := false
	for _, b := range backups {
		if b.ID == metadata.ID {
			found = true
			break
		}
	}

	if !found {
		t.Error("created backup not found in list")
	}
}

func TestGetBackup(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	sourceDir := filepath.Join(tmpDir, "source")

	os.MkdirAll(sourceDir, 0755)
	os.WriteFile(filepath.Join(sourceDir, "file.txt"), []byte("test"), 0644)

	config := DefaultConfig()
	config.BackupDir = backupDir

	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	metadata, err := manager.CreateBackup(ctx, sourceDir, BackupTypeFull)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// Get backup
	retrieved, err := manager.GetBackup(metadata.ID)
	if err != nil {
		t.Fatalf("failed to get backup: %v", err)
	}

	if retrieved.ID != metadata.ID {
		t.Errorf("expected ID %s, got %s", metadata.ID, retrieved.ID)
	}
}

func TestGetBackupNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.BackupDir = tmpDir

	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	_, err = manager.GetBackup("nonexistent")
	if err != ErrBackupNotFound {
		t.Errorf("expected ErrBackupNotFound, got %v", err)
	}
}

func TestDeleteBackup(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	sourceDir := filepath.Join(tmpDir, "source")

	os.MkdirAll(sourceDir, 0755)
	os.WriteFile(filepath.Join(sourceDir, "file.txt"), []byte("test"), 0644)

	config := DefaultConfig()
	config.BackupDir = backupDir

	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	metadata, err := manager.CreateBackup(ctx, sourceDir, BackupTypeFull)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// Delete backup
	err = manager.DeleteBackup(metadata.ID)
	if err != nil {
		t.Fatalf("failed to delete backup: %v", err)
	}

	// Verify deleted
	_, err = manager.GetBackup(metadata.ID)
	if err != ErrBackupNotFound {
		t.Error("expected backup to be deleted")
	}

	// Verify files deleted
	if _, err := os.Stat(metadata.BackupPath); !os.IsNotExist(err) {
		t.Error("backup file was not deleted")
	}
}

func TestVerifyBackup(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	sourceDir := filepath.Join(tmpDir, "source")

	os.MkdirAll(sourceDir, 0755)
	os.WriteFile(filepath.Join(sourceDir, "file.txt"), []byte("test"), 0644)

	config := DefaultConfig()
	config.BackupDir = backupDir

	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	metadata, err := manager.CreateBackup(ctx, sourceDir, BackupTypeFull)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// Verify backup
	err = manager.verifyBackup(metadata)
	if err != nil {
		t.Errorf("backup verification failed: %v", err)
	}
}

func TestVerifyCorruptedBackup(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	sourceDir := filepath.Join(tmpDir, "source")

	os.MkdirAll(sourceDir, 0755)
	os.WriteFile(filepath.Join(sourceDir, "file.txt"), []byte("test"), 0644)

	config := DefaultConfig()
	config.BackupDir = backupDir

	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	metadata, err := manager.CreateBackup(ctx, sourceDir, BackupTypeFull)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// Corrupt the backup file
	f, _ := os.OpenFile(metadata.BackupPath, os.O_APPEND|os.O_WRONLY, 0644)
	f.WriteString("corrupt data")
	f.Close()

	// Verify should fail
	err = manager.verifyBackup(metadata)
	if err != ErrInvalidBackup {
		t.Errorf("expected ErrInvalidBackup, got %v", err)
	}
}

func TestRestoreNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	config := DefaultConfig()
	config.BackupDir = tmpDir

	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	err = manager.RestoreBackup(ctx, "nonexistent", tmpDir)
	if err != ErrBackupNotFound {
		t.Errorf("expected ErrBackupNotFound, got %v", err)
	}
}

func TestConcurrentRestore(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	sourceDir := filepath.Join(tmpDir, "source")
	restoreDir1 := filepath.Join(tmpDir, "restore1")
	restoreDir2 := filepath.Join(tmpDir, "restore2")

	// Create large source to make restore take longer
	os.MkdirAll(sourceDir, 0755)
	for i := 0; i < 500; i++ {
		data := make([]byte, 50000)
		os.WriteFile(filepath.Join(sourceDir, fmt.Sprintf("file%d.txt", i)), data, 0644)
	}

	config := DefaultConfig()
	config.BackupDir = backupDir

	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	metadata, err := manager.CreateBackup(ctx, sourceDir, BackupTypeFull)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// Start first restore in background
	restoreStarted := make(chan bool)
	done := make(chan error)
	go func() {
		manager.restoreMu.Lock()
		manager.restoring = true
		manager.restoreMu.Unlock()
		restoreStarted <- true

		// Simulate long restore
		time.Sleep(500 * time.Millisecond)

		manager.restoreMu.Lock()
		manager.restoring = false
		manager.restoreMu.Unlock()

		err := manager.RestoreBackup(ctx, metadata.ID, restoreDir1)
		done <- err
	}()

	// Wait for restore to start
	<-restoreStarted

	// Try second restore - should fail
	err = manager.RestoreBackup(ctx, metadata.ID, restoreDir2)
	if err != ErrRestoreInProgress {
		t.Errorf("expected ErrRestoreInProgress, got %v", err)
	}

	// Wait for first restore to complete
	restoreErr := <-done
	if restoreErr != nil {
		t.Logf("first restore error (expected in test setup): %v", restoreErr)
	}
}

func TestRetentionPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	sourceDir := filepath.Join(tmpDir, "source")

	os.MkdirAll(sourceDir, 0755)
	os.WriteFile(filepath.Join(sourceDir, "file.txt"), []byte("test"), 0644)

	config := DefaultConfig()
	config.BackupDir = backupDir
	config.MaxBackups = 3

	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	// Create 5 backups
	for i := 0; i < 5; i++ {
		_, err := manager.CreateBackup(ctx, sourceDir, BackupTypeFull)
		if err != nil {
			t.Fatalf("failed to create backup %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for retention to apply
	time.Sleep(100 * time.Millisecond)

	// Should only have 3 backups
	backups := manager.ListBackups()
	if len(backups) > 3 {
		t.Errorf("expected at most 3 backups after retention, got %d", len(backups))
	}
}

func TestLoadBackups(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	sourceDir := filepath.Join(tmpDir, "source")

	os.MkdirAll(sourceDir, 0755)
	os.WriteFile(filepath.Join(sourceDir, "file.txt"), []byte("test"), 0644)

	config := DefaultConfig()
	config.BackupDir = backupDir

	// Create first manager and backup
	manager1, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	metadata, err := manager1.CreateBackup(ctx, sourceDir, BackupTypeFull)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	// Create new manager (should load existing backups)
	manager2, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("failed to create second manager: %v", err)
	}

	// Verify backup was loaded
	loaded, err := manager2.GetBackup(metadata.ID)
	if err != nil {
		t.Fatalf("failed to get loaded backup: %v", err)
	}

	if loaded.ID != metadata.ID {
		t.Errorf("expected ID %s, got %s", metadata.ID, loaded.ID)
	}
}

func TestBackupWithSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()
	backupDir := filepath.Join(tmpDir, "backups")
	sourceDir := filepath.Join(tmpDir, "source")

	// Create source with subdirectories
	os.MkdirAll(filepath.Join(sourceDir, "subdir1"), 0755)
	os.MkdirAll(filepath.Join(sourceDir, "subdir2"), 0755)
	os.WriteFile(filepath.Join(sourceDir, "file1.txt"), []byte("test1"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "subdir1", "file2.txt"), []byte("test2"), 0644)
	os.WriteFile(filepath.Join(sourceDir, "subdir2", "file3.txt"), []byte("test3"), 0644)

	config := DefaultConfig()
	config.BackupDir = backupDir

	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx := context.Background()

	metadata, err := manager.CreateBackup(ctx, sourceDir, BackupTypeFull)
	if err != nil {
		t.Fatalf("failed to create backup: %v", err)
	}

	if len(metadata.Files) != 3 {
		t.Errorf("expected 3 files, got %d", len(metadata.Files))
	}

	// Restore and verify structure
	restoreDir := filepath.Join(tmpDir, "restore")
	err = manager.RestoreBackup(ctx, metadata.ID, restoreDir)
	if err != nil {
		t.Fatalf("failed to restore backup: %v", err)
	}

	// Verify all files exist
	files := []string{
		"file1.txt",
		"subdir1/file2.txt",
		"subdir2/file3.txt",
	}

	for _, file := range files {
		path := filepath.Join(restoreDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("file %s was not restored", file)
		}
	}
}
