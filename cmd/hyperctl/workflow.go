// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pterm/pterm"
)

// WorkflowStatus represents the daemon status
type WorkflowStatus struct {
	Mode           string `json:"mode"`
	Running        bool   `json:"running"`
	QueueDepth     int    `json:"queue_depth"`
	ActiveJobs     int    `json:"active_jobs"`
	ProcessedToday int    `json:"processed_today"`
	FailedToday    int    `json:"failed_today"`
	MaxWorkers     int    `json:"max_workers"`
	UptimeSeconds  int    `json:"uptime_seconds"`
}

// WorkflowJob represents a job in the workflow
type WorkflowJob struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Stage          string    `json:"stage"`
	Progress       int       `json:"progress"`
	StartedAt      time.Time `json:"started_at"`
	ElapsedSeconds int       `json:"elapsed_seconds"`
	Status         string    `json:"status"`
}

// QueueStats represents queue statistics
type QueueStats struct {
	ToBeProcessed  int `json:"to_be_processed"`
	Processing     int `json:"processing"`
	ProcessedToday int `json:"processed_today"`
	FailedToday    int `json:"failed_today"`
}

// handleWorkflow handles workflow-related commands
func handleWorkflow(daemonURL, operation, workflowDir string) {
	switch operation {
	case "status":
		handleWorkflowStatus(daemonURL)
	case "list":
		handleWorkflowList(daemonURL)
	case "queue":
		handleWorkflowQueue(workflowDir)
	case "watch":
		handleWorkflowWatch(workflowDir)
	default:
		pterm.Error.Printfln("Unknown workflow operation: %s", operation)
		pterm.Info.Println("Available operations: status, list, queue, watch")
		os.Exit(1)
	}
}

// handleWorkflowStatus shows workflow daemon status
func handleWorkflowStatus(daemonURL string) {
	spinner, _ := pterm.DefaultSpinner.Start("📊 Getting workflow status...")

	// Check if workflow daemon is accessible
	workflowURL := daemonURL + "/api/workflow/status"
	resp, err := apiRequest(workflowURL, "GET", "", nil)
	if err != nil {
		// Fallback to file-based status if API not available
		spinner.Warning("API not available, checking local status...")
		showLocalWorkflowStatus()
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		spinner.Fail("Workflow daemon not running or not configured")
		pterm.Info.Println("")
		pterm.Info.Println("To enable workflow mode:")
		pterm.Println("  1. Create workflow config: /etc/hyper2kvm/workflow-daemon.yaml")
		pterm.Println("  2. Start daemon: sudo hyper2kvm --config /etc/hyper2kvm/workflow-daemon.yaml")
		pterm.Println("  3. Or use systemd: sudo systemctl start hyper2kvm-workflow.service")
		return
	}

	var status WorkflowStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		os.Exit(1)
	}

	spinner.Success("Workflow daemon is running")
	pterm.Println()

	// Display status
	displayWorkflowStatus(status)
}

// displayWorkflowStatus renders workflow status in a nice format
func displayWorkflowStatus(status WorkflowStatus) {
	pterm.DefaultSection.Println("🔄 Workflow Daemon Status")
	pterm.Println()

	// Status overview
	statusData := [][]string{
		{"Property", "Value"},
		{"Status", colorizeRunningStatus(status.Running)},
		{"Mode", strings.ToUpper(status.Mode)},
		{"Max Workers", fmt.Sprintf("%d", status.MaxWorkers)},
		{"Uptime", formatDuration(status.UptimeSeconds)},
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(statusData).
		Render()

	pterm.Println()

	// Queue statistics
	pterm.DefaultSection.Println("📊 Queue Statistics")
	pterm.Println()

	queueData := [][]string{
		{"Queue", "Count"},
		{"📥 To Be Processed", pterm.LightCyan(fmt.Sprintf("%d", status.QueueDepth))},
		{"🔄 Processing", pterm.Yellow(fmt.Sprintf("%d", status.ActiveJobs))},
		{"✅ Processed (today)", pterm.Green(fmt.Sprintf("%d", status.ProcessedToday))},
		{"❌ Failed (today)", pterm.Red(fmt.Sprintf("%d", status.FailedToday))},
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(queueData).
		Render()

	pterm.Println()

	// Show helpful next steps
	if status.QueueDepth > 0 {
		pterm.Info.Printfln("💡 You have %d job(s) queued for processing", status.QueueDepth)
	} else if status.ActiveJobs > 0 {
		pterm.Info.Printfln("💡 Currently processing %d job(s)", status.ActiveJobs)
	} else {
		pterm.Success.Println("✅ All queues are empty!")
		pterm.Info.Println("")
		pterm.Info.Println("To submit jobs:")
		if status.Mode == "disk" {
			pterm.Println("  cp my-vm.vmdk /var/lib/hyper2kvm/workflow/to_be_processed/")
		} else {
			pterm.Println("  hyperctl manifest submit -file my-vm-manifest.json")
		}
	}
}

// handleWorkflowList lists workflow jobs
func handleWorkflowList(daemonURL string) {
	spinner, _ := pterm.DefaultSpinner.Start("📋 Fetching workflow jobs...")

	resp, err := apiRequest(daemonURL+"/api/workflow/jobs", "GET", "", nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to fetch jobs: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	var jobsResp struct {
		Jobs  []WorkflowJob `json:"jobs"`
		Total int           `json:"total"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&jobsResp); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		os.Exit(1)
	}

	spinner.Success(fmt.Sprintf("Found %d job(s)", jobsResp.Total))
	pterm.Println()

	if jobsResp.Total == 0 {
		pterm.Info.Println("No jobs found")
		return
	}

	displayWorkflowJobs(jobsResp.Jobs)
}

// displayWorkflowJobs displays jobs in a table
func displayWorkflowJobs(jobs []WorkflowJob) {
	pterm.DefaultSection.Println("📦 Workflow Jobs")
	pterm.Println()

	data := [][]string{
		{"ID", "Name", "Stage", "Status", "Progress", "Elapsed"},
	}

	for _, job := range jobs {
		id := job.ID
		if len(id) > 8 {
			id = id[:8] + "..."
		}

		status := colorizeStatus(job.Status)

		progress := "-"
		if job.Progress > 0 {
			progress = fmt.Sprintf("%d%%", job.Progress)
		}

		elapsed := formatDuration(job.ElapsedSeconds)

		data = append(data, []string{
			id,
			truncate(job.Name, 30),
			job.Stage,
			status,
			progress,
			elapsed,
		})
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()
}

// handleWorkflowQueue shows queue status from filesystem
func handleWorkflowQueue(workflowDir string) {
	if workflowDir == "" {
		workflowDir = "/var/lib/hyper2kvm/workflow"
	}

	spinner, _ := pterm.DefaultSpinner.Start("📊 Scanning workflow directories...")

	stats := scanWorkflowDirectories(workflowDir)

	spinner.Success("Scanned workflow directories")
	pterm.Println()

	displayQueueStats(stats)
}

// scanWorkflowDirectories scans workflow directories for stats
func scanWorkflowDirectories(workflowDir string) QueueStats {
	var stats QueueStats

	// Count files in each directory
	toBeProcessedDir := filepath.Join(workflowDir, "to_be_processed")
	processingDir := filepath.Join(workflowDir, "processing")
	processedDir := filepath.Join(workflowDir, "processed")
	failedDir := filepath.Join(workflowDir, "failed")

	stats.ToBeProcessed = countFiles(toBeProcessedDir)
	stats.Processing = countFiles(processingDir)

	// Count today's processed and failed
	today := time.Now().Format("2006-01-02")
	stats.ProcessedToday = countFiles(filepath.Join(processedDir, today))
	stats.FailedToday = countFiles(filepath.Join(failedDir, today))

	return stats
}

// countFiles counts files in a directory
func countFiles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			count++
		}
	}
	return count
}

// displayQueueStats displays queue statistics
func displayQueueStats(stats QueueStats) {
	pterm.DefaultSection.Println("📊 Queue Statistics")
	pterm.Println()

	data := [][]string{
		{"Queue", "Files"},
		{"📥 To Be Processed", pterm.LightCyan(fmt.Sprintf("%d", stats.ToBeProcessed))},
		{"🔄 Processing", pterm.Yellow(fmt.Sprintf("%d", stats.Processing))},
		{"✅ Processed (today)", pterm.Green(fmt.Sprintf("%d", stats.ProcessedToday))},
		{"❌ Failed (today)", pterm.Red(fmt.Sprintf("%d", stats.FailedToday))},
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()

	pterm.Println()

	// Show directory paths
	pterm.Info.Println("📂 Directory Locations:")
	pterm.Println("  to_be_processed/  - Drop files here to queue them")
	pterm.Println("  processing/       - Currently being processed")
	pterm.Println("  processed/        - Successfully completed")
	pterm.Println("  failed/           - Failed jobs with error details")
}

// handleWorkflowWatch watches workflow directory for changes
func handleWorkflowWatch(workflowDir string) {
	if workflowDir == "" {
		workflowDir = "/var/lib/hyper2kvm/workflow"
	}

	pterm.Info.Printfln("👀 Watching workflow directory: %s", workflowDir)
	pterm.Info.Println("Press Ctrl+C to stop")
	pterm.Println()

	// Initial scan
	lastStats := scanWorkflowDirectories(workflowDir)
	displayQueueStats(lastStats)

	// Watch for changes
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := scanWorkflowDirectories(workflowDir)

		// Check if anything changed
		if stats != lastStats {
			// Clear screen (simple version)
			pterm.Println("\n" + strings.Repeat("=", 80))
			pterm.Info.Printfln("Updated at %s", time.Now().Format("15:04:05"))
			pterm.Println()
			displayQueueStats(stats)
			lastStats = stats
		}
	}
}

// showLocalWorkflowStatus shows status from local filesystem
func showLocalWorkflowStatus() {
	workflowDirs := []string{
		"/var/lib/hyper2kvm/workflow",
		"/var/lib/hyper2kvm/manifest-workflow",
	}

	pterm.DefaultSection.Println("📊 Local Workflow Status")
	pterm.Println()

	for _, dir := range workflowDirs {
		if _, err := os.Stat(dir); err == nil {
			mode := "disk"
			if strings.Contains(dir, "manifest") {
				mode = "manifest"
			}

			pterm.Info.Printfln("Found %s workflow: %s", mode, dir)
			stats := scanWorkflowDirectories(dir)

			pterm.Println(fmt.Sprintf("  📥 Queue: %d | 🔄 Active: %d | ✅ Done: %d | ❌ Failed: %d",
				stats.ToBeProcessed,
				stats.Processing,
				stats.ProcessedToday,
				stats.FailedToday))
			pterm.Println()
		}
	}
}

// colorizeRunningStatus colorizes running status
func colorizeRunningStatus(running bool) string {
	if running {
		return pterm.Green("🟢 Running")
	}
	return pterm.Red("🔴 Stopped")
}
