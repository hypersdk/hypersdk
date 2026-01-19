// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pterm/pterm"

	"hypersdk/daemon/models"
	"hypersdk/daemon/webhooks"
)

// handleSchedules manages scheduled jobs
func handleSchedules(daemonURL string, action string, args []string) {
	switch action {
	case "list":
		listSchedules(daemonURL)
	case "create":
		if len(args) < 2 {
			pterm.Error.Println("Usage: hyperctl schedules create <name> <cron-schedule> [-vm <path>] [-output <dir>]")
			pterm.Info.Println("Example: hyperctl schedules create daily-backup '0 2 * * *' -vm /dc/vm/prod -output /backups")
			os.Exit(1)
		}
		createSchedule(daemonURL, args)
	case "delete":
		if len(args) < 1 {
			pterm.Error.Println("Usage: hyperctl schedules delete <schedule-id>")
			os.Exit(1)
		}
		deleteSchedule(daemonURL, args[0])
	case "enable":
		if len(args) < 1 {
			pterm.Error.Println("Usage: hyperctl schedules enable <schedule-id>")
			os.Exit(1)
		}
		toggleSchedule(daemonURL, args[0], true)
	case "disable":
		if len(args) < 1 {
			pterm.Error.Println("Usage: hyperctl schedules disable <schedule-id>")
			os.Exit(1)
		}
		toggleSchedule(daemonURL, args[0], false)
	case "trigger":
		if len(args) < 1 {
			pterm.Error.Println("Usage: hyperctl schedules trigger <schedule-id>")
			os.Exit(1)
		}
		triggerSchedule(daemonURL, args[0])
	default:
		pterm.Error.Printfln("Unknown schedule action: %s", action)
		pterm.Info.Println("Available actions: list, create, delete, enable, disable, trigger")
		os.Exit(1)
	}
}

func listSchedules(daemonURL string) {
	spinner, _ := pterm.DefaultSpinner.Start("📅 Loading schedules...")

	resp, err := apiRequest(daemonURL+"/schedules", "GET", "", nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to list schedules: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	var schedResp struct {
		Schedules []models.ScheduledJob `json:"schedules"`
		Total     int                   `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&schedResp); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		os.Exit(1)
	}

	spinner.Success(fmt.Sprintf("Found %d schedules", len(schedResp.Schedules)))

	if len(schedResp.Schedules) == 0 {
		pterm.Info.Println("No schedules found")
		pterm.Println()
		pterm.Info.Println("💡 Create a schedule:")
		pterm.Println("  hyperctl schedules create daily-backup '0 2 * * *' -vm /dc/vm/prod")
		return
	}

	// Display schedules
	pterm.DefaultSection.Println("📅 Scheduled Jobs")
	data := [][]string{
		{"ID", "Name", "Schedule", "Enabled", "Last Run", "Next Run"},
	}

	for _, sched := range schedResp.Schedules {
		id := sched.ID
		if len(id) > 12 {
			id = id[:12] + "..."
		}

		enabled := "❌"
		if sched.Enabled {
			enabled = "✅"
		}

		lastRun := "-"
		if sched.LastRun != nil {
			lastRun = sched.LastRun.Format("15:04 01/02")
		}

		nextRun := "-"
		if !sched.NextRun.IsZero() {
			nextRun = sched.NextRun.Format("15:04 01/02")
		}

		data = append(data, []string{
			id,
			sched.Name,
			sched.Schedule,
			enabled,
			lastRun,
			nextRun,
		})
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()

	pterm.Println()
	pterm.Info.Println("💡 Manage schedules:")
	pterm.Println("  hyperctl schedules enable <id>    - Enable a schedule")
	pterm.Println("  hyperctl schedules disable <id>   - Disable a schedule")
	pterm.Println("  hyperctl schedules trigger <id>   - Run now")
	pterm.Println("  hyperctl schedules delete <id>    - Delete schedule")
}

func createSchedule(daemonURL string, args []string) {
	name := args[0]
	cronSchedule := args[1]

	// Parse additional flags
	var vmPath, outputPath string
	for i := 2; i < len(args); i++ {
		if args[i] == "-vm" && i+1 < len(args) {
			vmPath = args[i+1]
			i++
		} else if args[i] == "-output" && i+1 < len(args) {
			outputPath = args[i+1]
			i++
		}
	}

	if vmPath == "" {
		pterm.Error.Println("VM path required: -vm <path>")
		os.Exit(1)
	}

	spinner, _ := pterm.DefaultSpinner.Start("Creating schedule...")

	schedule := models.ScheduledJob{
		ID:       generateID(),
		Name:     name,
		Schedule: cronSchedule,
		Enabled:  true,
		JobTemplate: models.JobDefinition{
			VMPath:     vmPath,
			OutputPath: outputPath,
		},
	}

	data, _ := json.Marshal(schedule)
	resp, err := apiRequest(daemonURL+"/schedules", "POST", "application/json", data)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to create schedule: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		os.Exit(1)
	}

	spinner.Success("Schedule created!")
	pterm.Success.Printfln("✅ Created schedule: %s", name)
	pterm.Info.Printfln("   Cron: %s", cronSchedule)
	pterm.Info.Printfln("   VM: %s", vmPath)
}

func deleteSchedule(daemonURL, scheduleID string) {
	spinner, _ := pterm.DefaultSpinner.Start("Deleting schedule...")

	resp, err := apiRequest(daemonURL+"/schedules/"+scheduleID, "DELETE", "", nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to delete schedule: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		os.Exit(1)
	}

	spinner.Success("Schedule deleted!")
	pterm.Success.Printfln("✅ Deleted schedule: %s", scheduleID)
}

func toggleSchedule(daemonURL, scheduleID string, enable bool) {
	action := "disable"
	if enable {
		action = "enable"
	}

	spinner, _ := pterm.DefaultSpinner.Start(fmt.Sprintf("%sing schedule...", strings.Title(action)))

	resp, err := apiRequest(daemonURL+"/schedules/"+scheduleID+"/"+action, "POST", "", nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to %s schedule: %v", action, err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		os.Exit(1)
	}

	spinner.Success(fmt.Sprintf("Schedule %sd!", action))
	pterm.Success.Printfln("✅ Schedule %sd: %s", action, scheduleID)
}

func triggerSchedule(daemonURL, scheduleID string) {
	spinner, _ := pterm.DefaultSpinner.Start("Triggering schedule...")

	resp, err := apiRequest(daemonURL+"/schedules/"+scheduleID+"/trigger", "POST", "", nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to trigger schedule: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		os.Exit(1)
	}

	spinner.Success("Schedule triggered!")
	pterm.Success.Printfln("✅ Triggered schedule: %s", scheduleID)
	pterm.Info.Println("💡 Check job status: hyperctl query -status running")
}

// handleWebhooks manages webhooks
func handleWebhooks(daemonURL string, action string, args []string) {
	switch action {
	case "list":
		listWebhooks(daemonURL)
	case "add":
		if len(args) < 1 {
			pterm.Error.Println("Usage: hyperctl webhooks add <url> [events...]")
			pterm.Info.Println("Events: job.started, job.completed, job.failed, schedule.triggered")
			pterm.Info.Println("Example: hyperctl webhooks add https://hooks.slack.com/xxx job.completed job.failed")
			pterm.Info.Println("If no events specified, defaults to: job.started, job.completed, job.failed")
			os.Exit(1)
		}
		addWebhook(daemonURL, args)
	case "delete":
		if len(args) < 1 {
			pterm.Error.Println("Usage: hyperctl webhooks delete <index>")
			os.Exit(1)
		}
		deleteWebhook(daemonURL, args[0])
	case "test":
		if len(args) < 1 {
			pterm.Error.Println("Usage: hyperctl webhooks test <index>")
			os.Exit(1)
		}
		testWebhook(daemonURL, args[0])
	default:
		pterm.Error.Printfln("Unknown webhook action: %s", action)
		pterm.Info.Println("Available actions: list, add, delete, test")
		os.Exit(1)
	}
}

func listWebhooks(daemonURL string) {
	spinner, _ := pterm.DefaultSpinner.Start("🔔 Loading webhooks...")

	resp, err := apiRequest(daemonURL+"/webhooks", "GET", "", nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to list webhooks: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	var webhookResp struct {
		Webhooks []webhooks.Webhook `json:"webhooks"`
		Total    int                `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&webhookResp); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		os.Exit(1)
	}

	spinner.Success(fmt.Sprintf("Found %d webhooks", len(webhookResp.Webhooks)))

	if len(webhookResp.Webhooks) == 0 {
		pterm.Info.Println("No webhooks configured")
		pterm.Println()
		pterm.Info.Println("💡 Add a webhook:")
		pterm.Println("  hyperctl webhooks add https://hooks.slack.com/xxx slack job.completed")
		return
	}

	// Display webhooks
	pterm.DefaultSection.Println("🔔 Webhooks")
	data := [][]string{
		{"#", "URL", "Events", "Enabled"},
	}

	for i, wh := range webhookResp.Webhooks {
		url := wh.URL
		if len(url) > 50 {
			url = url[:47] + "..."
		}

		events := strings.Join(wh.Events, ", ")
		if len(events) > 40 {
			events = events[:37] + "..."
		}

		enabled := "❌"
		if wh.Enabled {
			enabled = "✅"
		}

		data = append(data, []string{
			fmt.Sprintf("%d", i),
			url,
			events,
			enabled,
		})
	}

	pterm.DefaultTable.
		WithHasHeader().
		WithHeaderRowSeparator("-").
		WithBoxed().
		WithData(data).
		Render()

	pterm.Println()
	pterm.Info.Println("💡 Manage webhooks:")
	pterm.Println("  hyperctl webhooks test <index>   - Test a webhook")
	pterm.Println("  hyperctl webhooks delete <index> - Delete webhook")
}

func addWebhook(daemonURL string, args []string) {
	url := args[0]
	events := args[1:]

	if len(events) == 0 {
		events = []string{"job.started", "job.completed", "job.failed"}
	}

	spinner, _ := pterm.DefaultSpinner.Start("Adding webhook...")

	webhook := webhooks.Webhook{
		URL:     url,
		Events:  events,
		Enabled: true,
	}

	data, _ := json.Marshal(webhook)
	resp, err := apiRequest(daemonURL+"/webhooks", "POST", "application/json", data)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to add webhook: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		os.Exit(1)
	}

	spinner.Success("Webhook added!")
	pterm.Success.Println("✅ Webhook added successfully")
	pterm.Info.Printfln("   URL: %s", url)
	pterm.Info.Printfln("   Events: %s", strings.Join(events, ", "))
}

func deleteWebhook(daemonURL, index string) {
	spinner, _ := pterm.DefaultSpinner.Start("Deleting webhook...")

	resp, err := apiRequest(daemonURL+"/webhooks/"+index, "DELETE", "", nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to delete webhook: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		os.Exit(1)
	}

	spinner.Success("Webhook deleted!")
	pterm.Success.Printfln("✅ Deleted webhook #%s", index)
}

func testWebhook(daemonURL, index string) {
	spinner, _ := pterm.DefaultSpinner.Start("Testing webhook...")

	resp, err := apiRequest(daemonURL+"/webhooks/"+index+"/test", "POST", "", nil)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to test webhook: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		os.Exit(1)
	}

	spinner.Success("Webhook test sent!")
	pterm.Success.Printfln("✅ Test message sent to webhook #%s", index)
	pterm.Info.Println("   Check your webhook endpoint for the test message")
}

// handleWatch provides real-time job monitoring
func handleWatch(daemonURL string, jobID string) {
	pterm.DefaultSection.Println("👀 Real-Time Job Monitor")
	pterm.Info.Printfln("Watching job: %s", jobID)
	pterm.Info.Println("Press Ctrl+C to exit")
	pterm.Println()

	lastStatus := ""
	lastProgress := 0.0

	for {
		resp, err := apiRequest(daemonURL+"/jobs/"+jobID, "GET", "", nil)
		if err != nil {
			pterm.Error.Printfln("Error: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		var job models.Job
		if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
			resp.Body.Close()
			pterm.Error.Printfln("Parse error: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}
		resp.Body.Close()

		// Check if status or progress changed
		currentStatus := string(job.Status)
		currentProgress := 0.0
		if job.Progress != nil {
			currentProgress = job.Progress.PercentComplete
		}

		if currentStatus != lastStatus || currentProgress != lastProgress {
			// Clear line and show update
			fmt.Printf("\r\033[K")

			statusColor := pterm.FgGray
			switch currentStatus {
			case "running":
				statusColor = pterm.FgLightCyan
			case "completed":
				statusColor = pterm.FgGreen
			case "failed":
				statusColor = pterm.FgRed
			}

			if job.Progress != nil {
				fmt.Printf("[%s] %s %.1f%% - %s",
					time.Now().Format("15:04:05"),
					statusColor.Sprint(currentStatus),
					currentProgress,
					job.Progress.Phase)
			} else {
				fmt.Printf("[%s] %s",
					time.Now().Format("15:04:05"),
					statusColor.Sprint(currentStatus))
			}

			lastStatus = currentStatus
			lastProgress = currentProgress
		}

		// Exit if job is complete
		if currentStatus == "completed" || currentStatus == "failed" || currentStatus == "cancelled" {
			fmt.Println()
			if currentStatus == "completed" {
				showSuccessArt()
				pterm.Success.Println("✅ Job completed successfully!")
			} else {
				pterm.Error.Printfln("❌ Job %s", currentStatus)
				if job.Error != "" {
					pterm.Error.Printfln("Error: %s", job.Error)
				}
			}
			break
		}

		time.Sleep(1 * time.Second)
	}
}

// handleLogs shows job logs
func handleLogs(daemonURL, jobID string, follow bool, tail int) {
	endpoint := fmt.Sprintf("/jobs/logs/%s", jobID)
	if tail > 0 {
		endpoint += fmt.Sprintf("?tail=%d", tail)
	}

	if !follow {
		// Fetch logs once
		resp, err := apiRequest(daemonURL+endpoint, "GET", "", nil)
		if err != nil {
			pterm.Error.Printfln("Failed to get logs: %v", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var logsResp struct {
			Logs []string `json:"logs"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&logsResp); err != nil {
			pterm.Error.Printfln("Failed to parse logs: %v", err)
			os.Exit(1)
		}

		for _, line := range logsResp.Logs {
			fmt.Println(line)
		}
		return
	}

	// Follow logs (polling mode)
	pterm.Info.Println("Following logs... (Press Ctrl+C to exit)")
	pterm.Println()

	lastCount := 0
	for {
		resp, err := apiRequest(daemonURL+endpoint, "GET", "", nil)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}

		var logsResp struct {
			Logs []string `json:"logs"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&logsResp); err != nil {
			resp.Body.Close()
			time.Sleep(1 * time.Second)
			continue
		}
		resp.Body.Close()

		// Print new logs only
		if len(logsResp.Logs) > lastCount {
			for i := lastCount; i < len(logsResp.Logs); i++ {
				fmt.Println(logsResp.Logs[i])
			}
			lastCount = len(logsResp.Logs)
		}

		time.Sleep(1 * time.Second)
	}
}

// generateID generates a simple ID for schedules
func generateID() string {
	return fmt.Sprintf("schedule-%d", time.Now().Unix())
}

// handleManifest manages manifest-based conversions
func handleManifest(daemonURL string, action string, args []string) {
	switch action {
	case "convert":
		if len(args) < 2 {
			pterm.Error.Println("Usage: hyperctl manifest convert <vm-path> <output-path> [options]")
			pterm.Info.Println("Options:")
			pterm.Info.Println("  -format <qcow2|raw|vdi>    Target format (default: qcow2)")
			pterm.Info.Println("  -compress                  Compress export")
			pterm.Info.Println("  -verify                    Verify checksums")
			pterm.Info.Println("Example: hyperctl manifest convert /dc/vm/web01 /exports/web01 -format qcow2 -verify")
			os.Exit(1)
		}
		convertWithManifest(daemonURL, args)
	case "generate":
		if len(args) < 2 {
			pterm.Error.Println("Usage: hyperctl manifest generate <vm-path> <output-path> [options]")
			pterm.Info.Println("Generate manifest only without conversion")
			os.Exit(1)
		}
		generateManifest(daemonURL, args)
	default:
		pterm.Error.Printfln("Unknown manifest action: %s", action)
		pterm.Info.Println("Available actions: convert, generate")
		os.Exit(1)
	}
}

func convertWithManifest(daemonURL string, args []string) {
	vmPath := args[0]
	outputPath := args[1]

	// Parse optional flags
	targetFormat := "qcow2"
	compress := false
	verify := true

	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "-format":
			if i+1 < len(args) {
				targetFormat = args[i+1]
				i++
			}
		case "-compress":
			compress = true
		case "-verify":
			verify = true
		case "-no-verify":
			verify = false
		}
	}

	spinner, _ := pterm.DefaultSpinner.Start("Submitting one-shot export + conversion job...")

	// Create job definition with manifest and auto-convert
	jobDef := map[string]interface{}{
		"vm_path":     vmPath,
		"output_path": outputPath,
		"format":      "ovf",
		"compress":    compress,
		"options": map[string]interface{}{
			"generate_manifest":      true,
			"manifest_target_format": targetFormat,
			"manifest_checksum":      verify,
			"auto_convert":           true,
			"stream_conversion":      true,
		},
	}

	data, _ := json.Marshal(jobDef)
	resp, err := apiRequest(daemonURL+"/jobs/submit", "POST", "application/json", data)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to submit job: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		os.Exit(1)
	}

	var result struct {
		JobID string `json:"job_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		os.Exit(1)
	}

	spinner.Success("One-shot conversion job submitted!")
	pterm.Success.Printfln("✅ Job ID: %s", result.JobID)
	pterm.Info.Println("This job will:")
	pterm.Info.Printfln("  1. Export VM from: %s", vmPath)
	pterm.Info.Printfln("  2. Generate Artifact Manifest v1.0")
	pterm.Info.Printfln("  3. Convert to %s format", strings.ToUpper(targetFormat))
	if verify {
		pterm.Info.Println("  4. Verify checksums")
	}
	pterm.Info.Printfln("  5. Output to: %s", outputPath)
	pterm.Println()
	pterm.Info.Println("Monitor progress:")
	pterm.Info.Printfln("  hyperctl watch %s", result.JobID)
	pterm.Info.Printfln("  hyperctl logs -f %s", result.JobID)
}

func generateManifest(daemonURL string, args []string) {
	vmPath := args[0]
	outputPath := args[1]

	// Parse optional flags
	targetFormat := "qcow2"
	for i := 2; i < len(args); i++ {
		if args[i] == "-format" && i+1 < len(args) {
			targetFormat = args[i+1]
			i++
		}
	}

	spinner, _ := pterm.DefaultSpinner.Start("Submitting manifest generation job...")

	jobDef := map[string]interface{}{
		"vm_path":     vmPath,
		"output_path": outputPath,
		"format":      "ovf",
		"options": map[string]interface{}{
			"generate_manifest":      true,
			"manifest_target_format": targetFormat,
			"manifest_checksum":      true,
			"auto_convert":           false, // Manifest only, no conversion
		},
	}

	data, _ := json.Marshal(jobDef)
	resp, err := apiRequest(daemonURL+"/jobs/submit", "POST", "application/json", data)
	if err != nil {
		spinner.Fail(fmt.Sprintf("Failed to submit job: %v", err))
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		spinner.Fail(fmt.Sprintf("Server error: %s", string(body)))
		os.Exit(1)
	}

	var result struct {
		JobID string `json:"job_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		spinner.Fail(fmt.Sprintf("Failed to parse response: %v", err))
		os.Exit(1)
	}

	spinner.Success("Manifest generation job submitted!")
	pterm.Success.Printfln("✅ Job ID: %s", result.JobID)
	pterm.Info.Println("This will generate an Artifact Manifest v1.0 file")
	pterm.Info.Printfln("Target format: %s", targetFormat)
	pterm.Info.Printfln("Output: %s/manifest.json", outputPath)
	pterm.Println()
	pterm.Info.Println("After export completes, manually convert with:")
	pterm.Info.Printfln("  hyper2kvm -manifest %s/manifest.json -output %s", outputPath, outputPath)
}
