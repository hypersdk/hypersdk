// SPDX-License-Identifier: LGPL-3.0-or-later

package exporters

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"hypersdk/daemon/capabilities"
	"hypersdk/daemon/models"
	"hypersdk/logger"
)

// GovcExporter implements export using govc binary
type GovcExporter struct {
	govcPath string
	logger   logger.Logger
}

// NewGovcExporter creates a new govc exporter
func NewGovcExporter(govcPath string, log logger.Logger) *GovcExporter {
	return &GovcExporter{
		govcPath: govcPath,
		logger:   log,
	}
}

// Export performs VM export using govc CLI
func (e *GovcExporter) Export(ctx context.Context, job *models.JobDefinition, progressCallback func(*models.JobProgress)) (*models.JobResult, error) {
	// Validate job first
	if err := e.Validate(job); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	e.logger.Info("starting govc export", "vm", job.VMPath, "govc_path", e.govcPath)

	// Set govc environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("GOVC_URL=https://%s/sdk", job.VCenter.Server))
	env = append(env, fmt.Sprintf("GOVC_USERNAME=%s", job.VCenter.Username))
	env = append(env, fmt.Sprintf("GOVC_PASSWORD=%s", job.VCenter.Password))

	if job.VCenter.Insecure {
		env = append(env, "GOVC_INSECURE=true")
	}

	// Build govc export command
	// govc export.ovf -vm <path> <output-dir>
	outputPath := filepath.Join(job.OutputDir, filepath.Base(job.VMPath))
	args := []string{
		"export.ovf",
		"-vm", job.VMPath,
		outputPath,
	}

	// Create command
	cmd := exec.CommandContext(ctx, e.govcPath, args...)
	cmd.Env = env

	// Get stdout/stderr pipes
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("create stderr pipe: %w", err)
	}

	// Update progress
	if progressCallback != nil {
		progressCallback(&models.JobProgress{
			Phase:        "connecting",
			ExportMethod: string(capabilities.ExportMethodGovc),
		})
	}

	// Start command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start govc: %w", err)
	}

	// Wrap pipes in bufio.Reader for monitoring
	stdout := bufio.NewReader(stdoutPipe)
	stderr := bufio.NewReader(stderrPipe)

	// Monitor output
	go e.monitorOutput(stdout, progressCallback)
	go e.monitorErrors(stderr)

	// Wait for completion
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("govc export failed: %w", err)
	}

	// Build result
	result := &models.JobResult{
		Success:      true,
		ExportMethod: string(capabilities.ExportMethodGovc),
		// Note: Would need to scan output dir to get file list
	}

	e.logger.Info("govc export completed successfully", "vm", job.VMPath)

	return result, nil
}

// monitorOutput parses stdout
func (e *GovcExporter) monitorOutput(stdout *bufio.Reader, progressCallback func(*models.JobProgress)) {
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		e.logger.Debug("govc output", "line", line)

		if progressCallback != nil {
			progressCallback(&models.JobProgress{
				Phase:        "exporting",
				CurrentStep:  line,
				ExportMethod: string(capabilities.ExportMethodGovc),
			})
		}
	}
}

// monitorErrors logs stderr
func (e *GovcExporter) monitorErrors(stderr *bufio.Reader) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()
		e.logger.Warn("govc stderr", "line", line)
	}
}

// Method returns the export method name
func (e *GovcExporter) Method() capabilities.ExportMethod {
	return capabilities.ExportMethodGovc
}

// Validate checks if this exporter can handle the job
func (e *GovcExporter) Validate(job *models.JobDefinition) error {
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
