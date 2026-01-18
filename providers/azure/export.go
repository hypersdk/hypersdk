// SPDX-License-Identifier: LGPL-3.0-or-later

package azure

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"

	"hypersdk/progress"
)

// ExportDiskToVHD exports an Azure managed disk to VHD format in blob storage
func (c *Client) ExportDiskToVHD(ctx context.Context, diskName, containerURL, outputDir string, reporter progress.ProgressReporter) (*ExportResult, error) {
	c.logger.Info("starting Azure disk export to VHD", "disk", diskName)

	if reporter != nil {
		reporter.Describe("Granting SAS access to managed disk")
	}

	// Grant read access to the disk (1 hour validity)
	grantAccess := &armcompute.GrantAccessData{
		Access:            to.Ptr(armcompute.AccessLevelRead),
		DurationInSeconds: to.Ptr(int32(3600)), // 1 hour
	}

	pollerAccess, err := c.diskClient.BeginGrantAccess(ctx, c.config.ResourceGroup, diskName, *grantAccess, nil)
	if err != nil {
		return nil, fmt.Errorf("grant disk access: %w", err)
	}

	accessResp, err := pollerAccess.PollUntilDone(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("wait for disk access: %w", err)
	}

	sasURL := *accessResp.AccessURI
	c.logger.Info("disk SAS URL obtained", "disk", diskName)

	// Get disk information
	diskResp, err := c.diskClient.Get(ctx, c.config.ResourceGroup, diskName, nil)
	if err != nil {
		return nil, fmt.Errorf("get disk info: %w", err)
	}

	diskSizeGB := int64(0)
	if diskResp.Properties != nil && diskResp.Properties.DiskSizeGB != nil {
		diskSizeGB = int64(*diskResp.Properties.DiskSizeGB)
	}

	// Copy VHD to blob storage if container URL provided
	var blobURL string
	if containerURL != "" {
		if reporter != nil {
			reporter.Describe("Copying VHD to blob storage")
		}

		blobName := fmt.Sprintf("%s-%d.vhd", diskName, time.Now().Unix())
		blobURL, err = c.copyVHDToBlob(ctx, sasURL, containerURL, blobName, reporter)
		if err != nil {
			return nil, fmt.Errorf("copy VHD to blob: %w", err)
		}

		c.logger.Info("VHD copied to blob storage", "blob", blobURL)
	}

	// Download VHD to local path
	if reporter != nil {
		reporter.Describe("Downloading VHD from Azure")
	}

	localPath := filepath.Join(outputDir, fmt.Sprintf("%s.vhd", diskName))
	size, err := c.downloadVHD(ctx, sasURL, localPath, reporter)
	if err != nil {
		return nil, fmt.Errorf("download VHD: %w", err)
	}

	// Revoke access to disk
	pollerRevoke, err := c.diskClient.BeginRevokeAccess(ctx, c.config.ResourceGroup, diskName, nil)
	if err != nil {
		c.logger.Warn("failed to revoke disk access", "error", err)
	} else {
		pollerRevoke.PollUntilDone(ctx, nil)
		c.logger.Info("disk access revoked", "disk", diskName)
	}

	if reporter != nil {
		reporter.Describe("VHD export complete")
		reporter.Update(100)
	}

	return &ExportResult{
		DiskName:   diskName,
		Format:     "vhd",
		LocalPath:  localPath,
		Size:       size,
		BlobURL:    blobURL,
		DiskSizeGB: diskSizeGB,
	}, nil
}

// ExportVMToVHD exports all disks of an Azure VM to VHD format
func (c *Client) ExportVMToVHD(ctx context.Context, vmName, containerURL, outputDir string, reporter progress.ProgressReporter) ([]*ExportResult, error) {
	c.logger.Info("starting Azure VM export to VHD", "vm", vmName)

	// Get VM information to identify disks
	vmInfo, err := c.GetVM(ctx, vmName)
	if err != nil {
		return nil, fmt.Errorf("get VM info: %w", err)
	}

	if len(vmInfo.DiskNames) == 0 {
		return nil, fmt.Errorf("VM has no attached disks")
	}

	c.logger.Info("exporting VM disks", "vm", vmName, "disk_count", len(vmInfo.DiskNames))

	results := make([]*ExportResult, 0, len(vmInfo.DiskNames))

	// Export each disk
	for i, diskName := range vmInfo.DiskNames {
		if reporter != nil {
			reporter.Describe(fmt.Sprintf("Exporting disk %d/%d: %s", i+1, len(vmInfo.DiskNames), diskName))
		}

		result, err := c.ExportDiskToVHD(ctx, diskName, containerURL, outputDir, reporter)
		if err != nil {
			c.logger.Error("failed to export disk", "disk", diskName, "error", err)
			// Continue with other disks even if one fails
			continue
		}

		result.DiskType = "OS"
		if i > 0 {
			result.DiskType = fmt.Sprintf("Data-%d", i)
		}
		results = append(results, result)

		c.logger.Info("disk exported", "disk", diskName, "path", result.LocalPath)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("failed to export any disks")
	}

	c.logger.Info("VM export complete", "vm", vmName, "disks_exported", len(results))
	return results, nil
}

// copyVHDToBlob copies VHD from SAS URL to blob storage
func (c *Client) copyVHDToBlob(ctx context.Context, sasURL, containerURL, blobName string, reporter progress.ProgressReporter) (string, error) {
	// Parse container URL
	containerURLParsed, err := url.Parse(containerURL)
	if err != nil {
		return "", fmt.Errorf("parse container URL: %w", err)
	}

	// Create blob client
	blobURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(containerURL, "/"), blobName)
	blobClient, err := azblob.NewClientWithNoCredential(blobURL, nil)
	if err != nil {
		return "", fmt.Errorf("create blob client: %w", err)
	}

	// Start copy operation
	copyResp, err := blobClient.StartCopyFromURL(ctx, sasURL, nil)
	if err != nil {
		return "", fmt.Errorf("start blob copy: %w", err)
	}

	copyID := *copyResp.CopyID
	c.logger.Info("blob copy started", "copy_id", copyID)

	// Wait for copy to complete
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(24 * time.Hour) // Large disks can take hours

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", fmt.Errorf("blob copy timed out after 24 hours")
		case <-ticker.C:
			// Get blob properties to check copy status
			props, err := blobClient.GetProperties(ctx, nil)
			if err != nil {
				return "", fmt.Errorf("get blob properties: %w", err)
			}

			if props.CopyStatus == nil {
				continue
			}

			status := string(*props.CopyStatus)

			// Update progress if available
			if reporter != nil && props.CopyProgress != nil {
				progress := *props.CopyProgress
				// Parse progress string "bytes_copied/total_bytes"
				parts := strings.Split(progress, "/")
				if len(parts) == 2 {
					var copied, total int64
					fmt.Sscanf(parts[0], "%d", &copied)
					fmt.Sscanf(parts[1], "%d", &total)
					if total > 0 {
						percentage := (copied * 100) / total
						reporter.Update(percentage)
					}
				}
			}

			switch status {
			case "success":
				c.logger.Info("blob copy completed", "copy_id", copyID)
				return blobURL, nil

			case "failed":
				statusDesc := ""
				if props.CopyStatusDescription != nil {
					statusDesc = *props.CopyStatusDescription
				}
				return "", fmt.Errorf("blob copy failed: %s", statusDesc)

			case "aborted":
				return "", fmt.Errorf("blob copy was aborted")

			case "pending":
				// Still copying, continue waiting
				continue
			}
		}
	}
}

// downloadVHD downloads VHD from SAS URL to local file
func (c *Client) downloadVHD(ctx context.Context, sasURL, localPath string, reporter progress.ProgressReporter) (int64, error) {
	// Create output directory
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return 0, fmt.Errorf("create output directory: %w", err)
	}

	// Create local file
	file, err := os.Create(localPath)
	if err != nil {
		return 0, fmt.Errorf("create local file: %w", err)
	}
	defer file.Close()

	// Download VHD
	httpClient := &http.Client{
		Timeout: 24 * time.Hour, // Large downloads can take hours
	}

	req, err := http.NewRequestWithContext(ctx, "GET", sasURL, nil)
	if err != nil {
		return 0, fmt.Errorf("create download request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("download VHD: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	totalSize := resp.ContentLength
	c.logger.Info("downloading VHD", "size_bytes", totalSize)

	// Create progress reader wrapper if reporter provided
	var reader io.Reader = resp.Body
	if reporter != nil && totalSize > 0 {
		reader = &progressReader{
			reader:   resp.Body,
			total:    totalSize,
			reporter: reporter,
		}
	}

	// Copy with progress tracking
	written, err := io.Copy(file, reader)
	if err != nil {
		return 0, fmt.Errorf("write VHD to disk: %w", err)
	}

	return written, nil
}

// ExportSnapshotToVHD exports an Azure snapshot to VHD format
func (c *Client) ExportSnapshotToVHD(ctx context.Context, snapshotName, containerURL, outputDir string, reporter progress.ProgressReporter) (*ExportResult, error) {
	c.logger.Info("starting Azure snapshot export to VHD", "snapshot", snapshotName)

	// The process is similar to disk export - snapshots can also be granted SAS access
	// For simplicity, we can reuse the disk export logic since the API is the same

	// Note: In a real implementation, you would use the Snapshots client
	// For now, we'll return an error indicating this needs snapshot-specific implementation

	return nil, fmt.Errorf("snapshot export not yet implemented - use disk export instead")
}

// progressReader wraps an io.Reader to report download progress
type progressReader struct {
	reader   io.Reader
	total    int64
	current  int64
	reporter progress.ProgressReporter
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.current += int64(n)

	if pr.reporter != nil && pr.total > 0 {
		percentage := (pr.current * 100) / pr.total
		pr.reporter.Update(percentage)
	}

	return n, err
}

// ExportResult contains the result of a VHD export operation
type ExportResult struct {
	DiskName   string
	DiskType   string // "OS", "Data-1", "Data-2", etc.
	Format     string
	LocalPath  string
	Size       int64
	BlobURL    string
	DiskSizeGB int64
}

// CreateExportManifest creates a JSON manifest file for multi-disk exports
func CreateExportManifest(vmName string, results []*ExportResult, outputPath string) error {
	manifestPath := filepath.Join(outputPath, fmt.Sprintf("%s-manifest.json", vmName))

	manifest := fmt.Sprintf(`{
  "vm_name": "%s",
  "export_time": "%s",
  "disk_count": %d,
  "disks": [
`, vmName, time.Now().Format(time.RFC3339), len(results))

	for i, result := range results {
		comma := ","
		if i == len(results)-1 {
			comma = ""
		}
		manifest += fmt.Sprintf(`    {
      "disk_name": "%s",
      "disk_type": "%s",
      "format": "%s",
      "local_path": "%s",
      "size_bytes": %d,
      "disk_size_gb": %d,
      "blob_url": "%s"
    }%s
`, result.DiskName, result.DiskType, result.Format, result.LocalPath,
			result.Size, result.DiskSizeGB, result.BlobURL, comma)
	}

	manifest += `  ]
}`

	return os.WriteFile(manifestPath, []byte(manifest), 0644)
}
