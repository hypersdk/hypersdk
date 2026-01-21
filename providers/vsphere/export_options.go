// SPDX-License-Identifier: LGPL-3.0-or-later

package vsphere

import (
	"fmt"
	"time"
)

type ExportOptions struct {
	Format                 string // "ovf" or "ova"
	OutputPath             string
	RemoveCDROM            bool
	ShutdownTimeout        time.Duration
	ParallelDownloads      int
	ValidateChecksum       bool // renamed from Validate to avoid conflict with Validate() method
	ShowIndividualProgress bool
	ShowOverallProgress    bool
	CleanupOVF             bool // Remove OVF files after OVA creation
	Compress               bool // Enable gzip compression for OVA
	CompressionLevel       int  // Gzip compression level (0-9, default 6)

	// Artifact Manifest v1.0 options
	GenerateManifest        bool   // Generate Artifact Manifest v1.0
	VerifyManifest          bool   // Verify manifest after generation
	ManifestComputeChecksum bool   // Compute SHA-256 checksums for all disks
	ManifestTargetFormat    string // Target format for hyper2kvm conversion (e.g., "qcow2")

	// Automatic conversion options (Phase 2)
	AutoConvert            bool          // Automatically run hyper2kvm after export
	Hyper2KVMBinary        string        // Path to hyper2kvm binary (auto-detect if empty)
	ConversionTimeout      time.Duration // Timeout for conversion process
	StreamConversionOutput bool          // Stream hyper2kvm output to console

	// Progress callback for TUI/API integration
	ProgressCallback func(current, total int64, fileName string, fileIndex, totalFiles int)
}

func DefaultExportOptions() ExportOptions {
	return ExportOptions{
		Format:                 "ovf",
		RemoveCDROM:            true,
		ShutdownTimeout:        5 * time.Minute,
		ParallelDownloads:      3,
		ValidateChecksum:       true,
		ShowIndividualProgress: false,
		ShowOverallProgress:    true,
	}
}

// Validate checks if the export options are valid
func (opts *ExportOptions) Validate() error {
	if opts.ParallelDownloads <= 0 {
		return fmt.Errorf("parallel downloads must be > 0, got %d", opts.ParallelDownloads)
	}

	if opts.ParallelDownloads > 16 {
		return fmt.Errorf("parallel downloads must be <= 16, got %d", opts.ParallelDownloads)
	}

	if opts.Format != "ovf" && opts.Format != "ova" {
		return fmt.Errorf("format must be 'ovf' or 'ova', got '%s'", opts.Format)
	}

	if opts.OutputPath == "" {
		return fmt.Errorf("output path cannot be empty")
	}

	return nil
}
