// SPDX-License-Identifier: LGPL-3.0-or-later

package gcp

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/compute/apiv1/computepb"
	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"hypersdk/progress"
)

// ExportImageToGCS exports a GCP disk image to Google Cloud Storage as VMDK
func (c *Client) ExportImageToGCS(ctx context.Context, imageName, bucket, outputDir string, reporter progress.ProgressReporter) (*ExportResult, error) {
	c.logger.Info("starting GCP image export to GCS", "image", imageName, "bucket", bucket)

	if reporter != nil {
		reporter.Describe("Exporting image to GCS")
	}

	// Construct GCS destination URI
	gcsURI := fmt.Sprintf("gs://%s/%s.vmdk", bucket, imageName)

	// Create export request
	exportReq := &computepb.ExportImageRequest{
		Project: c.config.ProjectID,
		Image:   imageName,
		ImageExportRequest: &computepb.ImageExportRequest{
			DestinationUri: &gcsURI,
			DiskImageFormat: makeStringPtr("vmdk"),
		},
	}

	// Start export operation
	op, err := c.imagesClient.Export(ctx, exportReq)
	if err != nil {
		return nil, fmt.Errorf("start image export: %w", err)
	}

	c.logger.Info("image export started", "image", imageName, "gcs_uri", gcsURI)

	// Wait for export to complete
	if reporter != nil {
		reporter.Describe("Waiting for GCS export to complete")
	}

	err = op.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("wait for image export: %w", err)
	}

	c.logger.Info("image exported to GCS", "gcs_uri", gcsURI)

	// Download from GCS
	if reporter != nil {
		reporter.Describe("Downloading VMDK from GCS")
	}

	localPath := filepath.Join(outputDir, fmt.Sprintf("%s.vmdk", imageName))
	size, err := c.downloadFromGCS(ctx, bucket, fmt.Sprintf("%s.vmdk", imageName), localPath, reporter)
	if err != nil {
		return nil, fmt.Errorf("download from GCS: %w", err)
	}

	c.logger.Info("VMDK downloaded successfully", "path", localPath, "size_bytes", size)

	return &ExportResult{
		ImageName:  imageName,
		Format:     "vmdk",
		LocalPath:  localPath,
		Size:       size,
		GCSBucket:  bucket,
		GCSObject:  fmt.Sprintf("%s.vmdk", imageName),
		GCSURI:     gcsURI,
	}, nil
}

// ExportDiskToGCS exports a persistent disk to GCS as VMDK
func (c *Client) ExportDiskToGCS(ctx context.Context, diskName, bucket, outputDir string, reporter progress.ProgressReporter) (*ExportResult, error) {
	c.logger.Info("starting GCP disk export to GCS", "disk", diskName, "bucket", bucket)

	if reporter != nil {
		reporter.Describe("Creating image from disk")
	}

	// Create image from disk first
	imageName := fmt.Sprintf("%s-image-%d", diskName, time.Now().Unix())

	createImageReq := &computepb.InsertImageRequest{
		Project: c.config.ProjectID,
		ImageResource: &computepb.Image{
			Name: &imageName,
			SourceDisk: makeStringPtr(fmt.Sprintf("projects/%s/zones/%s/disks/%s",
				c.config.ProjectID, c.config.Zone, diskName)),
		},
	}

	op, err := c.imagesClient.Insert(ctx, createImageReq)
	if err != nil {
		return nil, fmt.Errorf("create image from disk: %w", err)
	}

	// Wait for image creation
	err = op.Wait(ctx)
	if err != nil {
		return nil, fmt.Errorf("wait for image creation: %w", err)
	}

	c.logger.Info("image created from disk", "image", imageName, "disk", diskName)

	// Export the image to GCS
	return c.ExportImageToGCS(ctx, imageName, bucket, outputDir, reporter)
}

// ExportInstanceToGCS exports a GCP instance by creating an image and exporting to GCS
func (c *Client) ExportInstanceToGCS(ctx context.Context, instanceName, bucket, outputDir string, reporter progress.ProgressReporter) ([]*ExportResult, error) {
	c.logger.Info("starting GCP instance export to GCS", "instance", instanceName, "bucket", bucket)

	// Get instance to identify disks
	vmInfo, err := c.GetInstance(ctx, instanceName)
	if err != nil {
		return nil, fmt.Errorf("get instance info: %w", err)
	}

	if len(vmInfo.DiskNames) == 0 {
		return nil, fmt.Errorf("instance has no attached disks")
	}

	c.logger.Info("exporting instance disks", "instance", instanceName, "disk_count", len(vmInfo.DiskNames))

	results := make([]*ExportResult, 0, len(vmInfo.DiskNames))

	// Export each disk
	for i, diskName := range vmInfo.DiskNames {
		if reporter != nil {
			reporter.Describe(fmt.Sprintf("Exporting disk %d/%d: %s", i+1, len(vmInfo.DiskNames), diskName))
		}

		result, err := c.ExportDiskToGCS(ctx, diskName, bucket, outputDir, reporter)
		if err != nil {
			c.logger.Error("failed to export disk", "disk", diskName, "error", err)
			// Continue with other disks
			continue
		}

		result.DiskName = diskName
		result.DiskType = "boot"
		if i > 0 {
			result.DiskType = fmt.Sprintf("data-%d", i)
		}
		results = append(results, result)

		c.logger.Info("disk exported", "disk", diskName, "path", result.LocalPath)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("failed to export any disks")
	}

	// Create manifest for multi-disk exports
	if len(results) > 1 {
		if err := CreateExportManifest(instanceName, results, outputDir); err != nil {
			c.logger.Warn("failed to create export manifest", "error", err)
		}
	}

	c.logger.Info("instance export complete", "instance", instanceName, "disks_exported", len(results))
	return results, nil
}

// downloadFromGCS downloads a file from Google Cloud Storage
func (c *Client) downloadFromGCS(ctx context.Context, bucket, object, localPath string, reporter progress.ProgressReporter) (int64, error) {
	// Create GCS client
	var opts []option.ClientOption
	if c.config.CredentialsJSON != "" {
		if _, err := os.Stat(c.config.CredentialsJSON); err == nil {
			opts = append(opts, option.WithCredentialsFile(c.config.CredentialsJSON))
		} else {
			opts = append(opts, option.WithCredentialsJSON([]byte(c.config.CredentialsJSON)))
		}
	}

	gcsClient, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return 0, fmt.Errorf("create GCS client: %w", err)
	}
	defer gcsClient.Close()

	// Get object attributes to determine size
	bucketHandle := gcsClient.Bucket(bucket)
	objectHandle := bucketHandle.Object(object)

	attrs, err := objectHandle.Attrs(ctx)
	if err != nil {
		return 0, fmt.Errorf("get object attributes: %w", err)
	}

	totalSize := attrs.Size
	c.logger.Info("downloading from GCS", "object", object, "size_bytes", totalSize)

	// Create local file
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return 0, fmt.Errorf("create output directory: %w", err)
	}

	file, err := os.Create(localPath)
	if err != nil {
		return 0, fmt.Errorf("create local file: %w", err)
	}
	defer file.Close()

	// Create reader
	reader, err := objectHandle.NewReader(ctx)
	if err != nil {
		return 0, fmt.Errorf("create GCS reader: %w", err)
	}
	defer reader.Close()

	// Create progress reader wrapper if reporter provided
	var ioReader io.Reader = reader
	if reporter != nil && totalSize > 0 {
		ioReader = &progressReader{
			reader:   reader,
			total:    totalSize,
			reporter: reporter,
		}
	}

	// Copy with progress tracking
	written, err := io.Copy(file, ioReader)
	if err != nil {
		return 0, fmt.Errorf("download file: %w", err)
	}

	if written != totalSize {
		return 0, fmt.Errorf("incomplete download: expected %d bytes, got %d", totalSize, written)
	}

	return written, nil
}

// uploadToGCS uploads a local file to Google Cloud Storage
func (c *Client) uploadToGCS(ctx context.Context, localPath, bucket, object string, reporter progress.ProgressReporter) error {
	// Get file size
	fileInfo, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("stat local file: %w", err)
	}

	totalSize := fileInfo.Size()
	c.logger.Info("uploading to GCS", "object", object, "size_bytes", totalSize)

	// Create GCS client
	var opts []option.ClientOption
	if c.config.CredentialsJSON != "" {
		if _, err := os.Stat(c.config.CredentialsJSON); err == nil {
			opts = append(opts, option.WithCredentialsFile(c.config.CredentialsJSON))
		} else {
			opts = append(opts, option.WithCredentialsJSON([]byte(c.config.CredentialsJSON)))
		}
	}

	gcsClient, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("create GCS client: %w", err)
	}
	defer gcsClient.Close()

	// Open local file
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	defer file.Close()

	// Create GCS writer
	bucketHandle := gcsClient.Bucket(bucket)
	objectHandle := bucketHandle.Object(object)
	writer := objectHandle.NewWriter(ctx)

	// Create progress reader wrapper if reporter provided
	var reader io.Reader = file
	if reporter != nil && totalSize > 0 {
		reader = &progressReader{
			reader:   file,
			total:    totalSize,
			reporter: reporter,
		}
	}

	// Copy with progress tracking
	_, err = io.Copy(writer, reader)
	if err != nil {
		writer.Close()
		return fmt.Errorf("upload file: %w", err)
	}

	// Close writer to finalize upload
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finalize upload: %w", err)
	}

	c.logger.Info("upload complete", "object", object)
	return nil
}

// ListGCSObjects lists objects in a GCS bucket with a given prefix
func (c *Client) ListGCSObjects(ctx context.Context, bucket, prefix string) ([]string, error) {
	// Create GCS client
	var opts []option.ClientOption
	if c.config.CredentialsJSON != "" {
		if _, err := os.Stat(c.config.CredentialsJSON); err == nil {
			opts = append(opts, option.WithCredentialsFile(c.config.CredentialsJSON))
		} else {
			opts = append(opts, option.WithCredentialsJSON([]byte(c.config.CredentialsJSON)))
		}
	}

	gcsClient, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("create GCS client: %w", err)
	}
	defer gcsClient.Close()

	bucketHandle := gcsClient.Bucket(bucket)

	// List objects
	query := &storage.Query{Prefix: prefix}
	it := bucketHandle.Objects(ctx, query)

	var objects []string
	for {
		attrs, err := it.Next()
		if err == storage.ErrObjectNotExist || (err != nil && strings.Contains(err.Error(), "no more items")) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("iterate objects: %w", err)
		}

		objects = append(objects, attrs.Name)
	}

	return objects, nil
}

// progressReader wraps an io.Reader to report download/upload progress
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
		percentage := int((pr.current * 100) / pr.total)
		pr.reporter.Update(percentage)
	}

	return n, err
}

// ExportResult contains the result of a GCS export operation
type ExportResult struct {
	ImageName  string
	DiskName   string
	DiskType   string // "boot", "data-1", "data-2", etc.
	Format     string
	LocalPath  string
	Size       int64
	GCSBucket  string
	GCSObject  string
	GCSURI     string
}

// CreateExportManifest creates a JSON manifest file for multi-disk exports
func CreateExportManifest(instanceName string, results []*ExportResult, outputPath string) error {
	manifestPath := filepath.Join(outputPath, fmt.Sprintf("%s-manifest.json", instanceName))

	manifest := fmt.Sprintf(`{
  "instance_name": "%s",
  "export_time": "%s",
  "disk_count": %d,
  "disks": [
`, instanceName, time.Now().Format(time.RFC3339), len(results))

	for i, result := range results {
		comma := ","
		if i == len(results)-1 {
			comma = ""
		}
		manifest += fmt.Sprintf(`    {
      "image_name": "%s",
      "disk_name": "%s",
      "disk_type": "%s",
      "format": "%s",
      "local_path": "%s",
      "size_bytes": %d,
      "gcs_bucket": "%s",
      "gcs_object": "%s",
      "gcs_uri": "%s"
    }%s
`, result.ImageName, result.DiskName, result.DiskType, result.Format, result.LocalPath,
			result.Size, result.GCSBucket, result.GCSObject, result.GCSURI, comma)
	}

	manifest += `  ]
}`

	return os.WriteFile(manifestPath, []byte(manifest), 0644)
}

// Helper function to create string pointer
func makeStringPtr(v string) *string {
	return &v
}
