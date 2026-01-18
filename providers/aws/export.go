// SPDX-License-Identifier: LGPL-3.0-or-later

package aws

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"hypersdk/progress"
)

// ExportInstanceToS3 exports an EC2 instance to S3 as VMDK
func (c *Client) ExportInstanceToS3(ctx context.Context, instanceID, outputDir string, reporter progress.ProgressReporter) (*ExportResult, error) {
	c.logger.Info("starting EC2 instance export to S3", "instance", instanceID)

	if reporter != nil {
		reporter.Describe("Exporting EC2 instance to S3")
	}

	// Get instance details
	instance, err := c.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("get instance details: %w", err)
	}

	// Create export task
	taskID, err := c.createExportTask(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("create export task: %w", err)
	}

	c.logger.Info("export task created", "task_id", taskID)

	// Wait for export to complete
	s3Key, err := c.waitForExportTask(ctx, taskID, reporter)
	if err != nil {
		return nil, fmt.Errorf("wait for export task: %w", err)
	}

	c.logger.Info("export task completed", "s3_key", s3Key)

	// Download from S3
	if reporter != nil {
		reporter.Describe("Downloading VMDK from S3")
	}

	localPath := filepath.Join(outputDir, fmt.Sprintf("%s.vmdk", instanceID))
	size, err := c.downloadFromS3(ctx, s3Key, localPath, reporter)
	if err != nil {
		return nil, fmt.Errorf("download from S3: %w", err)
	}

	c.logger.Info("VMDK downloaded successfully", "path", localPath, "size_bytes", size)

	return &ExportResult{
		InstanceID: instanceID,
		ImageID:    instance.ImageID,
		Format:     "vmdk",
		LocalPath:  localPath,
		Size:       size,
		S3Bucket:   c.config.S3Bucket,
		S3Key:      s3Key,
	}, nil
}

// ExportSnapshotToS3 exports an EBS snapshot to S3 as VMDK
func (c *Client) ExportSnapshotToS3(ctx context.Context, snapshotID, outputDir string, reporter progress.ProgressReporter) (*ExportResult, error) {
	c.logger.Info("starting EBS snapshot export to S3", "snapshot", snapshotID)

	if reporter != nil {
		reporter.Describe("Exporting EBS snapshot to S3")
	}

	// Create export task for snapshot
	input := &ec2.ExportSnapshotInput{
		SnapshotId: aws.String(snapshotID),
		DiskImageFormat: types.DiskImageFormatVmdk,
		S3Bucket: aws.String(c.config.S3Bucket),
		S3Prefix: aws.String("exports/snapshots/"),
	}

	result, err := c.ec2Client.ExportSnapshot(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("create snapshot export task: %w", err)
	}

	exportTaskID := aws.ToString(result.SnapshotTaskDetail.SnapshotId)
	c.logger.Info("snapshot export task created", "task_id", exportTaskID)

	// Wait for export completion
	s3Key, err := c.waitForSnapshotExport(ctx, exportTaskID, reporter)
	if err != nil {
		return nil, fmt.Errorf("wait for snapshot export: %w", err)
	}

	// Download from S3
	localPath := filepath.Join(outputDir, fmt.Sprintf("%s.vmdk", snapshotID))
	size, err := c.downloadFromS3(ctx, s3Key, localPath, reporter)
	if err != nil {
		return nil, fmt.Errorf("download snapshot from S3: %w", err)
	}

	return &ExportResult{
		SnapshotID: snapshotID,
		Format:     "vmdk",
		LocalPath:  localPath,
		Size:       size,
		S3Bucket:   c.config.S3Bucket,
		S3Key:      s3Key,
	}, nil
}

// createExportTask creates an EC2 instance export task
func (c *Client) createExportTask(ctx context.Context, instanceID string) (string, error) {
	input := &ec2.CreateInstanceExportTaskInput{
		InstanceId: aws.String(instanceID),
		TargetEnvironment: types.ExportEnvironmentVmware,
		ExportToS3Task: &types.ExportToS3TaskSpecification{
			DiskImageFormat: types.DiskImageFormatVmdk,
			S3Bucket:        aws.String(c.config.S3Bucket),
			S3Prefix:        aws.String("exports/instances/"),
		},
	}

	result, err := c.ec2Client.CreateInstanceExportTask(ctx, input)
	if err != nil {
		return "", err
	}

	return aws.ToString(result.ExportTask.ExportTaskId), nil
}

// waitForExportTask polls export task status until completion
func (c *Client) waitForExportTask(ctx context.Context, taskID string, reporter progress.ProgressReporter) (string, error) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	timeout := time.After(2 * time.Hour) // EC2 exports can take a while

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", fmt.Errorf("export task timed out after 2 hours")
		case <-ticker.C:
			input := &ec2.DescribeExportTasksInput{
				ExportTaskIds: []string{taskID},
			}

			result, err := c.ec2Client.DescribeExportTasks(ctx, input)
			if err != nil {
				return "", fmt.Errorf("describe export task: %w", err)
			}

			if len(result.ExportTasks) == 0 {
				return "", fmt.Errorf("export task %s not found", taskID)
			}

			task := result.ExportTasks[0]
			state := task.State

			c.logger.Debug("export task status", "task_id", taskID, "state", state)

			// Update progress if available
			if reporter != nil && task.StatusMessage != nil {
				reporter.Describe(aws.ToString(task.StatusMessage))
			}

			switch state {
			case types.ExportTaskStateCompleted:
				// Extract S3 key from task
				if task.ExportToS3Task != nil && task.ExportToS3Task.S3Key != nil {
					return aws.ToString(task.ExportToS3Task.S3Key), nil
				}
				return "", fmt.Errorf("export completed but S3 key not found")

			case types.ExportTaskStateCancelled:
				return "", fmt.Errorf("export task was cancelled")

			case types.ExportTaskStateCancelling:
				return "", fmt.Errorf("export task is being cancelled")

			case types.ExportTaskStateActive:
				// Still in progress, continue polling
				continue

			default:
				c.logger.Warn("unknown export task state", "state", state)
				continue
			}
		}
	}
}

// waitForSnapshotExport polls snapshot export task status until completion
func (c *Client) waitForSnapshotExport(ctx context.Context, snapshotID string, reporter progress.ProgressReporter) (string, error) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	timeout := time.After(2 * time.Hour)

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", fmt.Errorf("snapshot export timed out after 2 hours")
		case <-ticker.C:
			input := &ec2.DescribeExportSnapshotTasksInput{
				ExportTaskIds: []string{snapshotID},
			}

			result, err := c.ec2Client.DescribeExportSnapshotTasks(ctx, input)
			if err != nil {
				return "", fmt.Errorf("describe snapshot export task: %w", err)
			}

			if len(result.ExportSnapshotTasks) == 0 {
				return "", fmt.Errorf("snapshot export task %s not found", snapshotID)
			}

			task := result.ExportSnapshotTasks[0]
			detail := task.SnapshotTaskDetail

			if detail == nil {
				continue
			}

			// Update progress
			if reporter != nil && detail.Progress != nil {
				progress := int(aws.ToFloat64(detail.Progress))
				reporter.Update(progress)
				if detail.StatusMessage != nil {
					reporter.Describe(aws.ToString(detail.StatusMessage))
				}
			}

			status := aws.ToString(detail.Status)

			switch status {
			case "completed":
				if detail.S3Key != nil {
					return aws.ToString(detail.S3Key), nil
				}
				return "", fmt.Errorf("snapshot export completed but S3 key not found")

			case "error":
				errMsg := "unknown error"
				if detail.StatusMessage != nil {
					errMsg = aws.ToString(detail.StatusMessage)
				}
				return "", fmt.Errorf("snapshot export failed: %s", errMsg)

			case "active":
				// Still in progress
				continue

			default:
				c.logger.Warn("unknown snapshot export status", "status", status)
				continue
			}
		}
	}
}

// downloadFromS3 downloads a file from S3 with progress tracking
func (c *Client) downloadFromS3(ctx context.Context, s3Key, localPath string, reporter progress.ProgressReporter) (int64, error) {
	// Get object metadata to determine size
	headInput := &s3.HeadObjectInput{
		Bucket: aws.String(c.config.S3Bucket),
		Key:    aws.String(s3Key),
	}

	headResult, err := c.s3Client.HeadObject(ctx, headInput)
	if err != nil {
		return 0, fmt.Errorf("get object metadata: %w", err)
	}

	totalSize := aws.ToInt64(headResult.ContentLength)
	c.logger.Info("downloading from S3", "key", s3Key, "size_bytes", totalSize)

	// Create local file
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return 0, fmt.Errorf("create output directory: %w", err)
	}

	file, err := os.Create(localPath)
	if err != nil {
		return 0, fmt.Errorf("create local file: %w", err)
	}
	defer file.Close()

	// Download object
	getInput := &s3.GetObjectInput{
		Bucket: aws.String(c.config.S3Bucket),
		Key:    aws.String(s3Key),
	}

	result, err := c.s3Client.GetObject(ctx, getInput)
	if err != nil {
		return 0, fmt.Errorf("get object from S3: %w", err)
	}
	defer result.Body.Close()

	// Create progress reader wrapper if reporter provided
	var reader io.Reader = result.Body
	if reporter != nil {
		reader = &progressReader{
			reader:   result.Body,
			total:    totalSize,
			reporter: reporter,
		}
	}

	// Copy with progress tracking
	written, err := io.Copy(file, reader)
	if err != nil {
		return 0, fmt.Errorf("download file: %w", err)
	}

	if written != totalSize {
		return 0, fmt.Errorf("incomplete download: expected %d bytes, got %d", totalSize, written)
	}

	return written, nil
}

// progressReader wraps an io.Reader to report progress
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

// ExportResult contains the result of an export operation
type ExportResult struct {
	InstanceID string
	ImageID    string
	SnapshotID string
	Format     string
	LocalPath  string
	Size       int64
	S3Bucket   string
	S3Key      string
}
