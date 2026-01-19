// SPDX-License-Identifier: LGPL-3.0-or-later

package azure

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"

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

	// AccessURI is a struct type in newer Azure SDK versions
	// Check if AccessSAS field is available
	if accessResp.AccessURI.AccessSAS == nil {
		return nil, fmt.Errorf("failed to get disk SAS URL - AccessSAS is nil")
	}
	sasURL := *accessResp.AccessURI.AccessSAS
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
// Note: This is a simplified implementation. The Azure Blob Storage SDK has changed
// significantly in recent versions. This function needs to be updated to use the
// latest SDK patterns with BlockBlobClient.
func (c *Client) copyVHDToBlob(ctx context.Context, sasURL, containerURL, blobName string, reporter progress.ProgressReporter) (string, error) {
	// TODO: Update to use current Azure Blob Storage SDK
	// The SDK API has changed - StartCopyFromURL and GetProperties are on BlockBlobClient
	// not on the base Client type.

	c.logger.Warn("blob copy not fully implemented - needs Azure SDK update")

	// For now, just return the constructed blob URL
	// In a real implementation, this would use:
	// - BlockBlobClient to start the copy
	// - Poll the copy status using GetProperties
	blobURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(containerURL, "/"), blobName)

	return blobURL, fmt.Errorf("blob copy requires Azure SDK update - download VHD directly from SAS URL instead")
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
