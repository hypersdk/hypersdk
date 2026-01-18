// SPDX-License-Identifier: LGPL-3.0-or-later

package vsphere

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"hypersdk/logger"
)

// CreateOVA packages an OVF export into a single OVA file
func CreateOVA(ovfDir string, ovaPath string, log logger.Logger) error {
	log.Info("Creating OVA package", "ovfDir", ovfDir, "ovaPath", ovaPath)

	// Find all files to package
	files, err := findExportFiles(ovfDir)
	if err != nil {
		return fmt.Errorf("failed to find export files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no export files found in %s", ovfDir)
	}

	log.Info("Packaging files into OVA", "fileCount", len(files))

	// Create OVA file (TAR archive)
	ovaFile, err := os.Create(ovaPath)
	if err != nil {
		return fmt.Errorf("failed to create OVA file: %w", err)
	}
	defer ovaFile.Close()

	tw := tar.NewWriter(ovaFile)
	defer tw.Close()

	// OVF must be first file in OVA (per OVF spec)
	var ovfFile string
	var otherFiles []string

	for _, file := range files {
		if strings.HasSuffix(file, ".ovf") {
			ovfFile = file
		} else {
			otherFiles = append(otherFiles, file)
		}
	}

	if ovfFile == "" {
		return fmt.Errorf("no OVF file found in %s", ovfDir)
	}

	// Add OVF first
	if err := addFileToTar(tw, ovfFile, log); err != nil {
		return fmt.Errorf("failed to add OVF to archive: %w", err)
	}

	// Add other files (manifest, disks, etc.)
	for _, file := range otherFiles {
		if err := addFileToTar(tw, file, log); err != nil {
			return fmt.Errorf("failed to add %s to archive: %w", filepath.Base(file), err)
		}
	}

	log.Info("OVA package created successfully", "path", ovaPath)
	return nil
}

// findExportFiles finds all files related to the export
func findExportFiles(dir string) ([]string, error) {
	var files []string

	// Extensions to include in OVA
	validExts := map[string]bool{
		".ovf":  true,
		".vmdk": true,
		".mf":   true, // manifest file
		".cert": true, // certificate (if present)
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if validExts[ext] {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// addFileToTar adds a file to a TAR archive
func addFileToTar(tw *tar.Writer, filePath string, log logger.Logger) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Create tar header
	header := &tar.Header{
		Name:    filepath.Base(filePath),
		Mode:    int64(info.Mode()),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}

	log.Debug("Adding file to OVA", "file", header.Name, "size", header.Size)

	if err := tw.WriteHeader(header); err != nil {
		return fmt.Errorf("failed to write tar header: %w", err)
	}

	// Copy file contents
	_, err = io.Copy(tw, file)
	if err != nil {
		return fmt.Errorf("failed to write file to tar: %w", err)
	}

	return nil
}

// ExtractOVA extracts an OVA file to a directory
func ExtractOVA(ovaPath string, destDir string, log logger.Logger) error {
	log.Info("Extracting OVA", "ovaPath", ovaPath, "destDir", destDir)

	// Create destination directory
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Open OVA file
	ovaFile, err := os.Open(ovaPath)
	if err != nil {
		return fmt.Errorf("failed to open OVA file: %w", err)
	}
	defer ovaFile.Close()

	tr := tar.NewReader(ovaFile)

	// Extract all files
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct destination path
		destPath := filepath.Join(destDir, header.Name)

		log.Debug("Extracting file from OVA", "file", header.Name, "size", header.Size)

		// Create file
		outFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", header.Name, err)
		}

		// Copy contents
		_, err = io.Copy(outFile, tr)
		outFile.Close()
		if err != nil {
			return fmt.Errorf("failed to extract file %s: %w", header.Name, err)
		}

		// Set file mode
		if err := os.Chmod(destPath, os.FileMode(header.Mode)); err != nil {
			log.Warn("Failed to set file mode", "file", header.Name, "error", err)
		}
	}

	log.Info("OVA extracted successfully", "destDir", destDir)
	return nil
}

// ValidateOVA validates an OVA file structure
func ValidateOVA(ovaPath string) error {
	ovaFile, err := os.Open(ovaPath)
	if err != nil {
		return fmt.Errorf("failed to open OVA file: %w", err)
	}
	defer ovaFile.Close()

	tr := tar.NewReader(ovaFile)

	foundOVF := false
	foundVMDK := false
	fileCount := 0

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar: %w", err)
		}

		fileCount++
		name := strings.ToLower(header.Name)

		if strings.HasSuffix(name, ".ovf") {
			if fileCount != 1 {
				return fmt.Errorf("OVF file must be first file in OVA (found at position %d)", fileCount)
			}
			foundOVF = true
		}

		if strings.HasSuffix(name, ".vmdk") {
			foundVMDK = true
		}
	}

	if !foundOVF {
		return fmt.Errorf("OVA does not contain an OVF file")
	}

	if !foundVMDK {
		return fmt.Errorf("OVA does not contain any VMDK files")
	}

	return nil
}
