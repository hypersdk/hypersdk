// SPDX-License-Identifier: LGPL-3.0-or-later

package vsphere

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vmware/govmomi/nfc"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/ovf"
	"github.com/vmware/govmomi/vim25/types"

	"hypersdk/progress"
)

const (
	// maxFilenameLength is the maximum allowed filename length on most filesystems
	maxFilenameLength = 255
)

// sanitizeVMName sanitizes a VM name for safe use in file paths
func sanitizeVMName(name string) string {
	// Replace directory separators and path traversal attempts
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, "..", "-")

	// Remove null bytes and other invalid characters
	name = strings.ReplaceAll(name, "\x00", "")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "*", "-")
	name = strings.ReplaceAll(name, "?", "-")
	name = strings.ReplaceAll(name, "\"", "-")
	name = strings.ReplaceAll(name, "<", "-")
	name = strings.ReplaceAll(name, ">", "-")
	name = strings.ReplaceAll(name, "|", "-")

	// Trim dangerous prefixes/suffixes
	name = strings.Trim(name, ".-")

	// Default if empty
	if name == "" {
		name = "unnamed-vm"
	}

	// Limit length
	if len(name) > maxFilenameLength {
		name = name[:maxFilenameLength]
	}

	return name
}

// progressWriter implements io.Writer to update progress bar
type progressWriter struct {
	progressBar progress.ProgressReporter
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.progressBar.Add(int64(n))
	return n, nil
}

// ExportOVF exports a VM as OVF format with progress tracking
func (c *VSphereClient) ExportOVF(ctx context.Context, vmPath string, opts ExportOptions) (*ExportResult, error) {
	startTime := time.Now()
	c.logger.Info("starting OVF export", "vm", vmPath, "output", opts.OutputPath)

	// Validate and prepare output directory
	outputDir, err := c.prepareOutputDirectory(opts.OutputPath)
	if err != nil {
		return nil, fmt.Errorf("prepare output directory: %w", err)
	}

	// Get VM
	vm, err := c.finder.VirtualMachine(ctx, vmPath)
	if err != nil {
		return nil, fmt.Errorf("find VM: %w", err)
	}

	// Remove CD/DVD devices if requested
	if opts.RemoveCDROM {
		if err := c.RemoveCDROMDevices(ctx, vmPath); err != nil {
			c.logger.Warn("failed to remove CD/DVD devices", "error", err)
		}
	}

	// Create OVF descriptor
	ovfPath, err := c.createOVFDescriptor(ctx, vm, outputDir)
	if err != nil {
		return nil, fmt.Errorf("create OVF descriptor: %w", err)
	}

	// Start export lease with proper cleanup
	lease, err := vm.Export(ctx)
	if err != nil {
		return nil, fmt.Errorf("start export lease: %w", err)
	}

	// Defer lease cleanup
	defer func() {
		if err != nil {
			if abortErr := lease.Abort(ctx, nil); abortErr != nil {
				c.logger.Warn("failed to abort lease", "error", abortErr)
			}
		}
	}()

	// Wait for lease to be ready with timeout
	leaseCtx, leaseCancel := context.WithTimeout(ctx, leaseWaitTimeout)
	defer leaseCancel()

	info, err := lease.Wait(leaseCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("wait for lease ready: %w", err)
	}

	// Calculate total size
	totalSize := int64(0)
	for _, item := range info.Items {
		totalSize += item.Size
	}

	c.logger.Info("starting download", "files", len(info.Items), "totalSize", totalSize)

	// Create multi-progress manager for parallel downloads
	multiProgress := progress.NewMultiProgress()
	defer multiProgress.Close()

	// Create overall progress bar
	overallBar := progress.NewOverallProgress(os.Stderr, vm.Name(), len(info.Items))
	overallBar.Start(int64(len(info.Items)), "Files")
	multiProgress.AddBar(overallBar)

	// Download files
	downloadCtx, downloadCancel := context.WithTimeout(ctx, downloadTimeout)
	defer downloadCancel()

	var fileBars []*progress.BarProgress
	if opts.ShowIndividualProgress {
		// Create individual progress bars for each file
		for _, item := range info.Items {
			bar := progress.NewDownloadProgress(os.Stderr, filepath.Base(item.Path), item.Size)
			bar.Start(item.Size, "Downloading")
			fileBars = append(fileBars, bar)
			multiProgress.AddBar(bar)
		}
	}

	downloadedFiles, err := c.downloadFilesParallel(
		downloadCtx,
		info.Items,
		outputDir,
		opts.ParallelDownloads,
		overallBar,
		fileBars,
	)
	if err != nil {
		return nil, fmt.Errorf("download files: %w", err)
	}

	// Finish progress bars
	overallBar.Finish()
	for _, bar := range fileBars {
		bar.Finish()
	}

	// Complete lease
	if err := lease.Complete(ctx); err != nil {
		return nil, fmt.Errorf("complete lease: %w", err)
	}

	duration := time.Since(startTime)
	result := &ExportResult{
		OutputDir: outputDir,
		OVFPath:   ovfPath,
		Format:    opts.Format,
		Files:     downloadedFiles,
		TotalSize: totalSize,
		Duration:  duration,
	}

	// Create OVA if requested
	if opts.Format == "ova" {
		c.logger.Info("packaging OVF export as OVA", "compress", opts.Compress)
		sanitizedName := sanitizeVMName(vm.Name())

		// Determine file extension based on compression
		var ovaPath string
		if opts.Compress {
			ovaPath = filepath.Join(outputDir, sanitizedName+".ova.gz")
		} else {
			ovaPath = filepath.Join(outputDir, sanitizedName+".ova")
		}

		// Set default compression level if not specified
		compressionLevel := opts.CompressionLevel
		if compressionLevel == 0 && opts.Compress {
			compressionLevel = 6 // gzip.DefaultCompression
		}

		if err := CreateOVA(outputDir, ovaPath, opts.Compress, compressionLevel, c.logger); err != nil {
			return nil, fmt.Errorf("create OVA: %w", err)
		}

		result.OVAPath = ovaPath
		result.Format = "ova"

		// Optionally cleanup OVF files
		if opts.CleanupOVF {
			c.logger.Info("cleaning up intermediate OVF files")
			for _, file := range downloadedFiles {
				if err := os.Remove(file); err != nil {
					c.logger.Warn("failed to remove OVF file", "file", file, "error", err)
				}
			}
			// Also remove the OVF descriptor
			if err := os.Remove(ovfPath); err != nil {
				c.logger.Warn("failed to remove OVF descriptor", "error", err)
			}
		}

		// Get file size for reporting
		if fi, err := os.Stat(ovaPath); err == nil {
			c.logger.Info("OVA package created successfully",
				"path", ovaPath,
				"size", fi.Size(),
				"compressed", opts.Compress)
		}
	}

	c.logger.Info("export completed successfully",
		"vm", vmPath,
		"format", result.Format,
		"duration", duration,
		"totalSize", totalSize,
		"files", len(downloadedFiles))

	return result, nil
}

func (c *VSphereClient) prepareOutputDirectory(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("get absolute path: %w", err)
	}

	if err := os.MkdirAll(absPath, defaultDirPerm); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	return absPath, nil
}

func (c *VSphereClient) createOVFDescriptor(ctx context.Context, vm *object.VirtualMachine, outputDir string) (string, error) {
	// Sanitize VM name to prevent path traversal
	sanitizedName := sanitizeVMName(vm.Name())
	ovfPath := filepath.Join(outputDir, sanitizedName+".ovf")

	// Create OVF manager
	manager := ovf.NewManager(c.client.Client)

	// Create descriptor with default parameters
	cdp := types.OvfCreateDescriptorParams{}
	desc, err := manager.CreateDescriptor(ctx, vm, cdp)
	if err != nil {
		return "", fmt.Errorf("create OVF descriptor: %w", err)
	}

	// Write descriptor to file
	file, err := os.Create(ovfPath)
	if err != nil {
		return "", fmt.Errorf("create OVF file: %w", err)
	}
	defer file.Close()

	if _, err := file.WriteString(desc.OvfDescriptor); err != nil {
		return "", fmt.Errorf("write OVF descriptor: %w", err)
	}

	c.logger.Debug("created OVF descriptor", "path", ovfPath)
	return ovfPath, nil
}

func (c *VSphereClient) downloadFilesParallel(
	ctx context.Context,
	items []nfc.FileItem,
	outputDir string,
	concurrency int,
	overallBar progress.ProgressReporter,
	fileBars []*progress.BarProgress,
) ([]string, error) {
	if concurrency < 1 {
		concurrency = 1
	}

	var (
		wg         sync.WaitGroup
		sem        = make(chan struct{}, concurrency)
		errCh      = make(chan error, len(items))
		results    = make([]string, len(items))
		resultsMux sync.Mutex
	)

	// Download files
	for i, item := range items {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, item nfc.FileItem) {
			defer wg.Done()
			defer func() { <-sem }()

			filePath := filepath.Join(outputDir, item.Path)

			// Create directory if needed
			if err := os.MkdirAll(filepath.Dir(filePath), defaultDirPerm); err != nil {
				errCh <- fmt.Errorf("create directory for %s: %w", item.Path, err)
				return
			}

			// Get progress bar for this file (if individual bars are enabled)
			var fileBar progress.ProgressReporter
			if fileBars != nil && idx < len(fileBars) {
				fileBar = fileBars[idx]
			}

			// Download with retry
			bytes, err := c.downloadFileWithRetry(ctx, item.URL.String(), filePath, c.config.RetryAttempts, fileBar)
			if err != nil {
				errCh <- fmt.Errorf("download %s: %w", item.Path, err)
				return
			}

			// Store result
			resultsMux.Lock()
			results[idx] = filePath
			resultsMux.Unlock()

			// Update overall progress
			if overallBar != nil {
				overallBar.Add(1)
			}

			c.logger.Debug("file downloaded", "file", item.Path, "size", bytes)
		}(i, item)
	}

	wg.Wait()
	close(errCh)

	// Collect errors
	var errs []string
	for err := range errCh {
		errs = append(errs, err.Error())
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("download errors: %s", strings.Join(errs, "; "))
	}

	// Filter out empty results
	finalResults := make([]string, 0, len(results))
	for _, result := range results {
		if result != "" {
			finalResults = append(finalResults, result)
		}
	}

	return finalResults, nil
}

func (c *VSphereClient) downloadFileWithRetry(
	ctx context.Context,
	urlStr, filePath string,
	maxRetries int,
	progressBar progress.ProgressReporter,
) (int64, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Debug("retrying download",
				"file", filepath.Base(filePath),
				"attempt", attempt,
				"delay", c.config.RetryDelay*time.Duration(attempt))

			time.Sleep(c.config.RetryDelay * time.Duration(attempt))

			if progressBar != nil {
				progressBar.SetTotal(0) // Reset progress bar for retry
			}
		}

		bytes, err := c.downloadFileResumable(ctx, urlStr, filePath, progressBar)
		if err == nil {
			if progressBar != nil {
				progressBar.Finish()
			}
			return bytes, nil
		}

		lastErr = err
		c.logger.Warn("download attempt failed",
			"file", filepath.Base(filePath),
			"attempt", attempt,
			"error", err)
	}

	return 0, fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}

func (c *VSphereClient) downloadFileResumable(
	ctx context.Context,
	urlStr, filePath string,
	progressBar progress.ProgressReporter,
) (int64, error) {
	// Parse URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return 0, fmt.Errorf("parse URL: %w", err)
	}

	// Check if file already exists for resume
	var startPos int64 = 0
	if fi, err := os.Stat(filePath); err == nil {
		startPos = fi.Size()
		c.logger.Debug("resuming download",
			"file", filepath.Base(filePath),
			"resumeFrom", startPos)

		if progressBar != nil {
			progressBar.Update(startPos)
		}
	}

	// Create HTTP request with range header for resume
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}

	if startPos > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startPos))
	}

	// Open file for writing (append if resuming)
	fileFlag := os.O_CREATE | os.O_WRONLY
	if startPos > 0 {
		fileFlag |= os.O_APPEND
	} else {
		fileFlag |= os.O_TRUNC
	}

	file, err := os.OpenFile(filePath, fileFlag, 0644)
	if err != nil {
		return 0, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	// Use SOAP client for download (handles auth automatically)
	// Must read the response body inside the callback
	var totalWritten int64
	var downloadErr error

	err = c.client.Client.Client.Do(ctx, req, func(res *http.Response) error {
		if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusPartialContent {
			return fmt.Errorf("HTTP error: %s", res.Status)
		}

		// Get total size from Content-Length or Content-Range
		var totalSize int64 = -1
		if cl := res.Header.Get("Content-Length"); cl != "" {
			if size, err := strconv.ParseInt(cl, 10, 64); err == nil {
				totalSize = size + startPos
			}
		} else if cr := res.Header.Get("Content-Range"); cr != "" {
			// Parse "bytes start-end/total" format
			parts := strings.Split(cr, "/")
			if len(parts) == 2 {
				if total, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
					totalSize = total
				}
			}
		}

		// Set progress bar total if not set
		if progressBar != nil && totalSize > 0 {
			progressBar.SetTotal(totalSize)
		}

		// Create a proxy reader that updates progress
		var reader io.Reader = res.Body
		if progressBar != nil {
			reader = io.TeeReader(res.Body, &progressWriter{progressBar: progressBar})
		}

		// Copy data
		written, err := io.Copy(file, reader)
		if err != nil {
			downloadErr = fmt.Errorf("copy data: %w", err)
			return downloadErr
		}

		totalWritten = startPos + written
		return nil
	})

	if err != nil {
		if downloadErr != nil {
			return totalWritten, downloadErr
		}
		return 0, fmt.Errorf("HTTP request: %w", err)
	}

	c.logger.Debug("download completed",
		"file", filepath.Base(filePath),
		"size", totalWritten)

	return totalWritten, nil
}
