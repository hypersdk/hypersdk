// SPDX-License-Identifier: LGPL-3.0-or-later

// Package backup provides backup and disaster recovery functionality
package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	// ErrBackupNotFound is returned when a backup does not exist
	ErrBackupNotFound = errors.New("backup not found")

	// ErrInvalidBackup is returned when a backup is corrupted
	ErrInvalidBackup = errors.New("invalid backup")

	// ErrRestoreInProgress is returned when a restore is already running
	ErrRestoreInProgress = errors.New("restore in progress")
)

// BackupType represents the type of backup
type BackupType string

const (
	// BackupTypeFull represents a full backup
	BackupTypeFull BackupType = "full"

	// BackupTypeIncremental represents an incremental backup
	BackupTypeIncremental BackupType = "incremental"
)

// BackupStatus represents the status of a backup
type BackupStatus string

const (
	// BackupStatusRunning indicates backup is in progress
	BackupStatusRunning BackupStatus = "running"

	// BackupStatusCompleted indicates backup completed successfully
	BackupStatusCompleted BackupStatus = "completed"

	// BackupStatusFailed indicates backup failed
	BackupStatusFailed BackupStatus = "failed"
)

// BackupMetadata contains metadata about a backup
type BackupMetadata struct {
	ID         string       `json:"id"`
	Type       BackupType   `json:"type"`
	Status     BackupStatus `json:"status"`
	StartTime  time.Time    `json:"start_time"`
	EndTime    time.Time    `json:"end_time"`
	Size       int64        `json:"size"`
	Checksum   string       `json:"checksum"`
	SourcePath string       `json:"source_path"`
	BackupPath string       `json:"backup_path"`
	Files      []string     `json:"files"`
	BaseBackup string       `json:"base_backup,omitempty"` // For incremental backups
	Encrypted  bool         `json:"encrypted"`
	Compressed bool         `json:"compressed"`
	Version    string       `json:"version"`
	Error      string       `json:"error,omitempty"`
}

// Config holds backup configuration
type Config struct {
	// Storage location
	BackupDir string

	// Backup settings
	EnableCompression bool
	EnableEncryption  bool
	EncryptionKey     []byte
	MaxBackups        int // Maximum number of backups to retain
	RetentionDays     int // Number of days to retain backups

	// Performance settings
	BufferSize   int
	MaxWorkers   int
	ChecksumType string // "sha256", "md5"

	// Scheduling
	EnableAutoBackup bool
	BackupInterval   time.Duration
}

// DefaultConfig returns default backup configuration
func DefaultConfig() *Config {
	return &Config{
		BackupDir:         "./backups",
		EnableCompression: true,
		EnableEncryption:  false,
		MaxBackups:        10,
		RetentionDays:     30,
		BufferSize:        32 * 1024,
		MaxWorkers:        4,
		ChecksumType:      "sha256",
		EnableAutoBackup:  false,
		BackupInterval:    24 * time.Hour,
	}
}

// Manager manages backup and restore operations
type Manager struct {
	config    *Config
	backups   map[string]*BackupMetadata
	mu        sync.RWMutex
	restoring bool
	restoreMu sync.Mutex
}

// NewManager creates a new backup manager
func NewManager(config *Config) (*Manager, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create backup directory
	if err := os.MkdirAll(config.BackupDir, 0755); err != nil {
		return nil, err
	}

	manager := &Manager{
		config:  config,
		backups: make(map[string]*BackupMetadata),
	}

	// Load existing backups
	if err := manager.loadBackups(); err != nil {
		return nil, err
	}

	// Start auto-backup if enabled
	if config.EnableAutoBackup {
		go manager.autoBackupLoop()
	}

	return manager, nil
}

// CreateBackup creates a new backup
func (m *Manager) CreateBackup(ctx context.Context, sourcePath string, backupType BackupType) (*BackupMetadata, error) {
	backupID := generateBackupID()

	metadata := &BackupMetadata{
		ID:         backupID,
		Type:       backupType,
		Status:     BackupStatusRunning,
		StartTime:  time.Now(),
		SourcePath: sourcePath,
		BackupPath: filepath.Join(m.config.BackupDir, backupID+".tar.gz"),
		Compressed: m.config.EnableCompression,
		Encrypted:  m.config.EnableEncryption,
		Version:    "1.0",
	}

	m.mu.Lock()
	m.backups[backupID] = metadata
	m.mu.Unlock()

	// Perform backup
	err := m.performBackup(ctx, metadata)
	if err != nil {
		metadata.Status = BackupStatusFailed
		metadata.Error = err.Error()
		metadata.EndTime = time.Now()
		return metadata, err
	}

	metadata.Status = BackupStatusCompleted
	metadata.EndTime = time.Now()

	// Save metadata
	if err := m.saveMetadata(metadata); err != nil {
		return metadata, err
	}

	// Apply retention policy
	go m.applyRetention()

	return metadata, nil
}

// performBackup performs the actual backup operation
func (m *Manager) performBackup(ctx context.Context, metadata *BackupMetadata) error {
	// Create backup file
	file, err := os.Create(metadata.BackupPath)
	if err != nil {
		return err
	}

	// Create writer chain
	var writer io.Writer = file
	var gzipWriter *gzip.Writer

	// Add gzip compression
	if m.config.EnableCompression {
		gzipWriter = gzip.NewWriter(writer)
		writer = gzipWriter
	}

	// Create tar writer
	tarWriter := tar.NewWriter(writer)

	// Walk source directory
	err = filepath.Walk(metadata.SourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		// Update header with relative path
		relPath, err := filepath.Rel(metadata.SourcePath, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// Write file content for regular files
		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()

			if _, err := io.Copy(tarWriter, f); err != nil {
				return err
			}

			metadata.Files = append(metadata.Files, relPath)
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Close writers in correct order to flush all data
	if err := tarWriter.Close(); err != nil {
		file.Close()
		return err
	}

	if gzipWriter != nil {
		if err := gzipWriter.Close(); err != nil {
			file.Close()
			return err
		}
	}

	if err := file.Close(); err != nil {
		return err
	}

	// Get file info for size
	fileInfo, err := os.Stat(metadata.BackupPath)
	if err != nil {
		return err
	}
	metadata.Size = fileInfo.Size()

	// Calculate checksum
	f, err := os.Open(metadata.BackupPath)
	if err != nil {
		return err
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return err
	}
	metadata.Checksum = hex.EncodeToString(hash.Sum(nil))

	return nil
}

// RestoreBackup restores from a backup
func (m *Manager) RestoreBackup(ctx context.Context, backupID string, targetPath string) error {
	m.restoreMu.Lock()
	if m.restoring {
		m.restoreMu.Unlock()
		return ErrRestoreInProgress
	}
	m.restoring = true
	m.restoreMu.Unlock()

	defer func() {
		m.restoreMu.Lock()
		m.restoring = false
		m.restoreMu.Unlock()
	}()

	// Get backup metadata
	m.mu.RLock()
	metadata, exists := m.backups[backupID]
	m.mu.RUnlock()

	if !exists {
		return ErrBackupNotFound
	}

	// Verify checksum
	if err := m.verifyBackup(metadata); err != nil {
		return err
	}

	// Perform restore
	return m.performRestore(ctx, metadata, targetPath)
}

// performRestore performs the actual restore operation
func (m *Manager) performRestore(ctx context.Context, metadata *BackupMetadata, targetPath string) error {
	// Open backup file
	file, err := os.Open(metadata.BackupPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create reader chain
	var reader io.Reader = file

	// Add gzip decompression
	if metadata.Compressed {
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			return err
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	// Create tar reader
	tarReader := tar.NewReader(reader)

	// Extract files
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Construct target path
		target := filepath.Join(targetPath, header.Name)

		// Handle directories
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
			continue
		}

		// Create parent directory
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		// Create file
		f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
		if err != nil {
			return err
		}

		// Copy content
		if _, err := io.Copy(f, tarReader); err != nil {
			f.Close()
			return err
		}
		f.Close()
	}

	return nil
}

// verifyBackup verifies the integrity of a backup
func (m *Manager) verifyBackup(metadata *BackupMetadata) error {
	file, err := os.Open(metadata.BackupPath)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	checksum := hex.EncodeToString(hash.Sum(nil))
	if checksum != metadata.Checksum {
		return ErrInvalidBackup
	}

	return nil
}

// ListBackups returns all backups
func (m *Manager) ListBackups() []*BackupMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	backups := make([]*BackupMetadata, 0, len(m.backups))
	for _, backup := range m.backups {
		backups = append(backups, backup)
	}

	return backups
}

// GetBackup returns a specific backup
func (m *Manager) GetBackup(backupID string) (*BackupMetadata, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	backup, exists := m.backups[backupID]
	if !exists {
		return nil, ErrBackupNotFound
	}

	return backup, nil
}

// DeleteBackup deletes a backup
func (m *Manager) DeleteBackup(backupID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	metadata, exists := m.backups[backupID]
	if !exists {
		return ErrBackupNotFound
	}

	// Delete backup file
	if err := os.Remove(metadata.BackupPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Delete metadata file
	metadataPath := filepath.Join(m.config.BackupDir, backupID+".json")
	if err := os.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	delete(m.backups, backupID)

	return nil
}

// saveMetadata saves backup metadata to disk
func (m *Manager) saveMetadata(metadata *BackupMetadata) error {
	metadataPath := filepath.Join(m.config.BackupDir, metadata.ID+".json")

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(metadataPath, data, 0644)
}

// loadBackups loads all backup metadata from disk
func (m *Manager) loadBackups() error {
	pattern := filepath.Join(m.config.BackupDir, "*.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	for _, path := range matches {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var metadata BackupMetadata
		if err := json.Unmarshal(data, &metadata); err != nil {
			continue
		}

		m.backups[metadata.ID] = &metadata
	}

	return nil
}

// applyRetention applies backup retention policy
func (m *Manager) applyRetention() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Sort backups by time
	type backupTime struct {
		id   string
		time time.Time
	}

	var sorted []backupTime
	for id, backup := range m.backups {
		sorted = append(sorted, backupTime{id, backup.StartTime})
	}

	// Sort by time (oldest first)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].time.After(sorted[j].time) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Delete old backups
	now := time.Now()
	for i, bt := range sorted {
		backup := m.backups[bt.id]

		// Delete if exceeds count limit
		if i < len(sorted)-m.config.MaxBackups {
			m.DeleteBackup(bt.id)
			continue
		}

		// Delete if exceeds retention days
		if m.config.RetentionDays > 0 {
			age := now.Sub(backup.StartTime)
			if age > time.Duration(m.config.RetentionDays)*24*time.Hour {
				m.DeleteBackup(bt.id)
			}
		}
	}
}

// autoBackupLoop runs periodic backups
func (m *Manager) autoBackupLoop() {
	ticker := time.NewTicker(m.config.BackupInterval)
	defer ticker.Stop()

	for range ticker.C {
		// Perform backup (source path should be configured)
		// This is a placeholder - actual implementation would need source path
		// m.CreateBackup(context.Background(), sourcePath, BackupTypeFull)
	}
}

// generateBackupID generates a unique backup ID
func generateBackupID() string {
	return fmt.Sprintf("backup-%d", time.Now().Unix())
}
