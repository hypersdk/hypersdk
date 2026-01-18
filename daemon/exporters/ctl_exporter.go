// SPDX-License-Identifier: LGPL-3.0-or-later

package exporters

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"hypersdk/daemon/capabilities"
	"hypersdk/daemon/models"
	"hypersdk/logger"
)

// CTLExporter implements export using hyperctl binary
type CTLExporter struct {
	ctlPath string
	logger  logger.Logger
}

// NewCTLExporter creates a new CTL exporter
func NewCTLExporter(ctlPath string, log logger.Logger) *CTLExporter {
	return &CTLExporter{
		ctlPath: ctlPath,
		logger:  log,
	}
}

// Export performs VM export using hyperctl CLI
func (e *CTLExporter) Export(ctx context.Context, job *models.JobDefinition, progressCallback func(*models.JobProgress)) (*models.JobResult, error) {
	// Validate job first
	if err := e.Validate(job); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	e.logger.Info("starting CTL export", "vm", job.VMPath, "ctl_path", e.ctlPath)

	// Build hyperctl export command
	args := []string{
		"export",
		"--vm", job.VMPath,
		"--output", job.OutputDir,
		"--format", job.Format,
		"--server", job.VCenter.Server,
		"--username", job.VCenter.Username,
		"--password", job.VCenter.Password,
	}

	if job.VCenter.Insecure {
		args = append(args, "--insecure")
	}

	if job.Compress {
		args = append(args, "--compress")
	}

	if job.Thin {
		args = append(args, "--thin")
	}

	// Create command
	cmd := exec.CommandContext(ctx, e.ctlPath, args...)

	// Get stdout/stderr pipes for progress monitoring
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start hyperctl: %w", err)
	}

	// Wrap pipes in bufio.Reader for monitoring
	stdout := bufio.NewReader(stdoutPipe)
	stderr := bufio.NewReader(stderrPipe)

	// Monitor progress from stdout
	go e.monitorProgress(stdout, progressCallback)

	// Monitor errors from stderr
	go e.monitorErrors(stderr)

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("hyperctl export failed: %w", err)
	}

	// Build result (hyperctl should write files to output dir)
	result := &models.JobResult{
		Success:      true,
		ExportMethod: string(capabilities.ExportMethodCTL),
		// Note: Would need to scan output dir to get file list and sizes
		// For now, assume success
	}

	e.logger.Info("CTL export completed successfully", "vm", job.VMPath)

	return result, nil
}

// monitorProgress parses stdout for progress updates
func (e *CTLExporter) monitorProgress(stdout *bufio.Reader, progressCallback func(*models.JobProgress)) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		e.logger.Debug("hyperctl output", "line", line)

		// Parse progress from output (format: "Progress: 45.5%")
		if strings.Contains(line, "Progress:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				percentStr := strings.TrimSpace(strings.TrimSuffix(parts[1], "%"))
				if percent, err := strconv.ParseFloat(percentStr, 64); err == nil {
					if progressCallback != nil {
						progressCallback(&models.JobProgress{
							Phase:           "exporting",
							PercentComplete: percent,
							CurrentStep:     line,
							ExportMethod:    string(capabilities.ExportMethodCTL),
						})
					}
				}
			}
		}
	}
}

// monitorErrors logs stderr output
func (e *CTLExporter) monitorErrors(stderr *bufio.Reader) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		e.logger.Warn("hyperctl stderr", "line", line)
	}
}

// Method returns the export method name
func (e *CTLExporter) Method() capabilities.ExportMethod {
	return capabilities.ExportMethodCTL
}

// Validate checks if this exporter can handle the job
func (e *CTLExporter) Validate(job *models.JobDefinition) error {
	if job.VMPath == "" {
		return fmt.Errorf("vm_path is required")
	}

	if job.OutputDir == "" {
		return fmt.Errorf("output_dir is required")
	}

	if job.VCenter == nil || job.VCenter.Server == "" {
		return fmt.Errorf("vcenter server is required")
	}

	return nil
}
